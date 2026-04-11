package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"back/internal/ent"
	"back/internal/services"
)

type UsersHandler struct {
	Svc *services.UsersService
}

func NewUsersHandler(svc *services.UsersService) *UsersHandler {
	return &UsersHandler{Svc: svc}
}

/* =========================
   REQUESTS
   ========================= */

type createUserRequest struct {
	Username     string  `json:"username" example:"juan.perez"`
	Password     string  `json:"password" example:"123456"`
	Role         string  `json:"role,omitempty" example:"user"`
	FirstName    string  `json:"first_name" example:"Juan"`
	LastName     string  `json:"last_name" example:"Perez"`
	MiddleName   *string `json:"middle_name,omitempty" example:"Andres"`
	Email        string  `json:"email" example:"juan.perez@empresa.cl"`
	EmployeeCode *string `json:"employee_code,omitempty" example:"EMP-001"`
	AccessCode   string  `json:"access_code" example:"ACC-001"`
	IsActive     *bool   `json:"is_active,omitempty" example:"true"`
}

type patchUserRequest struct {
	Username     *string `json:"username,omitempty" example:"juan.perez"`
	Password     *string `json:"password,omitempty" example:"123456"`
	Role         *string `json:"role,omitempty" example:"user"`
	FirstName    *string `json:"first_name,omitempty" example:"Juan"`
	LastName     *string `json:"last_name,omitempty" example:"Perez"`
	MiddleName   *string `json:"middle_name,omitempty" example:"Andres"`
	Email        *string `json:"email,omitempty" example:"juan.perez@empresa.cl"`
	EmployeeCode *string `json:"employee_code,omitempty" example:"EMP-001"`
	AccessCode   *string `json:"access_code,omitempty" example:"ACC-001"`
	IsActive     *bool   `json:"is_active,omitempty" example:"true"`
}

/* =========================
   RESPONSES
   ========================= */

type UserDTO struct {
	ID           int     `json:"id"`
	Username     string  `json:"username"`
	Role         string  `json:"role"`
	IsActive     bool    `json:"is_active"`
	FirstName    *string `json:"first_name,omitempty"`
	LastName     *string `json:"last_name,omitempty"`
	MiddleName   *string `json:"middle_name,omitempty"`
	Email        *string `json:"email,omitempty"`
	EmployeeCode *string `json:"employee_code,omitempty"`
	AccessCode   *string `json:"access_code,omitempty"`
}

type BranchOverviewDTO struct {
	ID   int    `json:"id" example:"10"`
	Name string `json:"name" example:"Bodega Central"`
}

type ShiftOverviewDTO struct {
	ShiftID   int    `json:"shift_id" example:"3"`
	ShiftName string `json:"shift_name" example:"Turno Mañana"`
	StartTime string `json:"start_time" example:"08:00"`
	EndTime   string `json:"end_time" example:"17:00"`
	StartDate string `json:"start_date" example:"2026-04-01"`
	EndDate   *string `json:"end_date,omitempty" example:"2026-04-30"`
}

type UserOverviewDTO struct {
	ID        int                   `json:"id" example:"1"`
	Name      string                `json:"name" example:"Juan Perez"`
	Email     *string               `json:"email,omitempty" example:"juan@empresa.cl"`
	Role      string                `json:"role" example:"user"`
	IsActive  bool                  `json:"is_active" example:"true"`
	Branches  []BranchOverviewDTO   `json:"branches,omitempty"`
	CurrentShift *ShiftOverviewDTO  `json:"current_shift,omitempty"`
}

/* ========================
   EXCEL EXPORT DTOs
   ======================== */

