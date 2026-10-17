package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
)

type CatalogUseCase struct {
	repo output.ToppingRepository
}

func NewCatalogUseCase(repo output.ToppingRepository) *CatalogUseCase {
	return &CatalogUseCase{repo: repo}
}

func (u *CatalogUseCase) CreateTopping(topping *domain.Topping) error {
	return u.repo.Save(topping)
}

func (u *CatalogUseCase) ListToppings() ([]domain.Topping, error) {
	return u.repo.GetAll()
}

func (u *CatalogUseCase) UpdateToppingPrices(id uint, singlePrice, comboPrice *float64) (*domain.Topping, error) {
	topping, err := u.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if topping == nil {
		return nil, errors.New("topping not found")
	}

	if singlePrice != nil {
		topping.SinglePrice = *singlePrice
	}
	if comboPrice != nil {
		topping.ComboPrice = *comboPrice
	}

	if err := u.repo.Update(topping); err != nil {
		return nil, err
	}

	return topping, nil
}

