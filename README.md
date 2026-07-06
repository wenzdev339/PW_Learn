# PW_Learn — โปรเจคเรียนรู้ Playwright ระดับ Production

โปรเจคนี้จำลองระบบ e-commerce ขึ้นมาเพื่อใช้ **ฝึก Playwright ในระดับ production**
Backend (Go) คือ "ระบบเป้าหมาย" ที่สร้างให้สมจริง ส่วน Playwright (TypeScript) คือ
ตัวที่เราจะใช้เขียนเทสยิงเข้าไปทดสอบ

**สถานะปัจจุบัน:**
- ✅ Backend (Go + Gin + GORM + PostgreSQL) — ใช้งานได้ครบ
- ✅ ตัวอย่าง Playwright tests (ยิง API ตรงๆ เพราะยังไม่มี Frontend)
- ⏳ Frontend (Next.js) — ยังไม่ได้สร้าง

---

## 1. ภาพรวมระบบ (Architecture)

```mermaid
graph TB
    subgraph client["ฝั่งที่คุณใช้งาน"]
        PW["Playwright Tests<br/>(apps/e2e)"]
        Browser["เบราว์เซอร์ / curl"]
    end

    subgraph backend["Backend — apps/backend (Go)"]
        Router["Router<br/>(กำหนดเส้นทาง URL)"]
        MW["Middleware<br/>CORS, Auth, จำลองเครือข่าย"]
        Handler["Handler<br/>รับ request, ตอบ response"]
        Service["Service<br/>ตรรกะทางธุรกิจ"]
        Model["Model<br/>โครงสร้างข้อมูล"]
    end

    DB[("PostgreSQL<br/>(Docker)")]

    PW -->|HTTP request| Router
    Browser -->|HTTP request| Router
    Router --> MW --> Handler --> Service --> Model --> DB
```

**อ่านง่ายๆ:** คุณ (หรือ Playwright) ยิง HTTP request เข้ามาที่ Router → ผ่าน
Middleware (เช็คสิทธิ์, จำลองความหน่วง/error) → Handler รับค่าแล้วเรียก Service →
Service คุยกับฐานข้อมูลผ่าน Model แล้วส่งผลลัพธ์กลับเป็นทอดๆ

## 2. โครงสร้างโฟลเดอร์

```
PW_Learn/
├── apps/
│   ├── backend/          ← โค้ด Go ทั้งหมด (ดูหัวข้อ 3)
│   └── e2e/               ← Playwright tests (TypeScript)
├── docker-compose.yml     ← สั่งรัน PostgreSQL
├── docker/                ← script สร้าง database ทดสอบ
└── docs/superpowers/      ← เอกสารดีไซน์/แผนงานฉบับเต็ม (ละเอียดมาก)
```

## 3. ข้างในเดียว `apps/backend` มีอะไรบ้าง

```mermaid
graph LR
    subgraph entry["จุดเริ่มโปรแกรม"]
        Server["cmd/server<br/>รัน HTTP server"]
        Seed["cmd/seed<br/>ใส่ข้อมูลตัวอย่าง"]
    end

    subgraph core["internal/ — โค้ดหลัก"]
        Config["config<br/>อ่านค่าจาก .env"]
        Models["models<br/>โครงสร้างตาราง"]
        Router2["router<br/>รวมทุก route"]
        Feature["auth, products, cart,<br/>orders, payments, admin ...<br/>(1 โฟลเดอร์ต่อ 1 ฟีเจอร์)"]
        MW2["middleware<br/>auth guard, จำลองเครือข่าย"]
    end

    Server --> Router2
    Router2 --> Feature
    Router2 --> MW2
    Feature --> Models
    Seed --> Models
```

แต่ละฟีเจอร์ (เช่น `internal/cart/`) จะมี 2 ไฟล์เสมอ:
- **`handler.go`** — รับ HTTP request, แปลง JSON, เรียก service, ตอบกลับ
- **`service.go`** — ตรรกะจริง (เช่น เช็ค stock, คำนวณราคา) ไม่ยุ่งกับ HTTP เลย

แยกแบบนี้เพื่อให้ทดสอบ service ได้โดยไม่ต้องยิง HTTP จริง และอ่านง่ายว่าอะไรคือ
"กติกาธุรกิจ" กับอะไรคือ "ท่อส่งข้อมูล"

## 4. Flow การทำงานของ 1 request (พร้อม middleware จำลองเครือข่าย)

```mermaid
flowchart LR
    A["HTTP Request เข้ามา"] --> B["CORS"]
    B --> C{"มี header<br/>X-Test-Scenario?"}
    C -->|มี| D["บังคับพฤติกรรม:<br/>slow / error / timeout /<br/>rate-limited / flaky:N"]
    C -->|ไม่มี| E["Ambient Latency<br/>(หน่วงแบบสุ่มตาม .env)"]
    E --> F["Ambient Error<br/>(error แบบสุ่มตาม .env)"]
    F --> G["Rate Limiter"]
    G --> H{"route ต้อง login?"}
    H -->|ต้อง| I["RequireAuth<br/>เช็ค JWT token"]
    H -->|ไม่ต้อง| J["Handler"]
    I --> J
    J --> K["Service"] --> L[("Database")]
```

