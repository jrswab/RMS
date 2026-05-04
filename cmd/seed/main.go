package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"lab37/internal/db"
	"lab37/migrations"
)

var seedTimestamp = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

type seedUser struct {
	id       string
	username string
	role     string
	password string
}

type seedRestaurant struct {
	id   string
	name string
}

type seedFood struct {
	id   string
	name string
}

type seedIngredient struct {
	foodName string
	quantity float64
	unit     string
}

type seedRecipe struct {
	id           string
	name         string
	restaurantID string
	instructions string
	yieldSize    int
	ingredients  []seedIngredient
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run() error {
	database, err := db.Open(db.DefaultPath())
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	if err := db.RunMigrations(database, migrations.FS); err != nil {
		return err
	}

	if err := Seed(database); err != nil {
		return err
	}

	fmt.Println("Seed data inserted successfully")

	return nil
}

// Seed inserts deterministic test data for local development and tests.
func Seed(database *sql.DB) error {
	// NOTE: Hardcoded development-only password. Never use in production.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	users := []seedUser{
		{
			id:       deterministicUUID("user:admin"),
			username: "admin",
			role:     "admin",
			password: string(passwordHash),
		},
		{
			id:       deterministicUUID("user:manager"),
			username: "manager",
			role:     "manager",
			password: string(passwordHash),
		},
		{
			id:       deterministicUUID("user:staff"),
			username: "staff",
			role:     "staff",
			password: string(passwordHash),
		},
	}

	restaurants := []seedRestaurant{
		{
			id:   deterministicUUID("restaurant:the-rusty-spoon"),
			name: "The Rusty Spoon",
		},
		{
			id:   deterministicUUID("restaurant:copper-kettle"),
			name: "Copper Kettle",
		},
	}

	foodItems := []seedFood{
		{id: deterministicUUID("food:flour"), name: "flour"},
		{id: deterministicUUID("food:sugar"), name: "sugar"},
		{id: deterministicUUID("food:butter"), name: "butter"},
		{id: deterministicUUID("food:eggs"), name: "eggs"},
		{id: deterministicUUID("food:salt"), name: "salt"},
		{id: deterministicUUID("food:milk"), name: "milk"},
		{id: deterministicUUID("food:chicken"), name: "chicken"},
		{id: deterministicUUID("food:rice"), name: "rice"},
	}

	rustySpoonID := deterministicUUID("restaurant:the-rusty-spoon")
	recipes := []seedRecipe{
		{
			id:           deterministicUUID("recipe:butter-chicken"),
			name:         "Butter Chicken",
			restaurantID: rustySpoonID,
			instructions: "Brown the chicken in butter, season with salt, stir in milk for a quick sauce, and serve over rice.",
			yieldSize:    4,
			ingredients: []seedIngredient{
				{foodName: "chicken", quantity: 2, unit: "lb"},
				{foodName: "butter", quantity: 0.5, unit: "cup"},
				{foodName: "salt", quantity: 2, unit: "tsp"},
				{foodName: "milk", quantity: 1, unit: "cup"},
				{foodName: "rice", quantity: 3, unit: "cup"},
			},
		},
		{
			id:           deterministicUUID("recipe:simple-pancakes"),
			name:         "Simple Pancakes",
			restaurantID: rustySpoonID,
			instructions: "Whisk the dry ingredients, mix in the eggs, milk, and melted butter, then cook on a hot griddle until golden.",
			yieldSize:    4,
			ingredients: []seedIngredient{
				{foodName: "flour", quantity: 2, unit: "cup"},
				{foodName: "eggs", quantity: 2, unit: "each"},
				{foodName: "milk", quantity: 1.5, unit: "cup"},
				{foodName: "butter", quantity: 0.25, unit: "cup"},
				{foodName: "sugar", quantity: 2, unit: "tbsp"},
				{foodName: "salt", quantity: 1, unit: "tsp"},
			},
		},
		{
			id:           deterministicUUID("recipe:fried-rice"),
			name:         "Fried Rice",
			restaurantID: rustySpoonID,
			instructions: "Cook the chicken, scramble the eggs, stir-fry with rice and salt, and cook until hot all the way through.",
			yieldSize:    4,
			ingredients: []seedIngredient{
				{foodName: "rice", quantity: 4, unit: "cup"},
				{foodName: "eggs", quantity: 2, unit: "each"},
				{foodName: "chicken", quantity: 1, unit: "lb"},
				{foodName: "salt", quantity: 1, unit: "tsp"},
			},
		},
	}

	foodIDsByName := make(map[string]string, len(foodItems))
	for _, foodItem := range foodItems {
		foodIDsByName[foodItem.name] = foodItem.id
	}

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, user := range users {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO users (id, username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			user.id,
			user.username,
			user.password,
			user.role,
			seedTimestamp,
			seedTimestamp,
		); err != nil {
			return fmt.Errorf("insert user %q: %w", user.username, err)
		}
	}

	for _, restaurant := range restaurants {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO restaurants (id, name) VALUES (?, ?)`,
			restaurant.id,
			restaurant.name,
		); err != nil {
			return fmt.Errorf("insert restaurant %q: %w", restaurant.name, err)
		}
	}

	for _, foodItem := range foodItems {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO food (id, name) VALUES (?, ?)`,
			foodItem.id,
			foodItem.name,
		); err != nil {
			return fmt.Errorf("insert food %q: %w", foodItem.name, err)
		}
	}

	userRestaurantLinks := []struct {
		userID       string
		restaurantID string
	}{
		{userID: deterministicUUID("user:admin"), restaurantID: deterministicUUID("restaurant:the-rusty-spoon")},
		{userID: deterministicUUID("user:admin"), restaurantID: deterministicUUID("restaurant:copper-kettle")},
		{userID: deterministicUUID("user:manager"), restaurantID: deterministicUUID("restaurant:the-rusty-spoon")},
		{userID: deterministicUUID("user:staff"), restaurantID: deterministicUUID("restaurant:the-rusty-spoon")},
	}

	for _, link := range userRestaurantLinks {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`,
			link.userID,
			link.restaurantID,
		); err != nil {
			return fmt.Errorf("insert user_restaurant link %q -> %q: %w", link.userID, link.restaurantID, err)
		}
	}

	for _, recipe := range recipes {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO recipes (id, name, restaurant_id, instructions, "yield", created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			recipe.id,
			recipe.name,
			recipe.restaurantID,
			recipe.instructions,
			recipe.yieldSize,
			seedTimestamp,
			seedTimestamp,
		); err != nil {
			return fmt.Errorf("insert recipe %q: %w", recipe.name, err)
		}

		for sortOrder, ingredient := range recipe.ingredients {
			foodID, ok := foodIDsByName[ingredient.foodName]
			if !ok {
				return fmt.Errorf("missing food id for ingredient %q in recipe %q", ingredient.foodName, recipe.name)
			}

			ingredientID := deterministicUUID(fmt.Sprintf("ingredient:%s:%02d:%s", recipe.name, sortOrder+1, ingredient.foodName))
			if _, err := tx.Exec(
				`INSERT OR IGNORE INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
				ingredientID,
				recipe.id,
				foodID,
				ingredient.quantity,
				ingredient.unit,
				sortOrder+1,
			); err != nil {
				return fmt.Errorf("insert ingredient %q for recipe %q: %w", ingredient.foodName, recipe.name, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}

	committed = true

	return nil
}

func deterministicUUID(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(name)).String()
}
