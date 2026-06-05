package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type ReportUseCase struct {
	repo      output.MonthlyClosingRepository
	orderRepo output.OrderRepository
}

func NewReportUseCase(repo output.MonthlyClosingRepository, orderRepo output.OrderRepository) *ReportUseCase {
	return &ReportUseCase{repo: repo, orderRepo: orderRepo}
}

func (u *ReportUseCase) GetReportHistory(page, pageSize int) ([]domain.MonthlyClosing, int64, error) {
	return u.repo.GetHistory(page, pageSize)
}

func (u *ReportUseCase) GetMonthlyClosing(month, year int) (*domain.MonthlyClosing, error) {
	if !isValidMonthYear(month, year) {
		return nil, errors.New("invalid month or year")
	}
	return u.repo.GetByMonthAndYear(month, year)
}

func (u *ReportUseCase) CloseMonth(year, month int) (*domain.MonthlyClosing, error) {
	if !isValidMonthYear(month, year) {
		return nil, errors.New("invalid month or year")
	}

	existing, err := u.repo.GetByMonthAndYear(month, year)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)

	orders, err := u.orderRepo.GetCompletedByDateRange(start, end)
	if err != nil {
		return nil, err
	}

	var totalRevenue float64
	var totalRolls int
	var totalCombos int

	metricsByTopping := make(map[string]*domain.MonthlyToppingMetric)

	for _, order := range orders {
		totalRevenue += order.TotalAmount
		for _, item := range order.Items {
			totalRolls += item.RolesCount
			if item.IsCombo {
				totalCombos += item.Quantity
			}

			name := item.Topping.Name
			metric, ok := metricsByTopping[name]
			if !ok {
				metric = &domain.MonthlyToppingMetric{ToppingName: name}
				metricsByTopping[name] = metric
			}
			metric.RolesCount += item.RolesCount
			metric.Revenue += item.Subtotal
		}
	}

	closing := &domain.MonthlyClosing{
		Year:         year,
		Month:        month,
		TotalRevenue: totalRevenue,
		TotalRolls:   totalRolls,
		TotalCombos:  totalCombos,
		TotalOrders:  len(orders),
		ClosedAt:     time.Now(),
	}

	for _, metric := range metricsByTopping {
		closing.ToppingMetrics = append(closing.ToppingMetrics, *metric)
	}

	if err := u.repo.SaveWithMetrics(closing); err != nil {
		if isDuplicateKeyError(err) {
			existing, findErr := u.repo.GetByMonthAndYear(month, year)
			if findErr == nil {
				return existing, nil
			}
			return nil, findErr
		}
		return nil, err
	}

	return closing, nil
}

func isValidMonthYear(month, year int) bool {
	return month >= 1 && month <= 12 && year > 0
}

func isDuplicateKeyError(err error) bool {
	var myErr *mysql.MySQLError
	if errors.As(err, &myErr) {
		return myErr.Number == 1062
	}
	return false
}
