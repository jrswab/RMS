package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"lab37/internal/auth"
	"lab37/internal/store"
)

func makeRecipeDeleteRequest(method, path, recipeID string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	return withURLParam(req, "id", recipeID)
}

func TestDeleteConfirm_ShowsDialog(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodPost, "/recipe/"+recipeID+"/delete-confirm", recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	body := rec.Body.String()
	if !strings.Contains(body, "Are you sure you want to delete") {
		t.Fatalf("response should contain delete confirmation text, got %q", body)
	}
	if !strings.Contains(body, "Tomato Soup") {
		t.Fatalf("response should contain recipe name, got %q", body)
	}
	if !strings.Contains(body, "Confirm Delete") {
		t.Fatalf("response should contain confirm delete action, got %q", body)
	}
}

func TestDeleteConfirm_ForbiddenRole(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "staff")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodPost, "/recipe/"+recipeID+"/delete-confirm", recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "staff", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestDeleteConfirm_ForbiddenRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	otherRestaurantID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, otherRestaurantID, "Other Restaurant"); err != nil {
		t.Fatalf("insert other restaurant: %v", err)
	}
	recipeID := seedRecipeForEdit(t, database, otherRestaurantID, "Other Recipe", "Simmer and serve.", 4)

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodPost, "/recipe/"+recipeID+"/delete-confirm", recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestDeleteCancel_RemovesDialog(t *testing.T) {
	t.Parallel()

	handler := &RecipeDeleteHandler{}

	req := makeRecipeDeleteRequest(http.MethodPost, "/recipe/some-id/delete-cancel", "some-id")
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: uuid.New().String(), Role: "admin", RestaurantIDs: []string{uuid.New().String()}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	body := rec.Body.String()
	if !strings.Contains(body, "selector #delete-confirmation") {
		t.Fatalf("response should target delete confirmation placeholder, got %q", body)
	}
	if !strings.Contains(body, `<div id="delete-confirmation"></div>`) {
		t.Fatalf("response should restore empty delete confirmation placeholder, got %q", body)
	}
}

func TestDeleteCancel_Unauthenticated(t *testing.T) {
	t.Parallel()

	handler := &RecipeDeleteHandler{}

	req := makeRecipeDeleteRequest(http.MethodPost, "/recipe/some-id/delete-cancel", "some-id")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want %q", location, "/login")
	}
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodDelete, "/recipe/"+recipeID, recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	_, err := store.GetRecipeByID(database, recipeID)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetRecipeByID error = %v, want %v", err, store.ErrNotFound)
	}

	if !strings.Contains(rec.Body.String(), `window.location.href = "/"`) {
		t.Fatalf("response should redirect to root, got %q", rec.Body.String())
	}
}

func TestDelete_NotFound(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodDelete, "/recipe/nonexistent", "nonexistent")
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDelete_ForbiddenRole(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "staff")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeDeleteHandler{DB: database}

	req := makeRecipeDeleteRequest(http.MethodDelete, "/recipe/"+recipeID, recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "staff", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
