package store

import (
	"errors"
	"testing"
)

const expectedUserTimestamp int64 = 1767225600

func TestGetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	user, err := GetUserByUsername(db, testUsername)
	if err != nil {
		t.Fatalf("GetUserByUsername returned error: %v", err)
	}
	if user == nil {
		t.Fatal("GetUserByUsername returned nil user")
	}

	if user.ID != deterministicUUID(testUserID) {
		t.Fatalf("ID = %q, want %q", user.ID, deterministicUUID(testUserID))
	}
	if user.Username != testUsername {
		t.Fatalf("Username = %q, want %q", user.Username, testUsername)
	}
	if user.Password == "" {
		t.Fatal("Password is empty")
	}
	if user.Role != testRole {
		t.Fatalf("Role = %q, want %q", user.Role, testRole)
	}
	if user.CreatedAt != expectedUserTimestamp {
		t.Fatalf("CreatedAt = %d, want %d", user.CreatedAt, expectedUserTimestamp)
	}
	if user.UpdatedAt != expectedUserTimestamp {
		t.Fatalf("UpdatedAt = %d, want %d", user.UpdatedAt, expectedUserTimestamp)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	user, err := GetUserByUsername(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if user != nil {
		t.Fatalf("user = %#v, want nil", user)
	}
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})
	seedTestData(t, db)

	user, err := GetUserByID(db, deterministicUUID(testUserID))
	if err != nil {
		t.Fatalf("GetUserByID returned error: %v", err)
	}
	if user == nil {
		t.Fatal("GetUserByID returned nil user")
	}

	if user.ID != deterministicUUID(testUserID) {
		t.Fatalf("ID = %q, want %q", user.ID, deterministicUUID(testUserID))
	}
	if user.Username != testUsername {
		t.Fatalf("Username = %q, want %q", user.Username, testUsername)
	}
	if user.Password == "" {
		t.Fatal("Password is empty")
	}
	if user.Role != testRole {
		t.Fatalf("Role = %q, want %q", user.Role, testRole)
	}
	if user.CreatedAt != expectedUserTimestamp {
		t.Fatalf("CreatedAt = %d, want %d", user.CreatedAt, expectedUserTimestamp)
	}
	if user.UpdatedAt != expectedUserTimestamp {
		t.Fatalf("UpdatedAt = %d, want %d", user.UpdatedAt, expectedUserTimestamp)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() {
		_ = db.Close()
	})

	user, err := GetUserByID(db, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrNotFound)
	}
	if user != nil {
		t.Fatalf("user = %#v, want nil", user)
	}
}
