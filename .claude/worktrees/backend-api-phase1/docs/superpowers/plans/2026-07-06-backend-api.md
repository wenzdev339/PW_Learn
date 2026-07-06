# Backend API Implementation Plan (Phase 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the mock e-commerce Backend API (Go + Gin + GORM + PostgreSQL) that Frontend (Phase 2) and the Playwright Framework (Phase 3) will target, including deterministic/ambient network-condition simulation and a test-reset endpoint.

**Architecture:** Layered Gin app (`handler → service → gorm`) inside `apps/backend`, a standalone Go module (not part of the pnpm workspace — Frontend/Playwright stay TypeScript/pnpm, Backend is Go with its own `go.mod`). Each feature is a package under `internal/<feature>/` with its own `handler.go` + `service.go` + colocated `_test.go`. Cross-cutting concerns (auth guard, error handling, network simulation) live in `internal/middleware/` and `internal/apperror/`.

**Tech Stack:** Go 1.22+, Gin (HTTP framework), GORM + `gorm.io/driver/postgres` (ORM), PostgreSQL 16 via Docker Compose, `testing` + `testify` (assertions) + `net/http/httptest`, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `golang.org/x/time/rate`, `google/uuid`, `swaggo/swag` + `gin-swagger`.

## Global Constraints

- Backend stack: Go + Gin + GORM + PostgreSQL (spec §4). Backend is its own Go module under `apps/backend`, separate from the pnpm workspace.
- All API routes are prefixed `/api/v1` (spec §4.2).
- Auth: access token expires in 15 minutes; refresh token is an httpOnly cookie expiring in 7 days (spec §4.2).
- Seed accounts, verbatim: `admin@example.com` / `Admin123!` (role `ADMIN`), `customer@example.com` / `Customer123!` (role `CUSTOMER`) (spec §4.4).
- Seed data: ~20 products across 4 categories, at least one product with `stock = 0`, varying prices (spec §4.4).
- Network scenario header is exactly `X-Test-Scenario`, values `slow | error | timeout | rate-limited | flaky:N`; ambient env vars are `SIMULATE_LATENCY_MS`, `SIMULATE_LATENCY_JITTER_MS`, `SIMULATE_ERROR_RATE`, `SIMULATE_RATE_LIMIT`. Header override always wins over ambient behavior (spec §4.3).
- `/test/reset` must be blocked unless `APP_ENV=test` or `ALLOW_TEST_RESET=true` (spec §4.3, §4.4).
- Payment mock rule: card number starting with `4242` = success, otherwise decline (spec §4.2).
- Non-goals: no real payment gateway, no real email delivery — log the reset token instead (spec §2).
- Money is stored as `int` (smallest currency unit, e.g. satang/cents) (spec §4.1).
- Primary keys are UUID strings (`google/uuid`), generated in a GORM `BeforeCreate` hook — not database-generated defaults, to stay portable across Postgres versions (spec §4.1).

---

## File Structure

```
pw-learn/
├── docker-compose.yml                    # PostgreSQL 16, dev + test databases (new)
├── docker/init-test-db.sql               # creates the pwlearn_test database (new)
└── apps/backend/
    ├── go.mod / go.sum
    ├── .env.example
    ├── .env.test                         # reference only; Go tests use config.TestConfig()
    ├── cmd/
    │   ├── server/main.go                # process entrypoint
    │   └── seed/main.go                  # `go run ./cmd/seed` — seeds the dev database
    └── internal/
        ├── config/config.go              # Load() (from OS env) + TestConfig() (hardcoded test defaults)
        ├── apperror/apperror.go          # AppError type + RespondError/RespondValidationError helpers
        ├── db/db.go                      # Connect(cfg), AutoMigrate(db)
        ├── models/models.go              # GORM structs: User, Address, Category, Product, Review, Cart, CartItem, Order, OrderItem, Payment
        ├── token/token.go                 # JWT sign/verify (access + refresh)
        ├── password/password.go          # bcrypt hash/verify
        ├── seed/seed.go                   # CleanDatabase(db), Run(db) — deterministic seed data
        ├── dbtest/dbtest.go               # Connect() — test DB connection + AutoMigrate, used by every package's TestMain
        ├── middleware/
        │   ├── auth.go                   # RequireAuth, RequireAdmin
        │   ├── test_scenario.go
        │   ├── ambient_latency.go
        │   ├── ambient_error.go
        │   └── rate_limiter.go
        ├── router/router.go              # New(cfg, db) *gin.Engine — assembles middleware + route groups
        └── modules (one package each, handler.go + service.go + _test.go):
            ├── auth/
            ├── products/
            ├── reviews/
            ├── cart/
            ├── orders/       (checkout + orders)
            ├── payments/
            ├── admin/
            └── testutils/    (/test/reset)
```

Tests are colocated as `_test.go` in the same package they exercise (e.g. `internal/auth/handler_test.go`), run against a dedicated `pwlearn_test` Postgres database (same container as dev, separate database) via `config.TestConfig()`.

---

## Task List Overview

1. Docker Compose (Postgres) + Go module scaffold + Gin skeleton with `/health`
2. GORM models + `AutoMigrate` + `dbtest`/`CleanDatabase` test harness
3. Deterministic seed/reset logic (`seed.Run`)
4. Password hashing + JWT (`token`) packages
5. Auth middleware (`RequireAuth`, `RequireAdmin`)
6. Auth routes: register + login
7. Auth routes: refresh + logout
8. Auth routes: forgot-password + reset-password
9. Network simulation: deterministic `X-Test-Scenario`
10. Network simulation: ambient latency/error + rate limiter
11. Categories + Products routes
12. Reviews routes
13. Cart routes
14. Checkout + Orders routes
15. Payments (mock) routes
16. Admin routes (products CRUD + order status)
17. `/test/reset` route
18. Swagger/OpenAPI docs (`swaggo`)
19. Final wiring + full customer-journey integration test
20. Backend CI workflow (GitHub Actions, with Postgres service container)

Tasks are added to this file one at a time below — do not start implementing until a task's full detail has been appended.

---

### Task 1: Docker Compose (Postgres) + Go module scaffold + Gin skeleton (`/health`)

**Files:**
- Create: `docker-compose.yml` (repo root)
- Create: `docker/init-test-db.sql`
- Create: `apps/backend/go.mod`, `apps/backend/go.sum` (generated by the `go get`/`go mod tidy` commands below — do not hand-write these)
- Create: `apps/backend/.env.example`
- Create: `apps/backend/.env.test`
- Create: `apps/backend/internal/config/config.go`
- Create: `apps/backend/internal/apperror/apperror.go`
- Create: `apps/backend/internal/router/router.go`
- Create: `apps/backend/cmd/server/main.go`
- Test: `apps/backend/internal/router/router_test.go`

**Interfaces:**
- Produces: `config.Load() Config` (reads OS env, panics if a required var is missing), `config.TestConfig() Config` (fixed test values, no `.env` dependency) — every later task's tests call `config.TestConfig()`. `router.New(cfg config.Config, db *gorm.DB) *gin.Engine` — every later task registers its routes on the `gin.Engine` this returns. `apperror.New(statusCode int, code, message string) *apperror.AppError`, `apperror.RespondError(c *gin.Context, err error)`, `apperror.RespondValidationError(c *gin.Context, err error)` — every handler in every later task uses these for error responses.

- [ ] **Step 1: Create the Docker Compose file for PostgreSQL**

`docker-compose.yml` (repo root):
```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: pwlearn_dev
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./docker/init-test-db.sql:/docker-entrypoint-initdb.d/init-test-db.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

`docker/init-test-db.sql`:
```sql
CREATE DATABASE pwlearn_test;
```

Run: `docker compose up -d` from the repo root.
Expected: the `postgres` container starts and becomes healthy (`docker compose ps` shows `healthy`); it now hosts two databases, `pwlearn_dev` and `pwlearn_test`.

- [ ] **Step 2: Initialize the Go module and fetch dependencies**

Run, from `apps/backend` (create the directory first if it doesn't exist):
```bash
mkdir -p apps/backend
cd apps/backend
go mod init backend
go get github.com/gin-gonic/gin@latest
go get gorm.io/gorm@latest
go get gorm.io/driver/postgres@latest
go get github.com/golang-jwt/jwt/v5@latest
go get golang.org/x/crypto@latest
go get golang.org/x/time@latest
go get github.com/google/uuid@latest
go get github.com/lib/pq@latest
go get github.com/joho/godotenv@latest
go get github.com/gin-contrib/cors@latest
go get github.com/stretchr/testify@latest
go get github.com/swaggo/swag@latest
go get github.com/swaggo/gin-swagger@latest
go get github.com/swaggo/files@latest
go mod tidy
```
Expected: `go.mod` and `go.sum` are created/updated with no errors; `go build ./...` succeeds (will build nothing yet, but confirms the module is well-formed).

- [ ] **Step 3: Create the env files**

`apps/backend/.env.example`:
```
APP_ENV=development
PORT=4000
DATABASE_URL=postgres://postgres:postgres@localhost:5432/pwlearn_dev?sslmode=disable
JWT_ACCESS_SECRET=dev-access-secret-change-me
JWT_REFRESH_SECRET=dev-refresh-secret-change-me
ALLOW_TEST_RESET=false
SIMULATE_LATENCY_MS=0
SIMULATE_LATENCY_JITTER_MS=0
SIMULATE_ERROR_RATE=0
SIMULATE_RATE_LIMIT=1000
```

`apps/backend/.env.test` (reference only — Go tests use `config.TestConfig()`, not this file):
```
APP_ENV=test
PORT=4001
DATABASE_URL=postgres://postgres:postgres@localhost:5432/pwlearn_test?sslmode=disable
JWT_ACCESS_SECRET=test-access-secret
JWT_REFRESH_SECRET=test-refresh-secret
ALLOW_TEST_RESET=true
SIMULATE_LATENCY_MS=0
SIMULATE_LATENCY_JITTER_MS=0
SIMULATE_ERROR_RATE=0
SIMULATE_RATE_LIMIT=1000
```

- [ ] **Step 4: Implement the config package**

`apps/backend/internal/config/config.go`:
```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv                  string
	Port                    string
	DatabaseURL             string
	JWTAccessSecret         string
	JWTRefreshSecret        string
	AllowTestReset          bool
	SimulateLatencyMs       int
	SimulateLatencyJitterMs int
	SimulateErrorRate       float64
	SimulateRateLimit       int
}

