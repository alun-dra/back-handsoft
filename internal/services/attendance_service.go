package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/attendanceday"
	"back/internal/ent/shiftday"
	"back/internal/ent/user"
	"back/internal/ent/userdayoverride"
	"back/internal/ent/usershiftassignment"
)

var (
	ErrAttendanceInvalidInput     = errors.New("invalid attendance input")
	ErrAttendanceAlreadyCompleted = errors.New("attendance already fully recorded")
	ErrAttendanceNotWorkDay       = errors.New("today is not a working day for this user's schedule")
	ErrAttendanceNoShiftAssigned  = errors.New("user has no active shift assigned")
)

type AttendanceService struct {
	Client *ent.Client
	QR     *QRSessionService
}

func NewAttendanceService(client *ent.Client, qr *QRSessionService) *AttendanceService {
	return &AttendanceService{Client: client, QR: qr}
}

func (s *AttendanceService) ValidateAndRecordAttendance(ctx context.Context, tokenPlain string, accessPointID int) (*ent.AttendanceDay, error) {
	if tokenPlain == "" || accessPointID <= 0 {
		return nil, ErrAttendanceInvalidInput
	}

	_, user, err := s.QR.ValidateAndGetQRSession(ctx, tokenPlain)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrAttendanceInvalidInput
	}

	accessPoint, err := s.Client.AccessPoint.Query().Where(accesspoint.IDEQ(accessPointID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAttendanceInvalidInput
		}
		return nil, err
	}

	// Nota: Removiendo validación de asignación a sucursal para permitir
	// que cualquier usuario con QR válido marque asistencia en cualquier access point.
	// El QR ya valida el usuario, y el access_point_id viene del dispositivo logueado.
	// assignedToBranch, err := s.Client.UserBranch.Query().
	//     Where(userbranch.UserIDEQ(user.ID)).
	//     Where(userbranch.BranchIDEQ(accessPoint.BranchID)).
	//     Exist(ctx)
	// if err != nil {
	//     return nil, err
	// }
	// if !assignedToBranch {
	//     return nil, ErrAttendanceUnauthorizedAccessPoint
	// }

	now := time.Now()
	shift, workDate, err := s.resolveShiftAndWorkDate(ctx, user.ID, now)
	if err != nil {
		return nil, err
	}

	attendance, err := s.Client.AttendanceDay.Query().
		Where(attendanceday.UserIDEQ(user.ID)).
		Where(attendanceday.BranchIDEQ(accessPoint.BranchID)).
		Where(attendanceday.WorkDateEQ(workDate)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return s.createAttendance(ctx, shift, user.ID, accessPoint.BranchID, accessPointID, workDate, now)
		}
		return nil, err
	}

	return s.updateAttendance(ctx, shift, attendance, accessPointID, now)
}

// ValidateAndRecordAttendanceByAccessCode valida un código de acceso y registra la asistencia
// Funciona igual que ValidateAndRecordAttendance pero usando access_code en lugar de QR
func (s *AttendanceService) ValidateAndRecordAttendanceByAccessCode(ctx context.Context, accessCode string, accessPointID int) (*ent.AttendanceDay, error) {
	if accessCode == "" || accessPointID <= 0 {
		return nil, ErrAttendanceInvalidInput
	}

	// Buscar el usuario por access_code
	targetUser, err := s.Client.User.Query().
		Where(user.AccessCodeEQ(accessCode)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAttendanceInvalidInput
		}
		return nil, err
	}

	if targetUser == nil || !targetUser.IsActive {
		return nil, ErrAttendanceInvalidInput
	}

	// Validar que el access_point exista
	accessPoint, err := s.Client.AccessPoint.Query().Where(accesspoint.IDEQ(accessPointID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrAttendanceInvalidInput
		}
		return nil, err
	}

	now := time.Now()
	shift, workDate, err := s.resolveShiftAndWorkDate(ctx, targetUser.ID, now)
	if err != nil {
		return nil, err
	}

	// Obtener o crear attendance del ciclo de turno
	attendance, err := s.Client.AttendanceDay.Query().
		Where(attendanceday.UserIDEQ(targetUser.ID)).
		Where(attendanceday.BranchIDEQ(accessPoint.BranchID)).
		Where(attendanceday.WorkDateEQ(workDate)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return s.createAttendance(ctx, shift, targetUser.ID, accessPoint.BranchID, accessPointID, workDate, now)
		}
		return nil, err
	}

	return s.updateAttendance(ctx, shift, attendance, accessPointID, now)
}

