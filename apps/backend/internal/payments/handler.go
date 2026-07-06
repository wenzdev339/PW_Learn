package payments

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type payRequest struct {
	OrderID    string `json:"orderId" binding:"required"`
	CardNumber string `json:"cardNumber" binding:"required,min=4"`
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.POST("/mock", middleware.RequireAuth(cfg.JWTAccessSecret), func(c *gin.Context) {
		var req payRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		payment, err := PayForOrder(db, userID, req.OrderID, req.CardNumber)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": payment})
	})
}
