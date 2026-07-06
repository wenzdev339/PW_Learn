package admin

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/config"
	"backend/internal/middleware"
	"backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createProductRequest struct {
	Name        string   `json:"name" binding:"required"`
	Slug        string   `json:"slug" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Price       int      `json:"price" binding:"required,min=0"`
	Stock       int      `json:"stock" binding:"min=0"`
	Images      []string `json:"images" binding:"required,min=1"`
	CategoryID  string   `json:"categoryId" binding:"required"`
}

type updateProductRequest struct {
	Name        *string   `json:"name"`
	Slug        *string   `json:"slug"`
	Description *string   `json:"description"`
	Price       *int      `json:"price"`
	Stock       *int      `json:"stock"`
	Images      *[]string `json:"images"`
	CategoryID  *string   `json:"categoryId"`
}

type updateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=PENDING PAID SHIPPED DELIVERED CANCELLED"`
}

func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, cfg config.Config) {
	rg.Use(middleware.RequireAuth(cfg.JWTAccessSecret), middleware.RequireAdmin())

	rg.POST("/products", func(c *gin.Context) {
		var req createProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		product, err := CreateProduct(db, ProductInput{
			Name: req.Name, Slug: req.Slug, Description: req.Description,
			Price: req.Price, Stock: req.Stock, Images: req.Images, CategoryID: req.CategoryID,
		})
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"data": product})
	})

	rg.PATCH("/products/:id", func(c *gin.Context) {
		var req updateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		product, err := UpdateProduct(db, c.Param("id"), ProductUpdateInput{
			Name: req.Name, Slug: req.Slug, Description: req.Description,
			Price: req.Price, Stock: req.Stock, Images: req.Images, CategoryID: req.CategoryID,
		})
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": product})
	})

	rg.DELETE("/products/:id", func(c *gin.Context) {
		if err := DeleteProduct(db, c.Param("id")); err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	rg.GET("/orders", func(c *gin.Context) {
		list, err := ListAllOrders(db)
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": list})
	})

	rg.PATCH("/orders/:orderId/status", func(c *gin.Context) {
		var req updateOrderStatusRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			apperror.RespondValidationError(c, err)
			return
		}
		order, err := UpdateOrderStatus(db, c.Param("orderId"), models.OrderStatus(req.Status))
		if err != nil {
			apperror.RespondError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": order})
	})
}
