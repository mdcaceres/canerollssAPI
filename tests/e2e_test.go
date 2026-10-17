package tests

import (
	"bytes"
	"canerollss/adapters/output/datasource"
	"canerollss/adapters/output/payment"
	"canerollss/app"
	"canerollss/core/domain"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestEndToEnd_WhatsAppOrderingFlow(t *testing.T) {
	// ISOLATION: Use dedicated SQLite file to guarantee ZERO impact on Docker / Production MySQL
	testDB := "canerollss_e2e_isolated.db"
	_ = os.Remove(testDB)
	defer func() {
		_ = os.Remove(testDB)
	}()

	os.Setenv("DB_DRIVER", "sqlite")
	os.Setenv("DB_NAME", testDB)
	os.Setenv("USE_MOCK_PROVIDERS", "true")
	os.Setenv("WHATSAPP_VERIFY_TOKEN", "canerollss_verify_secret")
	os.Setenv("ADMIN_USERNAME", "admin")
	os.Setenv("ADMIN_PASSWORD", "admin123")

	fiberApp := app.SetupApp()
	db := datasource.GetDB()

	// -------------------------------------------------------------
	// SCENARIO 1: Webhook Handshake Verification (GET /webhook/whatsapp)
	// -------------------------------------------------------------
	t.Run("Scenario 1: Webhook Meta Challenge", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=canerollss_verify_secret&hub.challenge=CHALLENGE_ACCEPTED_XYZ",
			nil,
		)
		resp, err := fiberApp.Test(req, 2000)
		if err != nil {
			t.Fatalf("webhook challenge error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		if string(bodyBytes) != "CHALLENGE_ACCEPTED_XYZ" {
			t.Fatalf("expected challenge echo, got %s", string(bodyBytes))
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 2: Open Cash Register (Prerequisite for Orders)
	// -------------------------------------------------------------
	t.Run("Scenario 2: Admin Login and Open Cash Register", func(t *testing.T) {
		// Login
		loginBody := `{"username":"admin","password":"admin123"}`
		loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		loginResp, err := fiberApp.Test(loginReq, 2000)
		if err != nil || loginResp.StatusCode != http.StatusOK {
			t.Fatalf("login failed: %v, status: %d", err, loginResp.StatusCode)
		}

		var sessionCookie *http.Cookie
		for _, c := range loginResp.Cookies() {
			if c.Name == "session_token" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatalf("session_token cookie not found in login response")
		}

		// Open Register
		openReq := httptest.NewRequest(http.MethodPost, "/api/cash/open", nil)
		openReq.AddCookie(sessionCookie)
		openResp, err := fiberApp.Test(openReq, 2000)
		if err != nil || (openResp.StatusCode != http.StatusOK && openResp.StatusCode != http.StatusCreated) {
			t.Fatalf("open cash register failed: %v, status: %d", err, openResp.StatusCode)
		}

		// Verify register is active in DB
		var activeRegister domain.CashRegister
		err = db.Where("status = ?", domain.RegisterStatusOpen).First(&activeRegister).Error
		if err != nil {
			t.Fatalf("expected open register in DB, got error: %v", err)
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 3: Customer Inquiry on WhatsApp (POST /webhook/whatsapp)
	// -------------------------------------------------------------
	phoneInquiry := "5491111111111"
	t.Run("Scenario 3: Flavor and Menu Inquiry", func(t *testing.T) {
		inquiryPayload := fmt.Sprintf(`{
			"entry": [{
				"changes": [{
					"value": {
						"contacts": [{"profile": {"name": "Matias"}, "wa_id": "%s"}],
						"messages": [{
							"from": "%s",
							"id": "wamid.inquiry.001",
							"type": "text",
							"text": { "body": "Hola CaneBot! Qué sabores tienen y cuánto salen?" }
						}]
					}
				}]
			}]
		}`, phoneInquiry, phoneInquiry)

		req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewBufferString(inquiryPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := fiberApp.Test(req, 2000)
		if err != nil {
			t.Fatalf("inquiry request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK from webhook, got %d", resp.StatusCode)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		if string(bodyBytes) != "EVENT_RECEIVED" {
			t.Fatalf("expected EVENT_RECEIVED, got %s", string(bodyBytes))
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 4: Save Customer Address 1:N tied to wa_id
	// -------------------------------------------------------------
	phoneCustomer := "5491122222222"
	t.Run("Scenario 4: Customer Address Management 1:N", func(t *testing.T) {
		addressPayload := fmt.Sprintf(`{
			"entry": [{
				"changes": [{
					"value": {
						"contacts": [{"profile": {"name": "Matias"}, "wa_id": "%s"}],
						"messages": [{
							"from": "%s",
							"id": "wamid.addr.001",
							"type": "text",
							"text": { "body": "Guardá mi dirección de Casa: Av. Santa Fe 2345 piso 6B" }
						}]
					}
				}]
			}]
		}`, phoneCustomer, phoneCustomer)

		req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewBufferString(addressPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := fiberApp.Test(req, 2000)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("address webhook failed: %v", err)
		}

		// Wait 4.2s for debounce timer to persist address
		time.Sleep(4200 * time.Millisecond)

		var savedAddress domain.CustomerAddress
		err = db.Where("customer_phone = ?", phoneCustomer).First(&savedAddress).Error
		if err != nil {
			t.Fatalf("address was not saved in DB for phone %s: %v", phoneCustomer, err)
		}
		if !strings.Contains(savedAddress.StreetAndNumber, "Av. Santa Fe") {
			t.Fatalf("unexpected street: %s", savedAddress.StreetAndNumber)
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 5: Multi-Item Order with Delivery & Payment Link
	// -------------------------------------------------------------
	var createdOrder domain.Order
	t.Run("Scenario 5: Multi-Item Order Placement with Delivery", func(t *testing.T) {
		orderPayload := fmt.Sprintf(`{
			"entry": [{
				"changes": [{
					"value": {
						"contacts": [{"profile": {"name": "Matias"}, "wa_id": "%s"}],
						"messages": [{
							"from": "%s",
							"id": "wamid.order.001",
							"type": "text",
							"text": { "body": "Quiero 1 combo de Clasico para delivery a mi Casa, confirmo mi pedido!" }
						}]
					}
				}]
			}]
		}`, phoneCustomer, phoneCustomer)

		req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", bytes.NewBufferString(orderPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := fiberApp.Test(req, 2000)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("order webhook failed: %v", err)
		}

		// Wait 4.2s for debounce buffer to process order creation
		time.Sleep(4200 * time.Millisecond)

		// Verify order in DB
		err = db.Preload("Items").Where("delivery_type = ?", domain.DeliveryTypeDelivery).Last(&createdOrder).Error
		if err != nil {
			t.Fatalf("order was not created in DB: %v", err)
		}

		if createdOrder.Status != domain.OrderStatusPendingPayment {
			t.Fatalf("expected status PENDING_PAYMENT, got %s", createdOrder.Status)
		}
		if createdOrder.DeliveryCost <= 0 {
			t.Fatalf("expected positive delivery cost, got %f", createdOrder.DeliveryCost)
		}
		if createdOrder.PaymentMethod != "TRANSFER_ALIAS" {
			t.Fatalf("expected PaymentMethod TRANSFER_ALIAS, got %s", createdOrder.PaymentMethod)
		}
		if len(createdOrder.Items) == 0 {
			t.Fatalf("expected order items, got 0")
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 6: Payment Confirmation via Webhook (POST /webhook/mercadopago)
	// -------------------------------------------------------------
	t.Run("Scenario 6: Payment Webhook Confirmation to PAID", func(t *testing.T) {
		transferID := fmt.Sprintf("OP-E2E-%d", createdOrder.ID)
		payment.SeedGlobalMockTransfer(domain.PaymentTransaction{
			ID:           transferID,
			Amount:       createdOrder.TotalAmount,
			Currency:     "ARS",
			Status:       "approved",
			DateApproved: time.Now(),
			PayerName:    "Matias",
		})

		mpPayload := fmt.Sprintf(`{
			"action": "payment.created",
			"type": "payment",
			"data": { "id": "%s" }
		}`, transferID)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/webhook/mercadopago?data.id=%s", transferID), bytes.NewBufferString(mpPayload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := fiberApp.Test(req, 2000)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("mercadopago webhook failed: %v", err)
		}

		// Wait briefly for async payment processing
		var updatedOrder domain.Order
		var reconciled bool
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			if err := db.First(&updatedOrder, createdOrder.ID).Error; err == nil {
				if updatedOrder.Status == domain.OrderStatusPaid {
					reconciled = true
					break
				}
			}
		}

		if !reconciled {
			t.Fatalf("expected order %d to transition to PAID, got status: %s", createdOrder.ID, updatedOrder.Status)
		}
		if updatedOrder.ExternalPaymentID != transferID {
			t.Errorf("expected external payment ID %s, got: %s", transferID, updatedOrder.ExternalPaymentID)
		}
	})
}
