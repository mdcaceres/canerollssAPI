package usecase_test

import (
	"canerollss/core/domain"
	"canerollss/core/usecase"
	"errors"
	"testing"
)

type mockToppingRepo struct {
	toppings map[uint]*domain.Topping
	updateFn func(topping *domain.Topping) error
}

func (m *mockToppingRepo) GetAll() ([]domain.Topping, error) {
	var list []domain.Topping
	for _, t := range m.toppings {
		list = append(list, *t)
	}
	return list, nil
}

func (m *mockToppingRepo) Save(topping *domain.Topping) error {
	m.toppings[topping.ID] = topping
	return nil
}

func (m *mockToppingRepo) GetByID(id uint) (*domain.Topping, error) {
	t, ok := m.toppings[id]
	if !ok {
		return nil, errors.New("topping not found")
	}
	return t, nil
}

func (m *mockToppingRepo) Update(topping *domain.Topping) error {
	if m.updateFn != nil {
		return m.updateFn(topping)
	}
	m.toppings[topping.ID] = topping
	return nil
}

func TestUpdateToppingPrices_Success(t *testing.T) {
	repo := &mockToppingRepo{
		toppings: map[uint]*domain.Topping{
			1: {ID: 1, Name: "Nutella", SinglePrice: 1.5, ComboPrice: 1.0, IsActive: true},
		},
	}
	uc := usecase.NewCatalogUseCase(repo)

	newSingle := 2.5
	newCombo := 1.75
	updated, err := uc.UpdateToppingPrices(1, &newSingle, &newCombo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.SinglePrice != 2.5 {
		t.Errorf("expected SinglePrice 2.5, got %v", updated.SinglePrice)
	}
	if updated.ComboPrice != 1.75 {
		t.Errorf("expected ComboPrice 1.75, got %v", updated.ComboPrice)
	}
}

func TestUpdateToppingPrices_PartialSingle(t *testing.T) {
	repo := &mockToppingRepo{
		toppings: map[uint]*domain.Topping{
			1: {ID: 1, Name: "Nutella", SinglePrice: 1.5, ComboPrice: 1.0, IsActive: true},
		},
	}
	uc := usecase.NewCatalogUseCase(repo)

	newSingle := 3.0
	updated, err := uc.UpdateToppingPrices(1, &newSingle, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.SinglePrice != 3.0 {
		t.Errorf("expected SinglePrice 3.0, got %v", updated.SinglePrice)
	}
	if updated.ComboPrice != 1.0 {
		t.Errorf("expected ComboPrice to remain 1.0, got %v", updated.ComboPrice)
	}
}

func TestUpdateToppingPrices_PartialCombo(t *testing.T) {
	repo := &mockToppingRepo{
		toppings: map[uint]*domain.Topping{
			1: {ID: 1, Name: "Nutella", SinglePrice: 1.5, ComboPrice: 1.0, IsActive: true},
		},
	}
	uc := usecase.NewCatalogUseCase(repo)

	newCombo := 1.25
	updated, err := uc.UpdateToppingPrices(1, nil, &newCombo)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if updated.SinglePrice != 1.5 {
		t.Errorf("expected SinglePrice to remain 1.5, got %v", updated.SinglePrice)
	}
	if updated.ComboPrice != 1.25 {
		t.Errorf("expected ComboPrice 1.25, got %v", updated.ComboPrice)
	}
}

func TestUpdateToppingPrices_NotFound(t *testing.T) {
	repo := &mockToppingRepo{
		toppings: map[uint]*domain.Topping{},
	}
	uc := usecase.NewCatalogUseCase(repo)

	newSingle := 2.0
	_, err := uc.UpdateToppingPrices(999, &newSingle, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent topping, got nil")
	}
	if err.Error() != "topping not found" {
		t.Errorf("expected 'topping not found', got '%v'", err.Error())
	}
}
