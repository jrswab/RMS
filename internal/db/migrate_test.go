package db

import (
	"database/sql"
	"strings"
	"testing"
	"testing/fstest"

	"lab37/migrations"
)

func TestRunMigrations(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	if got := schemaMigrationCount(t, db); got != 1 {
		t.Fatalf("schema_migrations row count = %d, want 1", got)
	}

	for _, tableName := range []string{
		"restaurants",
		"users",
		"food",
		"user_restaurants",
		"recipes",
		"ingredients",
	} {
		if !sqliteTableExists(t, db, tableName) {
			t.Fatalf("table %q does not exist", tableName)
		}
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("first RunMigrations() error = %v", err)
	}

	if err := RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("second RunMigrations() error = %v", err)
	}

	if got := schemaMigrationCount(t, db); got != 1 {
		t.Fatalf("schema_migrations row count after rerun = %d, want 1", got)
	}
}

func TestRunMigrations_EmptyFS(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := RunMigrations(db, fstest.MapFS{}); err != nil {
		t.Fatalf("RunMigrations() with empty fs error = %v", err)
	}

	if !sqliteTableExists(t, db, "schema_migrations") {
		t.Fatal("schema_migrations table does not exist")
	}

	if got := schemaMigrationCount(t, db); got != 0 {
		t.Fatalf("schema_migrations row count = %d, want 0", got)
	}
}

func TestRunMigrations_ForeignKeys(t *testing.T) {
	db := openMigrationTestDB(t)

	if err := RunMigrations(db, migrations.FS); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	if _, err := db.Exec(`INSERT INTO restaurants (id, name) VALUES (?, ?)`, "restaurant-1", "Test Restaurant"); err != nil {
		t.Fatalf("insert restaurant error = %v", err)
	}

	_, err := db.Exec(`INSERT INTO user_restaurants (user_id, restaurant_id) VALUES (?, ?)`, "missing-user", "restaurant-1")
	if err == nil {
		t.Fatal("insert invalid user_restaurants row error = nil, want foreign key error")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Fatalf("insert invalid user_restaurants row error = %v, want foreign key error", err)
	}
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})

	return db
}

func schemaMigrationCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations count error = %v", err)
	}

	return count
}

func sqliteTableExists(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, tableName).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master for table %q error = %v", tableName, err)
	}

	return count == 1
}
