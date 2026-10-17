package geocoding

import (
	"canerollss/core/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrMissingStreetNumber  = errors.New("Incluí la altura (número de calle) de tu dirección.")
	ErrOutsideCordoba       = errors.New("Por ahora hacemos envíos solo dentro de Córdoba Capital.")
	ErrAddressNotFound      = errors.New("No pudimos encontrar esa calle y altura. Revisalas e intentá de nuevo.")
	ErrGeocodingUnavailable = errors.New("No pudimos validar tu dirección en este momento. Intentá de nuevo en unos minutos.")
)

type GoogleMapsProvider struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewGoogleMapsProvider() *GoogleMapsProvider {
	base := os.Getenv("GOOGLE_GEOCODING_BASE_URL")
	if base == "" {
		base = "https://maps.googleapis.com/maps/api/geocode/json"
	}
	return &GoogleMapsProvider{APIKey: os.Getenv("GOOGLE_MAPS_API_KEY"), BaseURL: base, Client: &http.Client{Timeout: 5 * time.Second}}
}

type googleComponent struct {
	LongName string   `json:"long_name"`
	Types    []string `json:"types"`
}
type googleResponse struct {
	Status  string `json:"status"`
	Results []struct {
		FormattedAddress string            `json:"formatted_address"`
		PartialMatch     bool              `json:"partial_match"`
		Types            []string          `json:"types"`
		Components       []googleComponent `json:"address_components"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
		} `json:"geometry"`
	} `json:"results"`
}

func hasType(types []string, wanted string) bool {
	for _, t := range types {
		if t == wanted {
			return true
		}
	}
	return false
}
func component(components []googleComponent, kind string) string {
	for _, c := range components {
		if hasType(c.Types, kind) {
			return c.LongName
		}
	}
	return ""
}
func cordobaLocality(components []googleComponent) bool {
	locality := component(components, "locality")
	return strings.EqualFold(locality, "Córdoba") || strings.EqualFold(locality, "Cordoba")
}

func (g *GoogleMapsProvider) ValidateAndGeocode(ctx context.Context, streetAndNumber string) (*domain.GeocodedResult, error) {
	if strings.TrimSpace(streetAndNumber) == "" {
		return nil, ErrMissingStreetNumber
	}
	if numberPattern.FindString(streetAndNumber) == "" {
		return nil, ErrMissingStreetNumber
	}
	lowerInput := strings.ToLower(streetAndNumber)
	for _, town := range []string{"villa allende", "carlos paz", "río ceballos", "rio ceballos"} {
		if strings.Contains(lowerInput, town) {
			return nil, ErrOutsideCordoba
		}
	}
	if g.APIKey == "" {
		return nil, ErrGeocodingUnavailable
	}
	endpoint, err := url.Parse(g.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("geocoding URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("address", streetAndNumber)
	query.Set("components", "locality:Cordoba|administrative_area:Cordoba|country:AR")
	query.Set("language", "es")
	query.Set("key", g.APIKey)
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, ErrGeocodingUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrGeocodingUnavailable
	}
	var body googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, ErrGeocodingUnavailable
	}
	if body.Status == "ZERO_RESULTS" || len(body.Results) == 0 {
		return nil, ErrAddressNotFound
	}
	if body.Status != "OK" {
		return nil, ErrGeocodingUnavailable
	}
	sawOutside, sawMissingNumber := false, false
	for _, result := range body.Results {
		if result.PartialMatch {
			continue
		}
		parts := result.Components
		if !cordobaLocality(parts) || component(parts, "country") != "Argentina" {
			sawOutside = true
			continue
		}
		street, number := component(parts, "route"), component(parts, "street_number")
		if street == "" || number == "" {
			sawMissingNumber = true
			continue
		}
		if !(hasType(result.Types, "street_address") || hasType(result.Types, "premise") || hasType(result.Types, "subpremise")) {
			continue
		}
		if result.Geometry.LocationType != "ROOFTOP" && result.Geometry.LocationType != "RANGE_INTERPOLATED" {
			continue
		}
		if result.Geometry.Location.Lat == 0 || result.Geometry.Location.Lng == 0 {
			continue
		}
		neighborhood := component(parts, "neighborhood")
		if neighborhood == "" {
			neighborhood = component(parts, "sublocality")
		}
		return &domain.GeocodedResult{
			FormattedAddress: result.FormattedAddress,
			StreetAndNumber:  street + " " + number,
			Neighborhood:     neighborhood, City: "Córdoba",
			State:   component(parts, "administrative_area_level_1"),
			ZipCode: component(parts, "postal_code"),
			Lat:     result.Geometry.Location.Lat, Lng: result.Geometry.Location.Lng,
			LocationType: result.Geometry.LocationType, Exact: true,
		}, nil
	}
	if sawOutside {
		return nil, ErrOutsideCordoba
	}
	if sawMissingNumber {
		return nil, ErrMissingStreetNumber
	}
	return nil, ErrAddressNotFound
}
