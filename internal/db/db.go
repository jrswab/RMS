package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const defaultPath = "recipes.db"

func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys for sqlite database %q: %w", dbPath, err)
	}

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL journal mode for sqlite database %q: %w", dbPath, err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database %q: %w", dbPath, err)
	}

	return db, nil
}

func DefaultPath() string {
	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		return dbPath
	}

	return defaultPath
}
