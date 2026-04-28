package server

import (
	"database/sql"
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

func New(cfg *config.Config, client *ent.Client, db *sql.DB) *http.Server {
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
	qrSessionService := services.NewQRSessionService(client)
	attendanceService := services.NewAttendanceService(client, qrSessionService)
	dashboardService := services.NewDashboardService(client, db)

	// =========================
	// Handlers
	// =========================
	authHandler := handlers.NewAuthHandler(cfg, usersService, tokenService)
	usersHandler := handlers.NewUsersHandler(usersService, qrSessionService)
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
	attendanceHandler := handlers.NewAttendanceHandler(attendanceService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	shiftHandler := handlers.NewShiftHandler(shiftService)
	shiftDayHandler := handlers.NewShiftDayHandler(shiftDayService)

	swaggerLoginHandler := handlers.NewSwaggerLoginHandler(
		cfg.Swagger.User,
		cfg.Swagger.Pass,
	)

	// =========================
	// Swagger UI (LOGIN PROPIO)
	// =========================
	mux.HandleFunc("/swagger-login", swaggerLoginHandler.Handle)
	mux.HandleFunc("/swagger-logout", swaggerLoginHandler.Logout)

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
	// Public routes (ATTENDANCE)
	// =========================
	mux.HandleFunc("/api/v1/attendance/validate-qr", attendanceHandler.ValidateQR)
	mux.HandleFunc("/api/v1/attendance/validate-access-code", attendanceHandler.ValidateAccessCode)

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

	protectedUsersOverview := middleware.Chain(
		http.HandlerFunc(usersHandler.UsersOverview),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/users/overview", protectedUsersOverview)

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
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userBranchHandler.UserBranches(w, r)
				return
			}

			// /api/v1/users/{id}/branches/{branch_id}
			if len(parts) == 6 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "branches" {
				if parseID(parts[3]) <= 0 || parseID(parts[5]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userBranchHandler.UserBranchByID(w, r)
				return
			}

			// /api/v1/users/{id}/access-points
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "access-points" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userAccessPointHandler.UserAccessPoints(w, r)
				return
			}

			// /api/v1/users/{id}/access-points/{access_point_id}
			if len(parts) == 6 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "access-points" {
				if parseID(parts[3]) <= 0 || parseID(parts[5]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userAccessPointHandler.UserAccessPointByID(w, r)
				return
			}

			// /api/v1/users/{id}/shift-assignments
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "shift-assignments" {
				userID := parseID(parts[3])
				if userID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userShiftAssignmentHandler.Assignments(w, r, userID)
				return
			}

			// /api/v1/users/{id}/day-overrides
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "day-overrides" {
				userID := parseID(parts[3])
				if userID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				userDayOverrideHandler.Overrides(w, r, userID)
				return
			}

			// /api/v1/users/{id}/overview
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "overview" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				usersHandler.UserOverview(w, r)
				return
			}

			// /api/v1/users/{id}/access-code
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "access-code" {
				userID := parseID(parts[3])
				if userID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				usersHandler.UserAccessCode(w, r, userID)
				return
			}

			// /api/v1/users/{id}/export
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "export" {
				userID := parseID(parts[3])
				if userID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				usersHandler.UserExcelExport(w, r, userID)
				return
			}

			// /api/v1/users/{id}/qr-session
			if len(parts) == 5 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" &&
				parts[4] == "qr-session" {
				userID := parseID(parts[3])
				if userID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				usersHandler.GenerateQRSession(w, r, userID)
				return
			}

			// /api/v1/users/{id}
			if len(parts) == 4 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "users" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				usersHandler.UserByID(w, r)
				return
			}

			http.NotFound(w, r)
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
				shiftID := parseID(parts[3])
				if shiftID <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				shiftDayHandler.ShiftDays(w, r, shiftID)
				return
			}

			// /api/v1/shifts/{id}
			if len(parts) == 4 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "shifts" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				shiftHandler.ShiftByID(w, r)
				return
			}

			http.NotFound(w, r)
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
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				accessPointHandler.BranchAccessPoints(w, r)
				return
			}

			// /api/v1/branches/{id}
			if len(parts) == 4 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "branches" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				branchHandler.BranchByID(w, r)
				return
			}

			http.NotFound(w, r)
		}),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/branches/", protectedBranchSubroutes)

	// =========================
	// Protected routes (ACCESS POINTS + DEVICES by Access Point)
	// =========================
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
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				deviceHandler.AccessPointDevices(w, r)
				return
			}

			// /api/v1/access-points/{id}
			if len(parts) == 4 &&
				parts[0] == "api" &&
				parts[1] == "v1" &&
				parts[2] == "access-points" {
				if parseID(parts[3]) <= 0 {
					http.Error(w, "Not Found", http.StatusNotFound)
					return
				}
				accessPointHandler.AccessPointByID(w, r)
				return
			}

			http.NotFound(w, r)
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
	// Protected routes (DASHBOARD)
	// =========================
	protectedDashboardStats := middleware.Chain(
		http.HandlerFunc(dashboardHandler.Stats),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/dashboard/stats", protectedDashboardStats)

	protectedDashboardLive := middleware.Chain(
		http.HandlerFunc(dashboardHandler.Live),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/dashboard/live", protectedDashboardLive)

	protectedDashboardPunctuality := middleware.Chain(
		http.HandlerFunc(dashboardHandler.Punctuality),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/dashboard/punctuality", protectedDashboardPunctuality)

	protectedDashboardExport := middleware.Chain(
		http.HandlerFunc(dashboardHandler.Export),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/dashboard/export", protectedDashboardExport)

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

func parseID(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}