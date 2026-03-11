package services

import (
	"context"
	"errors"
	"strings"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/branch"
	"back/internal/ent/device"
	"back/internal/ent/useraccesspoint"
)

var (
	ErrAccessPointInvalidInput = errors.New("invalid access point input")
	ErrAccessPointNameTaken    = errors.New("access point name already exists")
)

type AccessPointService struct {
	Client *ent.Client
}

func NewAccessPointService(client *ent.Client) *AccessPointService {
	return &AccessPointService{Client: client}
}

type CreateAccessPointInput struct {
	Name     string
	IsActive *bool
}

type PatchAccessPointInput struct {
	Name     *string
	IsActive *bool
}

func (s *AccessPointService) ListForBranch(ctx context.Context, branchID int) ([]*ent.AccessPoint, error) {
	if branchID <= 0 {
		return nil, ErrAccessPointInvalidInput
	}

	return s.Client.AccessPoint.
		Query().
		Where(accesspoint.BranchIDEQ(branchID)).
		WithDevices().
		Order(ent.Asc(accesspoint.FieldName)).
		All(ctx)
}

func (s *AccessPointService) CreateForBranch(ctx context.Context, branchID int, in CreateAccessPointInput) (*ent.AccessPoint, error) {
	name := strings.TrimSpace(in.Name)
	if branchID <= 0 || name == "" {
		return nil, ErrAccessPointInvalidInput
	}

	// validar branch existente
	if _, err := s.Client.Branch.Query().Where(branch.IDEQ(branchID)).Only(ctx); err != nil {
		return nil, err
	}

	// validar nombre único por sucursal
	exists, err := s.Client.AccessPoint.
		Query().
		Where(accesspoint.BranchIDEQ(branchID)).
		Where(accesspoint.NameEQ(name)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAccessPointNameTaken
	}

	create := s.Client.AccessPoint.Create().
		SetBranchID(branchID).
		SetName(name)

	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *AccessPointService) Patch(ctx context.Context, accessPointID int, in PatchAccessPointInput) (*ent.AccessPoint, error) {
	if accessPointID <= 0 {
		return nil, ErrAccessPointInvalidInput
	}
	if in.Name == nil && in.IsActive == nil {
		return nil, ErrAccessPointInvalidInput
	}

	ap, err := s.Client.AccessPoint.Get(ctx, accessPointID)
	if err != nil {
		return nil, err
	}

	upd := ap.Update()

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrAccessPointInvalidInput
		}

		exists, err := s.Client.AccessPoint.
			Query().
			Where(accesspoint.BranchIDEQ(ap.BranchID)).
			Where(accesspoint.NameEQ(name)).
			Where(accesspoint.IDNEQ(ap.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrAccessPointNameTaken
		}

		upd.SetName(name)
	}

	if in.IsActive != nil {
		upd.SetIsActive(*in.IsActive)
	}

	return upd.Save(ctx)
}

func (s *AccessPointService) DeleteCascade(ctx context.Context, accessPointID int) error {
	if accessPointID <= 0 {
		return ErrAccessPointInvalidInput
	}

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.AccessPoint.Get(ctx, accessPointID); err != nil {
		return err
	}

	// borrar devices
	if _, err := tx.Device.Delete().
		Where(device.AccessPointIDEQ(accessPointID)).
		Exec(ctx); err != nil {
		return err
	}

	// borrar user_access_points
	if _, err := tx.UserAccessPoint.Delete().
		Where(useraccesspoint.AccessPointIDEQ(accessPointID)).
		Exec(ctx); err != nil {
		return err
	}

	// borrar access_point
	if err := tx.AccessPoint.DeleteOneID(accessPointID).Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}
