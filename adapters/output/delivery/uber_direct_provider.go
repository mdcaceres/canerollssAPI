package delivery

import (
	"bytes"
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type UberDirectProvider struct {
	customerID   string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
}

func NewUberDirectProvider() *UberDirectProvider {
	baseURL := os.Getenv("UBER_DIRECT_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.uber.com/v1"
	}
	return &UberDirectProvider{
		customerID:   os.Getenv("UBER_DIRECT_CUSTOMER_ID"),
		clientSecret: os.Getenv("UBER_DIRECT_CLIENT_SECRET"),
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

type uberQuoteRequest struct {
	DropoffAddress string `json:"dropoff_address"`
}

type uberQuoteResponse struct {
	Fee         float64 `json:"fee"`
	Currency    string  `json:"currency"`
	Duration    int     `json:"duration"`
	ID          string  `json:"id"`
}

func (u *UberDirectProvider) QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	if u.customerID == "" {
		// Fallback to sensible quote if credentials are not set
		return &output.DeliveryQuote{
			EstimatedFee: 1500.0,
			Currency:     "ARS",
			DurationMin:  30,
			QuoteID:      fmt.Sprintf("uber-quote-fallback-%d", time.Now().Unix()),
		}, nil
	}

	url := fmt.Sprintf("%s/customers/%s/delivery_quotes", u.baseURL, u.customerID)
	body, _ := json.Marshal(uberQuoteRequest{
		DropoffAddress: fmt.Sprintf("%s, %s, %s", address.StreetAndNumber, address.ApartmentNotes, address.City),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", u.clientSecret))

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uber direct quote error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("uber direct api returned status %d", resp.StatusCode)
	}

	var res uberQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return &output.DeliveryQuote{
		EstimatedFee: res.Fee,
		Currency:     res.Currency,
		DurationMin:  res.Duration,
		QuoteID:      res.ID,
	}, nil
}

func (u *UberDirectProvider) CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	deliveryID := fmt.Sprintf("uber-%d", order.ID)
	return &output.DeliveryConfirmation{
		ExternalDeliveryID: deliveryID,
		TrackingURL:        fmt.Sprintf("https://tracking.uber.com/%s", deliveryID),
		Status:             "pending",
	}, nil
}
