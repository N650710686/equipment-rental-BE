package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"equipmentrental/store"
)

// ทุก handler ในไฟล์นี้ถูกครอบด้วย RequireLogin + RequireRole("admin") ใน main.go แล้ว

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	st, err := h.Store.Stats(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

type equipmentReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	ImageURL    string `json:"imageUrl"`
	TotalQty    int    `json:"totalQty"`
}

func optional(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// parseEquipment ตรวจข้อมูลอุปกรณ์จาก body ถ้าไม่ผ่านตอบ 400 เองแล้วคืน ok = false
func parseEquipment(w http.ResponseWriter, r *http.Request) (store.EquipmentInput, bool) {
	var req equipmentReq
	if !readJSON(w, r, &req) {
		return store.EquipmentInput{}, false
	}
	name := strings.TrimSpace(req.Name)
	switch {
	case name == "" || utf8.RuneCountInString(name) > 100:
		writeError(w, http.StatusBadRequest, "BAD_NAME", "ต้องมีชื่ออุปกรณ์ และยาวไม่เกิน 100 ตัวอักษร")
		return store.EquipmentInput{}, false
	case req.TotalQty < 0 || req.TotalQty > 100000:
		writeError(w, http.StatusBadRequest, "BAD_QTY", "จำนวนต้องเป็นเลข 0 ขึ้นไป")
		return store.EquipmentInput{}, false
	case utf8.RuneCountInString(req.Description) > 2000:
		writeError(w, http.StatusBadRequest, "BAD_DESCRIPTION", "รายละเอียดยาวเกิน 2000 ตัวอักษร")
		return store.EquipmentInput{}, false
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "อื่นๆ"
	}
	if !store.ValidCategory(category) {
		writeError(w, http.StatusBadRequest, "BAD_CATEGORY", "หมวดหมู่ต้องเป็น กีฬา, ไอที, อุปกรณ์ถ่ายภาพ หรือ อื่นๆ")
		return store.EquipmentInput{}, false
	}
	imageURL := strings.TrimSpace(req.ImageURL)
	if imageURL != "" && (!strings.HasPrefix(imageURL, "https://") || utf8.RuneCountInString(imageURL) > 500) {
		writeError(w, http.StatusBadRequest, "BAD_IMAGE_URL", "ลิงก์รูปต้องขึ้นต้นด้วย https:// และยาวไม่เกิน 500 ตัวอักษร")
		return store.EquipmentInput{}, false
	}
	return store.EquipmentInput{
		Name:        name,
		Description: optional(req.Description),
		Category:    category,
		ImageURL:    optional(imageURL),
		TotalQty:    req.TotalQty,
	}, true
}

func (h *Handler) AdminCreateEquipment(w http.ResponseWriter, r *http.Request) {
	in, ok := parseEquipment(w, r)
	if !ok {
		return
	}
	e, err := h.Store.CreateEquipment(r.Context(), in)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) AdminUpdateEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	in, ok := parseEquipment(w, r)
	if !ok {
		return
	}
	e, err := h.Store.UpdateEquipment(r.Context(), id, in)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์นี้")
	case errors.Is(err, store.ErrStockTooLow):
		writeError(w, http.StatusConflict, "STOCK_TOO_LOW", "จำนวนทั้งหมดต้องไม่น้อยกว่าจำนวนที่กำลังถูกยืมอยู่")
	case err != nil:
		internal(w, err)
	default:
		writeJSON(w, http.StatusOK, e)
	}
}

func (h *Handler) AdminDeleteEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	err := h.Store.DeleteEquipment(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์นี้")
	case errors.Is(err, store.ErrHasActiveLoans):
		writeError(w, http.StatusConflict, "HAS_ACTIVE_LOANS", "ลบไม่ได้ ยังมีคำขอที่รออนุมัติหรือยังไม่คืนอุปกรณ์ (เมื่อลบสำเร็จ ระบบแค่ซ่อนอุปกรณ์ ประวัติการยืมยังอยู่)")
	case err != nil:
		internal(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
