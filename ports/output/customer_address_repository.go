package output

import "canerollss/core/domain"

type CustomerAddressRepository interface {
	Save(address *domain.CustomerAddress) error
	FindByPhone(phone string) ([]domain.CustomerAddress, error)
	FindByID(id uint) (*domain.CustomerAddress, error)
	SetDefault(id uint, phone string) error
	Delete(id uint, phone string) error
}
