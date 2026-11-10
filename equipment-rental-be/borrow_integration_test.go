package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"equipmentrental/store"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	store       *store.Store
	pool        *pgxpool.Pool
	memberID    int64
	adminID     int64
	equipmentID int64
	totalQty    int
}

func newFixture(t *testing.T, totalQty int) *fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is not set. Refusing to run integration tests.")
	}

	if dsn == os.Getenv("DATABASE_URL") {
		t.Fatal("TEST_DATABASE_URL must not be the production DATABASE_URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	f := &fixture{
		store:    &store.Store{Pool: pool},
		pool:     pool,
		totalQty: totalQty,
	}

	unique := time.Now().UnixNano()

	err = pool.QueryRow(ctx,
		`INSERT INTO members (email, password_hash, display_name, role)
		 VALUES ($1, 'integration-test-hash', $2, 'member')
		 RETURNING id`,
		fmt.Sprintf("it-member-%d@example.test", unique),
		"Integration Member",
	).Scan(&f.memberID)
	if err != nil {
		t.Fatalf("create test member: %v", err)
	}

	err = pool.QueryRow(ctx,
		`INSERT INTO members (email, password_hash, display_name, role)
		 VALUES ($1, 'integration-test-hash', $2, 'admin')
		 RETURNING id`,
		fmt.Sprintf("it-admin-%d@example.test", unique),
		"Integration Admin",
	).Scan(&f.adminID)
	if err != nil {
		t.Fatalf("create test admin: %v", err)
	}

	err = pool.QueryRow(ctx,
		`INSERT INTO equipment (name, total_qty, available_qty)
		 VALUES ($1, $2, $2)
		 RETURNING id`,
		fmt.Sprintf("Integration Equipment %d", unique),
		totalQty,
	).Scan(&f.equipmentID)
	if err != nil {
		t.Fatalf("create test equipment: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(), 10*time.Second,
		)
		defer cleanupCancel()

		_, _ = pool.Exec(cleanupCtx,
			`DELETE FROM borrow_requests WHERE equipment_id = $1`,
			f.equipmentID,
		)

		_, _ = pool.Exec(cleanupCtx,
			`DELETE FROM equipment WHERE id = $1`,
			f.equipmentID,
		)

		_, _ = pool.Exec(cleanupCtx,
			`DELETE FROM members WHERE id = $1 OR id = $2`,
			f.memberID, f.adminID,
		)
	})

	return f
}

func (f *fixture) createBorrow(t *testing.T, qty int) store.BorrowRequest {
	t.Helper()

	ctx := context.Background()

	b, err := f.store.CreateBorrowRequest(
		ctx,
		f.memberID,
		f.equipmentID,
		qty,
		"2026-11-10",
		"2026-11-12",
	)
	if err != nil {
		t.Fatalf("CreateBorrowRequest: %v", err)
	}

	return b
}

func (f *fixture) availableQty(t *testing.T) int {
	t.Helper()

	var qty int
	err := f.pool.QueryRow(context.Background(),
		`SELECT available_qty FROM equipment WHERE id = $1`,
		f.equipmentID,
	).Scan(&qty)

	if err != nil {
		t.Fatalf("read available_qty: %v", err)
	}

	return qty
}

// 1. สร้างคำขอยืม สถานะต้องเป็น pending และยังไม่หัก Stock
func TestCreateBorrowRequest(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	if b.Status != "pending" {
		t.Fatalf("expected pending, got %s", b.Status)
	}

	if b.Quantity != 2 {
		t.Fatalf("expected quantity 2, got %d", b.Quantity)
	}

	if f.availableQty(t) != 5 {
		t.Fatalf("stock should remain 5, got %d", f.availableQty(t))
	}
}

// 2. อนุมัติคำขอ ต้องหัก Stock
func TestApproveBorrow(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	result, err := f.store.ApproveBorrow(
		context.Background(), b.ID, f.adminID,
	)
	if err != nil {
		t.Fatalf("ApproveBorrow: %v", err)
	}

	if result.Status != "approved" {
		t.Fatalf("expected approved, got %s", result.Status)
	}

	if f.availableQty(t) != 3 {
		t.Fatalf("expected stock 3, got %d", f.availableQty(t))
	}
}

// 3. ปฏิเสธคำขอ Stock ต้องไม่เปลี่ยน
func TestRejectBorrow(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	result, err := f.store.RejectBorrow(
		context.Background(), b.ID, f.adminID,
	)
	if err != nil {
		t.Fatalf("RejectBorrow: %v", err)
	}

	if result.Status != "rejected" {
		t.Fatalf("expected rejected, got %s", result.Status)
	}

	if f.availableQty(t) != 5 {
		t.Fatalf("stock should remain 5, got %d", f.availableQty(t))
	}

	_, err = f.store.RejectBorrow(
		context.Background(), b.ID, f.adminID,
	)
	if !errors.Is(err, store.ErrBadState) {
		t.Fatalf("expected ErrBadState, got %v", err)
	}
}

