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
