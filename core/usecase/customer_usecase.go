package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
)

type CustomerUseCase struct {
	customerRepo output.CustomerRepository
}

func NewCustomerUseCase(repo output.CustomerRepository) *CustomerUseCase {
	return &CustomerUseCase{customerRepo: repo}
}

func (u *CustomerUseCase) LookupByWhatsapp(whatsapp string) (*domain.Customer, error) {
	if whatsapp == "" {
		return nil, errors.New("whatsapp is required")
	}
	return u.customerRepo.FindByWhatsapp(whatsapp)
}

func (u *CustomerUseCase) Create(name, whatsapp string) (*domain.Customer, error) {
	if name == "" || whatsapp == "" {
		return nil, errors.New("name and whatsapp are required")
	}

	existing, err := u.customerRepo.FindByWhatsapp(whatsapp)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("whatsapp already registered")
	}

	customer := &domain.Customer{
		Name:     name,
		Whatsapp: whatsapp,
	}
	if err := u.customerRepo.Save(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (u *CustomerUseCase) GetByID(id uint) (*domain.Customer, error) {
	return u.customerRepo.FindByID(id)
}
