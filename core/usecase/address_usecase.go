package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"errors"
	"log"
	"strings"
)

type AddressUseCase struct {
	addressRepo output.CustomerAddressRepository
	geocoding   output.GeocodingProvider
}

func NewAddressUseCase(repo output.CustomerAddressRepository, geo ...output.GeocodingProvider) *AddressUseCase {
	u := &AddressUseCase{addressRepo: repo}
	if len(geo) > 0 {
		u.geocoding = geo[0]
	}
	return u
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
	if u.geocoding == nil {
		return nil, errors.New("address validation provider is not configured")
	}
	geocoded, err := u.geocoding.ValidateAndGeocode(context.Background(), streetAndNumber)
	if err != nil {
		log.Printf("component=address action=validate status=failed error=%q", err)
		return nil, err
	}
	if geocoded == nil || !geocoded.Exact || geocoded.Lat == 0 || geocoded.Lng == 0 {
		return nil, errors.New("No pudimos validar la dirección exacta.")
	}

	trimmedAlias := strings.TrimSpace(alias)
	if trimmedAlias == "" {
		trimmedAlias = "Casa"
	}

	existing, err := u.addressRepo.FindByPhone(customerPhone)
	if err != nil {
		return nil, err
	}
	isDefault := len(existing) == 0

	address := &domain.CustomerAddress{
		CustomerPhone:    customerPhone,
		Alias:            trimmedAlias,
		StreetAndNumber:  geocoded.StreetAndNumber,
		RawInput:         strings.TrimSpace(streetAndNumber),
		FormattedAddress: geocoded.FormattedAddress,
		Neighborhood:     geocoded.Neighborhood,
		ZipCode:          geocoded.ZipCode,
		ApartmentNotes:   strings.TrimSpace(apartmentNotes),
		City:             "Córdoba",
		Lat:              geocoded.Lat,
		Lng:              geocoded.Lng,
		IsDefault:        isDefault,
	}

	if err := u.addressRepo.Save(address); err != nil {
		log.Printf("component=address action=save status=failed error=%q", err)
		return nil, err
	}

	return address, nil
}

func (u *AddressUseCase) GetAddressByID(id uint) (*domain.CustomerAddress, error) {
	return u.addressRepo.FindByID(id)
}