// goWeekdayToSchema convierte time.Weekday (0=domingo) al esquema (1=lunes ... 7=domingo)
func goWeekdayToSchema(wd time.Weekday) int {
	if wd == time.Sunday {
		return 7
	}
	return int(wd)
}

// parseShiftTime convierte "HH:MM" en un time.Time usando la fecha base dada
func parseShiftTime(base time.Time, hhmm string) (time.Time, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid shift time %q: %w", hhmm, err)
	}
	return time.Date(base.Year(), base.Month(), base.Day(), t.Hour(), t.Minute(), 0, 0, base.Location()), nil
}

// resolveShiftAndWorkDate determina el turno activo y la fecha del ciclo de turno para el usuario.
// Valida:
//   - Si hay un UserDayOverride con is_day_off=true → error
//   - Si hay override con turno especial → usa ese turno
//   - Si no es un día laborable del turno (shift_days) → error
//   - Para turnos nocturnos (crosses_midnight): si marcamos en la madrugada, workDate = ayer
func (s *AttendanceService) resolveShiftAndWorkDate(ctx context.Context, userID int, now time.Time) (*ent.Shift, time.Time, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 1. Verificar override para hoy (día libre o turno especial)
	override, err := s.Client.UserDayOverride.Query().
		Where(
			userdayoverride.UserIDEQ(userID),
			userdayoverride.DateEQ(today),
		).
		WithShift().
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, time.Time{}, err
	}

	if override != nil && override.IsDayOff {
		return nil, time.Time{}, ErrAttendanceNotWorkDay
	}

	// Si el override tiene un turno especial para hoy, usarlo (no validar shift_days)
	if override != nil && override.ShiftID != nil {
		if overrideShift, shiftErr := override.Edges.ShiftOrErr(); shiftErr == nil && overrideShift != nil {
			return overrideShift, today, nil
		}
	}

	// 2. Buscar asignación de turno activa
	assignment, err := s.Client.UserShiftAssignment.Query().
		Where(
			usershiftassignment.UserIDEQ(userID),
			usershiftassignment.IsActiveEQ(true),
			usershiftassignment.StartDateLTE(today),
		).
		WithShift().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, time.Time{}, ErrAttendanceNoShiftAssigned
		}
		return nil, time.Time{}, err
	}
	if assignment.EndDate != nil && today.After(*assignment.EndDate) {
		return nil, time.Time{}, ErrAttendanceNoShiftAssigned
	}

	shift, err := assignment.Edges.ShiftOrErr()
	if err != nil {
		return nil, time.Time{}, err
	}

	// 3. Calcular work_date: para turnos nocturnos, si estamos en la madrugada
	// antes del end_time y antes del start_time → el turno empezó ayer
	workDate := today
	if shift.CrossesMidnight {
		startT, err1 := parseShiftTime(now, shift.StartTime)
		endT, err2 := parseShiftTime(now, shift.EndTime)
		if err1 == nil && err2 == nil && now.Before(endT) && now.Before(startT) {
			workDate = today.AddDate(0, 0, -1)
		}
	}

	// 4. Validar que el día de la semana de workDate sea laborable según shift_days
	weekday := goWeekdayToSchema(workDate.Weekday())
	isWorkDay, err := s.Client.ShiftDay.Query().
		Where(
			shiftday.ShiftIDEQ(shift.ID),
			shiftday.WeekdayEQ(weekday),
			shiftday.IsWorkingDayEQ(true),
		).
		Exist(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !isWorkDay {
		return nil, time.Time{}, ErrAttendanceNotWorkDay
	}

	return shift, workDate, nil
}

