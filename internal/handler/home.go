package handler

import (
	"log"
	"net/http"

	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type HomeHandler struct {
	DB        store.DBTX
	JWTSecret []byte
}

func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	user, err := store.GetUserByID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error looking up user by id: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	restaurants, err := store.ListRestaurantsByUserID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error listing restaurants by user id: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_ = layouts.Base("Recipe Manager", pages.Home(user.Username, restaurants)).Render(r.Context(), w)
}
