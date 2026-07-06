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
