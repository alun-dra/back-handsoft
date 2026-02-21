package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/services"
)

/* =========================
   RESPONSES (Swagger)
   ========================= */

type RegionResponse struct {
	ID        int    `json:"id" example:"13"`
	Name      string `json:"name" example:"Metropolitana"`
	Code      string `json:"code" example:"RM"`
	CountryID int    `json:"country_id" example:"1"`
}

type CityResponse struct {
	ID       int    `json:"id" example:"13101"`
	Name     string `json:"name" example:"Santiago"`
	RegionID int    `json:"region_id" example:"13"`
}

type CommuneResponse struct {
	ID     int    `json:"id" example:"13101"`
	Name   string `json:"name" example:"Santiago"`
	CityID int    `json:"city_id" example:"13101"`
}

type LocationHandler struct {
	Loc *services.LocationService
}

func NewLocationHandler(loc *services.LocationService) *LocationHandler {
	return &LocationHandler{Loc: loc}
}

// Regions godoc
// @Summary      Listar regiones
// @Tags         Location
// @Produce      json
// @Success      200  {array}   RegionResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/regions [get]
func (h *LocationHandler) Regions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := h.Loc.ListRegions(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// RegionSubroutes godoc
// @Summary      Listar ciudades por región
// @Tags         Location
// @Produce      json
// @Param        id   path      int  true  "Region ID"
// @Success      200  {array}   CityResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/regions/{id}/cities [get]
func (h *LocationHandler) RegionSubroutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/regions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "cities" {
		http.NotFound(w, r)
		return
	}

	regionID, err := strconv.Atoi(parts[0])
	if err != nil || regionID <= 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	items, err := h.Loc.ListCitiesByRegion(r.Context(), regionID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

// CitySubroutes godoc
// @Summary      Listar comunas por ciudad
// @Tags         Location
// @Produce      json
// @Param        id   path      int  true  "City ID"
// @Success      200  {array}   CommuneResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/cities/{id}/communes [get]
func (h *LocationHandler) CitySubroutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cities/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "communes" {
		http.NotFound(w, r)
		return
	}

	cityID, err := strconv.Atoi(parts[0])
	if err != nil || cityID <= 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	items, err := h.Loc.ListCommunesByCity(r.Context(), cityID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}
