package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"back/internal/services"
)

type DashboardHandler struct {
	Svc *services.DashboardService
}

func NewDashboardHandler(svc *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{Svc: svc}
}

func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "today"
	}

	var branchID *int
	branchStr := r.URL.Query().Get("branch_id")
	if branchStr != "" && branchStr != "all" {
		id, err := strconv.Atoi(branchStr)
		if err != nil || id <= 0 {
			http.Error(w, "branch_id invalido", http.StatusBadRequest)
			return
		}
		branchID = &id
	}

	var startDate, endDate *time.Time
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")
	if startStr != "" && endStr != "" {
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			http.Error(w, "start_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			http.Error(w, "end_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		if end.Before(start) {
			http.Error(w, "end_date debe ser posterior a start_date", http.StatusBadRequest)
			return
		}
		startDate = &start
		endDate = &end
		rangeValue = "custom" // override range when custom dates provided
	}

	resp, err := h.Svc.GetStats(r.Context(), services.DashboardFilters{
		BranchID:  branchID,
		Range:     rangeValue,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidDashboardRange) {
			http.Error(w, "range invalido", http.StatusBadRequest)
			return
		}

		log.Printf(
			"[dashboard] error getting stats: range=%s branch_id=%v start=%v end=%v err=%v",
			rangeValue,
			branchID,
			startDate,
			endDate,
			err,
		)

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[dashboard] error encoding response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardHandler) Live(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var branchID *int
	branchStr := r.URL.Query().Get("branch_id")
	if branchStr != "" && branchStr != "all" {
		id, err := strconv.Atoi(branchStr)
		if err != nil || id <= 0 {
			http.Error(w, "branch_id invalido", http.StatusBadRequest)
			return
		}
		branchID = &id
	}

	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "today"
	}

	var startDate, endDate *time.Time
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")
	if startStr != "" || endStr != "" {
		if startStr == "" || endStr == "" {
			http.Error(w, "start_date y end_date deben enviarse juntos", http.StatusBadRequest)
			return
		}

		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			http.Error(w, "start_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			http.Error(w, "end_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}

		if end.Before(start) {
			http.Error(w, "end_date debe ser posterior a start_date", http.StatusBadRequest)
			return
		}

		startDate = &start
		endDate = &end
		rangeValue = "custom"
	}

	limit := 5
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		value, err := strconv.Atoi(limitStr)
		if err != nil || value <= 0 {
			http.Error(w, "limit invalido", http.StatusBadRequest)
			return
		}
		limit = value
	}

	resp, err := h.Svc.GetLiveData(r.Context(), services.DashboardLiveFilters{
		BranchID:  branchID,
		Limit:     limit,
		Range:     rangeValue,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		log.Printf("[dashboard] error getting live data: branch_id=%v limit=%d err=%v", branchID, limit, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[dashboard] error encoding live response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardHandler) Punctuality(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var branchID *int
	branchStr := r.URL.Query().Get("branch_id")
	if branchStr != "" && branchStr != "all" {
		id, err := strconv.Atoi(branchStr)
		if err != nil || id <= 0 {
			http.Error(w, "branch_id invalido", http.StatusBadRequest)
			return
		}
		branchID = &id
	}

	resp, err := h.Svc.GetPunctuality(r.Context(), branchID)
	if err != nil {
		log.Printf("[dashboard] error getting punctuality: branch_id=%v err=%v", branchID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[dashboard] error encoding punctuality response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardHandler) Export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rangeValue := r.URL.Query().Get("range")
	if rangeValue == "" {
		rangeValue = "today"
	}

	var branchID *int
	branchStr := r.URL.Query().Get("branch_id")
	if branchStr != "" && branchStr != "all" {
		id, err := strconv.Atoi(branchStr)
		if err != nil || id <= 0 {
			http.Error(w, "branch_id invalido", http.StatusBadRequest)
			return
		}
		branchID = &id
	}

	var startDate, endDate *time.Time
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")
	if startStr != "" && endStr != "" {
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			http.Error(w, "start_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			http.Error(w, "end_date invalido (formato: YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		if end.Before(start) {
			http.Error(w, "end_date debe ser posterior a start_date", http.StatusBadRequest)
			return
		}
		startDate = &start
		endDate = &end
		rangeValue = "custom"
	}

	resp, err := h.Svc.GetExport(r.Context(), services.DashboardFilters{
		BranchID:  branchID,
		Range:     rangeValue,
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidDashboardRange) {
			http.Error(w, "range invalido", http.StatusBadRequest)
			return
		}
		log.Printf("[dashboard] error getting export: range=%s branch_id=%v start=%v end=%v err=%v",
			rangeValue, branchID, startDate, endDate, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[dashboard] error encoding export response: %v", err)
		http.Error(w, "error serializando respuesta", http.StatusInternalServerError)
		return
	}
}