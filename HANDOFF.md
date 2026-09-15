# GishFlow — คู่มือส่งมอบ (Handoff)

GishFlow เป็น fork ของ **Gophish** (phishing-simulation framework เขียนด้วย Go) ที่เพิ่มความสามารถสำหรับงานวิจัย/ฝึกอบรม:

- **Multi-channel**: Email, SMS (ThaiBulkSMS), LINE (Official Account webhook เก็บ userId อัตโนมัติ), QR code
- **AI**: สร้าง pretext + quiz อัตโนมัติ (OpenAI-compatible เช่น KKU IntelSphere)
- **Analytics**: NIST Phish Scale, engagement timing, risk score + trend + time-decay, export PDF
- **Teachable moment**: quiz + completion tracking
- **Deliverability**: Amazon SES + SPF/DKIM/DMARC, Cloudflare tunnel (URL ถาวร)
- เริ่มย้าย frontend เป็น **React ทีละหน้า** (islands) ใน `static/react/`

> เอกสารนี้ทำให้ทำงานต่อได้โดย **ระบบไม่พัง** — อ่านหัวข้อ **"กฎห้ามแตะ"** ก่อนแก้โค้ด

---

## 1. สิ่งที่ต้องมี (Prerequisites)

- **Docker Desktop** (จำเป็น — ใช้ build/run ทั้งหมด เครื่องไม่ต้องลง Go/Node เอง)
- **Git**
- (ทางเลือก) VS Code

> ทำไมต้อง Docker: ตัว sqlite driver ต้องใช้ CGO/gcc — เลย build ในคอนเทนเนอร์ `golang:1.15.2` ไม่ build บนเครื่องตรงๆ

---

## 2. รันครั้งแรก (Local dev)

```bash
# 1) เตรียม config ของตัวเอง (คัดลอกจาก default — ไฟล์นี้ถูก gitignore แล้ว ปลอดภัย)
cp config.json run-config.json

# 2) สร้าง network + รัน mailhog (ตัวดักอีเมลตอน dev)
docker network create gishflow-net
docker run -d --name mailhog --network gishflow-net -p 8025:8025 mailhog/mailhog

# 3) build + run ตัวแอป (mount source ที่ /app, เก็บ DB ใน volume)
docker run -d --name gishflow --network gishflow-net \
  -p 3333:3333 -p 8080:80 \
  -v "$(pwd)":/app -v gishflow-data:/data -v gishflow-gocache:/go \
  -v "$(pwd)/run-config.json":/run/run-config.json \
  -w /app golang:1.15.2 \
  sh -c "go build -o /tmp/gophish . && exec /tmp/gophish --config /run/run-config.json"

# ดู log (ครั้งแรกจะพิมพ์ user/password ของ admin ให้)
docker logs -f gishflow
```

- **Admin UI**: http://localhost:3333 (ล็อกอินด้วย user/password จาก log ครั้งแรก)
- **Phishing server**: http://localhost:8080
- **MailHog inbox**: http://localhost:8025

> Windows: ถ้า `$(pwd)` มีปัญหาเรื่อง path ให้ใช้ PowerShell แล้วแทนด้วย path เต็ม เช่น `-v C:\path\to\gophish:/app`

---

## 3. แก้โค้ดแล้วเห็นผลยังไง

| แก้อะไร | ต้องทำอะไร |
|---|---|
| **Go** (`.go`) | rebuild → `docker restart gishflow` |
| **Templates** (`templates/*.html`) | `docker restart gishflow` (โหลด template ตอน start) |
| **CSS / JS ต้นฉบับที่เสิร์ฟตรง** (`static/js/src/app/*` ที่ template โหลดจาก `/js/src/...`) | เสิร์ฟสด — แค่ **Ctrl+F5** |
| **JS ที่ build เป็น dist** (`static/js/src/app/*` ที่ template โหลดจาก `/js/dist/...`) | ต้อง **build gulp** (ดูข้อ 4) |
| **React islands** (`static/react/`) | build ด้วย esbuild (ดูข้อ 5) |

---

## 4. Build JS (gulp) — ระวัง "gulp churn"

```bash
docker run --rm -v "$(pwd)":/app -w /app node:16 sh -c "npm install && npx gulp scripts"
```

⚠️ **gulp จะ rebuild ไฟล์ dist ทุกตัว** ทำให้ไฟล์ที่คุณไม่ได้แก้ก็ถูกแตะไปด้วย (churn)
→ หลัง build ให้ **revert ไฟล์ dist ที่ไม่เกี่ยว** เก็บไว้เฉพาะไฟล์ที่แก้จริง:

```bash
git checkout -- static/js/dist/app/<ไฟล์ที่ไม่ได้แก้>.min.js
```

> `npm install` อาจไปแก้ `yarn.lock` / สร้าง `package-lock.json` — repo ใช้ yarn ให้ `git checkout -- yarn.lock` และลบ `package-lock.json` ทิ้ง

---

## 5. React islands (`static/react/`)

toolchain แยกเด็ดขาดจาก gulp (มี `package.json` ของตัวเอง) — แปลง frontend ทีละหน้า

```bash
docker run --rm -v "$(pwd)":/app -w /app/static/react node:20 sh -c "yarn install && yarn build"
```

- เขียน component ที่ `static/react/src/<page>.jsx` (reuse global `window.query(path,method,{},false)` เรียก API เดิม)
- build แล้วได้ไฟล์เดี่ยวที่ `static/js/dist/app/<page>.island.js`
- ใน template ให้เก็บ grid `col` เดิมไว้ แล้ววาง `<div id="react-...">` ข้างใน + โหลด island bundle ใน `{{define "scripts"}}`
- **ทำแล้ว 1 หน้า**: LINE Contacts (`static/react/src/line_contacts.jsx`) — ใช้เป็นแม่แบบได้

