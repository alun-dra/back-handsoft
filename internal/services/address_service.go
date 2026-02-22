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

type UpdateAddressInput struct {
	CommuneID *int
	Street    *string
	Number    *string
	Apartment *string // si viene: actualiza; si viene "" => limpia
}

func (s *AddressService) GetForUser(ctx context.Context, userID, addressID int) (*ent.Address, error) {
	return s.Client.Address.
		Query().
		Where(
			address.IDEQ(addressID),
			address.HasUserWith(user.IDEQ(userID)),
		).
		Only(ctx)
}

func (s *AddressService) UpdateForUser(ctx context.Context, userID, addressID int, in UpdateAddressInput) (*ent.Address, error) {
	// 1) asegurar que existe y pertenece al usuario
	a, err := s.GetForUser(ctx, userID, addressID)
	if err != nil {
		return nil, err // ent.NotFoundError -> handler devuelve 404
	}

	// 2) preparar update con validaciones
	upd := a.Update()

	if in.CommuneID != nil {
		if *in.CommuneID <= 0 {
			return nil, ErrAddressInvalidInput
		}
		upd.SetCommuneID(*in.CommuneID)
	}

	if in.Street != nil {
		st := strings.TrimSpace(*in.Street)
		if st == "" {
			return nil, ErrAddressInvalidInput
		}
		upd.SetStreet(st)
	}

	if in.Number != nil {
		num := strings.TrimSpace(*in.Number)
		if num == "" {
			return nil, ErrAddressInvalidInput
		}
		upd.SetNumber(num)
	}

	// Apartment: si viene nil => no tocar; si viene "" => limpiar
	if in.Apartment != nil {
		ap := strings.TrimSpace(*in.Apartment)
		if ap == "" {
			upd.ClearApartment()
		} else {
			upd.SetApartment(ap)
		}
	}

	return upd.Save(ctx)
}

func (s *AddressService) DeleteForUser(ctx context.Context, userID, addressID int) error {
	n, err := s.Client.Address.
		Delete().
		Where(
			address.IDEQ(addressID),
			address.HasUserWith(user.IDEQ(userID)),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &ent.NotFoundError{} // para que tu handler devuelva 404
	}
	return nil
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
