package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/attendanceday"
	"back/internal/ent/branch"
	"back/internal/ent/branchaddress"
	"back/internal/ent/commune"
	"back/internal/ent/device"
	"back/internal/ent/useraccesspoint"
	"back/internal/ent/userbranch"
)

var (
	ErrBranchInvalidInput = errors.New("invalid branch input")
	ErrForbidden          = errors.New("forbidden")
)

type BranchService struct {
	Client *ent.Client
}

func NewBranchService(client *ent.Client) *BranchService {
	return &BranchService{Client: client}
}

/* =========================
   DTO INPUTS
   ========================= */

type BranchAddressInput struct {
	CommuneID int
	Street    string
	Number    string
	Apartment *string
	Extra     *string
}

type CreateBranchInput struct {
	Name     string
	Code     *string
	IsActive *bool
	Address  BranchAddressInput
	Accesses []string // nombres de accesos iniciales (opcional)
}

type PatchBranchInput struct {
	Name     *string
	Code     *string
	IsActive *bool

	Address *PatchBranchAddressInput
}

type PatchBranchAddressInput struct {
	CommuneID *int
	Street    *string
	Number    *string
	Apartment *string // "" => limpiar
	Extra     *string // "" => limpiar
}

/* =========================
   QUERIES
   ========================= */

// ListSummary: listado para pantalla "Sucursales" (solo nombre + dirección)
func (s *BranchService) ListSummary(ctx context.Context) ([]*ent.Branch, error) {
	return s.Client.Branch.
		Query().
		Where(branch.IsActiveEQ(true)).
		WithAddress(func(aq *ent.BranchAddressQuery) {
			aq.WithCommune(func(cq *ent.CommuneQuery) {
				cq.WithCity(func(cyq *ent.CityQuery) {
					cyq.WithRegion()
				})
			})
		}).
		Order(ent.Asc(branch.FieldName)).
		All(ctx)
}

// GetDetail: detalle completo (branch + address + access_points + devices)
func (s *BranchService) GetDetail(ctx context.Context, branchID int) (*ent.Branch, error) {
	return s.Client.Branch.
		Query().
		Where(branch.IDEQ(branchID)).
		WithAddress(func(aq *ent.BranchAddressQuery) {
			aq.WithCommune(func(cq *ent.CommuneQuery) {
				cq.WithCity(func(cyq *ent.CityQuery) {
					cyq.WithRegion()
				})
			})
		}).
		WithAccessPoints(func(apq *ent.AccessPointQuery) {
			apq.WithDevices()
			apq.Order(ent.Asc(accesspoint.FieldName))
		}).
		Only(ctx)
}

/* =========================
   MUTATIONS
   ========================= */