// 4. คืนอุปกรณ์ ต้องเพิ่ม Stock กลับ
func TestReturnBorrow(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	_, err := f.store.ApproveBorrow(
		context.Background(), b.ID, f.adminID,
	)
	if err != nil {
		t.Fatalf("ApproveBorrow: %v", err)
	}

	result, err := f.store.ReturnBorrow(context.Background(), b.ID)
	if err != nil {
		t.Fatalf("ReturnBorrow: %v", err)
	}

	if result.Status != "returned" {
		t.Fatalf("expected returned, got %s", result.Status)
	}

	if result.ReturnedAt == nil {
		t.Fatal("returned_at should not be nil")
	}

	if f.availableQty(t) != 5 {
		t.Fatalf("expected stock 5, got %d", f.availableQty(t))
	}

	_, err = f.store.ReturnBorrow(context.Background(), b.ID)
	if !errors.Is(err, store.ErrBadState) {
		t.Fatalf("expected ErrBadState on second return, got %v", err)
	}

	if f.availableQty(t) != 5 {
		t.Fatalf("stock changed after second return: %d", f.availableQty(t))
	}
}

// 5. Approve พร้อมกัน 2 ครั้ง ต้องสำเร็จเพียงครั้งเดียว
func TestConcurrentApprove(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := f.store.ApproveBorrow(
				context.Background(), b.ID, f.adminID,
			)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	success := 0
	badState := 0

	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, store.ErrBadState):
			badState++
		default:
			t.Fatalf("unexpected approve error: %v", err)
		}
	}

	if success != 1 {
		t.Fatalf("expected 1 successful approval, got %d", success)
	}

	if badState != 1 {
		t.Fatalf("expected 1 ErrBadState, got %d", badState)
	}

	if f.availableQty(t) != 3 {
		t.Fatalf("expected stock 3, got %d", f.availableQty(t))
	}
}

// 6. Return พร้อมกัน 2 ครั้ง ต้องเพิ่ม Stock เพียงครั้งเดียว
func TestConcurrentReturn(t *testing.T) {
	f := newFixture(t, 5)

	b := f.createBorrow(t, 2)

	_, err := f.store.ApproveBorrow(
		context.Background(), b.ID, f.adminID,
	)
	if err != nil {
		t.Fatalf("ApproveBorrow: %v", err)
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 2)

	for i := 0; i < 2; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, err := f.store.ReturnBorrow(
				context.Background(), b.ID,
			)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	success := 0
	badState := 0

	for err := range results {
		switch {
		case err == nil:
			success++
		case errors.Is(err, store.ErrBadState):
			badState++
		default:
			t.Fatalf("unexpected return error: %v", err)
		}
	}

	if success != 1 {
		t.Fatalf("expected 1 successful return, got %d", success)
	}

	if badState != 1 {
		t.Fatalf("expected 1 ErrBadState, got %d", badState)
	}

	if f.availableQty(t) != 5 {
		t.Fatalf("expected stock 5, got %d", f.availableQty(t))
	}
}

// 7. CreateBorrowRequest แข่งกับ DeleteEquipment
// ต้องไม่มีคำขอยืมเกิดขึ้นหลังอุปกรณ์ถูกลบ (soft delete)
func TestConcurrentBorrowAndDelete(t *testing.T) {
	f := newFixture(t, 5)

	var wg sync.WaitGroup
	start := make(chan struct{})

	var borrowErr error
	var deleteErr error
	var borrowResult store.BorrowRequest

	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start

		borrowResult, borrowErr = f.store.CreateBorrowRequest(
			context.Background(),
			f.memberID,
			f.equipmentID,
			1,
			"2026-11-10",
			"2026-11-12",
		)
	}()

	go func() {
		defer wg.Done()
		<-start

		deleteErr = f.store.DeleteEquipment(
			context.Background(), f.equipmentID,
		)
	}()

	close(start)
	wg.Wait()

	switch {
	case borrowErr == nil:
		if borrowResult.Status != "pending" {
			t.Fatalf("expected pending, got %s", borrowResult.Status)
		}

		if !errors.Is(deleteErr, store.ErrHasActiveLoans) {
			t.Fatalf("expected ErrHasActiveLoans, got %v", deleteErr)
		}

		var active bool
		err := f.pool.QueryRow(context.Background(),
			`SELECT is_active FROM equipment WHERE id = $1`,
			f.equipmentID,
		).Scan(&active)
		if err != nil {
			t.Fatalf("read equipment active status: %v", err)
		}

		if !active {
			t.Fatal("equipment must remain active while pending request exists")
		}

	case errors.Is(borrowErr, store.ErrNotFound):
		if deleteErr != nil {
			t.Fatalf("expected successful delete, got %v", deleteErr)
		}

		var active bool
		err := f.pool.QueryRow(context.Background(),
			`SELECT is_active FROM equipment WHERE id = $1`,
			f.equipmentID,
		).Scan(&active)
		if err != nil {
			t.Fatalf("read equipment active status: %v", err)
		}

		if active {
			t.Fatal("equipment should be inactive after successful delete")
		}

	default:
		t.Fatalf("unexpected borrow error: %v", borrowErr)
	}
}
