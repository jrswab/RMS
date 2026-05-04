package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that validates JWT tokens from the Authorization header or auth_token cookie.
func Middleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				if !strings.HasPrefix(authHeader, "Bearer ") {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}

				tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
				if tokenString == "" {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}

				claims, err := ValidateToken(secret, tokenString)
				if err != nil {
					writeJSONError(w, http.StatusUnauthorized, "unauthorized")
					return
				}

				r = ContextWithClaims(r, claims)
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("auth_token")
			if err == nil && cookie.Value != "" {
				claims, err := ValidateToken(secret, cookie.Value)
				if err == nil {
					r = ContextWithClaims(r, claims)
					next.ServeHTTP(w, r)
					return
				}
			}

			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		})
	}
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