// Load reads configuration from OS environment variables. Call
// godotenv.Load() before Load() if you want a local .env file merged in.
func Load() Config {
	return Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		Port:                    getEnv("PORT", "4000"),
		DatabaseURL:             mustGetEnv("DATABASE_URL"),
		JWTAccessSecret:         mustGetEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:        mustGetEnv("JWT_REFRESH_SECRET"),
		AllowTestReset:          getEnvBool("ALLOW_TEST_RESET", false),
		SimulateLatencyMs:       getEnvInt("SIMULATE_LATENCY_MS", 0),
		SimulateLatencyJitterMs: getEnvInt("SIMULATE_LATENCY_JITTER_MS", 0),
		SimulateErrorRate:       getEnvFloat("SIMULATE_ERROR_RATE", 0),
		SimulateRateLimit:       getEnvInt("SIMULATE_RATE_LIMIT", 1000),
	}
}

// TestConfig returns fixed configuration for Go tests, so tests never depend
// on .env files or the process's working directory. TEST_DATABASE_URL can
// override the connection string (e.g. in CI).
func TestConfig() Config {
	return Config{
		AppEnv:                  "test",
		Port:                    "4001",
		DatabaseURL:             getEnv("TEST_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pwlearn_test?sslmode=disable"),
		JWTAccessSecret:         "test-access-secret",
		JWTRefreshSecret:        "test-refresh-secret",
		AllowTestReset:          true,
		SimulateLatencyMs:       0,
		SimulateLatencyJitterMs: 0,
		SimulateErrorRate:       0,
		SimulateRateLimit:       1000,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		panic("missing required env var: " + key)
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
```

- [ ] **Step 5: Implement the apperror package**

`apps/backend/internal/apperror/apperror.go`:
```go
package apperror

import "github.com/gin-gonic/gin"

type AppError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *AppError) Error() string {
	return e.Message
}

func New(statusCode int, code, message string) *AppError {
	return &AppError{StatusCode: statusCode, Code: code, Message: message}
}

// RespondError writes the standard {"error": {"code", "message"}} envelope.
// If err is an *AppError its status/code are used, otherwise a generic 500.
func RespondError(c *gin.Context, err error) {
	if appErr, ok := err.(*AppError); ok {
		c.JSON(appErr.StatusCode, gin.H{
			"error": gin.H{"code": appErr.Code, "message": appErr.Message},
		})
		return
	}
	c.JSON(500, gin.H{
		"error": gin.H{"code": "INTERNAL_ERROR", "message": "Internal server error"},
	})
}

// RespondValidationError writes a 400 VALIDATION_ERROR envelope, used when
// request binding/validation (e.g. c.ShouldBindJSON) fails.
func RespondValidationError(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
	})
}
```

- [ ] **Step 6: Write the failing test for the router skeleton**

`apps/backend/internal/router/router_test.go`:
```go
package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	"backend/internal/router"

	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	r := router.New(config.TestConfig(), nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"status":"ok"}}`, rec.Body.String())
}
```

- [ ] **Step 7: Run the test to verify it fails**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `router` package does not exist yet

- [ ] **Step 8: Implement the router and main entrypoint**

`apps/backend/internal/router/router.go`:
```go
package router

import (
	"backend/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Test-Scenario", "X-Test-Run-Id"},
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok"}})
	})

	// Feature routers are registered under r.Group("/api/v1") by later tasks.
	_ = cfg
	_ = db

	return r
}
```

`apps/backend/cmd/server/main.go`:
```go
package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/router"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	r := router.New(cfg, nil)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 9: Run the test to verify it passes**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/router`

- [ ] **Step 10: Commit**

```bash
git add docker-compose.yml docker/init-test-db.sql apps/backend
git commit -m "feat(backend): scaffold Go module, Docker Compose Postgres, and Gin skeleton"
```

---

### Task 2: GORM models + `AutoMigrate` + `dbtest`/`CleanDatabase` test harness

**Files:**
- Create: `apps/backend/internal/models/models.go`
- Create: `apps/backend/internal/db/db.go`
- Create: `apps/backend/internal/dbtest/dbtest.go`
- Create: `apps/backend/internal/seed/seed.go`
- Modify: `apps/backend/cmd/server/main.go`
- Test: `apps/backend/internal/seed/seed_test.go`

**Interfaces:**
- Consumes: `config.Config`, `config.TestConfig()` (Task 1).
- Produces: 10 GORM structs in `models` (`User, Address, Category, Product, Review, Cart, CartItem, Order, OrderItem, Payment`), each with a `BeforeCreate` hook assigning a UUID. `db.Connect(cfg) (*gorm.DB, error)`, `db.AutoMigrate(db *gorm.DB) error`. `dbtest.Connect() *gorm.DB` — every later package's `TestMain` calls this once. `seed.CleanDatabase(db *gorm.DB) error` — every later handler test calls this to reset state before each test.

- [ ] **Step 1: Write the failing test for `CleanDatabase`**

`apps/backend/internal/seed/seed_test.go`:
```go
package seed_test

import (
	"os"
	"testing"

	"backend/internal/dbtest"
	"backend/internal/models"
	"backend/internal/seed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Connect()
	os.Exit(m.Run())
}

func TestCleanDatabase_RemovesAllCategoriesAndProducts(t *testing.T) {
	require.NoError(t, seed.CleanDatabase(testDB))

	category := models.Category{Name: "Temp", Slug: "temp"}
	require.NoError(t, testDB.Create(&category).Error)
	product := models.Product{
		Name: "Temp Product", Slug: "temp-product", Description: "temp",
		Price: 100, Stock: 1, CategoryID: category.ID,
	}
	require.NoError(t, testDB.Create(&product).Error)

	require.NoError(t, seed.CleanDatabase(testDB))

	var categoryCount, productCount int64
	testDB.Model(&models.Category{}).Count(&categoryCount)
	testDB.Model(&models.Product{}).Count(&productCount)
	assert.Equal(t, int64(0), categoryCount)
	assert.Equal(t, int64(0), productCount)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `models`, `db`, `dbtest`, and `seed` packages do not exist yet

- [ ] **Step 3: Implement the models package**

`apps/backend/internal/models/models.go`:
```go
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleAdmin    Role = "ADMIN"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type PaymentStatus string

const (
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// All models tag `json:` explicitly in camelCase to match the REST API
// convention the Frontend (Phase 2) expects. User.PasswordHash is `json:"-"`
// — it must never be serialized, even when a User is embedded via an
// association (e.g. Review.User, Order via admin listings).

type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Name         string    `gorm:"not null" json:"name"`
	Role         Role      `gorm:"type:varchar(20);not null;default:'CUSTOMER'" json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

func (m *User) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Address struct {
	ID         string `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string `gorm:"type:uuid;not null;index" json:"userId"`
	Label      string `gorm:"not null" json:"label"`
	Line1      string `gorm:"not null" json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `gorm:"not null" json:"city"`
	PostalCode string `gorm:"not null" json:"postalCode"`
	Country    string `gorm:"not null" json:"country"`
	IsDefault  bool   `gorm:"not null;default:false" json:"isDefault"`
}

func (m *Address) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Category struct {
	ID   string `gorm:"type:uuid;primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Slug string `gorm:"uniqueIndex;not null" json:"slug"`
}

