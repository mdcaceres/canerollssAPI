package tests

import (
	"bytes"
	"canerollss/adapters/output/datasource"
	"canerollss/adapters/output/payment"
	"canerollss/app"
	"canerollss/core/domain"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMercadoPagoWebhook_ReconciliationAndHMAC(t *testing.T) {
	testDB := "canerollss_webhook_recon_test.db"
	_ = os.Remove(testDB)
	defer func() {
		_ = os.Remove(testDB)
		payment.ClearGlobalMockTransfers()
	}()

	secret := "test_mp_secret_key_456"
	os.Setenv("DB_DRIVER", "sqlite")
	os.Setenv("DB_NAME", testDB)
	os.Setenv("USE_MOCK_PROVIDERS", "true")
	os.Setenv("MP_WEBHOOK_SECRET", secret)

	payment.ClearGlobalMockTransfers()
	fiberApp := app.SetupApp()
	db := datasource.GetDB()

	// 1. Setup Cash Register and Customer
	register := domain.CashRegister{
		Status:   domain.RegisterStatusOpen,
		OpenedAt: time.Now(),
	}
	db.Create(&register)

	customer := domain.Customer{
		Name:     "Julian Alvarez",
		Whatsapp: "5493512223344",
	}
	db.Create(&customer)

	// 2. Setup Pending Order
	order := domain.Order{
		CustomerID:     customer.ID,
		CashRegisterID: register.ID,
		TotalAmount:    8500.0,
		Status:         domain.OrderStatusPendingPayment,
		PaymentMethod:  "TRANSFER_ALIAS",
		CreatedAt:      time.Now(),
	}
	db.Create(&order)

	// 3. Seed transfer in global mock payment ledger
	payment.SeedGlobalMockTransfer(domain.PaymentTransaction{
		ID:           "OP-WH-8500",
		Amount:       8500.0,
		Currency:     "ARS",
		Status:       "approved",
		DateApproved: time.Now(),
		PayerName:    "Julian Alvarez",
	})

	// Helper to calculate HMAC header
	calcSignature := func(dataID, reqID, ts string) string {
		manifest := fmt.Sprintf("id:%s;request-id:%s;ts:%s;", dataID, reqID, ts)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(manifest))
		v1 := hex.EncodeToString(mac.Sum(nil))
		return fmt.Sprintf("ts=%s,v1=%s", ts, v1)
	}

	// -------------------------------------------------------------
	// SCENARIO 1: Invalid HMAC Signature rejection (401)
	// -------------------------------------------------------------
	t.Run("Scenario 1: Reject invalid HMAC signature", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/mercadopago?data.id=OP-WH-8500",
			bytes.NewBufferString(`{"action":"payment.created","data":{"id":"OP-WH-8500"}}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-request-id", "req-invalid")
		req.Header.Set("x-signature", "ts=1710000000,v1=badbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbad")

		resp, err := fiberApp.Test(req, 2000)
		if err != nil {
			t.Fatalf("request error: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized for bad HMAC, got %d", resp.StatusCode)
		}
	})

	// -------------------------------------------------------------
	// SCENARIO 2: Valid HMAC Signature and automatic reconciliation
	// -------------------------------------------------------------
	t.Run("Scenario 2: Accept valid signature and reconcile order to PAID", func(t *testing.T) {
		reqID := "req-valid-123"
		ts := "1710000000"
		sig := calcSignature("OP-WH-8500", reqID, ts)

		req := httptest.NewRequest(
			http.MethodPost,
			"/webhook/mercadopago?data.id=OP-WH-8500",
			bytes.NewBufferString(`{"action":"payment.created","data":{"id":"OP-WH-8500"}}`),
		)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-request-id", reqID)
		req.Header.Set("x-signature", sig)

		resp, err := fiberApp.Test(req, 2000)
		if err != nil {
			t.Fatalf("webhook request error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK for valid webhook, got %d", resp.StatusCode)
		}

		// Wait briefly for asynchronous goroutine in HandleWebhook to complete reconciliation
		var updatedOrder domain.Order
		var reconciled bool
		for i := 0; i < 20; i++ {
			time.Sleep(50 * time.Millisecond)
			if err := db.First(&updatedOrder, order.ID).Error; err == nil {
				if updatedOrder.Status == domain.OrderStatusPaid {
					reconciled = true
					break
				}
			}
		}

		if !reconciled {
			t.Fatalf("expected order %d to transition to PAID, got status: %s", order.ID, updatedOrder.Status)
		}
		if updatedOrder.ExternalPaymentID != "OP-WH-8500" {
			t.Errorf("expected external payment ID OP-WH-8500, got: %s", updatedOrder.ExternalPaymentID)
		}
	})
}
