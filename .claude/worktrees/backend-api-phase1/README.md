# PW_Learn — E-commerce Backend (Phase 1)

Mock e-commerce backend API built with **Go + Gin + GORM + PostgreSQL**, designed as the
target system for a future Playwright test-automation learning project (Frontend and
Playwright phases come later).

## Quick start

```bash
# 1. Start PostgreSQL (dev + test databases)
docker compose up -d

# 2. Configure environment
cd apps/backend
cp .env.example .env    # already done if you're reading this after setup

# 3. Install deps, seed the dev database, and run the server
go mod tidy
go run ./cmd/seed        # seeds ~20 products, 4 categories, admin/customer accounts
go run ./cmd/server      # starts on http://localhost:4000
```

Health check: `curl http://localhost:4000/health`

## Seeded accounts

| Email | Password | Role |
|---|---|---|
| `admin@example.com` | `Admin123!` | ADMIN |
| `customer@example.com` | `Customer123!` | CUSTOMER |

## Mock payment

`POST /api/v1/payments/mock` — card numbers starting with `4242` succeed, anything else
is declined.

## API overview (`/api/v1`)

- `auth`: register, login, refresh, logout, forgot-password, reset-password
- `products`, `categories`: browse/search/filter/sort/paginate
- `reviews`: list + create (auth required to post)
- `cart`: get/add/update/remove items (auth required)
- `addresses`: list/create shipping addresses (auth required)
- `checkout`: create an order from the cart (auth required)
- `orders`: list/get your orders (auth required)
- `payments/mock`: pay for a pending order
- `admin/*`: product CRUD + order management (admin role required)
- `/test/reset`: wipes and reseeds the database — blocked unless `APP_ENV=test` or
  `ALLOW_TEST_RESET=true`

## Network condition simulation

Every request can be forced into a specific behavior via the `X-Test-Scenario` header
(non-production only): `slow`, `error`, `timeout`, `rate-limited`, `flaky:N`. Ambient
(random) latency/error/rate-limit can also be configured via env vars — see
`apps/backend/.env.example`.

## Running the test suite

Requires the `pwlearn_test` database (already created by `docker/init-test-db.sql`).

```bash
cd apps/backend
go test ./...
```

## Design docs

Full design spec and implementation plan (including the originally-planned Frontend and
Playwright phases) are under `docs/superpowers/`.
