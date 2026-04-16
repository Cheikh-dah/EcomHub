package middleware

import (
	"net/http"
	"strings"

	"ecomhub/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const ctxUserID = "userID"

func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization"})
			return
		}
		claims, err := auth.ParseToken(token, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Next()
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

func OptionalAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.Next()
			return
		}
		claims, err := auth.ParseToken(token, jwtSecret)
		if err != nil {
			c.Next()
			return
		}
		c.Set(ctxUserID, claims.UserID)
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return strings.TrimSpace(h[7:])
	}
	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		return cookie
	}
	return ""
}
