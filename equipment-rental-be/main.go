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

	// อัปโหลดรูปไป Supabase Storage (ไม่ตั้งค่าก็รันได้ แต่ endpoint upload จะตอบ 503)
	if u, k := os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_SERVICE_KEY"); u != "" && k != "" {
		bucket := os.Getenv("SUPABASE_BUCKET")
		if bucket == "" {
			bucket = "equipment-images"
		}
		h.Images = handler.NewImageStorage(u, k, bucket)
	} else {
		log.Println("คำเตือน: ไม่ได้ตั้ง SUPABASE_URL/SUPABASE_SERVICE_KEY จึงอัปโหลดรูปไม่ได้")
	}

	// login = ต้องมีตั๋ว, member = ต้องเป็น member, admin = ต้องเป็น admin
	login := func(next http.HandlerFunc) http.HandlerFunc { return mw.RequireLogin(tickets, next) }
	// member = สมาชิกหรือผู้ดูแล (admin ใช้ Profile/ยืมได้ด้วย) ส่วน guest ยังใช้ไม่ได้
	member := func(next http.HandlerFunc) http.HandlerFunc { return login(mw.RequireAnyRole(next, "member", "admin")) }
	admin := func(next http.HandlerFunc) http.HandlerFunc { return login(mw.RequireRole("admin", next)) }

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", h.Health)

	// ทุกคน (ไม่ต้อง login): ดูอุปกรณ์ + สมัคร/เข้าสู่ระบบ
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/categories", h.Categories)
	mux.HandleFunc("GET /api/equipment", h.ListEquipment)
	mux.HandleFunc("GET /api/equipment/{id}", h.GetEquipment)

	// Member เท่านั้น (Guest login ได้ แต่ไม่มีสิทธิ์ใช้ Profile/ยืม/ดูประวัติ)
	mux.HandleFunc("GET /api/me", member(h.Me))
	mux.HandleFunc("PUT /api/me/password", member(h.ChangePassword))
	mux.HandleFunc("GET /api/me/borrow-requests", member(h.MyBorrows))
	mux.HandleFunc("POST /api/borrow-requests", member(h.CreateBorrow))

	// Admin
	mux.HandleFunc("GET /api/admin/stats", admin(h.AdminStats))
	mux.HandleFunc("POST /api/admin/upload-image", admin(h.AdminUploadImage))
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