package middleware

import (
	"context"
	"net/http"
	"strings"

	"back/internal/auth"
	"back/internal/config"
)

type ctxKeyClaims struct{}

func GetClaims(r *http.Request) (*auth.Claims, bool) {
	v := r.Context().Value(ctxKeyClaims{})
	c, ok := v.(*auth.Claims)
	return c, ok
}

func JWT(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if h == "" || !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
			claims, err := auth.ParseAndValidate(cfg, tokenStr)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyClaims{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
