package store

import (
	"errors"
	"testing"
)

func TestGetRestaurantByID(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	wantID := deterministicUUID(testRestaurantID)

	restaurant, err := GetRestaurantByID(db, wantID)
	if err != nil {
		t.Fatalf("GetRestaurantByID returned error: %v", err)
	}

	if restaurant == nil {
		t.Fatal("GetRestaurantByID returned nil restaurant")
	}

	if restaurant.ID != wantID {
		t.Errorf("restaurant.ID = %q, want %q", restaurant.ID, wantID)
	}

	if restaurant.Name != testRestaurantName {
		t.Errorf("restaurant.Name = %q, want %q", restaurant.Name, testRestaurantName)
	}
}

func TestGetRestaurantByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	restaurant, err := GetRestaurantByID(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetRestaurantByID error = %v, want %v", err, ErrNotFound)
	}

	if restaurant != nil {
		t.Fatalf("GetRestaurantByID returned %#v, want nil", restaurant)
	}
}

func TestListRestaurantsByUserID(t *testing.T) {
	db := setupTestDB(t)

	userID := deterministicUUID("user:list-restaurants")
	restaurantOneID := deterministicUUID("restaurant:list-restaurants:one")
	restaurantTwoID := deterministicUUID("restaurant:list-restaurants:two")

	if _, err := db.Exec(
		`INSERT INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID,
		"list-restaurants-user",
		"password-hash",
		"staff",
		0,
		0,
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, restaurantOneID, "Restaurant One"); err != nil {
		t.Fatalf("insert first restaurant: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, restaurantTwoID, "Restaurant Two"); err != nil {
		t.Fatalf("insert second restaurant: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`, userID, restaurantOneID); err != nil {
		t.Fatalf("insert first user_restaurant link: %v", err)
	}

	if _, err := db.Exec(`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`, userID, restaurantTwoID); err != nil {
		t.Fatalf("insert second user_restaurant link: %v", err)
	}

	restaurants, err := ListRestaurantsByUserID(db, userID)
	if err != nil {
		t.Fatalf("ListRestaurantsByUserID returned error: %v", err)
	}

	if len(restaurants) != 2 {
		t.Fatalf("len(restaurants) = %d, want 2", len(restaurants))
	}
}

func TestListRestaurantsByUserID_NoResults(t *testing.T) {
	db := setupTestDB(t)

	restaurants, err := ListRestaurantsByUserID(db, "nonexistent")
	if err != nil {
		t.Fatalf("ListRestaurantsByUserID returned error: %v", err)
	}

	if len(restaurants) != 0 {
		t.Fatalf("len(restaurants) = %d, want 0", len(restaurants))
	}

	if restaurants == nil {
		t.Error("restaurants = nil, want empty slice")
	}
}
