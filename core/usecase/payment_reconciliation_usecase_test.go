package usecase

import (
	"canerollss/adapters/output/payment"
	"canerollss/core/domain"
	"canerollss/ports/input"
	"context"
	"testing"
	"time"
)

type memoryOrderRepo struct {
	orders map[uint]*domain.Order
}

func newMemoryOrderRepo() *memoryOrderRepo {
	return &memoryOrderRepo{orders: make(map[uint]*domain.Order)}
}

func (m *memoryOrderRepo) Save(o *domain.Order) error {
	if o.ID == 0 {
		o.ID = uint(len(m.orders) + 1)
	}
	m.orders[o.ID] = o
	return nil
}

func (m *memoryOrderRepo) GetByID(id uint) (*domain.Order, error) {
	if o, ok := m.orders[id]; ok {
		return o, nil
	}
	return nil, nil
}

func (m *memoryOrderRepo) GetByRegisterID(regID uint) ([]domain.Order, error) {
	var list []domain.Order
	for _, o := range m.orders {
		if o.CashRegisterID == regID {
			list = append(list, *o)
		}
	}
	return list, nil
}

func (m *memoryOrderRepo) GetByExternalPaymentID(extID string) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.ExternalPaymentID == extID {
			return o, nil
		}
	}
	return nil, nil
}

func (m *memoryOrderRepo) Update(o *domain.Order) error {
	m.orders[o.ID] = o
	return nil
}

func (m *memoryOrderRepo) GetCompletedByRegisterID(regID uint) ([]domain.Order, error) {
	return nil, nil
}

func (m *memoryOrderRepo) GetCompletedByDateRange(start, end time.Time) ([]domain.Order, error) {
	return nil, nil
}

func TestPaymentReconciliation_ExactMatch(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             1,
		CashRegisterID: 1,
		CustomerID:     10,
		TotalAmount:    8500.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	// Seed matching transfer in mock ledger
	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "MP-TX-001",
		Amount:       8500.0,
		Currency:     "ARS",
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Matias Caceres",
	})

	ctx := context.Background()
	res, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusMatched {
		t.Errorf("expected MATCHED, got %s (msg: %s)", res.Status, res.Message)
	}

	// Verify order updated in repo
	updated, _ := orderRepo.GetByID(order.ID)
	if updated.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status PAID, got %s", updated.Status)
	}
	if updated.ExternalPaymentID != "MP-TX-001" {
		t.Errorf("expected external payment ID MP-TX-001, got %s", updated.ExternalPaymentID)
	}
}

func TestPaymentReconciliation_OperationIDMatch(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             2,
		CashRegisterID: 1,
		CustomerID:     11,
		TotalAmount:    7000.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "OP-998877",
		Amount:       7000.0,
		Currency:     "ARS",
		Status:       "approved",
		DateApproved: time.Now(),
	})

	ctx := context.Background()
	res, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{
		OperationID: "OP-998877",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusMatched {
		t.Errorf("expected MATCHED, got %s", res.Status)
	}
}

func TestPaymentReconciliation_AmbiguousMultipleMatches(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             3,
		CashRegisterID: 1,
		CustomerID:     12,
		TotalAmount:    7000.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	// Seed 2 transfers with the exact same amount
	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "TX-AAA",
		Amount:       7000.0,
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Carlos Tevez",
	})
	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "TX-BBB",
		Amount:       7000.0,
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Lionel Messi",
	})

	ctx := context.Background()
	// Without disambiguation -> AMBIGUOUS
	res, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != domain.ReconciliationStatusAmbiguousMultipleMatches {
		t.Errorf("expected AMBIGUOUS_MULTIPLE_MATCHES, got %s", res.Status)
	}

	// With PayerName disambiguation -> MATCHED
	res2, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{
		PayerName: "Messi",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.Status != domain.ReconciliationStatusMatched {
		t.Errorf("expected MATCHED after payer disambiguation, got %s", res2.Status)
	}
	if res2.MatchedTransaction.ID != "TX-BBB" {
		t.Errorf("expected TX-BBB, got %s", res2.MatchedTransaction.ID)
	}
}

func TestPaymentReconciliation_NotFoundYet(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             4,
		CashRegisterID: 1,
		CustomerID:     14,
		TotalAmount:    11000.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	ctx := context.Background()
	res, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusNotFoundYet {
		t.Errorf("expected NOT_FOUND_YET, got %s", res.Status)
	}
}

func TestPaymentReconciliation_AmountMismatch(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             5,
		CashRegisterID: 1,
		CustomerID:     15,
		TotalAmount:    8500.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	// Seed transfer with different amount
	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "TX-DIFF",
		Amount:       5000.0,
		Status:       "approved",
		DateApproved: time.Now(),
	})

	ctx := context.Background()
	res, err := uc.ReconcileCustomerOrder(ctx, order.ID, input.DisambiguationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusAmountMismatch {
		t.Errorf("expected AMOUNT_MISMATCH, got %s", res.Status)
	}
}

func TestPaymentReconciliation_AlreadyUsed_AntiDoubleSpending(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	// Existing completed order already using this transaction ID
	existingOrder := &domain.Order{
		ID:                6,
		TotalAmount:       8500.0,
		Status:            domain.OrderStatusPaid,
		ExternalPaymentID: "TX-USED-123",
	}
	_ = orderRepo.Save(existingOrder)

	// New pending order
	newOrder := &domain.Order{
		ID:          7,
		TotalAmount: 8500.0,
		Status:      domain.OrderStatusPendingPayment,
		CreatedAt:   time.Now(),
	}
	_ = orderRepo.Save(newOrder)

	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "TX-USED-123",
		Amount:       8500.0,
		Status:       "approved",
		DateApproved: time.Now(),
	})

	ctx := context.Background()
	// Attempting to reconcile using already used operation ID
	res, err := uc.ReconcileCustomerOrder(ctx, newOrder.ID, input.DisambiguationOptions{
		OperationID: "TX-USED-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusAlreadyUsed {
		t.Errorf("expected ALREADY_USED, got %s", res.Status)
	}
}

func TestPaymentReconciliation_ReconcileIncomingTransfer_Webhook(t *testing.T) {
	orderRepo := newMemoryOrderRepo()
	mockPayment := payment.NewMockPaymentProvider()
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}

	uc := NewPaymentReconciliationUseCase(orderRepo, cashRepo, mockPayment)

	order := &domain.Order{
		ID:             8,
		CashRegisterID: 1,
		TotalAmount:    12000.0,
		Status:         domain.OrderStatusPendingPayment,
		CreatedAt:      time.Now(),
	}
	_ = orderRepo.Save(order)

	mockPayment.SeedTransfer(domain.PaymentTransaction{
		ID:           "WEBHOOK-TX-99",
		Amount:       12000.0,
		Status:       "approved",
		DateApproved: time.Now(),
	})

	ctx := context.Background()
	res, err := uc.ReconcileIncomingTransfer(ctx, "WEBHOOK-TX-99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != domain.ReconciliationStatusMatched {
		t.Errorf("expected MATCHED, got %s", res.Status)
	}
	if res.OrderID != 8 {
		t.Errorf("expected OrderID 8, got %d", res.OrderID)
	}

	updated, _ := orderRepo.GetByID(8)
	if updated.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status PAID, got %s", updated.Status)
	}
}
