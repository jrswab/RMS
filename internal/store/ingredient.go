package store

import "fmt"

func ListIngredientsByRecipeID(db DBTX, recipeID string) ([]Ingredient, error) {
	rows, err := db.Query(
		`SELECT id, recipe_id, food_id, quantity, unit, sort_order FROM ingredients WHERE recipe_id = ? ORDER BY sort_order`,
		recipeID,
	)
	if err != nil {
		return nil, fmt.Errorf("query ingredients for recipe %q: %w", recipeID, err)
	}
	defer rows.Close()

	ingredients := make([]Ingredient, 0)
	for rows.Next() {
		var ingredient Ingredient
		if err := rows.Scan(
			&ingredient.ID,
			&ingredient.RecipeID,
			&ingredient.FoodID,
			&ingredient.Quantity,
			&ingredient.Unit,
			&ingredient.SortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan ingredient for recipe %q: %w", recipeID, err)
		}

		ingredients = append(ingredients, ingredient)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ingredients for recipe %q: %w", recipeID, err)
	}

	return ingredients, nil
}

func ReplaceIngredients(db DBTX, recipeID string, ingredients []Ingredient) error {
	if _, err := db.Exec(`DELETE FROM ingredients WHERE recipe_id = ?`, recipeID); err != nil {
		return fmt.Errorf("delete ingredients for recipe %q: %w", recipeID, err)
	}

	for _, ingredient := range ingredients {
		if _, err := db.Exec(
			`INSERT INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
			ingredient.ID,
			ingredient.RecipeID,
			ingredient.FoodID,
			ingredient.Quantity,
			ingredient.Unit,
			ingredient.SortOrder,
		); err != nil {
			return fmt.Errorf("insert ingredient %q for recipe %q: %w", ingredient.ID, recipeID, err)
		}
	}

	return nil
}
