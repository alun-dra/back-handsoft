package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"back/internal/config"
	"back/internal/middleware"
	"back/internal/services"
)

type AuthHandler struct {
	Users  *services.UsersService
	Tokens *services.TokenService
	Cfg    *config.Config
}

func NewAuthHandler(cfg *config.Config, users *services.UsersService, tokens *services.TokenService) *AuthHandler {
	return &AuthHandler{Cfg: cfg, Users: users, Tokens: tokens}
}

/* =========================
   REQUESTS
   ========================= */

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

/* =========================
   REGISTER (admin-only)
   ========================= */

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	id, err := h.Users.Register(r.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		switch err {
		case services.ErrInvalidInput:
			http.Error(w, "username and password are required", http.StatusBadRequest)
			return
		case services.ErrUserAlreadyExists:
			http.Error(w, "username already exists", http.StatusConflict)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	role := req.Role
	if role == "" {
		role = "user"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       id,
		"username": req.Username,
		"role":     role,
	})
}

/* =========================
   LOGIN (access + refresh)
   ========================= */

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	u, err := h.Users.VerifyLogin(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	pair, err := h.Tokens.IssueForUser(r.Context(), u)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	expiresIn := int(time.Until(pair.AccessExp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":       pair.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         expiresIn,
		"expires_at":         pair.AccessExp,
		"refresh_token":      pair.RefreshToken,
		"refresh_expires_at": pair.RefreshExp,
		"role":               u.Role,
		"username":           u.Username,
	})
}

/* =========================
   REFRESH (rotation)
   ========================= */

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pair, u, err := h.Tokens.Rotate(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	expiresIn := int(time.Until(pair.AccessExp).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token":       pair.AccessToken,
		"token_type":         "Bearer",
		"expires_in":         expiresIn,
		"expires_at":         pair.AccessExp,
		"refresh_token":      pair.RefreshToken,
		"refresh_expires_at": pair.RefreshExp,
		"role":               u.Role,
		"username":           u.Username,
	})
}

/* =========================
   ME (protected)
   ========================= */

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	claims, ok := middleware.GetClaims(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       claims.Subject,
		"username": claims.Username,
		"role":     claims.Role,
		"issuer":   claims.Issuer,
		"audience": []string(claims.Audience),
		"expires":  claims.ExpiresAt,
	})
}

/* =========================
   LOGOUT (revoke current refresh)
   ========================= */

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Seguridad: respondemos 204 siempre, aunque el token no exista o ya esté revocado
	_ = h.Tokens.Revoke(r.Context(), req.RefreshToken)

	w.WriteHeader(http.StatusNoContent)
}

/* =========================
   LOGOUT ALL (revoke all refresh for user)
   ========================= */

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.Tokens.RevokeAllForUser(r.Context(), userID); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Sessions(w http.ResponseWriter, r *http.Request) {
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

	sessions, err := h.Tokens.ListActiveSessions(r.Context(), userID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"count":    len(sessions),
		"sessions": sessions,
	})
}
