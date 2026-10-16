package dto

type CreateOrderRequest struct {
	CustomerID uint              `json:"customer_id" validate:"required"`
	Note       string            `json:"note"`
	Items      []CreateOrderItem `json:"items" validate:"required,min=1"`
}

type CreateOrderItem struct {
	ToppingID uint `json:"topping_id" validate:"required"`
	Quantity  int  `json:"quantity"   validate:"required,min=1"`
	IsCombo   bool `json:"is_combo"`
}
