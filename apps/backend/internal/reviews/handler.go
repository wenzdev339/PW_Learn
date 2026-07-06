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
