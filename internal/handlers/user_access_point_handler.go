package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/services"
)

type UserAccessPointHandler struct {
	Svc *services.UserAccessPointService
}

func NewUserAccessPointHandler(svc *services.UserAccessPointService) *UserAccessPointHandler {
	return &UserAccessPointHandler{Svc: svc}
}

type assignUserAccessPointsRequest struct {
	AccessPointIDs []int `json:"access_point_ids"`
}

type UserAccessPointDTO struct {
	ID            int    `json:"id"`
	UserID        int    `json:"user_id"`
	AccessPointID int    `json:"access_point_id"`
	Name          string `json:"access_point_name,omitempty"`
	BranchID      int    `json:"branch_id,omitempty"`
}

func (h *UserAccessPointHandler) UserAccessPoints(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserIDSubresourcePath(r.URL.Path, "access-points")
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.list(w, r, userID)
	case http.MethodPost:
		h.assignMany(w, r, userID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UserAccessPointHandler) UserAccessPointByID(w http.ResponseWriter, r *http.Request) {
	userID, accessPointID, ok := parseUserAccessPointDeletePath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.delete(w, r, userID, accessPointID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UserAccessPointHandler) list(w http.ResponseWriter, r *http.Request, userID int) {
	items, err := h.Svc.ListForUser(r.Context(), userID)
	if err != nil {
		if err == services.ErrUserAccessPointInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]UserAccessPointDTO, 0, len(items))
	for _, item := range items {
		dto := UserAccessPointDTO{
			ID:            item.ID,
			UserID:        item.UserID,
			AccessPointID: item.AccessPointID,
		}
		if item.Edges.AccessPoint != nil {
			dto.Name = item.Edges.AccessPoint.Name
			dto.BranchID = item.Edges.AccessPoint.BranchID
		}
		resp = append(resp, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UserAccessPointHandler) assignMany(w http.ResponseWriter, r *http.Request, userID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req assignUserAccessPointsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err := h.Svc.AssignMany(r.Context(), userID, services.AssignUserAccessPointsInput{
		AccessPointIDs: req.AccessPointIDs,
	})
	if err != nil {
		switch err {
		case services.ErrUserAccessPointInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case services.ErrUserAccessPointBranchMismatch:
			http.Error(w, "User is not assigned to the branch of this access point", http.StatusBadRequest)
			return
		default:
			if ent.IsNotFound(err) {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserAccessPointHandler) delete(w http.ResponseWriter, r *http.Request, userID, accessPointID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.Delete(r.Context(), userID, accessPointID); err != nil {
		if err == services.ErrUserAccessPointInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseUserAccessPointDeletePath(path string) (int, int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 6 {
		return 0, 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "access-points" {
		return 0, 0, false
	}
	userID, err1 := strconv.Atoi(parts[3])
	accessPointID, err2 := strconv.Atoi(parts[5])
	if err1 != nil || err2 != nil || userID <= 0 || accessPointID <= 0 {
		return 0, 0, false
	}
	return userID, accessPointID, true
}
