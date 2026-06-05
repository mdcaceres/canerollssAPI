package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"errors"
	"time"
)

type OrderUseCase struct {
	orderRepo    output.OrderRepository
	cashRepo     output.CashRegisterRepository
	customerRepo output.CustomerRepository
	toppingRepo  output.ToppingRepository
}

func NewOrderUseCase(o output.OrderRepository, c output.CashRegisterRepository, cr output.CustomerRepository, tr output.ToppingRepository) *OrderUseCase {
	return &OrderUseCase{orderRepo: o, cashRepo: c, customerRepo: cr, toppingRepo: tr}
}

func (u *OrderUseCase) CreateOrder(order *domain.Order, customerName, customerWhatsapp string) error {
	reg, err := u.cashRepo.GetActive()
	if err != nil {
		return errors.New("no open register found")
	}
	order.CashRegisterID = reg.ID

	customer, err := u.customerRepo.FindByWhatsapp(customerWhatsapp)
	if err != nil {
		return err
	}

	if customer == nil {
		customer = &domain.Customer{
			Name:     customerName,
			Whatsapp: customerWhatsapp,
		}
		if err := u.customerRepo.Save(customer); err != nil {
			return err
		}
	}

	customer.TotalOrders++
	if customer.FirstOrderAt == nil {
		now := time.Now()
		customer.FirstOrderAt = &now
	}

	if err := u.customerRepo.Update(customer); err != nil {
		return err
	}

	order.CustomerID = customer.ID
	order.Status = domain.OrderStatusCompleted

	var totalAmount float64

	for i := range order.Items {
		topping, err := u.toppingRepo.GetByID(order.Items[i].ToppingID)
		if err != nil {
			return errors.New("topping not found or invalid")
		}

		price := topping.SinglePrice
		if order.Items[i].IsCombo {
			price = topping.ComboPrice
		}

		order.Items[i].UnitPrice = price
		order.Items[i].Subtotal = price * float64(order.Items[i].Quantity)
		order.Items[i].RolesCount = order.Items[i].Quantity

		totalAmount += order.Items[i].Subtotal
	}

	order.TotalAmount = totalAmount

	return u.orderRepo.Save(order)
}

func (u *OrderUseCase) CancelOrder(orderID uint) error {
	order, err := u.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusCancelled {
		return errors.New("order is already cancelled")
	}

	order.Status = domain.OrderStatusCancelled
	return u.orderRepo.Update(order)
}
