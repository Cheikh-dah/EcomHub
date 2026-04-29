package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"ecomhub/internal/auth"
	"ecomhub/internal/config"
	"ecomhub/internal/middleware"
	"ecomhub/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type hubProductRow struct {
	ID        int64
	Name      string
	Price     float64
	StoreName string
	Subdomain string
}

type cartLineView struct {
	ProductID int64
	Name      string
	Quantity  int
	LineTotal float64
}

func (s *Server) loadOwnedStore(ctx context.Context, userID uuid.UUID, storeID int64) (models.Store, error) {
	var st models.Store
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, subdomain, description, status, created_at
		 FROM stores WHERE id = $1 AND user_id = $2`,
		storeID, userID,
	).Scan(&st.ID, &st.UserID, &st.Name, &st.Subdomain, &st.Description, &st.Status, &st.CreatedAt)
	return st, err
}

func (s *Server) hubProductsHTML(c *gin.Context) {
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT p.id, p.name, p.price::float8, s.name, s.subdomain
		 FROM products p JOIN stores s ON s.id = p.store_id
		 WHERE s.status = 'active'
		 ORDER BY p.id DESC LIMIT 200`,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}
	defer rows.Close()
	var list []hubProductRow
	for rows.Next() {
		var r hubProductRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Price, &r.StoreName, &r.Subdomain); err != nil {
			c.String(http.StatusInternalServerError, "error")
			return
		}
		list = append(list, r)
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "hub_products", gin.H{"Products": list})
	if err != nil {
		log.Printf("hub_products render error: %v", err)
	}
}

func (s *Server) hubStoresHTML(c *gin.Context) {
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT name, subdomain FROM stores WHERE status = 'active' ORDER BY id DESC LIMIT 200`,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}
	defer rows.Close()
	var list []models.Store
	for rows.Next() {
		var st models.Store
		if err := rows.Scan(&st.Name, &st.Subdomain); err != nil {
			c.String(http.StatusInternalServerError, "error")
			return
		}
		list = append(list, st)
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "hub_stores", gin.H{"Stores": list})
	if err != nil {
		log.Printf("hub_stores render error: %v", err)
	}
}

func (s *Server) hubSearchHTML(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	data := gin.H{"Query": q, "Products": []hubProductRow{}, "Stores": []models.Store{}}
	if q == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		err := s.tmpl.ExecuteTemplate(c.Writer, "hub_search", data)
		if err != nil {
			log.Printf("hub_search render error: %v", err)
		}
		return
	}
	pat := "%" + q + "%"
	prows, err := s.pool.Query(c.Request.Context(),
		`SELECT p.id, p.name, p.price::float8, s.name, s.subdomain
		 FROM products p JOIN stores s ON s.id = p.store_id
		 WHERE s.status = 'active' AND (p.name ILIKE $1 OR p.description ILIKE $1)
		 ORDER BY p.id DESC LIMIT 50`, pat,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}
	defer prows.Close()
	var plist []hubProductRow
	for prows.Next() {
		var r hubProductRow
		if err := prows.Scan(&r.ID, &r.Name, &r.Price, &r.StoreName, &r.Subdomain); err != nil {
			c.String(http.StatusInternalServerError, "error")
			return
		}
		plist = append(plist, r)
	}
	srows, err := s.pool.Query(c.Request.Context(),
		`SELECT name, subdomain FROM stores WHERE status = 'active' AND (name ILIKE $1 OR description ILIKE $1 OR subdomain ILIKE $1) ORDER BY id DESC LIMIT 50`,
		pat,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "error")
		return
	}
	defer srows.Close()
	var slist []models.Store
	for srows.Next() {
		var st models.Store
		if err := srows.Scan(&st.Name, &st.Subdomain); err != nil {
			c.String(http.StatusInternalServerError, "error")
			return
		}
		slist = append(slist, st)
	}
	data["Products"] = plist
	data["Stores"] = slist
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "hub_search", data)
	if err != nil {
		log.Printf("hub_search render error: %v", err)
	}
}

func (s *Server) loadStoreBySubdomain(ctx context.Context, sub string) (models.Store, error) {
	var st models.Store
	err := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, subdomain, description, status, created_at FROM stores WHERE subdomain = $1 AND status = 'active'`,
		sub,
	).Scan(&st.ID, &st.UserID, &st.Name, &st.Subdomain, &st.Description, &st.Status, &st.CreatedAt)
	return st, err
}

