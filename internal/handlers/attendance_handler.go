package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"back/internal/services"
)

type AttendanceHandler struct {
	Svc *services.AttendanceService
}

func NewAttendanceHandler(svc *services.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{Svc: svc}
}

type validateQRRequest struct {
	Token         string `json:"token"`
	AccessPointID int    `json:"access_point_id"`
}

type validateAccessCodeRequest struct {
	AccessCode    string `json:"access_code"`
	AccessPointID int    `json:"access_point_id"`
}

type validateQRResponse struct {
	UserID            int     `json:"user_id"`
	BranchID          int     `json:"branch_id"`
	AccessPointID     *int    `json:"access_point_id,omitempty"`
	WorkDate          string  `json:"work_date"`
	WorkInAt          *string `json:"work_in_at,omitempty"`
	BreakOutAt        *string `json:"break_out_at,omitempty"`
	BreakInAt         *string `json:"break_in_at,omitempty"`
	WorkOutAt         *string `json:"work_out_at,omitempty"`
	LateMinutes       *int    `json:"late_minutes,omitempty"`
	OvertimeMinutes   *int    `json:"overtime_minutes,omitempty"`
	EarlyExitMinutes  *int    `json:"early_exit_minutes,omitempty"`
}

func (h *AttendanceHandler) ValidateQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req validateQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if req.Token == "" || req.AccessPointID <= 0 {
		http.Error(w, "token and access_point_id are required", http.StatusBadRequest)
		return
	}

	attendance, err := h.Svc.ValidateAndRecordAttendance(r.Context(), req.Token, req.AccessPointID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAttendanceInvalidInput),
			errors.Is(err, services.ErrAttendanceAlreadyCompleted),
			errors.Is(err, services.ErrAttendanceNotWorkDay),
			errors.Is(err, services.ErrAttendanceNoShiftAssigned),
			errors.Is(err, services.ErrQRSessionNotFound),
			errors.Is(err, services.ErrQRSessionExpired),
			errors.Is(err, services.ErrQRSessionRevoked):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := validateQRResponse{
		UserID:   attendance.UserID,
		BranchID: attendance.BranchID,
		WorkDate: attendance.WorkDate.Format("2006-01-02"),
	}
	if attendance.AccessPointID != nil {
		resp.AccessPointID = attendance.AccessPointID
	}
	if attendance.WorkInAt != nil {
		v := attendance.WorkInAt.Format(time.RFC3339)
		resp.WorkInAt = &v
	}
	if attendance.BreakOutAt != nil {
		v := attendance.BreakOutAt.Format(time.RFC3339)
		resp.BreakOutAt = &v
	}
	if attendance.BreakInAt != nil {
		v := attendance.BreakInAt.Format(time.RFC3339)
		resp.BreakInAt = &v
	}
	if attendance.WorkOutAt != nil {
		v := attendance.WorkOutAt.Format(time.RFC3339)
		resp.WorkOutAt = &v
	}
	if attendance.LateMinutes != nil {
		resp.LateMinutes = attendance.LateMinutes
	}
	if attendance.OvertimeMinutes != nil {
		resp.OvertimeMinutes = attendance.OvertimeMinutes
	}
	if attendance.EarlyExitMinutes != nil {
		resp.EarlyExitMinutes = attendance.EarlyExitMinutes
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AttendanceHandler) ValidateAccessCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req validateAccessCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if req.AccessCode == "" || req.AccessPointID <= 0 {
		http.Error(w, "access_code and access_point_id are required", http.StatusBadRequest)
		return
	}

	attendance, err := h.Svc.ValidateAndRecordAttendanceByAccessCode(r.Context(), req.AccessCode, req.AccessPointID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAttendanceInvalidInput),
			errors.Is(err, services.ErrAttendanceAlreadyCompleted),
			errors.Is(err, services.ErrAttendanceNotWorkDay),
			errors.Is(err, services.ErrAttendanceNoShiftAssigned):
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := validateQRResponse{
		UserID:   attendance.UserID,
		BranchID: attendance.BranchID,
		WorkDate: attendance.WorkDate.Format("2006-01-02"),
	}
	if attendance.AccessPointID != nil {
		resp.AccessPointID = attendance.AccessPointID
	}
	if attendance.WorkInAt != nil {
		v := attendance.WorkInAt.Format(time.RFC3339)
		resp.WorkInAt = &v
	}
	if attendance.BreakOutAt != nil {
		v := attendance.BreakOutAt.Format(time.RFC3339)
		resp.BreakOutAt = &v
	}
	if attendance.BreakInAt != nil {
		v := attendance.BreakInAt.Format(time.RFC3339)
		resp.BreakInAt = &v
	}
	if attendance.WorkOutAt != nil {
		v := attendance.WorkOutAt.Format(time.RFC3339)
		resp.WorkOutAt = &v
	}
	if attendance.LateMinutes != nil {
		resp.LateMinutes = attendance.LateMinutes
	}
	if attendance.OvertimeMinutes != nil {
		resp.OvertimeMinutes = attendance.OvertimeMinutes
	}
	if attendance.EarlyExitMinutes != nil {
		resp.EarlyExitMinutes = attendance.EarlyExitMinutes
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
