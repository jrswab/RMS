package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestBrowserAuth_ValidCookie(t *testing.T) {
	t.Parallel()

	token, err := CreateToken(testSecret, "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	var capturedReq *http.Request
	handler := BrowserMiddleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		okHandler.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if capturedReq == nil {
		t.Fatal("handler did not receive request")
	}

	claims, err := ClaimsFromContext(capturedReq)
	if err != nil {
		t.Fatalf("ClaimsFromContext: %v", err)
	}

	if claims.UserID != "u1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "u1")
	}

	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	if len(claims.RestaurantIDs) != 1 || claims.RestaurantIDs[0] != "r1" {
		t.Errorf("RestaurantIDs = %v, want [r1]", claims.RestaurantIDs)
	}
}

func TestBrowserAuth_NoCookie(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := BrowserMiddleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

func TestBrowserAuth_ExpiredCookie(t *testing.T) {
	t.Parallel()

	claims := &Claims{
		UserID:        "u1",
		Role:          "admin",
		RestaurantIDs: []string{"r1"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testSecret)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	handler := BrowserMiddleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}

func TestBrowserAuth_InvalidCookie(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "garbage"})
	rec := httptest.NewRecorder()

	handler := BrowserMiddleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Errorf("Location = %q, want %q", location, "/login")
	}
}
