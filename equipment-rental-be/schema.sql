-- รันใน Supabase: SQL Editor > New query > วางทั้งไฟล์ > Run
-- (รันซ้ำได้ถ้าต้องการล้างข้อมูลทดสอบ: เอาคอมเมนต์ DROP ด้านล่างออก)
-- DROP TABLE IF EXISTS borrow_requests, equipment, members CASCADE;

CREATE TABLE members (
    id             BIGSERIAL     PRIMARY KEY,
    email          VARCHAR(255)  NOT NULL,
    password_hash  VARCHAR(255)  NOT NULL,   -- bcrypt ห้ามเก็บรหัสผ่านตรง ๆ
    display_name   VARCHAR(50)   NOT NULL CHECK (char_length(btrim(display_name)) >= 1),
    role           VARCHAR(10)   NOT NULL DEFAULT 'member' CHECK (role IN ('guest', 'member', 'admin')),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_members_email ON members (lower(email));

CREATE TABLE equipment (
    id             BIGSERIAL     PRIMARY KEY,
    name           VARCHAR(100)  NOT NULL CHECK (char_length(btrim(name)) >= 1),
    description    TEXT,
    category       VARCHAR(50),
    image_url      VARCHAR(500),
    total_qty      INT           NOT NULL CHECK (total_qty >= 0),      -- จำนวนทั้งหมดในคลัง
    available_qty  INT           NOT NULL CHECK (available_qty >= 0),  -- จำนวนที่ยังว่างให้ยืม
    is_active      BOOLEAN       NOT NULL DEFAULT true,                -- false = เลิกใช้งาน/ซ่อนจากระบบ (ปุ่มลบของ admin ทำแค่นี้ ไม่ลบแถวจริง)
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CONSTRAINT ck_equipment_stock CHECK (available_qty <= total_qty)
);

-- สถานะ: pending (รออนุมัติ) -> approved (อนุมัติ/กำลังยืม, หักสต็อก) -> returned (คืนแล้ว, คืนสต็อก)
--        pending -> rejected (ปฏิเสธ, ไม่แตะสต็อก)
CREATE TABLE borrow_requests (
    id            BIGSERIAL    PRIMARY KEY,
    -- RESTRICT: ห้ามลบสมาชิก/อุปกรณ์ที่มีประวัติการยืม เพื่อเก็บประวัติให้ครบ (ซ่อนอุปกรณ์ด้วย is_active แทน)
    member_id     BIGINT       NOT NULL REFERENCES members(id)   ON DELETE RESTRICT,
    equipment_id  BIGINT       NOT NULL REFERENCES equipment(id) ON DELETE RESTRICT,
    quantity      INT          NOT NULL DEFAULT 1 CHECK (quantity >= 1),
    borrow_date   DATE         NOT NULL,
    return_date   DATE         NOT NULL,
    status        VARCHAR(10)  NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'approved', 'rejected', 'returned')),
    approved_by   BIGINT       REFERENCES members(id) ON DELETE SET NULL,
    returned_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT ck_borrow_dates CHECK (return_date >= borrow_date)
);
CREATE INDEX idx_borrow_member    ON borrow_requests (member_id, created_at DESC);
CREATE INDEX idx_borrow_status    ON borrow_requests (status, created_at DESC);
CREATE INDEX idx_borrow_equipment ON borrow_requests (equipment_id);

-- Supabase เปิด REST API (PostgREST) ให้ตารางใน schema public อัตโนมัติ
-- เราใช้ Go API ต่อ DB ตรง ๆ จึงเปิด RLS โดยไม่ใส่ policy = กันคนยิงตารางผ่าน anon key (กัน password_hash รั่ว)
-- (role postgres ที่ Go ใช้ ข้าม RLS ได้ จึงไม่กระทบ)
ALTER TABLE members         ENABLE ROW LEVEL SECURITY;
ALTER TABLE equipment       ENABLE ROW LEVEL SECURITY;
ALTER TABLE borrow_requests ENABLE ROW LEVEL SECURITY;
