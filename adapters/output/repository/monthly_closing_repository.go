package repository

import (
	"canerollss/core/domain"

	"gorm.io/gorm"
)

type monthlyClosingRepository struct{ db *gorm.DB }

func NewMonthlyClosingRepository(db *gorm.DB) *monthlyClosingRepository {
	return &monthlyClosingRepository{db: db}
}

func (r *monthlyClosingRepository) Save(m *domain.MonthlyClosing) error { return r.db.Create(m).Error }

func (r *monthlyClosingRepository) SaveWithMetrics(m *domain.MonthlyClosing) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("ToppingMetrics").Create(m).Error; err != nil {
			return err
		}
		if len(m.ToppingMetrics) == 0 {
			return nil
		}
		for i := range m.ToppingMetrics {
			m.ToppingMetrics[i].MonthlyClosingID = m.ID
		}
		return tx.Create(&m.ToppingMetrics).Error
	})
}

func (r *monthlyClosingRepository) GetByMonthAndYear(month, year int) (*domain.MonthlyClosing, error) {
	var m domain.MonthlyClosing
	err := r.db.Preload("ToppingMetrics").Where("month = ? AND year = ?", month, year).First(&m).Error
	return &m, err
}

func (r *monthlyClosingRepository) GetHistory(page, pageSize int) ([]domain.MonthlyClosing, int64, error) {
	var history []domain.MonthlyClosing
	var total int64

	r.db.Model(&domain.MonthlyClosing{}).Count(&total)

	err := r.db.Preload("ToppingMetrics").Order("year DESC, month DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&history).Error

	return history, total, err
}
