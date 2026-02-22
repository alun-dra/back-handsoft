package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"back/internal/ent"
	"back/internal/middleware"
	"back/internal/services"
)

type AddressHandler struct {
	Addr *services.AddressService
}

func NewAddressHandler(addr *services.AddressService) *AddressHandler {
	return &AddressHandler{Addr: addr}
}

type createAddressRequest struct {
	CommuneID int     `json:"commune_id" example:"10"`
	Street    string  `json:"street" example:"Av. Siempre Viva"`
	Number    string  `json:"number" example:"742"`
	Apartment *string `json:"apartment,omitempty" example:"12B"`
}

type updateAddressRequest struct {
	CommuneID *int    `json:"commune_id,omitempty" example:"10"`
	Street    *string `json:"street,omitempty" example:"Av. Siempre Viva"`
	Number    *string `json:"number,omitempty" example:"742"`
	Apartment *string `json:"apartment,omitempty" example:"12B"` // "" limpia
}

/* =========================
   RESPONSES (Swagger)
   ========================= */

type AddressResponse struct {
	ID        int        `json:"id" example:"1"`
	Street    string     `json:"street" example:"Av. Siempre Viva"`
	Number    string     `json:"number" example:"742"`
	Apartment *string    `json:"apartment,omitempty" example:"12B"`
	Commune   any        `json:"commune,omitempty"`
	City      any        `json:"city,omitempty"`
	Region    any        `json:"region,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

/* =========================
   ROUTER
   ========================= */

// Addresses godoc
// @Summary      Direcciones del usuario
// @Description  GET lista direcciones del usuario. POST crea una dirección nueva.
// @Tags         Addresses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  createAddressRequest  false  "Crear dirección (solo POST)"
// @Success      200   {array}   AddressResponse
// @Success      201   {object}  AddressResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/addresses [get]
// @Router       /api/v1/addresses [post]
func (h *AddressHandler) Addresses(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.ListMine(w, r)
	case http.MethodPost:
		h.CreateMine(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// ListMine godoc
// @Summary     Listar direcciones del usuario
// @Tags        Addresses
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   AddressResponse
// @Failure     401  {object}  handlers.ErrorResponse
// @Failure     500  {object}  handlers.ErrorResponse
// @Router      /api/v1/addresses [get]
func (h *AddressHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil || userID <= 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	items, err := h.Addr.ListForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// CreateMine godoc
// @Summary     Crear dirección para el usuario autenticado
// @Tags        Addresses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body  createAddressRequest  true  "Dirección"
// @Success     201  {object}  AddressResponse
// @Failure     400  {object}  handlers.ErrorResponse
// @Failure     401  {object}  handlers.ErrorResponse
// @Failure     500  {object}  handlers.ErrorResponse
// @Router      /api/v1/addresses [post]
func (h *AddressHandler) CreateMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil || userID <= 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req createAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	addr, err := h.Addr.CreateForUser(r.Context(), userID, services.CreateAddressInput{
		CommuneID: req.CommuneID,
		Street:    req.Street,
		Number:    req.Number,
		Apartment: req.Apartment,
	})
	if err != nil {
		if err == services.ErrAddressInvalidInput {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(addr)
}

// AddressByID godoc
// @Summary      Actualizar o eliminar dirección del usuario
// @Description  PATCH actualiza campos. DELETE elimina la dirección. Solo si pertenece al usuario autenticado.
// @Tags         Addresses
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int                  true  "ID de la dirección"
// @Param        body  body     updateAddressRequest  false "Actualizar dirección (solo PATCH)"
// @Success      200   {object} AddressResponse
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/addresses/{id} [patch]
// @Router       /api/v1/addresses/{id} [delete]
func (h *AddressHandler) AddressByID(w http.ResponseWriter, r *http.Request) {
	addrID, ok := parseAddressIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPatch:
		h.UpdateMine(w, r, addrID)
	case http.MethodDelete:
		h.DeleteMine(w, r, addrID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AddressHandler) UpdateMine(w http.ResponseWriter, r *http.Request, addrID int) {
	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil || userID <= 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req updateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// evitar PATCH vacío
	if req.CommuneID == nil && req.Street == nil && req.Number == nil && req.Apartment == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	updated, err := h.Addr.UpdateForUser(r.Context(), userID, addrID, services.UpdateAddressInput{
		CommuneID: req.CommuneID,
		Street:    req.Street,
		Number:    req.Number,
		Apartment: req.Apartment,
	})
	if err != nil {
		if err == services.ErrAddressInvalidInput {
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *AddressHandler) DeleteMine(w http.ResponseWriter, r *http.Request, addrID int) {
	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil || userID <= 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.Addr.DeleteForUser(r.Context(), userID, addrID); err != nil {
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

// Espera rutas tipo: /api/v1/addresses/{id}
func parseAddressIDFromPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")

	// ["api","v1","addresses","{id}"]
	if len(parts) != 4 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "addresses" {
		return 0, false
	}

	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
