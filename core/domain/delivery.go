package domain

import "time"

// GeocodedResult is a provider-independent, exact street address.
type GeocodedResult struct {
	FormattedAddress string
	StreetAndNumber  string
	Neighborhood     string
	City             string
	State            string
	ZipCode          string
	Lat              float64
	Lng              float64
	LocationType     string
	Exact            bool
}

type DeliveryQuote struct {
	ID                 string
	FeeARS             float64
	Currency           string
	DurationMinutes    int
	EstimatedArrivalAt time.Time
	ExpiresAt          time.Time
}

type DeliveryRequest struct {
	OrderReference string
	QuoteID        string
	CustomerName   string
	CustomerPhone  string
	Destination    CustomerAddress
	Instructions   string
	Items          []DeliveryItem
}

type DeliveryItem struct {
	Name     string
	Quantity int
	Size     string
	Weight   int
}

type DeliveryDispatchResult struct {
	ExternalDeliveryID string
	Status             string
	TrackingURL        string
	FinalFeeARS        float64
}
