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
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) Save(customer *domain.Customer) error {
	return r.db.Create(customer).Error
}

func (r *customerRepository) Update(customer *domain.Customer) error {
	return r.db.Save(customer).Error
}
