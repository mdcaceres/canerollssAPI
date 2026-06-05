package repository

import (
	"canerollss/core/domain"
	"time"

	"gorm.io/gorm"
)

type orderRepository struct{ db *gorm.DB }

func NewOrderRepository(db *gorm.DB) *orderRepository { return &orderRepository{db: db} }

func (r *orderRepository) Save(order *domain.Order) error {
	if err := r.db.Create(order).Error; err != nil {
		return err
	}

	return r.db.Preload("Customer").Preload("Items.Topping").First(order, order.ID).Error
}

func (r *orderRepository) GetByID(id uint) (*domain.Order, error) {
	var order domain.Order
	err := r.db.Preload("Items.Topping").First(&order, id).Error
	return &order, err
}

func (r *orderRepository) GetByRegisterID(regID uint) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Where("cash_register_id = ?", regID).Find(&orders).Error
	return orders, err
}

func (r *orderRepository) Update(order *domain.Order) error {
	return r.db.Save(order).Error
}

func (r *orderRepository) GetCompletedByRegisterID(regID uint) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Preload("Items").
		Where("cash_register_id = ? AND status = ?", regID, domain.OrderStatusCompleted).
		Find(&orders).Error

	return orders, err
}

func (r *orderRepository) GetCompletedByDateRange(start, end time.Time) ([]domain.Order, error) {
	var orders []domain.Order
	err := r.db.Preload("Items.Topping").
		Where("status = ? AND created_at >= ? AND created_at < ?", domain.OrderStatusCompleted, start, end).
		Find(&orders).Error
	return orders, err
}
