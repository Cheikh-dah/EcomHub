package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"ecomhub/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const ctxUserID = "userID"

// RequireAuth verifies a Clerk session JWT and returns JSON 401 if missing/invalid.
func RequireAuth(pool *pgxpool.Pool, authorizedParties []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
			return
		}
		if uid, ok := authenticate(c.Request.Context(), pool, token, authorizedParties); ok {
			c.Set(ctxUserID, uid)
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
	}
}

// RequireAuthRedirect verifies a Clerk session JWT and redirects to /dashboard if missing/invalid.
func RequireAuthRedirect(pool *pgxpool.Pool, authorizedParties []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.Redirect(http.StatusSeeOther, dashboardLoginPath(c))
			c.Abort()
			return
		}
		if uid, ok := authenticate(c.Request.Context(), pool, token, authorizedParties); ok {
			c.Set(ctxUserID, uid)
			c.Next()
			return
		}
		c.Redirect(http.StatusSeeOther, dashboardLoginPath(c))
		c.Abort()
	}
}

func dashboardLoginPath(c *gin.Context) string {
	next := c.Request.URL.RequestURI()
	if strings.TrimSpace(next) == "" {
		next = "/dashboard"
	}
	return "/dashboard?next=" + url.QueryEscape(next)
}

func UserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(ctxUserID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// OptionalAuth sets ctxUserID when a valid Clerk session token is present; invalid tokens are ignored.
func OptionalAuth(pool *pgxpool.Pool, authorizedParties []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.Next()
			return
		}
		if uid, ok := authenticate(c.Request.Context(), pool, token, authorizedParties); ok {
			c.Set(ctxUserID, uid)
		}
		c.Next()
	}
}

func authenticate(ctx context.Context, pool *pgxpool.Pool, token string, authorizedParties []string) (uuid.UUID, bool) {
	clerkUserID, _, err := auth.VerifyClerkSessionJWT(ctx, token, authorizedParties)
	if err != nil {
		return uuid.Nil, false
	}
	internalID, err := auth.ResolveClerkUser(ctx, pool, clerkUserID)
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
