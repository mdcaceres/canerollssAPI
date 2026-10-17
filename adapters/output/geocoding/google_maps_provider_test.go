package geocoding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleGeocodingExactCordoba(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("language") != "es" || !strings.Contains(r.URL.Query().Get("components"), "country:AR") {
			t.Error("missing query restrictions")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"OK","results":[{"formatted_address":"Av. Colón 1500, Córdoba, Argentina","types":["street_address"],"geometry":{"location":{"lat":-31.401,"lng":-64.202},"location_type":"ROOFTOP"},"address_components":[{"long_name":"Av. Colón","types":["route"]},{"long_name":"1500","types":["street_number"]},{"long_name":"Córdoba","types":["locality"]},{"long_name":"Argentina","types":["country"]},{"long_name":"5000","types":["postal_code"]}]}]}`))
	}))
	defer server.Close()
	provider := &GoogleMapsProvider{APIKey: "test", BaseURL: server.URL, Client: server.Client()}
	result, err := provider.ValidateAndGeocode(context.Background(), "Av. Colón 1500")
	if err != nil || !result.Exact || result.StreetAndNumber != "Av. Colón 1500" || result.Lat == 0 || result.ZipCode != "5000" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestGoogleGeocodingRejectsOtherLocality(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"OK","results":[{"formatted_address":"Villa Allende","types":["street_address"],"geometry":{"location":{"lat":-31.3,"lng":-64.3},"location_type":"ROOFTOP"},"address_components":[{"long_name":"Av. San Martín","types":["route"]},{"long_name":"200","types":["street_number"]},{"long_name":"Villa Allende","types":["locality"]},{"long_name":"Argentina","types":["country"]}]}]}`))
	}))
	defer server.Close()
	provider := &GoogleMapsProvider{APIKey: "test", BaseURL: server.URL, Client: server.Client()}
	_, err := provider.ValidateAndGeocode(context.Background(), "Av. San Martín 200")
	if err != ErrOutsideCordoba {
		t.Fatalf("expected outside Córdoba, got %v", err)
	}
}
