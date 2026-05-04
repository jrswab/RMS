package store

import (
	"database/sql"
	"errors"
	"testing"
)

const expectedRecipeTimestamp int64 = 1767225600

func TestGetRecipeByID(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	recipe, err := GetRecipeByID(db, deterministicUUID(testRecipeID))
	if err != nil {
		t.Fatalf("GetRecipeByID returned error: %v", err)
	}
	if recipe == nil {
		t.Fatal("GetRecipeByID returned nil recipe")
	}

	assertRecipeEqual(t, *recipe, Recipe{
		ID:           deterministicUUID(testRecipeID),
		Name:         testRecipeName,
		RestaurantID: deterministicUUID(testRestaurantID),
		Instructions: "Mix ingredients and bake.",
		Yield:        4,
		CreatedAt:    expectedRecipeTimestamp,
		UpdatedAt:    expectedRecipeTimestamp,
	})
}

func TestGetRecipeByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	recipe, err := GetRecipeByID(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if recipe != nil {
		t.Fatalf("recipe = %#v, want nil", recipe)
	}
}

func TestSearchRecipes(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	insertRecipe(t, db, deterministicUUID("recipe:butter-chicken"), "Butter Chicken")
	insertRecipe(t, db, deterministicUUID("recipe:butter-cookies"), "Butter Cookies")
	insertRecipe(t, db, deterministicUUID("recipe:fried-rice"), "Fried Rice")

	recipes, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "Butter")
	if err != nil {
		t.Fatalf("SearchRecipes returned error: %v", err)
	}

	if len(recipes) != 2 {
		t.Fatalf("len(recipes) = %d, want 2", len(recipes))
	}

	names := make(map[string]bool, len(recipes))
	for _, r := range recipes {
		names[r.Name] = true
	}
	if !names["Butter Chicken"] {
		t.Errorf("expected 'Butter Chicken' in results, got %v", names)
	}
	if !names["Butter Cookies"] {
		t.Errorf("expected 'Butter Cookies' in results, got %v", names)
	}
}

func TestSearchRecipes_OrdersByName(t *testing.T) {
	expectedErr := errors.New("query failed")
	db := &recordingSearchRecipesDB{err: expectedErr}

	_, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "Butter")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("error = %v, want %v", err, expectedErr)
	}

	wantQuery := `SELECT id, name, restaurant_id, instructions, yield, created_at, updated_at FROM recipes WHERE restaurant_id = ? AND name LIKE ? ESCAPE '\' ORDER BY name`
	if db.query != wantQuery {
		t.Fatalf("query = %q, want %q", db.query, wantQuery)
	}
}

func TestSearchRecipes_EscapeSpecialChars(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	insertRecipe(t, db, deterministicUUID("recipe:percent-target"), "100% That Recipe")
	insertRecipe(t, db, deterministicUUID("recipe:percent-decoy"), "1000 That Recipe")
	insertRecipe(t, db, deterministicUUID("recipe:underscore-target"), "A_B_C")
	insertRecipe(t, db, deterministicUUID("recipe:underscore-decoy"), "A1B_C")

	percentRecipes, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "100%")
	if err != nil {
		t.Fatalf("SearchRecipes returned error for percent query: %v", err)
	}

	if len(percentRecipes) != 1 {
		t.Fatalf("len(percentRecipes) = %d, want 1", len(percentRecipes))
	}
	if percentRecipes[0].Name != "100% That Recipe" {
		t.Fatalf("percentRecipes[0].Name = %q, want %q", percentRecipes[0].Name, "100% That Recipe")
	}

	underscoreRecipes, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "A_B")
	if err != nil {
		t.Fatalf("SearchRecipes returned error for underscore query: %v", err)
	}

	if len(underscoreRecipes) != 1 {
		t.Fatalf("len(underscoreRecipes) = %d, want 1", len(underscoreRecipes))
	}
	if underscoreRecipes[0].Name != "A_B_C" {
		t.Fatalf("underscoreRecipes[0].Name = %q, want %q", underscoreRecipes[0].Name, "A_B_C")
	}
}

func TestSearchRecipes_NoResults(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	recipes, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "NonexistentTerm")
	if err != nil {
		t.Fatalf("SearchRecipes returned error: %v", err)
	}
	if recipes == nil {
		t.Fatal("expected non-nil recipes slice")
	}
	if len(recipes) != 0 {
		t.Fatalf("len(recipes) = %d, want 0", len(recipes))
	}
}

