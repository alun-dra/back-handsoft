package services

import (
	"context"

	"back/internal/ent"
	"back/internal/ent/city"
	"back/internal/ent/commune"
	"back/internal/ent/region"
)

type LocationService struct {
	Client *ent.Client
}

func NewLocationService(client *ent.Client) *LocationService {
	return &LocationService{Client: client}
}

func (s *LocationService) ListRegions(ctx context.Context) ([]*ent.Region, error) {
	return s.Client.Region.
		Query().
		Order(ent.Asc(region.FieldName)).
		All(ctx)
}

func (s *LocationService) ListCitiesByRegion(ctx context.Context, regionID int) ([]*ent.City, error) {
	return s.Client.City.
		Query().
		Where(city.HasRegionWith(region.IDEQ(regionID))).
		Order(ent.Asc(city.FieldName)).
		All(ctx)
}

func (s *LocationService) ListCommunesByCity(ctx context.Context, cityID int) ([]*ent.Commune, error) {
	return s.Client.Commune.
		Query().
		Where(commune.HasCityWith(city.IDEQ(cityID))).
		Order(ent.Asc(commune.FieldName)).
		All(ctx)
}
