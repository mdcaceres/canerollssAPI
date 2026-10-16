package output

import "canerollss/core/domain"

type CustomerRepository interface {
	FindByWhatsapp(whatsapp string) (*domain.Customer, error)
	FindByID(id uint) (*domain.Customer, error)
	Save(customer *domain.Customer) error
	Update(customer *domain.Customer) error
}
