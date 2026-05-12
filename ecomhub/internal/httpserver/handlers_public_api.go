package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
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

type publicStoreSummaryDTO struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
}

type publicProductDTO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type publicStoreProductsResponse struct {
	Store      publicStoreSummaryDTO `json:"store"`
	Products   []publicProductDTO    `json:"products"`
	Pagination publicPaginationDTO   `json:"pagination"`
}

type publicStoreProductResponse struct {
	Store   publicStoreSummaryDTO `json:"store"`
	Product publicProductDTO      `json:"product"`
}

type publicPaginationDTO struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Count   int  `json:"count"`
	HasMore bool `json:"has_more"`
}

type publicPaginationParams struct {
	Limit  int
	Offset int
}

type publicStoreLoader func(context.Context, string) (models.Store, error)
type publicThemeLoader func(context.Context, int64) (models.StoreTheme, error)
type publicProductsLoader func(context.Context, int64, int, int) ([]models.Product, error)
type publicProductLoader func(context.Context, int64, int64) (models.Product, error)

const (
	defaultPublicProductsLimit = 24
	maxPublicProductsLimit     = 50
)

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

func (s *Server) apiPublicStoreProducts(c *gin.Context) {
	apiPublicStoreProducts(c, s.loadStoreBySubdomain, s.loadPublicProductsByStoreID)
}

func apiPublicStoreProducts(c *gin.Context, loadStore publicStoreLoader, loadProducts publicProductsLoader) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}

	pagination, err := parsePublicPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	products, err := loadProducts(c.Request.Context(), store.ID, pagination.Limit+1, pagination.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "products query failed"})
		return
	}

	hasMore := len(products) > pagination.Limit
	if hasMore {
		products = products[:pagination.Limit]
	}

	out := make([]publicProductDTO, 0, len(products))
	for _, product := range products {
		out = append(out, publicProductToDTO(product))
	}

	c.JSON(http.StatusOK, publicStoreProductsResponse{
		Store: publicStoreSummaryDTO{
			ID:        store.ID,
			Name:      store.Name,
			Subdomain: store.Subdomain,
		},
		Products: out,
		Pagination: publicPaginationDTO{
			Limit:   pagination.Limit,
			Offset:  pagination.Offset,
			Count:   len(out),
			HasMore: hasMore,
		},
	})
}

func (s *Server) apiPublicStoreProduct(c *gin.Context) {
	apiPublicStoreProduct(c, s.loadStoreBySubdomain, s.loadPublicProductByStoreID)
}

func apiPublicStoreProduct(c *gin.Context, loadStore publicStoreLoader, loadProduct publicProductLoader) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}

	productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || productID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product id"})
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

	product, err := loadProduct(c.Request.Context(), store.ID, productID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "product query failed"})
		return
	}

	c.JSON(http.StatusOK, publicStoreProductResponse{
		Store: publicStoreSummaryDTO{
			ID:        store.ID,
			Name:      store.Name,
			Subdomain: store.Subdomain,
		},
		Product: publicProductToDTO(product),
	})
}

func publicProductToDTO(product models.Product) publicProductDTO {
	return publicProductDTO{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
		ImageURL:    product.ImageURL,
		CreatedAt:   product.CreatedAt,
	}
}

func parsePublicPagination(c *gin.Context) (publicPaginationParams, error) {
	limit := defaultPublicProductsLimit
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return publicPaginationParams{}, errors.New("invalid limit")
		}
		limit = parsed
	}
	if limit < 1 {
		return publicPaginationParams{}, errors.New("invalid limit")
	}
	if limit > maxPublicProductsLimit {
		return publicPaginationParams{}, errors.New("limit must be less than or equal to 50")
	}

	offset := 0
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return publicPaginationParams{}, errors.New("invalid offset")
		}
		offset = parsed
	}
	if offset < 0 {
		return publicPaginationParams{}, errors.New("invalid offset")
	}

	return publicPaginationParams{Limit: limit, Offset: offset}, nil
}

func (s *Server) loadPublicProductsByStoreID(ctx context.Context, storeID int64, limit int, offset int) ([]models.Product, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, store_id, name, description, price::float8, stock, COALESCE(image_url,''), created_at
		 FROM products
		 WHERE store_id = $1
		 ORDER BY id DESC
		 LIMIT $2 OFFSET $3`,
		storeID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.ID,
			&product.StoreID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Stock,
			&product.ImageURL,
			&product.CreatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *Server) loadPublicProductByStoreID(ctx context.Context, storeID int64, productID int64) (models.Product, error) {
	var product models.Product
	err := s.pool.QueryRow(ctx,
		`SELECT id, store_id, name, description, price::float8, stock, COALESCE(image_url,''), created_at
		 FROM products
		 WHERE id = $1 AND store_id = $2`,
		productID, storeID,
	).Scan(
		&product.ID,
		&product.StoreID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Stock,
		&product.ImageURL,
		&product.CreatedAt,
	)
	return product, err
}