func (m *Category) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Product struct {
	ID          string         `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"not null" json:"name"`
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`
	Description string         `gorm:"not null" json:"description"`
	Price       int            `gorm:"not null" json:"price"`
	Stock       int            `gorm:"not null" json:"stock"`
	Images      pq.StringArray `gorm:"type:text[]" json:"images"`
	CategoryID  string         `gorm:"type:uuid;not null;index" json:"categoryId"`
	Category    Category       `gorm:"foreignKey:CategoryID" json:"category"`
	Reviews     []Review       `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

func (m *Product) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Review struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID string    `gorm:"type:uuid;not null;index" json:"productId"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Rating    int       `gorm:"not null" json:"rating"`
	Comment   string    `gorm:"not null" json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

func (m *Review) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Cart struct {
	ID     string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID string     `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	Items  []CartItem `gorm:"foreignKey:CartID" json:"items"`
}

func (m *Cart) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type CartItem struct {
	ID        string  `gorm:"type:uuid;primaryKey" json:"id"`
	CartID    string  `gorm:"type:uuid;not null;index" json:"cartId"`
	ProductID string  `gorm:"type:uuid;not null;index" json:"productId"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `gorm:"not null" json:"quantity"`
}

func (m *CartItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Order struct {
	ID                string      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            string      `gorm:"type:uuid;not null;index" json:"userId"`
	Status            OrderStatus `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	TotalAmount       int         `gorm:"not null" json:"totalAmount"`
	ShippingAddressID string      `gorm:"type:uuid;not null" json:"shippingAddressId"`
	CreatedAt         time.Time   `json:"createdAt"`
	Items             []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	Payment           *Payment    `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
}

func (m *Order) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type OrderItem struct {
	ID              string  `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID         string  `gorm:"type:uuid;not null;index" json:"orderId"`
	ProductID       string  `gorm:"type:uuid;not null;index" json:"productId"`
	Product         Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity        int     `gorm:"not null" json:"quantity"`
	PriceAtPurchase int     `gorm:"not null" json:"priceAtPurchase"`
}

func (m *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Payment struct {
	ID                string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID           string        `gorm:"type:uuid;uniqueIndex;not null" json:"orderId"`
	Method            string        `gorm:"not null" json:"method"`
	Status            PaymentStatus `gorm:"type:varchar(20);not null" json:"status"`
	MockTransactionID string        `gorm:"not null" json:"mockTransactionId"`
	CreatedAt         time.Time     `json:"createdAt"`
}

func (m *Payment) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
```

- [ ] **Step 4: Implement the db package**

`apps/backend/internal/db/db.go`:
```go
package db

import (
	"backend/internal/config"
	"backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.Category{},
		&models.Product{},
		&models.Review{},
		&models.Cart{},
		&models.CartItem{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
	)
}
```

- [ ] **Step 5: Implement the dbtest helper**

`apps/backend/internal/dbtest/dbtest.go`:
```go
package dbtest

import (
	"backend/internal/config"
	"backend/internal/db"

	"gorm.io/gorm"
)

// Connect opens a connection to the test database and ensures its schema is
// up to date. Call once per package in TestMain and reuse the returned
// *gorm.DB across that package's tests.
func Connect() *gorm.DB {
	cfg := config.TestConfig()
	gdb, err := db.Connect(cfg)
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		panic(err)
	}
	return gdb
}
```

- [ ] **Step 6: Implement `CleanDatabase`**

`apps/backend/internal/seed/seed.go`:
```go
package seed

import "gorm.io/gorm"

// CleanDatabase truncates every table so tests start from a blank slate.
// CASCADE handles foreign-key ordering.
func CleanDatabase(db *gorm.DB) error {
	return db.Exec(`TRUNCATE TABLE
		payments, order_items, orders, reviews, cart_items, carts, addresses, products, categories, users
		RESTART IDENTITY CASCADE`).Error
}
```

- [ ] **Step 7: Wire the database into `main.go`**

Replace `apps/backend/cmd/server/main.go` with:
```go
package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/router"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	gdb, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	r := router.New(cfg, gdb)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 8: Run the test to verify it passes**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/router`, `ok backend/internal/seed`

- [ ] **Step 9: Commit**

```bash
git add apps/backend/internal/models apps/backend/internal/db apps/backend/internal/dbtest apps/backend/internal/seed apps/backend/cmd/server/main.go
git commit -m "feat(backend): add GORM models, AutoMigrate, and CleanDatabase test harness"
```

---

### Task 3: Deterministic seed/reset logic (`seed.Run`)

**Files:**
- Create: `apps/backend/internal/password/password.go` (pulled forward from Task 4 — seeding needs to hash the fixed accounts' passwords; Task 4 only adds the JWT `token` package)
- Modify: `apps/backend/internal/seed/seed.go`
- Create: `apps/backend/cmd/seed/main.go`
- Test: `apps/backend/internal/seed/seed_test.go` (extend)

**Interfaces:**
- Consumes: `models.*` (Task 2), `seed.CleanDatabase` (Task 2).
- Produces: `password.Hash(plain string) (string, error)`, `password.Verify(plain, hash string) bool` — used by Task 6's auth service. `seed.Run(db *gorm.DB) error` — used by Task 17's `/test/reset` handler and `cmd/seed`.

- [ ] **Step 1: Write the failing test for `seed.Run`**

Append to `apps/backend/internal/seed/seed_test.go`:
```go
func TestRun_SeedsCategoriesProductsAndFixedAccounts(t *testing.T) {
	require.NoError(t, seed.Run(testDB))

	var categoryCount, productCount int64
	testDB.Model(&models.Category{}).Count(&categoryCount)
	testDB.Model(&models.Product{}).Count(&productCount)
	assert.Equal(t, int64(4), categoryCount)
	assert.Equal(t, int64(20), productCount)

	var outOfStockCount int64
	testDB.Model(&models.Product{}).Where("stock = 0").Count(&outOfStockCount)
	assert.GreaterOrEqual(t, outOfStockCount, int64(1))

	var admin models.User
	require.NoError(t, testDB.Where("email = ?", "admin@example.com").First(&admin).Error)
	assert.Equal(t, models.RoleAdmin, admin.Role)

	var customer models.User
	require.NoError(t, testDB.Where("email = ?", "customer@example.com").First(&customer).Error)
	assert.Equal(t, models.RoleCustomer, customer.Role)

	var cart models.Cart
	require.NoError(t, testDB.Where("user_id = ?", customer.ID).First(&cart).Error)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `seed.Run` is not defined

- [ ] **Step 3: Implement the password package**

`apps/backend/internal/password/password.go`:
```go
package password

import "golang.org/x/crypto/bcrypt"

const cost = 10

func Hash(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func Verify(plain, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
```

- [ ] **Step 4: Extend `seed.go` with seed data and `Run`**

Append to `apps/backend/internal/seed/seed.go` (add `"backend/internal/models"` and `"backend/internal/password"` to the imports):
```go
type categorySeed struct {
	Name string
	Slug string
}

var categorySeeds = []categorySeed{
	{Name: "Electronics", Slug: "electronics"},
	{Name: "Home & Kitchen", Slug: "home-kitchen"},
	{Name: "Books", Slug: "books"},
	{Name: "Sports", Slug: "sports"},
}

type productSeed struct {
	Name         string
	Slug         string
	Description  string
	Price        int
	Stock        int
	CategorySlug string
}

var productSeeds = []productSeed{
	{Name: "Wireless Mouse", Slug: "wireless-mouse", Description: "Ergonomic wireless mouse", Price: 59900, Stock: 25, CategorySlug: "electronics"},
	{Name: "Mechanical Keyboard", Slug: "mechanical-keyboard", Description: "RGB mechanical keyboard", Price: 189900, Stock: 10, CategorySlug: "electronics"},
	{Name: "4K Monitor", Slug: "4k-monitor", Description: "27-inch 4K monitor", Price: 899900, Stock: 5, CategorySlug: "electronics"},
	{Name: "USB-C Hub", Slug: "usb-c-hub", Description: "7-in-1 USB-C hub", Price: 129900, Stock: 0, CategorySlug: "electronics"},
	{Name: "Bluetooth Speaker", Slug: "bluetooth-speaker", Description: "Portable Bluetooth speaker", Price: 149900, Stock: 16, CategorySlug: "electronics"},
	{Name: "Non-stick Frying Pan", Slug: "non-stick-frying-pan", Description: "28cm non-stick pan", Price: 45900, Stock: 30, CategorySlug: "home-kitchen"},
	{Name: "Electric Kettle", Slug: "electric-kettle", Description: "1.7L electric kettle", Price: 69900, Stock: 18, CategorySlug: "home-kitchen"},
	{Name: "Coffee Grinder", Slug: "coffee-grinder", Description: "Burr coffee grinder", Price: 159900, Stock: 12, CategorySlug: "home-kitchen"},
	{Name: "Bamboo Cutting Board", Slug: "bamboo-cutting-board", Description: "Set of 3 boards", Price: 29900, Stock: 40, CategorySlug: "home-kitchen"},
	{Name: "Desk Lamp", Slug: "desk-lamp", Description: "LED desk lamp with USB port", Price: 49900, Stock: 28, CategorySlug: "home-kitchen"},
	{Name: "Clean Code", Slug: "clean-code", Description: "A Handbook of Agile Software Craftsmanship", Price: 89900, Stock: 20, CategorySlug: "books"},
	{Name: "The Pragmatic Programmer", Slug: "pragmatic-programmer", Description: "20th Anniversary Edition", Price: 99900, Stock: 15, CategorySlug: "books"},
	{Name: "Designing Data-Intensive Applications", Slug: "ddia", Description: "By Martin Kleppmann", Price: 129900, Stock: 8, CategorySlug: "books"},
	{Name: "Atomic Habits", Slug: "atomic-habits", Description: "By James Clear", Price: 59900, Stock: 50, CategorySlug: "books"},
	{Name: "Notebook Set", Slug: "notebook-set", Description: "Set of 3 hardcover notebooks", Price: 19900, Stock: 60, CategorySlug: "books"},
	{Name: "Yoga Mat", Slug: "yoga-mat", Description: "Non-slip 6mm yoga mat", Price: 39900, Stock: 22, CategorySlug: "sports"},
	{Name: "Adjustable Dumbbells", Slug: "adjustable-dumbbells", Description: "2x 20kg adjustable dumbbells", Price: 349900, Stock: 6, CategorySlug: "sports"},
	{Name: "Running Shoes", Slug: "running-shoes", Description: "Lightweight running shoes", Price: 219900, Stock: 14, CategorySlug: "sports"},
	{Name: "Resistance Bands Set", Slug: "resistance-bands", Description: "5-level resistance bands", Price: 24900, Stock: 35, CategorySlug: "sports"},
	{Name: "Water Bottle", Slug: "water-bottle", Description: "1L insulated water bottle", Price: 34900, Stock: 45, CategorySlug: "sports"},
}

// Run truncates the database and inserts deterministic seed data: 4
// categories, 20 products (at least one with Stock == 0), and the two
// fixed accounts admin@example.com / customer@example.com.
func Run(db *gorm.DB) error {
	if err := CleanDatabase(db); err != nil {
		return err
	}

	categoryBySlug := make(map[string]models.Category, len(categorySeeds))
	for _, c := range categorySeeds {
		category := models.Category{Name: c.Name, Slug: c.Slug}
		if err := db.Create(&category).Error; err != nil {
			return err
		}
		categoryBySlug[c.Slug] = category
	}

	for _, p := range productSeeds {
		category, ok := categoryBySlug[p.CategorySlug]
		if !ok {
			continue
		}
		product := models.Product{
			Name:        p.Name,
			Slug:        p.Slug,
			Description: p.Description,
			Price:       p.Price,
			Stock:       p.Stock,
			Images:      []string{"https://picsum.photos/seed/" + p.Slug + "/600/600"},
			CategoryID:  category.ID,
		}
		if err := db.Create(&product).Error; err != nil {
			return err
		}
	}

	adminHash, err := password.Hash("Admin123!")
	if err != nil {
		return err
	}
	admin := models.User{Email: "admin@example.com", PasswordHash: adminHash, Name: "Admin User", Role: models.RoleAdmin}
	if err := db.Create(&admin).Error; err != nil {
		return err
	}

	customerHash, err := password.Hash("Customer123!")
	if err != nil {
		return err
	}
	customer := models.User{Email: "customer@example.com", PasswordHash: customerHash, Name: "Test Customer", Role: models.RoleCustomer}
	if err := db.Create(&customer).Error; err != nil {
		return err
	}

	cart := models.Cart{UserID: customer.ID}
	return db.Create(&cart).Error
}
```

- [ ] **Step 5: Create the seed CLI**

`apps/backend/cmd/seed/main.go`:
```go
package main

import (
	"log"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/seed"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	gdb, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(gdb); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	if err := seed.Run(gdb); err != nil {
		log.Fatalf("failed to seed database: %v", err)
	}
	log.Println("Seed complete")
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/seed`

- [ ] **Step 7: Commit**

```bash
git add apps/backend/internal/password apps/backend/internal/seed apps/backend/cmd/seed
git commit -m "feat(backend): add deterministic seed data and seed CLI"
```

---

### Task 4: JWT (`token`) package

**Files:**
- Create: `apps/backend/internal/token/token.go`
- Test: `apps/backend/internal/token/token_test.go`

**Interfaces:**
- Produces: `token.Claims{ Role string; jwt.RegisteredClaims }` (`.Subject` holds the user ID), `token.SignAccessToken(userID, role, secret string) (string, error)`, `token.SignRefreshToken(userID, role, secret string) (string, error)`, `token.Verify(tokenString, secret string) (*token.Claims, error)` — used by Task 5's middleware and Task 6's auth service.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/token/token_test.go`:
```go
package token_test

import (
	"testing"

	"backend/internal/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignAndVerifyAccessToken(t *testing.T) {
	signed, err := token.SignAccessToken("user-1", "CUSTOMER", "secret")
	require.NoError(t, err)

	claims, err := token.Verify(signed, "secret")
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "CUSTOMER", claims.Role)
}

func TestSignAndVerifyRefreshToken(t *testing.T) {
	signed, err := token.SignRefreshToken("user-2", "ADMIN", "secret")
	require.NoError(t, err)

	claims, err := token.Verify(signed, "secret")
	require.NoError(t, err)
	assert.Equal(t, "user-2", claims.Subject)
	assert.Equal(t, "ADMIN", claims.Role)
}

func TestVerify_RejectsTamperedToken(t *testing.T) {
	signed, err := token.SignAccessToken("user-3", "CUSTOMER", "secret")
	require.NoError(t, err)

	_, err = token.Verify(signed+"x", "secret")
	assert.Error(t, err)
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	signed, err := token.SignAccessToken("user-4", "CUSTOMER", "secret-a")
	require.NoError(t, err)

	_, err = token.Verify(signed, "secret-b")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `token` package does not exist yet

- [ ] **Step 3: Implement the token package**

`apps/backend/internal/token/token.go`:
```go
package token

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func SignAccessToken(userID, role, secret string) (string, error) {
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

func SignRefreshToken(userID, role, secret string) (string, error) {
	claims := Claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Verify checks the token's signature and expiry against secret and returns
// its claims. Used for both access and refresh tokens — callers pass
// whichever secret matches the token they are verifying.
func Verify(tokenString, secret string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, jwt.ErrTokenSignatureInvalid
	}
	return claims, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/token`

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/token
git commit -m "feat(backend): add JWT signing/verification package"
```

---

### Task 5: Auth middleware (`RequireAuth`, `RequireAdmin`)

**Files:**
- Create: `apps/backend/internal/middleware/auth.go`
- Test: `apps/backend/internal/middleware/auth_test.go`

**Interfaces:**
- Consumes: `token.Verify`, `token.SignAccessToken` (Task 4), `apperror.New`, `apperror.RespondError` (Task 1).
- Produces: `middleware.RequireAuth(secret string) gin.HandlerFunc` — sets Gin context keys `"userID"` (string) and `"userRole"` (string). `middleware.RequireAdmin() gin.HandlerFunc`. Both used by every protected route from Task 6 onward.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/middleware/auth_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/middleware"
	"backend/internal/token"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRouter(secret string, requireAdmin bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers := []gin.HandlerFunc{middleware.RequireAuth(secret)}
	if requireAdmin {
		handlers = append(handlers, middleware.RequireAdmin())
	}
	handlers = append(handlers, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"userID": c.MustGet("userID")}})
	})
	r.GET("/protected", handlers...)
	return r
}

func TestRequireAuth_AllowsValidToken(t *testing.T) {
	r := newTestRouter("secret", false)
	signed, err := token.SignAccessToken("user-1", "CUSTOMER", "secret")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"userID":"user-1"}}`, rec.Body.String())
}

func TestRequireAuth_RejectsMissingHeader(t *testing.T) {
	r := newTestRouter("secret", false)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAuth_RejectsInvalidToken(t *testing.T) {
	r := newTestRouter("secret", false)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAdmin_AllowsAdminRole(t *testing.T) {
	r := newTestRouter("secret", true)
	signed, err := token.SignAccessToken("admin-1", "ADMIN", "secret")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireAdmin_RejectsNonAdminRole(t *testing.T) {
	r := newTestRouter("secret", true)
	signed, err := token.SignAccessToken("user-1", "CUSTOMER", "secret")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `middleware` package does not exist yet

- [ ] **Step 3: Implement the middleware**

`apps/backend/internal/middleware/auth.go`:
```go
package middleware

import (
	"net/http"
	"strings"

	"backend/internal/apperror"
	"backend/internal/token"

	"github.com/gin-gonic/gin"
)

func RequireAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Missing access token"))
			c.Abort()
			return
		}
		claims, err := token.Verify(strings.TrimPrefix(header, "Bearer "), secret)
		if err != nil {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired access token"))
			c.Abort()
			return
		}
		c.Set("userID", claims.Subject)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("userRole")
		if role != "ADMIN" {
			apperror.RespondError(c, apperror.New(http.StatusForbidden, "FORBIDDEN", "Admin role required"))
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/middleware`

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/middleware
git commit -m "feat(backend): add RequireAuth/RequireAdmin middleware"
```

---

### Task 6: Auth routes — register + login

**Files:**
- Create: `apps/backend/internal/apitest/apitest.go` (shared HTTP test helpers — created now because this is the first task that needs full router+DB integration tests; every later handler test in Tasks 7–17 reuses it)
- Create: `apps/backend/internal/auth/service.go`
- Create: `apps/backend/internal/auth/handler.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/auth/handler_test.go`

**Interfaces:**
- Consumes: `models.*` (Task 2), `password.Hash`/`Verify` (Task 3), `token.SignAccessToken`/`SignRefreshToken` (Task 4), `apperror.*` (Task 1), `router.New` (Task 1), `seed.CleanDatabase` (Task 2).
- Produces: `apitest.NewRouter(t, db) http.Handler`, `apitest.DoJSON(t, r, method, path, body) *httptest.ResponseRecorder`, `apitest.DoJSONAuth(t, r, method, path, accessToken, body) *httptest.ResponseRecorder`, `apitest.DecodeData(t, rec) map[string]any` (for object-shaped `data`), `apitest.DecodeDataList(t, rec) []any` (for array-shaped `data`), `apitest.DecodeError(t, rec) map[string]any` — used by every handler test from here on. `auth.RegisterUser(db, email, plainPassword, name) (*models.User, error)`, `auth.LoginUser(db, email, plainPassword) (*models.User, error)` (from `auth/service.go`). `auth.RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config)` mounted at `/api/v1/auth` — response shape `{"data": {"user": {"id","email","name","role"}, "accessToken": "..."}}`, refresh token set as httpOnly cookie named `refreshToken`.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/apitest/apitest.go`:
```go
package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/config"
	"backend/internal/router"
	"backend/internal/seed"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// NewRouter cleans the database and returns a fresh router wired to it,
// ready for a single test.
func NewRouter(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	require.NoError(t, seed.CleanDatabase(db))
	return router.New(config.TestConfig(), db)
}

// DoJSON serializes body as JSON (if non-nil) and performs the request
// against r, returning the recorder.
func DoJSON(t *testing.T, r http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// DoJSONAuth is DoJSON with an Authorization: Bearer header attached.
func DoJSONAuth(t *testing.T, r http.Handler, method, path, accessToken string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// DecodeData unmarshals rec.Body into a map and returns the "data" field.
func DecodeData(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, _ := body["data"].(map[string]any)
	return data
}

// DecodeError unmarshals rec.Body into a map and returns the "error" field.
func DecodeError(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errObj, _ := body["error"].(map[string]any)
	return errObj
}

// DecodeDataList unmarshals rec.Body and returns the "data" field as a list,
// for endpoints whose data payload is a JSON array rather than an object.
func DecodeDataList(t *testing.T, rec *httptest.ResponseRecorder) []any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	list, _ := body["data"].([]any)
	return list
}
```

`apps/backend/internal/auth/handler_test.go`:
```go
package auth_test

import (
	"net/http"
	"os"
	"testing"

	"backend/internal/apitest"
	"backend/internal/dbtest"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Connect()
	os.Exit(m.Run())
}

func TestRegister_CreatesUserAndReturnsAccessToken(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "new@example.com", "password": "Password123!", "name": "New User",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)
	data := apitest.DecodeData(t, rec)
	user := data["user"].(map[string]any)
	assert.Equal(t, "new@example.com", user["email"])
	assert.NotEmpty(t, data["accessToken"])
	assert.Contains(t, rec.Header().Get("Set-Cookie"), "refreshToken=")
}

func TestRegister_RejectsDuplicateEmail(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "dup@example.com", "password": "Password123!", "name": "Dup User",
	})
	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "dup@example.com", "password": "Password123!", "name": "Dup User 2",
	})

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "EMAIL_TAKEN", apitest.DecodeError(t, rec)["code"])
}

