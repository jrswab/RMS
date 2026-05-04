package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"lab37/internal/db"
	"lab37/migrations"
)

const (
	testUserID         = "user:testuser"
	testUsername       = "testuser"
	testPassword       = "testpass"
	testRole           = "staff"
	testRestaurantID   = "restaurant:test-restaurant"
	testRestaurantName = "Test Restaurant"
	testFoodID         = "food:flour"
	testFoodName       = "Flour"
	testRecipeID       = "recipe:test-recipe"
	testRecipeName     = "Test Recipe"
	testIngredientID   = "ingredient:test-recipe:01:flour"
)

func deterministicUUID(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(name)).String()
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		t.Fatal(err)
	}

	return database
}

func seedTestData(t *testing.T, database *sql.DB) {
	t.Helper()

	fixedTimestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		deterministicUUID(testUserID),
		testUsername,
		string(passwordHash),
		testRole,
		fixedTimestamp,
		fixedTimestamp,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO restaurants (id, name) VALUES (?, ?)`,
		deterministicUUID(testRestaurantID),
		testRestaurantName,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`,
		deterministicUUID(testUserID),
		deterministicUUID(testRestaurantID),
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO food (id, name) VALUES (?, ?)`,
		deterministicUUID(testFoodID),
		testFoodName,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		deterministicUUID(testRecipeID),
		testRecipeName,
		deterministicUUID(testRestaurantID),
		"Mix ingredients and bake.",
		4,
		fixedTimestamp,
		fixedTimestamp,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(
		`INSERT OR IGNORE INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
		deterministicUUID(testIngredientID),
		deterministicUUID(testRecipeID),
		deterministicUUID(testFoodID),
		1.0,
		"cup",
		1,
	); err != nil {
		t.Fatal(err)
	}
}
