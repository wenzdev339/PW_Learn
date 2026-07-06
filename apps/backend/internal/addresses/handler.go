package addresses

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createAddressRequest struct {
	Label      string `json:"label" binding:"required"`
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2"`
	City       string `json:"city" binding:"required"`
	PostalCode string `json:"postalCode" binding:"required"`
	Country    string `json:"country" binding:"required"`
	IsDefault  bool   `json:"isDefault"`
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.Use(middleware.RequireAuth(cfg.JWTAccessSecret))

	rg.GET("", func(c *gin.Context) {
		userID := c.MustGet("userID").(string)
		list, err := ListAddresses(db, userID)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})

	rg.POST("", func(c *gin.Context) {
		var req createAddressRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		userID := c.MustGet("userID").(string)
		address, err := CreateAddress(db, userID, CreateInput{
			Label: req.Label, Line1: req.Line1, Line2: req.Line2,
			City: req.City, PostalCode: req.PostalCode, Country: req.Country, IsDefault: req.IsDefault,
		})
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": address})
	})
}