func (s *BranchService) Create(ctx context.Context, in CreateBranchInput) (*ent.Branch, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrBranchInvalidInput
	}

	// validar address mínima
	in.Address.Street = strings.TrimSpace(in.Address.Street)
	in.Address.Number = strings.TrimSpace(in.Address.Number)
	if in.Address.CommuneID <= 0 || in.Address.Street == "" || in.Address.Number == "" {
		return nil, ErrBranchInvalidInput
	}

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	bCreate := tx.Branch.Create().SetName(name)

	if in.Code != nil && strings.TrimSpace(*in.Code) != "" {
		bCreate.SetCode(strings.TrimSpace(*in.Code))
	}
	if in.IsActive != nil {
		bCreate.SetIsActive(*in.IsActive)
	}

	b, err := bCreate.Save(ctx)
	if err != nil {
		return nil, err
	}

	// 1:1 address
	aCreate := tx.BranchAddress.Create().
		SetBranchID(b.ID).
		SetCommuneID(in.Address.CommuneID).
		SetStreet(in.Address.Street).
		SetNumber(in.Address.Number)

	if in.Address.Apartment != nil && strings.TrimSpace(*in.Address.Apartment) != "" {
		aCreate.SetApartment(strings.TrimSpace(*in.Address.Apartment))
	}
	if in.Address.Extra != nil && strings.TrimSpace(*in.Address.Extra) != "" {
		aCreate.SetExtra(strings.TrimSpace(*in.Address.Extra))
	}

	if _, err := aCreate.Save(ctx); err != nil {
		return nil, err
	}

	// accesos iniciales (opcional)
	for _, n := range in.Accesses {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, err := tx.AccessPoint.Create().
			SetBranchID(b.ID).
			SetName(n).
			SetIsActive(true).
			Save(ctx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetDetail(ctx, b.ID)
}

func (s *BranchService) Patch(ctx context.Context, branchID int, in PatchBranchInput) (*ent.Branch, error) {
	if in.Name == nil && in.Code == nil && in.IsActive == nil && in.Address == nil {
		return nil, ErrBranchInvalidInput
	}

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	b, err := tx.Branch.Get(ctx, branchID)
	if err != nil {
		return nil, err
	}

	upd := b.Update()

	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrBranchInvalidInput
		}
		upd.SetName(n)
	}
	if in.Code != nil {
		c := strings.TrimSpace(*in.Code)
		if c == "" {
			upd.ClearCode()
		} else {
			upd.SetCode(c)
		}
	}
	if in.IsActive != nil {
		upd.SetIsActive(*in.IsActive)
	}

	if _, err := upd.Save(ctx); err != nil {
		return nil, err
	}

	if in.Address != nil {
		addr, err := tx.BranchAddress.Query().
			Where(branchaddress.BranchIDEQ(branchID)).
			Only(ctx)
		if err != nil {
			return nil, err
		}

		au := addr.Update()

		if in.Address.CommuneID != nil {
			if *in.Address.CommuneID <= 0 {
				return nil, ErrBranchInvalidInput
			}
			if _, err := tx.Commune.Query().Where(commune.IDEQ(*in.Address.CommuneID)).Only(ctx); err != nil {
				return nil, ErrBranchInvalidInput
			}
			au.SetCommuneID(*in.Address.CommuneID)
		}
		if in.Address.Street != nil {
			st := strings.TrimSpace(*in.Address.Street)
			if st == "" {
				return nil, ErrBranchInvalidInput
			}
			au.SetStreet(st)
		}
		if in.Address.Number != nil {
			num := strings.TrimSpace(*in.Address.Number)
			if num == "" {
				return nil, ErrBranchInvalidInput
			}
			au.SetNumber(num)
		}
		if in.Address.Apartment != nil {
			ap := strings.TrimSpace(*in.Address.Apartment)
			if ap == "" {
				au.ClearApartment()
			} else {
				au.SetApartment(ap)
			}
		}
		if in.Address.Extra != nil {
			ex := strings.TrimSpace(*in.Address.Extra)
			if ex == "" {
				au.ClearExtra()
			} else {
				au.SetExtra(ex)
			}
		}

		if _, err := au.Save(ctx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetDetail(ctx, branchID)
}

// DeleteCascade: borra sucursal + todo lo dependiente (manual, en tx)
// Ahora incluye devices (porque AccessPoint -> Devices)
func (s *BranchService) DeleteCascade(ctx context.Context, branchID int) error {
	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Branch.Get(ctx, branchID); err != nil {
		return err
	}

	// 1) access points de la sucursal
	aps, err := tx.AccessPoint.Query().
		Where(accesspoint.BranchIDEQ(branchID)).
		All(ctx)
	if err != nil {
		return err
	}
	apIDs := make([]int, 0, len(aps))
	for _, ap := range aps {
		apIDs = append(apIDs, ap.ID)
	}

	// 2) borrar devices asociados a esos access_points
	if len(apIDs) > 0 {
		if _, err := tx.Device.Delete().
			Where(device.AccessPointIDIn(apIDs...)).
			Exec(ctx); err != nil {
			return err
		}
	}

	// 3) borrar user_access_points asociados a esos accesos
	if len(apIDs) > 0 {
		if _, err := tx.UserAccessPoint.Delete().
			Where(useraccesspoint.AccessPointIDIn(apIDs...)).
			Exec(ctx); err != nil {
			return err
		}
	}

	// 4) borrar access_points
	if _, err := tx.AccessPoint.Delete().
		Where(accesspoint.BranchIDEQ(branchID)).
		Exec(ctx); err != nil {
		return err
	}

	// 5) borrar branch_address (1:1)
	if _, err := tx.BranchAddress.Delete().
		Where(branchaddress.BranchIDEQ(branchID)).
		Exec(ctx); err != nil {
		return err
	}

	// 6) borrar user_branches
	if _, err := tx.UserBranch.Delete().
		Where(userbranch.BranchIDEQ(branchID)).
		Exec(ctx); err != nil {
		return err
	}

	// 7) borrar attendance_days (si ya existe uso)
	if _, err := tx.AttendanceDay.Delete().
		Where(attendanceday.BranchIDEQ(branchID)).
		Exec(ctx); err != nil {
		return err
	}

	// 8) borrar branch
	if err := tx.Branch.DeleteOneID(branchID).Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

// helper por si después quieres “fecha de trabajo” a medianoche
func normalizeWorkDate(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
