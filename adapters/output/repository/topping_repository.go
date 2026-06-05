package repository

import (
	"canerollss/core/domain"

	"gorm.io/gorm"
)

type toppingRepository struct{ db *gorm.DB }

func NewToppingRepository(db *gorm.DB) *toppingRepository {
	return &toppingRepository{db: db}
}

func (r *toppingRepository) GetAll() ([]domain.Topping, error) {
	var toppings []domain.Topping
	err := r.db.Find(&toppings).Error
	return toppings, err
}

func (r *toppingRepository) Save(topping *domain.Topping) error {
	return r.db.Create(topping).Error
}

func (r *toppingRepository) GetByID(id uint) (*domain.Topping, error) {
	var topping domain.Topping
	err := r.db.First(&topping, id).Error
	return &topping, err
}

func (r *toppingRepository) Update(topping *domain.Topping) error {
	return r.db.Save(topping).Error
}
