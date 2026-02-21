package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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

/* =========================
   RESPONSES (Swagger)
   ========================= */

type AddressResponse struct {
	ID        int        `json:"id" example:"1"`
	Street    string     `json:"street" example:"Av. Siempre Viva"`
	Number    string     `json:"number" example:"742"`
	Apartment *string    `json:"apartment,omitempty" example:"12B"`
	Commune   any        `json:"commune,omitempty"` // opcional: depende de tu service
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
// @Param  body  body  createAddressRequest  true  "Crear dirección (solo POST)"
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

// List my addresses
// @Summary     Listar direcciones del usuario
// @Tags        Addresses
// @Produce     json
// @Security    BearerAuth
// @Success     200  {array}   map[string]any
// @Failure     401  {object}  handlers.ErrorResponse
// @Failure     500  {object}  handlers.ErrorResponse
// @Router      /api/v1/addresses [get]
func (h *AddressHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
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

// Create my address
// @Summary     Crear dirección para el usuario autenticado
// @Tags        Addresses
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body  body  createAddressRequest  true  "Dirección"
// @Success     201  {object}  map[string]any
// @Failure     400  {object}  handlers.ErrorResponse
// @Failure     401  {object}  handlers.ErrorResponse
// @Failure     500  {object}  handlers.ErrorResponse
// @Router      /api/v1/addresses [post]
func (h *AddressHandler) CreateMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok || claims.Subject == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
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
