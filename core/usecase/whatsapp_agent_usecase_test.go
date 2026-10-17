package usecase

import (
	"canerollss/adapters/output/payment"
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"strings"
	"testing"
	"time"
)

// Mock repos
type mockOrderRepo struct {
	orders []domain.Order
}

func (m *mockOrderRepo) Save(order *domain.Order) error {
	if order.ID == 0 {
		order.ID = uint(len(m.orders) + 1)
	}
	m.orders = append(m.orders, *order)
	return nil
}
func (m *mockOrderRepo) GetByID(id uint) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.ID == id {
			return &o, nil
		}
	}
	return nil, nil
}
func (m *mockOrderRepo) GetByRegisterID(registerID uint) ([]domain.Order, error) { return nil, nil }
func (m *mockOrderRepo) GetByExternalPaymentID(extID string) (*domain.Order, error) {
	for _, o := range m.orders {
		if o.ExternalPaymentID == extID {
			return &o, nil
		}
	}
	return nil, nil
}
func (m *mockOrderRepo) Update(order *domain.Order) error {
	for i, o := range m.orders {
		if o.ID == order.ID {
			m.orders[i] = *order
			return nil
		}
	}
	return nil
}
func (m *mockOrderRepo) GetCompletedByRegisterID(regID uint) ([]domain.Order, error) { return nil, nil }
func (m *mockOrderRepo) GetCompletedByDateRange(start, end time.Time) ([]domain.Order, error) {
	return nil, nil
}

type mockCashRepo struct {
	active *domain.CashRegister
}

func (m *mockCashRepo) GetActive() (*domain.CashRegister, error) {
	if m.active == nil {
		return &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}, nil
	}
	return m.active, nil
}
func (m *mockCashRepo) Save(c *domain.CashRegister) error          { return nil }
func (m *mockCashRepo) Update(c *domain.CashRegister) error        { return nil }
func (m *mockCashRepo) GetHistory(page, pageSize int) ([]domain.CashRegister, int64, error) {
	return nil, 0, nil
}

type mockCustomerRepo struct {
	customers map[string]*domain.Customer
}

