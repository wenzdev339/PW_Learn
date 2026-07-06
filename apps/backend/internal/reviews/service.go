package reviews

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

func ListReviewsForProduct(db *gorm.DB, productID string) ([]models.Review, error) {
	var list []models.Review
	err := db.Preload("User").Where("product_id = ?", productID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func CreateReview(db *gorm.DB, productID, userID string, rating int, comment string) (*models.Review, error) {
	review := models.Review{ProductID: productID, UserID: userID, Rating: rating, Comment: comment}
	if err := db.Create(&review).Error; err != nil {
		return nil, err
	}
	return &review, nil
}
