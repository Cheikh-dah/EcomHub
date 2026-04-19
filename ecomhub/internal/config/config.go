package config

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Environment is one of: development, staging, production (lowercase after load).
	Environment string
	HTTPPort    string
	DatabaseURL string
	// AppURL is the public base URL of this app (e.g. https://ecomhub.com). Optional until redirects or auth callbacks need it.
	AppURL string
	// BaseHost is the apex domain for subdomain resolution (e.g. "ecomhub.local" or "example.com").
	// Empty disables host-based tenant resolution; use /s/{subdomain} paths only.
	BaseHost string

	// SupabaseURL is the project URL (e.g. https://xxxx.supabase.co).
	SupabaseURL string
	// SupabaseJWTSecret is the JWT signing secret from Supabase Dashboard → Settings → API.
	// Used to verify Supabase-issued user access tokens (not the service_role key).
	SupabaseJWTSecret string
	// SupabaseAnonKey is the public anon key (browser-safe) for the dashboard Supabase client.
	SupabaseAnonKey string
	// SupabaseServiceKey is optional; server-only when used.
	SupabaseServiceKey string
}

func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		Environment:        strings.ToLower(getenv("ENVIRONMENT", "development")),
		HTTPPort:           getenv("PORT", "8080"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AppURL:             strings.TrimSpace(os.Getenv("APP_URL")),
		BaseHost:           strings.TrimSpace(os.Getenv("BASE_HOST")),
		SupabaseURL:        strings.TrimSpace(os.Getenv("SUPABASE_URL")),
		SupabaseJWTSecret:  strings.TrimSpace(os.Getenv("SUPABASE_JWT_SECRET")),
		SupabaseAnonKey:    strings.TrimSpace(os.Getenv("SUPABASE_ANON_KEY")),
		SupabaseServiceKey: strings.TrimSpace(os.Getenv("SUPABASE_SERVICE_KEY")),
	}

	validate(cfg)
	return cfg
}

func validate(cfg Config) {
	switch cfg.Environment {
	case "development", "staging", "production":
	default:
		log.Fatalf("ENVIRONMENT must be development, staging, or production (got %q)", cfg.Environment)
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required (e.g. postgres://ecomhub:ecomhub@localhost:5432/ecomhub?sslmode=disable)")
	}
	if cfg.AppURL != "" {
		appu, err := url.Parse(cfg.AppURL)
		if err != nil || appu.Scheme == "" || appu.Host == "" {
			log.Fatalf("APP_URL must be a valid URL with host when set: %q", cfg.AppURL)
		}
		if appu.Scheme != "http" && appu.Scheme != "https" {
			log.Fatalf("APP_URL scheme must be http or https, got %q", appu.Scheme)
		}
		if cfg.Environment == "production" && appu.Scheme != "https" {
			log.Fatal("APP_URL must use https in production")
		}
	}
	if cfg.SupabaseURL == "" {
		log.Fatal("SUPABASE_URL is required (Supabase project URL, e.g. https://xxxx.supabase.co)")
	}
	u, err := url.Parse(cfg.SupabaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Fatalf("SUPABASE_URL must be a valid URL with host: %q", cfg.SupabaseURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		log.Fatalf("SUPABASE_URL scheme must be http or https, got %q", u.Scheme)
	}
	if (cfg.Environment == "production" || cfg.Environment == "staging") && u.Scheme != "https" {
		log.Fatal("SUPABASE_URL must use https when ENVIRONMENT is staging or production")
	}
	if cfg.SupabaseJWTSecret == "" || len(cfg.SupabaseJWTSecret) < 16 {
		log.Fatal("SUPABASE_JWT_SECRET is required and must be at least 16 characters (JWT Secret from Supabase API settings — not the service_role key)")
	}
	if cfg.SupabaseAnonKey == "" || len(cfg.SupabaseAnonKey) < 20 {
		log.Fatal("SUPABASE_ANON_KEY is required and must be at least 20 characters (public anon key for the dashboard Supabase client)")
	}
	if cfg.SupabaseServiceKey != "" && len(cfg.SupabaseServiceKey) < 20 {
		log.Fatal("SUPABASE_SERVICE_KEY must be at least 20 characters when set")
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}
