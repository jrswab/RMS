package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"lab37/internal/auth"
	"lab37/internal/store"
)

// LoginHandler handles POST /login requests.
type LoginHandler struct {
	DB        store.DBTX
	JWTSecret []byte
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID          string   `json:"id"`
	Role        string   `json:"role"`
	Restaurants []string `json:"restaurants"`
}

// ServeHTTP handles the login request.
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := store.GetUserByUsername(h.DB, req.Username)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		log.Printf("error looking up user: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := auth.VerifyPassword(user.Password, req.Password); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	restaurantIDs, err := store.ListUserRestaurantIDs(h.DB, user.ID)
	if err != nil {
		log.Printf("error fetching restaurant IDs: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if restaurantIDs == nil {
		restaurantIDs = []string{}
	}

	token, err := auth.CreateToken(h.JWTSecret, user.ID, user.Role, restaurantIDs)
	if err != nil {
		log.Printf("error creating token: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginResponse{
		Token: token,
		User: userResponse{
			ID:          user.ID,
			Role:        user.Role,
			Restaurants: restaurantIDs,
		},
	})
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
