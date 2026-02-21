package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"back/internal/config"
	"back/internal/ent"
	"back/internal/ent/user"
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
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin123"`
	Role     string `json:"role" example:"admin"`
}

type loginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin123"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"441c5a..."`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"441c5a..."`
}

/* =========================
   RESPONSES (Swagger)
   ========================= */

type RegisterResponse struct {
	ID       int    `json:"id" example:"1"`
	Username string `json:"username" example:"admin"`
	Role     string `json:"role" example:"admin"`
}

type TokenResponse struct {
	AccessToken      string    `json:"access_token" example:"eyJhbGciOi..."`
	TokenType        string    `json:"token_type" example:"Bearer"`
	ExpiresIn        int       `json:"expires_in" example:"3600"`
	ExpiresAt        time.Time `json:"expires_at"`
	RefreshToken     string    `json:"refresh_token" example:"441c5a..."`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	Role             string    `json:"role" example:"admin"`
	Username         string    `json:"username" example:"admin"`
}

type MeRegion struct {
	ID   int    `json:"id" example:"13"`
	Name string `json:"name" example:"Metropolitana"`
	Code string `json:"code" example:"RM"`
}

type MeCity struct {
	ID   int    `json:"id" example:"13101"`
	Name string `json:"name" example:"Santiago"`
}

type MeCommune struct {
	ID   int    `json:"id" example:"13101"`
	Name string `json:"name" example:"Santiago"`
}

type MeAddress struct {
	ID        int        `json:"id" example:"1"`
	Street    string     `json:"street" example:"Av. Siempre Viva"`
	Number    string     `json:"number" example:"742"`
	Apartment *string    `json:"apartment,omitempty" example:"12B"`
	Commune   *MeCommune `json:"commune,omitempty"`
	City      *MeCity    `json:"city,omitempty"`
	Region    *MeRegion  `json:"region,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type MeResponse struct {
	ID        int         `json:"id" example:"1"`
	Username  string      `json:"username" example:"admin"`
	Role      string      `json:"role" example:"admin"`
	IsActive  bool        `json:"is_active" example:"true"`
	Issuer    string      `json:"issuer" example:"dominio-api-development"`
	Audience  []string    `json:"audience" example:"web,ios,android"`
	Expires   any         `json:"expires"` // jwt.NumericDate en runtime
	Addresses []MeAddress `json:"addresses"`
}

type SessionsResponse struct {
	Count    int   `json:"count" example:"2"`
	Sessions []any `json:"sessions"` // si tienes DTO en tu service, lo tipamos
}

/* =========================
   REGISTER (admin-only)
   ========================= */

// Register godoc
// @Summary      Registrar usuario
// @Description  Crea un usuario nuevo. Requiere rol admin.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param  body  body  registerRequest  true  "Datos de registro"
// @Success      201   {object}  RegisterResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/auth/register [post]
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

// Login godoc
// @Summary      Login
// @Description  Retorna access_token y refresh_token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param  body  body  loginRequest  true  "Credenciales"
// @Success      200   {object}  TokenResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/auth/login [post]
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

// Refresh godoc
// @Summary      Refresh token
// @Description  Rota refresh_token y entrega un nuevo access_token + refresh_token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param  body  body  refreshRequest  true  "Refresh token"
// @Success      200   {object}  TokenResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/v1/auth/refresh [post]
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

// Me godoc
// @Summary      Perfil del usuario autenticado
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MeResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
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

	u, err := h.Users.Client.User.
		Query().
		Where(user.IDEQ(userID)).
		WithAddresses(func(aq *ent.AddressQuery) {
			aq.WithCommune(func(cq *ent.CommuneQuery) {
				cq.WithCity(func(cyq *ent.CityQuery) {
					cyq.WithRegion()
				})
			})
		}).
		Only(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addresses := make([]map[string]any, 0, len(u.Edges.Addresses))
	for _, a := range u.Edges.Addresses {
		var commune any = nil
		var city any = nil
		var region any = nil

		if a.Edges.Commune != nil {
			c := a.Edges.Commune
			commune = map[string]any{"id": c.ID, "name": c.Name}

			if c.Edges.City != nil {
				cy := c.Edges.City
				city = map[string]any{"id": cy.ID, "name": cy.Name}

				if cy.Edges.Region != nil {
					rg := cy.Edges.Region
					region = map[string]any{"id": rg.ID, "name": rg.Name, "code": rg.Code}
				}
			}
		}

		addresses = append(addresses, map[string]any{
			"id":         a.ID,
			"street":     a.Street,
			"number":     a.Number,
			"apartment":  a.Apartment,
			"commune":    commune,
			"city":       city,
			"region":     region,
			"created_at": a.CreatedAt,
			"updated_at": a.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":        u.ID,
		"username":  u.Username,
		"role":      u.Role,
		"is_active": u.IsActive,

		"issuer":   claims.Issuer,
		"audience": []string(claims.Audience),
		"expires":  claims.ExpiresAt,

		"addresses": addresses,
	})
}

/* =========================
   LOGOUT (revoke current refresh)
   ========================= */

// Logout godoc
// @Summary      Logout
// @Description  Revoca el refresh_token actual (cierra sesión actual)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param  body  body  logoutRequest  true  "Refresh token a revocar"
// @Success      204   {object}  NoContentResponse
// @Failure      400   {object}  ErrorResponse
// @Router       /api/v1/auth/logout [post]
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

	_ = h.Tokens.Revoke(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

/* =========================
   LOGOUT ALL (revoke all refresh for user)
   ========================= */

// LogoutAll godoc
// @Summary      Logout de todos los dispositivos
// @Description  Revoca todos los refresh_tokens activos del usuario
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      204  {object}  NoContentResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/auth/logout-all [post]
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

// Sessions godoc
// @Summary      Sesiones activas
// @Description  Lista sesiones activas (refresh tokens no revocados) del usuario
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  SessionsResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/auth/sessions [get]
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
