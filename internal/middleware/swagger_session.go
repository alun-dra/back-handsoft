package middleware

import "net/http"

func RequireSwaggerSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("swagger_session")
		if err != nil || cookie.Value != "authenticated" {
			http.Redirect(w, r, "/swagger-login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
