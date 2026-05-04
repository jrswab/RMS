package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var testSecret = []byte("test-secret-for-middleware")

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})

func TestMiddleware_ValidToken(t *testing.T) {
	t.Parallel()

	token, err := CreateToken(testSecret, "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMiddleware_NoHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if body := rec.Body.String(); !strings.Contains(body, "unauthorized") {
		t.Errorf("body = %q, want it to contain %q", body, "unauthorized")
	}
}

func TestMiddleware_MalformedHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_BearerOnly(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_ExpiredToken(t *testing.T) {
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
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_ValidCookie(t *testing.T) {
	t.Parallel()

	token, err := CreateToken(testSecret, "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	var gotClaims *Claims
	handler := Middleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ClaimsFromContext(r)
		if err != nil {
			t.Errorf("ClaimsFromContext: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		gotClaims = claims
		okHandler.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if gotClaims == nil {
		t.Fatal("claims = nil, want claims in context")
	}

	if gotClaims.UserID != "u1" {
		t.Errorf("claims.UserID = %q, want %q", gotClaims.UserID, "u1")
	}
}

func TestMiddleware_ExpiredCookie(t *testing.T) {
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

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_InvalidCookie(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "garbage"})
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_AuthHeaderTakesPrecedence(t *testing.T) {
	t.Parallel()

	cookieToken, err := CreateToken(testSecret, "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken cookie token: %v", err)
	}

	headerToken, err := CreateToken(testSecret, "u2", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken header token: %v", err)
	}

	var gotClaims *Claims
	handler := Middleware(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ClaimsFromContext(r)
		if err != nil {
			t.Errorf("ClaimsFromContext: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		gotClaims = claims
		okHandler.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+headerToken)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: cookieToken})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if gotClaims == nil {
		t.Fatal("claims = nil, want claims in context")
	}

	if gotClaims.UserID != "u2" {
		t.Errorf("claims.UserID = %q, want %q", gotClaims.UserID, "u2")
	}
}

func TestMiddleware_InvalidSignature(t *testing.T) {
	t.Parallel()

	token, err := CreateToken([]byte("other-secret"), "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(okHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddleware_ClaimsInContext(t *testing.T) {
	t.Parallel()

	token, err := CreateToken(testSecret, "u1", "admin", []string{"r1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	claimsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := ClaimsFromContext(r)
		if err != nil {
			t.Errorf("ClaimsFromContext: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(claims.UserID))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler := Middleware(testSecret)(claimsHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if body := rec.Body.String(); body != "u1" {
		t.Errorf("body = %q, want %q", body, "u1")
	}
}
