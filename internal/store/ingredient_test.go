package store

import "testing"

func TestListIngredientsByRecipeID(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	seedTestData(t, db)

	recipeID := deterministicUUID(testRecipeID)
	secondIngredient := Ingredient{
		ID:        deterministicUUID("ingredient:test-recipe:02:sugar"),
		RecipeID:  recipeID,
		FoodID:    deterministicUUID("food:sugar"),
		Quantity:  2.0,
		Unit:      "tbsp",
		SortOrder: 2,
	}
	thirdIngredient := Ingredient{
		ID:        deterministicUUID("ingredient:test-recipe:03:salt"),
		RecipeID:  recipeID,
		FoodID:    deterministicUUID("food:salt"),
		Quantity:  0.5,
		Unit:      "tsp",
		SortOrder: 3,
	}

	insertFood(t, db, secondIngredient.FoodID, "Sugar")
	insertFood(t, db, thirdIngredient.FoodID, "Salt")
	insertIngredient(t, db, secondIngredient)
	insertIngredient(t, db, thirdIngredient)

	ingredients, err := ListIngredientsByRecipeID(db, deterministicUUID(testRecipeID))
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID returned error: %v", err)
	}

	if len(ingredients) != 3 {
		t.Fatalf("expected 3 ingredients, got %d", len(ingredients))
	}

	assertIngredientEqual(t, ingredients[0], Ingredient{
		ID:        deterministicUUID(testIngredientID),
		RecipeID:  recipeID,
		FoodID:    deterministicUUID(testFoodID),
		Quantity:  1.0,
		Unit:      "cup",
		SortOrder: 1,
	})
	assertIngredientEqual(t, ingredients[1], secondIngredient)
	assertIngredientEqual(t, ingredients[2], thirdIngredient)
}

func TestListIngredientsByRecipeID_NoResults(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	seedTestData(t, db)

	recipeID := deterministicUUID("recipe:no-ingredients")
	insertRecipe(t, db, recipeID, "No Ingredients Recipe")

	ingredients, err := ListIngredientsByRecipeID(db, deterministicUUID("recipe:no-ingredients"))
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID returned error: %v", err)
	}

	if ingredients == nil {
		t.Fatal("expected non-nil ingredients slice")
	}

	if len(ingredients) != 0 {
		t.Fatalf("expected 0 ingredients, got %d", len(ingredients))
	}
}

func TestReplaceIngredients(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	seedTestData(t, db)

	recipeID := deterministicUUID(testRecipeID)
	replacementIngredients := []Ingredient{
		{
			ID:        deterministicUUID("ingredient:test-recipe:01:sugar"),
			RecipeID:  recipeID,
			FoodID:    deterministicUUID("food:sugar"),
			Quantity:  2.5,
			Unit:      "cups",
			SortOrder: 1,
		},
		{
			ID:        deterministicUUID("ingredient:test-recipe:02:butter"),
			RecipeID:  recipeID,
			FoodID:    deterministicUUID("food:butter"),
			Quantity:  3.0,
			Unit:      "tbsp",
			SortOrder: 2,
		},
		{
			ID:        deterministicUUID("ingredient:test-recipe:03:eggs"),
			RecipeID:  recipeID,
			FoodID:    deterministicUUID("food:eggs"),
			Quantity:  2.0,
			Unit:      "count",
			SortOrder: 3,
		},
	}

	insertFood(t, db, replacementIngredients[0].FoodID, "Sugar")
	insertFood(t, db, replacementIngredients[1].FoodID, "Butter")
	insertFood(t, db, replacementIngredients[2].FoodID, "Eggs")

	if err := ReplaceIngredients(db, deterministicUUID(testRecipeID), replacementIngredients); err != nil {
		t.Fatalf("ReplaceIngredients returned error: %v", err)
	}

	ingredients, err := ListIngredientsByRecipeID(db, deterministicUUID(testRecipeID))
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID returned error: %v", err)
	}

	if len(ingredients) != 3 {
		t.Fatalf("expected 3 ingredients, got %d", len(ingredients))
	}

	assertIngredientEqual(t, ingredients[0], replacementIngredients[0])
	assertIngredientEqual(t, ingredients[1], replacementIngredients[1])
	assertIngredientEqual(t, ingredients[2], replacementIngredients[2])
}

func TestReplaceIngredients_Empty(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	seedTestData(t, db)

	if err := ReplaceIngredients(db, deterministicUUID(testRecipeID), []Ingredient{}); err != nil {
		t.Fatalf("ReplaceIngredients returned error: %v", err)
	}

	ingredients, err := ListIngredientsByRecipeID(db, deterministicUUID(testRecipeID))
	if err != nil {
		t.Fatalf("ListIngredientsByRecipeID returned error: %v", err)
	}

	if len(ingredients) != 0 {
		t.Fatalf("expected 0 ingredients, got %d", len(ingredients))
	}
}

func insertFood(t *testing.T, db DBTX, id, name string) {
	t.Helper()

	if _, err := db.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, id, name); err != nil {
		t.Fatal(err)
	}
}

func insertRecipe(t *testing.T, db DBTX, id, name string) {
	t.Helper()

	if _, err := db.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id,
		name,
		deterministicUUID(testRestaurantID),
		"No ingredients yet.",
		1,
		1,
		1,
	); err != nil {
		t.Fatal(err)
	}
}

func insertIngredient(t *testing.T, db DBTX, ingredient Ingredient) {
	t.Helper()

	if _, err := db.Exec(
		`INSERT INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
		ingredient.ID,
		ingredient.RecipeID,
		ingredient.FoodID,
		ingredient.Quantity,
		ingredient.Unit,
		ingredient.SortOrder,
	); err != nil {
		t.Fatal(err)
	}
}

func assertIngredientEqual(t *testing.T, got, want Ingredient) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("expected ingredient ID %q, got %q", want.ID, got.ID)
	}

	if got.RecipeID != want.RecipeID {
		t.Fatalf("expected ingredient RecipeID %q, got %q", want.RecipeID, got.RecipeID)
	}

	if got.FoodID != want.FoodID {
		t.Fatalf("expected ingredient FoodID %q, got %q", want.FoodID, got.FoodID)
	}

	if got.Quantity != want.Quantity {
		t.Fatalf("expected ingredient Quantity %v, got %v", want.Quantity, got.Quantity)
	}

	if got.Unit != want.Unit {
		t.Fatalf("expected ingredient Unit %q, got %q", want.Unit, got.Unit)
	}

	if got.SortOrder != want.SortOrder {
		t.Fatalf("expected ingredient SortOrder %d, got %d", want.SortOrder, got.SortOrder)
	}
}
