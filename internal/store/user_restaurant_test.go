package store

import "testing"

func TestListUserRestaurantIDs(t *testing.T) {
	db := setupTestDB(t)

	userID := deterministicUUID("user:list-user")
	restaurantIDOne := deterministicUUID("restaurant:list-user-restaurant-1")
	restaurantIDTwo := deterministicUUID("restaurant:list-user-restaurant-2")
	const fixedTimestamp = int64(1735689600)

	if _, err := db.Exec(
		`INSERT INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID,
		"list-user",
		"hashed-password",
		"staff",
		fixedTimestamp,
		fixedTimestamp,
	); err != nil {
		t.Fatalf("insert user error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, restaurantIDOne, "List Restaurant One"); err != nil {
		t.Fatalf("insert first restaurant error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, restaurantIDTwo, "List Restaurant Two"); err != nil {
		t.Fatalf("insert second restaurant error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`, userID, restaurantIDOne); err != nil {
		t.Fatalf("insert first user_restaurants row error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`, userID, restaurantIDTwo); err != nil {
		t.Fatalf("insert second user_restaurants row error = %v", err)
	}

	got, err := ListUserRestaurantIDs(db, userID)
	if err != nil {
		t.Fatalf("ListUserRestaurantIDs() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(ListUserRestaurantIDs()) = %d, want 2", len(got))
	}

	gotIDs := make(map[string]bool, len(got))
	for _, restaurantID := range got {
		gotIDs[restaurantID] = true
	}

	if !gotIDs[restaurantIDOne] {
		t.Fatalf("first restaurant ID %q not found in results %v", restaurantIDOne, got)
	}

	if !gotIDs[restaurantIDTwo] {
		t.Fatalf("second restaurant ID %q not found in results %v", restaurantIDTwo, got)
	}
}

func TestListUserRestaurantIDs_NoResults(t *testing.T) {
	db := setupTestDB(t)

	got, err := ListUserRestaurantIDs(db, "nonexistent")
	if err != nil {
		t.Fatalf("ListUserRestaurantIDs() error = %v", err)
	}

	if got == nil {
		t.Fatal("ListUserRestaurantIDs() returned nil slice, want empty slice")
	}

	if len(got) != 0 {
		t.Fatalf("len(ListUserRestaurantIDs()) = %d, want 0", len(got))
	}
}
