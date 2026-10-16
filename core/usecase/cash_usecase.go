package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
	"time"
)

type CashUseCase struct {
	cashRepo  output.CashRegisterRepository
	orderRepo output.OrderRepository
}

func NewCashUseCase(cashRepo output.CashRegisterRepository, orderRepo output.OrderRepository) *CashUseCase {
	return &CashUseCase{cashRepo: cashRepo, orderRepo: orderRepo}
}

func (u *CashUseCase) OpenRegister() (*domain.CashRegister, error) {
	_, err := u.cashRepo.GetActive()
	if err == nil {
		return nil, errors.New("there is already an open cash register")
	}

	reg := &domain.CashRegister{
		Status:   domain.RegisterStatusOpen,
		OpenedAt: time.Now(),
	}

	if err := u.cashRepo.Save(reg); err != nil {
		return nil, err
	}

	return reg, nil
}

func (u *CashUseCase) GetActive() (*domain.CashRegister, error) {
	reg, err := u.cashRepo.GetActive()
	if err != nil {
		return nil, errors.New("there is no open cash register")
	}

	orders, err := u.orderRepo.GetCompletedByRegisterID(reg.ID)
	if err == nil {
		var totalSales float64
		var totalRolls int
		var totalCombos int

		for _, order := range orders {
			totalSales += order.TotalAmount
			for _, item := range order.Items {
				totalRolls += item.RolesCount
				if item.IsCombo {
					totalCombos += item.Quantity
				}
			}
		}

		reg.TotalSales = totalSales
		reg.TotalRolls = totalRolls
		reg.TotalCombos = totalCombos
		reg.Orders = orders
	}

	return reg, nil
}

func (u *CashUseCase) CloseRegister() (*domain.CashRegister, error) {
	reg, err := u.cashRepo.GetActive()
	if err != nil {
		return nil, errors.New("there is no open cash register")
	}

	orders, err := u.orderRepo.GetCompletedByRegisterID(reg.ID)
	if err != nil {
		return nil, err
	}

	var totalSales float64
	var totalRolls int
	var totalCombos int

	for _, order := range orders {
		totalSales += order.TotalAmount
		for _, item := range order.Items {
			totalRolls += item.RolesCount
			if item.IsCombo {
				totalCombos += item.Quantity
			}
		}
	}

	reg.TotalSales = totalSales
	reg.TotalRolls = totalRolls
	reg.TotalCombos = totalCombos
	reg.Orders = orders
	reg.Status = domain.RegisterStatusClosed

	now := time.Now()
	reg.ClosedAt = &now

	if err := u.cashRepo.Update(reg); err != nil {
		return nil, err
	}

	return reg, nil
}

func (u *CashUseCase) GetCashHistory(page, pageSize int) ([]domain.CashRegister, int64, error) {
	return u.cashRepo.GetHistory(page, pageSize)
}
