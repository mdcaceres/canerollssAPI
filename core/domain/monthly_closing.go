package domain

import "time"

type MonthlyClosing struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Year         int       `gorm:"not null;uniqueIndex:idx_year_month" json:"year"`
	Month        int       `gorm:"not null;uniqueIndex:idx_year_month" json:"month"`
	TotalRevenue float64   `gorm:"not null" json:"total_revenue"`
	TotalRolls   int       `gorm:"not null" json:"total_rolls"`
	TotalCombos  int       `gorm:"not null" json:"total_combos"`
	TotalOrders  int       `gorm:"not null" json:"total_orders"`
	ClosedAt     time.Time `gorm:"not null" json:"closed_at"`

	ToppingMetrics []MonthlyToppingMetric `gorm:"foreignKey:MonthlyClosingID" json:"topping_metrics"`
}

type MonthlyToppingMetric struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	MonthlyClosingID uint    `gorm:"not null;index" json:"monthly_closing_id"`
	ToppingName      string  `gorm:"size:50;not null" json:"topping_name"`
	RolesCount       int     `gorm:"not null" json:"roles_count"`
	Revenue          float64 `gorm:"not null" json:"revenue"`
}
