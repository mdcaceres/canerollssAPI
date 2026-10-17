package domain

import (
	"time"

	"gorm.io/gorm"
)

type CustomerAddress struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	CustomerPhone    string         `gorm:"size:20;not null;index" json:"customer_phone"`
	Alias            string         `gorm:"size:50;not null;default:'Casa'" json:"alias"`
	StreetAndNumber  string         `gorm:"size:255;not null" json:"street_and_number"`
	RawInput         string         `gorm:"size:255" json:"raw_input"`
	FormattedAddress string         `gorm:"size:500" json:"formatted_address"`
	Neighborhood     string         `gorm:"size:100" json:"neighborhood"`
	ZipCode          string         `gorm:"size:20" json:"zip_code"`
	ApartmentNotes   string         `gorm:"size:255" json:"apartment_notes,omitempty"`
	City             string         `gorm:"size:100;default:'Córdoba'" json:"city"`
	Lat              float64        `gorm:"default:0.0" json:"lat"`
	Lng              float64        `gorm:"default:0.0" json:"lng"`
	IsDefault        bool           `gorm:"default:false" json:"is_default"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}
