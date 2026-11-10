-- ข้อมูลอุปกรณ์ตัวอย่าง (รันหลัง schema.sql)
INSERT INTO equipment (name, description, category, total_qty, available_qty) VALUES
 ('กล้อง DSLR Canon EOS 90D', 'พร้อมเลนส์คิท 18-135mm แบตเตอรี่ 2 ก้อน', 'กล้อง', 3, 3),
 ('โปรเจคเตอร์ Epson EB-X06', 'ความสว่าง 3600 ลูเมน พร้อมสาย HDMI', 'โสตทัศนูปกรณ์', 2, 2),
 ('ไมโครโฟนไร้สาย Shure', 'ชุดคู่ พร้อมตัวรับสัญญาณ', 'เครื่องเสียง', 4, 4),
 ('ขาตั้งกล้อง Manfrotto', 'สูงสุด 160 ซม. รับน้ำหนัก 5 กก.', 'อุปกรณ์เสริมกล้อง', 5, 5),
 ('โน้ตบุ๊ก Dell Latitude', 'RAM 16GB พร้อมที่ชาร์จ', 'คอมพิวเตอร์', 1, 1),
 ('ลำโพงบลูทูธ JBL PartyBox', 'ใช้งานต่อเนื่อง 12 ชม.', 'เครื่องเสียง', 2, 2);

-- ทำให้ตัวเองเป็น admin: สมัครสมาชิกผ่าน API ก่อน แล้วรัน (เปลี่ยนอีเมล)
-- UPDATE members SET role = 'admin' WHERE lower(email) = 'you@example.com';
