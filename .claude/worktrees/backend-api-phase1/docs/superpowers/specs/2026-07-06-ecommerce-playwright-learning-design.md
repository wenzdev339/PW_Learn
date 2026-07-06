# PW_Learn — Production-Grade E-commerce Playwright Learning Project

**Status:** Approved for planning
**Date:** 2026-07-06

## 1. เป้าหมายของโปรเจค

โปรเจคนี้สร้างขึ้นเพื่อ **เรียนรู้ Playwright ในระดับ Production** โดยมี Playwright Test
Framework เป็นแกนหลักของการเรียนรู้ ส่วน Backend API และ Frontend Website ทำหน้าที่เป็น
"ระบบเป้าหมาย" (System Under Test) ที่ต้องสมจริงและมีฟีเจอร์ครบพอที่จะสาธิตเทคนิคการ
ทดสอบระดับ production ได้ครบทุกมิติ

**Success criteria:**
- มี e-commerce web app ที่ใช้งานได้จริงครบวงจร (auth → catalog → cart → checkout →
  payment mock → order history → reviews → admin panel)
- Backend สามารถจำลอง network condition (latency, error, timeout, rate-limit, flaky
  retry) ได้ทั้งแบบ ambient (สุ่ม) และแบบ deterministic (บังคับผ่าน header สำหรับเทส)
- มี Playwright test suite ที่ครอบคลุม: E2E UI, API testing, visual regression,
  accessibility, performance, (component testing เป็น stretch), cross-browser/device,
  auth reuse, test data management, sharding/parallelization, CI/CD บน GitHub Actions,
  reporting (HTML + Allure)
- ทุกส่วนรันผ่าน CI (GitHub Actions) ได้จริง ไม่ใช่แค่รันบนเครื่อง local

## 2. Non-Goals (ขอบเขตที่ตัดออกโดยตั้งใจ)

- **ไม่ผูก payment gateway จริง** (เช่น Stripe test mode) — ใช้ mock payment ภายในระบบเอง
  เพื่อไม่ต้องพึ่ง external service/secret ในโปรเจคเรียนรู้
- **ไม่ส่งอีเมลจริง** — reset password token จะ log ออก console/log แทนการส่งอีเมลจริง
- **ไม่มี i18n/multi-currency**
- **ไม่มี real image upload/storage** — ใช้ placeholder image URL ที่ seed มาให้
- **ไม่ทำ Kubernetes หรือ production deployment** — ใช้ Docker เฉพาะเพื่อให้ Playwright
  browser environment ใน CI สม่ำเสมอเท่านั้น
- **ไม่ใช้เครื่องมือ load testing แยก** (k6/Artillery) — แทนที่ด้วย network condition
  simulation middleware ใน backend ตามที่ตกลงกันไว้ในขั้น brainstorming

## 3. สถาปัตยกรรมภาพรวม

Monorepo เดียว, **Backend เป็น Go module แยกต่างหาก** (ไม่ได้อยู่ใน pnpm workspace เพราะ
คนละ toolchain) ส่วน Frontend และ Playwright Framework ยังเป็น **TypeScript** ทั้งคู่และ
ใช้ pnpm workspace ร่วมกัน

```
pw-learn/
├── apps/
│   ├── backend/          # Go + Gin + GORM + PostgreSQL (Go module แยก, go.mod ของตัวเอง)
│   ├── frontend/         # Next.js (App Router) + TypeScript + Tailwind
│   └── e2e/              # Playwright Test Framework (TypeScript)
├── packages/
│   └── shared-types/     # Product/Order/User TS types ใช้ร่วมกันระหว่าง frontend/e2e
├── docs/superpowers/specs/
├── docker-compose.yml     # PostgreSQL (dev + test database) สำหรับ backend
├── .github/workflows/
│   ├── backend-ci.yml    # รันเทส Go backend
│   ├── pw-smoke.yml      # รันเทส @smoke ตอน PR (เร็ว)
│   └── pw-regression.yml # รันเทสเต็มชุด ตอน push main + nightly cron
├── pnpm-workspace.yaml    # ครอบคลุมเฉพาะ apps/frontend, apps/e2e, packages/*
├── package.json           # workspace root (TypeScript ฝั่งเดียว)
└── README.md
```

