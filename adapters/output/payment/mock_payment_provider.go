package payment

import (
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	globalMockLedgerMu sync.RWMutex
	globalMockLedger   = make(map[string]domain.PaymentTransaction)
)

// SeedGlobalMockTransfer seeds a transaction into the global mock ledger accessible by any MockPaymentProvider instance
func SeedGlobalMockTransfer(t domain.PaymentTransaction) {
	globalMockLedgerMu.Lock()
	defer globalMockLedgerMu.Unlock()
	if t.Currency == "" {
		t.Currency = "ARS"
	}
	if t.Status == "" {
		t.Status = "approved"
	}
	if t.DateApproved.IsZero() {
		t.DateApproved = time.Now()
	}
	globalMockLedger[t.ID] = t
}

// ClearGlobalMockTransfers clears the global mock ledger
func ClearGlobalMockTransfers() {
	globalMockLedgerMu.Lock()
	defer globalMockLedgerMu.Unlock()
	globalMockLedger = make(map[string]domain.PaymentTransaction)
}

type MockPaymentProvider struct {
	mu        sync.RWMutex
	transfers map[string]domain.PaymentTransaction
}

func NewMockPaymentProvider() *MockPaymentProvider {
	return &MockPaymentProvider{
		transfers: make(map[string]domain.PaymentTransaction),
	}
}

// SeedTransfer adds an incoming transaction to this provider's in-memory mock ledger
func (m *MockPaymentProvider) SeedTransfer(t domain.PaymentTransaction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t.Currency == "" {
		t.Currency = "ARS"
	}
	if t.Status == "" {
		t.Status = "approved"
	}
	if t.DateApproved.IsZero() {
		t.DateApproved = time.Now()
	}
	m.transfers[t.ID] = t
}

// ClearTransfers wipes all seeded mock transactions in this provider
func (m *MockPaymentProvider) ClearTransfers() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transfers = make(map[string]domain.PaymentTransaction)
}

func (m *MockPaymentProvider) SearchApprovedTransfers(ctx context.Context, filter output.TransferSearchFilter) ([]domain.PaymentTransaction, error) {
	merged := make(map[string]domain.PaymentTransaction)

	globalMockLedgerMu.RLock()
	for k, v := range globalMockLedger {
		merged[k] = v
	}
	globalMockLedgerMu.RUnlock()

	m.mu.RLock()
	for k, v := range m.transfers {
		merged[k] = v
	}
	m.mu.RUnlock()

	var results []domain.PaymentTransaction
	for _, t := range merged {
		if filter.Status != "" && !strings.EqualFold(t.Status, filter.Status) {
			continue
		}
		if !filter.FromDate.IsZero() && t.DateApproved.Before(filter.FromDate) {
			continue
		}
		if !filter.ToDate.IsZero() && t.DateApproved.After(filter.ToDate) {
			continue
		}
		results = append(results, t)
	}

	return results, nil
}

func (m *MockPaymentProvider) GetTransferByID(ctx context.Context, transferID string) (*domain.PaymentTransaction, error) {
	m.mu.RLock()
	t, ok := m.transfers[transferID]
	m.mu.RUnlock()
	if ok {
		return &t, nil
	}

	globalMockLedgerMu.RLock()
	gt, gok := globalMockLedger[transferID]
	globalMockLedgerMu.RUnlock()
	if gok {
		return &gt, nil
	}

	return nil, nil
}

func (m *MockPaymentProvider) GeneratePaymentLink(ctx context.Context, order *domain.Order) (*output.PaymentPreferenceResult, error) {
	fakeID := fmt.Sprintf("mock-pref-%d", order.ID)
	return &output.PaymentPreferenceResult{
		PaymentLink:  fmt.Sprintf("https://mercadopago.com.ar/checkout/mock/%s", fakeID),
		PreferenceID: fakeID,
	}, nil
}

func (m *MockPaymentProvider) VerifyPaymentStatus(ctx context.Context, paymentID string) (*output.PaymentStatusResult, error) {
	return &output.PaymentStatusResult{
		ExternalPaymentID: paymentID,
		Status:            "approved",
		IsPaid:            true,
	}, nil
}
