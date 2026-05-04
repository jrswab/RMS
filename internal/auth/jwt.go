package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims extends jwt.RegisteredClaims with application-specific fields.
type Claims struct {
	UserID        string   `json:"user_id"`
	Role          string   `json:"role"`
	RestaurantIDs []string `json:"restaurant_ids"`
	jwt.RegisteredClaims
}

// CreateToken generates a signed JWT with the given claims.
func CreateToken(secret []byte, userID string, role string, restaurantIDs []string) (string, error) {
	claims := &Claims{
		UserID:        userID,
		Role:          role,
		RestaurantIDs: restaurantIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT, returning the claims if valid.
func ValidateToken(secret []byte, tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	if claims.UserID == "" {
		return nil, fmt.Errorf("token missing user_id claim")
	}

	if claims.Role == "" {
		return nil, fmt.Errorf("token missing role claim")
	}

	return claims, nil
}
