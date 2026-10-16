package repository

import (
	"canerollss/core/domain"
	"errors"

	"gorm.io/gorm"
)

type customerRepository struct{ db *gorm.DB }

func NewCustomerRepository(db *gorm.DB) *customerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) FindByWhatsapp(whatsapp string) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.Where("whatsapp = ?", whatsapp).First(&customer).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &customer, err
}

func (r *customerRepository) FindByID(id uint) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.First(&customer, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("customer not found")
	}
	return &customer, err
}

func (r *customerRepository) Save(customer *domain.Customer) error {
	return r.db.Create(customer).Error
}

func (r *customerRepository) Update(customer *domain.Customer) error {
	return r.db.Save(customer).Error
}
