package orders

import (
	"backend/internal/apperror"
	"backend/internal/models"

	"gorm.io/gorm"
)

// Checkout แปลงตะกร้าปัจจุบันของ user ให้กลายเป็น Order สถานะ PENDING
// ขั้นตอนเปลี่ยนแปลงข้อมูลหลายตาราง (สร้าง order, สร้าง order item ทีละชิ้น,
// ตัด stock, ล้างตะกร้า) จึงต้องทำใน db.Transaction เดียวกันทั้งหมด — ถ้าขั้น
// ไหนพัง (เช่น ตัด stock ไม่สำเร็จ) ทุกอย่างที่ทำไปก่อนหน้าจะถูก rollback
// กลับหมด ไม่ให้เกิดสภาพ "สร้าง order แล้วแต่ไม่ได้ตัด stock" ค้างอยู่
func Checkout(db *gorm.DB, userID, shippingAddressID string) (*models.Order, error) {
	var c models.Cart
	if err := db.Preload("Items.Product").Where("user_id = ?", userID).First(&c).Error; err != nil {
		return nil, apperror.New(400, "CART_EMPTY", "Cart is empty")
	}
	if len(c.Items) == 0 {
		return nil, apperror.New(400, "CART_EMPTY", "Cart is empty")
	}

	var address models.Address
	if err := db.Where("id = ?", shippingAddressID).First(&address).Error; err != nil || address.UserID != userID {
		return nil, apperror.New(404, "ADDRESS_NOT_FOUND", "Shipping address not found")
	}

	for _, item := range c.Items {
		if item.Product.Stock < item.Quantity {
			return nil, apperror.New(409, "OUT_OF_STOCK", "Insufficient stock for \""+item.Product.Name+"\"")
		}
	}

	totalAmount := 0
	for _, item := range c.Items {
		totalAmount += item.Product.Price * item.Quantity
	}

	var order models.Order
	err := db.Transaction(func(tx *gorm.DB) error {
		order = models.Order{
			UserID:            userID,
			Status:            models.OrderStatusPending,
			TotalAmount:       totalAmount,
			ShippingAddressID: shippingAddressID,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		for _, item := range c.Items {
			orderItem := models.OrderItem{
				OrderID:         order.ID,
				ProductID:       item.ProductID,
				Quantity:        item.Quantity,
				PriceAtPurchase: item.Product.Price,
			}
			if err := tx.Create(&orderItem).Error; err != nil {
				return err
			}
			// gorm.Expr("stock - ?", ...) สั่งให้ Postgres คำนวณ "stock - N"
			// เองที่ฐานข้อมูลโดยตรง (ไม่ใช่ดึงค่ามาลบใน Go แล้วเซฟกลับ) กัน
			// race condition เวลามีคนสั่งซื้อสินค้าตัวเดียวกันพร้อมกันหลาย
			// คนแล้ว stock เพี้ยนจากการ overwrite ทับกัน
			if err := tx.Model(&models.Product{}).Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
				return err
			}
		}

		return tx.Where("cart_id = ?", c.ID).Delete(&models.CartItem{}).Error
	})
	if err != nil {
		return nil, err
	}

	if err := db.Preload("Items.Product").First(&order, "id = ?", order.ID).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func ListOrders(db *gorm.DB, userID string) ([]models.Order, error) {
	var list []models.Order
	err := db.Preload("Items.Product").Preload("Payment").Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error
	return list, err
}

func GetOrder(db *gorm.DB, userID, orderID string) (*models.Order, error) {
	var order models.Order
	if err := db.Preload("Items.Product").Preload("Payment").Where("id = ?", orderID).First(&order).Error; err != nil || order.UserID != userID {
		return nil, apperror.New(404, "ORDER_NOT_FOUND", "Order not found")
	}
	return &order, nil
}
