package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"ecomhub/internal/models"

	"github.com/gin-gonic/gin"
)

const cartCookieName = "ecomhub_cart"

func readCart(c *gin.Context) (models.CartPayload, error) {
	raw, err := c.Cookie(cartCookieName)
	if err != nil || raw == "" {
		return models.CartPayload{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return models.CartPayload{}, err
	}
	var p models.CartPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return models.CartPayload{}, err
	}
	return p, nil
}

func writeCart(c *gin.Context, p models.CartPayload) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	enc := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cartCookieName,
		Value:    enc,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func clearCartCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cartCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func setAuthCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