**นี่คือจุดเด่นของ backend นี้:** ปกติ backend จริงจะสุ่มช้า/error แบบควบคุมไม่ได้
ทำให้เทสไม่เสถียร (flaky test) แต่ backend นี้ให้คุณ**สั่งพฤติกรรมที่แน่นอน**ผ่าน
header `X-Test-Scenario` ได้ เช่น สั่งให้ fail 2 ครั้งแรกแล้วครั้งที่ 3 สำเร็จ
(`flaky:2`) เพื่อเขียนเทสพิสูจน์ retry logic แบบ reproducible 100%

## 5. Sequence Diagram — เดิน flow ซื้อของทั้งหมด

นี่คือสิ่งที่ไฟล์ `apps/e2e/tests/checkout-journey.spec.ts` ทดสอบ:

```mermaid
sequenceDiagram
    actor U as ผู้ใช้ (Playwright)
    participant Auth as /api/v1/auth
    participant Cart as /api/v1/cart
    participant Addr as /api/v1/addresses
    participant Out as /api/v1/checkout
    participant Pay as /api/v1/payments

    U->>Auth: POST /register {email, password}
    Auth-->>U: accessToken (ใช้แนบ Authorization header ต่อไป)

    U->>Cart: POST /cart/items {productId, quantity}
    Cart-->>U: cart item ที่เพิ่มแล้ว

    U->>Addr: POST /addresses {ที่อยู่จัดส่ง}
    Addr-->>U: address.id

    U->>Out: POST /checkout {shippingAddressId}
    Note over Out: ตัด stock, คำนวณราคารวม,<br/>ล้างตะกร้า (ทำใน transaction เดียว)
    Out-->>U: Order สถานะ PENDING

    U->>Pay: POST /payments/mock {orderId, cardNumber}
    Note over Pay: เลข 4242... = สำเร็จ<br/>เลขอื่น = ถูกปฏิเสธ
    Pay-->>U: Payment SUCCESS, Order → PAID
```

## 6. ER Diagram — ความสัมพันธ์ของข้อมูล

```mermaid
erDiagram
    USER ||--o{ ADDRESS : "มีได้หลายที่อยู่"
    USER ||--|| CART : "มี 1 ตะกร้า"
    USER ||--o{ ORDER : "สั่งซื้อได้หลายครั้ง"
    USER ||--o{ REVIEW : "เขียนรีวิวได้"
    CATEGORY ||--o{ PRODUCT : "มีหลายสินค้า"
    PRODUCT ||--o{ REVIEW : "ถูกรีวิวได้หลายคน"
    CART ||--o{ CART_ITEM : "มีหลายรายการ"
    PRODUCT }o--o{ CART_ITEM : "ถูกใส่ตะกร้า"
    ORDER ||--o{ ORDER_ITEM : "มีหลายรายการ"
    PRODUCT }o--o{ ORDER_ITEM : "ถูกสั่งซื้อ"
    ORDER ||--o| PAYMENT : "มี 1 การจ่ายเงิน"
```

จุดที่น่าสังเกต: **`OrderItem.priceAtPurchase`** เก็บราคา ณ ตอนสั่งซื้อแยกจาก
`Product.price` — เพราะถ้าร้านค้าขึ้นราคาสินค้าทีหลัง ออเดอร์เก่าต้องยังแสดงราคาเดิม
ที่ลูกค้าจ่ายจริง ไม่ใช่ราคาปัจจุบัน

## 7. วิธีรันทั้งระบบ

```bash
# 1) เปิด PostgreSQL
docker compose up -d

# 2) รัน backend
cd apps/backend
go run ./cmd/seed      # ใส่ข้อมูลตัวอย่าง (สินค้า 20 ชิ้น, บัญชีทดสอบ)
go run ./cmd/server    # เปิดที่ http://localhost:4000

# 3) รัน Playwright tests (เปิด terminal ใหม่)
cd apps/e2e
npm install
npx playwright test
npx playwright show-report   # ดูผลแบบ HTML
```

## 8. บัญชีทดสอบที่ seed ไว้ให้

| Email | Password | สิทธิ์ |
|---|---|---|
| `admin@example.com` | `Admin123!` | ADMIN |
| `customer@example.com` | `Customer123!` | CUSTOMER |

## 9. Endpoint ทั้งหมด (`/api/v1`)

| กลุ่ม | หน้าที่ |
|---|---|
| `auth` | register, login, refresh, logout, forgot/reset password |
| `products`, `categories` | ค้นหา/กรอง/เรียง/แบ่งหน้าสินค้า |
| `reviews` | ดู + เขียนรีวิว (เขียนต้อง login) |
| `cart` | ดู/เพิ่ม/แก้/ลบ ของในตะกร้า (ต้อง login) |
| `addresses` | ดู/สร้างที่อยู่จัดส่ง (ต้อง login) |
| `checkout` | สร้างออเดอร์จากตะกร้า (ต้อง login) |
| `orders` | ดูออเดอร์ของตัวเอง (ต้อง login) |
| `payments/mock` | จ่ายเงินจำลอง |
| `admin/*` | จัดการสินค้า/ออเดอร์ (ต้องเป็น admin) |
| `/test/reset` | ล้าง+ใส่ข้อมูลใหม่ (ใช้เฉพาะตอนเทส) |

## 10. รันชุดเทสของ backend เอง (Go)

```bash
cd apps/backend
go test ./...
```

## 11. เอกสารฉบับเต็ม

แผนงานและดีไซน์แบบละเอียดมาก (รวมส่วน Frontend/Playwright ที่ยังไม่ได้สร้าง) อยู่ที่
`docs/superpowers/specs/` และ `docs/superpowers/plans/`
