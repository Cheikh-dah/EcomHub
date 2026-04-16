package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment string
	HTTPPort    string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	// BaseHost is the apex domain for subdomain resolution (e.g. "ecomhub.local" or "example.com").
	// Empty disables host-based tenant resolution; use /s/{subdomain} paths only.
	BaseHost string
}

func Load() Config {
	_ = godotenv.Load()

	jwtExp := getenv("JWT_EXPIRY_HOURS", "72")
	hours, err := time.ParseDuration(jwtExp + "h")
	if err != nil {
		hours = 72 * time.Hour
	}

	cfg := Config{
		Environment: strings.ToLower(getenv("ENVIRONMENT", "development")),
		HTTPPort:    getenv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTExpiry:   hours,
		BaseHost:    strings.TrimSpace(os.Getenv("BASE_HOST")),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" || len(cfg.JWTSecret) < 16 {
		log.Fatal("JWT_SECRET is required and must be at least 16 characters")
	}
	return cfg
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
