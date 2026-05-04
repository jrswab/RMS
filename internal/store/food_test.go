package store

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestGetFoodByID(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	food, err := GetFoodByID(db, deterministicUUID(testFoodID))
	if err != nil {
		t.Fatalf("GetFoodByID returned error: %v", err)
	}

	if food == nil {
		t.Fatal("expected food, got nil")
	}

	if food.ID != deterministicUUID(testFoodID) {
		t.Fatalf("expected ID %q, got %q", deterministicUUID(testFoodID), food.ID)
	}

	if food.Name != testFoodName {
		t.Fatalf("expected name %q, got %q", testFoodName, food.Name)
	}
}

func TestGetFoodByID_NotFound(t *testing.T) {
	db := setupTestDB(t)

	food, err := GetFoodByID(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if food != nil {
		t.Fatalf("expected nil food, got %#v", food)
	}
}

func TestListFood(t *testing.T) {
	db := setupTestDB(t)

	rows := []struct {
		id   string
		name string
	}{
		{id: deterministicUUID("food:zucchini"), name: "Zucchini"},
		{id: deterministicUUID("food:apple"), name: "Apple"},
		{id: deterministicUUID("food:milk"), name: "Milk"},
	}

	for _, row := range rows {
		if _, err := db.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, row.id, row.name); err != nil {
			t.Fatalf("insert food %q: %v", row.name, err)
		}
	}

	food, err := ListFood(db)
	if err != nil {
		t.Fatalf("ListFood returned error: %v", err)
	}

	if len(food) != 3 {
		t.Fatalf("expected 3 food items, got %d", len(food))
	}

	if food[0].Name != "Apple" {
		t.Fatalf("expected first food name %q, got %q", "Apple", food[0].Name)
	}

	if food[1].Name != "Milk" {
		t.Fatalf("expected second food name %q, got %q", "Milk", food[1].Name)
	}

	if food[2].Name != "Zucchini" {
		t.Fatalf("expected third food name %q, got %q", "Zucchini", food[2].Name)
	}
}

func TestListFood_Empty(t *testing.T) {
	db := setupTestDB(t)

	food, err := ListFood(db)
	if err != nil {
		t.Fatalf("ListFood returned error: %v", err)
	}

	if food == nil {
		t.Fatal("expected non-nil slice, got nil")
	}

	if len(food) != 0 {
		t.Fatalf("expected 0 food items, got %d", len(food))
	}
}

func TestGetFoodByName_Found(t *testing.T) {
	db := setupTestDB(t)
	seedTestData(t, db)

	food, err := GetFoodByName(db, testFoodName)
	if err != nil {
		t.Fatalf("GetFoodByName returned error: %v", err)
	}

	if food == nil {
		t.Fatal("expected food, got nil")
	}

	if food.ID != deterministicUUID(testFoodID) {
		t.Fatalf("expected ID %q, got %q", deterministicUUID(testFoodID), food.ID)
	}

	if food.Name != testFoodName {
		t.Fatalf("expected name %q, got %q", testFoodName, food.Name)
	}
}

func TestGetFoodByName_NotFound(t *testing.T) {
	db := setupTestDB(t)

	food, err := GetFoodByName(db, "does-not-exist")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if food != nil {
		t.Fatalf("expected nil food, got %#v", food)
	}
}

func TestCreateFood(t *testing.T) {
	db := setupTestDB(t)

	food := Food{ID: uuid.New().String(), Name: "Butter"}
	if err := CreateFood(db, &food); err != nil {
		t.Fatalf("CreateFood returned error: %v", err)
	}

	persistedFood, err := GetFoodByID(db, food.ID)
	if err != nil {
		t.Fatalf("GetFoodByID returned error: %v", err)
	}

	if persistedFood == nil {
		t.Fatal("expected persisted food, got nil")
	}

	if persistedFood.ID != food.ID {
		t.Fatalf("expected ID %q, got %q", food.ID, persistedFood.ID)
	}

	if persistedFood.Name != food.Name {
		t.Fatalf("expected name %q, got %q", food.Name, persistedFood.Name)
	}
}

func TestCreateFood_DuplicateName(t *testing.T) {
	db := setupTestDB(t)

	firstFood := Food{ID: uuid.New().String(), Name: "Butter"}
	secondFood := Food{ID: uuid.New().String(), Name: "Butter"}

	if err := CreateFood(db, &firstFood); err != nil {
		t.Fatalf("CreateFood first row returned error: %v", err)
	}

	if err := CreateFood(db, &secondFood); err != nil {
		t.Fatalf("CreateFood second row returned error: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM food WHERE name = ?`, "Butter").Scan(&count); err != nil {
		t.Fatalf("count food rows: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 food rows, got %d", count)
	}
}
