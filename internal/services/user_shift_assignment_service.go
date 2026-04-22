package services

import (
	"context"
	"errors"
	"time"

	"back/internal/ent"
	"back/internal/ent/shift"
	"back/internal/ent/user"
	"back/internal/ent/usershiftassignment"
)

var ErrUserShiftAssignmentInvalidInput = errors.New("invalid user shift assignment input")

type UserShiftAssignmentService struct {
	Client *ent.Client
}

func NewUserShiftAssignmentService(client *ent.Client) *UserShiftAssignmentService {
	return &UserShiftAssignmentService{Client: client}
}

type CreateUserShiftAssignmentInput struct {
	ShiftID   int
	StartDate time.Time
	EndDate   *time.Time
	IsActive  *bool
}

func (s *UserShiftAssignmentService) ListForUser(ctx context.Context, userID int) ([]*ent.UserShiftAssignment, error) {
	if userID <= 0 {
		return nil, ErrUserShiftAssignmentInvalidInput
	}

	return s.Client.UserShiftAssignment.
		Query().
		Where(usershiftassignment.UserIDEQ(userID)).
		WithShift().
		Order(ent.Desc(usershiftassignment.FieldStartDate)).
		All(ctx)
}

func (s *UserShiftAssignmentService) Create(ctx context.Context, userID int, in CreateUserShiftAssignmentInput) (*ent.UserShiftAssignment, error) {
	if userID <= 0 || in.ShiftID <= 0 || in.StartDate.IsZero() {
		return nil, ErrUserShiftAssignmentInvalidInput
	}

	if _, err := s.Client.User.Query().Where(user.IDEQ(userID)).Only(ctx); err != nil {
		return nil, err
	}
	if _, err := s.Client.Shift.Query().Where(shift.IDEQ(in.ShiftID)).Only(ctx); err != nil {
		return nil, err
	}

	if _, err := s.Client.Shift.Query().Where(shift.IDEQ(in.ShiftID)).Only(ctx); err != nil {
		return nil, err
	}

	_, err := s.Client.UserShiftAssignment.Update().Where(
		usershiftassignment.UserID(userID),
		usershiftassignment.IsActiveEQ(true),
	).
		SetIsActive(false).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	create := s.Client.UserShiftAssignment.Create().
		SetUserID(userID).
		SetShiftID(in.ShiftID).
		SetStartDate(in.StartDate)

	if in.EndDate != nil {
		create.SetEndDate(*in.EndDate)
	}
	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *UserShiftAssignmentService) Delete(ctx context.Context, assignmentID int) error {
	if assignmentID <= 0 {
		return ErrUserShiftAssignmentInvalidInput
	}
	return s.Client.UserShiftAssignment.DeleteOneID(assignmentID).Exec(ctx)
}