func TestLogin_SucceedsWithCorrectCredentials(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "login@example.com", "password": "Password123!", "name": "Login User",
	})

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "login@example.com", "password": "Password123!",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, apitest.DecodeData(t, rec)["accessToken"])
}

func TestLogin_RejectsWrongPassword(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "wrongpass@example.com", "password": "Password123!", "name": "User",
	})

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "wrongpass@example.com", "password": "WrongPassword!",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "INVALID_CREDENTIALS", apitest.DecodeError(t, rec)["code"])
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `auth` package does not exist yet

- [ ] **Step 3: Implement the auth service**

`apps/backend/internal/auth/service.go`:
```go
package auth

import (
	"errors"

	"backend/internal/apperror"
	"backend/internal/models"
	"backend/internal/password"

	"gorm.io/gorm"
)

func RegisterUser(db *gorm.DB, email, plainPassword, name string) (*models.User, error) {
	var existing models.User
	err := db.Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, apperror.New(409, "EMAIL_TAKEN", "Email is already registered")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, err
	}

	user := models.User{Email: email, PasswordHash: hash, Name: name, Role: models.RoleCustomer}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}

	cart := models.Cart{UserID: user.ID}
	if err := db.Create(&cart).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func LoginUser(db *gorm.DB, email, plainPassword string) (*models.User, error) {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, apperror.New(401, "INVALID_CREDENTIALS", "Invalid email or password")
	}
	if !password.Verify(plainPassword, user.PasswordHash) {
		return nil, apperror.New(401, "INVALID_CREDENTIALS", "Invalid email or password")
	}
	return &user, nil
}
```

