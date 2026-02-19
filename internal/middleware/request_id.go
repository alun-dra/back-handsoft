package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKeyRequestID struct{}

func GetRequestID(r *http.Request) string {
	v := r.Context().Value(ctxKeyRequestID{})
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			rid = newRequestID()
		}

		ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, rid)
		r = r.WithContext(ctx)

		w.Header().Set("X-Request-Id", rid)

		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
