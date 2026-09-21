# GishFlow SMS Gateway (self-built Android app)

โค้ดทั้งหมดในโฟลเดอร์นี้เขียนขึ้นเอง ไม่ได้ใช้แอปหรือ APK จากที่อื่น และไม่พึ่ง library
ภายนอกใดๆ ในการรับ HTTP request หรือส่ง SMS (ใช้แค่ `java.net.ServerSocket` และ
`android.telephony.SmsManager` ที่มากับ Android SDK ตรงๆ)

พูดคุยกับ backend ของ GishFlow ด้วย contract เดียวกับที่ `models.SendSMSViaGateway`
(`models/sms.go`) เรียกอยู่แล้ว:

```
POST /message
Authorization: Basic base64(username:password)
Content-Type: application/json

{"message": "...", "phoneNumbers": ["+66812345678"]}
```

## Build โดยไม่ใช้ Android Studio

Build ผ่าน command line ล้วนๆ ได้ (ตามที่ตั้งไว้ในเครื่องนี้แล้ว):

```
cd android-sms-gateway
./gradlew.bat assembleDebug
```

ได้ไฟล์ APK ที่ `app/build/outputs/apk/debug/app-debug.apk` เอาไฟล์นี้ไปติดตั้งบนมือถือ
(ก็อปปี้ผ่าน USB/คลาวด์ แล้วเปิดไฟล์ติดตั้งบนมือถือ ต้องเปิด "อนุญาตติดตั้งจากแหล่งที่ไม่รู้จัก"
ก่อน หรือใช้ `adb install app-debug.apk` ถ้าต่อสาย USB + เปิด USB debugging)

## ตั้งค่าในแอป

1. เปิดแอป → กด Allow ตอนขอ permission ส่ง SMS
2. ตั้ง Port/Username/Password (ค่า default: port `8080`, user `gishflow`,
   pass `changeme` — **ควรเปลี่ยนก่อนใช้จริง**)
3. กด **Start** — ด้านบนจะโชว์ `Running - http://<ip>:<port>` ใช้ IP นี้ตอนตั้งค่าใน GishFlow
4. มือถือ + เครื่องที่รัน GishFlow ต้องอยู่ **Wi-Fi/LAN เดียวกัน**

## ตั้งค่าใน GishFlow

Sending Profiles → New Profile → Interface Type = SMS → **Gateway URL** =
`http://<ip>:<port>` ที่แอปโชว์ → Username/Password ให้ตรงกับที่ตั้งในแอป → ปิด
Mock mode → Save

## หมายเหตุ

- แอปต้องเปิดค้างไว้ (หน้าจอไม่ดับ) ระหว่างส่งแคมเปญ - ยังไม่ได้ทำเป็น foreground
  service ในเวอร์ชันนี้
- ข้อความยาวเกิน 160 ตัวอักษร (หรือสั้นกว่านั้นถ้ามีอักษรไทย เพราะนับเป็น Unicode SMS)
  จะถูกแบ่งส่งหลาย part อัตโนมัติผ่าน `divideMessage`/`sendMultipartTextMessage`
- ส่งหาเบอร์ที่ยินยอมให้ทดสอบเท่านั้น
