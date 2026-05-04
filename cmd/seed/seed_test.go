package main

import (
	"database/sql"
	"testing"

	"lab37/internal/db"
	"lab37/migrations"
)

type seedUserRestaurantLink struct {
	username       string
	restaurantName string
}

type seedRecipeData struct {
	restaurantName  string
	ingredientCount int
}

func TestSeed(t *testing.T) {
	database := openSeedTestDB(t)

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		t.Fatalf("db.RunMigrations() error = %v", err)
	}

	if err := Seed(database); err != nil {
		t.Fatalf("Seed() error = %v", err)
	}

	wantUsers := map[string]string{
		"admin":   "admin",
		"manager": "manager",
		"staff":   "staff",
	}
	gotUsers := userRolesByUsername(t, database)
	if len(gotUsers) != len(wantUsers) {
		t.Fatalf("users row count = %d, want %d", len(gotUsers), len(wantUsers))
	}

	for username, wantRole := range wantUsers {
		gotRole, ok := gotUsers[username]
		if !ok {
			t.Fatalf("user %q not found", username)
		}

		if gotRole != wantRole {
			t.Fatalf("user %q role = %q, want %q", username, gotRole, wantRole)
		}
	}

	wantRestaurants := []string{"Copper Kettle", "The Rusty Spoon"}
	gotRestaurants := restaurantNames(t, database)
	if len(gotRestaurants) != len(wantRestaurants) {
		t.Fatalf("restaurants row count = %d, want %d", len(gotRestaurants), len(wantRestaurants))
	}

	for i, wantRestaurant := range wantRestaurants {
		if gotRestaurants[i] != wantRestaurant {
			t.Fatalf("restaurant[%d] = %q, want %q", i, gotRestaurants[i], wantRestaurant)
		}
	}

	wantLinks := []seedUserRestaurantLink{
		{username: "admin", restaurantName: "Copper Kettle"},
		{username: "admin", restaurantName: "The Rusty Spoon"},
		{username: "manager", restaurantName: "The Rusty Spoon"},
		{username: "staff", restaurantName: "The Rusty Spoon"},
	}
	gotLinks := userRestaurantLinks(t, database)
	if len(gotLinks) != len(wantLinks) {
		t.Fatalf("user_restaurants row count = %d, want %d", len(gotLinks), len(wantLinks))
	}

	for i, wantLink := range wantLinks {
		if gotLinks[i] != wantLink {
			t.Fatalf("user_restaurants[%d] = %+v, want %+v", i, gotLinks[i], wantLink)
		}
	}

	if got := rowCount(t, database, "food"); got != 8 {
		t.Fatalf("food row count = %d, want 8", got)
	}

	wantRecipes := map[string]seedRecipeData{
		"Butter Chicken":  {restaurantName: "The Rusty Spoon", ingredientCount: 5},
		"Simple Pancakes": {restaurantName: "The Rusty Spoon", ingredientCount: 6},
		"Fried Rice":      {restaurantName: "The Rusty Spoon", ingredientCount: 4},
	}
	gotRecipes := recipeDataByName(t, database)
	if len(gotRecipes) != len(wantRecipes) {
		t.Fatalf("recipes row count = %d, want %d", len(gotRecipes), len(wantRecipes))
	}

	for recipeName, wantRecipe := range wantRecipes {
		gotRecipe, ok := gotRecipes[recipeName]
		if !ok {
			t.Fatalf("recipe %q not found", recipeName)
		}

		if gotRecipe.restaurantName != wantRecipe.restaurantName {
			t.Fatalf("recipe %q restaurant = %q, want %q", recipeName, gotRecipe.restaurantName, wantRecipe.restaurantName)
		}

		if gotRecipe.ingredientCount != wantRecipe.ingredientCount {
			t.Fatalf("recipe %q ingredient count = %d, want %d", recipeName, gotRecipe.ingredientCount, wantRecipe.ingredientCount)
		}
	}
}

func TestSeed_Idempotent(t *testing.T) {
	database := openSeedTestDB(t)

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		t.Fatalf("db.RunMigrations() error = %v", err)
	}

	if err := Seed(database); err != nil {
		t.Fatalf("first Seed() error = %v", err)
	}

	tables := []string{"users", "restaurants", "user_restaurants", "food", "recipes", "ingredients"}
	firstCounts := tableCounts(t, database, tables)
	wantCounts := map[string]int{
		"users":            3,
		"restaurants":      2,
		"user_restaurants": 4,
		"food":             8,
		"recipes":          3,
		"ingredients":      15,
	}

	assertTableCounts(t, firstCounts, wantCounts, "after first Seed()")

	if err := Seed(database); err != nil {
		t.Fatalf("second Seed() error = %v", err)
	}

	secondCounts := tableCounts(t, database, tables)
	assertTableCounts(t, secondCounts, wantCounts, "after second Seed()")

	for _, tableName := range tables {
		if secondCounts[tableName] != firstCounts[tableName] {
			t.Fatalf("%s row count changed after reseed: got %d, want %d", tableName, secondCounts[tableName], firstCounts[tableName])
		}
	}
}

func openSeedTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open(:memory:) error = %v", err)
	}

	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	})

	return database
}

func userRolesByUsername(t *testing.T, database *sql.DB) map[string]string {
	t.Helper()

	rows, err := database.Query(`SELECT username, role FROM users ORDER BY username`)
	if err != nil {
		t.Fatalf("query users error = %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("rows.Close() for users error = %v", err)
		}
	}()

	users := make(map[string]string)
	for rows.Next() {
		var username string
		var role string
		if err := rows.Scan(&username, &role); err != nil {
			t.Fatalf("scan user row error = %v", err)
		}

		users[username] = role
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate user rows error = %v", err)
	}

	return users
}

func restaurantNames(t *testing.T, database *sql.DB) []string {
	t.Helper()

	rows, err := database.Query(`SELECT name FROM restaurants ORDER BY name`)
	if err != nil {
		t.Fatalf("query restaurants error = %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("rows.Close() for restaurants error = %v", err)
		}
	}()

	var restaurants []string
	for rows.Next() {
		var restaurantName string
		if err := rows.Scan(&restaurantName); err != nil {
			t.Fatalf("scan restaurant row error = %v", err)
		}

		restaurants = append(restaurants, restaurantName)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate restaurant rows error = %v", err)
	}

	return restaurants
}

func userRestaurantLinks(t *testing.T, database *sql.DB) []seedUserRestaurantLink {
	t.Helper()

	rows, err := database.Query(`
		SELECT u.username, r.name
		FROM user_restaurants ur
		JOIN users u ON u.id = ur.user_id
		JOIN restaurants r ON r.id = ur.restaurant_id
		ORDER BY u.username, r.name
	`)
	if err != nil {
		t.Fatalf("query user_restaurants error = %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("rows.Close() for user_restaurants error = %v", err)
		}
	}()

	var links []seedUserRestaurantLink
	for rows.Next() {
		var link seedUserRestaurantLink
		if err := rows.Scan(&link.username, &link.restaurantName); err != nil {
			t.Fatalf("scan user_restaurants row error = %v", err)
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate user_restaurants rows error = %v", err)
	}

	return links
}

func recipeDataByName(t *testing.T, database *sql.DB) map[string]seedRecipeData {
	t.Helper()

	rows, err := database.Query(`
		SELECT r.name, restaurants.name, COUNT(i.id)
		FROM recipes r
		JOIN restaurants ON restaurants.id = r.restaurant_id
		LEFT JOIN ingredients i ON i.recipe_id = r.id
		GROUP BY r.id, r.name, restaurants.name
	`)
	if err != nil {
		t.Fatalf("query recipes error = %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("rows.Close() for recipes error = %v", err)
		}
	}()

	recipes := make(map[string]seedRecipeData)
	for rows.Next() {
		var recipeName string
		var recipe seedRecipeData
		if err := rows.Scan(&recipeName, &recipe.restaurantName, &recipe.ingredientCount); err != nil {
			t.Fatalf("scan recipe row error = %v", err)
		}

		recipes[recipeName] = recipe
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate recipe rows error = %v", err)
	}

	return recipes
}

func tableCounts(t *testing.T, database *sql.DB, tableNames []string) map[string]int {
	t.Helper()

	counts := make(map[string]int, len(tableNames))
	for _, tableName := range tableNames {
		counts[tableName] = rowCount(t, database, tableName)
	}

	return counts
}

func assertTableCounts(t *testing.T, gotCounts map[string]int, wantCounts map[string]int, context string) {
	t.Helper()

	for tableName, wantCount := range wantCounts {
		gotCount, ok := gotCounts[tableName]
		if !ok {
			t.Fatalf("missing row count for %s %s", tableName, context)
		}

		if gotCount != wantCount {
			t.Fatalf("%s row count %s = %d, want %d", tableName, context, gotCount, wantCount)
		}
	}
}

func rowCount(t *testing.T, database *sql.DB, tableName string) int {
	t.Helper()

	var count int
	query := "SELECT COUNT(*) FROM " + tableName
	if err := database.QueryRow(query).Scan(&count); err != nil {
		t.Fatalf("query %s row count error = %v", tableName, err)
	}

	return count
}
