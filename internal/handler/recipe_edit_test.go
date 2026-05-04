package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"lab37/internal/auth"
	"lab37/internal/store"
)

func makeRecipeEditPatchRequest(t *testing.T, recipeID string, payload recipeCreatePayload) *http.Request {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPatch, "/recipe/"+recipeID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return withURLParam(req, "id", recipeID)
}

func seedRecipeForEdit(t *testing.T, database *sql.DB, restaurantID, name, instructions string, yield int64) string {
	t.Helper()

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, name, restaurantID, instructions, yield,
	); err != nil {
		t.Fatalf("insert recipe: %v", err)
	}

	return recipeID
}

func seedFoodForEdit(t *testing.T, database *sql.DB, name string) string {
	t.Helper()

	foodID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, foodID, name); err != nil {
		t.Fatalf("insert food: %v", err)
	}

	return foodID
}

func seedRecipeIngredientForEdit(t *testing.T, database *sql.DB, recipeID, foodID string, quantity float64, unit string, sortOrder int64) string {
	t.Helper()

	ingredientID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
		ingredientID, recipeID, foodID, quantity, unit, sortOrder,
	); err != nil {
		t.Fatalf("insert ingredient: %v", err)
	}

	return ingredientID
}

func TestRecipeEdit_GetRendersForm(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Bread", "Mix and bake.", 4)
	foodID := seedFoodForEdit(t, database, "Flour")
	seedRecipeIngredientForEdit(t, database, recipeID, foodID, 2.5, "cups", 1)

	handler := &RecipeEditHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID+"/edit", nil)
	req = withURLParam(req, "id", recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Edit Recipe") {
		t.Errorf("body should contain edit title, got %q", body)
	}
	if !strings.Contains(body, "Bread") {
		t.Errorf("body should contain recipe name, got %q", body)
	}
	if !strings.Contains(body, "Flour") {
		t.Errorf("body should contain ingredient food name, got %q", body)
	}

	unescapedBody := html.UnescapeString(body)
	wantSignal := fmt.Sprintf(`<div id="recipe-form-page" data-signals="{"selectedRestaurant":"%s","ingredients":[{"foodName":"Flour","quantity":"2.5","unit":"cups"}],"_ingredientRowsVersion":0}">`, restaurantID)
	if !strings.Contains(unescapedBody, wantSignal) {
		t.Errorf("body should render selectedRestaurant and ingredients as form signals, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `class="restaurant-display">Test Restaurant</span>`) {
		t.Errorf("body should render the restaurant as read-only text, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `<input type="hidden" data-bind:selectedRestaurant value="`+restaurantID+`">`) {
		t.Errorf("body should keep selectedRestaurant bound through a hidden input, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, `<select id="restaurant"`) {
		t.Errorf("body should not render a restaurant select, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, "/recipe/ingredient-add") {
		t.Errorf("body should not reference deleted ingredient add endpoint, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, "/recipe/ingredient-remove") {
		t.Errorf("body should not reference deleted ingredient remove endpoint, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, "data-bind:ingredients[") {
		t.Errorf("body should not render invalid data-bind key/value ingredient bindings, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `window.renderIngredientRows`) {
		t.Errorf("body should include client-side ingredient row renderer, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `@peek(() => window.renderIngredientRows($ingredients))`) {
		t.Errorf("body should render ingredient rows from Datastar signals, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `$ingredients.push({foodName: '', quantity: '', unit: ''}); $_ingredientRowsVersion++`) {
		t.Errorf("body should add ingredient rows client-side, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `filter((_, idx) => idx !== ${i}); $_ingredientRowsVersion++`) {
		t.Errorf("body should remove ingredient rows client-side, got %q", unescapedBody)
	}
}

func TestRecipeEdit_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeEditHandler{DB: database}

	req := makeRecipeEditPatchRequest(t, recipeID, recipeCreatePayload{
		Name:         "Roasted Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: uuid.New().String(),
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if !strings.Contains(rec.Body.String(), "/recipe/"+recipeID) {
		t.Fatalf("response should redirect to %q, got %q", "/recipe/"+recipeID, rec.Body.String())
	}

	recipe, err := store.GetRecipeByID(database, recipeID)
	if err != nil {
		t.Fatalf("GetRecipeByID: %v", err)
	}

	if recipe.Name != "Roasted Tomato Soup" {
		t.Errorf("recipe.Name = %q, want %q", recipe.Name, "Roasted Tomato Soup")
	}
	if recipe.RestaurantID != restaurantID {
		t.Errorf("recipe.RestaurantID = %q, want %q", recipe.RestaurantID, restaurantID)
	}
	if recipe.UpdatedAt == 0 {
		t.Errorf("recipe.UpdatedAt = %d, want non-zero", recipe.UpdatedAt)
	}
}

func TestRecipeEdit_UpdatesIngredients(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Bread", "Mix ingredients.", 2)
	oldFoodID := seedFoodForEdit(t, database, "Flour")
	seedRecipeIngredientForEdit(t, database, recipeID, oldFoodID, 2.5, "cups", 1)

	handler := &RecipeEditHandler{DB: database}

	req := makeRecipeEditPatchRequest(t, recipeID, recipeCreatePayload{
		Name:         "Bread",
		Yield:        "2",
		Instructions: "Mix ingredients.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: "Salt", Quantity: "1", Unit: "tsp"},
			{FoodName: "Water", Quantity: "2", Unit: "cups"},
		},
		RestaurantID: uuid.New().String(),
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if got := countRows(t, database, `SELECT COUNT(*) FROM ingredients WHERE recipe_id = ? AND food_id = ?`, recipeID, oldFoodID); got != 0 {
		t.Fatalf("old ingredient rows = %d, want 0", got)
	}

	ingredients, err := store.ListIngredientsByRecipeID(database, recipeID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}

	if len(ingredients) != 2 {
		t.Fatalf("ingredient count = %d, want 2", len(ingredients))
	}

	foodOne, err := store.GetFoodByID(database, ingredients[0].FoodID)
	if err != nil {
		t.Fatalf("GetFoodByID first ingredient: %v", err)
	}
	if foodOne.Name != "Salt" {
		t.Errorf("first ingredient food = %q, want %q", foodOne.Name, "Salt")
	}
	if ingredients[0].Quantity != 1 {
		t.Errorf("ingredients[0].Quantity = %v, want 1", ingredients[0].Quantity)
	}
	if ingredients[0].Unit != "tsp" {
		t.Errorf("ingredients[0].Unit = %q, want %q", ingredients[0].Unit, "tsp")
	}
	if ingredients[0].SortOrder != 1 {
		t.Errorf("ingredients[0].SortOrder = %d, want 1", ingredients[0].SortOrder)
	}

	foodTwo, err := store.GetFoodByID(database, ingredients[1].FoodID)
	if err != nil {
		t.Fatalf("GetFoodByID second ingredient: %v", err)
	}
	if foodTwo.Name != "Water" {
		t.Errorf("second ingredient food = %q, want %q", foodTwo.Name, "Water")
	}
	if ingredients[1].Quantity != 2 {
		t.Errorf("ingredients[1].Quantity = %v, want 2", ingredients[1].Quantity)
	}
	if ingredients[1].Unit != "cups" {
		t.Errorf("ingredients[1].Unit = %q, want %q", ingredients[1].Unit, "cups")
	}
	if ingredients[1].SortOrder != 2 {
		t.Errorf("ingredients[1].SortOrder = %d, want 2", ingredients[1].SortOrder)
	}
}

func TestRecipeEdit_ForbiddenRole(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "staff")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeEditHandler{DB: database}

	req := makeRecipeEditPatchRequest(t, recipeID, recipeCreatePayload{
		Name:         "Updated Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "staff", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRecipeEdit_ForbiddenRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	otherRestaurantID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, otherRestaurantID, "Other Restaurant"); err != nil {
		t.Fatalf("insert other restaurant: %v", err)
	}
	recipeID := seedRecipeForEdit(t, database, otherRestaurantID, "Other Recipe", "Simmer and serve.", 4)

	handler := &RecipeEditHandler{DB: database}

	req := makeRecipeEditPatchRequest(t, recipeID, recipeCreatePayload{
		Name:         "Updated Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: otherRestaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRecipeEdit_NotFound(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeEditHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/nonexistent/edit", nil)
	req = withURLParam(req, "id", "nonexistent")
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRecipeEdit_ValidationFailure(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	recipeID := seedRecipeForEdit(t, database, restaurantID, "Tomato Soup", "Simmer and serve.", 4)

	handler := &RecipeEditHandler{DB: database}

	req := makeRecipeEditPatchRequest(t, recipeID, recipeCreatePayload{
		Name:         "   ",
		Yield:        "4",
		Instructions: "Updated instructions.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: uuid.New().String(),
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	body := rec.Body.String()
	if !strings.Contains(body, "Name is required") {
		t.Fatalf("response should contain name error, got %q", body)
	}

	recipe, err := store.GetRecipeByID(database, recipeID)
	if err != nil {
		t.Fatalf("GetRecipeByID: %v", err)
	}

	if recipe.Name != "Tomato Soup" {
		t.Errorf("recipe.Name = %q, want %q", recipe.Name, "Tomato Soup")
	}
	if recipe.Instructions != "Simmer and serve." {
		t.Errorf("recipe.Instructions = %q, want %q", recipe.Instructions, "Simmer and serve.")
	}
}
