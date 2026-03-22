package services

import (
	"context"
	"errors"

	"back/internal/ent"
	"back/internal/ent/shift"
	"back/internal/ent/shiftday"
)

var ErrShiftDayInvalidInput = errors.New("invalid shift day input")

type ShiftDayService struct {
	Client *ent.Client
}

func NewShiftDayService(client *ent.Client) *ShiftDayService {
	return &ShiftDayService{Client: client}
}

type ShiftDayInput struct {
	Weekday      int
	IsWorkingDay bool
	Mode         string
}

func (s *ShiftDayService) ListForShift(ctx context.Context, shiftID int) ([]*ent.ShiftDay, error) {
	if shiftID <= 0 {
		return nil, ErrShiftDayInvalidInput
	}

	return s.Client.ShiftDay.
		Query().
		Where(shiftday.ShiftIDEQ(shiftID)).
		Order(ent.Asc(shiftday.FieldWeekday)).
		All(ctx)
}

func (s *ShiftDayService) ReplaceForShift(ctx context.Context, shiftID int, days []ShiftDayInput) error {
	if shiftID <= 0 || len(days) == 0 {
		return ErrShiftDayInvalidInput
	}

	if _, err := s.Client.Shift.Query().Where(shift.IDEQ(shiftID)).Only(ctx); err != nil {
		return err
	}

	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ShiftDay.Delete().
		Where(shiftday.ShiftIDEQ(shiftID)).
		Exec(ctx); err != nil {
		return err
	}

	for _, d := range days {
		if d.Weekday < 1 || d.Weekday > 7 {
			return ErrShiftDayInvalidInput
		}
		if d.Mode == "" {
			return ErrShiftDayInvalidInput
		}

		if _, err := tx.ShiftDay.Create().
			SetShiftID(shiftID).
			SetWeekday(d.Weekday).
			SetIsWorkingDay(d.IsWorkingDay).
			SetMode(d.Mode).
			Save(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}
