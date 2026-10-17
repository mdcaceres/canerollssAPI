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

func (u *OrderUseCase) CreateOrder(order *domain.Order) error {
	reg, err := u.cashRepo.GetActive()
	if err != nil {
		return errors.New("no open register found")
	}
	order.CashRegisterID = reg.ID

	customer, err := u.customerRepo.FindByID(order.CustomerID)
	if err != nil || customer == nil {
		return errors.New("customer not found")
	}

	var totalAmount float64
	for i := range order.Items {
		topping, err := u.toppingRepo.GetByID(order.Items[i].ToppingID)
		if err != nil {
			return errors.New("topping not found or invalid")
		}

		price := topping.SinglePrice
		rolesCount := order.Items[i].Quantity
		if order.Items[i].IsCombo {
			price = topping.ComboPrice
			rolesCount = order.Items[i].Quantity * 2
		}

		order.Items[i].UnitPrice = price
		order.Items[i].Subtotal = price * float64(order.Items[i].Quantity)
		order.Items[i].RolesCount = rolesCount
		totalAmount += order.Items[i].Subtotal
	}

	order.TotalAmount = totalAmount + order.DeliveryCost
	if order.Status == "" {
		order.Status = domain.OrderStatusCompleted
	}

	customer.TotalOrders++
	if customer.FirstOrderAt == nil {
		now := time.Now()
		customer.FirstOrderAt = &now
	}
	if err := u.customerRepo.Update(customer); err != nil {
		return err
	}

	return u.orderRepo.Save(order)
}

func (u *OrderUseCase) GetOrderByID(orderID uint) (*domain.Order, error) {
	return u.orderRepo.GetByID(orderID)
}

func (u *OrderUseCase) UpdateOrderStatus(orderID uint, status domain.OrderStatus) error {
	order, err := u.orderRepo.GetByID(orderID)
	if err != nil {
		return err
	}
	order.Status = status
	return u.orderRepo.Update(order)
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

func (u *OrderUseCase) UpdateOrder(order *domain.Order) error {
	return u.orderRepo.Update(order)
}

func (u *OrderUseCase) GetPendingOrderByCustomer(registerID uint, customerID uint) (*domain.Order, error) {
	orders, err := u.orderRepo.GetByRegisterID(registerID)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		if orders[i].CustomerID == customerID && orders[i].Status == domain.OrderStatusPendingPayment {
			return &orders[i], nil
		}
	}
	return nil, nil
}

