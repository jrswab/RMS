package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

var testHashedPassword []byte

func init() {
	hash, err := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to generate test bcrypt hash: " + err.Error())
	}
	testHashedPassword = hash
}

func TestVerifyPassword_Correct(t *testing.T) {
	t.Parallel()

	err := VerifyPassword(string(testHashedPassword), "correctpassword")
	if err != nil {
		t.Errorf("VerifyPassword returned error for correct password: %v", err)
	}
}

func TestVerifyPassword_Incorrect(t *testing.T) {
	t.Parallel()

	err := VerifyPassword(string(testHashedPassword), "wrongpassword")
	if err == nil {
		t.Error("VerifyPassword should have returned error for incorrect password")
	}
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	t.Parallel()

	err := VerifyPassword(string(testHashedPassword), "")
	if err == nil {
		t.Error("VerifyPassword should have returned error for empty password")
	}
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	t.Parallel()

	err := VerifyPassword("", "password")
	if err == nil {
		t.Error("VerifyPassword should have returned error for empty hash")
	}
}
