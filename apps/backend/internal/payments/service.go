package payments

import (
	"strings"

	"backend/internal/apperror"
	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PayForOrder จำลองการจ่ายเงินโดยไม่ผูก payment gateway จริงใดๆ (ดู README
// หัวข้อ non-goals) — กติกาง่ายๆ ที่ตั้งใจให้จำง่ายตอนเขียนเทส: เลขบัตรขึ้นต้น
// ด้วย "4242" = จ่ายสำเร็จ, เลขอื่นๆ = ถูกปฏิเสธเสมอ
//
// สังเกตว่า order ต้องมีสถานะ PENDING เท่านั้นถึงจะจ่ายได้ — กันไม่ให้จ่ายซ้ำ
// order ที่ PAID ไปแล้ว หรือจ่าย order ที่ถูกยกเลิกไปแล้ว
func PayForOrder(db *gorm.DB, userID, orderID, cardNumber string) (*models.Payment, error) {
	var order models.Order
	if err := db.Where("id = ?", orderID).First(&order).Error; err != nil || order.UserID != userID {
		return nil, apperror.New(404, "ORDER_NOT_FOUND", "Order not found")
	}
	if order.Status != models.OrderStatusPending {
		return nil, apperror.New(409, "ORDER_NOT_PAYABLE", "Order is in \""+string(order.Status)+"\" state and cannot be paid")
	}

	success := strings.HasPrefix(cardNumber, "4242")
	mockTransactionID := uuid.NewString()
	status := models.PaymentStatusFailed
	if success {
		status = models.PaymentStatusSuccess
	}

	// ใช้ upsert แบบ manual (หา record เดิมก่อน มีก็ update ไม่มีก็ create)
	// เพื่อรองรับกรณีลูกค้าจ่ายบัตรที่ถูกปฏิเสธ แล้วกลับมาลองจ่ายบัตรใบใหม่กับ
	// order เดิมอีกครั้ง — ไม่ให้เกิด Payment ซ้ำหลายแถวต่อ 1 order
	var payment models.Payment
	err := db.Where("order_id = ?", orderID).First(&payment).Error
	if err == nil {
		payment.Status = status
		payment.MockTransactionID = mockTransactionID
		if err := db.Save(&payment).Error; err != nil {
			return nil, err
		}
	} else {
		payment = models.Payment{OrderID: orderID, Method: "mock-card", Status: status, MockTransactionID: mockTransactionID}
		if err := db.Create(&payment).Error; err != nil {
			return nil, err
		}
	}

	if !success {
		return nil, apperror.New(402, "PAYMENT_DECLINED", "Payment was declined")
	}

	if err := db.Model(&models.Order{}).Where("id = ?", orderID).Update("status", models.OrderStatusPaid).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}
