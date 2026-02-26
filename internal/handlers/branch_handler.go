package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/middleware"
	"back/internal/services"
)

type BranchHandler struct {
	Svc *services.BranchService
}

func NewBranchHandler(svc *services.BranchService) *BranchHandler {
	return &BranchHandler{Svc: svc}
}

/* =========================
   REQUESTS
   ========================= */

type createBranchRequest struct {
	Name     string  `json:"name" example:"Bodega Central"`
	Code     *string `json:"code,omitempty" example:"BOG-001"`
	IsActive *bool   `json:"is_active,omitempty" example:"true"`

	Address struct {
		CommuneID int     `json:"commune_id" example:"10"`
		Street    string  `json:"street" example:"Av. Siempre Viva"`
		Number    string  `json:"number" example:"742"`
		Apartment *string `json:"apartment,omitempty" example:"12B"`
		Extra     *string `json:"extra,omitempty" example:"Portón lateral"`
	} `json:"address"`

	AccessPoints []string `json:"access_points,omitempty"`
}

type patchBranchRequest struct {
	Name     *string `json:"name,omitempty" example:"Bodega Central"`
	Code     *string `json:"code,omitempty" example:"BOG-001"`
	IsActive *bool   `json:"is_active,omitempty" example:"true"`

	Address *struct {
		CommuneID *int    `json:"commune_id,omitempty" example:"10"`
		Street    *string `json:"street,omitempty" example:"Av. Siempre Viva"`
		Number    *string `json:"number,omitempty" example:"742"`
		Apartment *string `json:"apartment,omitempty" example:"12B"`        // "" limpia
		Extra     *string `json:"extra,omitempty" example:"Portón lateral"` // "" limpia
	} `json:"address,omitempty"`
}

/* =========================
   RESPONSES
   ========================= */

type BranchAddressDTO struct {
	Region    any     `json:"region,omitempty"`
	City      any     `json:"city,omitempty"`
	Commune   any     `json:"commune,omitempty"`
	Street    string  `json:"street,omitempty"`
	Number    string  `json:"number,omitempty"`
	Apartment *string `json:"apartment,omitempty"`
	Extra     *string `json:"extra,omitempty"`
}

type DeviceDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Serial    string `json:"serial"`
	Direction string `json:"direction"` // "in" | "out"
	IsActive  bool   `json:"is_active"`
}

type AccessPointDTO struct {
	ID       int         `json:"id"`
	Name     string      `json:"name"`
	IsActive bool        `json:"is_active"`
	Devices  []DeviceDTO `json:"devices,omitempty"`
}

type BranchListItemDTO struct {
	ID      int               `json:"id"`
	Name    string            `json:"name"`
	Address *BranchAddressDTO `json:"address,omitempty"`
}

type BranchDetailDTO struct {
	ID           int               `json:"id"`
	Name         string            `json:"name"`
	Code         *string           `json:"code,omitempty"`
	IsActive     bool              `json:"is_active"`
	Address      *BranchAddressDTO `json:"address,omitempty"`
	AccessPoints []AccessPointDTO  `json:"access_points,omitempty"`
}

/* =========================
   ROUTES
   ========================= */