- [ ] **Step 4: Implement the auth handler**

`apps/backend/internal/auth/handler.go`:
```go
package auth

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/models"
	"backend/internal/token"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const refreshCookieName = "refreshToken"
const refreshCookieMaxAge = 7 * 24 * 60 * 60 // seconds

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type tokenPair struct {
	AccessToken  string
	RefreshToken string
}

func issueTokens(cfg config.Config, userID, role string) (tokenPair, error) {
	access, err := token.SignAccessToken(userID, role, cfg.JWTAccessSecret)
	if err != nil {
		return tokenPair{}, err
	}
	refresh, err := token.SignRefreshToken(userID, role, cfg.JWTRefreshSecret)
	if err != nil {
		return tokenPair{}, err
	}
	return tokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func setRefreshCookie(c *gin.Context, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(refreshCookieName, value, refreshCookieMaxAge, "/", "", false, true)
}

func userJSON(u *models.User) gin.H {
	return gin.H{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role}
}

// RegisterRoutes mounts the auth endpoints on rg (expected to be the
// "/api/v1/auth" group).
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.POST("/register", func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		user, err := RegisterUser(db, req.Email, req.Password, req.Name)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		tokens, err := issueTokens(cfg, user.ID, string(user.Role))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusCreated, gin.H{
			"data": gin.H{"user": userJSON(user), "accessToken": tokens.AccessToken},
		})
	})

	rg.POST("/login", func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		user, err := LoginUser(db, req.Email, req.Password)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		tokens, err := issueTokens(cfg, user.ID, string(user.Role))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"user": userJSON(user), "accessToken": tokens.AccessToken},
		})
	})
}
```

- [ ] **Step 5: Mount the auth routes**

Replace `apps/backend/internal/router/router.go` with:
```go
package router

import (
	"backend/internal/auth"
	"backend/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Test-Scenario", "X-Test-Run-Id"},
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok"}})
	})

	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1.Group("/auth"), db, cfg)

	return r
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/auth`

- [ ] **Step 7: Commit**

```bash
git add apps/backend/internal/apitest apps/backend/internal/auth apps/backend/internal/router
git commit -m "feat(backend): add register and login endpoints"
```

---

### Task 7: Auth routes — refresh + logout

**Files:**
- Modify: `apps/backend/internal/auth/service.go`
- Modify: `apps/backend/internal/auth/handler.go`
- Test: `apps/backend/internal/auth/handler_test.go` (extend)

**Interfaces:**
- Produces: `auth.RotateRefreshToken(db, cfg, refreshToken string) (tokenPair, error)` (added to `service.go`). `POST /api/v1/auth/refresh` → `{"data": {"accessToken"}}`, rotates the `refreshToken` cookie. `POST /api/v1/auth/logout` → `204`, clears the `refreshToken` cookie.

- [ ] **Step 1: Write the failing tests**

Append to `apps/backend/internal/auth/handler_test.go` (add `"net/http/httptest"` to the imports):
```go
func TestRefresh_IssuesNewAccessTokenFromValidCookie(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	registerRec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "refresh@example.com", "password": "Password123!", "name": "Refresh User",
	})
	cookie := registerRec.Header().Get("Set-Cookie")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, apitest.DecodeData(t, rec)["accessToken"])
}

func TestRefresh_RejectsMissingCookie(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/refresh", nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestLogout_ClearsCookieAndReturns204(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/logout", nil)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Contains(t, rec.Header().Get("Set-Cookie"), "refreshToken=;")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `404`-shaped responses (routes not registered)

- [ ] **Step 3: Add `RotateRefreshToken` to the service**

Append to `apps/backend/internal/auth/service.go` (add `"backend/internal/config"` and `"backend/internal/token"` to the imports):
```go
func RotateRefreshToken(db *gorm.DB, cfg config.Config, refreshToken string) (tokenPair, error) {
	claims, err := token.Verify(refreshToken, cfg.JWTRefreshSecret)
	if err != nil {
		return tokenPair{}, apperror.New(401, "INVALID_REFRESH_TOKEN", "Invalid or expired refresh token")
	}
	var user models.User
	if err := db.First(&user, "id = ?", claims.Subject).Error; err != nil {
		return tokenPair{}, apperror.New(401, "INVALID_REFRESH_TOKEN", "User no longer exists")
	}
	return issueTokens(cfg, user.ID, string(user.Role))
}
```

- [ ] **Step 4: Add the routes**

Append to `apps/backend/internal/auth/handler.go`'s `RegisterRoutes` function, after the `/login` route:
```go
	rg.POST("/refresh", func(c *gin.Context) {
		refreshToken, err := c.Cookie(refreshCookieName)
		if err != nil || refreshToken == "" {
			apperror.RespondError(c, apperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token"))
			return
		}
		tokens, err := RotateRefreshToken(db, cfg, refreshToken)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		setRefreshCookie(c, tokens.RefreshToken)
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"accessToken": tokens.AccessToken}})
	})

	rg.POST("/logout", func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(refreshCookieName, "", -1, "/", "", false, true)
		c.Status(http.StatusNoContent)
	})
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/auth`

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/auth
git commit -m "feat(backend): add refresh and logout endpoints"
```

---

### Task 8: Auth routes — forgot-password + reset-password

**Files:**
- Modify: `apps/backend/internal/auth/service.go`
- Modify: `apps/backend/internal/auth/handler.go`
- Test: `apps/backend/internal/auth/handler_test.go` (extend)

**Interfaces:**
- Produces: `auth.RequestPasswordReset(db, email string) error` (always returns nil to the caller, even for unknown emails — logs the token via the standard `log` package instead of sending an email), `auth.ResetPassword(db, resetToken, newPassword string) error`. `POST /api/v1/auth/forgot-password` → always `200 {"data": {"message"}}`. `POST /api/v1/auth/reset-password` → `200` on success, `400 INVALID_RESET_TOKEN` on bad/expired token.

- [ ] **Step 1: Write the failing tests**

Append to `apps/backend/internal/auth/handler_test.go` (add `"bytes"`, `"log"`, `"regexp"` to the imports):
```go
func TestForgotPassword_LogsTokenAndAllowsReset(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "forgot@example.com", "password": "OldPassword123!", "name": "Forgot User",
	})

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	forgotRec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/forgot-password", map[string]string{
		"email": "forgot@example.com",
	})
	log.SetOutput(os.Stderr)

	assert.Equal(t, http.StatusOK, forgotRec.Code)
	match := regexp.MustCompile(`token for forgot@example\.com: ([a-f0-9]+)`).FindStringSubmatch(logBuf.String())
	require.NotNil(t, match)

	resetRec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/reset-password", map[string]string{
		"token": match[1], "newPassword": "NewPassword123!",
	})
	assert.Equal(t, http.StatusOK, resetRec.Code)

	loginRec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email": "forgot@example.com", "password": "NewPassword123!",
	})
	assert.Equal(t, http.StatusOK, loginRec.Code)
}

func TestForgotPassword_ReturnsOKEvenWhenEmailDoesNotExist(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/forgot-password", map[string]string{
		"email": "nobody@example.com",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResetPassword_RejectsInvalidToken(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/reset-password", map[string]string{
		"token": "not-a-real-token", "newPassword": "NewPassword123!",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "INVALID_RESET_TOKEN", apitest.DecodeError(t, rec)["code"])
}
```

Note: `require` must already be imported for `TestMain`'s dbtest wiring; if not already present, add `"github.com/stretchr/testify/require"` to the imports too.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `404`-shaped responses (routes not registered)

- [ ] **Step 3: Add the reset-token store and service functions**

Append to `apps/backend/internal/auth/service.go` (add `"crypto/rand"`, `"encoding/hex"`, `"log"`, `"sync"`, `"time"` to the imports):
```go
type resetEntry struct {
	userID    string
	expiresAt time.Time
}

var (
	resetTokensMu sync.Mutex
	resetTokens   = map[string]resetEntry{}
)

// RequestPasswordReset never reveals whether email exists: it always
// returns nil, logging the reset token instead of sending an email
// (spec §2 non-goals — no real email delivery).
func RequestPasswordReset(db *gorm.DB, email string) error {
	var user models.User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	resetToken := hex.EncodeToString(tokenBytes)

	resetTokensMu.Lock()
	resetTokens[resetToken] = resetEntry{userID: user.ID, expiresAt: time.Now().Add(15 * time.Minute)}
	resetTokensMu.Unlock()

	log.Printf("[password-reset] token for %s: %s", email, resetToken)
	return nil
}

func ResetPassword(db *gorm.DB, resetToken, newPassword string) error {
	resetTokensMu.Lock()
	entry, ok := resetTokens[resetToken]
	if ok {
		delete(resetTokens, resetToken)
	}
	resetTokensMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return apperror.New(400, "INVALID_RESET_TOKEN", "Invalid or expired reset token")
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("id = ?", entry.userID).Update("password_hash", hash).Error
}
```

- [ ] **Step 4: Add the routes**

