package delivery

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
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
	if address == nil || address.Lat == 0 || address.Lng == 0 {
		return nil, fmt.Errorf("validated destination coordinates are required")
	}
	max, err := strconv.ParseFloat(os.Getenv("DELIVERY_MAX_RADIUS_KM"), 64)
	if err != nil || max <= 0 {
		max = 20
	}
	lat, e1 := strconv.ParseFloat(os.Getenv("STORE_LAT"), 64)
	lng, e2 := strconv.ParseFloat(os.Getenv("STORE_LNG"), 64)
	if e1 != nil || e2 != nil {
		lat, lng = -31.4031, -64.2026
	}
	dLat, dLng := (address.Lat-lat)*math.Pi/180, (address.Lng-lng)*math.Pi/180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat*math.Pi/180)*math.Cos(address.Lat*math.Pi/180)*math.Sin(dLng/2)*math.Sin(dLng/2)
	if 6371*2*math.Atan2(math.Sqrt(h), math.Sqrt(1-h)) > max {
		return nil, output.ErrOutsideCoverage
	}
	quoteID := fmt.Sprintf("mock-quote-%d", time.Now().UnixNano())
	return &output.DeliveryQuote{
		EstimatedFee: m.DefaultFee,
		Currency:     "ARS",
		DurationMin:  25,
		QuoteID:      quoteID,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}, nil
}

func (m *MockDeliveryProvider) CancelDelivery(_ context.Context, _ string) error { return nil }

func (m *MockDeliveryProvider) CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	deliveryID := fmt.Sprintf("mock-uber-%d", order.ID)
	return &output.DeliveryConfirmation{
		ExternalDeliveryID: deliveryID,
		TrackingURL:        fmt.Sprintf("https://tracking.uber.com/%s", deliveryID),
		Status:             "pending",
		FinalFee:           m.DefaultFee,
	}, nil
}