`packages/shared-types` ลด drift ระหว่าง response shape ของ backend กับสิ่งที่ frontend/e2e
คาดหวัง — เป็น pattern ที่ใช้จริงใน production monorepo (นิยามมือใน TypeScript ให้ตรงกับ
JSON ที่ Go backend คืนมา เนื่องจากคนละภาษาจึงไม่มี codegen อัตโนมัติข้ามภาษาในสโคปนี้)

## 4. Backend API

**Stack:** Go + Gin (HTTP framework) + GORM (ORM) + PostgreSQL (ผ่าน Docker), ทดสอบด้วย
`testing` + `testify` + `net/http/httptest`

### 4.1 Data Model (GORM structs)

| Model | ฟิลด์สำคัญ |
|---|---|
| `User` | ID (uuid), Email (unique), PasswordHash, Name, Role (`CUSTOMER`\|`ADMIN`), CreatedAt |
| `Address` | ID, UserID, Label, Line1, Line2 (nullable), City, PostalCode, Country, IsDefault |
| `Category` | ID, Name, Slug (unique) |
| `Product` | ID, Name, Slug (unique), Description, Price (int, สตางค์), Stock, Images (`pq.StringArray` หรือ `jsonb`), CategoryID, CreatedAt |
| `Review` | ID, ProductID, UserID, Rating (1-5), Comment, CreatedAt |
| `Cart` / `CartItem` | Cart 1:1 ต่อ user, CartItem อ้าง ProductID + Quantity |
| `Order` / `OrderItem` | Status: `PENDING`\|`PAID`\|`SHIPPED`\|`DELIVERED`\|`CANCELLED`; OrderItem เก็บ PriceAtPurchase (snapshot ราคา ณ เวลาสั่งซื้อ) |
| `Payment` | OrderID (unique), Method, Status (`SUCCESS`\|`FAILED`), MockTransactionID |

ใช้ UUID (`github.com/google/uuid`) เป็น primary key ทุกตาราง แทน auto-increment int
เพื่อไม่ให้ ID เดาได้ (เหมาะกับ URL/API ที่ expose ID ตรงๆ)

### 4.2 API Endpoints (`/api/v1`)

| Method | Path | Auth | หมายเหตุ |
|---|---|---|---|
| POST | `/auth/register` | - | |
| POST | `/auth/login` | - | คืน access token (15 นาที) + httpOnly refresh cookie (7 วัน) |
| POST | `/auth/refresh` | refresh cookie | |
| POST | `/auth/logout` | - | |
| POST | `/auth/forgot-password` | - | log reset token แทนการส่งอีเมล |
| POST | `/auth/reset-password` | - | |
| GET | `/products` | - | search, filter (category/price), sort, pagination |
| GET | `/products/:slug` | - | |
| GET | `/categories` | - | |
| GET/POST | `/reviews` | POST ต้อง login | |
| GET/POST/PATCH/DELETE | `/cart/items` | login | |
| POST | `/checkout` | login | สร้าง Order จาก Cart |
| GET | `/orders`, `/orders/:id` | login | |
| POST | `/payments/mock` | login | เลขบัตร `4242...`=success, `0000...`=decline |
| CRUD | `/admin/products` | admin | |
| GET/PATCH | `/admin/orders` | admin | |
| GET | `/health` | - | readiness check |
| POST | `/test/reset` | - (เฉพาะ `APP_ENV=test` หรือ `ALLOW_TEST_RESET=true`) | รีเซ็ต DB กลับ seed data |

Auth guard middleware แยก 2 ระดับ: `requireAuth` (ต้อง login) และ `requireAdmin` (ต้องเป็น
role ADMIN) — คืน 401/403 ตามลำดับ เพื่อให้ API test เช็ค contract ได้ชัดเจน

### 4.3 Network Condition Simulation Middleware

หัวใจสำคัญของโปรเจคที่เชื่อม backend เข้ากับสิ่งที่ Playwright จะทดสอบ ลำดับ Gin middleware
(`router.Use(...)` ตามลำดับนี้):

1. **`TestScenarioMiddleware`** (ใช้ได้เฉพาะเมื่อ `APP_ENV != "production"`) — อ่าน header
   `X-Test-Scenario` แล้ว override พฤติกรรมแบบ deterministic:
   - `slow` → หน่วงเพิ่ม 3000ms ก่อนตอบ (`time.Sleep`)
   - `error` → ตอบ 500 ทันทีพร้อม JSON error body แล้ว `c.Abort()`
   - `timeout` → ไม่ตอบเลย (ไม่เรียก `c.Next()`/ไม่ write response) จำลอง request ค้าง
   - `rate-limited` → ตอบ 429 ทันที
   - `flaky:N` → ใช้ key จาก header `X-Test-Run-Id` + path เก็บ counter ใน
     `sync.Map` (in-memory, thread-safe) — fail (500) N ครั้งแรก แล้วครั้งถัดไป
     success เพื่อพิสูจน์ retry logic แบบ reproducible
