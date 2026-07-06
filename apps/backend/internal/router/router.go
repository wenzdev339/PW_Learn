package router

import (
	"backend/internal/addresses"
	"backend/internal/admin"
	"backend/internal/auth"
	"backend/internal/cart"
	"backend/internal/config"
	"backend/internal/middleware"
	"backend/internal/orders"
	"backend/internal/payments"
	"backend/internal/products"
	"backend/internal/reviews"
	"backend/internal/testutils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// New ประกอบทุกชิ้นส่วนของแอปเข้าด้วยกันเป็น *gin.Engine ตัวเดียว — เรียกครั้ง
// เดียวตอน server เริ่มทำงาน (cmd/server/main.go) และเรียกซ้ำได้ในทุกเทส
// (internal/apitest ใช้ฟังก์ชันนี้สร้าง router ใหม่ทุกเทส เพื่อไม่ให้ state
// ของเทสก่อนหน้าหลงเหลือมาปนกัน)
//
// ลำดับการ r.Use(...) มีผลจริง ๆ: Gin จะรัน middleware ตามลำดับที่ผูกไว้
// ก่อน-หลังเสมอ ในนี้จงใจเรียงเป็น:
//  1. Recovery      — กัน panic ทำให้ทั้ง process ล่ม (ตอบ 500 แทน)
//  2. CORS          — อนุญาต frontend เรียกข้าม origin ได้
//  3. TestScenario  — ถ้ามี header บังคับพฤติกรรม ให้ตัดจบตรงนี้เลยก่อนถึง
//     ambient middleware ตัวถัดไป (ดูคอมเมนต์ใน middleware/test_scenario.go)
//  4-6. Ambient latency/error/rate-limit — จำลองสภาพเครือข่ายแบบสุ่ม (ใช้เมื่อ
//     ไม่มี header บังคับจาก TestScenario)
func New(cfg config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Test-Scenario", "X-Test-Run-Id"},
	}))

	r.Use(middleware.TestScenario(cfg.AppEnv))
	r.Use(middleware.AmbientLatency(cfg.SimulateLatencyMs, cfg.SimulateLatencyJitterMs))
	r.Use(middleware.AmbientError(cfg.SimulateErrorRate))
	r.Use(middleware.RateLimiter(cfg.SimulateRateLimit))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok"}})
	})

	// แต่ละฟีเจอร์เป็นคนละ package กัน (auth, products, cart, ...) แล้วมา
	// "ประกาศตัวเอง" กับ router ผ่าน RegisterRoutes — router.go เองไม่รู้จัก
	// รายละเอียดข้างในแต่ละฟีเจอร์เลย รู้แค่ว่าจะ mount ไว้ตรงไหน
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1.Group("/auth"), db, cfg)
	products.RegisterRoutes(v1.Group("/products"), v1.Group("/categories"), db)
	reviews.RegisterRoutes(v1.Group("/reviews"), db, cfg)
	cart.RegisterRoutes(v1.Group("/cart"), db, cfg)
	addresses.RegisterRoutes(v1.Group("/addresses"), db, cfg)
	orders.RegisterCheckoutRoutes(v1.Group("/checkout"), db, cfg)
	orders.RegisterOrdersRoutes(v1.Group("/orders"), db, cfg)
	payments.RegisterRoutes(v1.Group("/payments"), db, cfg)
	admin.RegisterRoutes(v1.Group("/admin"), db, cfg)
	// /test/reset ไม่ได้อยู่ใต้ /api/v1 เพราะไม่ใช่ endpoint ทางธุรกิจ —
	// เป็นแค่เครื่องมือช่วยเทส (ดู internal/testutils/handler.go ว่าทำไมมัน
	// ถูกปิดกั้นไม่ให้เรียกตอน production)
	testutils.RegisterRoutes(r.Group("/test"), db, cfg)

	return r
}
