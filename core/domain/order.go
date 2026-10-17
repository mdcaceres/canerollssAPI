package domain

import (
	"time"

	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	OrderStatusPaid           OrderStatus = "PAID"
	OrderStatusPreparing      OrderStatus = "PREPARING"
	OrderStatusDispatched     OrderStatus = "DISPATCHED"
	OrderStatusDelivered      OrderStatus = "DELIVERED"
	OrderStatusCompleted      OrderStatus = "COMPLETED"
	OrderStatusCancelled      OrderStatus = "CANCELLED"
)

type DeliveryType string

const (
	DeliveryTypePickup   DeliveryType = "PICKUP"
	DeliveryTypeDelivery DeliveryType = "DELIVERY"
)

type Order struct {
	ID                     uint             `gorm:"primaryKey" json:"id"`
	CashRegisterID         uint             `gorm:"not null;index" json:"cash_register_id"`
	CustomerID             uint             `gorm:"not null;index" json:"customer_id"`
	Customer               Customer         `gorm:"foreignKey:CustomerID" json:"customer"`
	Status                 OrderStatus      `gorm:"type:varchar(30);default:'PENDING_PAYMENT';not null" json:"status"`
	DeliveryType           DeliveryType     `gorm:"type:varchar(20);default:'PICKUP';not null" json:"delivery_type"`
	AddressID              *uint            `gorm:"index" json:"address_id,omitempty"`
	Address                *CustomerAddress `gorm:"foreignKey:AddressID" json:"address,omitempty"`
	DeliveryCost           float64          `gorm:"default:0" json:"delivery_cost"`
	DeliveryProviderFee    float64          `gorm:"default:0" json:"delivery_provider_fee"`
	DeliveryQuoteID        string           `gorm:"size:100" json:"delivery_quote_id,omitempty"`
	DeliveryQuotedAt       *time.Time       `json:"delivery_quoted_at,omitempty"`
	DeliveryQuoteExpiresAt *time.Time       `json:"delivery_quote_expires_at,omitempty"`
	TotalAmount            float64          `gorm:"not null" json:"total_amount"`
	PaymentMethod          string           `gorm:"type:varchar(30);default:'TRANSFER_ALIAS'" json:"payment_method"`
	PaymentLink            string           `gorm:"size:500" json:"payment_link,omitempty"`
	ExternalPaymentID      string           `gorm:"size:100;index" json:"external_payment_id,omitempty"`
	ExternalDeliveryID     string           `gorm:"size:100" json:"external_delivery_id,omitempty"`
	TrackingURL            string           `gorm:"size:500" json:"tracking_url,omitempty"`
	Note                   string           `gorm:"size:255" json:"note,omitempty"`
	PaidAt                 *time.Time       `json:"paid_at,omitempty"`
	Items                  []OrderItem      `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt              time.Time        `json:"created_at"`
	UpdatedAt              time.Time        `json:"updated_at"`
	DeletedAt              gorm.DeletedAt   `gorm:"index" json:"-"`
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
