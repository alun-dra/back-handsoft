package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/services"
)

type UserBranchHandler struct {
	Svc *services.UserBranchService
}

func NewUserBranchHandler(svc *services.UserBranchService) *UserBranchHandler {
	return &UserBranchHandler{Svc: svc}
}

type assignUserBranchRequest struct {
	BranchID int `json:"branch_id" example:"1"`
}

type UserBranchDTO struct {
	ID       int    `json:"id"`
	UserID   int    `json:"user_id"`
	BranchID int    `json:"branch_id"`
	Name     string `json:"branch_name,omitempty"`
}

func (h *UserBranchHandler) UserBranches(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserIDSubresourcePath(r.URL.Path, "branches")
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.list(w, r, userID)
	case http.MethodPost:
		h.assign(w, r, userID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UserBranchHandler) UserBranchByID(w http.ResponseWriter, r *http.Request) {
	userID, branchID, ok := parseUserBranchDeletePath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.delete(w, r, userID, branchID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *UserBranchHandler) list(w http.ResponseWriter, r *http.Request, userID int) {
	items, err := h.Svc.ListForUser(r.Context(), userID)
	if err != nil {
		if err == services.ErrUserBranchInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]UserBranchDTO, 0, len(items))
	for _, item := range items {
		dto := UserBranchDTO{
			ID:       item.ID,
			UserID:   item.UserID,
			BranchID: item.BranchID,
		}
		if item.Edges.Branch != nil {
			dto.Name = item.Edges.Branch.Name
		}
		resp = append(resp, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UserBranchHandler) assign(w http.ResponseWriter, r *http.Request, userID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req assignUserBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	item, err := h.Svc.Assign(r.Context(), userID, services.AssignUserBranchInput{
		BranchID: req.BranchID,
	})
	if err != nil {
		switch err {
		case services.ErrUserBranchInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case services.ErrUserBranchAlreadyExists:
			http.Error(w, "User already assigned to branch", http.StatusConflict)
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

	resp := UserBranchDTO{
		ID:       item.ID,
		UserID:   item.UserID,
		BranchID: item.BranchID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UserBranchHandler) delete(w http.ResponseWriter, r *http.Request, userID, branchID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.Delete(r.Context(), userID, branchID); err != nil {
		if err == services.ErrUserBranchInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseUserIDSubresourcePath(path, subresource string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" || parts[4] != subresource {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func parseUserBranchDeletePath(path string) (int, int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 6 {
		return 0, 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" || parts[4] != "branches" {
		return 0, 0, false
	}
	userID, err1 := strconv.Atoi(parts[3])
	branchID, err2 := strconv.Atoi(parts[5])
	if err1 != nil || err2 != nil || userID <= 0 || branchID <= 0 {
		return 0, 0, false
	}
	return userID, branchID, true
}
