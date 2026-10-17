package repository

import (
	"canerollss/core/domain"
	"errors"

	"gorm.io/gorm"
)

type customerAddressRepository struct {
	db *gorm.DB
}

func NewCustomerAddressRepository(db *gorm.DB) *customerAddressRepository {
	return &customerAddressRepository{db: db}
}

func (r *customerAddressRepository) Save(address *domain.CustomerAddress) error {
	if address.IsDefault {
		return r.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&domain.CustomerAddress{}).Where("customer_phone = ?", address.CustomerPhone).Update("is_default", false).Error; err != nil {
				return err
			}
			return tx.Save(address).Error
		})
	}
	return r.db.Save(address).Error
}

func (r *customerAddressRepository) FindByPhone(phone string) ([]domain.CustomerAddress, error) {
	var addresses []domain.CustomerAddress
	err := r.db.Where("customer_phone = ?", phone).
		Order("is_default desc, id desc").
		Find(&addresses).Error
	return addresses, err
}

func (r *customerAddressRepository) FindByID(id uint) (*domain.CustomerAddress, error) {
	var address domain.CustomerAddress
	err := r.db.First(&address, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("address not found")
	}
	return &address, err
}

func (r *customerAddressRepository) SetDefault(id uint, phone string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.CustomerAddress{}).
			Where("customer_phone = ?", phone).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&domain.CustomerAddress{}).
			Where("id = ? AND customer_phone = ?", id, phone).
			Update("is_default", true).Error
	})
}

func (r *customerAddressRepository) Delete(id uint, phone string) error {
	return r.db.Where("id = ? AND customer_phone = ?", id, phone).
		Delete(&domain.CustomerAddress{}).Error
}
