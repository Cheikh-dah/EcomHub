package httpserver

import (
	"context"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"ecomhub/internal/config"
	"ecomhub/internal/middleware"
	"ecomhub/internal/web"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	pool *pgxpool.Pool
	cfg  config.Config
	tmpl *template.Template
}

func New(pool *pgxpool.Pool, cfg config.Config, tmpl *template.Template) *Server {
	return &Server{pool: pool, cfg: cfg, tmpl: tmpl}
}

func (s *Server) Mount(r *gin.Engine) {
	r.GET("/health", s.health)
	r.StaticFS("/static", http.FS(staticSubFS()))

	r.GET("/", s.home)

	r.GET("/products", s.hubProductsHTML)
	r.GET("/stores", s.hubStoresHTML)
	r.GET("/search", s.hubSearchHTML)

	api := r.Group("/api")
	{
		api.POST("/logout", s.apiLogout)

		authd := api.Group("")
		authd.Use(middleware.RequireAuth(s.pool, s.cfg.ClerkAuthorizedParties))
		{
			authd.GET("/me", s.apiMe)
			authd.GET("/stores", s.apiListStores)
			authd.POST("/stores", s.apiCreateStore)
			authd.PUT("/stores/:id", s.apiUpdateStore)

			authd.GET("/products", s.apiListProducts)
			authd.POST("/products", s.apiCreateProduct)
			authd.PUT("/products/:id", s.apiUpdateProduct)
			authd.DELETE("/products/:id", s.apiDeleteProduct)

			authd.GET("/cart", s.apiGetCart)
			authd.POST("/cart/add", s.apiCartAdd)
			authd.POST("/cart/remove", s.apiCartRemove)
			authd.POST("/cart/clear", s.apiCartClear)

			authd.POST("/orders", s.apiCreateOrder)
			authd.GET("/orders", s.apiListOrders)
		}
	}

	r.GET("/dashboard", middleware.OptionalAuth(s.pool, s.cfg.ClerkAuthorizedParties), s.dashboardGet)
	r.POST("/dashboard/session", s.dashboardSession)
	r.POST("/dashboard/logout", s.dashboardLogout)

	dashAuth := r.Group("/dashboard")
	dashAuth.Use(middleware.RequireAuth(s.pool, s.cfg.ClerkAuthorizedParties))
	dashAuth.POST("/stores", s.dashboardCreateStore)

	sub := r.Group("/s/:subdomain")
	{
		sub.GET("", s.storeHomeHTML)
		sub.GET("/products/:id", s.storeProductHTML)
		sub.POST("/cart/add", s.storeCartAdd)
		sub.POST("/cart/remove", s.storeCartRemove)
		sub.GET("/cart", s.storeCartHTML)
		sub.POST("/checkout", middleware.OptionalAuth(s.pool, s.cfg.ClerkAuthorizedParties), s.storeCheckout)
	}
}

func (s *Server) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) home(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>EcomHub</title><link rel="stylesheet" href="/static/style.css"></head><body>`+
		`<header class="site-header"><a class="brand" href="/">EcomHub</a><nav>`+
		`<a href="/products">Products</a> <a href="/stores">Stores</a> <a href="/search">Search</a> <a href="/dashboard">Dashboard</a>`+
		`</nav></header><main class="container"><h1>EcomHub</h1><p class="muted">Multi-tenant storefronts and a lightweight marketplace hub.</p></main></body></html>`)
}

func normalizeSubdomain(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func staticSubFS() fs.FS {
	sub, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
