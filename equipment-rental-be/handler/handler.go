package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"equipmentrental/auth"
	"equipmentrental/store"
)

type Handler struct {
	Store   *store.Store
	Tickets *auth.Issuer
	Images  *ImageStorage // nil = ยังไม่ตั้งค่า upload
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

// internal จดรายละเอียดไว้ใน log ฝั่งเรา แต่บอกผู้ใช้แค่ว่าระบบขัดข้อง
func internal(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "INTERNAL", "ระบบขัดข้อง ลองใหม่อีกครั้ง")
}

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

// readJSON อ่าน body (จำกัด 1MB) ถ้าอ่านไม่ได้ตอบ 400 เองแล้วคืน false
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "อ่านข้อมูล JSON ไม่ออก")
		return false
	}
	return true
}

// Health ให้ Render ใช้เช็กว่าเซิร์ฟเวอร์และ DB ยังอยู่
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.Pool.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "DB_DOWN", "ต่อฐานข้อมูลไม่ได้")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
