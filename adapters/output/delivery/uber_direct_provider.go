package delivery

import (
	"bytes"
	"canerollss/core/domain"
	"canerollss/ports/output"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UberDirectProvider struct {
	customerID, clientID, clientSecret, baseURL, authURL string
	httpClient                                           *http.Client
	mu                                                   sync.RWMutex
	token                                                string
	tokenExpires                                         time.Time
}

func NewUberDirectProvider() *UberDirectProvider {
	base, auth := os.Getenv("UBER_DIRECT_BASE_URL"), os.Getenv("UBER_AUTH_URL")
	if base == "" {
		base = "https://api.uber.com/v1"
	}
	if auth == "" {
		auth = "https://auth.uber.com/oauth/v2/token"
	}
	return &UberDirectProvider{customerID: os.Getenv("UBER_DIRECT_CUSTOMER_ID"), clientID: os.Getenv("UBER_DIRECT_CLIENT_ID"), clientSecret: os.Getenv("UBER_DIRECT_CLIENT_SECRET"), baseURL: base, authURL: auth, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (u *UberDirectProvider) accessToken(ctx context.Context) (string, error) {
	u.mu.RLock()
	if u.token != "" && time.Until(u.tokenExpires) > 5*time.Minute {
		token := u.token
		u.mu.RUnlock()
		return token, nil
	}
	u.mu.RUnlock()
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.token != "" && time.Until(u.tokenExpires) > 5*time.Minute {
		return u.token, nil
	}
	if u.clientID == "" || u.clientSecret == "" {
		return "", fmt.Errorf("Uber Direct credentials are not configured")
	}
	form := url.Values{"client_id": {u.clientID}, "client_secret": {u.clientSecret}, "grant_type": {"client_credentials"}, "scope": {"eats.deliveries"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.authURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Uber OAuth: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Uber OAuth HTTP %d", resp.StatusCode)
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.AccessToken == "" || data.ExpiresIn <= 0 {
		return "", fmt.Errorf("invalid Uber OAuth token")
	}
	u.token, u.tokenExpires = data.AccessToken, time.Now().Add(time.Duration(data.ExpiresIn)*time.Second)
	return u.token, nil
}

func (u *UberDirectProvider) request(ctx context.Context, method, path string, payload, result any) error {
	if u.customerID == "" {
		return fmt.Errorf("Uber Direct customer ID is not configured")
	}
	token, err := u.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		b, e := json.Marshal(payload)
		if e != nil {
			return e
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(u.baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Uber Direct: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.ToLower(string(b))
		if strings.Contains(msg, "quote") && strings.Contains(msg, "expir") {
			return output.ErrQuoteExpired
		}
		if strings.Contains(msg, "out_of_range") || strings.Contains(msg, "undeliverable") || strings.Contains(msg, "outside") {
			return output.ErrOutsideCoverage
		}
		return fmt.Errorf("Uber Direct HTTP %d: %s", resp.StatusCode, string(b))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func structuredAddress(street, postal string) string {
	b, _ := json.Marshal(map[string]any{"street_address": []string{street}, "city": "Córdoba", "state": "Córdoba", "zip_code": postal, "country": "AR"})
	return string(b)
}
func pickup() (string, float64, float64, error) {
	address := os.Getenv("STORE_ADDRESS")
	lat, e1 := strconv.ParseFloat(os.Getenv("STORE_LAT"), 64)
	lng, e2 := strconv.ParseFloat(os.Getenv("STORE_LNG"), 64)
	if address == "" || e1 != nil || e2 != nil || lat == 0 || lng == 0 {
		return "", 0, 0, fmt.Errorf("STORE_ADDRESS, STORE_LAT and STORE_LNG are required")
	}
	return address, lat, lng, nil
}

func (u *UberDirectProvider) QuoteDelivery(ctx context.Context, address *domain.CustomerAddress) (*output.DeliveryQuote, error) {
	if address == nil || address.Lat == 0 || address.Lng == 0 {
		return nil, fmt.Errorf("validated destination coordinates are required")
	}
	pickupAddress, lat, lng, err := pickup()
	if err != nil {
		return nil, err
	}
	body := map[string]any{"pickup_address": structuredAddress(pickupAddress, os.Getenv("STORE_ZIP_CODE")), "pickup_latitude": lat, "pickup_longitude": lng, "dropoff_address": structuredAddress(address.StreetAndNumber, address.ZipCode), "dropoff_latitude": address.Lat, "dropoff_longitude": address.Lng}
	var res struct {
		ID       string `json:"id"`
		Fee      int64  `json:"fee"`
		Currency string `json:"currency"`
		Duration int    `json:"duration"`
		Expires  string `json:"expires"`
	}
	err = u.request(ctx, http.MethodPost, "/customers/"+url.PathEscape(u.customerID)+"/delivery_quotes", body, &res)
	if err != nil {
		return nil, err
	}
	if res.ID == "" || res.Currency != "ARS" || res.Fee < 0 {
		return nil, fmt.Errorf("invalid Uber delivery quote")
	}
	expires := time.Now().Add(15 * time.Minute)
	if t, e := time.Parse(time.RFC3339, res.Expires); e == nil {
		expires = t
	}
	return &output.DeliveryQuote{EstimatedFee: float64(res.Fee) / 100, Currency: res.Currency, DurationMin: res.Duration, QuoteID: res.ID, ExpiresAt: expires}, nil
}

func (u *UberDirectProvider) CreateDelivery(ctx context.Context, order *domain.Order, address *domain.CustomerAddress) (*output.DeliveryConfirmation, error) {
	if order == nil || address == nil || order.DeliveryQuoteID == "" {
		return nil, fmt.Errorf("order, address and delivery quote are required")
	}
	pickupAddress, lat, lng, err := pickup()
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(order.Items))
	for _, item := range order.Items {
		name := item.Topping.Name
		if name == "" {
			name = "Caneroll"
		}
		items = append(items, map[string]any{"name": name, "quantity": item.Quantity, "size": "small"})
	}
	body := map[string]any{"quote_id": order.DeliveryQuoteID, "external_id": strconv.FormatUint(uint64(order.ID), 10), "pickup_address": structuredAddress(pickupAddress, os.Getenv("STORE_ZIP_CODE")), "pickup_latitude": lat, "pickup_longitude": lng, "pickup_name": os.Getenv("STORE_NAME"), "pickup_phone_number": os.Getenv("STORE_PHONE_E164"), "dropoff_address": structuredAddress(address.StreetAndNumber, address.ZipCode), "dropoff_latitude": address.Lat, "dropoff_longitude": address.Lng, "dropoff_name": order.Customer.Name, "dropoff_phone_number": order.Customer.Whatsapp, "dropoff_notes": address.ApartmentNotes, "deliverable_action": "deliverable_action_meet_at_door", "manifest_items": items}
	var res struct {
		UUID        string `json:"uuid"`
		TrackingURL string `json:"tracking_url"`
		Status      string `json:"status"`
		Fee         int64  `json:"fee"`
	}
	err = u.request(ctx, http.MethodPost, "/customers/"+url.PathEscape(u.customerID)+"/deliveries", body, &res)
	if err != nil {
		return nil, err
	}
	if res.UUID == "" || res.TrackingURL == "" {
		return nil, fmt.Errorf("Uber delivery response is incomplete")
	}
	return &output.DeliveryConfirmation{ExternalDeliveryID: res.UUID, TrackingURL: res.TrackingURL, Status: res.Status, FinalFee: float64(res.Fee) / 100}, nil
}

func (u *UberDirectProvider) CancelDelivery(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("external delivery ID is required")
	}
	return u.request(ctx, http.MethodPost, "/customers/"+url.PathEscape(u.customerID)+"/deliveries/"+url.PathEscape(id)+"/cancel", nil, nil)
}
