package dto

type CreateOrderRequest struct {
	CustomerName     string             `json:"customer_name" validate:"required"`
	CustomerWhatsapp string             `json:"customer_whatsapp" validate:"required"`
	Note             string             `json:"note"`
	Items            []OrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

type OrderItemRequest struct {
	ToppingID uint `json:"topping_id" validate:"required"`
	Quantity  int  `json:"quantity" validate:"required,min=1"`
	IsCombo   bool `json:"is_combo"`
}
