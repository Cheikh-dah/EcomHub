package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ecomhub/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func testPublicStoreRouter(loadStore publicStoreLoader, loadTheme publicThemeLoader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/public/stores/:subdomain", func(c *gin.Context) {
		apiPublicStoreBySubdomain(c, loadStore, loadTheme)
	})
	return r
}

func performPublicStoreRequest(r http.Handler, subdomain string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/public/stores/"+subdomain, nil)
	r.ServeHTTP(w, req)
	return w
}

func TestPublicStoreBySubdomainInvalidSubdomain(t *testing.T) {
	r := testPublicStoreRouter(
		func(context.Context, string) (models.Store, error) {
			t.Fatal("store loader should not be called for invalid subdomain")
			return models.Store{}, nil
		},
		func(context.Context, int64) (models.StoreTheme, error) {
			t.Fatal("theme loader should not be called for invalid subdomain")
			return models.StoreTheme{}, nil
		},
	)

	w := performPublicStoreRequest(r, "-bad")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublicStoreBySubdomainMissingOrInactiveStore(t *testing.T) {
	r := testPublicStoreRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "missing" {
				t.Fatalf("expected normalized subdomain missing, got %q", subdomain)
			}
			return models.Store{}, pgx.ErrNoRows
		},
		func(context.Context, int64) (models.StoreTheme, error) {
			t.Fatal("theme loader should not be called when store is missing")
			return models.StoreTheme{}, nil
		},
	)

	w := performPublicStoreRequest(r, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublicStoreBySubdomainActiveStore(t *testing.T) {
	createdAt := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	storeUserID := uuid.New()
	theme := defaultStoreTheme()
	theme.PrimaryColor = "#111827"
	theme.LogoURL = "https://example.com/logo.png"

	r := testPublicStoreRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "my-store" {
				t.Fatalf("expected normalized subdomain my-store, got %q", subdomain)
			}
			return models.Store{
				ID:          42,
				UserID:      storeUserID,
				Name:        "My Store",
				Subdomain:   "my-store",
				Description: "Public description",
				Status:      "active",
				CreatedAt:   createdAt,
			}, nil
		},
		func(_ context.Context, storeID int64) (models.StoreTheme, error) {
			if storeID != 42 {
				t.Fatalf("expected store id 42, got %d", storeID)
			}
			return theme, nil
		},
	)

	w := performPublicStoreRequest(r, "My-Store")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "user_id") || strings.Contains(w.Body.String(), storeUserID.String()) {
		t.Fatalf("response leaked user identity: %s", w.Body.String())
	}

	var got publicStoreResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if got.Store.ID != 42 || got.Store.Name != "My Store" || got.Store.Subdomain != "my-store" {
		t.Fatalf("unexpected store payload: %#v", got.Store)
	}
	if got.Store.Status != "active" {
		t.Fatalf("expected active status, got %q", got.Store.Status)
	}
	if got.Theme.PrimaryColor != "#111827" {
		t.Fatalf("expected normalized theme in response, got %#v", got.Theme)
	}
	if got.Theme.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("expected logo URL in response, got %#v", got.Theme)
	}
}
