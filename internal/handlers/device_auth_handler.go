package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"back/internal/services"
)

type DeviceAuthHandler struct {
	Svc *services.DeviceAuthService
}

func NewDeviceAuthHandler(svc *services.DeviceAuthService) *DeviceAuthHandler {
	return &DeviceAuthHandler{Svc: svc}
}

type deviceLoginRequest struct {
	Username string `json:"username" example:"device_entrada_1"`
	Password string `json:"password" example:"123456"`
}

type DeviceTokenResponse struct {
	AccessToken   string    `json:"access_token" example:"eyJhbGciOi..."`
	TokenType     string    `json:"token_type" example:"Bearer"`
	ExpiresIn     int       `json:"expires_in" example:"3600"`
	ExpiresAt     time.Time `json:"expires_at"`
	Role          string    `json:"role" example:"device"`
	Username      string    `json:"username" example:"device_entrada_1"`
	DeviceID      int       `json:"device_id" example:"1"`
	AccessPointID int       `json:"access_point_id" example:"1"`
	Direction     string    `json:"direction" example:"in"`
	Name          string    `json:"name" example:"Lector Entrada Principal"`
}

// Login godoc
// @Summary      Login dispositivo
// @Description  Login para dispositivos (reloj / lector QR)
// @Tags         Device Auth
// @Accept       json
// @Produce      json
// @Param        body  body      deviceLoginRequest  true  "Credenciales del dispositivo"
// @Success      200   {object}  DeviceTokenResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/device-auth/login [post]
func (h *DeviceAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req deviceLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if err == services.ErrInvalidDeviceCredentials {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
