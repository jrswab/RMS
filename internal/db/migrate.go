package db

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

func RunMigrations(db *sql.DB, migrationFS fs.FS) error {
	filenames, err := fs.Glob(migrationFS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}

	sort.Strings(filenames)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migrations transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	appliedVersions := make(map[string]struct{}, len(filenames))

	rows, err := tx.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan applied migration version: %w", err)
		}

		appliedVersions[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate applied migrations: %w", err)
	}

	if err := rows.Close(); err != nil {
		return fmt.Errorf("close applied migrations rows: %w", err)
	}

	for _, filename := range filenames {
		version := filename
		if _, alreadyApplied := appliedVersions[version]; alreadyApplied {
			continue
		}

		contents, err := fs.ReadFile(migrationFS, filename)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", filename, err)
		}

		sqlText := strings.TrimSpace(string(contents))

		if sqlText != "" {
			if _, err := tx.Exec(sqlText); err != nil {
				return fmt.Errorf("execute migration %q: %w", filename, err)
			}
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			version,
			time.Now().Unix(),
		); err != nil {
			return fmt.Errorf("record migration %q: %w", filename, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}

	committed = true

	return nil
}
