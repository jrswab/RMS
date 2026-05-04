package auth

import "net/http"

// BrowserMiddleware returns an HTTP middleware that validates JWT tokens from the auth cookie.
func BrowserMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth_token")
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			claims, err := ValidateToken(secret, cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			r = ContextWithClaims(r, claims)
			next.ServeHTTP(w, r)
		})
	}
}
