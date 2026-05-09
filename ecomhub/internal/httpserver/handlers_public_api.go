package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"ecomhub/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type publicStoreDTO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Subdomain   string    `json:"subdomain"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type publicStoreResponse struct {
	Store publicStoreDTO    `json:"store"`
	Theme models.StoreTheme `json:"theme"`
}

type publicStoreLoader func(context.Context, string) (models.Store, error)
type publicThemeLoader func(context.Context, int64) (models.StoreTheme, error)

func (s *Server) apiPublicStoreBySubdomain(c *gin.Context) {
	apiPublicStoreBySubdomain(c, s.loadStoreBySubdomain, s.loadStoreThemeByID)
}

func apiPublicStoreBySubdomain(c *gin.Context, loadStore publicStoreLoader, loadTheme publicThemeLoader) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}

	store, err := loadStore(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	theme, err := loadTheme(c.Request.Context(), store.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "theme query failed"})
		return
	}

	c.JSON(http.StatusOK, publicStoreResponse{
		Store: publicStoreDTO{
			ID:          store.ID,
			Name:        store.Name,
			Subdomain:   store.Subdomain,
			Description: store.Description,
			Status:      store.Status,
			CreatedAt:   store.CreatedAt,
		},
		Theme: theme,
	})
}
