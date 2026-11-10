package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrOutOfStock = errors.New("out of stock")
	ErrBadState   = errors.New("bad request state") // เช่น อนุมัติคำขอที่ไม่ใช่ pending
)

type BorrowRequest struct {
	ID            int64      `json:"id"`
	MemberID      int64      `json:"memberId"`
	MemberName    string     `json:"memberName"`
	MemberEmail   string     `json:"memberEmail"`
	EquipmentID   int64      `json:"equipmentId"`
	EquipmentName string     `json:"equipmentName"`
	Quantity      int        `json:"quantity"`
	BorrowDate    string     `json:"borrowDate"` // YYYY-MM-DD
	ReturnDate    string     `json:"returnDate"` // YYYY-MM-DD
	Status        string     `json:"status"`     // pending | approved | rejected | returned
	CreatedAt     time.Time  `json:"createdAt"`
	ReturnedAt    *time.Time `json:"returnedAt"`
}

const selectBorrow = `
SELECT r.id, r.member_id, m.display_name, m.email, r.equipment_id, e.name, r.quantity,
       to_char(r.borrow_date, 'YYYY-MM-DD'), to_char(r.return_date, 'YYYY-MM-DD'),
       r.status, r.created_at, r.returned_at
  FROM borrow_requests r
  JOIN members   m ON m.id = r.member_id
  JOIN equipment e ON e.id = r.equipment_id`

func scanBorrow(row scanner) (BorrowRequest, error) {
	var b BorrowRequest
	err := row.Scan(&b.ID, &b.MemberID, &b.MemberName, &b.MemberEmail, &b.EquipmentID, &b.EquipmentName,
		&b.Quantity, &b.BorrowDate, &b.ReturnDate, &b.Status, &b.CreatedAt, &b.ReturnedAt)
	return b, err
}