Append to `apps/backend/internal/auth/handler.go`, above `RegisterRoutes` add:
```go
type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}
```
and inside `RegisterRoutes`, after the `/logout` route:
```go
	rg.POST("/forgot-password", func(c *gin.Context) {
		var req forgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		if err := RequestPasswordReset(db, req.Email); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "If the email exists, a reset link has been sent"}})
	})

	rg.POST("/reset-password", func(c *gin.Context) {
		var req resetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		if err := ResetPassword(db, req.Token, req.NewPassword); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Password has been reset"}})
	})
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/auth`

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/auth
git commit -m "feat(backend): add forgot-password and reset-password endpoints"
```

---

### Task 9: Network simulation — deterministic `X-Test-Scenario`

**Files:**
- Create: `apps/backend/internal/middleware/test_scenario.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/middleware/test_scenario_test.go`

**Interfaces:**
- Consumes: `config.Config.AppEnv` (Task 1).
- Produces: `middleware.TestScenario(appEnv string) gin.HandlerFunc` — reads header `X-Test-Scenario` with values `slow | error | timeout | rate-limited | flaky:N`; header override always short-circuits ambient behavior (Task 10). Mounted globally in `router.go` before all routes, so it applies to `/health` too (used directly by the tests below).

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/middleware/test_scenario_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newScenarioRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.TestScenario("test"))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})
	return r
}

func TestTestScenario_PassesThroughWithoutHeader(t *testing.T) {
	r := newScenarioRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTestScenario_DelaysResponseForSlow(t *testing.T) {
	r := newScenarioRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Test-Scenario", "slow")
	rec := httptest.NewRecorder()

	start := time.Now()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.GreaterOrEqual(t, time.Since(start), 2900*time.Millisecond)
}

func TestTestScenario_ReturnsErrorForError(t *testing.T) {
	r := newScenarioRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Test-Scenario", "error")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTestScenario_NeverRespondsWithinOneSecondForTimeout(t *testing.T) {
	r := newScenarioRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Test-Scenario", "timeout")
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		r.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("expected the handler to never respond, but it did")
	case <-time.After(1 * time.Second):
		// expected: handler is still blocked after 1s
	}
}

func TestTestScenario_ReturnsRateLimitedForRateLimited(t *testing.T) {
	r := newScenarioRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Test-Scenario", "rate-limited")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestTestScenario_FailsFirstNCallsThenSucceedsForFlaky(t *testing.T) {
	r := newScenarioRouter()
	runID := "flaky-test-run-1"

	makeRequest := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Test-Scenario", "flaky:2")
		req.Header.Set("X-Test-Run-Id", runID)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	first := makeRequest()
	second := makeRequest()
	third := makeRequest()

	assert.Equal(t, http.StatusInternalServerError, first.Code)
	assert.Equal(t, http.StatusInternalServerError, second.Code)
	assert.Equal(t, http.StatusOK, third.Code)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `middleware.TestScenario` does not exist yet

- [ ] **Step 3: Implement the middleware**

`apps/backend/internal/middleware/test_scenario.go`:
```go
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var flakyCounters sync.Map // map[string]int

func TestScenario(appEnv string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if appEnv == "production" {
			c.Next()
			return
		}
		scenario := c.GetHeader("X-Test-Scenario")
		if scenario == "" {
			c.Next()
			return
		}

		switch {
		case scenario == "slow":
			time.Sleep(3 * time.Second)
			c.Next()

		case scenario == "error":
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "SIMULATED_ERROR", "message": "Simulated server error"},
			})
			c.Abort()

		case scenario == "timeout":
			// Block until the client gives up or a generous ceiling passes,
			// simulating a hung request without leaking the goroutine forever.
			select {
			case <-time.After(time.Hour):
			case <-c.Request.Context().Done():
			}
			c.Abort()

		case scenario == "rate-limited":
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "SIMULATED_RATE_LIMIT", "message": "Simulated rate limit"},
			})
			c.Abort()

		case strings.HasPrefix(scenario, "flaky:"):
			failCount, _ := strconv.Atoi(strings.TrimPrefix(scenario, "flaky:"))
			key := c.GetHeader("X-Test-Run-Id") + ":" + c.Request.URL.Path
			value, _ := flakyCounters.LoadOrStore(key, 0)
			attempts := value.(int)
			flakyCounters.Store(key, attempts+1)
			if attempts < failCount {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "SIMULATED_FLAKY_ERROR", "message": "Simulated failure"},
				})
				c.Abort()
				return
			}
			flakyCounters.Delete(key)
			c.Next()

		default:
			c.Next()
		}
	}
}
```

- [ ] **Step 4: Mount the middleware**

In `apps/backend/internal/router/router.go`, add the import:
```go
"backend/internal/middleware"
```
and add this line directly after the `cors.New(...)` middleware, before the `/health` route:
```go
	r.Use(middleware.TestScenario(cfg.AppEnv))
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/middleware`

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/middleware apps/backend/internal/router
git commit -m "feat(backend): add deterministic X-Test-Scenario middleware"
```

---

### Task 10: Network simulation — ambient latency/error + rate limiter

**Files:**
- Create: `apps/backend/internal/middleware/ambient_latency.go`
- Create: `apps/backend/internal/middleware/ambient_error.go`
- Create: `apps/backend/internal/middleware/rate_limiter.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/middleware/ambient_latency_test.go`
- Test: `apps/backend/internal/middleware/ambient_error_test.go`
- Test: `apps/backend/internal/middleware/rate_limiter_test.go`

**Interfaces:**
- Consumes: `config.Config.SimulateLatencyMs/SimulateLatencyJitterMs/SimulateErrorRate/SimulateRateLimit` (Task 1).
- Produces: `middleware.AmbientLatency(latencyMs, jitterMs int) gin.HandlerFunc`, `middleware.AmbientError(errorRate float64) gin.HandlerFunc`, `middleware.RateLimiter(limitPerMinute int) gin.HandlerFunc` — mounted globally after `TestScenario`, so the deterministic header always takes effect first.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/middleware/ambient_latency_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newPingRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(mw)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}})
	})
	return r
}

func TestAmbientLatency_NoDelayWhenZero(t *testing.T) {
	r := newPingRouter(middleware.AmbientLatency(0, 0))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	r.ServeHTTP(rec, req)

	assert.Less(t, time.Since(start), 200*time.Millisecond)
}

func TestAmbientLatency_DelaysApproximatelyConfiguredMs(t *testing.T) {
	r := newPingRouter(middleware.AmbientLatency(300, 0))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	start := time.Now()
	r.ServeHTTP(rec, req)

	assert.GreaterOrEqual(t, time.Since(start), 290*time.Millisecond)
}
```

`apps/backend/internal/middleware/ambient_error_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/middleware"

	"github.com/stretchr/testify/assert"
)

