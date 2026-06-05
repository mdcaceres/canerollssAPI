package output

import (
	"canerollss/core/domain"
	"time"
)

type OrderRepository interface {
	Save(order *domain.Order) error
	GetByID(id uint) (*domain.Order, error)
	GetByRegisterID(registerID uint) ([]domain.Order, error)
	Update(order *domain.Order) error
	GetCompletedByRegisterID(regID uint) ([]domain.Order, error)
	GetCompletedByDateRange(start, end time.Time) ([]domain.Order, error)
}
