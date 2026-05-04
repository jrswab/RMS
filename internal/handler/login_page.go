package handler

import (
	"context"
	"net/http"

	"lab37/internal/auth"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type LoginPageHandler struct {
	JWTSecret []byte
}

func (h *LoginPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("auth_token"); err == nil {
		if _, err := auth.ValidateToken(h.JWTSecret, cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	errorMsg := r.URL.Query().Get("error")
	var ctx context.Context = r.Context()
	_ = layouts.Base("Login", pages.Login(errorMsg)).Render(ctx, w)
}
