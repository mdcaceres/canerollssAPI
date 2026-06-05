package dto

type CreateToppingRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=50"`
	SinglePrice float64 `json:"single_price" validate:"required,min=0"`
	ComboPrice  float64 `json:"combo_price" validate:"required,min=0"`
	IsActive    *bool   `json:"is_active" validate:"required"` // Puntero para obligar a enviar true/false y evitar el default
}

type UpdateToppingRequest struct {
	Name        string  `json:"name" validate:"omitempty,min=2,max=50"`
	SinglePrice float64 `json:"single_price" validate:"omitempty,min=0"`
	ComboPrice  float64 `json:"combo_price" validate:"omitempty,min=0"`
	IsActive    *bool   `json:"is_active" validate:"omitempty"`
}
