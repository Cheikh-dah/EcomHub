package httpserver

import (
	"context"
	"html/template"
	"io/fs"
	"log"
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
		api.GET("/public/stores/:subdomain", s.apiPublicStoreBySubdomain)
		api.GET("/public/stores/:subdomain/products", s.apiPublicStoreProducts)

		authd := api.Group("")
		authd.Use(middleware.RequireAuth(s.pool, s.cfg.ClerkAuthorizedParties))
		{
			authd.GET("/me", s.apiMe)
			authd.GET("/stores", s.apiListStores)
			authd.POST("/stores", s.apiCreateStore)
			authd.PUT("/stores/:id", s.apiUpdateStore)
			authd.GET("/stores/:id/theme", s.apiGetStoreTheme)
			authd.PUT("/stores/:id/theme", s.apiUpdateStoreTheme)

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
	dashAuth.Use(middleware.RequireAuthRedirect(s.pool, s.cfg.ClerkAuthorizedParties))
	dashAuth.GET("/stores", s.dashboardStoresGet)
	dashAuth.POST("/stores", s.dashboardCreateStore)
	dashAuth.POST("/stores/:id/delete", s.dashboardStoreDelete)
	dashAuth.GET("/stores/:id/theme", s.dashboardStoreThemeGet)
	dashAuth.GET("/products", s.dashboardProductsGet)
	dashAuth.POST("/products", s.dashboardProductsPost)
	dashAuth.POST("/products/:id/delete", s.dashboardProductDelete)

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
	err := s.tmpl.ExecuteTemplate(c.Writer, "home", gin.H{})
	if err != nil {
		log.Printf("home render error: %v", err)
	}
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
