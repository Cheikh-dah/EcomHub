package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

type publicHubProductDTO struct {
	publicProductDTO
	Store publicStoreSummaryDTO `json:"store"`
}

type publicHubProductsResponse struct {
	Products   []publicHubProductDTO `json:"products"`
	Pagination publicPaginationDTO   `json:"pagination"`
}

type publicHubStoresResponse struct {
	Stores     []publicStoreDTO    `json:"stores"`
	Pagination publicPaginationDTO `json:"pagination"`
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

type publicHubProductRow struct {
	Product models.Product
	Store   models.Store
}

type publicStoreLoader func(context.Context, string) (models.Store, error)
type publicThemeLoader func(context.Context, int64) (models.StoreTheme, error)
type publicProductsLoader func(context.Context, int64, int, int) ([]models.Product, error)
type publicProductLoader func(context.Context, int64, int64) (models.Product, error)
type publicHubProductsLoader func(context.Context, int, int, string) ([]publicHubProductRow, error)
type publicHubStoresLoader func(context.Context, int, int, string) ([]models.Store, error)

const (
	defaultPublicProductsLimit = 24
	maxPublicProductsLimit     = 50
	maxPublicSearchLength      = 100
)

func (s *Server) apiPublicStoreBySubdomain(c *gin.Context) {
	apiPublicStoreBySubdomain(c, s.loadStoreBySubdomain, s.loadStoreThemeByID)
}

func (s *Server) apiPublicHubProducts(c *gin.Context) {
	apiPublicHubProducts(c, s.loadPublicHubProducts)
}

func apiPublicHubProducts(c *gin.Context, loadProducts publicHubProductsLoader) {
	pagination, search, ok := parsePublicHubQuery(c)
	if !ok {
		return
	}

	rows, err := loadProducts(c.Request.Context(), pagination.Limit+1, pagination.Offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "products query failed"})
		return
	}

	hasMore := len(rows) > pagination.Limit
	if hasMore {
		rows = rows[:pagination.Limit]
	}

	out := make([]publicHubProductDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, publicHubProductDTO{
			publicProductDTO: publicProductToDTO(row.Product),
			Store: publicStoreSummaryDTO{
				ID:        row.Store.ID,
				Name:      row.Store.Name,
				Subdomain: row.Store.Subdomain,
			},
		})
	}

	c.JSON(http.StatusOK, publicHubProductsResponse{
		Products: out,
		Pagination: publicPaginationDTO{
			Limit:   pagination.Limit,
			Offset:  pagination.Offset,
			Count:   len(out),
			HasMore: hasMore,
		},
	})
}

func (s *Server) apiPublicHubStores(c *gin.Context) {
	apiPublicHubStores(c, s.loadPublicHubStores)
}

func apiPublicHubStores(c *gin.Context, loadStores publicHubStoresLoader) {
	pagination, search, ok := parsePublicHubQuery(c)
	if !ok {
		return
	}

	stores, err := loadStores(c.Request.Context(), pagination.Limit+1, pagination.Offset, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "stores query failed"})
		return
	}

	hasMore := len(stores) > pagination.Limit
	if hasMore {
		stores = stores[:pagination.Limit]
	}

	out := make([]publicStoreDTO, 0, len(stores))
	for _, store := range stores {
		out = append(out, publicStoreToDTO(store))
	}

	c.JSON(http.StatusOK, publicHubStoresResponse{
		Stores: out,
		Pagination: publicPaginationDTO{
			Limit:   pagination.Limit,
			Offset:  pagination.Offset,
			Count:   len(out),
			HasMore: hasMore,
		},
	})
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
		Store: publicStoreToDTO(store),
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

func publicStoreToDTO(store models.Store) publicStoreDTO {
	return publicStoreDTO{
		ID:          store.ID,
		Name:        store.Name,
		Subdomain:   store.Subdomain,
		Description: store.Description,
		Status:      store.Status,
		CreatedAt:   store.CreatedAt,
	}
}

func parsePublicHubQuery(c *gin.Context) (publicPaginationParams, string, bool) {
	pagination, err := parsePublicPagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return publicPaginationParams{}, "", false
	}

	search := strings.TrimSpace(c.Query("search"))
	if len(search) > maxPublicSearchLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search must be 100 characters or less"})
		return publicPaginationParams{}, "", false
	}

	return pagination, search, true
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

func (s *Server) loadPublicHubProducts(ctx context.Context, limit int, offset int, search string) ([]publicHubProductRow, error) {
	args := []any{limit, offset}
	where := "WHERE stores.status = 'active'"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (products.name ILIKE $3 OR products.description ILIKE $3 OR stores.name ILIKE $3 OR stores.subdomain ILIKE $3)"
	}

	rows, err := s.pool.Query(ctx,
		`SELECT products.id, products.store_id, products.name, products.description, products.price::float8, products.stock, COALESCE(products.image_url,''), products.created_at,
		        stores.id, stores.user_id, stores.name, stores.subdomain, stores.description, stores.status, stores.created_at
		 FROM products
		 JOIN stores ON stores.id = products.store_id
		 `+where+`
		 ORDER BY products.id DESC
		 LIMIT $1 OFFSET $2`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []publicHubProductRow
	for rows.Next() {
		var row publicHubProductRow
		if err := rows.Scan(
			&row.Product.ID,
			&row.Product.StoreID,
			&row.Product.Name,
			&row.Product.Description,
			&row.Product.Price,
			&row.Product.Stock,
			&row.Product.ImageURL,
			&row.Product.CreatedAt,
			&row.Store.ID,
			&row.Store.UserID,
			&row.Store.Name,
			&row.Store.Subdomain,
			&row.Store.Description,
			&row.Store.Status,
			&row.Store.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) loadPublicHubStores(ctx context.Context, limit int, offset int, search string) ([]models.Store, error) {
	args := []any{limit, offset}
	where := "WHERE stores.status = 'active'"
	if search != "" {
		args = append(args, "%"+search+"%")
		where += " AND (stores.name ILIKE $3 OR stores.subdomain ILIKE $3 OR stores.description ILIKE $3)"
	}

	rows, err := s.pool.Query(ctx,
		`SELECT stores.id, stores.user_id, stores.name, stores.subdomain, stores.description, stores.status, stores.created_at
		 FROM stores
		 `+where+`
		 ORDER BY stores.id DESC
		 LIMIT $1 OFFSET $2`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []models.Store
	for rows.Next() {
		var store models.Store
		if err := rows.Scan(
			&store.ID,
			&store.UserID,
			&store.Name,
			&store.Subdomain,
			&store.Description,
			&store.Status,
			&store.CreatedAt,
		); err != nil {
			return nil, err
		}
		stores = append(stores, store)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return stores, nil
}
