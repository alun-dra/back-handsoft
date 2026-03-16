package services

import (
	"context"
	"errors"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/user"
	"back/internal/ent/useraccesspoint"
	"back/internal/ent/userbranch"
)

var (
	ErrUserAccessPointInvalidInput   = errors.New("invalid user access point input")
	ErrUserAccessPointBranchMismatch = errors.New("user is not assigned to the branch of this access point")
)

type UserAccessPointService struct {
	Client *ent.Client
}

func NewUserAccessPointService(client *ent.Client) *UserAccessPointService {
	return &UserAccessPointService{Client: client}
}

type AssignUserAccessPointsInput struct {
	AccessPointIDs []int
}

func (s *UserAccessPointService) ListForUser(ctx context.Context, userID int) ([]*ent.UserAccessPoint, error) {
	if userID <= 0 {
		return nil, ErrUserAccessPointInvalidInput
	}

	return s.Client.UserAccessPoint.
		Query().
		Where(useraccesspoint.UserIDEQ(userID)).
		WithAccessPoint().
		All(ctx)
}

func (s *UserAccessPointService) AssignMany(ctx context.Context, userID int, in AssignUserAccessPointsInput) error {
	if userID <= 0 || len(in.AccessPointIDs) == 0 {
		return ErrUserAccessPointInvalidInput
	}

	if _, err := s.Client.User.Query().Where(user.IDEQ(userID)).Only(ctx); err != nil {
		return err
	}

	for _, accessPointID := range in.AccessPointIDs {
		if accessPointID <= 0 {
			return ErrUserAccessPointInvalidInput
		}

		ap, err := s.Client.AccessPoint.Query().
			Where(accesspoint.IDEQ(accessPointID)).
			Only(ctx)
		if err != nil {
			return err
		}

		// validar que el usuario pertenezca a la sucursal del access point
		assignedToBranch, err := s.Client.UserBranch.Query().
			Where(userbranch.UserIDEQ(userID)).
			Where(userbranch.BranchIDEQ(ap.BranchID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if !assignedToBranch {
			return ErrUserAccessPointBranchMismatch
		}

		exists, err := s.Client.UserAccessPoint.Query().
			Where(useraccesspoint.UserIDEQ(userID)).
			Where(useraccesspoint.AccessPointIDEQ(accessPointID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if _, err := s.Client.UserAccessPoint.Create().
			SetUserID(userID).
			SetAccessPointID(accessPointID).
			Save(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *UserAccessPointService) Delete(ctx context.Context, userID, accessPointID int) error {
	if userID <= 0 || accessPointID <= 0 {
		return ErrUserAccessPointInvalidInput
	}

	_, err := s.Client.UserAccessPoint.Delete().
		Where(useraccesspoint.UserIDEQ(userID)).
		Where(useraccesspoint.AccessPointIDEQ(accessPointID)).
		Exec(ctx)
	return err
}
