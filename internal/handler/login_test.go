package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"lab37/internal/auth"
	"lab37/internal/db"
	"lab37/migrations"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		t.Fatal(err)
	}

	return database
}

func seedTestUser(t *testing.T, database *sql.DB, username, password, role string) (userID string, restaurantID string) {
	t.Helper()

	userID = uuid.New().String()
	restaurantID = uuid.New().String()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 0)`,
		userID, username, string(hash), role,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO restaurants (id, name) VALUES (?, ?)`,
		restaurantID, "Test Restaurant",
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`,
		userID, restaurantID,
	); err != nil {
		t.Fatal(err)
	}

	return userID, restaurantID
}

var testJWTSecret = []byte("test-jwt-secret")

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Token == "" {
		t.Error("token is empty")
	}

	if resp.User.ID != userID {
		t.Errorf("User.ID = %q, want %q", resp.User.ID, userID)
	}

	if resp.User.Role != "admin" {
		t.Errorf("User.Role = %q, want %q", resp.User.Role, "admin")
	}

	if len(resp.User.Restaurants) != 1 {
		t.Errorf("User.Restaurants length = %d, want 1", len(resp.User.Restaurants))
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"testuser","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["error"] != "invalid credentials" {
		t.Errorf("error = %q, want %q", resp["error"], "invalid credentials")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"nonexistent","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogin_MalformedJSON(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"username":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLogin_EmptyBody(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestLogin_EmptyUsername(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"testuser","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestLogin_UserWithNoRestaurants(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	// Insert user without restaurant links
	userID := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, 0, 0)`,
		userID, "lonelyuser", string(hash), "staff",
	); err != nil {
		t.Fatal(err)
	}

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"lonelyuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.User.Restaurants == nil {
		t.Error("User.Restaurants is nil, want empty slice")
	}

	if len(resp.User.Restaurants) != 0 {
		t.Errorf("User.Restaurants length = %d, want 0", len(resp.User.Restaurants))
	}
}

func TestLogin_TokenIsValid(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &LoginHandler{DB: database, JWTSecret: testJWTSecret}

	body := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp loginResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	claims, err := auth.ValidateToken(testJWTSecret, resp.Token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, userID)
	}

	if claims.Role != "admin" {
		t.Errorf("claims.Role = %q, want %q", claims.Role, "admin")
	}
}
