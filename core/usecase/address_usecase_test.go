package usecase

import (
	"canerollss/core/domain"
	"context"
	"errors"
	"testing"
)

type fakeGeocoder struct {
	result *domain.GeocodedResult
	err    error
}

func (f fakeGeocoder) ValidateAndGeocode(_ context.Context, _ string) (*domain.GeocodedResult, error) {
	return f.result, f.err
}

func TestSaveAddressValidatesBeforePersistence(t *testing.T) {
	repo := &mockAddressRepo{}
	valid := &domain.GeocodedResult{Exact: true, StreetAndNumber: "Av. Colón 1500", FormattedAddress: "Av. Colón 1500, Córdoba", City: "Córdoba", Neighborhood: "Centro", ZipCode: "5000", Lat: -31.4, Lng: -64.2}
	u := NewAddressUseCase(repo, fakeGeocoder{result: valid})
	first, err := u.SaveAddress("5493515551234", "Casa", "Colón 1500", "4to B")
	if err != nil || first.ID == 0 || !first.IsDefault || first.ApartmentNotes != "4to B" || first.Lat == 0 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := u.SaveAddress("5493515551234", "Trabajo", "Colón 1500", "")
	if err != nil || second.IsDefault {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if len(repo.addresses) != 2 {
		t.Fatal("expected two saved addresses")
	}
}

func TestSaveAddressRejectsGeocodingError(t *testing.T) {
	repo := &mockAddressRepo{}
	u := NewAddressUseCase(repo, fakeGeocoder{err: errors.New("not found")})
	if _, err := u.SaveAddress("5493515551234", "Casa", "Inventada 9999", ""); err == nil {
		t.Fatal("expected validation error")
	}
	if len(repo.addresses) != 0 {
		t.Fatal("unvalidated address was persisted")
	}
}