// Branches godoc
// @Summary      Sucursales
// @Description  GET lista sucursales (nombre + dirección) para usuario autenticado. POST crea sucursal (solo admin).
// @Tags         Branches
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body     createBranchRequest  true  "Crear sucursal (solo POST, admin)"
// @Success      200   {array}  BranchListItemDTO
// @Success      201   {object} BranchDetailDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/branches [get]
// @Router       /api/v1/branches [post]
func (h *BranchHandler) Branches(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// BranchByID godoc
// @Summary      Detalle sucursal
// @Description  GET detalle con accesos y dispositivos. PATCH edita parcial (solo admin). DELETE elimina (solo admin) con cascada.
// @Tags         Branches
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                true  "ID sucursal"
// @Param        body  body     patchBranchRequest false "Patch sucursal (solo PATCH, admin)"
// @Success      200   {object} BranchDetailDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/branches/{id} [get]
// @Router       /api/v1/branches/{id} [patch]
// @Router       /api/v1/branches/{id} [delete]
func (h *BranchHandler) BranchByID(w http.ResponseWriter, r *http.Request) {
	branchID, ok := parseBranchIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.Detail(w, r, branchID)
	case http.MethodPatch:
		h.Patch(w, r, branchID)
	case http.MethodDelete:
		h.Delete(w, r, branchID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

/* =========================
   HANDLERS
   ========================= */

func (h *BranchHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Svc.ListSummary(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]BranchListItemDTO, 0, len(items))
	for _, b := range items {
		resp = append(resp, BranchListItemDTO{
			ID:      b.ID,
			Name:    b.Name,
			Address: mapBranchAddress(b.Edges.Address),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BranchHandler) Detail(w http.ResponseWriter, r *http.Request, branchID int) {
	b, err := h.Svc.GetDetail(r.Context(), branchID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := mapBranchDetail(b)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req createBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	b, err := h.Svc.Create(r.Context(), services.CreateBranchInput{
		Name:     req.Name,
		Code:     req.Code,
		IsActive: req.IsActive,
		Address: services.BranchAddressInput{
			CommuneID: req.Address.CommuneID,
			Street:    req.Address.Street,
			Number:    req.Address.Number,
			Apartment: req.Address.Apartment,
			Extra:     req.Address.Extra,
		},
		Accesses: req.AccessPoints,
	})
	if err != nil {
		if err == services.ErrBranchInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := mapBranchDetail(b)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BranchHandler) Patch(w http.ResponseWriter, r *http.Request, branchID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req patchBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var addr *services.PatchBranchAddressInput
	if req.Address != nil {
		addr = &services.PatchBranchAddressInput{
			CommuneID: req.Address.CommuneID,
			Street:    req.Address.Street,
			Number:    req.Address.Number,
			Apartment: req.Address.Apartment,
			Extra:     req.Address.Extra,
		}
	}

	b, err := h.Svc.Patch(r.Context(), branchID, services.PatchBranchInput{
		Name:     req.Name,
		Code:     req.Code,
		IsActive: req.IsActive,
		Address:  addr,
	})
	if err != nil {
		if err == services.ErrBranchInvalidInput {
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

	resp := mapBranchDetail(b)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request, branchID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.DeleteCascade(r.Context(), branchID); err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/* =========================
   HELPERS
   ========================= */

func parseBranchIDFromPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "branches" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func isAdmin(r *http.Request) bool {
	claims, ok := middleware.GetClaims(r)
	if !ok {
		return false
	}
	return strings.ToLower(claims.Role) == "admin"
}

func mapBranchAddress(a *ent.BranchAddress) *BranchAddressDTO {
	if a == nil {
		return nil
	}

	dto := &BranchAddressDTO{
		Street:    a.Street,
		Number:    a.Number,
		Apartment: a.Apartment,
		Extra:     a.Extra,
	}

	if a.Edges.Commune != nil {
		dto.Commune = map[string]any{
			"id":   a.Edges.Commune.ID,
			"name": a.Edges.Commune.Name,
		}
		if a.Edges.Commune.Edges.City != nil {
			dto.City = map[string]any{
				"id":   a.Edges.Commune.Edges.City.ID,
				"name": a.Edges.Commune.Edges.City.Name,
			}
			if a.Edges.Commune.Edges.City.Edges.Region != nil {
				dto.Region = map[string]any{
					"id":   a.Edges.Commune.Edges.City.Edges.Region.ID,
					"name": a.Edges.Commune.Edges.City.Edges.Region.Name,
				}
			}
		}
	}

	return dto
}

func mapBranchDetail(b *ent.Branch) BranchDetailDTO {
	resp := BranchDetailDTO{
		ID:       b.ID,
		Name:     b.Name,
		Code:     b.Code,
		IsActive: b.IsActive,
		Address:  mapBranchAddress(b.Edges.Address),
	}

	for _, ap := range b.Edges.AccessPoints {
		apDTO := AccessPointDTO{
			ID:       ap.ID,
			Name:     ap.Name,
			IsActive: ap.IsActive,
		}

		for _, d := range ap.Edges.Devices {
			apDTO.Devices = append(apDTO.Devices, DeviceDTO{
				ID:        d.ID,
				Name:      d.Name,
				Serial:    d.Serial,
				Direction: d.Direction,
				IsActive:  d.IsActive,
			})
		}

		resp.AccessPoints = append(resp.AccessPoints, apDTO)
	}

	return resp
}
