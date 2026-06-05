package dto

type CloseMonthRequest struct {
	Year  int `json:"year" validate:"required,min=1"`
	Month int `json:"month" validate:"required,min=1,max=12"`
}
