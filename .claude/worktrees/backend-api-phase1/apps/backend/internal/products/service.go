package products

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

type ListQuery struct {
	Search       string
	CategorySlug string
	MinPrice     *int
	MaxPrice     *int
	Sort         string // "price_asc" | "price_desc" | "newest"
	Page         int
	PageSize     int
}

type ListResult struct {
	Items    []models.Product `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

func ListProducts(db *gorm.DB, q ListQuery) (ListResult, error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 12
	}

	query := db.Model(&models.Product{})
	if q.Search != "" {
		query = query.Where("name ILIKE ?", "%"+q.Search+"%")
	}
	if q.CategorySlug != "" {
		var category models.Category
		if err := db.Where("slug = ?", q.CategorySlug).First(&category).Error; err != nil {
			return ListResult{Items: []models.Product{}, Total: 0, Page: page, PageSize: pageSize}, nil
		}
		query = query.Where("category_id = ?", category.ID)
	}
	if q.MinPrice != nil {
		query = query.Where("price >= ?", *q.MinPrice)
	}
	if q.MaxPrice != nil {
		query = query.Where("price <= ?", *q.MaxPrice)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return ListResult{}, err
	}

	switch q.Sort {
	case "price_asc":
		query = query.Order("price ASC")
	case "price_desc":
		query = query.Order("price DESC")
	default:
		query = query.Order("created_at DESC")
	}

	var items []models.Product
	if err := query.Preload("Category").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return ListResult{}, err
	}

	return ListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func GetProductBySlug(db *gorm.DB, slug string) (*models.Product, error) {
	var product models.Product
	err := db.Preload("Category").Preload("Reviews").Where("slug = ?", slug).First(&product).Error
	if err != nil {
		return nil, apperror.New(404, "PRODUCT_NOT_FOUND", "Product \""+slug+"\" not found")
	}
	return &product, nil
}
