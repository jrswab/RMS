package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"lab37/internal/auth"
)

func assertHTMLDocument(t *testing.T, body, title string) {
	t.Helper()

	normalizedBody := strings.ToLower(body)

	for _, want := range []string{"<!doctype html>", "<html", "<body>"} {
		if !strings.Contains(normalizedBody, want) {
			t.Fatalf("body missing %q\nbody=%s", want, body)
		}
	}

	for _, want := range []string{"<title>" + title + "</title>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q\nbody=%s", want, body)
		}
	}
}

func assertHTMLFragment(t *testing.T, body string) {
	t.Helper()

	normalizedBody := strings.ToLower(body)

	for _, unwanted := range []string{"<!doctype html>", "<html", "<head>", "<body>"} {
		if strings.Contains(normalizedBody, unwanted) {
			t.Fatalf("body should not contain %q\nbody=%s", unwanted, body)
		}
	}
}

func TestHomePage_RendersDocumentLayout(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID := seedUserWithRestaurants(t, database, "testuser", []string{"The Rusty Spoon"})

	homeHandler := &HomeHandler{DB: database, JWTSecret: testJWTSecret}
	handler := auth.BrowserMiddleware(testJWTSecret)(homeHandler)

	req := makeAuthenticatedRequest(t, testJWTSecret, userID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	assertHTMLDocument(t, rec.Body.String(), "Recipe Manager")
}

func TestLoginPage_RendersDocumentLayout(t *testing.T) {
	t.Parallel()

	handler := &LoginPageHandler{}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	assertHTMLDocument(t, rec.Body.String(), "Login")
}

func TestBrowserLogin_ErrorRendersDocumentLayout(t *testing.T) {
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
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	assertHTMLDocument(t, rec.Body.String(), "Login")
}

func TestRecipeView_RendersDocumentLayout(t *testing.T) {
	t.Parallel()

	database := setupTestDB(t)
	defer database.Close()

	userID, restaurantID := seedTestUser(t, database, "testuser", "password123", "admin")

	recipeID := uuid.New().String()
	if _, err := database.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, 0)`,
		recipeID, "Test Recipe", restaurantID, "Mix and bake.", 4,
	); err != nil {
		t.Fatal(err)
	}

	handler := &RecipeViewHandler{DB: database}

	req := httptest.NewRequest(http.MethodGet, "/recipe/"+recipeID, nil)
	req = withURLParam(req, "id", recipeID)
	req = auth.ContextWithClaims(req, &auth.Claims{UserID: userID, Role: "admin", RestaurantIDs: []string{restaurantID}})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	assertHTMLDocument(t, rec.Body.String(), "Test Recipe")
}

func TestRecipeCreate_GetRendersDocumentLayout(t *testing.T) {
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

	assertHTMLDocument(t, rec.Body.String(), "Create Recipe")
}

func TestRecipeCreate_ValidationPatchStaysFragment(t *testing.T) {
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
	assertHTMLFragment(t, rec.Body.String())
}

func TestRecipeEdit_GetRendersDocumentLayout(t *testing.T) {
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

	assertHTMLDocument(t, rec.Body.String(), "Edit Recipe")
}

func TestRecipeEdit_ValidationPatchStaysFragment(t *testing.T) {
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
	assertHTMLFragment(t, rec.Body.String())
}
