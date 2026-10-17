package geocoding

import (
	"canerollss/core/domain"
	"context"
	"regexp"
	"strings"
)

type MockGeocodingProvider struct{}

func NewMockGeocodingProvider() *MockGeocodingProvider { return &MockGeocodingProvider{} }

var numberPattern = regexp.MustCompile(`\b\d{1,5}\b`)

func (*MockGeocodingProvider) ValidateAndGeocode(_ context.Context, raw string) (*domain.GeocodedResult, error) {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	if numberPattern.FindString(trimmed) == "" {
		return nil, ErrMissingStreetNumber
	}
	for _, town := range []string{"villa allende", "carlos paz", "río ceballos", "rio ceballos", "buenos aires", "caba"} {
		if strings.Contains(lower, town) {
			return nil, ErrOutsideCordoba
		}
	}
	if strings.Contains(lower, "inventada") {
		return nil, ErrAddressNotFound
	}
	return &domain.GeocodedResult{
		FormattedAddress: trimmed + ", Córdoba, Argentina", StreetAndNumber: trimmed,
		City: "Córdoba", ZipCode: "5000", Lat: -31.4031, Lng: -64.2026,
		LocationType: "ROOFTOP", Exact: true,
	}, nil
}
