package dto

type LoginRequest struct {
	Username string `json:"username" validate:"required,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=4,max=50"`
	Password string `json:"password" validate:"required,min=6"`
}
