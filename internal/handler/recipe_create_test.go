package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"lab37/internal/auth"
	"lab37/internal/store"
)

type recipeCreatePayload struct {
	Name         string                    `json:"name"`
	Yield        string                    `json:"yield"`
	Instructions string                    `json:"instructions"`
	Ingredients  []recipeIngredientPayload `json:"ingredients"`
	RestaurantID string                    `json:"selectedRestaurant"`
}

type recipeIngredientPayload struct {
	FoodName string `json:"foodName"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}

func makeRecipeCreatePostRequest(t *testing.T, payload recipeCreatePayload) *http.Request {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/recipe/new", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return req
}

func countRows(t *testing.T, database *sql.DB, query string, args ...any) int {
	t.Helper()

	var count int
	if err := database.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("QueryRow count failed: %v", err)
	}

	return count
}

func singleRecipe(t *testing.T, database *sql.DB) store.Recipe {
	t.Helper()

	var recipe store.Recipe
	if err := database.QueryRow(
		`SELECT id, name, restaurant_id, instructions, "yield", created_at, updated_at FROM recipes`,
	).Scan(
		&recipe.ID,
		&recipe.Name,
		&recipe.RestaurantID,
		&recipe.Instructions,
		&recipe.Yield,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
	); err != nil {
		t.Fatalf("QueryRow recipe failed: %v", err)
	}

	return recipe
}

func assertSSEResponse(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", contentType)
	}
}

func TestRecipeCreate_GetRendersForm(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/new?restaurant="+restaurantID, nil)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Create New Recipe") {
		t.Errorf("body should contain create title, got %q", body)
	}
	if !strings.Contains(body, "Recipe Name") {
		t.Errorf("body should contain recipe name field, got %q", body)
	}
	if !strings.Contains(body, "Add Ingredient") {
		t.Errorf("body should contain add ingredient button, got %q", body)
	}

	unescapedBody := html.UnescapeString(body)
	if !strings.Contains(unescapedBody, `<header data-signals="{selectedRestaurant: ''}">`) {
		t.Errorf("body should define selectedRestaurant as a string signal in the header, got %q", unescapedBody)
	}
	wantSignal := `<div id="recipe-form-page" data-signals="{"selectedRestaurant":"` + restaurantID + `","ingredients":[{"foodName":"","quantity":"","unit":""}],"_ingredientRowsVersion":0}">`
	if !strings.Contains(unescapedBody, wantSignal) {
		t.Errorf("body should render form signals with the selected restaurant and ingredient row, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `class="restaurant-display">Test Restaurant</span>`) {
		t.Errorf("body should render the selected restaurant as read-only text, got %q", unescapedBody)
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

func TestRecipeCreate_GetRedirectsHomeWhenRestaurantMissing(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/new", nil)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/" {
		t.Fatalf("Location = %q, want %q", location, "/")
	}
}

func TestRecipeCreate_GetForbidsUnauthorizedRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")
	unauthorizedRestaurantID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, unauthorizedRestaurantID, "Unauthorized Restaurant"); err != nil {
		t.Fatalf("insert unauthorized restaurant: %v", err)
	}

	handler := &RecipeCreateHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/new?restaurant="+unauthorizedRestaurantID, nil)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestRecipeCreate_GetRendersSelectedRestaurantFromQuery(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/new?restaurant="+restaurantID, nil)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	unescapedBody := html.UnescapeString(rec.Body.String())
	wantSignal := `<div id="recipe-form-page" data-signals="{"selectedRestaurant":"` + restaurantID + `","ingredients":[{"foodName":"","quantity":"","unit":""}],"_ingredientRowsVersion":0}">`
	if !strings.Contains(unescapedBody, wantSignal) {
		t.Errorf("body should render selected restaurant from query in form signals, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `class="restaurant-display">Test Restaurant</span>`) {
		t.Errorf("body should render the selected restaurant name as read-only text, got %q", unescapedBody)
	}
	if !strings.Contains(unescapedBody, `<input type="hidden" data-bind:selectedRestaurant value="`+restaurantID+`">`) {
		t.Errorf("body should keep selectedRestaurant bound through a hidden input, got %q", unescapedBody)
	}
	if strings.Contains(unescapedBody, `<select id="restaurant"`) {
		t.Errorf("body should not render a restaurant select, got %q", unescapedBody)
	}
}

func TestRecipeCreate_Success(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "staff")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "staff", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 1 {
		t.Fatalf("recipe count = %d, want 1", got)
	}

	recipe := singleRecipe(t, database)
	if recipe.Name != "Tomato Soup" {
		t.Errorf("recipe.Name = %q, want %q", recipe.Name, "Tomato Soup")
	}
	if recipe.RestaurantID != restaurantID {
		t.Errorf("recipe.RestaurantID = %q, want %q", recipe.RestaurantID, restaurantID)
	}
	if recipe.Instructions != "Simmer and serve." {
		t.Errorf("recipe.Instructions = %q, want %q", recipe.Instructions, "Simmer and serve.")
	}
	if recipe.Yield != 4 {
		t.Errorf("recipe.Yield = %d, want %d", recipe.Yield, 4)
	}

	if !strings.Contains(rec.Body.String(), "/recipe/"+recipe.ID) {
		t.Errorf("response should redirect to %q, got %q", "/recipe/"+recipe.ID, rec.Body.String())
	}

	ingredients, err := store.ListIngredientsByRecipeID(database, recipe.ID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}
	if len(ingredients) != 0 {
		t.Fatalf("ingredient count = %d, want 0", len(ingredients))
	}
}

func TestRecipeCreate_WithIngredients(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Bread",
		Yield:        "2",
		Instructions: "Mix ingredients.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: "Flour", Quantity: "2.5", Unit: "cups"},
			{FoodName: "Salt", Quantity: "1", Unit: "tsp"},
		},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	recipe := singleRecipe(t, database)
	ingredients, err := store.ListIngredientsByRecipeID(database, recipe.ID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}

	if len(ingredients) != 2 {
		t.Fatalf("ingredient count = %d, want 2", len(ingredients))
	}

	if ingredients[0].SortOrder != 1 {
		t.Errorf("ingredients[0].SortOrder = %d, want 1", ingredients[0].SortOrder)
	}
	if ingredients[0].Quantity != 2.5 {
		t.Errorf("ingredients[0].Quantity = %v, want 2.5", ingredients[0].Quantity)
	}
	if ingredients[0].Unit != "cups" {
		t.Errorf("ingredients[0].Unit = %q, want %q", ingredients[0].Unit, "cups")
	}

	foodOne, err := store.GetFoodByID(database, ingredients[0].FoodID)
	if err != nil {
		t.Fatalf("GetFoodByID first ingredient: %v", err)
	}
	if foodOne.Name != "Flour" {
		t.Errorf("first ingredient food = %q, want %q", foodOne.Name, "Flour")
	}

	if ingredients[1].SortOrder != 2 {
		t.Errorf("ingredients[1].SortOrder = %d, want 2", ingredients[1].SortOrder)
	}
	if ingredients[1].Quantity != 1 {
		t.Errorf("ingredients[1].Quantity = %v, want 1", ingredients[1].Quantity)
	}
	if ingredients[1].Unit != "tsp" {
		t.Errorf("ingredients[1].Unit = %q, want %q", ingredients[1].Unit, "tsp")
	}

	foodTwo, err := store.GetFoodByID(database, ingredients[1].FoodID)
	if err != nil {
		t.Fatalf("GetFoodByID second ingredient: %v", err)
	}
	if foodTwo.Name != "Salt" {
		t.Errorf("second ingredient food = %q, want %q", foodTwo.Name, "Salt")
	}
}

func TestRecipeCreate_NewFood(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Seasoning Blend",
		Yield:        "1",
		Instructions: "Combine.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: "Paprika", Quantity: "1", Unit: "tbsp"},
		},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if got := countRows(t, database, `SELECT COUNT(*) FROM food WHERE name = ?`, "Paprika"); got != 1 {
		t.Fatalf("Paprika food rows = %d, want 1", got)
	}

	food, err := store.GetFoodByName(database, "Paprika")
	if err != nil {
		t.Fatalf("GetFoodByName: %v", err)
	}

	recipe := singleRecipe(t, database)
	ingredients, err := store.ListIngredientsByRecipeID(database, recipe.ID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}
	if len(ingredients) != 1 {
		t.Fatalf("ingredient count = %d, want 1", len(ingredients))
	}
	if ingredients[0].FoodID != food.ID {
		t.Errorf("ingredients[0].FoodID = %q, want %q", ingredients[0].FoodID, food.ID)
	}
}

func TestRecipeCreate_ExistingFood(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	existingFoodID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, existingFoodID, "Sugar"); err != nil {
		t.Fatalf("insert existing food: %v", err)
	}

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Simple Syrup",
		Yield:        "1",
		Instructions: "Heat and stir.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: "Sugar", Quantity: "1", Unit: "cup"},
		},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if got := countRows(t, database, `SELECT COUNT(*) FROM food WHERE name = ?`, "Sugar"); got != 1 {
		t.Fatalf("Sugar food rows = %d, want 1", got)
	}

	recipe := singleRecipe(t, database)
	ingredients, err := store.ListIngredientsByRecipeID(database, recipe.ID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}
	if len(ingredients) != 1 {
		t.Fatalf("ingredient count = %d, want 1", len(ingredients))
	}
	if ingredients[0].FoodID != existingFoodID {
		t.Errorf("ingredients[0].FoodID = %q, want %q", ingredients[0].FoodID, existingFoodID)
	}
}

func TestRecipeCreate_MissingName(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "   ",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if !strings.Contains(rec.Body.String(), "Name is required") {
		t.Fatalf("response should contain name error, got %q", rec.Body.String())
	}

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 0 {
		t.Fatalf("recipe count = %d, want 0", got)
	}
}

func TestRecipeCreate_InvalidYield(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "not-a-number",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if !strings.Contains(rec.Body.String(), "Yield must be a valid integer.") {
		t.Fatalf("response should contain yield error, got %q", rec.Body.String())
	}

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 0 {
		t.Fatalf("recipe count = %d, want 0", got)
	}
}

func TestRecipeCreate_IngredientMissingFoodName(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: " ", Quantity: "1", Unit: "cup"},
		},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if !strings.Contains(rec.Body.String(), "Ingredient 1 food name is required.") {
		t.Fatalf("response should contain ingredient food error, got %q", rec.Body.String())
	}

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 0 {
		t.Fatalf("recipe count = %d, want 0", got)
	}
}

func TestRecipeCreate_BlankIngredientRowIsIgnored(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients: []recipeIngredientPayload{
			{FoodName: " ", Quantity: " ", Unit: " "},
		},
		RestaurantID: restaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 1 {
		t.Fatalf("recipe count = %d, want 1", got)
	}

	recipe := singleRecipe(t, database)
	ingredients, err := store.ListIngredientsByRecipeID(database, recipe.ID)
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID: %v", err)
	}
	if len(ingredients) != 0 {
		t.Fatalf("ingredient count = %d, want 0", len(ingredients))
	}
}

func TestRecipeCreate_UnauthorizedRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	unauthorizedRestaurantID := uuid.New().String()
	if _, err := database.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, unauthorizedRestaurantID, "Unauthorized Restaurant"); err != nil {
		t.Fatalf("insert unauthorized restaurant: %v", err)
	}

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: unauthorizedRestaurantID,
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 0 {
		t.Fatalf("recipe count = %d, want 0", got)
	}
}

func TestRecipeCreate_EmptyRestaurant(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, _ := seedTestUser(t, database, "testuser", "password123", "admin")

	handler := &RecipeCreateHandler{DB: database}

	req := makeRecipeCreatePostRequest(t, recipeCreatePayload{
		Name:         "Tomato Soup",
		Yield:        "4",
		Instructions: "Simmer and serve.",
		Ingredients:  []recipeIngredientPayload{},
		RestaurantID: "",
	})
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{"some-restaurant-id"}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertSSEResponse(t, rec)

	if !strings.Contains(rec.Body.String(), "Restaurant is required") {
		t.Fatalf("response should contain restaurant error, got %q", rec.Body.String())
	}

	if got := countRows(t, database, `SELECT COUNT(*) FROM recipes`); got != 0 {
		t.Fatalf("recipe count = %d, want 0", got)
	}
}

func TestRecipeCreate_Unauthenticated(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	handler := &RecipeCreateHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/new", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	if location := rec.Header().Get("Location"); location != "/login" {
		t.Fatalf("Location = %q, want %q", location, "/login")
	}
}
