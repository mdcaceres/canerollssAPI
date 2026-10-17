package input

import (
	"canerollss/core/domain"
	"context"
)

// DisambiguationOptions contains optional metadata supplied by the customer or webhook to resolve ambiguities
type DisambiguationOptions struct {
	OperationID string `json:"operation_id,omitempty"`
	PayerName   string `json:"payer_name,omitempty"`
	PayerID     string `json:"payer_id,omitempty"`
}

// PaymentReconciliationUseCasePort defines the driving contract for reconciling incoming transfers against pending orders.
type PaymentReconciliationUseCasePort interface {
	// ReconcileCustomerOrder evaluates whether an incoming bank/CVU transfer matches a specific customer pending order.
	ReconcileCustomerOrder(ctx context.Context, orderID uint, opts DisambiguationOptions) (*domain.ReconciliationResult, error)

	// ReconcileIncomingTransfer reconciles a specific incoming transfer event (e.g. from webhook notification).
	ReconcileIncomingTransfer(ctx context.Context, operationID string) (*domain.ReconciliationResult, error)
}