func TestAmbientError_NeverErrorsWhenRateIsZero(t *testing.T) {
	r := newPingRouter(middleware.AmbientError(0))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestAmbientError_AlwaysErrorsWhenRateIsOne(t *testing.T) {
	r := newPingRouter(middleware.AmbientError(1))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

`apps/backend/internal/middleware/rate_limiter_test.go`:
```go
package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/middleware"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_ReturnsTooManyRequestsAfterLimit(t *testing.T) {
	r := newPingRouter(middleware.RateLimiter(2))

	doPing := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	first := doPing()
	second := doPing()
	third := doPing()

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, http.StatusTooManyRequests, third.Code)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — `middleware.AmbientLatency`/`AmbientError`/`RateLimiter` do not exist yet

- [ ] **Step 3: Implement the three middlewares**

`apps/backend/internal/middleware/ambient_latency.go`:
```go
package middleware

import (
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

func AmbientLatency(latencyMs, jitterMs int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if latencyMs == 0 && jitterMs == 0 {
			c.Next()
			return
		}
		jitter := 0
		if jitterMs > 0 {
			jitter = rand.Intn(jitterMs)
		}
		time.Sleep(time.Duration(latencyMs+jitter) * time.Millisecond)
		c.Next()
	}
}
```

`apps/backend/internal/middleware/ambient_error.go`:
```go
package middleware

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

func AmbientError(errorRate float64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if errorRate > 0 && rand.Float64() < errorRate {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "AMBIENT_SIMULATED_ERROR", "message": "Ambient simulated error"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

`apps/backend/internal/middleware/rate_limiter.go`:
```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is a process-wide token bucket: capacity and refill rate are
// both derived from limitPerMinute. rate.Limiter is safe for concurrent use.
func RateLimiter(limitPerMinute int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(float64(limitPerMinute)/60.0), limitPerMinute)
	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "Too many requests"},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: Mount the three middlewares**

In `apps/backend/internal/router/router.go`, add these lines directly after `r.Use(middleware.TestScenario(cfg.AppEnv))`, before the `/health` route:
```go
	r.Use(middleware.AmbientLatency(cfg.SimulateLatencyMs, cfg.SimulateLatencyJitterMs))
	r.Use(middleware.AmbientError(cfg.SimulateErrorRate))
	r.Use(middleware.RateLimiter(cfg.SimulateRateLimit))
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/middleware`

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/middleware apps/backend/internal/router
git commit -m "feat(backend): add ambient latency/error and rate-limit middleware"
```

---

### Task 11: Categories + Products routes

**Files:**
- Create: `apps/backend/internal/products/service.go`
- Create: `apps/backend/internal/products/handler.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/products/handler_test.go`

**Interfaces:**
- Consumes: `models.*` (Task 2), `apperror.*` (Task 1).
- Produces: `products.ListQuery{ Search, CategorySlug, Sort string; MinPrice, MaxPrice *int; Page, PageSize int }`, `products.ListResult{ Items []models.Product; Total int64; Page, PageSize int }` (JSON: `items, total, page, pageSize`), `products.ListProducts(db, q) (ListResult, error)`, `products.GetProductBySlug(db, slug) (*models.Product, error)` (from `service.go`). `products.RegisterRoutes(products, categories *gin.RouterGroup, db *gorm.DB)` — `GET /api/v1/products` (query params `search, category, minPrice, maxPrice, sort, page, pageSize`), `GET /api/v1/products/:slug`, `GET /api/v1/categories`.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/products/handler_test.go`:
```go
package products_test

import (
	"net/http"
	"os"
	"testing"

	"backend/internal/apitest"
	"backend/internal/dbtest"
	"backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Connect()
	os.Exit(m.Run())
}

func TestListProducts_ReturnsPaginatedResultsWithCategory(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Gadgets", Slug: "gadgets"}
	require.NoError(t, testDB.Create(&category).Error)
	require.NoError(t, testDB.Create(&models.Product{
		Name: "Widget A", Slug: "widget-a", Description: "desc", Price: 1000, Stock: 5, CategoryID: category.ID,
	}).Error)
	require.NoError(t, testDB.Create(&models.Product{
		Name: "Widget B", Slug: "widget-b", Description: "desc", Price: 2000, Stock: 0, CategoryID: category.ID,
	}).Error)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/products", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	data := apitest.DecodeData(t, rec)
	assert.Equal(t, float64(2), data["total"])
	items := data["items"].([]any)
	assert.Len(t, items, 2)
	first := items[0].(map[string]any)
	category0 := first["category"].(map[string]any)
	assert.Equal(t, "gadgets", category0["slug"])
}

func TestListProducts_FiltersByCategoryAndPriceRange(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	books := models.Category{Name: "Books", Slug: "books"}
	toys := models.Category{Name: "Toys", Slug: "toys"}
	require.NoError(t, testDB.Create(&books).Error)
	require.NoError(t, testDB.Create(&toys).Error)
	require.NoError(t, testDB.Create(&models.Product{Name: "Cheap Book", Slug: "cheap-book", Description: "d", Price: 500, Stock: 1, CategoryID: books.ID}).Error)
	require.NoError(t, testDB.Create(&models.Product{Name: "Expensive Book", Slug: "expensive-book", Description: "d", Price: 5000, Stock: 1, CategoryID: books.ID}).Error)
	require.NoError(t, testDB.Create(&models.Product{Name: "Toy", Slug: "toy", Description: "d", Price: 1000, Stock: 1, CategoryID: toys.ID}).Error)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/products?category=books&maxPrice=1000", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	data := apitest.DecodeData(t, rec)
	items := data["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, "cheap-book", items[0].(map[string]any)["slug"])
}

func TestListProducts_SortsByPriceAscending(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Misc", Slug: "misc"}
	require.NoError(t, testDB.Create(&category).Error)
	require.NoError(t, testDB.Create(&models.Product{Name: "B", Slug: "b", Description: "d", Price: 2000, Stock: 1, CategoryID: category.ID}).Error)
	require.NoError(t, testDB.Create(&models.Product{Name: "A", Slug: "a", Description: "d", Price: 1000, Stock: 1, CategoryID: category.ID}).Error)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/products?sort=price_asc", nil)

	data := apitest.DecodeData(t, rec)
	items := data["items"].([]any)
	require.Len(t, items, 2)
	assert.Equal(t, "a", items[0].(map[string]any)["slug"])
	assert.Equal(t, "b", items[1].(map[string]any)["slug"])
}

func TestGetProductBySlug_ReturnsProductWithCategoryAndReviews(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Gadgets", Slug: "gadgets"}
	require.NoError(t, testDB.Create(&category).Error)
	require.NoError(t, testDB.Create(&models.Product{
		Name: "Widget", Slug: "widget", Description: "desc", Price: 1000, Stock: 5, CategoryID: category.ID,
	}).Error)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/products/widget", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	data := apitest.DecodeData(t, rec)
	assert.Equal(t, "Widget", data["name"])
	assert.Empty(t, data["reviews"])
}

func TestGetProductBySlug_ReturnsNotFoundForUnknownSlug(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/products/does-not-exist", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "PRODUCT_NOT_FOUND", apitest.DecodeError(t, rec)["code"])
}

func TestListCategories_ReturnsAllCategories(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	require.NoError(t, testDB.Create(&models.Category{Name: "Gadgets", Slug: "gadgets"}).Error)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/categories", nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, apitest.DecodeDataList(t, rec), 1)
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `products` package does not exist yet

- [ ] **Step 3: Implement the service**

`apps/backend/internal/products/service.go`:
```go
package products

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

type ListQuery struct {
	Search       string
	CategorySlug string
	MinPrice     *int
	MaxPrice     *int
	Sort         string // "price_asc" | "price_desc" | "newest"
	Page         int
	PageSize     int
}

type ListResult struct {
	Items    []models.Product `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

func ListProducts(db *gorm.DB, q ListQuery) (ListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 12
	}

	query := db.Model(&models.Product{})
	if q.Search != "" {
		query = query.Where("name ILIKE ?", "%"+q.Search+"%")
	}
	if q.CategorySlug != "" {
		var category models.Category
		if err := db.Where("slug = ?", q.CategorySlug).First(&category).Error; err != nil {
			return ListResult{Items: []models.Product{}, Total: 0, Page: page, PageSize: pageSize}, nil
		}
		query = query.Where("category_id = ?", category.ID)
	}
	if q.MinPrice != nil {
		query = query.Where("price >= ?", *q.MinPrice)
	}
	if q.MaxPrice != nil {
		query = query.Where("price <= ?", *q.MaxPrice)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return ListResult{}, err
	}

	switch q.Sort {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	default:
		query = query.Order("created_at DESC")
	}

	var items []models.Product
	if err := query.Preload("Category").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return ListResult{}, err
	}

	return ListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func GetProductBySlug(db *gorm.DB, slug string) (*models.Product, error) {
	var product models.Product
	err := db.Preload("Category").Preload("Reviews").Where("slug = ?", slug).First(&product).Error
	if err != nil {
		return nil, apperror.New(404, "PRODUCT_NOT_FOUND", "Product \""+slug+"\" not found")
	}
	return &product, nil
}
```

- [ ] **Step 4: Implement the handler**

`apps/backend/internal/products/handler.go`:
```go
package products

import (
	"net/http"
	"strconv"

	"backend/internal/apperror"
	"backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(products *gin.RouterGroup, categories *gin.RouterGroup, db *gorm.DB) {
	products.GET("", func(c *gin.Context) {
		q := ListQuery{
			Search:       c.Query("search"),
			CategorySlug: c.Query("category"),
			Sort:         c.Query("sort"),
		}
		if v := c.Query("minPrice"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				q.MinPrice = &n
			}
		}
		if v := c.Query("maxPrice"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				q.MaxPrice = &n
			}
		}
		if v := c.Query("page"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				q.Page = n
			}
		}
		if v := c.Query("pageSize"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				q.PageSize = n
			}
		}

		result, err := ListProducts(db, q)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	products.GET("/:slug", func(c *gin.Context) {
		product, err := GetProductBySlug(db, c.Param("slug"))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": product})
	})

	categories.GET("", func(c *gin.Context) {
		var list []models.Category
		if err := db.Find(&list).Error; err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})
}
```

- [ ] **Step 5: Mount the routers**

In `apps/backend/internal/router/router.go`, add the import:
```go
"backend/internal/products"
```
and add these lines directly after the `auth.RegisterRoutes(...)` line:
```go
	products.RegisterRoutes(v1.Group("/products"), v1.Group("/categories"), db)
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/products`

- [ ] **Step 7: Commit**

```bash
git add apps/backend/internal/products apps/backend/internal/router
git commit -m "feat(backend): add products and categories endpoints"
```

---

### Task 12: Reviews routes

**Files:**
- Create: `apps/backend/internal/reviews/service.go`
- Create: `apps/backend/internal/reviews/handler.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/reviews/handler_test.go`

**Interfaces:**
- Consumes: `models.*` (Task 2), `middleware.RequireAuth` (Task 5), `apperror.*` (Task 1).
- Produces: `reviews.ListReviewsForProduct(db, productID) ([]models.Review, error)`, `reviews.CreateReview(db, productID, userID string, rating int, comment string) (*models.Review, error)` (from `service.go`). `reviews.RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config)` mounted at `/api/v1/reviews` (`GET ?productId=`, `POST` requires auth).

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/reviews/handler_test.go`:
```go
package reviews_test

import (
	"net/http"
	"os"
	"testing"

	"backend/internal/apitest"
	"backend/internal/dbtest"
	"backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Connect()
	os.Exit(m.Run())
}

func registerAndGetToken(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": "Password123!", "name": "Reviewer",
	})
	return apitest.DecodeData(t, rec)["accessToken"].(string)
}

func TestCreateReview_RejectsWithoutAuth(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Books", Slug: "books"}
	require.NoError(t, testDB.Create(&category).Error)
	product := models.Product{Name: "Book", Slug: "book", Description: "d", Price: 100, Stock: 1, CategoryID: category.ID}
	require.NoError(t, testDB.Create(&product).Error)

	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/reviews", map[string]any{
		"productId": product.ID, "rating": 5, "comment": "Great!",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateReview_CreatesAndListsByProductID(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Books", Slug: "books"}
	require.NoError(t, testDB.Create(&category).Error)
	product := models.Product{Name: "Book", Slug: "book", Description: "d", Price: 100, Stock: 1, CategoryID: category.ID}
	require.NoError(t, testDB.Create(&product).Error)
	accessToken := registerAndGetToken(t, r, "reviewer@example.com")

	createRec := apitest.DoJSONAuth(t, r, http.MethodPost, "/api/v1/reviews", accessToken, map[string]any{
		"productId": product.ID, "rating": 5, "comment": "Great!",
	})
	assert.Equal(t, http.StatusCreated, createRec.Code)

	listRec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/reviews?productId="+product.ID, nil)
	assert.Equal(t, http.StatusOK, listRec.Code)
	list := apitest.DecodeDataList(t, listRec)
	require.Len(t, list, 1)
	assert.Equal(t, "Great!", list[0].(map[string]any)["comment"])
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `reviews` package does not exist yet

- [ ] **Step 3: Implement the service**

`apps/backend/internal/reviews/service.go`:
```go
package reviews

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

func ListReviewsForProduct(db *gorm.DB, productID string) ([]models.Review, error) {
	var list []models.Review
	err := db.Preload("User").Where("product_id = ?", productID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func CreateReview(db *gorm.DB, productID, userID string, rating int, comment string) (*models.Review, error) {
	review := models.Review{ProductID: productID, UserID: userID, Rating: rating, Comment: comment}
	if err := db.Create(&review).Error; err != nil {
		return nil, err
	}
	return &review, nil
}
```

- [ ] **Step 4: Implement the handler**

`apps/backend/internal/reviews/handler.go`:
```go
package reviews

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createReviewRequest struct {
	ProductID string `json:"productId" binding:"required"`
	Rating    int    `json:"rating" binding:"required,min=1,max=5"`
	Comment   string `json:"comment" binding:"required"`
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.GET("", func(c *gin.Context) {
		productID := c.Query("productId")
		if productID == "" {
			apperror.RespondError(c, apperror.New(http.StatusBadRequest, "VALIDATION_ERROR", "productId is required"))
			return
		}
		list, err := ListReviewsForProduct(db, productID)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})

	rg.POST("", middleware.RequireAuth(cfg.JWTAccessSecret), func(c *gin.Context) {
		var req createReviewRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		review, err := CreateReview(db, req.ProductID, userID, req.Rating, req.Comment)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": review})
	})
}
```

- [ ] **Step 5: Mount the router**

In `apps/backend/internal/router/router.go`, add the import:
```go
"backend/internal/reviews"
```
and add this line directly after the `products.RegisterRoutes(...)` line:
```go
	reviews.RegisterRoutes(v1.Group("/reviews"), db, cfg)
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/reviews`

- [ ] **Step 7: Commit**

```bash
git add apps/backend/internal/reviews apps/backend/internal/router
git commit -m "feat(backend): add reviews endpoints"
```

---

### Task 13: Cart routes

**Files:**
- Create: `apps/backend/internal/cart/service.go`
- Create: `apps/backend/internal/cart/handler.go`
- Modify: `apps/backend/internal/router/router.go`
- Test: `apps/backend/internal/cart/handler_test.go`

**Interfaces:**
- Consumes: `models.*` (Task 2), `middleware.RequireAuth` (Task 5), `apperror.*` (Task 1).
- Produces: `cart.GetCart(db, userID) (*models.Cart, error)`, `cart.AddItem(db, userID, productID string, quantity int) (*models.CartItem, error)`, `cart.UpdateItem(db, userID, itemID string, quantity int) (*models.CartItem, error)`, `cart.RemoveItem(db, userID, itemID string) error` (from `service.go`) — `GetCart`/checkout flow reused by Task 14. `cart.RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config)` mounted at `/api/v1/cart`, all routes require auth: `GET ""`, `POST /items`, `PATCH /items/:itemId`, `DELETE /items/:itemId`.

- [ ] **Step 1: Write the failing tests**

`apps/backend/internal/cart/handler_test.go`:
```go
package cart_test

import (
	"net/http"
	"os"
	"testing"

	"backend/internal/apitest"
	"backend/internal/dbtest"
	"backend/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	testDB = dbtest.Connect()
	os.Exit(m.Run())
}

func registerAndGetToken(t *testing.T, r http.Handler, email string) string {
	t.Helper()
	rec := apitest.DoJSON(t, r, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": email, "password": "Password123!", "name": "Shopper",
	})
	return apitest.DecodeData(t, rec)["accessToken"].(string)
}

func TestCart_RejectsWithoutAuth(t *testing.T) {
	r := apitest.NewRouter(t, testDB)

	rec := apitest.DoJSON(t, r, http.MethodGet, "/api/v1/cart", nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCart_AddUpdateAndRemoveItem(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Gadgets", Slug: "gadgets"}
	require.NoError(t, testDB.Create(&category).Error)
	product := models.Product{Name: "Widget", Slug: "widget", Description: "d", Price: 1000, Stock: 10, CategoryID: category.ID}
	require.NoError(t, testDB.Create(&product).Error)
	accessToken := registerAndGetToken(t, r, "shopper@example.com")

	addRec := apitest.DoJSONAuth(t, r, http.MethodPost, "/api/v1/cart/items", accessToken, map[string]any{
		"productId": product.ID, "quantity": 2,
	})
	assert.Equal(t, http.StatusCreated, addRec.Code)
	itemID := apitest.DecodeData(t, addRec)["id"].(string)

	cartRec := apitest.DoJSONAuth(t, r, http.MethodGet, "/api/v1/cart", accessToken, nil)
	items := apitest.DecodeData(t, cartRec)["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, float64(2), items[0].(map[string]any)["quantity"])

	updateRec := apitest.DoJSONAuth(t, r, http.MethodPatch, "/api/v1/cart/items/"+itemID, accessToken, map[string]any{
		"quantity": 5,
	})
	assert.Equal(t, http.StatusOK, updateRec.Code)
	assert.Equal(t, float64(5), apitest.DecodeData(t, updateRec)["quantity"])

	deleteRec := apitest.DoJSONAuth(t, r, http.MethodDelete, "/api/v1/cart/items/"+itemID, accessToken, nil)
	assert.Equal(t, http.StatusNoContent, deleteRec.Code)

	finalCartRec := apitest.DoJSONAuth(t, r, http.MethodGet, "/api/v1/cart", accessToken, nil)
	finalItems := apitest.DecodeData(t, finalCartRec)["items"].([]any)
	assert.Len(t, finalItems, 0)
}

func TestCart_AddingSameProductTwiceIncrementsQuantity(t *testing.T) {
	r := apitest.NewRouter(t, testDB)
	category := models.Category{Name: "Gadgets", Slug: "gadgets"}
	require.NoError(t, testDB.Create(&category).Error)
	product := models.Product{Name: "Widget", Slug: "widget", Description: "d", Price: 1000, Stock: 10, CategoryID: category.ID}
	require.NoError(t, testDB.Create(&product).Error)
	accessToken := registerAndGetToken(t, r, "shopper2@example.com")

	apitest.DoJSONAuth(t, r, http.MethodPost, "/api/v1/cart/items", accessToken, map[string]any{"productId": product.ID, "quantity": 1})
	apitest.DoJSONAuth(t, r, http.MethodPost, "/api/v1/cart/items", accessToken, map[string]any{"productId": product.ID, "quantity": 2})

	cartRec := apitest.DoJSONAuth(t, r, http.MethodGet, "/api/v1/cart", accessToken, nil)
	items := apitest.DecodeData(t, cartRec)["items"].([]any)
	require.Len(t, items, 1)
	assert.Equal(t, float64(3), items[0].(map[string]any)["quantity"])
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd apps/backend && go test ./...`
Expected: FAIL — build error, `cart` package does not exist yet

- [ ] **Step 3: Implement the service**

`apps/backend/internal/cart/service.go`:
```go
package cart

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

func GetCart(db *gorm.DB, userID string) (*models.Cart, error) {
	var c models.Cart
	if err := db.Preload("Items.Product").Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, apperror.New(404, "CART_NOT_FOUND", "Cart not found for user")
	}
	return &c, nil
}

func AddItem(db *gorm.DB, userID, productID string, quantity int) (*models.CartItem, error) {
	var c models.Cart
	if err := db.Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, apperror.New(404, "CART_NOT_FOUND", "Cart not found for user")
	}
	var product models.Product
	if err := db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, apperror.New(404, "PRODUCT_NOT_FOUND", "Product not found")
	}

	var existing models.CartItem
	err := db.Where("cart_id = ? AND product_id = ?", c.ID, productID).First(&existing).Error
	if err == nil {
		existing.Quantity += quantity
		if err := db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	item := models.CartItem{CartID: c.ID, ProductID: productID, Quantity: quantity}
	if err := db.Create(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func findOwnedCartItem(db *gorm.DB, userID, itemID string) (*models.CartItem, error) {
	var item models.CartItem
	if err := db.Where("id = ?", itemID).First(&item).Error; err != nil {
		return nil, apperror.New(404, "CART_ITEM_NOT_FOUND", "Cart item not found")
	}
	var c models.Cart
	if err := db.Where("id = ?", item.CartID).First(&c).Error; err != nil || c.UserID != userID {
		return nil, apperror.New(404, "CART_ITEM_NOT_FOUND", "Cart item not found")
	}
	return &item, nil
}

func UpdateItem(db *gorm.DB, userID, itemID string, quantity int) (*models.CartItem, error) {
	item, err := findOwnedCartItem(db, userID, itemID)
	if err != nil {
		return nil, err
	}
	item.Quantity = quantity
	if err := db.Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func RemoveItem(db *gorm.DB, userID, itemID string) error {
	item, err := findOwnedCartItem(db, userID, itemID)
	if err != nil {
		return err
	}
	return db.Delete(item).Error
}
```

- [ ] **Step 4: Implement the handler**

`apps/backend/internal/cart/handler.go`:
```go
package cart

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type addItemRequest struct {
	ProductID string `json:"productId" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
}

type updateItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=1"`
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.Use(middleware.RequireAuth(cfg.JWTAccessSecret))

	rg.GET("", func(c *gin.Context) {
		userID := c.MustGet("userID").(string)
		result, err := GetCart(db, userID)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	})

	rg.POST("/items", func(c *gin.Context) {
		var req addItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		item, err := AddItem(db, userID, req.ProductID, req.Quantity)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": item})
	})

	rg.PATCH("/items/:itemId", func(c *gin.Context) {
		var req updateItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		item, err := UpdateItem(db, userID, c.Param("itemId"), req.Quantity)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": item})
	})

	rg.DELETE("/items/:itemId", func(c *gin.Context) {
		userID := c.MustGet("userID").(string)
		if err := RemoveItem(db, userID, c.Param("itemId")); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
```

- [ ] **Step 5: Mount the router**

In `apps/backend/internal/router/router.go`, add the import:
```go
"backend/internal/cart"
```
and add this line directly after the `reviews.RegisterRoutes(...)` line:
```go
	cart.RegisterRoutes(v1.Group("/cart"), db, cfg)
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd apps/backend && go test ./...`
Expected: PASS — `ok backend/internal/cart`

- [ ] **Step 7: Commit**

```bash
git add apps/backend/internal/cart apps/backend/internal/router
git commit -m "feat(backend): add cart endpoints"
```

---
