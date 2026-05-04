package store

import "fmt"

func ListUserRestaurantIDs(db DBTX, userID string) ([]string, error) {
	rows, err := db.Query(`SELECT restaurant_id FROM user_restaurants WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user_restaurants for user %q: %w", userID, err)
	}
	defer rows.Close()

	restaurantIDs := make([]string, 0)
	for rows.Next() {
		var restaurantID string
		if err := rows.Scan(&restaurantID); err != nil {
			return nil, fmt.Errorf("scan user_restaurants row for user %q: %w", userID, err)
		}

		restaurantIDs = append(restaurantIDs, restaurantID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user_restaurants rows for user %q: %w", userID, err)
	}

	return restaurantIDs, nil
}
