package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrStockTooLow   = errors.New("stock too low")       // ลดจำนวนทั้งหมดต่ำกว่าจำนวนที่ถูกยืมอยู่
	ErrHasActiveLoans = errors.New("equipment has active loans") // ลบไม่ได้ เพราะยังมีคำขอ pending/approved
)

type Equipment struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Category     *string `json:"category"`
	ImageURL     *string `json:"imageUrl"`
	TotalQty     int     `json:"totalQty"`
	AvailableQty int     `json:"availableQty"`
	Available    bool    `json:"available"` // true = พร้อมให้ยืม, false = ถูกยืมหมดแล้ว
}

type EquipmentInput struct {
	Name        string
	Description *string
	Category    *string
	ImageURL    *string
	TotalQty    int
}

const equipCols = `id, name, description, category, image_url, total_qty, available_qty`

type scanner interface{ Scan(dest ...any) error }

func scanEquipment(row scanner) (Equipment, error) {
	var e Equipment
	err := row.Scan(&e.ID, &e.Name, &e.Description, &e.Category, &e.ImageURL, &e.TotalQty, &e.AvailableQty)
	e.Available = e.AvailableQty > 0
	return e, err
}

// ListEquipment ค้นจากชื่อหรือหมวดหมู่ (q ว่าง = ทั้งหมด) ค่าจากผู้ใช้ไปทาง $1 เสมอ
func (s *Store) ListEquipment(ctx context.Context, q string) ([]Equipment, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+equipCols+` FROM equipment
		 WHERE is_active
		   AND ($1::text = ''
		        OR lower(name) LIKE '%' || lower($1::text) || '%'
		        OR lower(coalesce(category, '')) LIKE '%' || lower($1::text) || '%')
		 ORDER BY name, id`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Equipment{}
	for rows.Next() {
		e, err := scanEquipment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetEquipment(ctx context.Context, id int64) (Equipment, error) {
	e, err := scanEquipment(s.Pool.QueryRow(ctx,
		`SELECT `+equipCols+` FROM equipment WHERE id = $1 AND is_active`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Equipment{}, ErrNotFound
	}
	return e, err
}

// CreateEquipment อุปกรณ์ใหม่: ของว่าง = ของทั้งหมด
func (s *Store) CreateEquipment(ctx context.Context, in EquipmentInput) (Equipment, error) {
	return scanEquipment(s.Pool.QueryRow(ctx,
		`INSERT INTO equipment (name, description, category, image_url, total_qty, available_qty)
		 VALUES ($1, $2, $3, $4, $5, $5)
		 RETURNING `+equipCols,
		in.Name, in.Description, in.Category, in.ImageURL, in.TotalQty))
}

// UpdateEquipment แก้ข้อมูล/จำนวนสต็อก: ปรับ available_qty ตามส่วนต่างของ total_qty
// เช่น total 5 (ว่าง 2, ถูกยืม 3) แก้เป็น 8 -> ว่าง 5 / แก้เป็น 2 -> ว่าง -1 ผิดกฎ CHECK จึงตอบ ErrStockTooLow
func (s *Store) UpdateEquipment(ctx context.Context, id int64, in EquipmentInput) (Equipment, error) {
	e, err := scanEquipment(s.Pool.QueryRow(ctx,
		`UPDATE equipment
		    SET name = $2, description = $3, category = $4, image_url = $5,
		        available_qty = available_qty + ($6 - total_qty),
		        total_qty = $6,
		        updated_at = now()
		  WHERE id = $1 AND is_active
		  RETURNING `+equipCols,
		id, in.Name, in.Description, in.Category, in.ImageURL, in.TotalQty))
	if errors.Is(err, pgx.ErrNoRows) {
		return Equipment{}, ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23514" { // check_violation
		return Equipment{}, ErrStockTooLow
	}
	return e, err
}

// DeleteEquipment = ซ่อนอุปกรณ์ (soft delete: is_active = false) ไม่ลบแถวจริง ประวัติการยืมจึงอยู่ครบ
// ซ่อนได้เมื่อไม่มีคำขอที่ยังค้าง (pending/approved) ล็อกอุปกรณ์ก่อน (FOR UPDATE) กันมีคนส่งคำขอใหม่แทรกระหว่างตรวจ
func (s *Store) DeleteEquipment(ctx context.Context, id int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var one int
	err = tx.QueryRow(ctx, `SELECT 1 FROM equipment WHERE id = $1 AND is_active FOR UPDATE`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var active bool
	if err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM borrow_requests
		                 WHERE equipment_id = $1 AND status IN ('pending', 'approved'))`, id).Scan(&active); err != nil {
		return err
	}
	if active {
		return ErrHasActiveLoans
	}
	if _, err = tx.Exec(ctx, `UPDATE equipment SET is_active = false, updated_at = now() WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
