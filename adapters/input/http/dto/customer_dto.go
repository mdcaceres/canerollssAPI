package dto

type CreateCustomerRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Whatsapp string `json:"whatsapp" validate:"required,min=10,max=20"`
}
