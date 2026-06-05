package repository

import (
	"canerollss/core/domain"

	"gorm.io/gorm"
)

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByUsername(username string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *userRepository) Save(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) Exists(username string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *userRepository) Update(user *domain.User) error {
	return r.db.Save(user).Error
}
