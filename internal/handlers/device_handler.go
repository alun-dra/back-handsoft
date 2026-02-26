package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/services"
)

type DeviceHandler struct {
	Svc *services.DeviceService
}

func NewDeviceHandler(svc *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{Svc: svc}
}

/* =========================
   REQUESTS
   ========================= */

type createDeviceRequest struct {
	Name      string `json:"name" example:"Lector Puerta Principal"`
	Serial    string `json:"serial" example:"SN-ABC-123"`
	Direction string `json:"direction" example:"in"` // "in" | "out"
	IsActive  *bool  `json:"is_active,omitempty" example:"true"`
}

type patchDeviceRequest struct {
	Name      *string `json:"name,omitempty" example:"Lector Puerta Principal"`
	Serial    *string `json:"serial,omitempty" example:"SN-ABC-123"`
	Direction *string `json:"direction,omitempty" example:"out"` // "in" | "out"
	IsActive  *bool   `json:"is_active,omitempty" example:"true"`
}

/* =========================
   ROUTES
   ========================= */

// AccessPointDevices godoc
// @Summary      Dispositivos por entrada
// @Description  GET lista dispositivos de un access point. POST crea dispositivo para un access point (solo admin).
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                 true  "ID del access point"
// @Param        body  body     createDeviceRequest  false "Crear device (solo POST, admin)"
// @Success      200   {array}  DeviceDTO
// @Success      201   {object} DeviceDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/access-points/{id}/devices [get]
// @Router       /api/v1/access-points/{id}/devices [post]
func (h *DeviceHandler) AccessPointDevices(w http.ResponseWriter, r *http.Request) {
	apID, ok := parseAccessPointIDFromDevicesPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.listForAccessPoint(w, r, apID)
	case http.MethodPost:
		h.createForAccessPoint(w, r, apID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// DeviceByID godoc
// @Summary      Editar o eliminar dispositivo
// @Description  PATCH edita parcialmente. DELETE elimina. (solo admin)
// @Tags         Devices
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int               true  "ID del device"
// @Param        body  body     patchDeviceRequest false "Patch device (solo PATCH, admin)"
// @Success      200   {object} DeviceDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/devices/{id} [patch]
// @Router       /api/v1/devices/{id} [delete]
func (h *DeviceHandler) DeviceByID(w http.ResponseWriter, r *http.Request) {
	devID, ok := parseDeviceIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.patch(w, r, devID)
	case http.MethodDelete:
		h.delete(w, r, devID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

/* =========================
   INTERNAL
   ========================= */

func (h *DeviceHandler) listForAccessPoint(w http.ResponseWriter, r *http.Request, apID int) {
	items, err := h.Svc.ListForAccessPoint(r.Context(), apID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]DeviceDTO, 0, len(items))
	for _, d := range items {
		resp = append(resp, DeviceDTO{
			ID:            d.ID,
			AccessPointID: d.AccessPointID,
			Name:          d.Name,
			Serial:        d.Serial,
			Direction:     d.Direction,
			IsActive:      d.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *DeviceHandler) createForAccessPoint(w http.ResponseWriter, r *http.Request, apID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req createDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	d, err := h.Svc.CreateForAccessPoint(r.Context(), apID, services.CreateDeviceInput{
		Name:      req.Name,
		Serial:    req.Serial,
		Direction: req.Direction,
		IsActive:  req.IsActive,
	})
	if err != nil {
		if err == services.ErrDeviceInvalidInput {
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

	resp := DeviceDTO{
		ID:            d.ID,
		AccessPointID: d.AccessPointID,
		Name:          d.Name,
		Serial:        d.Serial,
		Direction:     d.Direction,
		IsActive:      d.IsActive,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *DeviceHandler) patch(w http.ResponseWriter, r *http.Request, devID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req patchDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	d, err := h.Svc.Patch(r.Context(), devID, services.PatchDeviceInput{
		Name:      req.Name,
		Serial:    req.Serial,
		Direction: req.Direction,
		IsActive:  req.IsActive,
	})
	if err != nil {
		if err == services.ErrDeviceInvalidInput {
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

	resp := DeviceDTO{
		ID:            d.ID,
		AccessPointID: d.AccessPointID,
		Name:          d.Name,
		Serial:        d.Serial,
		Direction:     d.Direction,
		IsActive:      d.IsActive,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *DeviceHandler) delete(w http.ResponseWriter, r *http.Request, devID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.Delete(r.Context(), devID); err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		if err == services.ErrDeviceInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/* =========================
   PATH PARSERS (ServeMux)
   ========================= */

// /api/v1/access-points/{id}/devices
func parseAccessPointIDFromDevicesPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "access-points" || parts[4] != "devices" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// /api/v1/devices/{id}
func parseDeviceIDFromPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "devices" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
