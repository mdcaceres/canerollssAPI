package output

import (
	"canerollss/core/domain"
	"context"
)

type DeliveryQuote struct {
	EstimatedFee float64 `json:"estimated_fee"`
	Currency     string  `json:"currency"`
	DurationMin  int     `json:"duration_min"`
	QuoteID      string  `json:"quote_id"`
}

type DeliveryConfirmation struct {
	ExternalDeliveryID string `json:"external_delivery_id"`
	TrackingURL        string `json:"tracking_url"`
	Status             string `json:"status"`
}

type DeliveryProvider interface {
	QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*DeliveryQuote, error)
	CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*DeliveryConfirmation, error)
}
