-- รันใน Supabase SQL Editor สำหรับฐานข้อมูลเดิมที่สร้างไว้แล้ว (รันซ้ำได้ ข้อมูลเดิมไม่หาย)
-- เพิ่มหมวดหมู่ 4 ประเภท: กีฬา, ไอที, อุปกรณ์ถ่ายภาพ, อื่นๆ

ALTER TABLE equipment ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT true;

-- แปลงหมวดหมู่เดิมที่เป็นข้อความอิสระ ให้เข้า 4 หมวด
UPDATE equipment SET category = 'อุปกรณ์ถ่ายภาพ' WHERE category IN ('กล้อง', 'อุปกรณ์เสริมกล้อง');
UPDATE equipment SET category = 'ไอที'            WHERE category IN ('คอมพิวเตอร์', 'โสตทัศนูปกรณ์');
UPDATE equipment SET category = 'อื่นๆ'
 WHERE category IS NULL OR category NOT IN ('กีฬา', 'ไอที', 'อุปกรณ์ถ่ายภาพ', 'อื่นๆ');

ALTER TABLE equipment ALTER COLUMN category SET DEFAULT 'อื่นๆ';
ALTER TABLE equipment ALTER COLUMN category SET NOT NULL;
ALTER TABLE equipment DROP CONSTRAINT IF EXISTS equipment_category_check;
ALTER TABLE equipment ADD CONSTRAINT equipment_category_check
    CHECK (category IN ('กีฬา', 'ไอที', 'อุปกรณ์ถ่ายภาพ', 'อื่นๆ'));
