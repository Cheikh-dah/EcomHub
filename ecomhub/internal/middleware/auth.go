package middleware

import (
	"net/http"
	"strings"

	"ecomhub/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ctxUserID = "userID"

// RequireAuth verifies a Supabase access token (HS256, SUPABASE_JWT_SECRET) and maps
// sub → user_identities → internal user id.
func RequireAuth(pool *pgxpool.Pool, supabaseJWTSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
			return
		}
		if uid, ok := authenticate(c, pool, token, supabaseJWTSecret); ok {
			c.Set(ctxUserID, uid)
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}

func UserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(ctxUserID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// OptionalAuth sets ctxUserID when a valid Supabase token is present; invalid tokens are ignored.
func OptionalAuth(pool *pgxpool.Pool, supabaseJWTSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.Next()
			return
		}
		if uid, ok := authenticate(c, pool, token, supabaseJWTSecret); ok {
			c.Set(ctxUserID, uid)
		}
		c.Next()
	}
}

func authenticate(c *gin.Context, pool *pgxpool.Pool, token, supabaseJWTSecret string) (uuid.UUID, bool) {
	ctx := c.Request.Context()
	sub, email, _, err := auth.VerifySupabaseAccessToken(token, supabaseJWTSecret)
	if err != nil {
		return uuid.Nil, false
	}
	internalID, err := auth.ResolveSupabaseUser(ctx, pool, sub, email)
	if err != nil {
		return uuid.Nil, false
	}
	return internalID, true
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		return strings.TrimSpace(cookie)
	}
	return ""
}
