package handlers

import (
	"encoding/json"
	"net/http"

	"back/internal/services"
)

type ShiftDayHandler struct {
	Svc *services.ShiftDayService
}

func NewShiftDayHandler(svc *services.ShiftDayService) *ShiftDayHandler {
	return &ShiftDayHandler{Svc: svc}
}

type shiftDayInput struct {
	Weekday      int    `json:"weekday" example:"1"`
	IsWorkingDay bool   `json:"is_working_day" example:"true"`
	Mode         string `json:"mode" example:"onsite"`
}

type ShiftDayDTO struct {
	ID           int    `json:"id" example:"1"`
	ShiftID      int    `json:"shift_id" example:"1"`
	Weekday      int    `json:"weekday" example:"1"`
	IsWorkingDay bool   `json:"is_working_day" example:"true"`
	Mode         string `json:"mode" example:"onsite"`
}

// ShiftDays godoc
// @Summary      Días del turno
// @Description  GET lista la configuración semanal del turno. PUT reemplaza toda la configuración semanal del turno.
// @Tags         Shift Days
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int              true  "ID del turno"
// @Param        body  body     []shiftDayInput  false "Reemplazar días del turno"
// @Success      200   {array}  ShiftDayDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/shifts/{id}/days [get]
// @Router       /api/v1/shifts/{id}/days [put]
func (h *ShiftDayHandler) ShiftDays(w http.ResponseWriter, r *http.Request, shiftID int) {
	switch r.Method {

	case http.MethodGet:
		items, err := h.Svc.ListForShift(r.Context(), shiftID)
		if err != nil {
			http.Error(w, "Error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)

	case http.MethodPut:
		var req []shiftDayInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}

		input := make([]services.ShiftDayInput, 0, len(req))
		for _, d := range req {
			input = append(input, services.ShiftDayInput{
				Weekday:      d.Weekday,
				IsWorkingDay: d.IsWorkingDay,
				Mode:         d.Mode,
			})
		}

		if err := h.Svc.ReplaceForShift(r.Context(), shiftID, input); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.WriteHeader(204)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
