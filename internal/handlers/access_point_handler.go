package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/services"
)

type AccessPointHandler struct {
	Svc *services.AccessPointService
}

func NewAccessPointHandler(svc *services.AccessPointService) *AccessPointHandler {
	return &AccessPointHandler{Svc: svc}
}

type createAccessPointRequest struct {
	Name     string `json:"name" example:"Puerta Principal"`
	IsActive *bool  `json:"is_active,omitempty" example:"true"`
}

type patchAccessPointRequest struct {
	Name     *string `json:"name,omitempty" example:"Puerta Principal"`
	IsActive *bool   `json:"is_active,omitempty" example:"true"`
}

type AccessPointDetailDTO struct {
	ID       int         `json:"id"`
	BranchID int         `json:"branch_id"`
	Name     string      `json:"name"`
	IsActive bool        `json:"is_active"`
	Devices  []DeviceDTO `json:"devices,omitempty"`
}

// BranchAccessPoints godoc
// @Summary      Accesos por sucursal
// @Description  GET lista accesos de una sucursal. POST crea un acceso para una sucursal (solo admin).
// @Tags         Access Points
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                       true  "ID de la sucursal"
// @Param        body  body     createAccessPointRequest  false "Crear access point (solo POST, admin)"
// @Success      200   {array}  AccessPointDetailDTO
// @Success      201   {object} AccessPointDetailDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      409   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/branches/{id}/access-points [get]
// @Router       /api/v1/branches/{id}/access-points [post]
func (h *AccessPointHandler) BranchAccessPoints(w http.ResponseWriter, r *http.Request) {
	branchID, ok := parseBranchIDFromAccessPointsPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listForBranch(w, r, branchID)
	case http.MethodPost:
		h.createForBranch(w, r, branchID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// AccessPointByID godoc
// @Summary      Editar o eliminar acceso
// @Description  PATCH edita parcialmente. DELETE elimina. (solo admin)
// @Tags         Access Points
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                      true  "ID del access point"
// @Param        body  body     patchAccessPointRequest  false "Patch access point (solo PATCH, admin)"
// @Success      200   {object} AccessPointDetailDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      409   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/access-points/{id} [patch]
// @Router       /api/v1/access-points/{id} [delete]
func (h *AccessPointHandler) AccessPointByID(w http.ResponseWriter, r *http.Request) {
	accessPointID, ok := parseAccessPointIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.patch(w, r, accessPointID)
	case http.MethodDelete:
		h.delete(w, r, accessPointID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AccessPointHandler) listForBranch(w http.ResponseWriter, r *http.Request, branchID int) {
	items, err := h.Svc.ListForBranch(r.Context(), branchID)
	if err != nil {
		if err == services.ErrAccessPointInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]AccessPointDetailDTO, 0, len(items))
	for _, ap := range items {
		dto := AccessPointDetailDTO{
			ID:       ap.ID,
			BranchID: ap.BranchID,
			Name:     ap.Name,
			IsActive: ap.IsActive,
		}

		for _, d := range ap.Edges.Devices {
			dto.Devices = append(dto.Devices, DeviceDTO{
				ID:            d.ID,
				AccessPointID: d.AccessPointID,
				Name:          d.Name,
				Serial:        d.Serial,
				Direction:     d.Direction,
				Username:      d.Username,
				Role:          d.Role,
				IsActive:      d.IsActive,
			})
		}

		resp = append(resp, dto)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AccessPointHandler) createForBranch(w http.ResponseWriter, r *http.Request, branchID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req createAccessPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ap, err := h.Svc.CreateForBranch(r.Context(), branchID, services.CreateAccessPointInput{
		Name:     req.Name,
		IsActive: req.IsActive,
	})
	if err != nil {
		switch {
		case err == services.ErrAccessPointInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case err == services.ErrAccessPointNameTaken:
			http.Error(w, "Access point name already exists", http.StatusConflict)
			return
		case ent.IsNotFound(err):
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := AccessPointDetailDTO{
		ID:       ap.ID,
		BranchID: ap.BranchID,
		Name:     ap.Name,
		IsActive: ap.IsActive,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AccessPointHandler) patch(w http.ResponseWriter, r *http.Request, accessPointID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req patchAccessPointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ap, err := h.Svc.Patch(r.Context(), accessPointID, services.PatchAccessPointInput{
		Name:     req.Name,
		IsActive: req.IsActive,
	})
	if err != nil {
		switch {
		case err == services.ErrAccessPointInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case err == services.ErrAccessPointNameTaken:
			http.Error(w, "Access point name already exists", http.StatusConflict)
			return
		case ent.IsNotFound(err):
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := AccessPointDetailDTO{
		ID:       ap.ID,
		BranchID: ap.BranchID,
		Name:     ap.Name,
		IsActive: ap.IsActive,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AccessPointHandler) delete(w http.ResponseWriter, r *http.Request, accessPointID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.DeleteCascade(r.Context(), accessPointID); err != nil {
		if err == services.ErrAccessPointInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// /api/v1/branches/{id}/access-points
func parseBranchIDFromAccessPointsPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "branches" || parts[4] != "access-points" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// /api/v1/access-points/{id}
func parseAccessPointIDFromPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "access-points" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
