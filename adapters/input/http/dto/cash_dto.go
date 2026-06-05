package dto

type CloseRegisterRequest struct {
	FinalAmount float64 `json:"final_amount" validate:"required,min=0"`
}
