package httpserver

import (
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

func (s *Server) apiPublicStoreBySubdomain(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}

	store, err := s.loadStoreBySubdomain(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	theme, err := s.loadStoreThemeByID(c.Request.Context(), store.ID)
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
