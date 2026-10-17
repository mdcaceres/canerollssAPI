package handlers

import (
	"bytes"
	"canerollss/adapters/output/repository"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUberWebhookVerifiesRawBodyAndTransitionsOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&domain.Order{}); err != nil {
		t.Fatal(err)
	}
	order := domain.Order{ID: 1, ExternalDeliveryID: "delivery-1", Status: domain.OrderStatusDispatched}
	if err = db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewDeliveryUseCase(repository.NewOrderRepository(db), nil, nil, nil, nil, nil)
	app := fiber.New()
	app.Post("/webhook/uber-delivery", NewUberDeliveryWebhookHandler(uc, "secret").HandleWebhook)
	payload := []byte(`{"kind":"event.delivery_status","delivery_id":"delivery-1","status":"delivered"}`)
	invalid := httptest.NewRequest(http.MethodPost, "/webhook/uber-delivery", bytes.NewReader(payload))
	invalid.Header.Set("X-Uber-Signature", "bad")
	response, err := app.Test(invalid)
	if err != nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid signature status=%v err=%v", response, err)
	}
	var unchanged domain.Order
	db.First(&unchanged, 1)
	if unchanged.Status != domain.OrderStatusDispatched {
		t.Fatal("invalid signature changed order")
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(payload)
	valid := httptest.NewRequest(http.MethodPost, "/webhook/uber-delivery", bytes.NewReader(payload))
	valid.Header.Set("X-Uber-Signature", hex.EncodeToString(mac.Sum(nil)))
	response, err = app.Test(valid)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("valid status=%v err=%v", response, err)
	}
	var updated domain.Order
	db.First(&updated, 1)
	if updated.Status != domain.OrderStatusDelivered {
		t.Fatalf("status=%s", updated.Status)
	}
}
