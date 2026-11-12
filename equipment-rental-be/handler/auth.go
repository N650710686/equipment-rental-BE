package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"equipmentrental/auth"
	"equipmentrental/mw"
	"equipmentrental/store"
)
var emailRe = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}$`)
type registerReq struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role,omitempty"`
}

// Register สมัครบัญชี: เลือกได้เฉพาะ guest หรือ member; admin ตั้งผ่าน SQL เท่านั้น
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !readJSON(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.DisplayName)
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role == "" {
		role = "member"
	}
	switch {
	case !emailRe.MatchString(email) || len(email) > 255:
    writeError(w, http.StatusBadRequest, "BAD_EMAIL", "รูปแบบอีเมลไม่ถูกต้อง ต้องมี @ และโดเมน เช่น name@example.com หรือ name@example.co.th")
		return
	case utf8.RuneCountInString(req.Password) < 8 || len(req.Password) > 72:
		writeError(w, http.StatusBadRequest, "BAD_PASSWORD", "รหัสผ่านต้องยาว 8 ตัวอักษรขึ้นไป และไม่เกิน 72 ไบต์")
		return
	case name == "" || utf8.RuneCountInString(name) > 50:
		writeError(w, http.StatusBadRequest, "BAD_NAME", "ต้องมีชื่อที่แสดง และยาวไม่เกิน 50 ตัวอักษร")
		return
	case role != "guest" && role != "member":
		writeError(w, http.StatusBadRequest, "BAD_ROLE", "role ต้องเป็น guest หรือ member เท่านั้น")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		internal(w, err)
		return
	}
	m, err := h.Store.CreateMember(r.Context(), email, hash, name, role)
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "อีเมลนี้มีบัญชีอยู่แล้ว")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !readJSON(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	m, err := h.Store.MemberByEmail(r.Context(), email)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "BAD_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if !auth.CheckPassword(m.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "BAD_CREDENTIALS", "อีเมลหรือรหัสผ่านไม่ถูกต้อง")
		return
	}
	token, err := h.Tickets.Issue(m.ID, m.Role)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "member": m})
}

// Me ข้อมูลส่วนตัวสำหรับหน้า Profile เลขสมาชิกมาจากตั๋วที่ตรวจแล้วเท่านั้น
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	m, err := h.Store.MemberByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "MEMBER_GONE", "ไม่พบบัญชีนี้แล้ว กรุณาเข้าสู่ระบบใหม่")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

type changePasswordReq struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword เปลี่ยนรหัสผ่าน: ต้องใส่รหัสเดิมให้ถูกก่อน
// (รหัสเดิมผิดตอบ 400 ไม่ใช่ 401 เพื่อไม่ให้ FE เข้าใจผิดว่าตั๋วหมดอายุแล้วเด้งออกจากระบบ)
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	var req changePasswordReq
	if !readJSON(w, r, &req) {
		return
	}
	if utf8.RuneCountInString(req.NewPassword) < 8 || len(req.NewPassword) > 72 {
		writeError(w, http.StatusBadRequest, "BAD_PASSWORD", "รหัสผ่านใหม่ต้องยาว 8 ตัวอักษรขึ้นไป และไม่เกิน 72 ไบต์")
		return
	}
	hash, err := h.Store.PasswordHashByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "MEMBER_GONE", "ไม่พบบัญชีนี้แล้ว กรุณาเข้าสู่ระบบใหม่")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if !auth.CheckPassword(hash, req.CurrentPassword) {
		writeError(w, http.StatusBadRequest, "WRONG_PASSWORD", "รหัสผ่านเดิมไม่ถูกต้อง")
		return
	}
	if req.CurrentPassword == req.NewPassword {
		writeError(w, http.StatusBadRequest, "SAME_PASSWORD", "รหัสผ่านใหม่ต้องไม่ซ้ำรหัสเดิม")
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		internal(w, err)
		return
	}
	if err := h.Store.UpdatePassword(r.Context(), id, newHash); err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "เปลี่ยนรหัสผ่านเรียบร้อย"})
}