func (s *Server) storeHomeHTML(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	st, err := s.loadStoreBySubdomain(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	theme, err := s.loadStoreThemeByID(c.Request.Context(), st.ID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	rows, err := s.pool.Query(c.Request.Context(),
		`SELECT id, store_id, name, description, price::float8, stock, COALESCE(image_url,''), created_at FROM products WHERE store_id = $1 ORDER BY id`,
		st.ID,
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.CreatedAt); err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		products = append(products, p)
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "store_home", gin.H{"Store": st, "Products": products, "Theme": theme})
	if err != nil {
		log.Printf("store_home render error: %v", err)
	}
}

func (s *Server) storeProductHTML(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	st, err := s.loadStoreBySubdomain(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	theme, err := s.loadStoreThemeByID(c.Request.Context(), st.ID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || pid < 1 {
		c.Status(http.StatusBadRequest)
		return
	}
	var p models.Product
	err = s.pool.QueryRow(c.Request.Context(),
		`SELECT id, store_id, name, description, price::float8, stock, COALESCE(image_url,''), created_at FROM products WHERE id = $1 AND store_id = $2`,
		pid, st.ID,
	).Scan(&p.ID, &p.StoreID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "store_product", gin.H{"Store": st, "Product": p, "Theme": theme, "Error": c.Query("err")})
	if err != nil {
		log.Printf("store_product render error: %v", err)
	}
}

func (s *Server) storeCartAdd(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if _, err := s.loadStoreBySubdomain(c.Request.Context(), sub); errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	} else if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart?err=form")
		return
	}
	pid, err := strconv.ParseInt(c.PostForm("product_id"), 10, 64)
	if err != nil || pid < 1 {
		c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart?err=product")
		return
	}
	qty, err := strconv.Atoi(c.PostForm("quantity"))
	if err != nil || qty < 1 {
		qty = 1
	}
	if err := s.mergeCartLine(c, pid, qty); err != nil {
		c.Redirect(http.StatusSeeOther, "/s/"+sub+"/products/"+strconv.FormatInt(pid, 10)+"?err="+url.QueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart")
}

func (s *Server) storeCartRemove(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	if _, err := s.loadStoreBySubdomain(c.Request.Context(), sub); errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	} else if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	_ = c.Request.ParseForm()
	pid, err := strconv.ParseInt(c.PostForm("product_id"), 10, 64)
	if err == nil && pid > 0 {
		cart, _ := readCart(c)
		var out []models.CartLine
		for _, ln := range cart.Lines {
			if ln.ProductID != pid {
				out = append(out, ln)
			}
		}
		cart.Lines = out
		if len(cart.Lines) == 0 {
			cart.StoreID = 0
		}
		_ = writeCart(c, cart)
	}
	c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart")
}

func (s *Server) storeCartHTML(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	st, err := s.loadStoreBySubdomain(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	theme, err := s.loadStoreThemeByID(c.Request.Context(), st.ID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	cart, err := readCart(c)
	if err != nil {
		clearCartCookie(c)
		cart = models.CartPayload{}
	}
	var errMsg string
	if cart.StoreID != 0 && cart.StoreID != st.ID {
		errMsg = "Your cart contains items from another store. Clear it from the API or finish that order first."
	}
	var lines []cartLineView
	var total float64
	if errMsg == "" && cart.StoreID == st.ID && len(cart.Lines) > 0 {
		ids := sortedUniqueProductIDs(cart.Lines)
		prows, e := fetchProductsByStore(c.Request.Context(), s.pool, st.ID, ids, false)
		if e != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		for _, ln := range cart.Lines {
			p, ok := prows[ln.ProductID]
			if !ok {
				continue
			}
			if p.StoreID != st.ID {
				continue
			}
			lt := p.Price * float64(ln.Quantity)
			total += lt
			lines = append(lines, cartLineView{ProductID: ln.ProductID, Name: p.Name, Quantity: ln.Quantity, LineTotal: lt})
		}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "store_cart", gin.H{
		"Store": st, "Theme": theme, "Lines": lines, "Total": total, "Error": errMsg,
		"Err": c.Query("err"), "Thanks": c.Query("thanks") != "",
	})
	if err != nil {
		log.Printf("store_cart render error: %v", err)
	}
}

func (s *Server) storeCheckout(c *gin.Context) {
	sub := normalizeSubdomain(c.Param("subdomain"))
	st, err := s.loadStoreBySubdomain(c.Request.Context(), sub)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	uid, ok := middleware.UserID(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/dashboard?next=/s/"+sub+"/cart")
		return
	}
	cart, err := readCart(c)
	if err != nil || cart.StoreID != st.ID || len(cart.Lines) == 0 {
		c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart?err=empty")
		return
	}
	_, _, err = s.placeOrder(c.Request.Context(), uid, st.ID, cart)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart?err="+url.QueryEscape(err.Error()))
		return
	}
	clearCartCookie(c)
	c.Redirect(http.StatusSeeOther, "/s/"+sub+"/cart?thanks=1")
}

// safeInternalRedirectPath returns a same-origin path for post-login redirects.
// Rejects empty, non-root-relative, protocol-relative, and absolute-URL values.
func safeInternalRedirectPath(next string) string {
	next = strings.TrimSpace(next)
	next = strings.ReplaceAll(next, "\r", "")
	next = strings.ReplaceAll(next, "\n", "")
	if next == "" {
		return "/dashboard"
	}
	if !strings.HasPrefix(next, "/") {
		return "/dashboard"
	}
	if strings.HasPrefix(next, "//") {
		return "/dashboard"
	}
	if strings.Contains(strings.ToLower(next), "://") {
		return "/dashboard"
	}
	return next
}

type dashboardData struct {
	LoggedIn           bool
	Token              bool
	Stores             []models.Store
	Error              string
	ClerkBootstrapJSON template.JS // raw JSON for <script type="application/json"> (avoids broken JS parse in IDEs)
	Theme              models.StoreTheme
	Store              models.Store
}

func clerkBootstrapJSON(cfg config.Config) template.JS {
	b, err := json.Marshal(map[string]string{
		"publishableKey": cfg.ClerkPublishableKey,
		"frontendAPI":    cfg.ClerkFrontendAPI,
	})
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b)
}

func (s *Server) dashboardSession(c *gin.Context) {
	var body struct {
		AccessToken  string `json:"access_token"`
		SessionToken string `json:"session_token"`
		Next         string `json:"next"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	tok := strings.TrimSpace(body.AccessToken)
	if tok == "" {
		tok = strings.TrimSpace(body.SessionToken)
	}
	if tok == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "access_token or session_token required"})
		return
	}
	ctx := c.Request.Context()
	clerkUserID, maxAge, err := auth.VerifyClerkSessionJWT(ctx, tok, s.cfg.ClerkAuthorizedParties)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if _, err := auth.ResolveClerkUser(ctx, s.pool, clerkUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not resolve user"})
		return
	}
	setAuthCookie(c, tok, s.cfg.Environment, maxAge)
	redirect := safeInternalRedirectPath(body.Next)
	c.JSON(http.StatusOK, gin.H{"ok": true, "redirect": redirect})
}

func (s *Server) dashboardGet(c *gin.Context) {
	uid, ok := middleware.UserID(c)
	if ok {
		if n := strings.TrimSpace(c.Query("next")); n != "" {
			dest := safeInternalRedirectPath(n)
			if dest != "/dashboard" {
				c.Redirect(http.StatusSeeOther, dest)
				return
			}
		}
	}
	data := dashboardData{
		LoggedIn:           ok,
		Token:              ok,
		ClerkBootstrapJSON: clerkBootstrapJSON(s.cfg),
	}
	switch strings.TrimSpace(c.Query("err")) {
	case "invalid_store":
		data.Error = "Invalid store name or subdomain. Use letters, numbers, and hyphens only; do not start or end with a hyphen (max 63 characters). Subdomains are saved in lowercase."
	case "taken":
		data.Error = "That subdomain is already taken, or the store could not be saved. Pick a different subdomain."
	}
	if ok {
		rows, err := s.pool.Query(c.Request.Context(),
			`SELECT id, user_id, name, subdomain, description, status, created_at FROM stores WHERE user_id = $1 ORDER BY id`,
			uid,
		)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var st models.Store
				if err := rows.Scan(&st.ID, &st.UserID, &st.Name, &st.Subdomain, &st.Description, &st.Status, &st.CreatedAt); err == nil {
					data.Stores = append(data.Stores, st)
				}
			}
		}
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	err := s.tmpl.ExecuteTemplate(c.Writer, "dashboard", data)
	if err != nil {
		log.Printf("dashboard render error: %v", err)
	}
}

func (s *Server) dashboardLogout(c *gin.Context) {
	clearAuthCookie(c, s.cfg.Environment)
	c.Redirect(http.StatusSeeOther, "/dashboard?signed_out=1")
}

func (s *Server) dashboardErr(c *gin.Context, msg string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	err := s.tmpl.ExecuteTemplate(c.Writer, "dashboard", dashboardData{
		Error:              msg,
		ClerkBootstrapJSON: clerkBootstrapJSON(s.cfg),
	})
	if err != nil {
		log.Printf("dashboardErr render error: %v", err)
	}
}

func (s *Server) dashboardCreateStore(c *gin.Context) {
	_ = c.Request.ParseForm()
	name := strings.TrimSpace(c.PostForm("name"))
	sub := normalizeSubdomain(c.PostForm("subdomain"))
	desc := strings.TrimSpace(c.PostForm("description"))
	if name == "" || !subdomainRe.MatchString(sub) {
		c.Redirect(http.StatusSeeOther, "/dashboard?err=invalid_store")
		return
	}
	uid, _ := middleware.UserID(c)
	_, err := s.pool.Exec(c.Request.Context(),
		`INSERT INTO stores (user_id, name, subdomain, description) VALUES ($1, $2, $3, $4)`,
		uid, name, sub, desc,
	)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/dashboard?err=taken")
		return
	}
	c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (s *Server) dashboardStoreThemeGet(c *gin.Context) {
	storeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || storeID < 1 {
		c.Status(http.StatusBadRequest)
		return
	}

	uid, ok := middleware.UserID(c)
	if !ok {
		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	st, err := s.loadOwnedStore(c.Request.Context(), uid, storeID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	theme, err := s.loadStoreThemeByID(c.Request.Context(), storeID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	err = s.tmpl.ExecuteTemplate(c.Writer, "theme_editor", gin.H{
		"Store":              st,
		"Theme":              theme,
		"ClerkBootstrapJSON": clerkBootstrapJSON(s.cfg),
	})
	if err != nil {
		log.Printf("theme_editor render error: %v", err)
	}
}

