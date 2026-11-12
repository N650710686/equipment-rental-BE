package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxImageBytes = 50 << 20 // 50MB

// ImageStorage อัปโหลดรูปไป Supabase Storage (bucket แบบ public) ผ่าน REST API
// ใช้ service key ที่อยู่ฝั่ง BE เท่านั้น ห้ามส่งให้ FE
type ImageStorage struct {
	BaseURL string // https://<ref>.supabase.co
	Key     string // service_role key (หรือ secret key)
	Bucket  string // เช่น equipment-images
	client  *http.Client
}

func NewImageStorage(baseURL, key, bucket string) *ImageStorage {
	return &ImageStorage{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Key:     key,
		Bucket:  bucket,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// Upload ส่งไฟล์ขึ้น bucket แล้วคืนลิงก์สาธารณะของรูป
func (s *ImageStorage) Upload(name, contentType string, data []byte) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.BaseURL, s.Bucket, name)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", s.Key)
	// key แบบ JWT (service_role, ขึ้นต้น eyJ) ส่งเป็น Bearer ได้ ส่วน sb_secret_... ใช้ header apikey อย่างเดียว
	if strings.HasPrefix(s.Key, "eyJ") {
		req.Header.Set("Authorization", "Bearer "+s.Key)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Cache-Control", "max-age=31536000")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("supabase storage %d: %s", resp.StatusCode, body)
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.BaseURL, s.Bucket, name), nil
}

var imageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

// AdminUploadImage รับไฟล์รูป (multipart field ชื่อ "image") -> คืน {"url": "..."}
// FE เอา url ไปใส่ใน imageUrl ตอนสร้าง/แก้อุปกรณ์ (POST/PUT /api/admin/equipment)
func (h *Handler) AdminUploadImage(w http.ResponseWriter, r *http.Request) {
	if h.Images == nil {
		writeError(w, http.StatusServiceUnavailable, "UPLOAD_NOT_CONFIGURED", "ยังไม่ได้ตั้งค่าที่เก็บรูป (SUPABASE_URL / SUPABASE_SERVICE_KEY)")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageBytes+1<<20)
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "NO_IMAGE", "ต้องแนบไฟล์รูปในฟิลด์ชื่อ image (multipart/form-data) และขนาดไม่เกิน 5MB")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "NO_IMAGE", "อ่านไฟล์ไม่สำเร็จ")
		return
	}
	if len(data) > maxImageBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "IMAGE_TOO_LARGE", "ไฟล์ใหญ่เกิน 5MB")
		return
	}
	// ดูชนิดไฟล์จากเนื้อหาจริง ไม่เชื่อนามสกุลหรือ Content-Type ที่ client ส่งมา
	contentType := http.DetectContentType(data)
	ext, ok := imageExt[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_IMAGE_TYPE", "รองรับเฉพาะไฟล์ JPG, PNG, WEBP, GIF")
		return
	}
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		internal(w, err)
		return
	}
	name := hex.EncodeToString(buf) + ext // ตั้งชื่อใหม่เอง กันชื่อซ้ำ/ชื่อแปลก

	url, err := h.Images.Upload(name, contentType, data)
	if err != nil {
		log.Printf("upload image: %v", err)
		writeError(w, http.StatusBadGateway, "UPLOAD_FAILED", "อัปโหลดรูปไม่สำเร็จ ลองใหม่อีกครั้ง")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}
