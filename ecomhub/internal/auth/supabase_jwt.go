package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// VerifySupabaseAccessToken validates a Supabase-issued JWT (HS256) using the
// project's JWT secret from Dashboard → Settings → API (not the service_role key).
// Returns the auth user's id (claim "sub"), optional "email", and a suggested
// auth cookie MaxAge in seconds derived from JWT exp (min 60, default 3600).
func VerifySupabaseAccessToken(tokenString, secret string) (subject uuid.UUID, email string, cookieMaxAge int, err error) {
	cookieMaxAge = 3600
	if secret == "" {
		return uuid.Nil, "", cookieMaxAge, errors.New("empty jwt secret")
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
		return uuid.Nil, "", cookieMaxAge, err
	}
	subStr, ok := mc["sub"].(string)
	if !ok || subStr == "" {
		return uuid.Nil, "", cookieMaxAge, errors.New("jwt missing sub")
	}
	subject, err = uuid.Parse(subStr)
	if err != nil {
		return uuid.Nil, "", cookieMaxAge, fmt.Errorf("jwt sub not a uuid: %w", err)
	}
	if v, ok := mc["email"].(string); ok {
		email = v
	}
	if exp := claimExpUnix(mc["exp"]); exp > 0 {
		sec := int(time.Until(time.Unix(exp, 0)).Seconds())
		if sec < 60 {
			sec = 60
		}
		const maxCookie = 86400 * 7
		if sec > maxCookie {
			sec = maxCookie
		}
		cookieMaxAge = sec
	}
	return subject, email, cookieMaxAge, nil
}

func claimExpUnix(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
