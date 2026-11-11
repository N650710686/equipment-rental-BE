package handler

import (
	"errors"
	"net/http"
	"strings"

	"equipmentrental/store"
)

// ListEquipment เปิดให้ทุกคนดู (ไม่ต้อง login)
// ?q= ค้นจากชื่อ · ?category=กีฬา|ไอที|อุปกรณ์ถ่ายภาพ|อื่นๆ กรองหมวดหมู่
// แต่ละชิ้นมี available = true (พร้อมให้ยืม) / false (ถูกยืมหมดแล้ว)
func (h *Handler) ListEquipment(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && !store.ValidCategory(category) {
		writeError(w, http.StatusBadRequest, "BAD_CATEGORY", "หมวดหมู่ไม่ถูกต้อง")
		return
	}
	items, err := h.Store.ListEquipment(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")), category)
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// Categories รายการหมวดหมู่ให้ FE ทำ dropdown / แท็บกรอง
func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": store.Categories})
}

func (h *Handler) GetEquipment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขอุปกรณ์ต้องเป็นตัวเลข")
		return
	}
	e, err := h.Store.GetEquipment(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "EQUIPMENT_NOT_FOUND", "ไม่พบอุปกรณ์นี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}
