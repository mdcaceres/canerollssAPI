package output

import (
	"canerollss/core/domain"
)

type CashRegisterRepository interface {
	GetActive() (*domain.CashRegister, error)
	Save(register *domain.CashRegister) error
	Update(register *domain.CashRegister) error
	GetHistory(page, pageSize int) ([]domain.CashRegister, int64, error)
}
