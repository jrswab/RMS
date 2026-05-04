package handler

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"lab37/internal/auth"
)

func withURLParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)

	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func TestRecipeView_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	foodID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, foodID, "Flour"); err != nil {
		t.Fatal(err)
	}

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Test Recipe", restaurantID, "Mix and bake.", 4,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), recipeID, foodID, 2.0, "cups", 1,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test Recipe") {
		t.Errorf("body should contain recipe name, got %q", body)
	}
	if !strings.Contains(body, "Mix and bake.") {
		t.Errorf("body should contain instructions, got %q", body)
	}
	if !strings.Contains(body, "Flour") {
		t.Errorf("body should contain ingredient food name, got %q", body)
	}
	if !strings.Contains(body, "Yield:</strong> 4") {
		t.Errorf("body should contain recipe yield, got %q", body)
	}

	unescapedBody := html.UnescapeString(body)
	wantSignal := fmt.Sprintf(`data-signals="{"selectedRestaurant":"%s"}"`, restaurantID)
	if !strings.Contains(unescapedBody, wantSignal) {
		t.Errorf("body should render selectedRestaurant as a string signal, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, `data-signals:selected-restaurant=`) {
		t.Errorf("body should not render the invalid selected-restaurant signal key, got %q", unescapedBody)
	}
}

func TestRecipeView_NoIngredients(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Empty Recipe", restaurantID, "No ingredients.", 1,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "No ingredients added yet") {
		t.Errorf("body should contain empty ingredients state, got %q", body)
	}
}

func TestRecipeView_NotFound(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/nonexistent", nil)
	req = withURLParam(req, "id", "nonexistent")
	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{"some-restaurant"}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRecipeView_ForbiddenRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Other Recipe", restaurantID, "instructions", 1,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{"different-restaurant"}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRecipeView_StaffCanView(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "staff")

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Staff Recipe", restaurantID, "instructions", 1,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	claims := &auth.Claims{UserID: userID, Role: "staff", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Contains(body, "/edit") {
		t.Errorf("staff should NOT see edit link, got %q", body)
	}
	if strings.Contains(body, "/delete-confirm") {
		t.Errorf("staff should NOT see delete button, got %q", body)
	}
}

func TestRecipeView_AdminCanEditDelete(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Admin Recipe", restaurantID, "instructions", 1,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	claims := &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}}
	req = auth.ContextWithClaims(req, claims)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "/edit") {
		t.Errorf("admin should see edit link, got %q", body)
	}
	if !strings.Contains(body, "/delete-confirm") {
		t.Errorf("admin should see delete button, got %q", body)
	}
}
