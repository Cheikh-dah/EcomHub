package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// VerifySupabaseAccessToken validates a Supabase-issued JWT (HS256) using the
// project's JWT secret from Dashboard → Settings → API (not the service_role key).
// Returns the auth user's id (claim "sub") and optional "email".
func VerifySupabaseAccessToken(tokenString, secret string) (subject uuid.UUID, email string, err error) {
	if secret == "" {
		return uuid.Nil, "", errors.New("empty jwt secret")
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	var mc jwt.MapClaims
	_, err = parser.ParseWithClaims(tokenString, &mc, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}
	subStr, ok := mc["sub"].(string)
	if !ok || subStr == "" {
		return uuid.Nil, "", errors.New("jwt missing sub")
	}
	subject, err = uuid.Parse(subStr)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("jwt sub not a uuid: %w", err)
	}
	if v, ok := mc["email"].(string); ok {
		email = v
	}
	return subject, email, nil
}
