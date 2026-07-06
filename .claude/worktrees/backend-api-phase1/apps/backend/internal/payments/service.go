package payments

import (
	"strings"

	"backend/internal/apperror"
	"backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
