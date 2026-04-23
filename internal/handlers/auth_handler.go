package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"back/internal/config"
	"back/internal/ent"

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

type MeAccessPoint struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type MeBranch struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	Code         *string         `json:"code,omitempty"`
	IsActive     bool            `json:"is_active"`
	RoleInBranch *string         `json:"role_in_branch,omitempty"`
	AccessPoints []MeAccessPoint `json:"access_points"`
}

type MeCurrentShift struct {
	ShiftID       int      `json:"shift_id"`
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	StartTime     string   `json:"start_time"`
	EndTime       string   `json:"end_time"`
	BreakMinutes  int      `json:"break_minutes"`
	CrossesMidnight bool   `json:"crosses_midnight"`
	WorkDays      []string `json:"work_days"`
}

type MeTodaySummary struct {
	WorkDate      string     `json:"work_date"`
	WorkInAt      *time.Time `json:"work_in_at,omitempty"`
	BreakOutAt    *time.Time `json:"break_out_at,omitempty"`
	BreakInAt     *time.Time `json:"break_in_at,omitempty"`
	WorkOutAt     *time.Time `json:"work_out_at,omitempty"`
	MarkingsCount int        `json:"markings_count"`
	WorkedMinutes int        `json:"worked_minutes"`
	WorkedHours   string     `json:"worked_hours"`
}

