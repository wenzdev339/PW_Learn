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
