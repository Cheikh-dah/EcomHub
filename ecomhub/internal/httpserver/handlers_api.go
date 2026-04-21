package httpserver

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"ecomhub/internal/middleware"
	"ecomhub/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var subdomainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

type storeBody struct {
	Name        string `json:"name" binding:"required"`
	Subdomain   string `json:"subdomain" binding:"required"`
	Description string `json:"description"`
}

type productBody struct {
	StoreID     int64   `json:"store_id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gte=0"`
	Stock       int     `json:"stock" binding:"gte=0"`
	ImageURL    string  `json:"image_url"`
}

type productUpdateBody struct {
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	Stock       *int     `json:"stock"`
	ImageURL    *string  `json:"image_url"`
}

type cartAddBody struct {
	ProductID int64 `json:"product_id" binding:"required"`
	Quantity  int   `json:"quantity" binding:"required,min=1"`
}

type cartRemoveBody struct {
	ProductID int64 `json:"product_id" binding:"required"`
}

type orderCreateBody struct {
	StoreID int64 `json:"store_id"`
}

func (s *Server) apiLogout(c *gin.Context) {
	clearAuthCookie(c, s.cfg.Environment)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// apiMe returns the internal user id after Clerk session JWT resolution (sub → user_identities, provider clerk).
func (s *Server) apiMe(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": uid.String()})
}

func (s *Server) apiListStores(c *gin.Context) {
	uid, _ := middleware.UserID(c)
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT id, user_id, name, subdomain, description, status, created_at FROM stores WHERE user_id = $1 ORDER BY id`,
		uid,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	var out []models.Store
	for rows.Next() {
		var st models.Store
		if err := rows.Scan(&st.ID, &st.UserID, &st.Name, &st.Subdomain, &st.Description, &st.Status, &st.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, st)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) apiCreateStore(c *gin.Context) {
	var body storeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	sub := normalizeSubdomain(body.Subdomain)
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}
	uid, _ := middleware.UserID(c)
	var id int64
	err := s.pool.QueryRow(c.Request.Context(),
		`INSERT INTO stores (user_id, name, subdomain, description) VALUES ($1, $2, $3, $4) RETURNING id`,
		uid, strings.TrimSpace(body.Name), sub, strings.TrimSpace(body.Description),
	).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "subdomain taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "subdomain": sub})
}

func (s *Server) apiUpdateStore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body storeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	sub := normalizeSubdomain(body.Subdomain)
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}
	uid, _ := middleware.UserID(c)
	cmd, err := s.pool.Exec(c.Request.Context(),
		`UPDATE stores SET name = $1, subdomain = $2, description = $3 WHERE id = $4 AND user_id = $5`,
		strings.TrimSpace(body.Name), sub, strings.TrimSpace(body.Description), id, uid,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			c.JSON(http.StatusConflict, gin.H{"error": "subdomain taken"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) assertStoreOwner(ctx context.Context, userID uuid.UUID, storeID int64) bool {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM stores WHERE id = $1 AND user_id = $2)`, storeID, userID).Scan(&ok)
	return err == nil && ok
}

