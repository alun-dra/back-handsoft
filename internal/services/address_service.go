package services

import (
	"context"
	"errors"
	"strings"

	"back/internal/ent"
	"back/internal/ent/address"
	"back/internal/ent/user"
)

var ErrAddressInvalidInput = errors.New("invalid address input")

type AddressService struct {
	Client *ent.Client
}

func NewAddressService(client *ent.Client) *AddressService {
	return &AddressService{Client: client}
}

type CreateAddressInput struct {
	CommuneID int
	Street    string
	Number    string
	Apartment *string
}

func (s *AddressService) CreateForUser(ctx context.Context, userID int, in CreateAddressInput) (*ent.Address, error) {
	in.Street = strings.TrimSpace(in.Street)
	in.Number = strings.TrimSpace(in.Number)
	if in.CommuneID <= 0 || in.Street == "" || in.Number == "" {
		return nil, ErrAddressInvalidInput
	}

	// Crea dirección y la asocia al usuario + comuna
	create := s.Client.Address.Create().
		SetUserID(userID).
		SetCommuneID(in.CommuneID).
		SetStreet(in.Street).
		SetNumber(in.Number)

	if in.Apartment != nil && strings.TrimSpace(*in.Apartment) != "" {
		create.SetApartment(strings.TrimSpace(*in.Apartment))
	}

	return create.Save(ctx)
}

func (s *AddressService) ListForUser(ctx context.Context, userID int) ([]*ent.Address, error) {
	return s.Client.Address.
		Query().
		Where(address.HasUserWith(user.IDEQ(userID))).
		WithCommune(func(cq *ent.CommuneQuery) {
			cq.WithCity(func(cyq *ent.CityQuery) {
				cyq.WithRegion()
			})
		}).
		Order(ent.Desc(address.FieldCreatedAt)).
		All(ctx)
}
