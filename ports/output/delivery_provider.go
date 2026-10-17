package output

import (
	"canerollss/core/domain"
	"context"
	"errors"
	"time"
)

var ErrQuoteExpired = errors.New("delivery quote expired")
var ErrOutsideCoverage = errors.New("delivery destination outside coverage")

type DeliveryQuote struct {
	EstimatedFee float64   `json:"estimated_fee"`
	Currency     string    `json:"currency"`
	DurationMin  int       `json:"duration_min"`
	QuoteID      string    `json:"quote_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type DeliveryConfirmation struct {
	ExternalDeliveryID string  `json:"external_delivery_id"`
	TrackingURL        string  `json:"tracking_url"`
	Status             string  `json:"status"`
	FinalFee           float64 `json:"final_fee"`
}

type DeliveryProvider interface {
	QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*DeliveryQuote, error)
	CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*DeliveryConfirmation, error)
	CancelDelivery(ctx context.Context, externalDeliveryID string) error
}
