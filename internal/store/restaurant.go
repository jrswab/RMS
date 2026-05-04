package store

import (
	"database/sql"
	"errors"
	"fmt"
)

func GetRestaurantByID(db DBTX, id string) (*Restaurant, error) {
	restaurant := &Restaurant{}

	err := db.QueryRow(`SELECT id, name FROM restaurants WHERE id = ?`, id).Scan(&restaurant.ID, &restaurant.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get restaurant by id %q: %w", id, err)
	}

	return restaurant, nil
}

func ListRestaurantsByUserID(db DBTX, userID string) ([]Restaurant, error) {
	rows, err := db.Query(
		`SELECT r.id, r.name FROM restaurants r INNER JOIN user_restaurants ur ON r.id = ur.restaurant_id WHERE ur.user_id = ?`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list restaurants by user id %q: %w", userID, err)
	}
	defer rows.Close()

	restaurants := make([]Restaurant, 0)
	for rows.Next() {
		var restaurant Restaurant

		if err := rows.Scan(&restaurant.ID, &restaurant.Name); err != nil {
			return nil, fmt.Errorf("scan restaurant for user id %q: %w", userID, err)
		}

		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate restaurants for user id %q: %w", userID, err)
	}

	return restaurants, nil
}
