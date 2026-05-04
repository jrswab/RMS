package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogout_ClearsCookie(t *testing.T) {
	handler := &LogoutHandler{}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want %q", location, "/login")
	}

	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "auth_token=") {
		t.Fatalf("Set-Cookie = %q, want auth_token cookie", setCookie)
	}

	if !strings.Contains(setCookie, "Max-Age=0") {
		t.Fatalf("Set-Cookie = %q, want Max-Age=0 for cookie deletion", setCookie)
	}
}

func TestLogout_WithoutCookie(t *testing.T) {
	handler := &LogoutHandler{}

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want %q", location, "/login")
	}
}
