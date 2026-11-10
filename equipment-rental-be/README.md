# Equipment Rental & Inventory — Backend (Go)

BE ของระบบ "คลังและยืม-คืนอุปกรณ์" ดัดแปลงจากโครงสร้าง MovieHub ใน lab
(`net/http` + `pgx` + JWT + bcrypt) · DB = Supabase (PostgreSQL) · Deploy = Render · FE = Vercel (ทีมอื่น)

```
Vercel (FE) ──HTTPS + Bearer token──▶ Render (Go API) ──DATABASE_URL──▶ Supabase (Postgres)
```

> BE มี 2 ส่วนเสมอ: **โค้ด API (Go)** ที่ deploy บน Render และ **ฐานข้อมูล** ที่อยู่บน Supabase
> (`schema.sql` คือตารางทั้งหมด) Render เก็บข้อมูลเองไม่ได้ จึงต้องต่อ Supabase ผ่าน `DATABASE_URL`

## 1. เริ่มใน GitHub Codespaces
1. push โฟลเดอร์นี้ขึ้น GitHub repo → **Code ▸ Codespaces ▸ Create codespace on main** (จะเปิด VS Code บนเว็บ)
2. ถ้า Codespace ไม่มี Go ให้ใช้ template/devcontainer ภาษา Go แล้วเปิด terminal: `go version`
3. `cp .env.example .env` แล้วกรอกค่า (ขั้นตอนที่ 2)
4. `go mod tidy && go run .` → เปิดพอร์ต 8080 (Codespaces จะ forward ให้) ทดสอบ `GET /api/health`

## 2. ตั้งค่า Supabase
1. สร้างโปรเจกต์ที่ supabase.com แล้วจดรหัสผ่านฐานข้อมูลไว้
2. **SQL Editor** → วาง `schema.sql` → Run → วาง `seed.sql` → Run
3. **Connect / Database settings** → คัดลอก connection string แบบ **Session pooler** ใส่ `DATABASE_URL`
   (อย่าใช้ Direct connection: เป็น IPv6 ซึ่ง Render ต่อไม่ได้) ใน `.env` ใส่ `JWT_SECRET` ด้วย: `openssl rand -hex 32`
4. สมัครสมาชิกผ่าน API 1 บัญชี แล้วเลื่อนเป็น admin ใน SQL Editor:
   `UPDATE members SET role='admin' WHERE lower(email)='you@example.com';` แล้ว login ใหม่เพื่อให้ตั๋วมี role admin

## 3. Deploy บน Render
New ▸ **Web Service** ▸ เลือก repo
- Runtime: **Go** · Build Command: `go build -o app .` · Start Command: `./app`
- (ถ้า BE อยู่ในโฟลเดอร์ย่อย ให้ตั้ง Root Directory)
- Environment: `DATABASE_URL`, `JWT_SECRET`, `ALLOWED_ORIGINS` (ไม่ต้องตั้ง `PORT`)
- Health Check Path: `/api/health`
- แพลนฟรีจะหลับเมื่อไม่มีคนใช้ ครั้งแรกหลังหลับอาจตอบช้า ~1 นาที

## 4. ประสานกับ FE (Vercel)
- ส่ง URL ของ Render (เช่น `https://xxx.onrender.com`) ให้ทีม FE ใส่เป็น API base URL ใน env ของ Vercel
- เอา URL ของ Vercel มาใส่ `ALLOWED_ORIGINS` บน Render (คั่น comma, ไม่มี `/` ท้าย) ไม่งั้นติด CORS
  ตอนพัฒนาใน Codespaces ใส่ origin ของ FE ด้วย เช่น `https://<codespace>-5173.app.github.dev`
- FE ส่ง `Authorization: Bearer <token>` ที่ได้จาก login · ตั๋วอายุ 1 ชั่วโมง
- ปุ่ม "ขอยืม": ถ้าไม่มี token ให้ FE เด้งไปหน้า Login (ถ้ายิง API ไปจะได้ `401 NO_TOKEN`)

## API
Error ทุกตัวหน้าตาเดียวกัน: `{"error":{"code":"...","message":"..."}}`

| Method | Path | สิทธิ์ | หมายเหตุ |
|---|---|---|---|
| GET | `/api/health` | ทุกคน | ใช้กับ Render |
| POST | `/api/auth/register` | ทุกคน | `{email, password, displayName}` |
| POST | `/api/auth/login` | ทุกคน | → `{token, member}` |
| GET | `/api/equipment?q=` | ทุกคน | แต่ละชิ้นมี `available` (true=พร้อมให้ยืม / false=ถูกยืมหมดแล้ว), `availableQty`, `totalQty` |
| GET | `/api/equipment/{id}` | ทุกคน | รายละเอียดอุปกรณ์ |
| GET | `/api/me` | member | ข้อมูลส่วนตัว (Profile) |
| PUT | `/api/me/password` | member | `{currentPassword, newPassword}` |
| GET | `/api/me/borrow-requests` | member | ประวัติการยืมของตัวเอง |
| POST | `/api/borrow-requests` | member | `{equipmentId, quantity?, borrowDate, returnDate}` วันที่แบบ `YYYY-MM-DD` |
| GET | `/api/admin/stats` | admin | ตัวเลข Dashboard |
| POST | `/api/admin/equipment` | admin | `{name, description, category, imageUrl, totalQty}` |
| PUT | `/api/admin/equipment/{id}` | admin | แก้ข้อมูล/จำนวน (ของว่างปรับตามส่วนต่างของ totalQty) |
| DELETE | `/api/admin/equipment/{id}` | admin | ลบไม่ได้ถ้ายังมีคำขอ pending/approved (409) |
| GET | `/api/admin/borrow-requests?status=` | admin | pending / approved / rejected / returned |
| PUT | `/api/admin/borrow-requests/{id}/approve` | admin | pending → approved **และหักสต็อก** |
| PUT | `/api/admin/borrow-requests/{id}/reject` | admin | pending → rejected |
| PUT | `/api/admin/borrow-requests/{id}/return` | admin | approved → returned **และคืนสต็อก** |

## ตรรกะสต็อก (จุดที่ควรอธิบายได้)
- ส่งคำขอยืม → สถานะ `pending` ยัง**ไม่หัก**สต็อก (ถ้าของหมดแล้วจะส่งไม่ได้: 409 `OUT_OF_STOCK`)
- admin อนุมัติ → หัก `available_qty` และเปลี่ยนสถานะใน **transaction เดียว** + ล็อกแถว (`FOR UPDATE`) กัน admin กดพร้อมกันแล้วสต็อกติดลบ
- admin รับคืน → บวก `available_qty` กลับ และบันทึก `returned_at`
- DB มี CHECK กันซ้ำอีกชั้น: `0 <= available_qty <= total_qty`, `return_date >= borrow_date`

## ลองยิงด้วย curl
```bash
curl -X POST localhost:8080/api/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password123"}'
curl localhost:8080/api/equipment
curl -X POST localhost:8080/api/borrow-requests -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"equipmentId":1,"quantity":1,"borrowDate":"2026-11-15","returnDate":"2026-11-18"}'
```

## ความปลอดภัย
- `.env` อยู่ใน `.gitignore` ห้าม commit · รหัสผ่านเก็บเป็น bcrypt · SQL ทุกคำสั่งส่งค่าผ่าน `$1,$2` (กัน SQL injection)
- ตาราง Supabase เปิด RLS ไว้ (ไม่มี policy) เพื่อกันการเข้าถึงตรงผ่าน anon key
