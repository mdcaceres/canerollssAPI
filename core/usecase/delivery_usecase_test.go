package usecase

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"strings"
	"testing"
	"time"
)

type courierFake struct {
	quotes      int
	dispatches  int
	expiredOnce bool
	phone       string
}

func (f *courierFake) QuoteDelivery(_ context.Context, _ *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	f.quotes++
	return &output.DeliveryQuote{EstimatedFee: 1500, Currency: "ARS", DurationMin: 25, QuoteID: "quote", ExpiresAt: time.Now().Add(15 * time.Minute)}, nil
}
func (f *courierFake) CreateDelivery(_ context.Context, o *domain.Order, _ *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	f.dispatches++
	f.phone = o.Customer.Whatsapp
	if f.expiredOnce && f.dispatches == 1 {
		return nil, output.ErrQuoteExpired
	}
	return &output.DeliveryConfirmation{ExternalDeliveryID: "delivery-1", TrackingURL: "https://tracking.example/1"}, nil
}
func (f *courierFake) CancelDelivery(context.Context, string) error { return nil }

type messageFake struct{ to, body string }

func (m *messageFake) SendTextMessage(_ context.Context, to, body string) error {
	m.to, m.body = to, body
	return nil
}

func TestQuoteForDraftUpdatesTotalAndAddress(t *testing.T) {
	t.Setenv("USE_MOCK_PROVIDERS", "true")
	orders := newMemoryOrderRepo()
	orders.Save(&domain.Order{ID: 1, CashRegisterID: 1, CustomerID: 1, Status: domain.OrderStatusPendingPayment, TotalAmount: 7000})
	addresses := &mockAddressRepo{addresses: []domain.CustomerAddress{{ID: 1, CustomerPhone: "5493515551234", StreetAndNumber: "Colón 1500", City: "Córdoba", Lat: -31.4, Lng: -64.2}}}
	customers := &mockCustomerRepo{customers: map[string]*domain.Customer{"5493515551234": {ID: 1, Whatsapp: "5493515551234"}}}
	courier := &courierFake{}
	u := NewDeliveryUseCase(orders, addresses, customers, &mockCashRepo{}, courier, nil)
	quote, draft, err := u.QuoteForDraft(context.Background(), "5493515551234", 1)
	if err != nil || quote == nil || draft.TotalAmount != 8500 || draft.AddressID == nil || draft.DeliveryQuoteID != "quote" {
		t.Fatalf("quote=%+v draft=%+v err=%v", quote, draft, err)
	}
}

func TestDispatchPaidOrderRefreshesExpiredQuoteAndNotifies(t *testing.T) {
	t.Setenv("USE_MOCK_PROVIDERS", "true")
	old := time.Now().Add(-20 * time.Minute)
	addrID := uint(1)
	orders := newMemoryOrderRepo()
	orders.Save(&domain.Order{ID: 1, CustomerID: 1, Status: domain.OrderStatusPaid, DeliveryType: domain.DeliveryTypeDelivery, AddressID: &addrID, DeliveryQuoteID: "old", DeliveryQuotedAt: &old})
	addresses := &mockAddressRepo{addresses: []domain.CustomerAddress{{ID: 1, StreetAndNumber: "Colón 1500", City: "Córdoba", Lat: -31.4, Lng: -64.2}}}
	customers := &mockCustomerRepo{customers: map[string]*domain.Customer{"5493515551234": {ID: 1, Whatsapp: "5493515551234", Name: "Ana"}}}
	courier := &courierFake{expiredOnce: true}
	message := &messageFake{}
	u := NewDeliveryUseCase(orders, addresses, customers, &mockCashRepo{}, courier, message)
	order, err := u.Dispatch(context.Background(), 1)
	if err != nil || order.Status != domain.OrderStatusDispatched || courier.quotes != 2 || courier.dispatches != 2 || courier.phone != "+5493515551234" || !strings.Contains(message.body, "https://tracking.example/1") {
		t.Fatalf("order=%+v quotes=%d dispatches=%d phone=%s message=%s err=%v", order, courier.quotes, courier.dispatches, courier.phone, message.body, err)
	}
}

func TestDispatchRejectsUnpaidOrder(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.Save(&domain.Order{ID: 1, Status: domain.OrderStatusPendingPayment})
	u := NewDeliveryUseCase(orders, nil, nil, nil, &courierFake{}, nil)
	if _, err := u.Dispatch(context.Background(), 1); err == nil {
		t.Fatal("expected unpaid-order rejection")
	}
}

func TestCourierStatusTransitions(t *testing.T) {
	orders := newMemoryOrderRepo()
	orders.Save(&domain.Order{ID: 1, ExternalDeliveryID: "delivery-1", Status: domain.OrderStatusDispatched})
	u := NewDeliveryUseCase(orders, nil, nil, nil, nil, nil)
	if err := u.ApplyCourierStatus("delivery-1", "delivered"); err != nil {
		t.Fatal(err)
	}
	o, _ := orders.GetByID(1)
	if o.Status != domain.OrderStatusDelivered {
		t.Fatalf("got %s", o.Status)
	}
	if err := u.ApplyCourierStatus("delivery-1", "delivered"); err != nil {
		t.Fatal(err)
	}
	if err := u.ApplyCourierStatus("delivery-1", "canceled"); err != nil {
		t.Fatal(err)
	}
	o, _ = orders.GetByID(1)
	if o.Status != domain.OrderStatusDelivered {
		t.Fatal("stale event changed terminal status")
	}
}
