package services

import (
	"context"
	"errors"
	"time"

	"back/internal/ent"
	"back/internal/ent/accesspoint"
	"back/internal/ent/attendanceday"
)

var (
	ErrAttendanceInvalidInput          = errors.New("invalid attendance input")
	ErrAttendanceAlreadyCompleted      = errors.New("attendance already fully recorded")
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
	workDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	attendance, err := s.Client.AttendanceDay.Query().
		Where(attendanceday.UserIDEQ(user.ID)).
		Where(attendanceday.BranchIDEQ(accessPoint.BranchID)).
		Where(attendanceday.WorkDateEQ(workDate)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return s.createAttendance(ctx, user.ID, accessPoint.BranchID, accessPointID, workDate, now)
		}
		return nil, err
	}

	return s.updateAttendance(ctx, attendance, accessPointID, now)
}

func (s *AttendanceService) createAttendance(ctx context.Context, userID, branchID, accessPointID int, workDate time.Time, now time.Time) (*ent.AttendanceDay, error) {
	return s.Client.AttendanceDay.Create().
		SetUserID(userID).
		SetBranchID(branchID).
		SetAccessPointID(accessPointID).
		SetWorkDate(workDate).
		SetWorkInAt(now).
		Save(ctx)
}

func (s *AttendanceService) updateAttendance(ctx context.Context, attendance *ent.AttendanceDay, accessPointID int, now time.Time) (*ent.AttendanceDay, error) {
	update := s.Client.AttendanceDay.UpdateOne(attendance)
	if attendance.AccessPointID == nil {
		update.SetAccessPointID(accessPointID)
	}

	switch {
	case attendance.WorkInAt == nil:
		update.SetWorkInAt(now)
	case attendance.BreakOutAt == nil:
		update.SetBreakOutAt(now)
	case attendance.BreakInAt == nil:
		update.SetBreakInAt(now)
	case attendance.WorkOutAt == nil:
		update.SetWorkOutAt(now)
	default:
		return nil, ErrAttendanceAlreadyCompleted
	}

	return update.Save(ctx)
}
