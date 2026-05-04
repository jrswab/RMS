package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBrowserLogin_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &BrowserLoginHandler{DB: database, JWTSecret: testJWTSecret}

	form := url.Values{}
	form.Set("username", "testuser")
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/login/browser", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if got := rec.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q, want %q", got, "/")
	}

	setCookie := rec.Header().Get("Set-Cookie")
	for _, want := range []string{"auth_token=", "HttpOnly", "Path=/", "Max-Age=3600"} {
		if !strings.Contains(setCookie, want) {
			t.Errorf("Set-Cookie = %q, want substring %q", setCookie, want)
		}
	}
}

func TestBrowserLogin_WrongPassword(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &BrowserLoginHandler{DB: database, JWTSecret: testJWTSecret}

	form := url.Values{}
	form.Set("username", "testuser")
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/login/browser", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if !strings.Contains(rec.Body.String(), "Invalid username or password") {
		t.Errorf("body = %q, want invalid credentials message", rec.Body.String())
	}

	if got := rec.Header().Get("Set-Cookie"); got != "" {
		t.Errorf("Set-Cookie = %q, want empty", got)
	}
}

func TestBrowserLogin_UserNotFound(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &BrowserLoginHandler{DB: database, JWTSecret: testJWTSecret}

	form := url.Values{}
	form.Set("username", "nonexistent")
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/login/browser", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	if !strings.Contains(rec.Body.String(), "Invalid username or password") {
		t.Errorf("body = %q, want invalid credentials message", rec.Body.String())
	}
}

func TestBrowserLogin_EmptyUsername(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &BrowserLoginHandler{DB: database, JWTSecret: testJWTSecret}

	form := url.Values{}
	form.Set("username", "")
	form.Set("password", "password123")

	req := httptest.NewRequest(http.MethodPost, "/login/browser", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestBrowserLogin_EmptyPassword(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &BrowserLoginHandler{DB: database, JWTSecret: testJWTSecret}

	form := url.Values{}
	form.Set("username", "testuser")
	form.Set("password", "")

	req := httptest.NewRequest(http.MethodPost, "/login/browser", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}
