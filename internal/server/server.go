package server

import (
	"net/http"
	"strings"

	"back/internal/config"
	"back/internal/ent"
	"back/internal/handlers"
	"back/internal/middleware"
	"back/internal/services"

	// Swagger
	_ "back/internal/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

func New(cfg *config.Config, client *ent.Client) *http.Server {
	mux := http.NewServeMux()

	// =========================
	// Health
	// =========================
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// =========================
	// Services
	// =========================
	usersService := services.NewUsersService(client)
	userBranchService := services.NewUserBranchService(client)
	userAccessPointService := services.NewUserAccessPointService(client)
	userShiftAssignmentService := services.NewUserShiftAssignmentService(client)
	userDayOverrideService := services.NewUserDayOverrideService(client)

	tokenService := services.NewTokenService(cfg, client)

	locationService := services.NewLocationService(client)
	addressService := services.NewAddressService(client)

	branchService := services.NewBranchService(client)
	accessPointService := services.NewAccessPointService(client)
	deviceService := services.NewDeviceService(client)
	deviceAuthService := services.NewDeviceAuthService(client, tokenService)

	shiftService := services.NewShiftService(client)
	shiftDayService := services.NewShiftDayService(client)

	// =========================
	// Handlers
	// =========================
	authHandler := handlers.NewAuthHandler(cfg, usersService, tokenService)
	usersHandler := handlers.NewUsersHandler(usersService)
	userBranchHandler := handlers.NewUserBranchHandler(userBranchService)
	userAccessPointHandler := handlers.NewUserAccessPointHandler(userAccessPointService)
	userShiftAssignmentHandler := handlers.NewUserShiftAssignmentHandler(userShiftAssignmentService)
	userDayOverrideHandler := handlers.NewUserDayOverrideHandler(userDayOverrideService)
	deviceAuthHandler := handlers.NewDeviceAuthHandler(deviceAuthService)

	locationHandler := handlers.NewLocationHandler(locationService)
	addressHandler := handlers.NewAddressHandler(addressService)

	branchHandler := handlers.NewBranchHandler(branchService)
	accessPointHandler := handlers.NewAccessPointHandler(accessPointService)
	deviceHandler := handlers.NewDeviceHandler(deviceService)

	shiftHandler := handlers.NewShiftHandler(shiftService)
	shiftDayHandler := handlers.NewShiftDayHandler(shiftDayService)

	swaggerLoginHandler := handlers.NewSwaggerLoginHandler(
		cfg.Swagger.User,
		cfg.Swagger.Pass,
	)

	// =========================
	// Swagger UI (LOGIN PROPIO)
	// =========================
	// Login visual para Swagger
	mux.HandleFunc("/swagger-login", swaggerLoginHandler.Handle)
	mux.HandleFunc("/swagger-logout", swaggerLoginHandler.Logout)

	// Protege toda la UI de Swagger y sus assets
	protectedDocs := middleware.Chain(
		httpSwagger.WrapHandler,
		middleware.RequireSwaggerSession,
	)
	mux.Handle("/docs/", protectedDocs)

	protectedDocsRedirect := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
		}),
		middleware.RequireSwaggerSession,
	)
	mux.Handle("/docs", protectedDocsRedirect)

	// =========================
	// Public routes (AUTH)
	// =========================
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// =========================
	// Public routes (DEVICE AUTH)
	// =========================
	mux.HandleFunc("/api/v1/device-auth/login", deviceAuthHandler.Login)

	// =========================
	// Public routes (CATÁLOGO)
	// =========================
	mux.HandleFunc("/api/v1/regions", locationHandler.Regions)
	mux.HandleFunc("/api/v1/regions/", locationHandler.RegionSubroutes)
	mux.HandleFunc("/api/v1/cities/", locationHandler.CitySubroutes)

	// =========================
	// Protected routes (AUTH)
	// =========================
	protectedRegister := middleware.Chain(
		http.HandlerFunc(authHandler.Register),
		middleware.JWT(cfg),
		middleware.RequireRole("admin"),
	)
	mux.Handle("/api/v1/auth/register", protectedRegister)

	protectedMe := middleware.Chain(
		http.HandlerFunc(authHandler.Me),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/me", protectedMe)

	protectedLogoutAll := middleware.Chain(
		http.HandlerFunc(authHandler.LogoutAll),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/auth/logout-all", protectedLogoutAll)

	protectedSessions := middleware.Chain(
		http.HandlerFunc(authHandler.Sessions),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/auth/sessions", protectedSessions)

	// =========================
	// Protected routes (USERS)
	// =========================
	protectedUsers := middleware.Chain(
		http.HandlerFunc(usersHandler.Users),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/users", protectedUsers)

	// Un solo prefijo para:
	// - /api/v1/users/{id}
	// - /api/v1/users/{id}/branches
	// - /api/v1/users/{id}/branches/{branch_id}
	// - /api/v1/users/{id}/access-points
	// - /api/v1/users/{id}/access-points/{access_point_id}
	// - /api/v1/users/{id}/shift-assignments
	// - /api/v1/users/{id}/day-overrides
	protectedUserSubroutes := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.Trim(r.URL.Path, "/")
			parts := strings.Split(path, "/")

			// /api/v1/users/{id}/branches
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "branches" {
				userBranchHandler.UserBranches(w, r)
				return
			}

			// /api/v1/users/{id}/branches/{branch_id}
			if len(parts) == 6 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "branches" {
				userBranchHandler.UserBranchByID(w, r)
				return
			}

			// /api/v1/users/{id}/access-points
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "access-points" {
				userAccessPointHandler.UserAccessPoints(w, r)
				return
			}

			// /api/v1/users/{id}/access-points/{access_point_id}
			if len(parts) == 6 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "access-points" {
				userAccessPointHandler.UserAccessPointByID(w, r)
				return
			}

			// /api/v1/users/{id}/shift-assignments
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "shift-assignments" {
				userShiftAssignmentHandler.Assignments(w, r, parseUserID(parts[3]))
				return
			}

			// /api/v1/users/{id}/day-overrides
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "day-overrides" {
				userDayOverrideHandler.Overrides(w, r, parseUserID(parts[3]))
				return
			}

			// /api/v1/users/{id}
			usersHandler.UserByID(w, r)
		}),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/users/", protectedUserSubroutes)

	// =========================
	// Protected routes (SHIFTS)
	// =========================
	protectedShifts := middleware.Chain(
		http.HandlerFunc(shiftHandler.Shifts),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/shifts", protectedShifts)

	protectedCalendar := middleware.Chain(
		http.HandlerFunc(shiftHandler.Calendar),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/calendar", protectedCalendar)
	mux.Handle("/api/v1/calendar/", protectedCalendar)

	// Un solo prefijo para:
	// - /api/v1/shifts/{id}
	// - /api/v1/shifts/{id}/days
	protectedShiftSubroutes := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.Trim(r.URL.Path, "/")
			parts := strings.Split(path, "/")

			// /api/v1/shifts/{id}/days
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "shifts" &&
				parts[4] == "days" {
				shiftID := parseUserID(parts[3])
				if shiftID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				shiftDayHandler.ShiftDays(w, r, shiftID)
				return
			}

			// /api/v1/shifts/{id}
			shiftHandler.ShiftByID(w, r)
		}),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/shifts/", protectedShiftSubroutes)

	// =========================
	// Protected routes (ADDRESSES)
	// =========================
	protectedAddresses := middleware.Chain(
		http.HandlerFunc(addressHandler.Addresses),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/addresses", protectedAddresses)

	protectedAddressByID := middleware.Chain(
		http.HandlerFunc(addressHandler.AddressByID),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/addresses/", protectedAddressByID)

	// =========================
	// Protected routes (BRANCHES + ACCESS POINTS by Branch)
	// =========================
	protectedBranches := middleware.Chain(
		http.HandlerFunc(branchHandler.Branches),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/branches", protectedBranches)

	// Un solo prefijo para:
	// - /api/v1/branches/{id}
	// - /api/v1/branches/{id}/access-points
	protectedBranchSubroutes := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.Trim(r.URL.Path, "/")
			parts := strings.Split(path, "/")

			// /api/v1/branches/{id}/access-points
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "branches" &&
				parts[4] == "access-points" {
				accessPointHandler.BranchAccessPoints(w, r)
				return
			}

			// /api/v1/branches/{id}
			branchHandler.BranchByID(w, r)
		}),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/branches/", protectedBranchSubroutes)

	// =========================
	// Protected routes (ACCESS POINTS + DEVICES by Access Point)
	// =========================
	// Un solo prefijo para:
	// - /api/v1/access-points/{id}
	// - /api/v1/access-points/{id}/devices
	protectedAccessPointSubroutes := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := strings.Trim(r.URL.Path, "/")
			parts := strings.Split(path, "/")

			// /api/v1/access-points/{id}/devices
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "access-points" &&
				parts[4] == "devices" {
				deviceHandler.AccessPointDevices(w, r)
				return
			}

			// /api/v1/access-points/{id}
			accessPointHandler.AccessPointByID(w, r)
		}),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/access-points/", protectedAccessPointSubroutes)

	// =========================
	// Protected routes (DEVICES)
	// =========================
	protectedDeviceByID := middleware.Chain(
		http.HandlerFunc(deviceHandler.DeviceByID),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/devices/", protectedDeviceByID)

	// =========================
	// Global middlewares
	// =========================
	handler := middleware.Chain(
		mux,
		middleware.Recover,
		middleware.RequestID,
		middleware.Logger,
		middleware.Timeout(cfg.RequestTimeout),
		middleware.CORS(cfg),
	)

	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}
}

func parseUserID(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