2. **`AmbientLatencyMiddleware`** — หน่วงตาม env `SIMULATE_LATENCY_MS` +
   random(0, `SIMULATE_LATENCY_JITTER_MS`) ใช้ตอนไม่มี header override
3. **`AmbientErrorMiddleware`** — สุ่มตอบ 500 ตามความน่าจะเป็นจาก env
   `SIMULATE_ERROR_RATE` (default 0 = ปิด)
4. **`RateLimiterMiddleware`** (token-bucket ง่ายๆ ด้วย `golang.org/x/time/rate` ต่อ IP
   หรือ global) — ambient rate limit ตาม `SIMULATE_RATE_LIMIT` (default ปล่อยหลวม)

Header override (ข้อ 1) มีสิทธิ์เหนือกว่า ambient config เสมอ — ทำให้เทสที่ต้องการผลลัพธ์
แน่นอนไม่ต้องพึ่งดวงจาก ambient randomness

การส่ง header เข้า browser-driven request ทำผ่าน
`page.context().setExtraHTTPHeaders({ 'X-Test-Scenario': 'flaky:2' })` แบบ scoped ต่อเทส
แล้ว reset หลังจบ

### 4.4 Seed & Reset Strategy

แพ็กเกจ `internal/seed` มีฟังก์ชัน `Run(db *gorm.DB) error` สร้างข้อมูลคงที่
(deterministic) ทุกครั้ง — เรียกได้ทั้งจาก CLI (`go run ./cmd/seed`) และจาก
`/test/reset` handler:
- บัญชี `admin@example.com` / `Admin123!` (role ADMIN)
- บัญชี `customer@example.com` / `Customer123!` (role CUSTOMER)
- ~20 สินค้า กระจาย 4 categories, มีอย่างน้อย 1 ชิ้น stock=0 (สำหรับเทส out-of-stock UI)
  และราคาหลากหลาย (สำหรับเทส filter/sort)

`/test/reset` เรียก `TRUNCATE ... CASCADE` ทุกตาราง แล้วเรียก `internal/seed.Run` ซ้ำ —
Playwright `global-setup.ts` เรียก endpoint นี้ครั้งเดียวก่อนเริ่ม test run ทั้งหมด เพื่อ
isolation

### 4.5 API Documentation

OpenAPI ผ่าน `swaggo/swag` (annotate handler ด้วย Go comment แบบ `// @Summary ...`,
รัน `swag init` เพื่อ generate `docs/swagger.json`) เสิร์ฟผ่าน `gin-swagger` ที่
`/api-docs` — ใช้เป็น contract อ้างอิงสำหรับเขียน API test และทำให้โปรเจคดูสมจริงแบบ
production

## 5. Frontend (Next.js App Router)

**Stack:** Next.js (App Router) + TypeScript + Tailwind CSS, React Context สำหรับ
auth session + cart state, react-hook-form + zod สำหรับ form validation

### 5.1 Routes

| Route | หน้าที่ |
|---|---|
| `/` | Home: featured products, categories |
| `/products` | Catalog: search, filter (category/price), sort, pagination |
| `/products/[slug]` | รายละเอียดสินค้า + gallery + reviews (ดู/เขียน) |
| `/cart` | ตะกร้า: แก้จำนวน, ลบ, subtotal |
| `/checkout` | 3 ขั้นตอน: ที่อยู่จัดส่ง → ชำระเงิน (mock card form) → ยืนยันคำสั่งซื้อ |
| `/checkout/confirmation/[orderId]` | หน้ายืนยันคำสั่งซื้อ |
| `/account`, `/account/orders`, `/account/orders/[id]`, `/account/addresses` | โปรไฟล์/ประวัติ/ที่อยู่ |
| `/login`, `/register`, `/forgot-password`, `/reset-password` | Auth |
| `/admin`, `/admin/products`, `/admin/orders` | Dashboard, จัดการสินค้า/ออเดอร์ |

