package server

import (
	"net/http"

	"back/internal/config"
	"back/internal/ent"
	"back/internal/handlers"
	"back/internal/middleware"
	"back/internal/services"
)

func New(cfg *config.Config, client *ent.Client) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Services
	usersService := services.NewUsersService(client)
	tokenService := services.NewTokenService(cfg, client)

	// Handlers
	authHandler := handlers.NewAuthHandler(cfg, usersService, tokenService)

	// Public routes
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// Register protegido (solo admin)
	protectedRegister := middleware.Chain(
		http.HandlerFunc(authHandler.Register),
		middleware.JWT(cfg),
		middleware.RequireRole("admin"),
	)
	mux.Handle("/api/v1/auth/register", protectedRegister)

	// /me protegido (solo JWT)
	protectedMe := middleware.Chain(
		http.HandlerFunc(authHandler.Me),
		middleware.JWT(cfg),
	)
	mux.Handle("/api/v1/me", protectedMe)

	// Logout-all protegido (JWT)
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

	// Middlewares globales
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
