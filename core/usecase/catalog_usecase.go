package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
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
