package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestCreateToken(t *testing.T) {
	t.Parallel()

	token, err := CreateToken([]byte("test-secret"), "user-123", "admin", []string{"rest-1", "rest-2"})
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	if token == "" {
		t.Fatal("CreateToken returned empty token")
	}

	// JWT format: three parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestCreateToken_ValidateRoundTrip(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	token, err := CreateToken(secret, "user-123", "admin", []string{"rest-1", "rest-2"})
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	claims, err := ValidateToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-123")
	}

	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	if len(claims.RestaurantIDs) != 2 {
		t.Fatalf("RestaurantIDs length = %d, want 2", len(claims.RestaurantIDs))
	}

	if claims.RestaurantIDs[0] != "rest-1" || claims.RestaurantIDs[1] != "rest-2" {
		t.Errorf("RestaurantIDs = %v, want [rest-1 rest-2]", claims.RestaurantIDs)
	}

	// ExpiresAt should be approximately 1 hour in the future
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	if diff := time.Until(claims.ExpiresAt.Time); diff < 59*time.Minute || diff > 61*time.Minute {
		t.Errorf("ExpiresAt is %v from now, want approximately 1 hour", diff)
	}

	if claims.IssuedAt == nil {
		t.Fatal("IssuedAt is nil")
	}
	if diff := time.Since(claims.IssuedAt.Time); diff > 5*time.Second {
		t.Errorf("IssuedAt is %v ago, want approximately now", diff)
	}
}

func TestCreateToken_Expired(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	claims := &Claims{
		UserID:        "user-123",
		Role:          "admin",
		RestaurantIDs: []string{"rest-1"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	_, err = ValidateToken(secret, tokenString)
	if err == nil {
		t.Fatal("ValidateToken should have returned error for expired token")
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	t.Parallel()

	token, err := CreateToken([]byte("secret-a"), "user-123", "admin", []string{"rest-1"})
	if err != nil {
		t.Fatalf("CreateToken returned error: %v", err)
	}

	_, err = ValidateToken([]byte("secret-b"), token)
	if err == nil {
		t.Fatal("ValidateToken should have returned error for invalid signature")
	}
}

func TestValidateToken_MissingUserID(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	claims := &Claims{
		UserID:        "",
		Role:          "admin",
		RestaurantIDs: []string{"rest-1"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	_, err = ValidateToken(secret, tokenString)
	if err == nil {
		t.Fatal("ValidateToken should have returned error for missing user_id")
	}
}

func TestValidateToken_MissingRole(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	claims := &Claims{
		UserID:        "user-123",
		Role:          "",
		RestaurantIDs: []string{"rest-1"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}

	_, err = ValidateToken(secret, tokenString)
	if err == nil {
		t.Fatal("ValidateToken should have returned error for missing role")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	t.Parallel()

	_, err := ValidateToken([]byte("secret"), "not-a-jwt")
	if err == nil {
		t.Fatal("ValidateToken should have returned error for malformed token")
	}
}

func TestValidateToken_EmptyString(t *testing.T) {
	t.Parallel()

	_, err := ValidateToken([]byte("secret"), "")
	if err == nil {
		t.Fatal("ValidateToken should have returned error for empty string")
	}
}