func (m *mockCustomerRepo) FindByWhatsapp(wa string) (*domain.Customer, error) {
	if c, ok := m.customers[wa]; ok {
		return c, nil
	}
	return nil, nil
}
func (m *mockCustomerRepo) FindByID(id uint) (*domain.Customer, error) {
	for _, c := range m.customers {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, nil
}
func (m *mockCustomerRepo) Save(c *domain.Customer) error {
	c.ID = uint(len(m.customers) + 1)
	m.customers[c.Whatsapp] = c
	return nil
}
func (m *mockCustomerRepo) Update(c *domain.Customer) error {
	m.customers[c.Whatsapp] = c
	return nil
}

type mockToppingRepo struct {
	toppings []domain.Topping
}

func (m *mockToppingRepo) GetAll() ([]domain.Topping, error) {
	return m.toppings, nil
}
func (m *mockToppingRepo) Save(t *domain.Topping) error {
	m.toppings = append(m.toppings, *t)
	return nil
}
func (m *mockToppingRepo) GetByID(id uint) (*domain.Topping, error) {
	for _, t := range m.toppings {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}
func (m *mockToppingRepo) Update(t *domain.Topping) error { return nil }

type mockAddressRepo struct {
	addresses []domain.CustomerAddress
}

func (m *mockAddressRepo) Save(a *domain.CustomerAddress) error {
	a.ID = uint(len(m.addresses) + 1)
	m.addresses = append(m.addresses, *a)
	return nil
}
func (m *mockAddressRepo) FindByPhone(phone string) ([]domain.CustomerAddress, error) {
	var list []domain.CustomerAddress
	for _, a := range m.addresses {
		if a.CustomerPhone == phone {
			list = append(list, a)
		}
	}
	return list, nil
}
func (m *mockAddressRepo) FindByID(id uint) (*domain.CustomerAddress, error) {
	for _, a := range m.addresses {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, nil
}
func (m *mockAddressRepo) SetDefault(id uint, phone string) error { return nil }
func (m *mockAddressRepo) Delete(id uint, phone string) error     { return nil }

type mockDelivery struct{}

func (m *mockDelivery) QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	return &output.DeliveryQuote{EstimatedFee: 1500.0, Currency: "ARS", DurationMin: 25}, nil
}
func (m *mockDelivery) CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	return &output.DeliveryConfirmation{ExternalDeliveryID: "uber-123", TrackingURL: "https://tracking.uber.com/123"}, nil
}

type mockPayment struct{}

func (m *mockPayment) SearchApprovedTransfers(ctx context.Context, filter output.TransferSearchFilter) ([]domain.PaymentTransaction, error) {
	return nil, nil
}
func (m *mockPayment) GetTransferByID(ctx context.Context, id string) (*domain.PaymentTransaction, error) {
	return nil, nil
}
func (m *mockPayment) GeneratePaymentLink(ctx context.Context, order *domain.Order) (*output.PaymentPreferenceResult, error) {
	return &output.PaymentPreferenceResult{
		PreferenceID: "pref-xyz",
		PaymentLink:  "https://sandbox.mercadopago.com/checkout/v1/redirect?pref_id=pref-xyz",
	}, nil
}
func (m *mockPayment) VerifyPaymentStatus(ctx context.Context, id string) (*output.PaymentStatusResult, error) {
	return &output.PaymentStatusResult{ExternalPaymentID: id, Status: "approved", IsPaid: true}, nil
}

func setupTestAgent() *WhatsAppAgentUseCase {
	orderRepo := &mockOrderRepo{}
	cashRepo := &mockCashRepo{active: &domain.CashRegister{ID: 1, Status: domain.RegisterStatusOpen}}
	custRepo := &mockCustomerRepo{customers: make(map[string]*domain.Customer)}
	topRepo := &mockToppingRepo{
		toppings: []domain.Topping{
			{ID: 1, Name: "Clasico", SinglePrice: 4000, ComboPrice: 7000, IsActive: true},
			{ID: 2, Name: "Nutella", SinglePrice: 4500, ComboPrice: 8000, IsActive: true},
			{ID: 3, Name: "Pistacho premium", SinglePrice: 6500, ComboPrice: 12000, IsActive: true},
		},
	}
	addrRepo := &mockAddressRepo{}

	orderUC := NewOrderUseCase(orderRepo, cashRepo, custRepo, topRepo)
	addrUC := NewAddressUseCase(addrRepo)

	payment.ClearGlobalMockTransfers()
	return NewWhatsAppAgentUseCase(orderUC, addrUC, custRepo, topRepo, cashRepo, &mockDelivery{}, payment.NewMockPaymentProvider())
}

func TestWhatsAppAgent_MenuInquiryWithoutToppingJargon(t *testing.T) {
	agent := setupTestAgent()
	ctx := context.Background()
	sessionCtx := agent.BuildSessionContext("5491112345678", "Carlos")

	reply, _, err := agent.ProcessMessage(ctx, sessionCtx, "¿Qué sabores tienen y cuánto salen?", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(strings.ToLower(reply), "topping") {
		t.Errorf("violation: response contains forbidden word 'topping': %s", reply)
	}

	if !strings.Contains(reply, "Nutella") || !strings.Contains(reply, "Pistacho premium") {
		t.Errorf("expected flavors in reply, got:\n%s", reply)
	}

	// Verify brand copy
	if !strings.Contains(reply, "¡Te compartimos nuestro menú! 🤎🍥") {
		t.Errorf("expected official welcome copy in reply, got:\n%s", reply)
	}
}

func TestWhatsAppAgent_CreateOrderWithAliasTransfer(t *testing.T) {
	agent := setupTestAgent()
	ctx := context.Background()
	sessionCtx := agent.BuildSessionContext("5491112345678", "Carlos")

	reply, _, err := agent.ProcessMessage(ctx, sessionCtx, "Quiero pedir 1 combo de Nutella, confirmo mi pedido para retirar", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(reply, "Total") {
		t.Errorf("expected order confirmation in reply, got:\n%s", reply)
	}

	// Verify NO third-party payment links are sent
	if strings.Contains(reply, "https://sandbox.mercadopago.com") {
		t.Errorf("checkout link should NOT be present in direct transfer mode, got:\n%s", reply)
	}

	// Verify official receipt fields (Alias Caneroll.mp and Titular Ivonne Abreu)
	if !strings.Contains(reply, "Caneroll.mp") || !strings.Contains(reply, "Ivonne Abreu") {
		t.Errorf("expected official receipt details (Caneroll.mp / Ivonne Abreu), got:\n%s", reply)
	}
}

func TestWhatsAppAgent_PaymentVerification(t *testing.T) {
	agent := setupTestAgent()
	ctx := context.Background()
	sessionCtx := agent.BuildSessionContext("5491112345678", "Carlos")

	reply, _, err := agent.ProcessMessage(ctx, sessionCtx, "Ya pagué mi pedido", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.ToLower(reply), "pago") && !strings.Contains(strings.ToLower(reply), "acreditad") {
		t.Errorf("expected payment confirmation in reply, got:\n%s", reply)
	}

	// Verify warming tip and instagram tag
	if !strings.Contains(reply, "5-10 segundos en microondas") || !strings.Contains(reply, "Instagram") {
		t.Errorf("expected warming and instagram tips in payment confirmation, got:\n%s", reply)
	}
}

func TestWhatsAppAgent_TransferReconciliationToolExecution(t *testing.T) {
	agent := setupTestAgent()
	ctx := context.Background()

	// 1. Seed customer and order in PENDING_PAYMENT
	cust := &domain.Customer{ID: 99, Whatsapp: "5493510009999", Name: "Matias"}
	_ = agent.customerRepo.Save(cust)

	order := &domain.Order{
		ID:             55,
		CustomerID:     cust.ID,
		CashRegisterID: 1,
		TotalAmount:    7000.0,
		Status:         domain.OrderStatusPendingPayment,
		PaymentMethod:  "TRANSFER_ALIAS",
		CreatedAt:      time.Now(),
	}
	_ = agent.orderUseCase.orderRepo.Save(order)

	// Seed transfer in mock ledger
	if mockP, ok := agent.paymentPort.(*payment.MockPaymentProvider); ok {
		mockP.SeedTransfer(domain.PaymentTransaction{
			ID:           "OP-TEST-777",
			Amount:       7000.0,
			Currency:     "ARS",
			Status:       "approved",
			DateApproved: time.Now(),
			PayerName:    "Matias Caceres",
		})
	}

	// 2. Execute verificar_estado_pago tool
	toolRes := agent.executeTool(ctx, cust.Whatsapp, cust.Name, "verificar_estado_pago", map[string]any{
		"pedido_id": uint(55),
	})

	if status, ok := toolRes["status"].(domain.ReconciliationStatus); !ok || status != domain.ReconciliationStatusMatched {
		t.Fatalf("expected tool status MATCHED, got: %v", toolRes)
	}

	if pagado, ok := toolRes["pagado"].(bool); !ok || !pagado {
		t.Fatalf("expected pagado = true, got: %v", toolRes)
	}

	// 3. Verify order transitioned to PAID in database
	updatedOrder, _ := agent.orderUseCase.GetOrderByID(55)
	if updatedOrder.Status != domain.OrderStatusPaid {
		t.Errorf("expected order status PAID, got: %s", updatedOrder.Status)
	}
}

func TestWhatsAppAgent_ToolConfigGating(t *testing.T) {
	agent := setupTestAgent()

	// 1. Closed Cash Register
	cfgClosed := agent.ResolveToolConfig(false)
	if cfgClosed == nil || cfgClosed.FunctionCallingConfig == nil {
		t.Fatal("expected non-nil FunctionCallingConfig when register is closed")
	}

	allowedClosed := cfgClosed.FunctionCallingConfig.AllowedFunctionNames
	if len(allowedClosed) != 2 {
		t.Fatalf("expected 2 allowed functions when register is closed, got %d: %v", len(allowedClosed), allowedClosed)
	}

	allowedMap := make(map[string]bool)
	for _, fn := range allowedClosed {
		allowedMap[fn] = true
	}

	if !allowedMap["consultar_catalogo"] || !allowedMap["verificar_estado_pago"] {
		t.Errorf("expected consultar_catalogo and verificar_estado_pago to be allowed, got: %v", allowedClosed)
	}

	if allowedMap["crear_pedido_con_link_pago"] {
		t.Errorf("SECURITY VIOLATION: crear_pedido_con_link_pago must NOT be allowed when cash register is closed")
	}

	// 2. Open Cash Register
	cfgOpen := agent.ResolveToolConfig(true)
	if cfgOpen == nil || cfgOpen.FunctionCallingConfig == nil {
		t.Fatal("expected non-nil FunctionCallingConfig when register is open")
	}

	allowedOpen := cfgOpen.FunctionCallingConfig.AllowedFunctionNames
	if len(allowedOpen) != 6 {
		t.Fatalf("expected 6 allowed functions when register is open, got %d: %v", len(allowedOpen), allowedOpen)
	}

	openMap := make(map[string]bool)
	for _, fn := range allowedOpen {
		openMap[fn] = true
	}

	for _, expectedFn := range []string{
		"consultar_catalogo",
		"obtener_direcciones_cliente",
		"guardar_direccion_cliente",
		"cotizar_envio",
		"crear_pedido_con_link_pago",
		"verificar_estado_pago",
	} {
		if !openMap[expectedFn] {
			t.Errorf("expected %s to be allowed when register is open", expectedFn)
		}
	}
}

func TestWhatsAppAgent_TransferDisambiguationAndEdgeCases(t *testing.T) {
	agent := setupTestAgent()
	ctx := context.Background()

	// Seed customer and order
	cust := &domain.Customer{ID: 101, Whatsapp: "5493517778888", Name: "Disambiguation User"}
	_ = agent.customerRepo.Save(cust)

	order1 := &domain.Order{
		ID:             60,
		CustomerID:     cust.ID,
		CashRegisterID: 1,
		TotalAmount:    8000.0,
		Status:         domain.OrderStatusPendingPayment,
		PaymentMethod:  "TRANSFER_ALIAS",
		CreatedAt:      time.Now(),
	}
	_ = agent.orderUseCase.orderRepo.Save(order1)

	// Case 1: NOT_FOUND_YET (no transfers in ledger)
	resNotFound := agent.executeTool(ctx, cust.Whatsapp, cust.Name, "verificar_estado_pago", map[string]any{
		"pedido_id": uint(60),
	})
	if resNotFound["status"] != domain.ReconciliationStatusNotFoundYet {
		t.Fatalf("expected NOT_FOUND_YET, got %v", resNotFound)
	}

	// Seed 2 transfers with the same amount of 8000
	mockP, _ := agent.paymentPort.(*payment.MockPaymentProvider)
	mockP.SeedTransfer(domain.PaymentTransaction{
		ID:           "OP-AMBIGUOUS-A",
		Amount:       8000.0,
		Currency:     "ARS",
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Ana Gomez",
	})
	mockP.SeedTransfer(domain.PaymentTransaction{
		ID:           "OP-AMBIGUOUS-B",
		Amount:       8000.0,
		Currency:     "ARS",
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Carlos Lopez",
	})

	// Case 2: Ambiguous match without operation ID
	resAmbiguous := agent.executeTool(ctx, cust.Whatsapp, cust.Name, "verificar_estado_pago", map[string]any{
		"pedido_id": uint(60),
	})
	if resAmbiguous["status"] != domain.ReconciliationStatusAmbiguousMultipleMatches {
		t.Fatalf("expected AMBIGUOUS_MULTIPLE_MATCHES, got %v", resAmbiguous)
	}

	// Case 3: Disambiguation with operation ID (passed as numeric / string)
	resResolved := agent.executeTool(ctx, cust.Whatsapp, cust.Name, "verificar_estado_pago", map[string]any{
		"pedido_id":        uint(60),
		"numero_operacion": "OP-AMBIGUOUS-A",
	})
	if resResolved["status"] != domain.ReconciliationStatusMatched {
		t.Fatalf("expected MATCHED, got %v", resResolved)
	}

	savedOrder1, _ := agent.orderUseCase.GetOrderByID(60)
	if savedOrder1.Status != domain.OrderStatusPaid {
		t.Fatalf("expected order 60 to be PAID, got %s", savedOrder1.Status)
	}
	if savedOrder1.ExternalPaymentID != "OP-AMBIGUOUS-A" {
		t.Fatalf("expected external payment ID OP-AMBIGUOUS-A, got %s", savedOrder1.ExternalPaymentID)
	}

	// Case 4: ALREADY_USED anti double-spending
	order2 := &domain.Order{
		ID:             61,
		CustomerID:     cust.ID,
		CashRegisterID: 1,
		TotalAmount:    8000.0,
		Status:         domain.OrderStatusPendingPayment,
		PaymentMethod:  "TRANSFER_ALIAS",
		CreatedAt:      time.Now(),
	}
	_ = agent.orderUseCase.orderRepo.Save(order2)

	resReused := agent.executeTool(ctx, cust.Whatsapp, cust.Name, "verificar_estado_pago", map[string]any{
		"pedido_id":        uint(61),
		"numero_operacion": "OP-AMBIGUOUS-A",
	})
	if resReused["status"] != domain.ReconciliationStatusAlreadyUsed {
		t.Fatalf("expected ALREADY_USED, got %v", resReused)
	}
}

