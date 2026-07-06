package admin

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

type ProductInput struct {
	Name        string
	Slug        string
	Description string
	Price       int
	Stock       int
	Images      []string
	CategoryID  string
}

func CreateProduct(db *gorm.DB, input ProductInput) (*models.Product, error) {
	product := models.Product{
		Name: input.Name, Slug: input.Slug, Description: input.Description,
		Price: input.Price, Stock: input.Stock, Images: input.Images, CategoryID: input.CategoryID,
	}
	if err := db.Create(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

type ProductUpdateInput struct {
	Name        *string
	Slug        *string
	Description *string
	Price       *int
	Stock       *int
	Images      *[]string
	CategoryID  *string
}

func UpdateProduct(db *gorm.DB, id string, input ProductUpdateInput) (*models.Product, error) {
	var product models.Product
	if err := db.Where("id = ?", id).First(&product).Error; err != nil {
		return nil, apperror.New(404, "PRODUCT_NOT_FOUND", "Product not found")
	}

	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.Slug != nil {
		product.Slug = *input.Slug
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.Stock != nil {
		product.Stock = *input.Stock
	}
	if input.Images != nil {
		product.Images = *input.Images
	}
	if input.CategoryID != nil {
		product.CategoryID = *input.CategoryID
	}

	if err := db.Save(&product).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func DeleteProduct(db *gorm.DB, id string) error {
	var product models.Product
	if err := db.Where("id = ?", id).First(&product).Error; err != nil {
		return apperror.New(404, "PRODUCT_NOT_FOUND", "Product not found")
	}
	return db.Delete(&product).Error
}

func ListAllOrders(db *gorm.DB) ([]models.Order, error) {
	var list []models.Order
	err := db.Preload("Items.Product").Preload("Payment").Order("created_at DESC").Find(&list).Error
	return list, err
}

func UpdateOrderStatus(db *gorm.DB, orderID string, status models.OrderStatus) (*models.Order, error) {
	var order models.Order
	if err := db.Where("id = ?", orderID).First(&order).Error; err != nil {
		return nil, apperror.New(404, "ORDER_NOT_FOUND", "Order not found")
	}
	if err := db.Model(&order).Update("status", status).Error; err != nil {
		return nil, err
	}
	order.Status = status
	return &order, nil
}
