package services

import (
	"context"
	"errors"
	"strings"

	"back/internal/ent"
	"back/internal/ent/shift"
)

var (
	ErrShiftInvalidInput = errors.New("invalid shift input")
	ErrShiftNameTaken    = errors.New("shift name already exists")
)

type ShiftService struct {
	Client *ent.Client
}

func NewShiftService(client *ent.Client) *ShiftService {
	return &ShiftService{Client: client}
}

type CreateShiftInput struct {
	Name            string
	Description     *string
	StartTime       string
	EndTime         string
	BreakMinutes    int
	CrossesMidnight *bool
	IsActive        *bool
}

type PatchShiftInput struct {
	Name            *string
	Description     *string
	StartTime       *string
	EndTime         *string
	BreakMinutes    *int
	CrossesMidnight *bool
	IsActive        *bool
}

func (s *ShiftService) List(ctx context.Context) ([]*ent.Shift, error) {
	return s.Client.Shift.
		Query().
		Order(ent.Asc(shift.FieldName)).
		All(ctx)
}

func (s *ShiftService) GetByID(ctx context.Context, shiftID int) (*ent.Shift, error) {
	if shiftID <= 0 {
		return nil, ErrShiftInvalidInput
	}
	return s.Client.Shift.Get(ctx, shiftID)
}

func (s *ShiftService) Create(ctx context.Context, in CreateShiftInput) (*ent.Shift, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.StartTime = strings.TrimSpace(in.StartTime)
	in.EndTime = strings.TrimSpace(in.EndTime)

	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		in.Description = &v
	}

	if in.Name == "" || in.StartTime == "" || in.EndTime == "" || in.BreakMinutes < 0 {
		return nil, ErrShiftInvalidInput
	}

	exists, err := s.Client.Shift.Query().
		Where(shift.NameEQ(in.Name)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrShiftNameTaken
	}

	create := s.Client.Shift.Create().
		SetName(in.Name).
		SetStartTime(in.StartTime).
		SetEndTime(in.EndTime).
		SetBreakMinutes(in.BreakMinutes)

	if in.Description != nil && *in.Description != "" {
		create.SetDescription(*in.Description)
	}
	if in.CrossesMidnight != nil {
		create.SetCrossesMidnight(*in.CrossesMidnight)
	}
	if in.IsActive != nil {
		create.SetIsActive(*in.IsActive)
	}

	return create.Save(ctx)
}

func (s *ShiftService) Patch(ctx context.Context, shiftID int, in PatchShiftInput) (*ent.Shift, error) {
	if shiftID <= 0 {
		return nil, ErrShiftInvalidInput
	}
	if in.Name == nil &&
		in.Description == nil &&
		in.StartTime == nil &&
		in.EndTime == nil &&
		in.BreakMinutes == nil &&
		in.CrossesMidnight == nil &&
		in.IsActive == nil {
		return nil, ErrShiftInvalidInput
	}

	row, err := s.Client.Shift.Get(ctx, shiftID)
	if err != nil {
		return nil, err
	}

	upd := row.Update()

	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" {
			return nil, ErrShiftInvalidInput
		}
		exists, err := s.Client.Shift.Query().
			Where(shift.NameEQ(v)).
			Where(shift.IDNEQ(row.ID)).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrShiftNameTaken
		}
		upd.SetName(v)
	}

	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		if v == "" {
			upd.ClearDescription()
		} else {
			upd.SetDescription(v)
		}
	}

	if in.StartTime != nil {
		v := strings.TrimSpace(*in.StartTime)
		if v == "" {
			return nil, ErrShiftInvalidInput
		}
		upd.SetStartTime(v)
	}

	if in.EndTime != nil {
		v := strings.TrimSpace(*in.EndTime)
		if v == "" {
			return nil, ErrShiftInvalidInput
		}
		upd.SetEndTime(v)
	}

	if in.BreakMinutes != nil {
		if *in.BreakMinutes < 0 {
			return nil, ErrShiftInvalidInput
		}
		upd.SetBreakMinutes(*in.BreakMinutes)
	}

	if in.CrossesMidnight != nil {
		upd.SetCrossesMidnight(*in.CrossesMidnight)
	}

	if in.IsActive != nil {
		upd.SetIsActive(*in.IsActive)
	}

	return upd.Save(ctx)
}

func (s *ShiftService) Delete(ctx context.Context, shiftID int) error {
	if shiftID <= 0 {
		return ErrShiftInvalidInput
	}
	return s.Client.Shift.DeleteOneID(shiftID).Exec(ctx)
}
