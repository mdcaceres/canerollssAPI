package delivery

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"fmt"
	"time"
)

type MockDeliveryProvider struct {
	DefaultFee float64
}

func NewMockDeliveryProvider(defaultFee ...float64) *MockDeliveryProvider {
	fee := 1500.0
	if len(defaultFee) > 0 && defaultFee[0] > 0 {
		fee = defaultFee[0]
	}
	return &MockDeliveryProvider{DefaultFee: fee}
}

func (m *MockDeliveryProvider) QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	quoteID := fmt.Sprintf("mock-quote-%d", time.Now().UnixNano())
	return &output.DeliveryQuote{
		EstimatedFee: m.DefaultFee,
		Currency:     "ARS",
		DurationMin:  25,
		QuoteID:      quoteID,
	}, nil
}

func (m *MockDeliveryProvider) CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	deliveryID := fmt.Sprintf("mock-uber-%d", order.ID)
	return &output.DeliveryConfirmation{
		ExternalDeliveryID: deliveryID,
		TrackingURL:        fmt.Sprintf("https://tracking.uber.com/%s", deliveryID),
		Status:             "pending",
	}, nil
}
