package output

import (
	"canerollss/core/domain"
	"context"
)

type GeocodingProvider interface {
	ValidateAndGeocode(ctx context.Context, streetAndNumber string) (*domain.GeocodedResult, error)
}
