package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpen(t *testing.T) {
	t.Run("memory database enables foreign keys", func(t *testing.T) {
		db, err := Open(":memory:")
		if err != nil {
			t.Fatalf("Open(:memory:) error = %v", err)
		}

		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Errorf("db.Close() error = %v", err)
			}
		})

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		conn := testConn(t, db)

		if got := pragmaInt(t, conn, "foreign_keys"); got != 1 {
			t.Fatalf("PRAGMA foreign_keys = %d, want 1", got)
		}
	})

	t.Run("file database enables wal journal mode", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		db, err := Open(dbPath)
		if err != nil {
			t.Fatalf("Open(%q) error = %v", dbPath, err)
		}

		t.Cleanup(func() {
			if err := db.Close(); err != nil {
				t.Errorf("db.Close() error = %v", err)
			}
		})

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		conn := testConn(t, db)

		if got := pragmaText(t, conn, "journal_mode"); !strings.EqualFold(got, "wal") {
			t.Fatalf("PRAGMA journal_mode = %q, want %q", got, "wal")
		}
	})
}

func TestOpen_InvalidPath(t *testing.T) {
	const dbPath = "/nonexistent/dir/db.sqlite"

	db, err := Open(dbPath)
	if err == nil {
		if db != nil {
			if closeErr := db.Close(); closeErr != nil {
				t.Errorf("db.Close() error = %v", closeErr)
			}
		}

		t.Fatalf("Open(%q) error = nil, want non-nil", dbPath)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("defaults to recipes.db when DB_PATH is unset", func(t *testing.T) {
		unsetEnv(t, "DB_PATH")

		if got := DefaultPath(); got != "recipes.db" {
			t.Fatalf("DefaultPath() = %q, want %q", got, "recipes.db")
		}
	})

	t.Run("uses DB_PATH when set", func(t *testing.T) {
		t.Setenv("DB_PATH", "/tmp/test.db")

		if got := DefaultPath(); got != "/tmp/test.db" {
			t.Fatalf("DefaultPath() = %q, want %q", got, "/tmp/test.db")
		}
	})
}

func testConn(t *testing.T, db *sql.DB) *sql.Conn {
	t.Helper()

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("db.Conn() error = %v", err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("conn.Close() error = %v", err)
		}
	})

	return conn
}

func pragmaInt(t *testing.T, conn *sql.Conn, pragma string) int {
	t.Helper()

	var value int
	if err := conn.QueryRowContext(context.Background(), "PRAGMA "+pragma).Scan(&value); err != nil {
		t.Fatalf("query PRAGMA %s error = %v", pragma, err)
	}

	return value
}

func pragmaText(t *testing.T, conn *sql.Conn, pragma string) string {
	t.Helper()

	var value string
	if err := conn.QueryRowContext(context.Background(), "PRAGMA "+pragma).Scan(&value); err != nil {
		t.Fatalf("query PRAGMA %s error = %v", pragma, err)
	}

	return value
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()

	oldValue, hadOldValue := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
	}

	t.Cleanup(func() {
		var err error
		if hadOldValue {
			err = os.Setenv(key, oldValue)
		} else {
			err = os.Unsetenv(key)
		}

		if err != nil {
			t.Errorf("restore env %q error = %v", key, err)
		}
	})
}