func (s *Store) getBorrow(ctx context.Context, id int64) (BorrowRequest, error) {
	b, err := scanBorrow(s.Pool.QueryRow(ctx, selectBorrow+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return BorrowRequest{}, ErrNotFound
	}
	return b, err
}

// CreateBorrowRequest สร้างคำขอสถานะ pending (ยังไม่หักสต็อก ไปหักตอน admin อนุมัติ)
// ล็อกอุปกรณ์ (FOR UPDATE) ก่อนเช็กสต็อกและสร้างคำขอ ใน transaction เดียว
// วันที่ตรวจรูปแบบมาแล้วที่ handler ส่งมาเป็น "YYYY-MM-DD"
func (s *Store) CreateBorrowRequest(ctx context.Context, memberID, equipmentID int64, qty int, borrowDate, returnDate string) (BorrowRequest, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return BorrowRequest{}, err
	}
	defer tx.Rollback(ctx)

	var avail int
	err = tx.QueryRow(ctx,
		`SELECT available_qty FROM equipment WHERE id = $1 AND is_active FOR UPDATE`, equipmentID).Scan(&avail)
	if errors.Is(err, pgx.ErrNoRows) {
		return BorrowRequest{}, ErrNotFound
	}
	if err != nil {
		return BorrowRequest{}, err
	}
	if avail < qty {
		return BorrowRequest{}, ErrOutOfStock
	}
	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO borrow_requests (member_id, equipment_id, quantity, borrow_date, return_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`, memberID, equipmentID, qty, borrowDate, returnDate).Scan(&id)
	if err != nil {
		return BorrowRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return BorrowRequest{}, err
	}
	return s.getBorrow(ctx, id)
}

// ListBorrowRequests memberID = 0 คือทุกคน (ฝั่ง admin), status ว่าง = ทุกสถานะ
func (s *Store) ListBorrowRequests(ctx context.Context, memberID int64, status string) ([]BorrowRequest, error) {
	rows, err := s.Pool.Query(ctx,
		selectBorrow+`
		 WHERE ($1::bigint = 0 OR r.member_id = $1::bigint)
		   AND ($2::text = '' OR r.status = $2::text)
		 ORDER BY r.created_at DESC, r.id DESC`, memberID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BorrowRequest{}
	for rows.Next() {
		b, err := scanBorrow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// lockForUpdate ล็อกตามลำดับเดียวกันทุกที่: อุปกรณ์ก่อน แล้วค่อยคำขอ (กัน deadlock กับ Create/Delete)
// คืน equipment_id, quantity, status ของคำขอที่ล็อกแล้ว
func lockForUpdate(ctx context.Context, tx pgx.Tx, id int64) (equipID int64, qty int, status string, err error) {
	// 1) หาว่าคำขอนี้เป็นของอุปกรณ์ไหน (ยังไม่ล็อก; equipment_id ไม่เคยเปลี่ยนหลังสร้าง)
	err = tx.QueryRow(ctx, `SELECT equipment_id FROM borrow_requests WHERE id = $1`, id).Scan(&equipID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, "", ErrNotFound
	}
	if err != nil {
		return 0, 0, "", err
	}
	// 2) ล็อกอุปกรณ์ก่อน
	var one int
	if err = tx.QueryRow(ctx, `SELECT 1 FROM equipment WHERE id = $1 FOR UPDATE`, equipID).Scan(&one); err != nil {
		return 0, 0, "", err
	}
	// 3) แล้วค่อยล็อกคำขอ และอ่านสถานะล่าสุดหลังได้ล็อก
	err = tx.QueryRow(ctx,
		`SELECT quantity, status FROM borrow_requests WHERE id = $1 FOR UPDATE`, id).Scan(&qty, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, "", ErrNotFound
	}
	return equipID, qty, status, err
}

// ApproveBorrow pending -> approved และหักสต็อกในธุรกรรมเดียวกัน
// เงื่อนไข available_qty >= qty ใน UPDATE เป็นด่านสุดท้ายกันสต็อกติดลบ
func (s *Store) ApproveBorrow(ctx context.Context, id, adminID int64) (BorrowRequest, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return BorrowRequest{}, err
	}
	defer tx.Rollback(ctx)

	equipID, qty, status, err := lockForUpdate(ctx, tx, id)
	if err != nil {
		return BorrowRequest{}, err
	}
	if status != "pending" {
		return BorrowRequest{}, ErrBadState
	}
	ct, err := tx.Exec(ctx,
		`UPDATE equipment SET available_qty = available_qty - $2, updated_at = now()
		  WHERE id = $1 AND available_qty >= $2`, equipID, qty)
	if err != nil {
		return BorrowRequest{}, err
	}
	if ct.RowsAffected() == 0 {
		return BorrowRequest{}, ErrOutOfStock
	}
	if _, err = tx.Exec(ctx,
		`UPDATE borrow_requests SET status = 'approved', approved_by = $2, updated_at = now() WHERE id = $1`,
		id, adminID); err != nil {
		return BorrowRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return BorrowRequest{}, err
	}
	return s.getBorrow(ctx, id)
}

// RejectBorrow pending -> rejected (ไม่แตะสต็อก)
func (s *Store) RejectBorrow(ctx context.Context, id, adminID int64) (BorrowRequest, error) {
	ct, err := s.Pool.Exec(ctx,
		`UPDATE borrow_requests SET status = 'rejected', approved_by = $2, updated_at = now()
		  WHERE id = $1 AND status = 'pending'`, id, adminID)
	if err != nil {
		return BorrowRequest{}, err
	}
	if ct.RowsAffected() == 0 {
		var one int
		err := s.Pool.QueryRow(ctx, `SELECT 1 FROM borrow_requests WHERE id = $1`, id).Scan(&one)
		if errors.Is(err, pgx.ErrNoRows) {
			return BorrowRequest{}, ErrNotFound
		}
		if err != nil {
			return BorrowRequest{}, err
		}
		return BorrowRequest{}, ErrBadState
	}
	return s.getBorrow(ctx, id)
}

// ReturnBorrow approved -> returned และคืนสต็อกให้อุปกรณ์ในธุรกรรมเดียวกัน (ล็อกลำดับเดียวกับ Approve)
func (s *Store) ReturnBorrow(ctx context.Context, id int64) (BorrowRequest, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return BorrowRequest{}, err
	}
	defer tx.Rollback(ctx)

	equipID, qty, status, err := lockForUpdate(ctx, tx, id)
	if err != nil {
		return BorrowRequest{}, err
	}
	if status != "approved" {
		return BorrowRequest{}, ErrBadState
	}
	if _, err = tx.Exec(ctx,
		`UPDATE equipment SET available_qty = available_qty + $2, updated_at = now() WHERE id = $1`,
		equipID, qty); err != nil {
		return BorrowRequest{}, err
	}
	if _, err = tx.Exec(ctx,
		`UPDATE borrow_requests SET status = 'returned', returned_at = now(), updated_at = now() WHERE id = $1`,
		id); err != nil {
		return BorrowRequest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return BorrowRequest{}, err
	}
	return s.getBorrow(ctx, id)
}

type Stats struct {
	Equipment       int `json:"equipment"`
	OutOfStock      int `json:"outOfStock"`
	Members         int `json:"members"`
	PendingRequests int `json:"pendingRequests"`
	ActiveLoans     int `json:"activeLoans"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.Pool.QueryRow(ctx,
		`SELECT (SELECT count(*) FROM equipment WHERE is_active),
		        (SELECT count(*) FROM equipment WHERE is_active AND available_qty = 0),
		        (SELECT count(*) FROM members),
		        (SELECT count(*) FROM borrow_requests WHERE status = 'pending'),
		        (SELECT count(*) FROM borrow_requests WHERE status = 'approved')`).
		Scan(&st.Equipment, &st.OutOfStock, &st.Members, &st.PendingRequests, &st.ActiveLoans)
	return st, err
}