---

## 6. ก่อน push — อย่าทำ CI พัง 🚨

CI (GitHub Actions) รันตอน push ขึ้น **master** ด้วย **Go 1.21–1.23** มี 3 ขั้น: **Build → Format (gofmt) → Test**

- **Format เข้มมาก**: `.go` ทุกไฟล์ต้อง gofmt สะอาด (`diff -u <(echo -n) <(gofmt -d .)`)
- ⚠️ **กับดัก Windows CRLF**: working tree เป็น CRLF ทำให้ `gofmt -l .` ในเครื่อง **ขึ้นทุกไฟล์ (บังผลจริง)** — เช็กแบบตรงกับ CI (ใช้เวอร์ชัน LF ที่ git เก็บ):

```bash
rm -rf /tmp/ghfmt && mkdir -p /tmp/ghfmt && git archive HEAD | tar -x -C /tmp/ghfmt
docker run --rm -v /tmp/ghfmt:/app -w /app golang:1.15.2 gofmt -l .   # ว่าง = สะอาด
```

- แก้ให้สะอาด: `gofmt -w <ไฟล์>` (รันในคอนเทนเนอร์ golang ได้)
- เช็ก build/test ก่อน push: `docker run --rm -v "$(pwd)":/app -v gishflow-gocache:/go -w /app golang:1.15.2 sh -c "go build ./... && go test ./..."`

---

## 7. กฎห้ามแตะ (ไม่งั้นพัง) ⛔

1. **ห้ามแก้ import path `github.com/gophish/gophish`** — เป็น module path build จะพังทันที
2. **ห้าม commit credential ใดๆ** — SES key, LINE channel token, ThaiBulkSMS key, KKU api_key, AWS key ทั้งหมดอยู่ใน `run-config.json` (local, gitignore แล้ว) หรือกรอกผ่าน UI เท่านั้น
3. **DB migrations = append-only** — เพิ่มไฟล์ใหม่ใน `db/db_sqlite3/` **และ** `db/db_mysql/` (goose) ห้ามแก้ของเก่า, ใส่ `DEFAULT ''` กัน NULL error
4. เพิ่ม event/status ใหม่ ต้องเพิ่ม mapping ใน **ทั้ง** `static/js/src/app/dashboard.js` **และ** `campaign_results.js` (ไม่งั้นหน้าเว็บ crash `undefined.color`)

---

## 8. ตั้ง credential ของตัวเอง (external services)

บริการภายนอกทั้งหมด**ผูกกับบัญชีของผู้พัฒนาเดิม** — คนทำต่อควรตั้งของตัวเอง:

| บริการ | ตั้งที่ไหน |
|---|---|
| **AI** (pretext/quiz) | `ai` block ใน `run-config.json`: `base_url`, `api_key`, `model`, `provider` (OpenAI-compatible) |
| **Email จริง** (Amazon SES) | Sending Profile: host `email-smtp.<region>.amazonaws.com:587`, SMTP user/pass; ตั้ง SPF/DKIM/DMARC ที่ DNS |
| **LINE** | Sending Profile แบบ LINE + channel access token; ตั้ง webhook URL → `/line/webhook` |
| **SMS** | Sending Profile แบบ SMS + ThaiBulkSMS key/secret |
| **Public URL** | Cloudflare Tunnel หรือ ngrok ชี้ไป `localhost:8080` |

> ตอน dev ใช้ **MailHog** ส่งอีเมลได้เลยโดยไม่ต้องตั้ง SES (Sending Profile host = `mailhog:1025`)

---

## 9. Deploy จริง

ดู **`deploy/DEPLOY.md`** — มี `docker-compose.yml` (แอป + Caddy auto-HTTPS) + `.env.example` พร้อมใช้บน VPS

---

## 10. แผนผังโค้ด (จุดสำคัญ)

```
controllers/         HTTP handlers — controllers/api/ = REST API, phish.go = ฝั่งเหยื่อ (tracking, webhook)
models/              ธุรกิจหลัก + DB — ai.go, phishscale.go, analytics.go, risk.go, line*.go, result.go
worker/              คิวส่ง + dispatch ตามช่องทาง (interface_type)
db/db_sqlite3/, db/db_mysql/   goose migrations (append-only)
templates/           Go html/template (server-rendered)
static/js/src/app/   JavaScript ต้นฉบับ (build เป็น dist ด้วย gulp)
static/react/        React islands (toolchain แยก, esbuild)
deploy/              ชุด deploy production (compose + Caddy)
```

---

## 11. สรุป Gotchas (ที่เคยทำพังมาแล้ว)

- CI พังเพราะ gofmt + CRLF บัง → เช็กด้วย `git archive` (ข้อ 6)
- แก้ dist JS แล้วไม่เห็นผล → browser cache, กด **Ctrl+F5**
- gulp churn → revert dist ที่ไม่เกี่ยว (ข้อ 4)
- SES ส่งไม่ออก 554 → From ต้องเป็นโดเมนที่ verify (แก้ที่ **Envelope Sender** ของ Email Template ด้วย ไม่ใช่แค่ Sending Profile)
- Gmail "Opened" ไม่ขึ้น → เมลอยู่ Spam (Gmail บล็อกรูป) ต้องเข้า inbox

---

_มีคำถามเรื่องสถาปัตยกรรม/ประวัติการแก้ ดู git log และคอมเมนต์ในโค้ด — เขียนไว้ค่อนข้างละเอียด_
