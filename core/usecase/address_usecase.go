package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
	"strings"
)

type AddressUseCase struct {
	addressRepo output.CustomerAddressRepository
}

func NewAddressUseCase(repo output.CustomerAddressRepository) *AddressUseCase {
	return &AddressUseCase{addressRepo: repo}
}

func (u *AddressUseCase) GetCustomerAddresses(customerPhone string) ([]domain.CustomerAddress, error) {
	if strings.TrimSpace(customerPhone) == "" {
		return nil, errors.New("customer phone is required")
	}
	return u.addressRepo.FindByPhone(customerPhone)
}

func (u *AddressUseCase) SaveAddress(customerPhone, alias, streetAndNumber, apartmentNotes string) (*domain.CustomerAddress, error) {
	if strings.TrimSpace(customerPhone) == "" {
		return nil, errors.New("customer phone is required")
	}
	if strings.TrimSpace(streetAndNumber) == "" {
		return nil, errors.New("street and number is required")
	}

	trimmedAlias := strings.TrimSpace(alias)
	if trimmedAlias == "" {
		trimmedAlias = "Casa"
	}

	existing, _ := u.addressRepo.FindByPhone(customerPhone)
	isDefault := len(existing) == 0

	address := &domain.CustomerAddress{
		CustomerPhone:   customerPhone,
		Alias:           trimmedAlias,
		StreetAndNumber: strings.TrimSpace(streetAndNumber),
		ApartmentNotes:  strings.TrimSpace(apartmentNotes),
		City:            "CABA",
		IsDefault:       isDefault,
	}

	if err := u.addressRepo.Save(address); err != nil {
		return nil, err
	}

	return address, nil
}

func (u *AddressUseCase) GetAddressByID(id uint) (*domain.CustomerAddress, error) {
	return u.addressRepo.FindByID(id)
}
