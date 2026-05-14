package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	neturl "net/url"
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
var hexColorRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)

const maxProductImageURLLength = 2048

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

type productUpdateValues struct {
	SetName  bool
	Name     string
	SetDesc  bool
	Desc     string
	SetPrice bool
	Price    float64
	SetStock bool
	Stock    int
	SetImage bool
	ImageURL string
}

type storeThemeBody struct {
	PrimaryColor string  `json:"primary_color"`
	AccentColor  string  `json:"accent_color"`
	LogoURL      string  `json:"logo_url"`
	LayoutPreset string  `json:"layout_preset"`
	Preset       string  `json:"preset"`
	Rounding     float64 `json:"rounding"`
	Version      int     `json:"version"`
	PageBg       string  `json:"page_bg"`
	TextColor    string  `json:"text_color"`
	CardBg       string  `json:"card_bg"`
	FooterBg     string  `json:"footer_bg"`
}

type storeThemeUpdateBody struct {
	PrimaryColor *string  `json:"primary_color"`
	AccentColor  *string  `json:"accent_color"`
	LogoURL      *string  `json:"logo_url"`
	LayoutPreset *string  `json:"layout_preset"`
	Preset       *string  `json:"preset"`
	Rounding     *float64 `json:"rounding"`
	Version      *int     `json:"version"`
	PageBg       *string  `json:"page_bg"`
	TextColor    *string  `json:"text_color"`
	CardBg       *string  `json:"card_bg"`
	FooterBg     *string  `json:"footer_bg"`
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

func normalizeStoreName(v string) (string, error) {
	name := strings.TrimSpace(v)
	if name == "" {
		return "", errors.New("store name is required")
	}
	return name, nil
}

func normalizeProductUpdate(body productUpdateBody) (productUpdateValues, error) {
	out := productUpdateValues{}
	if body.Name != nil {
		out.SetName = true
		out.Name = strings.TrimSpace(*body.Name)
		if out.Name == "" {
			return out, errors.New("product name is required")
		}
	}
	if body.Description != nil {
		out.SetDesc = true
		out.Desc = strings.TrimSpace(*body.Description)
	}
	if body.Price != nil {
		out.SetPrice = true
		out.Price = *body.Price
		if out.Price < 0 {
			return out, errors.New("price must be greater than or equal to 0")
		}
	}
	if body.Stock != nil {
		out.SetStock = true
		out.Stock = *body.Stock
		if out.Stock < 0 {
			return out, errors.New("stock must be greater than or equal to 0")
		}
	}
	if body.ImageURL != nil {
		out.SetImage = true
		imageURL, err := normalizeProductImageURL(*body.ImageURL)
		if err != nil {
			return out, err
		}
		out.ImageURL = imageURL
	}
	return out, nil
}

func normalizeProductImageURL(v string) (string, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return "", nil
	}
	if len(s) > maxProductImageURLLength {
		return "", errors.New("image_url is too long")
	}
	u, err := neturl.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid image_url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("invalid image_url")
	}
	return s, nil
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
	name, err := normalizeStoreName(body.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub := normalizeSubdomain(body.Subdomain)
	if !subdomainRe.MatchString(sub) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain"})
		return
	}
	uid, _ := middleware.UserID(c)
	var id int64
	err = s.pool.QueryRow(c.Request.Context(),
		`INSERT INTO stores (user_id, name, subdomain, description) VALUES ($1, $2, $3, $4) RETURNING id`,
		uid, name, sub, strings.TrimSpace(body.Description),
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

func defaultStoreTheme() models.StoreTheme {
	t := models.StoreTheme{
		PrimaryColor: "#1d9bf0",
		AccentColor:  "#00ba7c",
		LogoURL:      "",
		LayoutPreset: "default",
		Preset:       "minimal",
		Rounding:     0.5,
		Version:      1,
		PageBg:       "#ffffff",
		TextColor:    "#111111",
		CardBg:       "#ffffff",
		FooterBg:     "transparent",
	}
	t.Normalize()
	return t
}

func normalizeColor(v string, fallback string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return fallback, nil
	}
	if !hexColorRe.MatchString(s) {
		return "", errors.New("invalid color")
	}
	return s, nil
}

func normalizeFooterColor(v string, fallback string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "transparent" {
		return s, nil
	}
	return normalizeColor(s, fallback)
}

func normalizeLayoutPreset(v string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	if s == "" {
		return "default", nil
	}
	switch s {
	case "default", "compact":
		return s, nil
	default:
		return "", errors.New("invalid layout_preset")
	}
}

func normalizeLogoURL(v string) (string, error) {
	s := strings.TrimSpace(v)
	if s == "" {
		return "", nil
	}
	u, err := neturl.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid logo_url")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("invalid logo_url")
	}
	return s, nil
}

