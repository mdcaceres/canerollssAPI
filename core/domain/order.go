package domain

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPreparing OrderStatus = "PREPARING"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CashRegisterID uint           `gorm:"not null;index" json:"cash_register_id"`
	CustomerID     uint           `gorm:"not null;index" json:"customer_id"`
	Customer       Customer       `gorm:"foreignKey:CustomerID" json:"customer"`
	Status         OrderStatus    `gorm:"type:varchar(20);default:'PENDING';not null" json:"status"`
	TotalAmount    float64        `gorm:"not null" json:"total_amount"`
	Note           string         `gorm:"size:255" json:"note,omitempty"`
	Items          []OrderItem    `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type OrderItem struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	OrderID    uint    `gorm:"not null;index" json:"order_id"`
	ToppingID  uint    `gorm:"not null" json:"topping_id"`
	Topping    Topping `gorm:"foreignKey:ToppingID" json:"topping"`
	Quantity   int     `gorm:"not null" json:"quantity"`
	IsCombo    bool    `gorm:"not null" json:"is_combo"`
	RolesCount int     `gorm:"not null" json:"roles_count"`
	UnitPrice  float64 `gorm:"not null" json:"unit_price"`
	Subtotal   float64 `gorm:"not null" json:"subtotal"`
}
