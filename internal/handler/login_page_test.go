package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lab37/internal/auth"
)

var loginPageTestJWTSecret = []byte("login-page-test-jwt-secret")

func TestLoginPage_Renders(t *testing.T) {
	handler := &LoginPageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	for _, want := range []string{"<form", "username", "password", "Log In"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q\nbody=%s", want, body)
		}
	}
}

func TestLoginPage_WithError(t *testing.T) {
	handler := &LoginPageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/login?error=Test+error", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test error") {
		t.Fatalf("body missing %q\nbody=%s", "Test error", body)
	}
}

func TestLoginPage_AlreadyAuthenticated(t *testing.T) {
	token, err := auth.CreateToken(loginPageTestJWTSecret, "user-1", "admin", []string{"restaurant-1"})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	handler := &LoginPageHandler{JWTSecret: loginPageTestJWTSecret}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf("Location = %q, want %q", location, "/")
	}
}
