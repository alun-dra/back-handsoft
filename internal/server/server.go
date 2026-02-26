package server

import (
	"net/http"

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
	// Swagger UI
	// =========================
	// UI:   /docs/index.html
	// JSON: /docs/doc.json
	mux.Handle("/docs/", httpSwagger.WrapHandler)

	// (Opcional) Redirigir /docs a /docs/index.html
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})

	// =========================
	// Services
	// =========================
	usersService := services.NewUsersService(client)
	tokenService := services.NewTokenService(cfg, client)

	locationService := services.NewLocationService(client)
	addressService := services.NewAddressService(client)

	branchService := services.NewBranchService(client)
	deviceService := services.NewDeviceService(client)

	// =========================
	// Handlers
	// =========================
	authHandler := handlers.NewAuthHandler(cfg, usersService, tokenService)
	locationHandler := handlers.NewLocationHandler(locationService)
	addressHandler := handlers.NewAddressHandler(addressService)

	branchHandler := handlers.NewBranchHandler(branchService)
	deviceHandler := handlers.NewDeviceHandler(deviceService)

	// =========================
	// Public routes (AUTH)
	// =========================
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// =========================
	// Public routes (CATÁLOGO)
	// =========================
	mux.HandleFunc("/api/v1/regions", locationHandler.Regions)
	mux.HandleFunc("/api/v1/regions/", locationHandler.RegionSubroutes)
	mux.HandleFunc("/api/v1/cities/", locationHandler.CitySubroutes)

	// =========================
	// Protected routes (AUTH)
	// =========================

	// Register (admin-only)
	protectedRegister := middleware.Chain(
		http.HandlerFunc(authHandler.Register),
		middleware.JWT(cfg),
		middleware.RequireRole("admin"),
	)
	mux.Handle("/api/v1/auth/register", protectedRegister)

	// /me (JWT)
	protectedMe := middleware.Chain(
		http.HandlerFunc(authHandler.Me),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/me", protectedMe)

	// Logout-all (JWT)
	protectedLogoutAll := middleware.Chain(
		http.HandlerFunc(authHandler.LogoutAll),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/auth/logout-all", protectedLogoutAll)

	// Sessions (JWT)
	protectedSessions := middleware.Chain(
		http.HandlerFunc(authHandler.Sessions),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/auth/sessions", protectedSessions)

	// =========================
	// Protected routes (ADDRESSES)
	// =========================

	// /api/v1/addresses (GET, POST)
	protectedAddresses := middleware.Chain(
		http.HandlerFunc(addressHandler.Addresses),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/addresses", protectedAddresses)

	// /api/v1/addresses/{id} (PATCH, DELETE) -> necesita slash final
	protectedAddressByID := middleware.Chain(
		http.HandlerFunc(addressHandler.AddressByID),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/addresses/", protectedAddressByID)

	// =========================
	// Protected routes (BRANCHES)
	// =========================

	// /api/v1/branches (GET, POST)
	protectedBranches := middleware.Chain(
		http.HandlerFunc(branchHandler.Branches),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/branches", protectedBranches)

	// /api/v1/branches/{id} (GET, PATCH, DELETE) -> necesita slash final
	protectedBranchByID := middleware.Chain(
		http.HandlerFunc(branchHandler.BranchByID),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/branches/", protectedBranchByID)

	// =========================
	// Protected routes (DEVICES)
	// =========================

	// /api/v1/access-points/{id}/devices (GET, POST) -> se registra como prefijo /api/v1/access-points/
	protectedAccessPointDevices := middleware.Chain(
		http.HandlerFunc(deviceHandler.AccessPointDevices),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/access-points/", protectedAccessPointDevices)

	// /api/v1/devices/{id} (PATCH, DELETE) -> necesita slash final
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
