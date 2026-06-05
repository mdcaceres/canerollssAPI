package output

import (
	"canerollss/core/domain"
)

type MonthlyClosingRepository interface {
	Save(closing *domain.MonthlyClosing) error
	SaveWithMetrics(closing *domain.MonthlyClosing) error
	GetByMonthAndYear(month, year int) (*domain.MonthlyClosing, error)
	GetHistory(page, pageSize int) ([]domain.MonthlyClosing, int64, error)
}
