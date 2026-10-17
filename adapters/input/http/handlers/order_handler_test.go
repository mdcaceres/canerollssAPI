package handlers

import (
	"canerollss/adapters/output/repository"
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDispatchHandlerRejectsUnpaidOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&domain.Topping{}, &domain.Order{}, &domain.OrderItem{}); err != nil {
		t.Fatal(err)
	}
	order := domain.Order{ID: 1, Status: domain.OrderStatusPendingPayment}
	if err = db.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewOrderRepository(db)
	delivery := usecase.NewDeliveryUseCase(repo, nil, nil, nil, nil, nil)
	app := fiber.New()
	app.Post("/api/orders/:id/dispatch-delivery", NewOrderHandler(nil, delivery).DispatchDelivery)
	req := httptest.NewRequest(http.MethodPost, "/api/orders/1/dispatch-delivery", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%v err=%v", resp, err)
	}
	var saved domain.Order
	db.First(&saved, 1)
	if saved.Status != domain.OrderStatusPendingPayment {
		t.Fatal("unpaid order changed status")
	}
}
