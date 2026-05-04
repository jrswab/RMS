package store

import (
	"database/sql"
	"errors"
	"fmt"
)

func GetUserByUsername(db DBTX, username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		`SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get user by username %q: %w", username, err)
	}

	return user, nil
}

func GetUserByID(db DBTX, id string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		`SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get user by id %q: %w", id, err)
	}

	return user, nil
}
