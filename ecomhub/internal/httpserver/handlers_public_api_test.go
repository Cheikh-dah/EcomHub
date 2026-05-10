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

func testPublicStoreProductsRouter(loadStore publicStoreLoader, loadProducts publicProductsLoader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/public/stores/:subdomain/products", func(c *gin.Context) {
		apiPublicStoreProducts(c, loadStore, loadProducts)
	})
	return r
}

func performPublicStoreRequest(r http.Handler, subdomain string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/public/stores/"+subdomain, nil)
	r.ServeHTTP(w, req)
	return w
}

func performPublicStoreProductsRequest(r http.Handler, subdomain string) *httptest.ResponseRecorder {
	return performPublicStoreProductsRequestWithQuery(r, subdomain, "")
}

func performPublicStoreProductsRequestWithQuery(r http.Handler, subdomain string, query string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	target := "/api/public/stores/" + subdomain + "/products"
	if query != "" {
		target += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, target, nil)
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

func TestPublicStoreProductsInvalidSubdomain(t *testing.T) {
	r := testPublicStoreProductsRouter(
		func(context.Context, string) (models.Store, error) {
			t.Fatal("store loader should not be called for invalid subdomain")
			return models.Store{}, nil
		},
		func(context.Context, int64, int, int) ([]models.Product, error) {
			t.Fatal("products loader should not be called for invalid subdomain")
			return nil, nil
		},
	)

	w := performPublicStoreProductsRequest(r, "-bad")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublicStoreProductsMissingOrInactiveStore(t *testing.T) {
	r := testPublicStoreProductsRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "missing" {
				t.Fatalf("expected normalized subdomain missing, got %q", subdomain)
			}
			return models.Store{}, pgx.ErrNoRows
		},
		func(context.Context, int64, int, int) ([]models.Product, error) {
			t.Fatal("products loader should not be called when store is missing")
			return nil, nil
		},
	)

	w := performPublicStoreProductsRequest(r, "missing")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublicStoreProductsActiveStoreEmptyList(t *testing.T) {
	r := testPublicStoreProductsRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "empty-store" {
				t.Fatalf("expected normalized subdomain empty-store, got %q", subdomain)
			}
			return models.Store{ID: 7, Name: "Empty Store", Subdomain: "empty-store", Status: "active"}, nil
		},
		func(_ context.Context, storeID int64, limit int, offset int) ([]models.Product, error) {
			if storeID != 7 {
				t.Fatalf("expected store id 7, got %d", storeID)
			}
			if limit != defaultPublicProductsLimit+1 {
				t.Fatalf("expected default limit+1 fetch, got %d", limit)
			}
			if offset != 0 {
				t.Fatalf("expected default offset 0, got %d", offset)
			}
			return []models.Product{}, nil
		},
	)

	w := performPublicStoreProductsRequest(r, "Empty-Store")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got publicStoreProductsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if got.Store.ID != 7 || got.Store.Name != "Empty Store" || got.Store.Subdomain != "empty-store" {
		t.Fatalf("unexpected store summary: %#v", got.Store)
	}
	if got.Products == nil {
		t.Fatal("expected products to be an empty JSON array, got nil")
	}
	if len(got.Products) != 0 {
		t.Fatalf("expected no products, got %#v", got.Products)
	}
	if got.Pagination.Limit != defaultPublicProductsLimit || got.Pagination.Offset != 0 || got.Pagination.Count != 0 || got.Pagination.HasMore {
		t.Fatalf("unexpected pagination: %#v", got.Pagination)
	}
}

