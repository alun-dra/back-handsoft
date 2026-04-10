package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/ent/shift"
	"back/internal/ent/shiftinstance"
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

type WorkDayInput struct {
	Weekday      int    `json:"weekday"`
	IsWorkingDay bool   `json:"is_working_day"`
	Mode         string `json:"mode"`
}

type CreateShiftInput struct {
	Name            string
	Description     *string
	StartTime       string
	EndTime         string
	BreakMinutes    int
	CrossesMidnight *bool
	IsActive        *bool
	ScheduleType    string         `json:"schedule_type"` // "monthly", "yearly", etc.
	WorkDays        []WorkDayInput `json:"work_days"`     // [1, 2, 3, 4, 5] (Lunes a Viernes)
	StartDate       time.Time      `json:"start_date"`    // Fecha de inicio del calendariox
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
	// 1. Validaciones previas (Trim y comprobaciones)
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.StartTime == "" || in.EndTime == "" {
		return nil, ErrShiftInvalidInput
	}

	// 2. Iniciar la Transacción
	tx, err := s.Client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("error iniciando transacción: %w", err)
	}

	// 3. Verificar si el nombre ya existe (dentro de la transacción)
	exists, err := tx.Shift.Query().Where(shift.NameEQ(in.Name)).Exist(ctx)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if exists {
		tx.Rollback()
		return nil, ErrShiftNameTaken
	}

	// 4. Crear el Shift (Cabecera)
	builder := tx.Shift.Create().
		SetName(in.Name).
		SetStartTime(in.StartTime).
		SetEndTime(in.EndTime).
		SetBreakMinutes(in.BreakMinutes)

	if in.Description != nil {
		builder.SetDescription(*in.Description)
	}
	if in.CrossesMidnight != nil {
		builder.SetCrossesMidnight(*in.CrossesMidnight)
	}
	if in.IsActive != nil {
		builder.SetIsActive(*in.IsActive)
	}

	newShift, err := builder.Save(ctx)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("error al crear turno: %w", err)
	}

	// 5. Generar Calendario (ShiftInstances)
	// Solo si se especificó un tipo de horario y días de trabajo
	if in.ScheduleType != "" && len(in.WorkDays) > 0 {
		// Si no viene fecha, usamos el tiempo actual
		start := in.StartDate
		if start.IsZero() {
			start = time.Now()
		}

		err = s.GenerateShiftInstances(ctx, tx, newShift.ID, start, in.ScheduleType, in.WorkDays)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("error generando instancias: %w", err)
		}
	}

	// 6. Confirmar todo
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	return newShift, nil
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

func (s *ShiftService) GenerateShiftInstances(
	ctx context.Context,
	tx *ent.Tx,
	shiftID int,
	startDate time.Time,
	scheduleType string,
	workDays []WorkDayInput) error {

	// 1. Normalizar fecha de inicio
	current := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.UTC)

	// 2. Calcular fecha de fin PRIMERO
	var endDate time.Time
	switch scheduleType {
	case "weekly":
		endDate = current.AddDate(0, 0, 7)
	case "monthly":
		endDate = current.AddDate(0, 1, 0)
	case "quarterly":
		endDate = current.AddDate(0, 3, 0)
	case "yearly":
		endDate = current.AddDate(1, 0, 0)
	default:
		return fmt.Errorf("schedule type not supported: %s", scheduleType)
	}

	var bulk []*ent.ShiftInstanceCreate

	// 3. Un solo bucle limpio
	for current.Before(endDate) {
		goWeekday := int(current.Weekday())
		if goWeekday == 0 {
			goWeekday = 7
		} // Normalizar Domingo a 7

		for _, wd := range workDays {
			if wd.Weekday == goWeekday && wd.IsWorkingDay {
				bulk = append(bulk, tx.ShiftInstance.Create().
					SetShiftID(shiftID).
					SetDate(current).
					SetState("scheduled"))
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	if len(bulk) == 0 {
		return nil
	}

	return tx.ShiftInstance.CreateBulk(bulk...).Exec(ctx)
}

func (s *ShiftService) GetCalendar(ctx context.Context, userID int, start, end time.Time) ([]*ent.ShiftInstance, error) {
	return s.Client.ShiftInstance.Query().
		Where(
			shiftinstance.DateGTE(start),
			shiftinstance.DateLTE(end),
		).
		WithShift(). // Importante para sacar el nombre del turno
		All(ctx)
}
