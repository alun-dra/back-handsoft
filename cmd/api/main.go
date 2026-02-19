package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"

	"back/internal/config"
	"back/internal/database"
	"back/internal/server"
)

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

	client, err := database.NewEntClient(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error conectando a Postgres (Ent):", err)
	}
	defer client.Close()

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatal("Error creando esquema (migración Ent):", err)
	}

	srv := server.New(cfg, client)

	log.Println("🚀 Server escuchando en puerto:", cfg.Port)
	log.Fatal(srv.ListenAndServe())
}
