package delivery

import (
	"canerollss/core/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUberQuoteUsesOAuthCoordinatesAndCentavos(t *testing.T) {
	authCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			authCalls++
			if r.FormValue("grant_type") != "client_credentials" || r.FormValue("scope") != "eats.deliveries" {
				t.Error("invalid OAuth form")
			}
			w.Write([]byte(`{"access_token":"token","expires_in":3600}`))
		case "/v1/customers/customer/delivery_quotes":
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Error("missing bearer token")
			}
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["dropoff_latitude"] != float64(-31.4) || body["dropoff_longitude"] != float64(-64.2) {
				t.Error("missing coordinates")
			}
			w.Write([]byte(`{"id":"quote-1","fee":155000,"currency":"ARS","duration":25}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("STORE_ADDRESS", "Av. Caraffa 1000")
	t.Setenv("STORE_LAT", "-31.4031")
	t.Setenv("STORE_LNG", "-64.2026")
	u := &UberDirectProvider{customerID: "customer", clientID: "id", clientSecret: "secret", baseURL: server.URL + "/v1", authURL: server.URL + "/token", httpClient: server.Client()}
	addr := &domain.CustomerAddress{StreetAndNumber: "Colón 1500", City: "Córdoba", Lat: -31.4, Lng: -64.2}
	for i := 0; i < 2; i++ {
		quote, err := u.QuoteDelivery(context.Background(), addr)
		if err != nil || quote.EstimatedFee != 1550 || quote.QuoteID != "quote-1" {
			t.Fatalf("quote=%+v err=%v", quote, err)
		}
	}
	if authCalls != 1 {
		t.Fatalf("expected cached token, got %d auth calls", authCalls)
	}
}
