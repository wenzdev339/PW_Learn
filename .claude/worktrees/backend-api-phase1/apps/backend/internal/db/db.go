package db

import (
	"backend/internal/config"
	"backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(cfg config.Config) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Address{},
		&models.Category{},
		&models.Product{},
		&models.Review{},
		&models.Cart{},
		&models.CartItem{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
	)
}