### 5.2 พฤติกรรมระดับ production ที่จงใจใส่ไว้เป็นเป้าให้ Playwright ทดสอบ

- **Loading skeleton** ระหว่างรอ API — จับคู่กับ backend latency simulation
- **Error boundary + retry UI** เมื่อ API fail — จับคู่กับ backend error simulation
- **Optimistic UI** สำหรับการแก้จำนวนสินค้าในตะกร้า
- **Toast notification** (success/error)
- **Protected routes**: `/account/*`, `/admin/*` redirect ไป `/login` ถ้าไม่ได้ login/ไม่ใช่ admin
- **`data-testid` convention**: `{feature}-{element}-{action?}` เช่น
  `data-testid="product-card-add-to-cart"`, `data-testid="checkout-step-shipping-submit"`
  — stable test hook ที่ตั้งใจออกแบบไว้ ไม่พึ่ง CSS/text selector ที่เปราะบาง
- **Semantic HTML + keyboard navigation** พื้นฐาน เพื่อให้ a11y test เจอปัญหาจริงได้เมื่อมีคน
  เผลอทำผิด (เป็น living example ไม่ใช่หน้าที่สมบูรณ์แบบ 100%)

## 6. Playwright Test Framework (แกนหลักของการเรียนรู้)

### 6.1 โครงสร้างโฟลเดอร์

```
apps/e2e/
├── playwright.config.ts
├── tests/
│   ├── e2e/{auth,catalog,cart-checkout,orders,reviews,admin}/
│   ├── api/
│   ├── visual/
│   ├── a11y/
│   └── component/            # stretch
├── pages/                     # Page Object Models
├── fixtures/
│   ├── auth.fixture.ts
│   ├── api.fixture.ts
│   ├── data-factory.fixture.ts
│   └── network-scenario.fixture.ts
├── utils/
├── global-setup.ts
├── global-teardown.ts
└── docker/Dockerfile
```

### 6.2 `playwright.config.ts` design

- `baseURL` จาก env `E2E_BASE_URL` (default `http://localhost:3000`), API base แยกจาก
  env `API_BASE_URL` (default `http://localhost:4000/api/v1`)
- **Projects:**
  - `setup` — รัน auth setup (login ผ่าน API, เซฟ storageState), ไม่ retry
  - `chromium-customer`, `chromium-admin` — depends on `setup`, ใช้ storageState ตาม role,
    รัน full regression suite
  - `firefox`, `webkit`, `mobile-chrome` (Pixel 7 viewport) — รันเฉพาะเทสที่ tag `@smoke`
    เพื่อประหยัดเวลา CI (เทรดออฟจริงที่ทีม production ต้องตัดสินใจ)
  - `api` — request-context only ไม่มี browser
  - `visual` — chromium, fixed viewport, ปิด animation
  - `a11y` — chromium
- **Reporter:** `[['html'], ['list'], ['allure-playwright']]`
- **Retries:** 0 local, 2 บน CI
- **Trace/video/screenshot:** `retain-on-failure` / `retain-on-failure` / `only-on-failure`

### 6.3 Fixtures

- **`auth.fixture.ts`** — extend `test` ด้วย `customerPage`/`adminPage` fixture (โหลด
  storageState ที่เซฟไว้) สำหรับเทสที่ต้องใช้ 2 role ในไฟล์เดียว
- **`api.fixture.ts`** — `apiClient` fixture ห่อ Playwright `request` context พร้อม helper
  (`createProduct`, `createUser`, `getOrder` ฯลฯ) ใช้ทั้งใน API test และใน UI test ที่ต้อง
  setup ข้อมูลแบบ just-in-time
- **`data-factory.fixture.ts`** — ใช้ `@faker-js/faker` สร้าง `buildProduct()`,
  `buildUser()`, `buildAddress()` เป็น plain object แล้วส่งต่อให้ `apiClient` persist
- **`network-scenario.fixture.ts`** — `simulateBackend(scenario, opts)` ตั้ง
  `X-Test-Scenario` header แบบ scoped ต่อเทส เป็นสะพานเชื่อมตรงจาก backend design (4.3)
  ไปสู่การพิสูจน์ UI behavior

### 6.4 Test Data Management Strategy

ผสม 3 แบบ:
1. **Full reset** ผ่าน `/test/reset` ใน `global-setup.ts` ก่อนรันทั้งชุด
2. **Just-in-time creation** ผ่าน `apiClient` + data factory ก่อนแต่ละเทสที่ต้องการข้อมูล
   เฉพาะ (เช่น สินค้าที่ stock=0)
