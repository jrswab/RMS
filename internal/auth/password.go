package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// VerifyPassword compares a bcrypt password hash with a plaintext password.
func VerifyPassword(hashedPassword string, password string) error {
	if hashedPassword == "" {
		return fmt.Errorf("password hash is empty")
	}

	if password == "" {
		return fmt.Errorf("password is empty")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return fmt.Errorf("verify password: %w", err)
	}

	return nil
}