func (s *Server) apiListProducts(c *gin.Context) {
	storeIDStr := c.Query("store_id")
	if storeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store_id required"})
		return
	}
	storeID, err := strconv.ParseInt(storeIDStr, 10, 64)
	if err != nil || storeID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid store_id"})
		return
	}
	uid, _ := middleware.UserID(c)
	if !s.assertStoreOwner(c.Request.Context(), uid, storeID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT id, store_id, name, description, price::float8, stock, COALESCE(image_url,''), created_at FROM products WHERE store_id = $1 ORDER BY id`,
		storeID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	var out []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) apiCreateProduct(c *gin.Context) {
	var body productBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	uid, _ := middleware.UserID(c)
	if !s.assertStoreOwner(c.Request.Context(), uid, body.StoreID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var id int64
	err := s.pool.QueryRow(c.Request.Context(),
		`INSERT INTO products (store_id, name, description, price, stock, image_url) VALUES ($1, $2, $3, $4, $5, NULLIF($6,'')) RETURNING id`,
		body.StoreID, strings.TrimSpace(body.Name), strings.TrimSpace(body.Description), body.Price, body.Stock, strings.TrimSpace(body.ImageURL),
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) apiUpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body productUpdateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	uid, _ := middleware.UserID(c)
	var storeID int64
	err = s.pool.QueryRow(c.Request.Context(),
		`SELECT store_id FROM products WHERE id = $1`, id,
	).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	if !s.assertStoreOwner(c.Request.Context(), uid, storeID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	setName := body.Name != nil
	nameVal := ""
	if body.Name != nil {
		nameVal = strings.TrimSpace(*body.Name)
	}
	setDesc := body.Description != nil
	descVal := ""
	if body.Description != nil {
		descVal = strings.TrimSpace(*body.Description)
	}
	setPrice := body.Price != nil
	priceVal := 0.0
	if body.Price != nil {
		priceVal = *body.Price
	}
	setStock := body.Stock != nil
	stockVal := 0
	if body.Stock != nil {
		stockVal = *body.Stock
	}
	setImg := body.ImageURL != nil
	imgVal := ""
	if body.ImageURL != nil {
		imgVal = strings.TrimSpace(*body.ImageURL)
	}
	cmd, err := s.pool.Exec(c.Request.Context(),
		`UPDATE products SET
			name = CASE WHEN $1 THEN $2::text ELSE name END,
			description = CASE WHEN $3 THEN $4::text ELSE description END,
			price = CASE WHEN $5 THEN $6::numeric ELSE price END,
			stock = CASE WHEN $7 THEN $8::int ELSE stock END,
			image_url = CASE WHEN $9 THEN NULLIF($10::text, '') ELSE image_url END
		WHERE id = $11`,
		setName, nameVal,
		setDesc, descVal,
		setPrice, priceVal,
		setStock, stockVal,
		setImg, imgVal,
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) apiDeleteProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	uid, _ := middleware.UserID(c)
	var storeID int64
	err = s.pool.QueryRow(c.Request.Context(), `SELECT store_id FROM products WHERE id = $1`, id).Scan(&storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	if !s.assertStoreOwner(c.Request.Context(), uid, storeID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	_, err = s.pool.Exec(c.Request.Context(), `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) apiGetCart(c *gin.Context) {
	cart, err := readCart(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cart"})
		return
	}
	lines, total, err := s.resolveCartLines(c.Request.Context(), cart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"store_id": cart.StoreID, "lines": lines, "total": total})
}

func (s *Server) apiCartAdd(c *gin.Context) {
	var body cartAddBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := s.mergeCartLine(c, body.ProductID, body.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) apiCartRemove(c *gin.Context) {
	var body cartRemoveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	cart, err := readCart(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cart"})
		return
	}
	out := cart.Lines[:0]
	for _, ln := range cart.Lines {
		if ln.ProductID != body.ProductID {
			out = append(out, ln)
		}
	}
	cart.Lines = out
	if len(cart.Lines) == 0 {
		cart.StoreID = 0
	}
	_ = writeCart(c, cart)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) apiCartClear(c *gin.Context) {
	clearCartCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Server) apiCreateOrder(c *gin.Context) {
	var body orderCreateBody
	_ = c.ShouldBindJSON(&body)
	uid, _ := middleware.UserID(c)
	cart, err := readCart(c)
	if err != nil || len(cart.Lines) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cart empty"})
		return
	}
	storeID := cart.StoreID
	if body.StoreID > 0 && body.StoreID != storeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "store mismatch"})
		return
	}
	if storeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cart"})
		return
	}
	orderID, total, err := s.placeOrder(c.Request.Context(), uid, storeID, cart)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	clearCartCookie(c)
	c.JSON(http.StatusCreated, gin.H{"order_id": orderID, "total": total})
}

