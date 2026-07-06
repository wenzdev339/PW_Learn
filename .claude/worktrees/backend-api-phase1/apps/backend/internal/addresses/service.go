package addresses

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type CreateInput struct {
	Label      string
	Line1      string
	Line2      string
	City       string
	PostalCode string
	Country    string
	IsDefault  bool
}

func CreateAddress(db *gorm.DB, userID string, input CreateInput) (*models.Address, error) {
	address := models.Address{
		UserID: userID, Label: input.Label, Line1: input.Line1, Line2: input.Line2,
		City: input.City, PostalCode: input.PostalCode, Country: input.Country, IsDefault: input.IsDefault,
	}
	if err := db.Create(&address).Error; err != nil {
		return nil, err
	}
	return &address, nil
}

func ListAddresses(db *gorm.DB, userID string) ([]models.Address, error) {
	var list []models.Address
	err := db.Where("user_id = ?", userID).Find(&list).Error
	return list, err
}
