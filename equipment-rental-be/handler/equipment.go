package handler

import (
	"errors"
	"net/http"
	"strings"

	"equipmentrental/store"
)

// ListEquipment เปิดให้ทุกคนดู (ไม่ต้อง login) ?q= ค้นจากชื่อ/หมวดหมู่
// แต่ละชิ้นมี available = true (พร้อมให้ยืม) / false (ถูกยืมหมดแล้ว)
func (h *Handler) ListEquipment(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.ListEquipment(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
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