func TestSearchRecipes_EmptyQuery(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	insertRecipe(t, db, deterministicUUID("recipe:empty-query:01"), "Empty Query One")
	insertRecipe(t, db, deterministicUUID("recipe:empty-query:02"), "Empty Query Two")

	recipes, err := SearchRecipes(db, deterministicUUID(testRestaurantID), "")
	if err != nil {
		t.Fatalf("SearchRecipes returned error: %v", err)
	}
	if recipes == nil {
		t.Fatal("expected non-nil recipes slice")
	}
	if len(recipes) != 3 {
		t.Fatalf("len(recipes) = %d, want 3", len(recipes))
	}
}

func TestCreateRecipe(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	recipe := Recipe{
		ID:           deterministicUUID("recipe:new-recipe"),
		Name:         "New Recipe",
		RestaurantID: deterministicUUID(testRestaurantID),
		Instructions: "Combine and serve.",
		Yield:        6,
		CreatedAt:    1767229200,
		UpdatedAt:    1767229200,
	}

	if err := CreateRecipe(db, &recipe); err != nil {
		t.Fatalf("CreateRecipe returned error: %v", err)
	}

	storedRecipe, err := GetRecipeByID(db, recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeByID returned error: %v", err)
	}
	if storedRecipe == nil {
		t.Fatal("GetRecipeByID returned nil recipe")
	}

	assertRecipeEqual(t, *storedRecipe, recipe)
}

func TestUpdateRecipe(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	recipe := Recipe{
		ID:           deterministicUUID(testRecipeID),
		Name:         "Updated Recipe",
		RestaurantID: deterministicUUID(testRestaurantID),
		Instructions: "Updated instructions.",
		Yield:        8,
		UpdatedAt:    1767229300,
	}

	if err := UpdateRecipe(db, &recipe); err != nil {
		t.Fatalf("UpdateRecipe returned error: %v", err)
	}

	storedRecipe, err := GetRecipeByID(db, recipe.ID)
	if err != nil {
		t.Fatalf("GetRecipeByID returned error: %v", err)
	}
	if storedRecipe == nil {
		t.Fatal("GetRecipeByID returned nil recipe")
	}

	assertRecipeEqual(t, *storedRecipe, Recipe{
		ID:           deterministicUUID(testRecipeID),
		Name:         "Updated Recipe",
		RestaurantID: deterministicUUID(testRestaurantID),
		Instructions: "Updated instructions.",
		Yield:        8,
		CreatedAt:    expectedRecipeTimestamp,
		UpdatedAt:    1767229300,
	})
}

func TestUpdateRecipe_NotFound(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	recipe := Recipe{
		ID:           "nonexistent",
		Name:         "Missing Recipe",
		Instructions: "No instructions.",
		Yield:        2,
		UpdatedAt:    1767229300,
	}

	err := UpdateRecipe(db, &recipe)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
}

func TestDeleteRecipe(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	if err := DeleteRecipe(db, deterministicUUID(testRecipeID)); err != nil {
		t.Fatalf("DeleteRecipe returned error: %v", err)
	}

	recipe, err := GetRecipeByID(db, deterministicUUID(testRecipeID))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if recipe != nil {
		t.Fatalf("recipe = %#v, want nil", recipe)
	}

	ingredients, err := ListIngredientsByRecipeID(db, deterministicUUID(testRecipeID))
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID returned error: %v", err)
	}
	if ingredients == nil {
		t.Fatal("expected non-nil ingredients slice")
	}
	if len(ingredients) != 0 {
		t.Fatalf("len(ingredients) = %d, want 0", len(ingredients))
	}
}

func TestDeleteRecipe_NotFound(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	err := DeleteRecipe(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
}

func assertRecipeEqual(t *testing.T, got, want Recipe) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.RestaurantID != want.RestaurantID {
		t.Fatalf("RestaurantID = %q, want %q", got.RestaurantID, want.RestaurantID)
	}
	if got.Instructions != want.Instructions {
		t.Fatalf("Instructions = %q, want %q", got.Instructions, want.Instructions)
	}
	if got.Yield != want.Yield {
		t.Fatalf("Yield = %d, want %d", got.Yield, want.Yield)
	}
	if got.CreatedAt != want.CreatedAt {
		t.Fatalf("CreatedAt = %d, want %d", got.CreatedAt, want.CreatedAt)
	}
	if got.UpdatedAt != want.UpdatedAt {
		t.Fatalf("UpdatedAt = %d, want %d", got.UpdatedAt, want.UpdatedAt)
	}
}

type recordingSearchRecipesDB struct {
	query string
	err   error
}

func (db *recordingSearchRecipesDB) Exec(query string, args ...any) (sql.Result, error) {
	panic("unexpected Exec call")
}

func (db *recordingSearchRecipesDB) Query(query string, args ...any) (*sql.Rows, error) {
	db.query = query
	return nil, db.err
}

func (db *recordingSearchRecipesDB) QueryRow(query string, args ...any) *sql.Row {
	panic("unexpected QueryRow call")
}
