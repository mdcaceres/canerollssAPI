package output

import (
	"canerollss/core/domain"
	"context"
	"time"
)

type PaymentPreferenceResult struct {
	PreferenceID string `json:"preference_id"`
	PaymentLink  string `json:"payment_link"`
}

type PaymentStatusResult struct {
	ExternalPaymentID string `json:"external_payment_id"`
	Status            string `json:"status"` // "approved", "pending", "rejected"
	IsPaid            bool   `json:"is_paid"`
}

type TransferSearchFilter struct {
	FromDate time.Time
	ToDate   time.Time
	Status   string // e.g. "approved"
}

type PaymentProvider interface {
	SearchApprovedTransfers(ctx context.Context, filter TransferSearchFilter) ([]domain.PaymentTransaction, error)
	GetTransferByID(ctx context.Context, operationID string) (*domain.PaymentTransaction, error)
	GeneratePaymentLink(ctx context.Context, order *domain.Order) (*PaymentPreferenceResult, error)
	VerifyPaymentStatus(ctx context.Context, externalPaymentID string) (*PaymentStatusResult, error)
}

