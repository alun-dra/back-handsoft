package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/ent/shift"
	"back/internal/ent/user"
	"back/internal/ent/userdayoverride"
)

var ErrUserDayOverrideInvalidInput = errors.New("invalid user day override input")

type UserDayOverrideService struct {
	Client *ent.Client
}

func NewUserDayOverrideService(client *ent.Client) *UserDayOverrideService {
	return &UserDayOverrideService{Client: client}
}

type CreateUserDayOverrideInput struct {
	Date     time.Time
	ShiftID  *int
	IsDayOff *bool
	Mode     string
	Notes    *string
}

type PatchUserDayOverrideInput struct {
	ShiftID  *int
	IsDayOff *bool
	Mode     *string
	Notes    *string
}

func (s *UserDayOverrideService) ListForUser(ctx context.Context, userID int) ([]*ent.UserDayOverride, error) {
	if userID <= 0 {
		return nil, ErrUserDayOverrideInvalidInput
	}

	return s.Client.UserDayOverride.
		Query().
		Where(userdayoverride.UserIDEQ(userID)).
		WithShift().
		Order(ent.Desc(userdayoverride.FieldDate)).
		All(ctx)
}

func (s *UserDayOverrideService) Create(ctx context.Context, userID int, in CreateUserDayOverrideInput) (*ent.UserDayOverride, error) {
	if userID <= 0 || in.Date.IsZero() {
		return nil, ErrUserDayOverrideInvalidInput
	}

	in.Mode = strings.TrimSpace(in.Mode)
	if in.Mode == "" {
		in.Mode = "onsite"
	}

	if _, err := s.Client.User.Query().Where(user.IDEQ(userID)).Only(ctx); err != nil {
		return nil, err
	}
	if in.ShiftID != nil {
		if _, err := s.Client.Shift.Query().Where(shift.IDEQ(*in.ShiftID)).Only(ctx); err != nil {
			return nil, err
		}
	}

	create := s.Client.UserDayOverride.Create().
		SetUserID(userID).
		SetDate(in.Date).
		SetMode(in.Mode)

	if in.ShiftID != nil {
		create.SetShiftID(*in.ShiftID)
	}
	if in.IsDayOff != nil {
		create.SetIsDayOff(*in.IsDayOff)
	}
	if in.Notes != nil {
		v := strings.TrimSpace(*in.Notes)
		if v != "" {
			create.SetNotes(v)
		}
	}

	return create.Save(ctx)
}

func (s *UserDayOverrideService) Patch(ctx context.Context, overrideID int, in PatchUserDayOverrideInput) (*ent.UserDayOverride, error) {
	if overrideID <= 0 {
		return nil, ErrUserDayOverrideInvalidInput
	}
	if in.ShiftID == nil && in.IsDayOff == nil && in.Mode == nil && in.Notes == nil {
		return nil, ErrUserDayOverrideInvalidInput
	}

	row, err := s.Client.UserDayOverride.Get(ctx, overrideID)
	if err != nil {
		return nil, err
	}

	upd := row.Update()

	if in.ShiftID != nil {
		if *in.ShiftID <= 0 {
			upd.ClearShiftID()
		} else {
			if _, err := s.Client.Shift.Query().Where(shift.IDEQ(*in.ShiftID)).Only(ctx); err != nil {
				return nil, err
			}
			upd.SetShiftID(*in.ShiftID)
		}
	}

	if in.IsDayOff != nil {
		upd.SetIsDayOff(*in.IsDayOff)
	}

	if in.Mode != nil {
		v := strings.TrimSpace(*in.Mode)
		if v == "" {
			return nil, ErrUserDayOverrideInvalidInput
		}
		upd.SetMode(v)
	}

	if in.Notes != nil {
		v := strings.TrimSpace(*in.Notes)
		if v == "" {
			upd.ClearNotes()
		} else {
			upd.SetNotes(v)
		}
	}

	return upd.Save(ctx)
}

func (s *UserDayOverrideService) Delete(ctx context.Context, overrideID int) error {
	if overrideID <= 0 {
		return ErrUserDayOverrideInvalidInput
	}
	return s.Client.UserDayOverride.DeleteOneID(overrideID).Exec(ctx)
}
