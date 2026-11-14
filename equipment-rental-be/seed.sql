-- ข้อมูลอุปกรณ์เริ่มต้น 31 รายการ (ตรงกับตาราง equipment ใน Supabase ยกเว้นรายการทดสอบ id 35)
-- รันหลัง schema.sql ใน Supabase SQL Editor รันซ้ำได้: id ที่มีอยู่แล้วจะถูกข้าม ไม่ซ้ำ ไม่ทับข้อมูลเดิม
-- available_qty = total_qty ทุกรายการ (เริ่มต้นโดยยังไม่มีใครยืม)
-- id 24 (โต๊ะพับอเนกประสงค์) เป็น is_active = false คือถูกซ่อนจากระบบ

INSERT INTO equipment (id, name, description, category, image_url, total_qty, available_qty, is_active) VALUES
  (1, 'ลูกฟุตบอล เบอร์ 5', 'ลูกหนังเย็บ ใช้แข่งขันได้', 'กีฬา', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQqS6sqMloCPrIWNHTB7UyJsHTC1af8bddsgvUQ0nmLNg&s=10', 6, 6, true),
  (2, 'ไม้แบดมินตัน Yonex (คู่)', 'พร้อมลูกขนไก่ 3 ลูก', 'กีฬา', 'https://www.supersports.co.th/cdn/shop/files/YO136SP748BATH-0.jpg?v=1759460966', 4, 4, true),
  (3, 'โน้ตบุ๊ก Dell Latitude', 'RAM 16GB พร้อมที่ชาร์จ', 'ไอที', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQdb8V-SbTjnz9_OEEeDDjdoe4FVKV605DUPd8gHwhDcOk1jWDDft9S260&s=10', 2, 2, true),
  (4, 'โปรเจคเตอร์ Epson EB-X06', 'ความสว่าง 3600 ลูเมน พร้อมสาย HDMI', 'ไอที', 'https://shop-image.readyplanet.com/QkaTBPfEP0aAHX4mLuuacBKnSeA=/544e164ca0be426f9bd433c0cf722618', 2, 2, true),
  (5, 'กล้อง DSLR Canon EOS 90D', 'พร้อมเลนส์คิท 18-135mm แบตเตอรี่ 2 ก้อน', 'อุปกรณ์ถ่ายภาพ', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQv3ZzA2eCpGIvLz_ejumkTkbRk9id_skLDV31TvrOOn_mbf7CYkpUtthc&s=10', 10, 10, true),
  (6, 'ขาตั้งกล้อง Manfrotto', 'สูงสุด 160 ซม. รับน้ำหนัก 5 กก.', 'อุปกรณ์ถ่ายภาพ', 'https://www.ec-mall.com/media/catalog/product/M/a/Manfrotto-Element-MII-Aluminium-tripod-Kit-1.jpg', 5, 5, true),
  (7, 'ไมโครโฟนไร้สาย Shure', 'ชุดคู่ พร้อมตัวรับสัญญาณ', 'อื่นๆ', 'https://www.musicarms.net/wp-content/uploads/2026/10/Shure-SLXD4D-System.jpeg', 4, 4, true),
  (8, 'ลำโพงบลูทูธ JBL PartyBox', 'ใช้งานต่อเนื่อง 12 ชม.', 'อื่นๆ', 'https://www.musicarms.net/wp-content/uploads/2022/01/JBL-Partybox-110.jpg', 2, 2, true),
  (9, 'เต็นท์ 4 คน', 'พร้อมถุงเก็บและสมอบก', 'อื่นๆ', 'https://f.btwcdn.com/store-41406/product/7db6e5f1-1b0a-9f68-9f64-6968a546c89f.jpg', 4, 4, true),
  (10, 'กิมบอลมือถือ', 'กันสั่น 3 แกน', 'อุปกรณ์ถ่ายภาพ', 'https://store.lnwgadget.com/media/catalog/product/cache/1/image/850x850/9df78eab33525d08d6e5fb8d27136e95/0/2/02_15.jpg', 4, 4, true),
  (11, 'เว็บแคม HD', '1080p พร้อมไมโครโฟนในตัว', 'ไอที', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQ_y6dE2lSoSrZAHjYpZR-PuK2TBtsRzK6nSiIjMU0-k4DC4zm-gTahNN4&s=10', 5, 5, true),
  (12, 'คีย์บอร์ดไร้สาย', 'คีย์ไทย-อังกฤษ', 'ไอที', 'https://www.jib.co.th/img_master/product/original/2025081416440579411_1.jpg', 10, 10, true),
  (13, 'แฟลช Speedlite', 'แฟลชติดหัวกล้อง พร้อมถ่าน', 'อุปกรณ์ถ่ายภาพ', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT1M_BpJGmzsDi7QGvamcnEEymqtVFKWn0bKvAZOfMgDQuQs7uIcF0C1bc&s=10', 3, 3, true),
  (14, 'เครื่องขยายเสียงพกพา', 'พร้อมไมค์สาย ใช้ได้ 8 ชม.', 'อื่นๆ', 'https://audiocity.co.th/pub/media/catalog/product/cache/765fdf86a9e550514b9df31691b36e7f/d/e/deccon-pws-210u--1200-_1.jpg', 3, 3, true),
  (15, 'ไฟ LED ถ่ายวิดีโอ', 'ปรับความสว่างและอุณหภูมิสีได้ พร้อมขาตั้ง', 'อุปกรณ์ถ่ายภาพ', 'https://down-th.img.susercontent.com/file/th-11134207-81zti-mj86gwri8yrlb4', 4, 4, true),
  (16, 'ลูกตะกร้อ', 'ลูกหวายสังเคราะห์', 'กีฬา', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcStDFeQsh90ta65mES9cer4QujcljWMT1nlK4TCJ1jvsA&s=10', 6, 6, true),
  (17, 'เสื่อโยคะ', 'หนา 8 มม. กันลื่น', 'กีฬา', 'https://contents.mediadecathlon.com/p2939785/k$84ca93a383ab82ea901a517640dc2590/%E0%B9%80%E0%B8%AA%E0%B8%B7%E0%B9%88%E0%B8%AD%E0%B9%82%E0%B8%A2%E0%B8%84%E0%B8%B0%E0%B8%97%E0%B8%B1%E0%B9%88%E0%B8%A7%E0%B9%84%E0%B8%9B%E0%B8%82%E0%B8%99%E0%B8%B2%E0%B8%94-%E0%B8%8B%E0%B8%A1-%E0%B8%8B%E0%B8%A1-%E0%B8%A1%E0%B8%A1-%E0%B8%A3%E0%B8%B8%E0%B9%88%E0%B8%99-%E0%B8%AA%E0%B8%B5%E0%B8%A1%E0%B9%88%E0%B8%A7%E0%B8%87%E0%B9%81%E0%B8%94%E0%B8%87-8644104.jpg?f=1920x0&format=auto', 8, 8, true),
  (18, 'กล้อง Mirrorless Sony A6400', 'พร้อมเลนส์คิทและแบตเตอรี่สำรอง', 'อุปกรณ์ถ่ายภาพ', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcSlHwy1uvKr_vnJYszWKZFJWHERWwoW6P-FWzkIFOPLWw&s=10', 2, 2, true),
  (19, 'การ์ดความจำ SD 128GB', 'ความเร็วสูง U3', 'อุปกรณ์ถ่ายภาพ', 'https://www.zoomcamera.net/wp-content/uploads/2022/07/%E0%B9%82%E0%B8%84%E0%B8%A3%E0%B8%87%E0%B8%81%E0%B8%B2%E0%B8%A3%E0%B9%83%E0%B8%AB%E0%B8%A1%E0%B9%88-2-2.jpg', 8, 8, true),
  (20, 'จอมอนิเตอร์ 24 นิ้ว', 'Full HD พร้อมสาย HDMI', 'ไอที', 'https://media-cdn.bnn.in.th/75945/LG-MONITOR-24MK600M-B-ATM-Notab-01-square_medium.jpg', 4, 4, true),
  (21, 'ชุดเทเบิลเทนนิส', 'ไม้ 2 อัน ลูก 3 ลูก พร้อมตาข่าย', 'กีฬา', 'https://img.ltwebstatic.com/v4/j/spmp/2025/09/17/09/17580908456e4da22d887a007f124f826ee6ee29e8_thumbnail_750x999.jpg', 3, 3, true),
  (22, 'ไม้เทนนิส', 'พร้อมกระเป๋า', 'กีฬา', 'https://img.ltwebstatic.com/v4/j/spmp/2026/03/29/89/1774788139055f2f6e7c6fbe05d55781c3868d4b58_thumbnail_405x552.jpg', 3, 3, true),
  (23, 'เลนส์ 50mm f/1.8', 'เลนส์ฟิกซ์ ถ่ายพอร์ตเทรต', 'อุปกรณ์ถ่ายภาพ', 'https://www.digital2home.com/wp-content/uploads/2019/11/1459251297_1242613.jpg', 3, 3, true),
  (24, 'โต๊ะพับอเนกประสงค์', 'ขนาด 180x75 ซม.', 'อื่นๆ', 'https://st.bigc-cs.com/cdn-cgi/image/format=webp,quality=90/public/media/catalog/product/95/88/8859714403995/8859714403995_1-20250428105028-.jpg', 10, 10, false),
  (25, 'แท็บเล็ต iPad', 'พร้อมเคสและที่ชาร์จ', 'ไอที', 'https://shopee.co.th/blog/wp-content/uploads/2021/07/Apple-iPad-10.2-1.jpg', 4, 4, true),
  (26, 'โน้ตบุ๊ก Lenovo ThinkPad', 'RAM 16GB SSD 512GB พร้อมที่ชาร์จ', 'ไอที', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTO6n7J3RKlg-AJL5wFAB7V3VWYkEC4nOp_Czjnw50dysqkV2JWumn2pOwI&s=10', 3, 3, true),
  (27, 'ปลั๊กพ่วง 5 เมตร', '4 ช่อง มีสวิตช์เปิด-ปิด', 'อื่นๆ', 'https://pim-cdn0.ofm.co.th/products/large/8004004.jpg?u=202408080501', 10, 10, true),
  (28, 'ลูกบาสเกตบอล เบอร์ 7', 'ยางคุณภาพสูง ใช้ได้ทั้งในและนอกร่ม', 'กีฬา', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcQk0B-INi4_NkPqqLdkLGqgPGC0sXDVNJc3G05l0Y7ZQXgA1xZPqMVVEVpv&s=10', 5, 5, true),
  (29, 'ลูกวอลเลย์บอล', 'มาตรฐานแข่งขัน', 'กีฬา', 'https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTWm9kBpAY7RIlZevKn1TgtHpQU9DpmQ-IZ1SzWcrpv2A&s=10', 5, 5, true),
  (30, 'เมาส์ไร้สาย', 'พร้อมถ่านและตัวรับสัญญาณ', 'ไอที', 'https://www.istudio.store/cdn/shop/files/m240_silent_mouse_bt.001.jpg?v=1759726785&width=823', 10, 10, true),
  (31, 'iphone duo', 'ไอฟง หมอลำ', 'ไอที', 'https://jffmqsowezwgnnjmtwrj.supabase.co/storage/v1/object/public/equipment-images/iphoneduostarwhitepdpimageposition1jpg09e15de2-9d74-43d1-9c71-0b6ebf1ccd74.png', 10, 10, true)
ON CONFLICT (id) DO NOTHING;

-- ปรับตัวนับ id ให้ต่อจาก 31 (ไม่ทำ จะชน id ตอนเพิ่มอุปกรณ์ใหม่ผ่านหน้า Admin)
SELECT setval(pg_get_serial_sequence('equipment', 'id'), (SELECT max(id) FROM equipment));

-- ทำให้ตัวเองเป็น admin: สมัครสมาชิกผ่านเว็บก่อน แล้วรัน (เปลี่ยนอีเมล)
-- UPDATE members SET role = 'admin' WHERE lower(email) = 'you@example.com';