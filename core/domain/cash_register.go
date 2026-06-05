package domain

import (
	"time"

	"gorm.io/gorm"
)

type RegisterStatus string

const (
	RegisterStatusOpen   RegisterStatus = "OPEN"
	RegisterStatusClosed RegisterStatus = "CLOSED"
)

type CashRegister struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Status      RegisterStatus `gorm:"type:varchar(20);default:'OPEN';not null" json:"status"`
	OpenedAt    time.Time      `gorm:"not null" json:"opened_at"`
	ClosedAt    *time.Time     `json:"closed_at,omitempty"`
	TotalSales  float64        `gorm:"default:0" json:"total_sales"`
	TotalRolls  int            `gorm:"default:0" json:"total_rolls"`
	TotalCombos int            `gorm:"default:0" json:"total_combos"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
