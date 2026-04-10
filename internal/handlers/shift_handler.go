package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/services"
)

type ShiftHandler struct {
	Svc *services.ShiftService
}

func NewShiftHandler(svc *services.ShiftService) *ShiftHandler {
	return &ShiftHandler{Svc: svc}
}

type createShiftRequest struct {
	Name            string                  `json:"name" example:"Turno mañana"`
	Description     *string                 `json:"description,omitempty" example:"Lunes a viernes 08:00 a 17:00"`
	StartTime       string                  `json:"start_time" example:"08:00"`
	EndTime         string                  `json:"end_time" example:"17:00"`
	BreakMinutes    int                     `json:"break_minutes" example:"60"`
	CrossesMidnight *bool                   `json:"crosses_midnight,omitempty" example:"false"`
	IsActive        *bool                   `json:"is_active,omitempty" example:"true"`
	ScheduleType    string                  `json:"schedule_type"`
	WorkDays        []services.WorkDayInput `json:"work_days"`
	StartDate       string                  `json:"start_date"` // Recibimos string "2026-04-10"
}

type patchShiftRequest struct {
	Name            *string `json:"name,omitempty" example:"Turno mañana"`
	Description     *string `json:"description,omitempty" example:"Lunes a viernes 08:00 a 17:00"`
	StartTime       *string `json:"start_time,omitempty" example:"08:00"`
	EndTime         *string `json:"end_time,omitempty" example:"17:00"`
	BreakMinutes    *int    `json:"break_minutes,omitempty" example:"60"`
	CrossesMidnight *bool   `json:"crosses_midnight,omitempty" example:"false"`
	IsActive        *bool   `json:"is_active,omitempty" example:"true"`
}

type ShiftDTO struct {
	ID              int       `json:"id" example:"1"`
	Name            string    `json:"name" example:"Turno mañana"`
	Description     *string   `json:"description,omitempty" example:"Lunes a viernes 08:00 a 17:00"`
	StartTime       string    `json:"start_time" example:"08:00"`
	EndTime         string    `json:"end_time" example:"17:00"`
	BreakMinutes    int       `json:"break_minutes" example:"60"`
	CrossesMidnight bool      `json:"crosses_midnight" example:"false"`
	IsActive        bool      `json:"is_active" example:"true"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Shifts godoc
// @Summary      Turnos
// @Description  GET lista turnos. POST crea un nuevo turno.
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body     createShiftRequest  false "Crear turno"
// @Success      200   {array}  ShiftDTO
// @Success      201   {object} ShiftDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/shifts [get]
// @Router       /api/v1/shifts [post]
func (h *ShiftHandler) Shifts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Svc.List(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		var req createShiftRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", 400)
			return
		}

		parsedDate, _ := time.Parse("2006-01-02", req.StartDate)

		s, err := h.Svc.Create(r.Context(), services.CreateShiftInput{
			Name:            req.Name,
			Description:     req.Description,
			StartTime:       req.StartTime,
			EndTime:         req.EndTime,
			BreakMinutes:    req.BreakMinutes,
			CrossesMidnight: req.CrossesMidnight,
			IsActive:        req.IsActive,
			ScheduleType:    req.ScheduleType, // <--- PASAR ESTO
			WorkDays:        req.WorkDays,     // <--- PASAR ESTO
			StartDate:       parsedDate,       // <--- PASAR ESTO
		})
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		_ = json.NewEncoder(w).Encode(s)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
}

// ShiftByID godoc
// @Summary      Turno por ID
// @Description  GET obtiene un turno. PATCH actualiza parcialmente. DELETE elimina.
// @Tags         Shifts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                true  "ID del turno"
// @Param        body  body     patchShiftRequest  false "Actualizar turno"
// @Success      200   {object} ShiftDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/shifts/{id} [get]
// @Router       /api/v1/shifts/{id} [patch]
// @Router       /api/v1/shifts/{id} [delete]
func (h *ShiftHandler) ShiftByID(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path)

	switch r.Method {
	case http.MethodGet:
		s, err := h.Svc.GetByID(r.Context(), id)
		if err != nil {
			if ent.IsNotFound(err) {
				http.Error(w, "Not Found", 404)
				return
			}
			http.Error(w, "Error", 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)

	case http.MethodPatch:
		var req patchShiftRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		s, err := h.Svc.Patch(r.Context(), id, services.PatchShiftInput{
			Name:            req.Name,
			Description:     req.Description,
			StartTime:       req.StartTime,
			EndTime:         req.EndTime,
			BreakMinutes:    req.BreakMinutes,
			CrossesMidnight: req.CrossesMidnight,
			IsActive:        req.IsActive,
		})
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)

	case http.MethodDelete:
		if err := h.Svc.Delete(r.Context(), id); err != nil {
			http.Error(w, "Error", 500)
			return
		}
		w.WriteHeader(204)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}
}

func parseID(path string) int {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	id, _ := strconv.Atoi(parts[len(parts)-1])
	return id
}

// Calendar godoc
// @Summary      Calendario de instancias
// @Description  Obtiene las instancias de turnos generadas entre dos fechas
// @Tags         Shifts
// @Produce      json
// @Security     BearerAuth
// @Param        start  query     string  true  "Fecha inicio YYYY-MM-DD"
// @Param        end    query     string  true  "Fecha fin YYYY-MM-DD"
// @Success      200    {array}   interface{}
// @Router       /api/v1/calendar [get]
func (h *ShiftHandler) Calendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", 405)
		return
	}

	// 1. Obtener parámetros de la URL
	startQuery := r.URL.Query().Get("start")
	endQuery := r.URL.Query().Get("end")

	if startQuery == "" || endQuery == "" {
		http.Error(w, "start and end dates are required", 400)
		return
	}

	// 2. Parsear fechas (Formato ISO YYYY-MM-DD)
	const layout = "2006-01-02"
	startTime, err := time.Parse(layout, startQuery)
	if err != nil {
		http.Error(w, "invalid start date format", 400)
		return
	}
	endTime, err := time.Parse(layout, endQuery)
	if err != nil {
		http.Error(w, "invalid end date format", 400)
		return
	}

	// 3. Llamar al servicio (Asegúrate de haber creado GetCalendar en el service)
	instances, err := h.Svc.GetCalendar(r.Context(), startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(instances)
}
