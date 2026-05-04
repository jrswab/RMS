package auth

import (
	"net/http"
)

// RequireRole returns an HTTP middleware that restricts access to users with one of the specified roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeJSONError(w, http.StatusForbidden, "forbidden")
		})
	}
}
