package handler

import (
	"errors"
	"net/http"
	"time"

	"equipmentrental/mw"
	"equipmentrental/store"
)

const dateLayout = "2006-01-02"

type borrowReq struct {
	EquipmentID int64  `json:"equipmentId"`
	Quantity    int    `json:"quantity"` // ไม่ส่ง = 1
	BorrowDate  string `json:"borrowDate"` // YYYY-MM-DD
	ReturnDate  string `json:"returnDate"` // YYYY-MM-DD
}

// today คือ "วันนี้" ตามเวลาไทย (เซิร์ฟเวอร์ Render ใช้ UTC)
func today() time.Time {
	now := time.Now().In(time.FixedZone("ICT", 7*3600))
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// CreateBorrow สมาชิกส่งคำขอยืม (ต้อง login: ถ้าไม่มีตั๋ว middleware ตอบ 401 NO_TOKEN ให้ FE เด้งไปหน้า Login)
func (h *Handler) CreateBorrow(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	var req borrowReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.EquipmentID <= 0 {
		writeError(w, http.StatusBadRequest, "BAD_ID", "ต้องระบุอุปกรณ์ที่ต้องการยืม")
		return
	}
	if req.Quantity == 0 {
		req.Quantity = 1
	}
	if req.Quantity < 1 || req.Quantity > 100 {
		writeError(w, http.StatusBadRequest, "BAD_QUANTITY", "จำนวนต้องอยู่ระหว่าง 1-100")
		return
	}
	from, err1 := time.Parse(dateLayout, req.BorrowDate)
	to, err2 := time.Parse(dateLayout, req.ReturnDate)
	switch {
	case err1 != nil || err2 != nil:
		writeError(w, http.StatusBadRequest, "BAD_DATE", "วันที่ต้องอยู่ในรูปแบบ YYYY-MM-DD")
		return
	case from.Before(today()):
		writeError(w, http.StatusBadRequest, "BAD_DATE", "วันที่ยืมต้องไม่ใช่วันที่ผ่านมาแล้ว")
		return
	case to.Before(from):
		writeError(w, http.StatusBadRequest, "BAD_DATE", "วันที่คืนต้องไม่ก่อนวันที่ยืม")
		return
	}
	b, err := h.Store.CreateBorrowRequest(r.Context(), memberID, req.EquipmentID, req.Quantity, req.BorrowDate, req.ReturnDate)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์นี้")
		return
	}
	if errors.Is(err, store.ErrOutOfStock) {
		writeError(w, http.StatusConflict, "OUT_OF_STOCK", "อุปกรณ์ถูกยืมหมดแล้ว หรือจำนวนที่ว่างไม่พอ")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// MyBorrows ประวัติการยืมของตัวเอง (หน้า Profile) เห็นเฉพาะของตัวเอง เพราะเลขสมาชิกมาจากตั๋ว
func (h *Handler) MyBorrows(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}
	items, err := h.Store.ListBorrowRequests(r.Context(), memberID, "")
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ---------- ฝั่ง admin (ครอบด้วย RequireLogin + RequireRole("admin") ใน main.go แล้ว) ----------

// AdminListBorrows ?status=pending|approved|rejected|returned (ไม่ส่ง = ทั้งหมด)
func (h *Handler) AdminListBorrows(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "pending", "approved", "rejected", "returned":
	default:
		writeError(w, http.StatusBadRequest, "BAD_STATUS", "status ต้องเป็น pending, approved, rejected หรือ returned")
		return
	}
	items, err := h.Store.ListBorrowRequests(r.Context(), 0, status)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func borrowErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "REQUEST_NOT_FOUND", "ไม่พบคำขอยืมนี้")
	case errors.Is(err, store.ErrBadState):
		writeError(w, http.StatusConflict, "BAD_STATE", "สถานะของคำขอไม่ตรงกับการทำรายการนี้")
	case errors.Is(err, store.ErrOutOfStock):
		writeError(w, http.StatusConflict, "OUT_OF_STOCK", "สต็อกไม่พอสำหรับอนุมัติคำขอนี้")
	default:
		internal(w, err)
	}
}

func (h *Handler) AdminApproveBorrow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	adminID, ok2 := mw.MemberID(r.Context())
	if !ok || !ok2 {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขคำขอต้องเป็นตัวเลข")
		return
	}
	b, err := h.Store.ApproveBorrow(r.Context(), id, adminID)
	if err != nil {
		borrowErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) AdminRejectBorrow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	adminID, ok2 := mw.MemberID(r.Context())
	if !ok || !ok2 {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขคำขอต้องเป็นตัวเลข")
		return
	}
	b, err := h.Store.RejectBorrow(r.Context(), id, adminID)
	if err != nil {
		borrowErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// AdminReturnBorrow กดรับคืนอุปกรณ์ -> คืนสต็อกใน DB
func (h *Handler) AdminReturnBorrow(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขคำขอต้องเป็นตัวเลข")
		return
	}
	b, err := h.Store.ReturnBorrow(r.Context(), id)
	if err != nil {
		borrowErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}