func (s *AttendanceService) createAttendance(ctx context.Context, shift *ent.Shift, userID, branchID, accessPointID int, workDate time.Time, now time.Time) (*ent.AttendanceDay, error) {
	create := s.Client.AttendanceDay.Create().
		SetUserID(userID).
		SetBranchID(branchID).
		SetAccessPointID(accessPointID).
		SetWorkDate(workDate).
		SetWorkInAt(now)

	metrics := computeAttendanceMetrics(workDate, attendanceMetricsSchedule{
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		CrossesMidnight: shift.CrossesMidnight,
		BreakMinutes:    shift.BreakMinutes,
	}, &now, nil, nil, nil)

	if metrics.LateMinutes != nil {
		create.SetLateMinutes(*metrics.LateMinutes)
	}
	if metrics.BreakDiffMinutes != nil {
		create.SetBreakDiffMinutes(*metrics.BreakDiffMinutes)
	}
	if metrics.OvertimeMinutes != nil {
		create.SetOvertimeMinutes(*metrics.OvertimeMinutes)
	}
	if metrics.EarlyExitMinutes != nil {
		create.SetEarlyExitMinutes(*metrics.EarlyExitMinutes)
	}
	if metrics.NetMinutes != nil {
		create.SetNetMinutesBalance(*metrics.NetMinutes)
	}

	return create.Save(ctx)
}

func (s *AttendanceService) updateAttendance(ctx context.Context, shift *ent.Shift, attendance *ent.AttendanceDay, accessPointID int, now time.Time) (*ent.AttendanceDay, error) {
	update := s.Client.AttendanceDay.UpdateOne(attendance)
	if attendance.AccessPointID == nil {
		update.SetAccessPointID(accessPointID)
	}

	workIn := attendance.WorkInAt
	breakOut := attendance.BreakOutAt
	breakIn := attendance.BreakInAt
	workOut := attendance.WorkOutAt

	switch {
	case attendance.WorkInAt == nil:
		update.SetWorkInAt(now)
		workIn = &now
	case attendance.BreakOutAt == nil:
		update.SetBreakOutAt(now)
		breakOut = &now
	case attendance.BreakInAt == nil:
		update.SetBreakInAt(now)
		breakIn = &now
	case attendance.WorkOutAt == nil:
		update.SetWorkOutAt(now)
		workOut = &now
	default:
		return nil, ErrAttendanceAlreadyCompleted
	}

	metrics := computeAttendanceMetrics(attendance.WorkDate, attendanceMetricsSchedule{
		StartTime:       shift.StartTime,
		EndTime:         shift.EndTime,
		CrossesMidnight: shift.CrossesMidnight,
		BreakMinutes:    shift.BreakMinutes,
	}, workIn, breakOut, breakIn, workOut)

	if metrics.LateMinutes != nil {
		update.SetLateMinutes(*metrics.LateMinutes)
	} else {
		update.ClearLateMinutes()
	}
	if metrics.BreakDiffMinutes != nil {
		update.SetBreakDiffMinutes(*metrics.BreakDiffMinutes)
	} else {
		update.ClearBreakDiffMinutes()
	}
	if metrics.OvertimeMinutes != nil {
		update.SetOvertimeMinutes(*metrics.OvertimeMinutes)
	} else {
		update.ClearOvertimeMinutes()
	}
	if metrics.EarlyExitMinutes != nil {
		update.SetEarlyExitMinutes(*metrics.EarlyExitMinutes)
	} else {
		update.ClearEarlyExitMinutes()
	}
	if metrics.NetMinutes != nil {
		update.SetNetMinutesBalance(*metrics.NetMinutes)
	} else {
		update.ClearNetMinutesBalance()
	}

	return update.Save(ctx)
}

