package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email taken")

type Member struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"displayName"`
	Role         string     `json:"role"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	PasswordHash string     `json:"-"` // ไม่มีวันออกไปกับ JSON
}

func (s *Store) CreateMember(ctx context.Context, email, hash, name, role string) (Member, error) {
	m := Member{Email: email, DisplayName: name, Role: role}
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO members (email, password_hash, display_name, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, role`, email, hash, name, role).Scan(&m.ID, &m.Role)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Member{}, ErrEmailTaken
	}
	return m, err
}

func (s *Store) MemberByEmail(ctx context.Context, email string) (Member, error) {
	var m Member
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, display_name, role, password_hash
		 FROM members WHERE lower(email) = $1`, email).
		Scan(&m.ID, &m.Email, &m.DisplayName, &m.Role, &m.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	return m, err
}

// MemberByID ใช้กับหน้า Profile: เลขมาจากตั๋วที่ตรวจแล้ว
func (s *Store) MemberByID(ctx context.Context, id int64) (Member, error) {
	var m Member
	err := s.Pool.QueryRow(ctx,
		`SELECT id, email, display_name, role, created_at
		 FROM members WHERE id = $1`, id).
		Scan(&m.ID, &m.Email, &m.DisplayName, &m.Role, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, ErrNotFound
	}
	return m, err
}

// PasswordHashByID ใช้ตรวจ "รหัสผ่านเดิม" ตอนเปลี่ยนรหัสผ่าน
func (s *Store) PasswordHashByID(ctx context.Context, id int64) (string, error) {
	var hash string
	err := s.Pool.QueryRow(ctx, `SELECT password_hash FROM members WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return hash, err
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, hash string) error {
	ct, err := s.Pool.Exec(ctx,
		`UPDATE members SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
