package services

import (
	"context"
	"errors"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/branch"
	"back/internal/ent/user"
	"back/internal/ent/useraccesspoint"
	"back/internal/ent/userbranch"
)

var (
	ErrUserBranchInvalidInput  = errors.New("invalid user branch input")
	ErrUserBranchAlreadyExists = errors.New("user already assigned to branch")
)

type UserBranchService struct {
	Client *ent.Client
}

func NewUserBranchService(client *ent.Client) *UserBranchService {
	return &UserBranchService{Client: client}
}

type AssignUserBranchInput struct {
	BranchID int
}

func (s *UserBranchService) ListForUser(ctx context.Context, userID int) ([]*ent.UserBranch, error) {
	if userID <= 0 {
		return nil, ErrUserBranchInvalidInput
	}

	return s.Client.UserBranch.
		Query().
		Where(userbranch.UserIDEQ(userID)).
		WithBranch().
		All(ctx)
}

func (s *UserBranchService) Assign(ctx context.Context, userID int, in AssignUserBranchInput) (*ent.UserBranch, error) {
	if userID <= 0 || in.BranchID <= 0 {
		return nil, ErrUserBranchInvalidInput
	}

	if _, err := s.Client.User.Query().Where(user.IDEQ(userID)).Only(ctx); err != nil {
		return nil, err
	}

	if _, err := s.Client.Branch.Query().Where(branch.IDEQ(in.BranchID)).Only(ctx); err != nil {
		return nil, err
	}

	exists, err := s.Client.UserBranch.
		Query().
		Where(userbranch.UserIDEQ(userID)).
		Where(userbranch.BranchIDEQ(in.BranchID)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserBranchAlreadyExists
	}

	return s.Client.UserBranch.
		Create().
		SetUserID(userID).
		SetBranchID(in.BranchID).
		Save(ctx)
}

func (s *UserBranchService) Delete(ctx context.Context, userID, branchID int) error {
	if userID <= 0 || branchID <= 0 {
		return ErrUserBranchInvalidInput
	}

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

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

	// borrar permisos a access points de esa sucursal para ese usuario
	if len(apIDs) > 0 {
		if _, err := tx.UserAccessPoint.Delete().
			Where(useraccesspoint.UserIDEQ(userID)).
			Where(useraccesspoint.AccessPointIDIn(apIDs...)).
			Exec(ctx); err != nil {
			return err
		}
	}

	// borrar asignación a sucursal
	if _, err := tx.UserBranch.Delete().
		Where(userbranch.UserIDEQ(userID)).
		Where(userbranch.BranchIDEQ(branchID)).
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}
