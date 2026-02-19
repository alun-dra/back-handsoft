package middleware

import (
	"net/http"
	"strings"

	"back/internal/config"
)

func CORS(cfg *config.Config) func(http.Handler) http.Handler {
	allowAll := len(cfg.CORS.AllowedOrigins) == 1 && cfg.CORS.AllowedOrigins[0] == "*"

	originAllowed := func(origin string) bool {
		if allowAll {
			return true
		}
		for _, o := range cfg.CORS.AllowedOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}

	allowedMethods := strings.Join(cfg.CORS.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.CORS.AllowedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && originAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

				if cfg.CORS.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
