package cart

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

func GetCart(db *gorm.DB, userID string) (*models.Cart, error) {
	var c models.Cart
	if err := db.Preload("Items.Product").Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, apperror.New(404, "CART_NOT_FOUND", "Cart not found for user")
	}
	return &c, nil
}

func AddItem(db *gorm.DB, userID, productID string, quantity int) (*models.CartItem, error) {
	var c models.Cart
	if err := db.Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, apperror.New(404, "CART_NOT_FOUND", "Cart not found for user")
	}
	var product models.Product
	if err := db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, apperror.New(404, "PRODUCT_NOT_FOUND", "Product not found")
	}

	var existing models.CartItem
	err := db.Where("cart_id = ? AND product_id = ?", c.ID, productID).First(&existing).Error
	if err == nil {
		existing.Quantity += quantity
		if err := db.Save(&existing).Error; err != nil {
			return nil, err
		}
		existing.Product = product
		return &existing, nil
	}

	item := models.CartItem{CartID: c.ID, ProductID: productID, Quantity: quantity}
	if err := db.Create(&item).Error; err != nil {
		return nil, err
	}
	item.Product = product
	return &item, nil
}

func findOwnedCartItem(db *gorm.DB, userID, itemID string) (*models.CartItem, error) {
	var item models.CartItem
	if err := db.Where("id = ?", itemID).First(&item).Error; err != nil {
		return nil, apperror.New(404, "CART_ITEM_NOT_FOUND", "Cart item not found")
	}
	var c models.Cart
	if err := db.Where("id = ?", item.CartID).First(&c).Error; err != nil || c.UserID != userID {
		return nil, apperror.New(404, "CART_ITEM_NOT_FOUND", "Cart item not found")
	}
	return &item, nil
}

func UpdateItem(db *gorm.DB, userID, itemID string, quantity int) (*models.CartItem, error) {
	item, err := findOwnedCartItem(db, userID, itemID)
	if err != nil {
		return nil, err
	}
	item.Quantity = quantity
	if err := db.Save(item).Error; err != nil {
		return nil, err
	}
	if err := db.Preload("Product").First(item, "id = ?", item.ID).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func RemoveItem(db *gorm.DB, userID, itemID string) error {
	item, err := findOwnedCartItem(db, userID, itemID)
	if err != nil {
		return err
	}
	return db.Delete(item).Error
}
