package domain

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleAdmin    UserRole = "ADMIN"
	RoleEmployee UserRole = "EMPLOYEE"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:50;not null;uniqueIndex" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         UserRole       `gorm:"type:varchar(20);default:'EMPLOYEE';not null" json:"role"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type LoginDTO struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}
