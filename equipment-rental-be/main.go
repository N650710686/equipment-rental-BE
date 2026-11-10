package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"

	"equipmentrental/auth"
	"equipmentrental/handler"
	"equipmentrental/mw"
	"equipmentrental/store"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร")
	}

	cfg, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("DATABASE_URL ไม่ถูกต้อง ", err)
	}
	// Supabase pooler (โหมด transaction, พอร์ต 6543) ไม่รองรับ prepared statement
	// ใช้ simple protocol แล้วใช้ได้ทั้ง session และ transaction pooler
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("ต่อฐานข้อมูลไม่ติด ", err)
	}

	tickets := &auth.Issuer{Secret: []byte(secret), TTL: time.Hour}
	h := &handler.Handler{Store: &store.Store{Pool: pool}, Tickets: tickets}

	// login = ต้องมีตั๋ว, admin = ต้องมีตั๋วและบทบาทเป็น admin (ตรวจตั๋วก่อนเสมอ ค่อยดูบทบาท)
	login := func(next http.HandlerFunc) http.HandlerFunc { return mw.RequireLogin(tickets, next) }
	admin := func(next http.HandlerFunc) http.HandlerFunc { return login(mw.RequireRole("admin", next)) }

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)

	// ทุกคน (ไม่ต้อง login): ดูอุปกรณ์ + สมัคร/เข้าสู่ระบบ
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/equipment", h.ListEquipment)
	mux.HandleFunc("GET /api/equipment/{id}", h.GetEquipment)

	// Member (ต้อง login)
	mux.HandleFunc("GET /api/me", login(h.Me))
	mux.HandleFunc("PUT /api/me/password", login(h.ChangePassword))
	mux.HandleFunc("GET /api/me/borrow-requests", login(h.MyBorrows))
	mux.HandleFunc("POST /api/borrow-requests", login(h.CreateBorrow))

	// Admin
	mux.HandleFunc("GET /api/admin/stats", admin(h.AdminStats))
	mux.HandleFunc("POST /api/admin/equipment", admin(h.AdminCreateEquipment))
	mux.HandleFunc("PUT /api/admin/equipment/{id}", admin(h.AdminUpdateEquipment))
	mux.HandleFunc("DELETE /api/admin/equipment/{id}", admin(h.AdminDeleteEquipment))
	mux.HandleFunc("GET /api/admin/borrow-requests", admin(h.AdminListBorrows))
	mux.HandleFunc("PUT /api/admin/borrow-requests/{id}/approve", admin(h.AdminApproveBorrow))
	mux.HandleFunc("PUT /api/admin/borrow-requests/{id}/reject", admin(h.AdminRejectBorrow))
	mux.HandleFunc("PUT /api/admin/borrow-requests/{id}/return", admin(h.AdminReturnBorrow))

	origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Equipment Rental API ฟังอยู่ที่ :"+port, "ต้อนรับ", origins)
	log.Fatal(http.ListenAndServe(":"+port, mw.Logging(mw.CORS(origins, mux))))
}
