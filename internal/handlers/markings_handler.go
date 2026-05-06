package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"back/internal/services"
)

type MarkingsHandler struct {
	Svc *services.MarkingsService
}

type updateMarkingRequest struct {
	WorkInAt      *string `json:"work_in_at"`
	WorkOutAt     *string `json:"work_out_at"`
	BreakOutAt    *string `json:"break_out_at"`
	BreakInAt     *string `json:"break_in_at"`
	Justification string  `json:"justification"`
}

func NewMarkingsHandler(svc *services.MarkingsService) *MarkingsHandler {
	return &MarkingsHandler{Svc: svc}
}

func parseOptionalPositiveInt(v string) (*int, error) {
	if v == "" || v == "all" {
		return nil, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return nil, errors.New("invalid int")
	}
	return &n, nil
}

func parseOptionalDate(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *MarkingsHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	branchID, err := parseOptionalPositiveInt(r.URL.Query().Get("branch_id"))
	if err != nil {
		http.Error(w, "branch_id invalido", http.StatusBadRequest)
		return
	}
	accessPointID, err := parseOptionalPositiveInt(r.URL.Query().Get("access_point_id"))
	if err != nil {
		http.Error(w, "access_point_id invalido", http.StatusBadRequest)
		return
	}

	startDate, err := parseOptionalDate(r.URL.Query().Get("start_date"))
	if err != nil {
		http.Error(w, "start_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	endDate, err := parseOptionalDate(r.URL.Query().Get("end_date"))
	if err != nil {
		http.Error(w, "end_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		http.Error(w, "end_date debe ser posterior a start_date", http.StatusBadRequest)
		return
	}

	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		n, convErr := strconv.Atoi(v)
		if convErr != nil || n <= 0 {
			http.Error(w, "page invalido", http.StatusBadRequest)
			return
		}
		page = n
	}

	limit := 5
	if v := r.URL.Query().Get("limit"); v != "" {
		n, convErr := strconv.Atoi(v)
		if convErr != nil || n <= 0 {
			http.Error(w, "limit invalido", http.StatusBadRequest)
			return
		}
		limit = n
	}

	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "today"
	}
	if startDate != nil && endDate != nil {
		rangeValue = "custom"
	}

	resp, err := h.Svc.List(r.Context(), services.MarkingsFilters{
		Range:         rangeValue,
		StartDate:     startDate,
		EndDate:       endDate,
		BranchID:      branchID,
		AccessPointID: accessPointID,
		Search:        r.URL.Query().Get("search"),
		Page:          page,
		Limit:         limit,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidMarkingsRange) {
			http.Error(w, "range invalido", http.StatusBadRequest)
			return
		}
		log.Printf("[markings] error listing markings: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[markings] error encoding list response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *MarkingsHandler) Filters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp, err := h.Svc.Filters(r.Context())
	if err != nil {
		log.Printf("[markings] error loading filters: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[markings] error encoding filters response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *MarkingsHandler) TopBranches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	branchID, err := parseOptionalPositiveInt(r.URL.Query().Get("branch_id"))
	if err != nil {
		http.Error(w, "branch_id invalido", http.StatusBadRequest)
		return
	}
	startDate, err := parseOptionalDate(r.URL.Query().Get("start_date"))
	if err != nil {
		http.Error(w, "start_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	endDate, err := parseOptionalDate(r.URL.Query().Get("end_date"))
	if err != nil {
		http.Error(w, "end_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		http.Error(w, "end_date debe ser posterior a start_date", http.StatusBadRequest)
		return
	}

	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "today"
	}
	if startDate != nil && endDate != nil {
		rangeValue = "custom"
	}

	resp, err := h.Svc.TopBranches(r.Context(), services.MarkingsFilters{
		Range:     rangeValue,
		StartDate: startDate,
		EndDate:   endDate,
		BranchID:  branchID,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidMarkingsRange) {
			http.Error(w, "range invalido", http.StatusBadRequest)
			return
		}
		log.Printf("[markings] error loading top branches: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[markings] error encoding top branches response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *MarkingsHandler) Update(w http.ResponseWriter, r *http.Request, markingID int) {
	if r.Method != http.MethodPatch && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var req updateMarkingRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		// El frontend puede enviar el JSON doblemente stringificado — intentar decodificar como string
		var inner string
		if jsonErr := json.Unmarshal(bodyBytes, &inner); jsonErr != nil {
			log.Printf("[markings.Update] decode error: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if jsonErr := json.Unmarshal([]byte(inner), &req); jsonErr != nil {
			log.Printf("[markings.Update] decode inner error: %v", jsonErr)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}

	item, err := h.Svc.Update(r.Context(), markingID, services.UpdateMarkingInput{
		WorkInAt:      req.WorkInAt,
		WorkOutAt:     req.WorkOutAt,
		BreakOutAt:    req.BreakOutAt,
		BreakInAt:     req.BreakInAt,
		Justification: req.Justification,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidMarkingUpdate):
			http.Error(w, "justification y formato HH:MM son obligatorios para editar", http.StatusBadRequest)
			return
		case errors.Is(err, services.ErrMarkingNotFound):
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		default:
			log.Printf("[markings] error updating marking %d: %v", markingID, err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(item); err != nil {
		log.Printf("[markings] error encoding update response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}
