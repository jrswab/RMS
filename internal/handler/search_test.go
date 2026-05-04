package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"lab37/internal/auth"
)

func TestSearch_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	for _, name := range []string{"Tomato Soup", "Chicken Soup"} {
		if _, err := database.Exec(
			`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
			uuid.New().String(), name, restaurantID, "instructions", 4,
		); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		uuid.New().String(), "Caesar Salad", restaurantID, "instructions", 2,
	); err != nil {
		t.Fatal(err)
	}

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "Soup",
		"selectedRestaurant": restaurantID,
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "Tomato Soup") {
		t.Errorf("body should contain 'Tomato Soup', got %q", bodyStr)
	}
	if !strings.Contains(bodyStr, "Chicken Soup") {
		t.Errorf("body should contain 'Chicken Soup', got %q", bodyStr)
	}
	if strings.Contains(bodyStr, "Caesar Salad") {
		t.Errorf("body should NOT contain 'Caesar Salad', got %q", bodyStr)
	}
}

func TestSearch_NoResults(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "NonExistent",
		"selectedRestaurant": restaurantID,
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "No recipes found") {
		t.Errorf("body should contain 'No recipes found', got %q", bodyStr)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	for _, name := range []string{"Recipe A", "Recipe B"} {
		if _, err := database.Exec(
			`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
			uuid.New().String(), name, restaurantID, "instructions", 4,
		); err != nil {
			t.Fatal(err)
		}
	}

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "",
		"selectedRestaurant": restaurantID,
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "Recipe A") {
		t.Errorf("body should contain 'Recipe A', got %q", bodyStr)
	}
	if !strings.Contains(bodyStr, "Recipe B") {
		t.Errorf("body should contain 'Recipe B', got %q", bodyStr)
	}
}

func TestSearch_NoRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "Soup",
		"selectedRestaurant": "",
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{"some-restaurant"}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "Select a restaurant") {
		t.Errorf("body should contain 'Select a restaurant', got %q", bodyStr)
	}
}

func TestSearch_UnauthorizedRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "Soup",
		"selectedRestaurant": "unauthorized-restaurant-id",
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{"other-restaurant"}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestSearch_Unauthenticated(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &SearchHandler{DB: database}

	signals := map[string]string{
		"searchQuery":        "Soup",
		"selectedRestaurant": "some-restaurant",
	}
	body, _ := json.Marshal(signals)
	req := httptest.NewRequest(http.MethodPost, "/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