func (s *Server) apiListOrders(c *gin.Context) {
	uid, _ := middleware.UserID(c)
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT id, store_id, user_id, total_price::float8, status, created_at FROM orders WHERE user_id = $1 ORDER BY id DESC`,
		uid,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	var out []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.StoreID, &o.UserID, &o.TotalPrice, &o.Status, &o.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan failed"})
			return
		}
		out = append(out, o)
	}
	c.JSON(http.StatusOK, out)
}

// mergeCartLine loads product store and stock, validates single-store cart.
func (s *Server) mergeCartLine(c *gin.Context, productID int64, qty int) error {
	var storeID int64
	var stock int
	err := s.pool.QueryRow(c.Request.Context(),
		`SELECT store_id, stock FROM products WHERE id = $1`, productID,
	).Scan(&storeID, &stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("product not found")
	}
	if err != nil {
		return errors.New("lookup failed")
	}
	if qty > stock {
		return errors.New("insufficient stock")
	}
	cart, err := readCart(c)
	if err != nil {
		return errors.New("invalid cart")
	}
	if cart.StoreID != 0 && cart.StoreID != storeID {
		return errors.New("cart is for a different store; clear cart first")
	}
	cart.StoreID = storeID
	found := false
	for i := range cart.Lines {
		if cart.Lines[i].ProductID == productID {
			newQ := cart.Lines[i].Quantity + qty
			if newQ > stock {
				return errors.New("insufficient stock")
			}
			cart.Lines[i].Quantity = newQ
			found = true
			break
		}
	}
	if !found {
		cart.Lines = append(cart.Lines, models.CartLine{ProductID: productID, Quantity: qty})
	}
	return writeCart(c, cart)
}

type resolvedLine struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	LineTotal float64 `json:"line_total"`
}

func (s *Server) resolveCartLines(ctx context.Context, cart models.CartPayload) ([]resolvedLine, float64, error) {
	if cart.StoreID == 0 || len(cart.Lines) == 0 {
		return nil, 0, nil
	}
	ids := sortedUniqueProductIDs(cart.Lines)
	prows, err := fetchProductsByStore(ctx, s.pool, cart.StoreID, ids, false)
	if err != nil {
		return nil, 0, err
	}
	if len(prows) != len(ids) {
		return nil, 0, errors.New("product not found in cart")
	}
	var total float64
	var out []resolvedLine
	for _, ln := range cart.Lines {
		p, ok := prows[ln.ProductID]
		if !ok {
			return nil, 0, errors.New("product not found in cart")
		}
		if ln.Quantity > p.Stock {
			return nil, 0, errors.New("insufficient stock for " + p.Name)
		}
		lineTotal := p.Price * float64(ln.Quantity)
		total += lineTotal
		out = append(out, resolvedLine{
			ProductID: ln.ProductID, Name: p.Name, Quantity: ln.Quantity, UnitPrice: p.Price, LineTotal: lineTotal,
		})
	}
	return out, total, nil
}

func (s *Server) placeOrder(ctx context.Context, userID uuid.UUID, storeID int64, cart models.CartPayload) (int64, float64, error) {
	if len(cart.Lines) == 0 {
		return 0, 0, errors.New("empty cart")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var storeStatus string
	err = tx.QueryRow(ctx, `SELECT status FROM stores WHERE id = $1`, storeID).Scan(&storeStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, errors.New("store not found")
	}
	if err != nil {
		return 0, 0, err
	}
	if storeStatus != "active" {
		return 0, 0, errors.New("store is not accepting orders")
	}

	ids := sortedUniqueProductIDs(cart.Lines)
	prows, err := fetchProductsByStore(ctx, tx, storeID, ids, true)
	if err != nil {
		return 0, 0, err
	}
	if len(prows) != len(ids) {
		return 0, 0, errors.New("product missing")
	}

	var total float64
	type line struct {
		pid  int64
		qty  int
		unit float64
	}
	var lines []line
	for _, ln := range cart.Lines {
		p, ok := prows[ln.ProductID]
		if !ok {
			return 0, 0, errors.New("product missing")
		}
		if ln.Quantity > p.Stock {
			return 0, 0, errors.New("insufficient stock")
		}
		lines = append(lines, line{pid: ln.ProductID, qty: ln.Quantity, unit: p.Price})
		total += p.Price * float64(ln.Quantity)
	}

	var orderID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (store_id, user_id, total_price, status) VALUES ($1, $2, $3, 'paid') RETURNING id`,
		storeID, userID, total,
	).Scan(&orderID)
	if err != nil {
		return 0, 0, err
	}
	for _, ln := range lines {
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, price) VALUES ($1, $2, $3, $4)`,
			orderID, ln.pid, ln.qty, ln.unit,
		)
		if err != nil {
			return 0, 0, err
		}
		_, err = tx.Exec(ctx, `UPDATE products SET stock = stock - $1 WHERE id = $2`, ln.qty, ln.pid)
		if err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, err
	}
	return orderID, total, nil
}

type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type productStockRow struct {
	ID      int64
	StoreID int64
	Name    string
	Price   float64
	Stock   int
}

func sortedUniqueProductIDs(lines []models.CartLine) []int64 {
	seen := make(map[int64]struct{})
	var ids []int64
	for _, ln := range lines {
		if ln.ProductID < 1 {
			continue
		}
		if _, ok := seen[ln.ProductID]; ok {
			continue
		}
		seen[ln.ProductID] = struct{}{}
		ids = append(ids, ln.ProductID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// fetchProductsByStore loads products for a store by id list. If forUpdate is true, rows are locked in id order.
func fetchProductsByStore(ctx context.Context, q dbQuerier, storeID int64, ids []int64, forUpdate bool) (map[int64]productStockRow, error) {
	out := make(map[int64]productStockRow)
	if len(ids) == 0 {
		return out, nil
	}
	sql := `SELECT id, store_id, name, price::float8, stock FROM products WHERE store_id = $1 AND id = ANY($2::bigint[]) ORDER BY id`
	if forUpdate {
		sql += ` FOR UPDATE`
	}
	rows, err := q.Query(ctx, sql, storeID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p productStockRow
		if scanErr := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Price, &p.Stock); scanErr != nil {
			return nil, scanErr
		}
		out[p.ID] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