func normalizeStoreTheme(in storeThemeBody) (models.StoreTheme, error) {
	d := defaultStoreTheme()
	var out models.StoreTheme
	var err error

	out.PrimaryColor, err = normalizeColor(in.PrimaryColor, d.PrimaryColor)
	if err != nil {
		return out, err
	}
	out.AccentColor, err = normalizeColor(in.AccentColor, d.AccentColor)
	if err != nil {
		return out, err
	}
	out.LogoURL, err = normalizeLogoURL(in.LogoURL)
	if err != nil {
		return out, err
	}
	out.LayoutPreset, err = normalizeLayoutPreset(in.LayoutPreset)
	if err != nil {
		return out, err
	}

	// DNA v2
	out.Preset = in.Preset
	out.Rounding = in.Rounding
	out.Version = in.Version

	// Phase 2: Design Tokens
	out.PageBg, err = normalizeColor(in.PageBg, d.PageBg)
	if err != nil {
		return out, err
	}
	out.TextColor, err = normalizeColor(in.TextColor, d.TextColor)
	if err != nil {
		return out, err
	}
	out.CardBg, err = normalizeColor(in.CardBg, d.CardBg)
	if err != nil {
		return out, err
	}
	out.FooterBg, err = normalizeFooterColor(in.FooterBg, d.FooterBg)
	if err != nil {
		return out, err
	}

	out.Normalize()

	return out, nil
}

func normalizeStoreThemePatch(curr models.StoreTheme, patch storeThemeUpdateBody) (models.StoreTheme, error) {
	out := curr
	var err error
	if patch.PrimaryColor != nil {
		out.PrimaryColor, err = normalizeColor(*patch.PrimaryColor, curr.PrimaryColor)
		if err != nil {
			return out, err
		}
	}
	if patch.AccentColor != nil {
		out.AccentColor, err = normalizeColor(*patch.AccentColor, curr.AccentColor)
		if err != nil {
			return out, err
		}
	}
	if patch.LogoURL != nil {
		out.LogoURL, err = normalizeLogoURL(*patch.LogoURL)
		if err != nil {
			return out, err
		}
	}
	if patch.LayoutPreset != nil {
		out.LayoutPreset, err = normalizeLayoutPreset(*patch.LayoutPreset)
		if err != nil {
			return out, err
		}
	}

	// DNA v2
	if patch.Preset != nil {
		out.Preset = *patch.Preset
	}
	if patch.Rounding != nil {
		out.Rounding = *patch.Rounding
	}
	if patch.Version != nil {
		out.Version = *patch.Version
	}

	// Phase 2: Design Tokens
	if patch.PageBg != nil {
		out.PageBg, err = normalizeColor(*patch.PageBg, curr.PageBg)
		if err != nil {
			return out, err
		}
	}
	if patch.TextColor != nil {
		out.TextColor, err = normalizeColor(*patch.TextColor, curr.TextColor)
		if err != nil {
			return out, err
		}
	}
	if patch.CardBg != nil {
		out.CardBg, err = normalizeColor(*patch.CardBg, curr.CardBg)
		if err != nil {
			return out, err
		}
	}
	if patch.FooterBg != nil {
		out.FooterBg, err = normalizeFooterColor(*patch.FooterBg, curr.FooterBg)
		if err != nil {
			return out, err
		}
	}

	out.Normalize()

	return out, nil
}

func (s *Server) loadStoreThemeByID(ctx context.Context, storeID int64) (models.StoreTheme, error) {
	theme := defaultStoreTheme()
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT theme_config FROM stores WHERE id = $1`, storeID).Scan(&raw)
	if err != nil {
		return theme, err
	}
	if len(raw) == 0 {
		return theme, nil
	}
	var persisted storeThemeBody
	if err := json.Unmarshal(raw, &persisted); err != nil {
		return theme, nil
	}
	norm, err := normalizeStoreTheme(persisted)
	if err != nil {
		return theme, nil
	}
	return norm, nil
}

func (s *Server) apiGetStoreTheme(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	uid, _ := middleware.UserID(c)
	if !s.assertStoreOwner(c.Request.Context(), uid, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	theme, err := s.loadStoreThemeByID(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, theme)
}

func (s *Server) apiUpdateStoreTheme(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	uid, _ := middleware.UserID(c)
	if !s.assertStoreOwner(c.Request.Context(), uid, id) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	current, err := s.loadStoreThemeByID(c.Request.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	var body storeThemeUpdateBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	updated, err := normalizeStoreThemePatch(current, body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	payload, err := json.Marshal(updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode failed"})
		return
	}
	cmd, err := s.pool.Exec(c.Request.Context(),
		`UPDATE stores SET theme_config = $1::jsonb WHERE id = $2 AND user_id = $3`,
		payload, id, uid,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if cmd.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "store not found"})
		return
	}
	c.JSON(http.StatusOK, updated)
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
	imageURL, err := normalizeProductImageURL(body.ImageURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var id int64
	err = s.pool.QueryRow(c.Request.Context(),
		`INSERT INTO products (store_id, name, description, price, stock, image_url) VALUES ($1, $2, $3, $4, $5, NULLIF($6,'')) RETURNING id`,
		body.StoreID, strings.TrimSpace(body.Name), strings.TrimSpace(body.Description), body.Price, body.Stock, imageURL,
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
	update, err := normalizeProductUpdate(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd, err := s.pool.Exec(c.Request.Context(),
		`UPDATE products SET
			name = CASE WHEN $1 THEN $2::text ELSE name END,
			description = CASE WHEN $3 THEN $4::text ELSE description END,
			price = CASE WHEN $5 THEN $6::numeric ELSE price END,
			stock = CASE WHEN $7 THEN $8::int ELSE stock END,
			image_url = CASE WHEN $9 THEN NULLIF($10::text, '') ELSE image_url END
		WHERE id = $11`,
		update.SetName, update.Name,
		update.SetDesc, update.Desc,
		update.SetPrice, update.Price,
		update.SetStock, update.Stock,
		update.SetImage, update.ImageURL,
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
