package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"canerollss/adapters/input/http/handlers"
	"canerollss/core/domain"
	"canerollss/core/usecase"

	"github.com/gofiber/fiber/v2"
)

type mockRepo struct {
	toppings map[uint]*domain.Topping
}

func (m *mockRepo) GetAll() ([]domain.Topping, error) {
	return nil, nil
}

func (m *mockRepo) Save(topping *domain.Topping) error {
	return nil
}

func (m *mockRepo) GetByID(id uint) (*domain.Topping, error) {
	t, ok := m.toppings[id]
	if !ok {
		return nil, errors.New("topping not found")
	}
	return t, nil
}

func (m *mockRepo) Update(topping *domain.Topping) error {
	m.toppings[topping.ID] = topping
	return nil
}

func setupTestApp() (*fiber.App, *mockRepo) {
	repo := &mockRepo{
		toppings: map[uint]*domain.Topping{
			1: {ID: 1, Name: "Nutella", SinglePrice: 1.5, ComboPrice: 1.0, IsActive: true},
		},
	}
	uc := usecase.NewCatalogUseCase(repo)
	handler := handlers.NewCatalogHandler(uc)

	app := fiber.New()
	app.Patch("/api/toppings/:id/price", handler.UpdateToppingPrices)
	return app, repo
}

func TestUpdateToppingPrices_HTTP_Success(t *testing.T) {
	app, _ := setupTestApp()

	payload := map[string]float64{
		"single_price": 2.5,
		"combo_price":  1.75,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/toppings/1/price", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, string(respBody))
	}

	var res domain.Topping
	json.NewDecoder(resp.Body).Decode(&res)
	if res.SinglePrice != 2.5 || res.ComboPrice != 1.75 {
		t.Errorf("expected prices 2.5 and 1.75, got %v and %v", res.SinglePrice, res.ComboPrice)
	}
}

func TestUpdateToppingPrices_HTTP_Partial(t *testing.T) {
	app, _ := setupTestApp()

	payload := map[string]float64{
		"combo_price": 1.9,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/toppings/1/price", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var res domain.Topping
	json.NewDecoder(resp.Body).Decode(&res)
	if res.SinglePrice != 1.5 || res.ComboPrice != 1.9 {
		t.Errorf("expected single 1.5 and combo 1.9, got %v and %v", res.SinglePrice, res.ComboPrice)
	}
}

func TestUpdateToppingPrices_HTTP_EmptyBody(t *testing.T) {
	app, _ := setupTestApp()

	req := httptest.NewRequest(http.MethodPatch, "/api/toppings/1/price", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for empty body, got %d", resp.StatusCode)
	}
}

func TestUpdateToppingPrices_HTTP_NegativePrice(t *testing.T) {
	app, _ := setupTestApp()

	payload := map[string]float64{
		"single_price": -1.0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/toppings/1/price", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for negative price, got %d", resp.StatusCode)
	}
}

func TestUpdateToppingPrices_HTTP_NotFound(t *testing.T) {
	app, _ := setupTestApp()

	payload := map[string]float64{
		"single_price": 2.0,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPatch, "/api/toppings/999/price", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404 for missing topping, got %d", resp.StatusCode)
	}
}
