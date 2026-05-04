package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var rbacOkHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
})

func makeRequestWithRole(t *testing.T, role string) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Simulate auth middleware having run by setting claims in context
	claims := &Claims{
		UserID:        "u1",
		Role:          role,
		RestaurantIDs: []string{"r1"},
	}
	return ContextWithClaims(req, claims)
}

func TestRequireRole_AllowedRole(t *testing.T) {
	t.Parallel()

	req := makeRequestWithRole(t, "admin")
	rec := httptest.NewRecorder()

	handler := RequireRole("admin", "manager")(rbacOkHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_DeniedRole(t *testing.T) {
	t.Parallel()

	req := makeRequestWithRole(t, "staff")
	rec := httptest.NewRecorder()

	handler := RequireRole("admin", "manager")(rbacOkHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if body := rec.Body.String(); !strings.Contains(body, "forbidden") {
		t.Errorf("body = %q, want it to contain %q", body, "forbidden")
	}
}

func TestRequireRole_NoClaims(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler := RequireRole("admin")(rbacOkHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRequireRole_SingleRole(t *testing.T) {
	t.Parallel()

	req := makeRequestWithRole(t, "manager")
	rec := httptest.NewRecorder()

	handler := RequireRole("manager")(rbacOkHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestRequireRole_AllRolesAllowed(t *testing.T) {
	t.Parallel()

	req := makeRequestWithRole(t, "staff")
	rec := httptest.NewRecorder()

	handler := RequireRole("admin", "manager", "staff")(rbacOkHandler)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
