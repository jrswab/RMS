package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func GetRecipeByID(db DBTX, id string) (*Recipe, error) {
	recipe := &Recipe{}

	err := db.QueryRow(
		`SELECT id, name, restaurant_id, instructions, yield, created_at, updated_at FROM recipes WHERE id = ?`,
		id,
	).Scan(
		&recipe.ID,
		&recipe.Name,
		&recipe.RestaurantID,
		&recipe.Instructions,
		&recipe.Yield,
		&recipe.CreatedAt,
		&recipe.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get recipe by id %q: %w", id, err)
	}

	return recipe, nil
}

func SearchRecipes(db DBTX, restaurantID string, query string) ([]Recipe, error) {
	escapedQuery := strings.ReplaceAll(query, `\`, `\\`)
	escapedQuery = strings.ReplaceAll(escapedQuery, `%`, `\%`)
	escapedQuery = strings.ReplaceAll(escapedQuery, `_`, `\_`)

	rows, err := db.Query(
		`SELECT id, name, restaurant_id, instructions, yield, created_at, updated_at FROM recipes WHERE restaurant_id = ? AND name LIKE ? ESCAPE '\' ORDER BY name`,
		restaurantID,
		"%"+escapedQuery+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search recipes for restaurant %q with query %q: %w", restaurantID, query, err)
	}
	defer rows.Close()

	recipes := make([]Recipe, 0)
	for rows.Next() {
		var recipe Recipe
		if err := rows.Scan(
			&recipe.ID,
			&recipe.Name,
			&recipe.RestaurantID,
			&recipe.Instructions,
			&recipe.Yield,
			&recipe.CreatedAt,
			&recipe.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan recipe for restaurant %q with query %q: %w", restaurantID, query, err)
		}

		recipes = append(recipes, recipe)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes for restaurant %q with query %q: %w", restaurantID, query, err)
	}

	return recipes, nil
}

func CreateRecipe(db DBTX, r *Recipe) error {
	if _, err := db.Exec(
		`INSERT INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID,
		r.Name,
		r.RestaurantID,
		r.Instructions,
		r.Yield,
		r.CreatedAt,
		r.UpdatedAt,
	); err != nil {
		return fmt.Errorf("create recipe %q: %w", r.ID, err)
	}

	return nil
}

func UpdateRecipe(db DBTX, r *Recipe) error {
	result, err := db.Exec(
		`UPDATE recipes SET name = ?, instructions = ?, "yield" = ?, updated_at = ? WHERE id = ?`,
		r.Name,
		r.Instructions,
		r.Yield,
		r.UpdatedAt,
		r.ID,
	)
	if err != nil {
		return fmt.Errorf("update recipe %q: %w", r.ID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected updating recipe %q: %w", r.ID, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func DeleteRecipe(db DBTX, id string) error {
	result, err := db.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete recipe %q: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected deleting recipe %q: %w", id, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
