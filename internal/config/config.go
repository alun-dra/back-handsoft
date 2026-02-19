package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvLab         Environment = "lab"
	EnvProduction  Environment = "production"
)

type Config struct {
	Env  Environment
	Port string

	DatabaseURL string

	JWT  JWTConfig
	CORS CORSConfig

	RequestTimeout time.Duration
	LogLevel       string
}

type JWTConfig struct {
	Secret           string
	Issuer           string
	Audience         []string
	AccessTTLMinutes int
	RefreshTTLDays   int
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

func Load() *Config {
	env := Environment(getEnv("ENV", string(EnvDevelopment)))
	if env != EnvDevelopment && env != EnvLab && env != EnvProduction {
		log.Fatalf("ENV inválido: %s (usa development|lab|production)", env)
	}

	cfg := &Config{
		Env:  env,
		Port: getEnv("PORT", "8080"),

		DatabaseURL: mustEnv("DATABASE_URL"),

		JWT: JWTConfig{
			Secret:           mustEnv("JWT_SECRET"),
			Issuer:           mustEnv("JWT_ISSUER"),
			Audience:         splitCSV(getEnv("JWT_AUDIENCE", "web,ios,android")),
			AccessTTLMinutes: mustInt("JWT_ACCESS_TTL_MINUTES"),
			RefreshTTLDays:   mustInt("JWT_REFRESH_TTL_DAYS"),
		},

		CORS: CORSConfig{
			AllowedOrigins:   splitCSV(mustEnv("CORS_ALLOWED_ORIGINS")),
			AllowedMethods:   splitCSV(getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS")),
			AllowedHeaders:   splitCSV(getEnv("CORS_ALLOWED_HEADERS", "Authorization,Content-Type,Accept")),
			AllowCredentials: mustBool("CORS_ALLOW_CREDENTIALS"),
		},

		RequestTimeout: time.Duration(mustInt("REQUEST_TIMEOUT_SECONDS")) * time.Second,
		LogLevel:       getEnv("LOG_LEVEL", defaultLogLevel(env)),
	}

	validate(cfg)
	return cfg
}

func validate(cfg *Config) {
	if cfg.Port == "" {
		log.Fatal("PORT vacío")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL vacío")
	}

	if len(cfg.JWT.Secret) < 16 {
		log.Fatal("JWT_SECRET demasiado corto (usa uno largo y aleatorio)")
	}
	if cfg.JWT.AccessTTLMinutes <= 0 {
		log.Fatal("JWT_ACCESS_TTL_MINUTES debe ser > 0")
	}
	if len(cfg.JWT.Audience) == 0 {
		log.Fatal("JWT_AUDIENCE no puede estar vacío")
	}

	if cfg.Env == EnvProduction {
		for _, o := range cfg.CORS.AllowedOrigins {
			if o == "*" {
				log.Fatal("En producción no se permite CORS_ALLOWED_ORIGINS='*'. Usa dominios explícitos.")
			}
		}
	}

	if cfg.CORS.AllowCredentials {

		for _, o := range cfg.CORS.AllowedOrigins {
			if o == "*" {
				log.Fatal("CORS_ALLOW_CREDENTIALS=true no es compatible con CORS_ALLOWED_ORIGINS='*'")
			}
		}
	}

	for _, o := range cfg.CORS.AllowedOrigins {
		if strings.TrimSpace(o) == "" {
			log.Fatal("CORS_ALLOWED_ORIGINS contiene un origen vacío")
		}
	}
}

func defaultLogLevel(env Environment) string {
	switch env {
	case EnvProduction:
		return "info"
	default:
		return "debug"
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		log.Fatalf("Falta variable requerida: %s", key)
	}
	return v
}

func mustInt(key string) int {
	raw := mustEnv(key)
	n, err := strconv.Atoi(raw)
	if err != nil {
		log.Fatalf("%s debe ser int, recibido: %q", key, raw)
	}
	return n
}

func mustBool(key string) bool {
	raw := strings.ToLower(strings.TrimSpace(mustEnv(key)))
	switch raw {
	case "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		log.Fatalf("%s debe ser boolean (true/false), recibido: %q", key, raw)
	}
	return false
}

func (c *Config) StringSafe() string {
	return fmt.Sprintf(
		"ENV=%s PORT=%s DB=set JWT_ISSUER=%s JWT_AUD=%v JWT_TTL=%d CORS_ORIGINS=%v TIMEOUT=%s LOG_LEVEL=%s",
		c.Env, c.Port, c.JWT.Issuer, c.JWT.Audience, c.JWT.AccessTTLMinutes, c.CORS.AllowedOrigins, c.RequestTimeout, c.LogLevel,
	)
}
