package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type DeliveryUseCase struct {
	orders    output.OrderRepository
	addresses output.CustomerAddressRepository
	customers output.CustomerRepository
	registers output.CashRegisterRepository
	provider  output.DeliveryProvider
	notifier  output.MessageSender
}

func NewDeliveryUseCase(orders output.OrderRepository, addresses output.CustomerAddressRepository, customers output.CustomerRepository, registers output.CashRegisterRepository, provider output.DeliveryProvider, notifier output.MessageSender) *DeliveryUseCase {
	return &DeliveryUseCase{orders: orders, addresses: addresses, customers: customers, registers: registers, provider: provider, notifier: notifier}
}

func validDestination(address *domain.CustomerAddress) bool {
	return address != nil && address.City == "Córdoba" && address.Lat != 0 && address.Lng != 0 && address.StreetAndNumber != ""
}
func radians(v float64) float64 { return v * math.Pi / 180 }
func distanceKM(aLat, aLng, bLat, bLng float64) float64 {
	dLat, dLng := radians(bLat-aLat), radians(bLng-aLng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(radians(aLat))*math.Cos(radians(bLat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 6371 * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
func coverage(address *domain.CustomerAddress) error {
	if !validDestination(address) {
		return fmt.Errorf("se requiere una dirección validada con coordenadas")
	}
	lat, e1 := strconv.ParseFloat(os.Getenv("STORE_LAT"), 64)
	lng, e2 := strconv.ParseFloat(os.Getenv("STORE_LNG"), 64)
	max, e3 := strconv.ParseFloat(os.Getenv("DELIVERY_MAX_RADIUS_KM"), 64)
	if e1 != nil || e2 != nil || e3 != nil || max <= 0 {
		if strings.EqualFold(os.Getenv("USE_MOCK_PROVIDERS"), "true") || os.Getenv("USE_MOCK_PROVIDERS") == "" {
			lat, lng, max = -31.4031, -64.2026, 20
		} else {
			return fmt.Errorf("store coordinates and delivery coverage radius are not configured")
		}
	}
	if distanceKM(lat, lng, address.Lat, address.Lng) > max {
		return output.ErrOutsideCoverage
	}
	return nil
}
func (u *DeliveryUseCase) Quote(ctx context.Context, address *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	if err := coverage(address); err != nil {
		return nil, err
	}
	q, err := u.provider.QuoteDelivery(ctx, address)
	if err != nil {
		log.Printf("component=delivery action=quote status=failed error=%q", err)
		return nil, err
	}
	if q == nil || q.QuoteID == "" || q.Currency != "ARS" || q.EstimatedFee < 0 {
		return nil, fmt.Errorf("invalid delivery quote")
	}
	if q.ExpiresAt.IsZero() {
		q.ExpiresAt = time.Now().Add(15 * time.Minute)
	}
	return q, nil
}

func applyQuote(order *domain.Order, addressID uint, q *output.DeliveryQuote) {
	order.TotalAmount = order.TotalAmount - order.DeliveryCost + q.EstimatedFee
	order.DeliveryCost = q.EstimatedFee
	order.DeliveryType = domain.DeliveryTypeDelivery
	order.AddressID = &addressID
	order.DeliveryQuoteID = q.QuoteID
	now := time.Now()
	order.DeliveryQuotedAt = &now
	order.DeliveryQuoteExpiresAt = &q.ExpiresAt
}

func (u *DeliveryUseCase) QuoteForDraft(ctx context.Context, customerPhone string, addressID uint) (*output.DeliveryQuote, *domain.Order, error) {
	address, err := u.addresses.FindByID(addressID)
	if err != nil {
		return nil, nil, err
	}
	if address == nil || address.CustomerPhone != customerPhone {
		return nil, nil, fmt.Errorf("dirección no encontrada para este cliente")
	}
	customer, err := u.customers.FindByWhatsapp(customerPhone)
	if err != nil || customer == nil {
		return nil, nil, fmt.Errorf("cliente no encontrado")
	}
	register, err := u.registers.GetActive()
	if err != nil || register == nil {
		return nil, nil, fmt.Errorf("no hay caja abierta")
	}
	orders, err := u.orders.GetByRegisterID(register.ID)
	if err != nil {
		return nil, nil, err
	}
	var draft *domain.Order
	for i := range orders {
		if orders[i].CustomerID == customer.ID && orders[i].Status == domain.OrderStatusPendingPayment {
			if draft == nil || orders[i].ID > draft.ID {
				draft = &orders[i]
			}
		}
	}
	quote, err := u.Quote(ctx, address)
	if err != nil {
		return nil, nil, err
	}
	if draft == nil {
		return quote, nil, nil
	}
	applyQuote(draft, addressID, quote)
	if err := u.orders.Update(draft); err != nil {
		return nil, nil, err
	}
	return quote, draft, nil
}

func normalizeE164(phone string) (string, error) {
	phone = strings.TrimPrefix(strings.TrimSpace(phone), "+")
	if len(phone) < 8 || len(phone) > 15 {
		return "", fmt.Errorf("invalid customer phone number")
	}
	for _, r := range phone {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("invalid customer phone number")
		}
	}
	return "+" + phone, nil
}

func (u *DeliveryUseCase) Dispatch(ctx context.Context, id uint) (*domain.Order, error) {
	order, err := u.orders.GetByID(id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("order not found")
	}
	if order.Status != domain.OrderStatusPaid {
		return nil, fmt.Errorf("order must be PAID before dispatch")
	}
	if order.DeliveryType != domain.DeliveryTypeDelivery || order.AddressID == nil {
		return nil, fmt.Errorf("order requires a validated delivery address")
	}
	address, err := u.addresses.FindByID(*order.AddressID)
	if err != nil {
		return nil, err
	}
	if err := coverage(address); err != nil {
		return nil, err
	}
	customer, err := u.customers.FindByID(order.CustomerID)
	if err != nil || customer == nil {
		return nil, fmt.Errorf("customer not found")
	}
	phone, err := normalizeE164(customer.Whatsapp)
	if err != nil {
		return nil, err
	}
	requestOrder := *order
	requestOrder.Customer = *customer
	requestOrder.Customer.Whatsapp = phone
	refresh := order.DeliveryQuoteID == "" || order.DeliveryQuotedAt == nil || time.Since(*order.DeliveryQuotedAt) > 14*time.Minute
	if order.DeliveryQuoteExpiresAt != nil && time.Until(*order.DeliveryQuoteExpiresAt) < time.Minute {
		refresh = true
	}
	if refresh {
		q, e := u.Quote(ctx, address)
		if e != nil {
			return nil, e
		}
		order.DeliveryQuoteID = q.QuoteID
		now := time.Now()
		order.DeliveryQuotedAt = &now
		order.DeliveryQuoteExpiresAt = &q.ExpiresAt
		requestOrder.DeliveryQuoteID = q.QuoteID
		if e = u.orders.Update(order); e != nil {
			return nil, e
		}
	}
	result, err := u.provider.CreateDelivery(ctx, &requestOrder, address)
	if errors.Is(err, output.ErrQuoteExpired) {
		q, e := u.Quote(ctx, address)
		if e != nil {
			return nil, e
		}
		order.DeliveryQuoteID = q.QuoteID
		now := time.Now()
		order.DeliveryQuotedAt = &now
		order.DeliveryQuoteExpiresAt = &q.ExpiresAt
		requestOrder.DeliveryQuoteID = q.QuoteID
		if e = u.orders.Update(order); e != nil {
			return nil, e
		}
		result, err = u.provider.CreateDelivery(ctx, &requestOrder, address)
	}
	if err != nil {
		log.Printf("component=delivery action=dispatch order_id=%d status=failed error=%q", id, err)
		return nil, err
	}
	if result == nil || result.ExternalDeliveryID == "" || result.TrackingURL == "" {
		return nil, fmt.Errorf("courier returned incomplete dispatch data")
	}
	order.ExternalDeliveryID = result.ExternalDeliveryID
	order.TrackingURL = result.TrackingURL
	order.DeliveryProviderFee = result.FinalFee
	order.Status = domain.OrderStatusDispatched
	if err := u.orders.Update(order); err != nil {
		log.Printf("component=delivery action=persist_dispatch order_id=%d status=failed error=%q", id, err)
		return nil, err
	}
	if u.notifier != nil {
		message := "🚚✨ ¡Canerollss en camino! ✨\nSeguí tu pedido en vivo: " + result.TrackingURL + "\nTip: calentá tus Canerollss unos segundos en el microondas para disfrutarlos tibios. 🤎"
		if err := u.notifier.SendTextMessage(ctx, customer.Whatsapp, message); err != nil {
			return order, fmt.Errorf("delivery dispatched, but WhatsApp notification failed: %w", err)
		}
	}
	return order, nil
}

func (u *DeliveryUseCase) ApplyCourierStatus(externalID, status string) error {
	order, err := u.orders.GetByExternalDeliveryID(externalID)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("delivery order not found")
	}
	if order.Status != domain.OrderStatusDispatched {
		if order.Status == domain.OrderStatusDelivered || order.Status == domain.OrderStatusCancelled {
			return nil
		}
		return fmt.Errorf("order is not dispatched")
	}
	switch status {
	case "delivered":
		order.Status = domain.OrderStatusDelivered
	case "canceled", "cancelled":
		order.Status = domain.OrderStatusCancelled
	default:
		return nil
	}
	return u.orders.Update(order)
}
