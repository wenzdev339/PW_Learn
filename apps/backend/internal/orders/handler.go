package orders

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type checkoutRequest struct {
	ShippingAddressID string `json:"shippingAddressId" binding:"required"`
}

// RegisterCheckoutRoutes mounts POST "" on rg (expected "/api/v1/checkout").
func RegisterCheckoutRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.POST("", middleware.RequireAuth(cfg.JWTAccessSecret), func(c *gin.Context) {
		var req checkoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		order, err := Checkout(db, userID, req.ShippingAddressID)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": order})
	})
}

// RegisterOrdersRoutes mounts GET "" and GET "/:orderId" on rg (expected
// "/api/v1/orders"), both requiring auth.
func RegisterOrdersRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.Use(middleware.RequireAuth(cfg.JWTAccessSecret))

	rg.GET("", func(c *gin.Context) {
		userID := c.MustGet("userID").(string)
		list, err := ListOrders(db, userID)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})

	rg.GET("/:orderId", func(c *gin.Context) {
		userID := c.MustGet("userID").(string)
		order, err := GetOrder(db, userID, c.Param("orderId"))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	})
}
