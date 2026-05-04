package auth

import (
	"context"
	"fmt"
	"net/http"
)

// contextKey is an unexported type for context keys to avoid collisions.
type contextKey struct{}

// claimsKey is the context key for storing JWT claims.
var claimsKey = contextKey{}

// ContextWithClaims returns a new request with the claims stored in the context.
func ContextWithClaims(r *http.Request, claims *Claims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
}

// ClaimsFromContext retrieves the JWT claims from the request context.
func ClaimsFromContext(r *http.Request) (*Claims, error) {
	claims, ok := r.Context().Value(claimsKey).(*Claims)
	if !ok || claims == nil {
		return nil, fmt.Errorf("no claims in context")
	}

	return claims, nil
}