3. **Unique identifiers ต่อ worker** (เช่น ต่อท้าย email/slug ด้วย `test.info().workerIndex`)
   เพื่อให้รัน parallel ได้โดยไม่ชนกัน

### 6.5 ประเภทเทสที่ครอบคลุม

| ประเภท | ตัวอย่างที่พิสูจน์ |
|---|---|
| **E2E UI (POM)** | สมัคร→เลือกสินค้า→ใส่ตะกร้า→checkout→จ่ายเงิน→เห็นคำยืนยัน; admin เพิ่มสินค้า→เห็นใน catalog |
| **API** | status code, schema/contract, auth guard 401/403, pagination ถูกต้อง |
| **Visual regression** | หน้า Home/Catalog/Detail/Cart/Checkout, mask ส่วนที่เปลี่ยนทุกครั้ง (timestamp/orderId) |
| **Accessibility** | รัน axe-core ทุกหน้าใน project `a11y`, assert zero critical/serious violation |
| **Performance** | trace + Web Vitals (LCP/CLS) ผ่าน CDP บนหน้า catalog/detail, เทียบ budget threshold |
| **Component (stretch)** | Playwright Component Testing สำหรับ `<ProductCard>`, `<CartSummary>` แยกจากทั้งแอป |

### 6.6 Cross-browser / Sharding / CI

- Full regression รันบน `chromium-customer`/`chromium-admin`; `@smoke` subset รันซ้ำบน
  firefox/webkit/mobile-chrome
- CI ใช้ `--shard 1/4 .. 4/4` แล้ว merge report ก่อนอัปโหลด artifact
- **`pw-smoke.yml`**: trigger บน PR, รันเฉพาะ `@smoke` (~5 นาที)
- **`pw-regression.yml`**: trigger บน push main + nightly cron, รันเต็มชุดพร้อม matrix
  (browser × shard)
- ขั้นตอน CI: checkout → setup pnpm + setup Go → install deps → start PostgreSQL
  (service container) → run Go migrations (`AutoMigrate`) + seed →
  start backend/frontend background + wait-on health check → install playwright
  browsers (cache ตาม version) → รันเทสตาม grep tag → merge report → upload artifact

### 6.7 Tagging Convention

Tag ในชื่อเทส: `@smoke` (critical path เร็ว), `@regression`, `@api`, `@visual`, `@a11y`,
`@admin` — ใช้ `--grep`/`--grep-invert` คุม pipeline แบบ staged

### 6.8 Tooling

ESLint + `eslint-plugin-playwright` (กฎ เช่น no-wait-for-timeout, expect-expect),
TypeScript strict mode, Prettier

## 7. ลำดับการสร้าง (Build Order)

การ implement จะแบ่งเป็น 3 phase ตามลำดับพึ่งพา (แต่ละ phase มี implementation plan
แยกของตัวเอง ต่อจาก spec นี้):

1. **Phase 1 — Backend**: schema, seed/reset, auth, product/cart/order/payment CRUD,
   network simulation middleware, Swagger docs
2. **Phase 2 — Frontend**: ทุกหน้าเชื่อมกับ backend จริง, production UX patterns
   (loading/error/optimistic UI), data-testid
3. **Phase 3 — Playwright Framework**: config + fixtures + POM ก่อน แล้วไล่ตามประเภทเทส
   (E2E smoke → API → E2E regression เต็ม → visual → a11y → performance → CI/CD wiring
   → component testing เป็น stretch สุดท้าย)

## 8. ความเสี่ยง/ข้อควรพิจารณาในอนาคต

- **`timeout` scenario** อาจทำให้ CI job ค้างถ้าไม่ตั้ง client-side timeout — ต้องกำหนด
  `expect.timeout`/`actionTimeout` ที่เหมาะสมใน config เพื่อไม่ให้เทสค้างเกินไป
- **Visual regression บน CI vs local** อาจ render ต่างกันเล็กน้อย (font rendering) —
  ต้องรัน baseline screenshot จากใน Docker image เดียวกับที่ CI ใช้ ไม่ใช่จากเครื่อง local
- **Allure report บน GitHub Actions** ต้องมี step แยกสำหรับ generate + publish (เช่น
  artifact หรือ GitHub Pages) — รายละเอียดจะอยู่ใน implementation plan ของ Phase 3
