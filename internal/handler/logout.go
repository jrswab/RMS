package handler

import "net/http"

// LogoutHandler handles logout requests.
type LogoutHandler struct{}

// ServeHTTP clears the auth cookie and redirects to the login page.
func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.Redirect(w, r, "/login", http.StatusFound)
}
