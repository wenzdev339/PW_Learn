// Package models เก็บ struct ทุกตัวที่แทนตารางในฐานข้อมูล (GORM จะอ่าน struct
// เหล่านี้แล้วสร้างตารางให้อัตโนมัติผ่าน AutoMigrate — ดู internal/db/db.go)
//
// ทุก struct ใช้ ID เป็น string (UUID) แทนเลขรัน 1,2,3,... เพราะ:
//  1. เดา ID ของคนอื่นไม่ได้ (ปลอดภัยกว่าเวลา expose ผ่าน URL/API)
//  2. สร้าง ID ได้จากฝั่งแอปเลยโดยไม่ต้องรอฐานข้อมูล (ผ่าน BeforeCreate hook
//     ด้านล่าง ซึ่ง GORM จะเรียกอัตโนมัติทุกครั้งก่อน insert)
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Role string

const (
	RoleCustomer Role = "CUSTOMER"
	RoleAdmin    Role = "ADMIN"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type PaymentStatus string

const (
	PaymentStatusSuccess PaymentStatus = "SUCCESS"
	PaymentStatusFailed  PaymentStatus = "FAILED"
)

// All models tag `json:` explicitly in camelCase to match the REST API
// convention the Frontend expects. User.PasswordHash is `json:"-"` — it
// must never be serialized, even when embedded via an association.

type User struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Name         string    `gorm:"not null" json:"name"`
	Role         Role      `gorm:"type:varchar(20);not null;default:'CUSTOMER'" json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
}

// BeforeCreate: GORM เรียกฟังก์ชันนี้อัตโนมัติก่อน INSERT ทุกครั้ง — ถ้ายังไม่มี
// ID (กรณีสร้าง record ใหม่) จะสุ่ม UUID ให้เอง ทุก model ในไฟล์นี้มี pattern
// เดียวกันนี้ซ้ำ (ไม่อธิบายซ้ำในทุกจุด)
func (m *User) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Address struct {
	ID         string `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     string `gorm:"type:uuid;not null;index" json:"userId"`
	Label      string `gorm:"not null" json:"label"`
	Line1      string `gorm:"not null" json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `gorm:"not null" json:"city"`
	PostalCode string `gorm:"not null" json:"postalCode"`
	Country    string `gorm:"not null" json:"country"`
	IsDefault  bool   `gorm:"not null;default:false" json:"isDefault"`
}

func (m *Address) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Category struct {
	ID   string `gorm:"type:uuid;primaryKey" json:"id"`
	Name string `gorm:"not null" json:"name"`
	Slug string `gorm:"uniqueIndex;not null" json:"slug"`
}

func (m *Category) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Product struct {
	ID          string `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string `gorm:"not null" json:"name"`
	Slug        string `gorm:"uniqueIndex;not null" json:"slug"`
	Description string `gorm:"not null" json:"description"`
	// Price เก็บเป็นหน่วยสตางค์ (int) ไม่ใช่บาท (float) — เลี่ยงปัญหาปัดเศษ
	// ทศนิยมของ float ที่มักคลาดเคลื่อนตอนคำนวณเงิน
	Price int `gorm:"not null" json:"price"`
	Stock int `gorm:"not null" json:"stock"`
	// pq.StringArray คือ type พิเศษของ driver postgres ที่แปลง []string
	// ให้เก็บเป็นคอลัมน์ text[] ของ Postgres ได้ตรงๆ (ปกติ []string ธรรมดา
	// ใช้กับคอลัมน์ Postgres array ไม่ได้เพราะ database/sql ไม่รู้จักวิธีแปลง)
	Images     pq.StringArray `gorm:"type:text[]" json:"images"`
	CategoryID string         `gorm:"type:uuid;not null;index" json:"categoryId"`
	Category   Category       `gorm:"foreignKey:CategoryID" json:"category"`
	Reviews    []Review       `gorm:"foreignKey:ProductID" json:"reviews,omitempty"`
	CreatedAt  time.Time      `json:"createdAt"`
}

func (m *Product) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Review struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	ProductID string    `gorm:"type:uuid;not null;index" json:"productId"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Rating    int       `gorm:"not null" json:"rating"`
	Comment   string    `gorm:"not null" json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
}

func (m *Review) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Cart struct {
	ID     string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID string     `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	Items  []CartItem `gorm:"foreignKey:CartID" json:"items"`
}

func (m *Cart) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type CartItem struct {
	ID        string  `gorm:"type:uuid;primaryKey" json:"id"`
	CartID    string  `gorm:"type:uuid;not null;index" json:"cartId"`
	ProductID string  `gorm:"type:uuid;not null;index" json:"productId"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `gorm:"not null" json:"quantity"`
}

func (m *CartItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Order struct {
	ID                string      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID            string      `gorm:"type:uuid;not null;index" json:"userId"`
	Status            OrderStatus `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	TotalAmount       int         `gorm:"not null" json:"totalAmount"`
	ShippingAddressID string      `gorm:"type:uuid;not null" json:"shippingAddressId"`
	CreatedAt         time.Time   `json:"createdAt"`
	Items             []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	Payment           *Payment    `gorm:"foreignKey:OrderID" json:"payment,omitempty"`
}

func (m *Order) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type OrderItem struct {
	ID        string  `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID   string  `gorm:"type:uuid;not null;index" json:"orderId"`
	ProductID string  `gorm:"type:uuid;not null;index" json:"productId"`
	Product   Product `gorm:"foreignKey:ProductID" json:"product"`
	Quantity  int     `gorm:"not null" json:"quantity"`
	// PriceAtPurchase คือราคา ณ วินาทีที่กด checkout — เก็บแยกจาก
	// Product.Price เพราะถ้าร้านค้าขึ้น/ลดราคาสินค้าทีหลัง ออเดอร์เก่าต้อง
	// ยังโชว์ราคาที่ลูกค้าจ่ายจริงตอนนั้น ไม่ใช่ราคาปัจจุบัน
	PriceAtPurchase int `gorm:"not null" json:"priceAtPurchase"`
}

func (m *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Payment struct {
	ID                string        `gorm:"type:uuid;primaryKey" json:"id"`
	OrderID           string        `gorm:"type:uuid;uniqueIndex;not null" json:"orderId"`
	Method            string        `gorm:"not null" json:"method"`
	Status            PaymentStatus `gorm:"type:varchar(20);not null" json:"status"`
	MockTransactionID string        `gorm:"not null" json:"mockTransactionId"`
	CreatedAt         time.Time     `json:"createdAt"`
}

func (m *Payment) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
