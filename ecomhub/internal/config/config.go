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
	Environment string
	HTTPPort    string
	DatabaseURL string
	AppURL      string
	BaseHost    string

	// ClerkSecretKey is the Clerk secret key (sk_test_... / sk_live_...). Used for Backend API (JWKS, user lookup).
	ClerkSecretKey string
	// ClerkPublishableKey is the browser-safe key (pk_test_... / pk_live_...) for Clerk JS on the dashboard.
	ClerkPublishableKey string
	// ClerkFrontendAPI is the Clerk Frontend API origin, e.g. https://your-instance.clerk.accounts.dev
	// (Dashboard → API keys → Frontend API URL). Optional if the dashboard derives it from the publishable key.
	ClerkFrontendAPI string
	// ClerkAuthorizedParties lists allowed `azp` origins for session JWTs (comma-separated in env).
	// Empty disables azp checking (less strict; OK for some local setups).
	ClerkAuthorizedParties []string
}

func Load() Config {
	// Overload so values from .env replace existing process env (including empty strings).
	// godotenv.Load would skip any key already set in the environment — on Windows a mistaken
	// empty CLERK_SECRET_KEY in User/System env blocks the real key from .env.
	_ = godotenv.Overload()

	parties := parseCSV(os.Getenv("CLERK_AUTHORIZED_PARTIES"))
	cfg := Config{
		Environment:            strings.ToLower(getenv("ENVIRONMENT", "development")),
		HTTPPort:               getenv("PORT", "8080"),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AppURL:                 strings.TrimSpace(os.Getenv("APP_URL")),
		BaseHost:               strings.TrimSpace(os.Getenv("BASE_HOST")),
		ClerkSecretKey:         strings.TrimSpace(os.Getenv("CLERK_SECRET_KEY")),
		ClerkPublishableKey:    strings.TrimSpace(os.Getenv("CLERK_PUBLISHABLE_KEY")),
		ClerkFrontendAPI:       strings.TrimSpace(strings.TrimRight(os.Getenv("CLERK_FRONTEND_API"), "/")),
		ClerkAuthorizedParties: parties,
	}

	if len(cfg.ClerkAuthorizedParties) == 0 && cfg.AppURL != "" {
		cfg.ClerkAuthorizedParties = append(cfg.ClerkAuthorizedParties, strings.TrimRight(cfg.AppURL, "/"))
	}

	validate(cfg)
	return cfg
}

func parseCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
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
	if cfg.ClerkSecretKey == "" || len(cfg.ClerkSecretKey) < 20 {
		log.Fatal("CLERK_SECRET_KEY is required (Clerk Dashboard → API keys → Secret key)")
	}
	if !strings.HasPrefix(cfg.ClerkSecretKey, "sk_test_") && !strings.HasPrefix(cfg.ClerkSecretKey, "sk_live_") {
		log.Fatal("CLERK_SECRET_KEY must start with sk_test_ or sk_live_")
	}
	if cfg.ClerkPublishableKey == "" || len(cfg.ClerkPublishableKey) < 20 {
		log.Fatal("CLERK_PUBLISHABLE_KEY is required (Clerk Dashboard → API keys → Publishable key)")
	}
	if !strings.HasPrefix(cfg.ClerkPublishableKey, "pk_test_") && !strings.HasPrefix(cfg.ClerkPublishableKey, "pk_live_") {
		log.Fatal("CLERK_PUBLISHABLE_KEY must start with pk_test_ or pk_live_")
	}
	if cfg.ClerkFrontendAPI != "" {
		u, err := url.Parse(cfg.ClerkFrontendAPI)
		if err != nil || u.Scheme == "" || u.Host == "" {
			log.Fatalf("CLERK_FRONTEND_API must be a valid URL with host when set: %q", cfg.ClerkFrontendAPI)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			log.Fatalf("CLERK_FRONTEND_API scheme must be http or https, got %q", u.Scheme)
		}
		if (cfg.Environment == "production" || cfg.Environment == "staging") && u.Scheme != "https" {
			log.Fatal("CLERK_FRONTEND_API must use https when ENVIRONMENT is staging or production")
		}
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
