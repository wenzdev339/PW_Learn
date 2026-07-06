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
	testutils.RegisterRoutes(r.Group("/test"), db, cfg)

	return r
}
