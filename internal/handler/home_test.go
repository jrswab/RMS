package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"lab37/internal/auth"
)

func seedUserWithRestaurants(t *testing.T, database *sql.DB, username string, restaurantNames []string) (userID string) {
	t.Helper()

	userID = uuid.New().String()

	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 0)`,
		userID, username, string(hash), "admin",
	); err != nil {
		t.Fatal(err)
	}

	for _, restaurantName := range restaurantNames {
		restaurantID := uuid.New().String()

		if _, err := database.Exec(
			`INSERT INTO restaurants (id, name) VALUES (?, ?)`,
			restaurantID, restaurantName,
		); err != nil {
			t.Fatal(err)
		}

		if _, err := database.Exec(
			`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`,
			userID, restaurantID,
		); err != nil {
			t.Fatal(err)
		}
	}

	return userID
}

func makeAuthenticatedRequest(t *testing.T, secret []byte, userID string) *http.Request {
	t.Helper()

	token, err := auth.CreateToken(secret, userID, "admin", []string{})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})

	return req
}

func TestHomePage_Renders(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID := seedUserWithRestaurants(t, database, "testuser", []string{"The Rusty Spoon"})

	homeHandler := &HomeHandler{DB: database, JWTSecret: testJWTSecret}
	handler := auth.BrowserMiddleware(testJWTSecret)(homeHandler)

	req := makeAuthenticatedRequest(t, testJWTSecret, userID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Count(body, "Recipe Manager") < 2 {
		t.Errorf("body = %q, want header and footer to contain %q", body, "Recipe Manager")
	}

	if !strings.Contains(body, "Welcome") {
		t.Errorf("body = %q, want to contain %q", body, "Welcome")
	}

	if !strings.Contains(body, "Search recipes...") {
		t.Errorf("body = %q, want to contain %q", body, "Search recipes...")
	}

	if !strings.Contains(body, `id="search-results"`) {
		t.Errorf("body = %q, want to contain %q", body, `id="search-results"`)
	}

	wantCreateButton := `<button data-on:click="if ($selectedRestaurant) window.location.href='/recipe/new?restaurant=' + $selectedRestaurant" data-attr:disabled="!$selectedRestaurant" class="button-primary">Create New Recipe</button>`
	if !strings.Contains(body, wantCreateButton) {
		t.Errorf("body = %q, want to contain %q", body, wantCreateButton)
	}

	if strings.Contains(body, `<a href="/recipe/new" class="button-primary">Create New Recipe</a>`) {
		t.Errorf("body = %q, want not to contain legacy create link", body)
	}
}

func TestHomePage_ShowsRestaurants(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID := seedUserWithRestaurants(t, database, "testuser", []string{"The Rusty Spoon", "Copper Kettle"})

	homeHandler := &HomeHandler{DB: database, JWTSecret: testJWTSecret}
	handler := auth.BrowserMiddleware(testJWTSecret)(homeHandler)

	req := makeAuthenticatedRequest(t, testJWTSecret, userID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, ">The Rusty Spoon</option>") {
		t.Errorf("body = %q, want restaurant option for %q", body, "The Rusty Spoon")
	}

	if !strings.Contains(body, ">Copper Kettle</option>") {
		t.Errorf("body = %q, want restaurant option for %q", body, "Copper Kettle")
	}
}

func TestHomePage_NoRestaurants(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID := seedUserWithRestaurants(t, database, "testuser", nil)

	homeHandler := &HomeHandler{DB: database, JWTSecret: testJWTSecret}
	handler := auth.BrowserMiddleware(testJWTSecret)(homeHandler)

	req := makeAuthenticatedRequest(t, testJWTSecret, userID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "No restaurants available") {
		t.Errorf("body = %q, want to contain %q", body, "No restaurants available")
	}

	if strings.Contains(body, "Search recipes...") {
		t.Errorf("body = %q, want not to contain %q", body, "Search recipes...")
	}

	if strings.Contains(body, "Create New Recipe") {
		t.Errorf("body = %q, want not to contain %q", body, "Create New Recipe")
	}
}

func TestHomePage_Unauthenticated(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	homeHandler := &HomeHandler{DB: database, JWTSecret: testJWTSecret}
	handler := auth.BrowserMiddleware(testJWTSecret)(homeHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}
