package store

import (
	"database/sql"
	"errors"
	"fmt"
)

func GetFoodByID(db DBTX, id string) (*Food, error) {
	var food Food

	err := db.QueryRow(`SELECT id, name FROM food WHERE id = ?`, id).Scan(&food.ID, &food.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get food by id %q: %w", id, err)
	}

	return &food, nil
}

func ListFood(db DBTX) ([]Food, error) {
	rows, err := db.Query(`SELECT id, name FROM food ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list food: %w", err)
	}
	defer rows.Close()

	food := make([]Food, 0)
	for rows.Next() {
		var item Food
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan food row: %w", err)
		}

		food = append(food, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate food rows: %w", err)
	}

	return food, nil
}

func GetFoodByName(db DBTX, name string) (*Food, error) {
	var food Food

	err := db.QueryRow(`SELECT id, name FROM food WHERE name = ? LIMIT 1`, name).Scan(&food.ID, &food.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get food by name %q: %w", name, err)
	}

	return &food, nil
}

func CreateFood(db DBTX, f *Food) error {
	if _, err := db.Exec(`INSERT INTO food (id, name) VALUES (?, ?)`, f.ID, f.Name); err != nil {
		return fmt.Errorf("create food %q: %w", f.ID, err)
	}

	return nil
}
