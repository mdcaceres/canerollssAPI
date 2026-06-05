package repository

import (
	"canerollss/core/domain"

	"gorm.io/gorm"
)

type cashRegisterRepository struct{ db *gorm.DB }

func NewCashRegisterRepository(db *gorm.DB) *cashRegisterRepository {
	return &cashRegisterRepository{db: db}
}

func (r *cashRegisterRepository) GetActive() (*domain.CashRegister, error) {
	var reg domain.CashRegister
	err := r.db.Where("status = ?", domain.RegisterStatusOpen).First(&reg).Error
	return &reg, err
}

func (r *cashRegisterRepository) Save(reg *domain.CashRegister) error {
	return r.db.Create(reg).Error
}

func (r *cashRegisterRepository) Update(reg *domain.CashRegister) error {
	return r.db.Save(reg).Error
}

func (r *cashRegisterRepository) GetHistory(page, pageSize int) ([]domain.CashRegister, int64, error) {
	var regs []domain.CashRegister
	var total int64
	r.db.Model(&domain.CashRegister{}).Count(&total)

	err := r.db.Order("opened_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&regs).Error
	return regs, total, err
}