type AddressDTO struct {
	ID         int     `json:"id"`
	Street     string  `json:"street"`
	Number     string  `json:"number"`
	Apartment  *string `json:"apartment,omitempty"`
	CommuneName string `json:"commune_name"`
	CityName   string  `json:"city_name"`
	RegionName string  `json:"region_name"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type BranchDTO struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Code     *string `json:"code,omitempty"`
	IsActive bool    `json:"is_active"`
}

type UserBranchExcelDTO struct {
	ID           int        `json:"id"`
	Branch       BranchDTO  `json:"branch"`
	RoleInBranch *string    `json:"role_in_branch,omitempty"`
	IsActive     bool       `json:"is_active"`
}

type UserAccessPointExcelDTO struct {
	ID           int    `json:"id"`
	AccessPointID int   `json:"access_point_id"`
	AccessPointName string `json:"access_point_name"`
	IsActive     bool    `json:"is_active"`
	AssignedAt   string  `json:"assigned_at"`
	RevokedAt    *string `json:"revoked_at,omitempty"`
}

type UserShiftAssignmentExcelDTO struct {
	ID        int     `json:"id"`
	ShiftID   int     `json:"shift_id"`
	ShiftName string  `json:"shift_name"`
	StartTime string  `json:"shift_start_time"`
	EndTime   string  `json:"shift_end_time"`
	StartDate string  `json:"start_date"`
	EndDate   *string `json:"end_date,omitempty"`
	IsActive  bool    `json:"is_active"`
	CreatedAt string  `json:"created_at"`
}

type UserDayOverrideExcelDTO struct {
	ID        int     `json:"id"`
	Date      string  `json:"date"`
	IsDayOff  bool    `json:"is_day_off"`
	Mode      string  `json:"mode"`
	ShiftID   *int    `json:"shift_id,omitempty"`
	ShiftName *string `json:"shift_name,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type AttendanceDayDTO struct {
	ID             int     `json:"id"`
	WorkDate       string  `json:"work_date"`
	WorkInAt       *string `json:"work_in_at,omitempty"`
	BreakOutAt     *string `json:"break_out_at,omitempty"`
	BreakInAt      *string `json:"break_in_at,omitempty"`
	WorkOutAt      *string `json:"work_out_at,omitempty"`
	BranchID       int     `json:"branch_id"`
	AccessPointID  *int    `json:"access_point_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type RefreshTokenDTO struct {
	ID        int     `json:"id"`
	CreatedAt string  `json:"created_at"`
	ExpiresAt string  `json:"expires_at"`
	RevokedAt *string `json:"revoked_at,omitempty"`
}

type UserQRSessionDTO struct {
	ID        int     `json:"id"`
	IssuedAt  string  `json:"issued_at"`
	ExpiresAt string  `json:"expires_at"`
	IsRevoked bool    `json:"is_revoked"`
	CreatedAt string  `json:"created_at"`
}

type UserExcelExportDTO struct {
	ID                int                            `json:"id"`
	Username          string                         `json:"username"`
	FirstName         *string                        `json:"first_name,omitempty"`
	LastName          *string                        `json:"last_name,omitempty"`
	MiddleName        *string                        `json:"middle_name,omitempty"`
	Email             *string                        `json:"email,omitempty"`
	Role              string                         `json:"role"`
	EmployeeCode      *string                        `json:"employee_code,omitempty"`
	AccessCode        *string                        `json:"access_code,omitempty"`
	IsActive          bool                           `json:"is_active"`
	CreatedAt         string                         `json:"created_at"`
	UpdatedAt         string                         `json:"updated_at"`

	Addresses         []AddressDTO                   `json:"addresses,omitempty"`
	UserBranches      []UserBranchExcelDTO           `json:"user_branches,omitempty"`
	UserAccessPoints  []UserAccessPointExcelDTO      `json:"user_access_points,omitempty"`
	ShiftAssignments  []UserShiftAssignmentExcelDTO  `json:"shift_assignments,omitempty"`
	DayOverrides      []UserDayOverrideExcelDTO      `json:"day_overrides,omitempty"`
	AttendanceDays    []AttendanceDayDTO             `json:"attendance_days,omitempty"`
	RefreshTokens     []RefreshTokenDTO              `json:"refresh_tokens,omitempty"`
	QRSessions        []UserQRSessionDTO             `json:"qr_sessions,omitempty"`
}

/* =========================
   ROUTES
   ========================= */

// Users godoc
// @Summary      Usuarios
// @Description  GET lista usuarios. POST crea usuario. (solo admin)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body     createUserRequest  false "Crear usuario (solo POST, admin)"
// @Success      200   {array}  UserDTO
// @Success      201   {object} UserDTO
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      409   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users [get]
// @Router       /api/v1/users [post]
func (h *UsersHandler) Users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// UserByID godoc
// @Summary      Usuario por ID
// @Description  GET detalle de usuario. PATCH edita usuario. DELETE elimina usuario. (solo admin para PATCH/DELETE)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int               true  "ID del usuario"
// @Param        body  body     patchUserRequest  false "Patch usuario (solo PATCH, admin)"
// @Success      200   {object} UserDTO
// @Success      204   "No Content"
// @Failure      400   {object} ErrorResponse
// @Failure      401   {object} ErrorResponse
// @Failure      403   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      409   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/{id} [get]
// @Router       /api/v1/users/{id} [patch]
// @Router       /api/v1/users/{id} [delete]
func (h *UsersHandler) UserByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseUserIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getByID(w, r, userID)
	case http.MethodPatch:
		h.patch(w, r, userID)
	case http.MethodDelete:
		h.delete(w, r, userID)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// UserOverview godoc
// @Summary      Vista general del usuario
// @Description  GET retorna información simplificada para tabla (nombre, email, role, sucursales, turno actual)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int  true  "ID del usuario"
// @Success      200   {object} UserOverviewDTO
// @Failure      401   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/{id}/overview [get]
func (h *UsersHandler) UserOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := parseUserIDFromPath(r.URL.Path)
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	u, err := h.Svc.GetOverviewData(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := mapUserOverviewDTO(u)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UsersOverview godoc
// @Summary      Lista de usuarios - Vista general
// @Description  GET retorna todos los usuarios con información simplificada (nombre, email, role, sucursales, turno actual)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200   {array}  UserOverviewDTO
// @Failure      401   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/overview [get]
func (h *UsersHandler) UsersOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	users, err := h.Svc.ListOverview(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]UserOverviewDTO, len(users))
	for i, u := range users {
		resp[i] = mapUserOverviewDTO(u)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UsersHandler) create(w http.ResponseWriter, r *http.Request) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	u, err := h.Svc.Create(r.Context(), services.CreateUserInput{
		Username:     req.Username,
		Password:     req.Password,
		Role:         req.Role,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		MiddleName:   req.MiddleName,
		Email:        req.Email,
		EmployeeCode: req.EmployeeCode,
		AccessCode:   req.AccessCode,
		IsActive:     req.IsActive,
	})
	if err != nil {
		switch err {
		case services.ErrInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case services.ErrUserAlreadyExists:
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		case services.ErrUserEmailAlreadyExists:
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		case services.ErrUserAccessCodeAlreadyExists:
			http.Error(w, "Access code already exists", http.StatusConflict)
			return
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := mapUserDTO(u)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UsersHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.Svc.List(r.Context())
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := make([]UserDTO, len(users))
	for i, u := range users {
		resp[i] = mapUserDTO(u)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UsersHandler) getByID(w http.ResponseWriter, r *http.Request, userID int) {
	u, err := h.Svc.GetByID(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := mapUserDTO(u)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UsersHandler) patch(w http.ResponseWriter, r *http.Request, userID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req patchUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	u, err := h.Svc.Patch(r.Context(), userID, services.PatchUserInput{
		Username:     req.Username,
		Password:     req.Password,
		Role:         req.Role,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		MiddleName:   req.MiddleName,
		Email:        req.Email,
		EmployeeCode: req.EmployeeCode,
		AccessCode:   req.AccessCode,
		IsActive:     req.IsActive,
	})
	if err != nil {
		switch err {
		case services.ErrInvalidInput:
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		case services.ErrUserAlreadyExists:
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		case services.ErrUserEmailAlreadyExists:
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		case services.ErrUserAccessCodeAlreadyExists:
			http.Error(w, "Access code already exists", http.StatusConflict)
			return
		default:
			if ent.IsNotFound(err) {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	resp := mapUserDTO(u)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *UsersHandler) delete(w http.ResponseWriter, r *http.Request, userID int) {
	if !isAdmin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := h.Svc.Delete(r.Context(), userID); err != nil {
		if err == services.ErrInvalidInput {
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

/* =========================
   HELPERS
   ========================= */

func parseUserIDFromPath(path string) (int, bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 {
		return 0, false
	}
	if parts[0] != "api" || parts[1] != "v1" || parts[2] != "users" {
		return 0, false
	}
	id, err := strconv.Atoi(parts[3])
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func mapUserDTO(u *ent.User) UserDTO {
	return UserDTO{
		ID:           u.ID,
		Username:     u.Username,
		Role:         u.Role,
		IsActive:     u.IsActive,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		MiddleName:   u.MiddleName,
		Email:        u.Email,
		EmployeeCode: u.EmployeeCode,
		AccessCode:   u.AccessCode,
	}
}

func mapUserOverviewDTO(u *ent.User) UserOverviewDTO {
	name := u.Username
	if u.FirstName != nil && u.LastName != nil {
		name = *u.FirstName + " " + *u.LastName
	}

	dto := UserOverviewDTO{
		ID:       u.ID,
		Name:     name,
		Email:    u.Email,
		Role:     u.Role,
		IsActive: u.IsActive,
	}

	// Agregar sucursales
	dto.Branches = make([]BranchOverviewDTO, 0)
	if len(u.Edges.UserBranches) > 0 {
		for _, ub := range u.Edges.UserBranches {
			if ub.Edges.Branch != nil {
				dto.Branches = append(dto.Branches, BranchOverviewDTO{
					ID:   ub.Edges.Branch.ID,
					Name: ub.Edges.Branch.Name,
				})
			}
		}
	}

	// Agregar turno actual (el primero activo)
	if len(u.Edges.ShiftAssignments) > 0 {
		for _, sa := range u.Edges.ShiftAssignments {
			if sa.IsActive && sa.Edges.Shift != nil {
				dto.CurrentShift = &ShiftOverviewDTO{
					ShiftID:   sa.Edges.Shift.ID,
					ShiftName: sa.Edges.Shift.Name,
					StartTime: sa.Edges.Shift.StartTime,
					EndTime:   sa.Edges.Shift.EndTime,
					StartDate: sa.StartDate.Format("2006-01-02"),
				}
				if sa.EndDate != nil {
					endDateStr := sa.EndDate.Format("2006-01-02")
					dto.CurrentShift.EndDate = &endDateStr
				}
				break // Solo toma el primer turno activo
			}
		}
	}

	return dto
}

// UserExcelExport godoc
// @Summary      Exportar usuario - Datos completos
// @Description  GET retorna todos los datos del usuario incluyendo direcciones, sucursales, turnos, asistencia, etc. Para exportar a Excel
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id    path     int  true  "ID del usuario"
// @Success      200   {object} UserExcelExportDTO
// @Failure      401   {object} ErrorResponse
// @Failure      404   {object} ErrorResponse
// @Failure      500   {object} ErrorResponse
// @Router       /api/v1/users/{id}/export [get]
func (h *UsersHandler) UserExcelExport(w http.ResponseWriter, r *http.Request, userID int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	u, err := h.Svc.GetUserFullData(r.Context(), userID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp := mapUserExcelExportDTO(u)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func mapUserExcelExportDTO(u *ent.User) UserExcelExportDTO {
	dto := UserExcelExportDTO{
		ID:           u.ID,
		Username:     u.Username,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		MiddleName:   u.MiddleName,
		Email:        u.Email,
		Role:         u.Role,
		EmployeeCode: u.EmployeeCode,
		AccessCode:   u.AccessCode,
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Mapear Addresses
	dto.Addresses = make([]AddressDTO, 0)
	if len(u.Edges.Addresses) > 0 {
		for _, addr := range u.Edges.Addresses {
			communeName := ""
			cityName := ""
			regionName := ""

			if addr.Edges.Commune != nil {
				communeName = addr.Edges.Commune.Name
				if addr.Edges.Commune.Edges.City != nil {
					cityName = addr.Edges.Commune.Edges.City.Name
					if addr.Edges.Commune.Edges.City.Edges.Region != nil {
						regionName = addr.Edges.Commune.Edges.City.Edges.Region.Name
					}
				}
			}

			dto.Addresses = append(dto.Addresses, AddressDTO{
				ID:         addr.ID,
				Street:     addr.Street,
				Number:     addr.Number,
				Apartment:  addr.Apartment,
				CommuneName: communeName,
				CityName:   cityName,
				RegionName: regionName,
				CreatedAt:  addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:  addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// Mapear UserBranches
	dto.UserBranches = make([]UserBranchExcelDTO, 0)
	if len(u.Edges.UserBranches) > 0 {
		for _, ub := range u.Edges.UserBranches {
			branchDTO := BranchDTO{}
			if ub.Edges.Branch != nil {
				branchDTO = BranchDTO{
					ID:       ub.Edges.Branch.ID,
					Name:     ub.Edges.Branch.Name,
					Code:     ub.Edges.Branch.Code,
					IsActive: ub.Edges.Branch.IsActive,
				}
			}
			dto.UserBranches = append(dto.UserBranches, UserBranchExcelDTO{
				ID:          ub.ID,
				Branch:      branchDTO,
				RoleInBranch: ub.RoleInBranch,
				IsActive:    ub.IsActive,
			})
		}
	}

	// Mapear UserAccessPoints
	dto.UserAccessPoints = make([]UserAccessPointExcelDTO, 0)
	if len(u.Edges.UserAccessPoints) > 0 {
		for _, uap := range u.Edges.UserAccessPoints {
			apName := ""
			apID := 0
			if uap.Edges.AccessPoint != nil {
				apID = uap.Edges.AccessPoint.ID
				apName = uap.Edges.AccessPoint.Name
			}
			var revokedAt *string
			if uap.RevokedAt != nil {
				t := uap.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
				revokedAt = &t
			}
			dto.UserAccessPoints = append(dto.UserAccessPoints, UserAccessPointExcelDTO{
				ID:               uap.ID,
				AccessPointID:    apID,
				AccessPointName:  apName,
				IsActive:         uap.IsActive,
				AssignedAt:       uap.AssignedAt.Format("2006-01-02T15:04:05Z07:00"),
				RevokedAt:        revokedAt,
			})
		}
	}

	// Mapear ShiftAssignments
	dto.ShiftAssignments = make([]UserShiftAssignmentExcelDTO, 0)
	if len(u.Edges.ShiftAssignments) > 0 {
		for _, sa := range u.Edges.ShiftAssignments {
			shiftID := 0
			shiftName := ""
			startTime := ""
			endTime := ""
			if sa.Edges.Shift != nil {
				shiftID = sa.Edges.Shift.ID
				shiftName = sa.Edges.Shift.Name
				startTime = sa.Edges.Shift.StartTime
				endTime = sa.Edges.Shift.EndTime
			}
			var endDate *string
			if sa.EndDate != nil {
				t := sa.EndDate.Format("2006-01-02")
				endDate = &t
			}
			dto.ShiftAssignments = append(dto.ShiftAssignments, UserShiftAssignmentExcelDTO{
				ID:        sa.ID,
				ShiftID:   shiftID,
				ShiftName: shiftName,
				StartTime: startTime,
				EndTime:   endTime,
				StartDate: sa.StartDate.Format("2006-01-02"),
				EndDate:   endDate,
				IsActive:  sa.IsActive,
				CreatedAt: sa.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// Mapear DayOverrides
	dto.DayOverrides = make([]UserDayOverrideExcelDTO, 0)
	if len(u.Edges.DayOverrides) > 0 {
		for _, dod := range u.Edges.DayOverrides {
			var shiftID *int
			var shiftName *string
			if dod.Edges.Shift != nil {
				shiftID = &dod.Edges.Shift.ID
				shiftName = &dod.Edges.Shift.Name
			}
			dto.DayOverrides = append(dto.DayOverrides, UserDayOverrideExcelDTO{
				ID:        dod.ID,
				Date:      dod.Date.Format("2006-01-02"),
				IsDayOff:  dod.IsDayOff,
				Mode:      dod.Mode,
				ShiftID:   shiftID,
				ShiftName: shiftName,
				Notes:     dod.Notes,
				CreatedAt: dod.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// Mapear AttendanceDays
	dto.AttendanceDays = make([]AttendanceDayDTO, 0)
	if len(u.Edges.AttendanceDays) > 0 {
		for _, ad := range u.Edges.AttendanceDays {
			var workInAt, breakOutAt, breakInAt, workOutAt *string
			if ad.WorkInAt != nil {
				t := ad.WorkInAt.Format("2006-01-02T15:04:05Z07:00")
				workInAt = &t
			}
			if ad.BreakOutAt != nil {
				t := ad.BreakOutAt.Format("2006-01-02T15:04:05Z07:00")
				breakOutAt = &t
			}
			if ad.BreakInAt != nil {
				t := ad.BreakInAt.Format("2006-01-02T15:04:05Z07:00")
				breakInAt = &t
			}
			if ad.WorkOutAt != nil {
				t := ad.WorkOutAt.Format("2006-01-02T15:04:05Z07:00")
				workOutAt = &t
			}
			dto.AttendanceDays = append(dto.AttendanceDays, AttendanceDayDTO{
				ID:            ad.ID,
				WorkDate:      ad.WorkDate.Format("2006-01-02"),
				WorkInAt:      workInAt,
				BreakOutAt:    breakOutAt,
				BreakInAt:     breakInAt,
				WorkOutAt:     workOutAt,
				BranchID:      ad.BranchID,
				AccessPointID: ad.AccessPointID,
				CreatedAt:     ad.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:     ad.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	// Mapear RefreshTokens
	dto.RefreshTokens = make([]RefreshTokenDTO, 0)
	if len(u.Edges.RefreshTokens) > 0 {
		for _, rt := range u.Edges.RefreshTokens {
			var revokedAt *string
			if rt.RevokedAt != nil {
				t := rt.RevokedAt.Format("2006-01-02T15:04:05Z07:00")
				revokedAt = &t
			}
			dto.RefreshTokens = append(dto.RefreshTokens, RefreshTokenDTO{
				ID:        rt.ID,
				CreatedAt: rt.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				ExpiresAt: rt.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
				RevokedAt: revokedAt,
			})
		}
	}

	// Mapear QRSessions
	dto.QRSessions = make([]UserQRSessionDTO, 0)
	if len(u.Edges.QrSessions) > 0 {
		for _, qs := range u.Edges.QrSessions {
			dto.QRSessions = append(dto.QRSessions, UserQRSessionDTO{
				ID:        qs.ID,
				IssuedAt:  qs.IssuedAt.Format("2006-01-02T15:04:05Z07:00"),
				ExpiresAt: qs.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
				IsRevoked: qs.IsRevoked,
				CreatedAt: qs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	}

	return dto
}
