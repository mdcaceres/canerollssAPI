package domain

import (
	"time"

	"gorm.io/gorm"
)

type Topping struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:50;not null;unique" json:"name"`
	SinglePrice float64        `gorm:"not null" json:"single_price"`
	ComboPrice  float64        `gorm:"not null" json:"combo_price"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
