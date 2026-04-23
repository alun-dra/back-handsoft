package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"back/internal/config"
	"back/internal/database"
	_ "back/internal/docs"
	"back/internal/server"
)

// @title Reloj Control API
// @version 1.0
// @description Plataforma de gestión de asistencia, turnos, sucursales, accesos y dispositivos
// @termsOfService https://minolsoft.cl/terms

// @contact.name Minolsoft
// @contact.url https://minolsoft.cl
// @contact.email soporte@minolsoft.cl

// @license.name Proprietary
// @license.url https://minolsoft.cl/license

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	if err := godotenv.Load(".env." + env); err != nil {
		log.Printf("No se pudo cargar .env.%s (puede ser normal en producción)", env)
	}

	cfg := config.Load()
	log.Println("Config:", cfg.StringSafe())

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error conectando a Postgres (sql.DB):", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Error validando conexión sql.DB:", err)
	}

	client, err := database.NewEntClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error conectando a Postgres (Ent):", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatal("Error creando esquema (migración Ent):", err)
	}

	srv := server.New(cfg, client, db)

	log.Println("Server escuchando en puerto:", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}