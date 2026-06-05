package output

import (
	"canerollss/core/domain"
)

type ToppingRepository interface {
	GetAll() ([]domain.Topping, error)
	Save(topping *domain.Topping) error
	GetByID(id uint) (*domain.Topping, error)
	Update(topping *domain.Topping) error
}
