package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type BrowserLoginHandler struct {
	DB        store.DBTX
	JWTSecret []byte
}

func (h *BrowserLoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("error parsing browser login form: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		renderLoginPage(w, "Invalid request.")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := store.GetUserByUsername(h.DB, username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			renderLoginPage(w, "Invalid username or password.")
			return
		}

		log.Printf("error looking up browser login user: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		renderLoginPage(w, "Something went wrong.")
		return
	}

	if err := auth.VerifyPassword(user.Password, password); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		renderLoginPage(w, "Invalid username or password.")
		return
	}

	restaurantIDs, err := store.ListUserRestaurantIDs(h.DB, user.ID)
	if err != nil {
		log.Printf("error listing browser login restaurant IDs: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		renderLoginPage(w, "Something went wrong.")
		return
	}

	if restaurantIDs == nil {
		restaurantIDs = []string{}
	}

	token, err := auth.CreateToken(h.JWTSecret, user.ID, user.Role, restaurantIDs)
	if err != nil {
		log.Printf("error creating browser login token: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		renderLoginPage(w, "Something went wrong.")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func renderLoginPage(w http.ResponseWriter, errorMsg string) {
	if err := layouts.Base("Login", pages.Login(errorMsg)).Render(context.Background(), w); err != nil {
		log.Printf("error rendering login page: %v", err)
	}
}
