# ส่ง SMS ด้วยมือถือตัวเอง (Self-hosted SMS Gateway)

วิธีส่ง SMS ของ GishFlow โดย**ไม่ใช้ผู้ให้บริการ/ API** — ใช้มือถือ Android + ซิมของเราเองเป็น gateway
(ต้นทุน = ค่าส่ง SMS ของซิม ไม่มีค่า API)

> ⚠️ ส่งหา **เบอร์ผู้เข้าร่วมที่ยินยอมเท่านั้น** (authorized simulation / consented recipients)

GishFlow จะส่งแบบนี้อัตโนมัติเมื่อ **Host** ของ SMS profile เป็น URL ขึ้นต้น `http` (ถ้าไม่ใช่ → ใช้ ThaiBulkSMS ตามเดิม)

---

## 📱 ฝั่งมือถือ Android

### 1. ลงแอป SMS Gateway
- ลงแอป **"SMS Gateway"** (โดย capcom6)
  - Play Store: ค้น **"SMS Gateway"**
  - หรือ APK: https://github.com/capcom6/android-sms-gateway/releases
- เปิดแอป → **อนุญาต permission ส่ง SMS**

### 2. เปิดโหมด Local Server
- เลือกโหมด **"Local Server"** (ไม่ใช่ Cloud)
- แอปจะโชว์ค่า 3 อย่าง — **จดไว้**:
  - **URL / IP:Port** เช่น `http://192.168.1.45:8080`
  - **Username**
  - **Password**

### 3. เงื่อนไขเครือข่าย (สำคัญ)
- มือถือ กับ **คอมที่รัน GishFlow ต้องอยู่ Wi-Fi/LAN เดียวกัน**
- เปิดแอปค้างไว้ + อย่าให้เครื่องหลับ ตอนจะส่ง

---

## 💻 ฝั่ง GishFlow

### 4. สร้าง Sending Profile
Sending Profiles → **+ New Profile**

| ช่อง | ค่า |
|---|---|
| Interface Type | `SMS` |
| **Host** | URL จากแอป เช่น `http://192.168.1.45:8080` |
| **Username** | username จากแอป |
| **Password** | password จากแอป |
| SMTP From | เว้นว่างได้ |
| Ignore Certificate Errors | **ไม่ต้องติ๊ก** |

→ Save

### 5. Group + เบอร์
Users & Groups → New Group → เพิ่ม target ใส่**เบอร์จริงในช่อง Phone** แบบ `+66`
(เช่น `0812345678` → **`+66812345678`**)

### 6. เทมเพลต SMS
Email Template ใหม่ → กรอกช่อง **Text** (SMS ใช้ข้อความ text):
```
แจ้งเตือน: ยืนยันบัญชีของคุณที่ {{.URL}}
```

### 7. ยิง Campaign
Campaign: template นี้ · Sending Profile ที่สร้าง · group เบอร์ · URL phishing ของคุณ → **Launch**
→ เช็กมือถือปลายทางว่าได้รับ SMS

---

## เบื้องหลัง (สำหรับ dev)
- GishFlow POST ไปที่ `Host` + `/message` พร้อม HTTP Basic auth (Username/Password) และ body
  `{"message": "...", "phoneNumbers": ["+66..."]}` — ตรงกับ Local Server API ของแอป capcom6
- โค้ด: `models.SendSMSViaGateway` (เรียกจาก `worker.sendViaSMS` เมื่อ Host เป็น http)

## แก้ปัญหาที่เจอบ่อย
- **ไม่ได้รับ SMS / error timeout** → มือถือกับคอมอยู่คนละ Wi-Fi, แอปปิด, หรือ firewall บล็อก → เช็ก IP ให้ตรง + ping ได้
- **401 Unauthorized** → Username/Password ไม่ตรงกับที่แอปโชว์
- **เบอร์ไม่ถูกส่ง** → ใช้รูปแบบ `+66xxxxxxxxx`
