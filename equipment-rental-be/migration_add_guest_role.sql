-- รันใน Supabase SQL Editor สำหรับฐานข้อมูลเดิมที่สร้าง schema.sql ไปแล้ว
-- เพิ่ม role guest โดยไม่ลบ/สร้างตารางใหม่ และไม่กระทบข้อมูลสมาชิกเดิม
ALTER TABLE members
    DROP CONSTRAINT IF EXISTS members_role_check;

ALTER TABLE members
    ADD CONSTRAINT members_role_check
    CHECK (role IN ('guest', 'member', 'admin'));

-- ค่า DEFAULT ยังคงเป็น member เพื่อให้การสมัครแบบเดิมไม่เปลี่ยนพฤติกรรม
ALTER TABLE members
    ALTER COLUMN role SET DEFAULT 'member';
