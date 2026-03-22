package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"back/internal/services"
)

type UserDayOverrideHandler struct {
	Svc *services.UserDayOverrideService
}

func NewUserDayOverrideHandler(svc *services.UserDayOverrideService) *UserDayOverrideHandler {
	return &UserDayOverrideHandler{Svc: svc}
}

type createOverrideRequest struct {
	Date     string  `json:"date" example:"2026-03-20"`
	ShiftID  *int    `json:"shift_id,omitempty" example:"2"`
	IsDayOff *bool   `json:"is_day_off,omitempty" example:"false"`
	Mode     string  `json:"mode" example:"remote"`
	Notes    *string `json:"notes,omitempty" example:"Trabajo desde casa"`
}

type UserDayOverrideDTO struct {
	ID        int       `json:"id" example:"1"`
	UserID    int       `json:"user_id" example:"10"`
	ShiftID   *int      `json:"shift_id,omitempty" example:"2"`
	Date      time.Time `json:"date"`
	IsDayOff  bool      `json:"is_day_off" example:"false"`
	Mode      string    `json:"mode" example:"remote"`
	Notes     *string   `json:"notes,omitempty" example:"Trabajo desde casa"`
	CreatedAt time.Time `json:"created_at"`
}

// UserDayOverrides godoc
// @Summary      Overrides diarios del usuario
// @Description  GET lista excepciones diarias del usuario. POST crea un override puntual para una fecha específica.
// @Tags         User Day Overrides
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                   true  "ID del usuario"
// @Param        body  body     createOverrideRequest false "Crear override diario"
// @Success      200   {array}  UserDayOverrideDTO
// @Success      201   {object} UserDayOverrideDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/{id}/day-overrides [get]
// @Router       /api/v1/users/{id}/day-overrides [post]
func (h *UserDayOverrideHandler) Overrides(w http.ResponseWriter, r *http.Request, userID int) {
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
		var req createOverrideRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		date, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		o, err := h.Svc.Create(r.Context(), userID, services.CreateUserDayOverrideInput{
			Date:     date,
			ShiftID:  req.ShiftID,
			IsDayOff: req.IsDayOff,
			Mode:     req.Mode,
			Notes:    req.Notes,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(o)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