func TestPublicStoreProductsActiveStoreWithProducts(t *testing.T) {
	createdAt := time.Date(2026, 5, 9, 13, 0, 0, 0, time.UTC)
	storeUserID := uuid.New()
	r := testPublicStoreProductsRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "my-store" {
				t.Fatalf("expected normalized subdomain my-store, got %q", subdomain)
			}
			return models.Store{
				ID:        42,
				UserID:    storeUserID,
				Name:      "My Store",
				Subdomain: "my-store",
				Status:    "active",
			}, nil
		},
		func(_ context.Context, storeID int64, limit int, offset int) ([]models.Product, error) {
			if storeID != 42 {
				t.Fatalf("expected store id 42, got %d", storeID)
			}
			if limit != defaultPublicProductsLimit+1 {
				t.Fatalf("expected default limit+1 fetch, got %d", limit)
			}
			if offset != 0 {
				t.Fatalf("expected default offset 0, got %d", offset)
			}
			return []models.Product{
				{
					ID:          100,
					StoreID:     42,
					Name:        "Perfume",
					Description: "Nice scent",
					Price:       19.99,
					Stock:       5,
					ImageURL:    "https://example.com/perfume.jpg",
					CreatedAt:   createdAt,
				},
			}, nil
		},
	)

	w := performPublicStoreProductsRequest(r, "My-Store")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "user_id") || strings.Contains(w.Body.String(), storeUserID.String()) {
		t.Fatalf("response leaked user identity: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "store_id") {
		t.Fatalf("response leaked product store_id: %s", w.Body.String())
	}

	var got publicStoreProductsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if got.Store.ID != 42 || got.Store.Name != "My Store" || got.Store.Subdomain != "my-store" {
		t.Fatalf("unexpected store summary: %#v", got.Store)
	}
	if len(got.Products) != 1 {
		t.Fatalf("expected 1 product, got %#v", got.Products)
	}
	product := got.Products[0]
	if product.ID != 100 || product.Name != "Perfume" || product.Price != 19.99 || product.Stock != 5 {
		t.Fatalf("unexpected product payload: %#v", product)
	}
	if product.Description != "Nice scent" || product.ImageURL != "https://example.com/perfume.jpg" {
		t.Fatalf("unexpected product detail payload: %#v", product)
	}
	if got.Pagination.Limit != defaultPublicProductsLimit || got.Pagination.Offset != 0 || got.Pagination.Count != 1 || got.Pagination.HasMore {
		t.Fatalf("unexpected pagination: %#v", got.Pagination)
	}
}

func TestPublicStoreProductsCustomPaginationAndHasMore(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 9, 0, 0, 0, time.UTC)
	r := testPublicStoreProductsRouter(
		func(_ context.Context, subdomain string) (models.Store, error) {
			if subdomain != "my-store" {
				t.Fatalf("expected normalized subdomain my-store, got %q", subdomain)
			}
			return models.Store{ID: 42, Name: "My Store", Subdomain: "my-store", Status: "active"}, nil
		},
		func(_ context.Context, storeID int64, limit int, offset int) ([]models.Product, error) {
			if storeID != 42 {
				t.Fatalf("expected store id 42, got %d", storeID)
			}
			if limit != 3 {
				t.Fatalf("expected requested limit+1 fetch of 3, got %d", limit)
			}
			if offset != 4 {
				t.Fatalf("expected offset 4, got %d", offset)
			}
			return []models.Product{
				{ID: 3, Name: "Third", Price: 3, CreatedAt: createdAt},
				{ID: 2, Name: "Second", Price: 2, CreatedAt: createdAt},
				{ID: 1, Name: "Extra", Price: 1, CreatedAt: createdAt},
			}, nil
		},
	)

	w := performPublicStoreProductsRequestWithQuery(r, "My-Store", "limit=2&offset=4")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var got publicStoreProductsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(got.Products) != 2 {
		t.Fatalf("expected returned products sliced to limit, got %#v", got.Products)
	}
	if got.Products[0].ID != 3 || got.Products[1].ID != 2 {
		t.Fatalf("unexpected products after slicing: %#v", got.Products)
	}
	if got.Pagination.Limit != 2 || got.Pagination.Offset != 4 || got.Pagination.Count != 2 || !got.Pagination.HasMore {
		t.Fatalf("unexpected pagination: %#v", got.Pagination)
	}
}

func TestPublicStoreProductsRejectsInvalidPagination(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "non integer limit", query: "limit=abc"},
		{name: "zero limit", query: "limit=0"},
		{name: "limit too high", query: "limit=51"},
		{name: "non integer offset", query: "offset=abc"},
		{name: "negative offset", query: "offset=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testPublicStoreProductsRouter(
				func(context.Context, string) (models.Store, error) {
					t.Fatal("store loader should not be called for invalid pagination")
					return models.Store{}, nil
				},
				func(context.Context, int64, int, int) ([]models.Product, error) {
					t.Fatal("products loader should not be called for invalid pagination")
					return nil, nil
				},
			)

			w := performPublicStoreProductsRequestWithQuery(r, "my-store", tt.query)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
