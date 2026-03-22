package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"back/internal/services"
)

type UserShiftAssignmentHandler struct {
	Svc *services.UserShiftAssignmentService
}

func NewUserShiftAssignmentHandler(svc *services.UserShiftAssignmentService) *UserShiftAssignmentHandler {
	return &UserShiftAssignmentHandler{Svc: svc}
}

type createAssignmentRequest struct {
	ShiftID   int     `json:"shift_id" example:"1"`
	StartDate string  `json:"start_date" example:"2026-03-01"`
	EndDate   *string `json:"end_date,omitempty" example:"2026-03-31"`
	IsActive  *bool   `json:"is_active,omitempty" example:"true"`
}

type UserShiftAssignmentDTO struct {
	ID        int        `json:"id" example:"1"`
	UserID    int        `json:"user_id" example:"10"`
	ShiftID   int        `json:"shift_id" example:"1"`
	StartDate time.Time  `json:"start_date"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	IsActive  bool       `json:"is_active" example:"true"`
	CreatedAt time.Time  `json:"created_at"`
}

// UserShiftAssignments godoc
// @Summary      Asignaciones de turno del usuario
// @Description  GET lista las asignaciones de turno de un usuario. POST asigna un turno al usuario desde una fecha dada.
// @Tags         User Shift Assignments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                      true  "ID del usuario"
// @Param        body  body     createAssignmentRequest  false "Crear asignación de turno"
// @Success      200   {array}  UserShiftAssignmentDTO
// @Success      201   {object} UserShiftAssignmentDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/{id}/shift-assignments [get]
// @Router       /api/v1/users/{id}/shift-assignments [post]
func (h *UserShiftAssignmentHandler) Assignments(w http.ResponseWriter, r *http.Request, userID int) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Svc.ListForUser(r.Context(), userID)
		if err != nil {
			http.Error(w, "Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var req createAssignmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			http.Error(w, "Invalid start_date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		var end *time.Time
		if req.EndDate != nil && *req.EndDate != "" {
			t, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				http.Error(w, "Invalid end_date format, expected YYYY-MM-DD", http.StatusBadRequest)
				return
			}
			end = &t
		}

		a, err := h.Svc.Create(r.Context(), userID, services.CreateUserShiftAssignmentInput{
			ShiftID:   req.ShiftID,
			StartDate: start,
			EndDate:   end,
			IsActive:  req.IsActive,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(a)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
