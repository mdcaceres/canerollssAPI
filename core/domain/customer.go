package domain

import (
	"time"

	"gorm.io/gorm"
)

type Customer struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	Whatsapp     string         `gorm:"size:20;not null;uniqueIndex" json:"whatsapp"`
	TotalOrders  int            `gorm:"default:0" json:"total_orders"`
	FirstOrderAt *time.Time     `json:"first_order_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}