type MeAttendanceRecord struct {
	ID              int        `json:"id"`
	WorkDate        string     `json:"work_date"`
	BranchID        int        `json:"branch_id"`
	BranchName      string     `json:"branch_name,omitempty"`
	AccessPointID   *int       `json:"access_point_id,omitempty"`
	AccessPointName *string    `json:"access_point_name,omitempty"`
	WorkInAt        *time.Time `json:"work_in_at,omitempty"`
	BreakOutAt      *time.Time `json:"break_out_at,omitempty"`
	BreakInAt       *time.Time `json:"break_in_at,omitempty"`
	WorkOutAt       *time.Time `json:"work_out_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	MarkingsCount   int        `json:"markings_count"`
	WorkedMinutes   int        `json:"worked_minutes"`
	WorkedHours     string     `json:"worked_hours"`
}

type MeResponse struct {
	ID               int                  `json:"id" example:"1"`
	Username         string               `json:"username" example:"admin"`
	Role             string               `json:"role" example:"admin"`
	IsActive         bool                 `json:"is_active" example:"true"`
	Issuer           string               `json:"issuer" example:"dominio-api-development"`
	Audience         []string             `json:"audience" example:"web,ios,android"`
	Expires          any                  `json:"expires"`
	Addresses        []MeAddress          `json:"addresses"`
	Branches         []MeBranch           `json:"branches"`
	CurrentShift     *MeCurrentShift      `json:"current_shift,omitempty"`
	TodaySummary     *MeTodaySummary      `json:"today_summary,omitempty"`
	AttendanceHistory []MeAttendanceRecord `json:"attendance_history"`
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

	u, err := h.Users.GetUserFullData(r.Context(), userID)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	addresses := make([]MeAddress, 0, len(u.Edges.Addresses))
	for _, a := range u.Edges.Addresses {
		var commune *MeCommune
		var city *MeCity
		var region *MeRegion

		if a.Edges.Commune != nil {
			c := a.Edges.Commune
			commune = &MeCommune{ID: c.ID, Name: c.Name}

			if c.Edges.City != nil {
				cy := c.Edges.City
				city = &MeCity{ID: cy.ID, Name: cy.Name}

				if cy.Edges.Region != nil {
					rg := cy.Edges.Region
					region = &MeRegion{ID: rg.ID, Name: rg.Name, Code: rg.Code}
				}
			}
		}

		addresses = append(addresses, MeAddress{
			ID:        a.ID,
			Street:    a.Street,
			Number:    a.Number,
			Apartment: a.Apartment,
			Commune:   commune,
			City:      city,
			Region:    region,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		})
	}

	branches := mapMeBranches(u)
	currentShift := resolveMeCurrentShift(u, time.Now())
	todaySummary := buildMeTodaySummary(u, time.Now())
	history := buildMeAttendanceHistory(u)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(MeResponse{
		ID:                u.ID,
		Username:          u.Username,
		Role:              u.Role,
		IsActive:          u.IsActive,
		Issuer:            claims.Issuer,
		Audience:          []string(claims.Audience),
		Expires:           claims.ExpiresAt,
		Addresses:         addresses,
		Branches:          branches,
		CurrentShift:      currentShift,
		TodaySummary:      todaySummary,
		AttendanceHistory: history,
	})
}

func mapMeBranches(u *ent.User) []MeBranch {
	branches := make([]MeBranch, 0, len(u.Edges.UserBranches))
	for _, ub := range u.Edges.UserBranches {
		if ub.Edges.Branch == nil {
			continue
		}

		accessPoints := make([]MeAccessPoint, 0, len(ub.Edges.Branch.Edges.AccessPoints))
		for _, ap := range ub.Edges.Branch.Edges.AccessPoints {
			accessPoints = append(accessPoints, MeAccessPoint{
				ID:       ap.ID,
				Name:     ap.Name,
				IsActive: ap.IsActive,
			})
		}

		branches = append(branches, MeBranch{
			ID:           ub.Edges.Branch.ID,
			Name:         ub.Edges.Branch.Name,
			Code:         ub.Edges.Branch.Code,
			IsActive:     ub.Edges.Branch.IsActive,
			RoleInBranch: ub.RoleInBranch,
			AccessPoints: accessPoints,
		})
	}
	return branches
}

func resolveMeCurrentShift(u *ent.User, now time.Time) *MeCurrentShift {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, override := range u.Edges.DayOverrides {
		if sameDate(override.Date, today) {
			if override.IsDayOff || override.Edges.Shift == nil {
				return nil
			}
			return mapMeCurrentShift(override.Edges.Shift)
		}
	}

	for _, assignment := range u.Edges.ShiftAssignments {
		if !assignment.IsActive || assignment.Edges.Shift == nil {
			continue
		}
		if assignment.StartDate.After(today) {
			continue
		}
		if assignment.EndDate != nil && assignment.EndDate.Before(today) {
			continue
		}
		return mapMeCurrentShift(assignment.Edges.Shift)
	}

	return nil
}

func mapMeCurrentShift(shift *ent.Shift) *MeCurrentShift {
	if shift == nil {
		return nil
	}

	workDays := make([]string, 0, len(shift.Edges.Days))
	for _, day := range shift.Edges.Days {
		if day.IsWorkingDay {
			workDays = append(workDays, weekdayName(day.Weekday))
		}
	}

	return &MeCurrentShift{
		ShiftID:          shift.ID,
		Name:             shift.Name,
		Description:      shift.Description,
		StartTime:        shift.StartTime,
		EndTime:          shift.EndTime,
		BreakMinutes:     shift.BreakMinutes,
		CrossesMidnight:  shift.CrossesMidnight,
		WorkDays:         workDays,
	}
}

func buildMeTodaySummary(u *ent.User, now time.Time) *MeTodaySummary {
	for _, ad := range u.Edges.AttendanceDays {
		if sameDate(ad.WorkDate, now) {
			workedMinutes := attendanceWorkedMinutes(ad)
			return &MeTodaySummary{
				WorkDate:      ad.WorkDate.Format("2006-01-02"),
				WorkInAt:      ad.WorkInAt,
				BreakOutAt:    ad.BreakOutAt,
				BreakInAt:     ad.BreakInAt,
				WorkOutAt:     ad.WorkOutAt,
				MarkingsCount: attendanceMarkingsCount(ad),
				WorkedMinutes: workedMinutes,
				WorkedHours:   formatWorkedHours(workedMinutes),
			}
		}
	}

	return &MeTodaySummary{WorkDate: now.Format("2006-01-02")}
}

func buildMeAttendanceHistory(u *ent.User) []MeAttendanceRecord {
	history := make([]MeAttendanceRecord, 0, len(u.Edges.AttendanceDays))
	for _, ad := range u.Edges.AttendanceDays {
		workedMinutes := attendanceWorkedMinutes(ad)
		record := MeAttendanceRecord{
			ID:            ad.ID,
			WorkDate:      ad.WorkDate.Format("2006-01-02"),
			BranchID:      ad.BranchID,
			WorkInAt:      ad.WorkInAt,
			BreakOutAt:    ad.BreakOutAt,
			BreakInAt:     ad.BreakInAt,
			WorkOutAt:     ad.WorkOutAt,
			CreatedAt:     ad.CreatedAt,
			UpdatedAt:     ad.UpdatedAt,
			MarkingsCount: attendanceMarkingsCount(ad),
			WorkedMinutes: workedMinutes,
			WorkedHours:   formatWorkedHours(workedMinutes),
		}

		if ad.Edges.Branch != nil {
			record.BranchName = ad.Edges.Branch.Name
		}
		if ad.AccessPointID != nil {
			record.AccessPointID = ad.AccessPointID
		}
		if ad.Edges.AccessPoint != nil {
			name := ad.Edges.AccessPoint.Name
			record.AccessPointName = &name
		}

		history = append(history, record)
	}
	return history
}

func attendanceMarkingsCount(ad *ent.AttendanceDay) int {
	count := 0
	if ad.WorkInAt != nil {
		count++
	}
	if ad.BreakOutAt != nil {
		count++
	}
	if ad.BreakInAt != nil {
		count++
	}
	if ad.WorkOutAt != nil {
		count++
	}
	return count
}

func attendanceWorkedMinutes(ad *ent.AttendanceDay) int {
	if ad.WorkInAt == nil {
		return 0
	}

	end := time.Now()
	if ad.WorkOutAt != nil {
		end = *ad.WorkOutAt
	}

	worked := int(end.Sub(*ad.WorkInAt).Minutes())
	if ad.BreakOutAt != nil && ad.BreakInAt != nil {
		worked -= int(ad.BreakInAt.Sub(*ad.BreakOutAt).Minutes())
	}
	if worked < 0 {
		return 0
	}
	return worked
}

func formatWorkedHours(minutes int) string {
	return strconv.Itoa(minutes/60) + ":" + twoDigitsAuth(minutes%60)
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func weekdayName(n int) string {
	switch n {
	case 1:
		return "Lunes"
	case 2:
		return "Martes"
	case 3:
		return "Miercoles"
	case 4:
		return "Jueves"
	case 5:
		return "Viernes"
	case 6:
		return "Sabado"
	case 7:
		return "Domingo"
	default:
		return ""
	}
}

func twoDigitsAuth(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
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
