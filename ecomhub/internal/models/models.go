package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Store struct {
	ID          int64     `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Subdomain   string    `json:"subdomain"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type StoreTheme struct {
	// Legacy fields (Maintained for backward compatibility)
	PrimaryColor string `json:"primary_color"`
	AccentColor  string `json:"accent_color"`
	LogoURL      string `json:"logo_url,omitempty"`
	LayoutPreset string `json:"layout_preset"` // legacy: default | compact

	// DNA v2 (Phase 1)
	Preset   string  `json:"preset"`   // minimal | luxury | playful
	Rounding float64 `json:"rounding"` // 0.0 -> 1.0
	Version  int     `json:"version"`

	// Phase 2: Design Tokens
	PageBg    string `json:"page_bg"`
	TextColor string `json:"text_color"`
	CardBg    string `json:"card_bg"`
	FooterBg  string `json:"footer_bg"`

	// Phase 2+ placeholders (commented out or empty for now to avoid confusion)
	// SpacingScale string `json:"spacing_scale,omitempty"`
}

func (t *StoreTheme) Normalize() {
	// 1. Enforce allowed Presets
	if t.Preset != "minimal" && t.Preset != "luxury" && t.Preset != "playful" {
		t.Preset = "minimal"
	}

	// 2. Clamp Rounding [0, 1]
	if t.Rounding < 0 {
		t.Rounding = 0
	} else if t.Rounding > 1 {
		t.Rounding = 1
	}

	// 3. Set defaults for design tokens
	if t.PageBg == "" {
		t.PageBg = "#ffffff"
	}
	if t.TextColor == "" {
		t.TextColor = "#111111"
	}
	if t.CardBg == "" {
		t.CardBg = "#ffffff"
	}
	if t.FooterBg == "" {
		t.FooterBg = "transparent"
	}

	// 4. Migration & Versioning
	if t.Version == 0 {
		t.ApplyLegacy()
		t.Version = 1
	}
}

// ApplyLegacy maps old fields to the new DNA system where applicable.
func (t *StoreTheme) ApplyLegacy() {
	// Example: If a legacy store has a specific color, we might want to preserve it
	// or map layout_preset to a future spacing_scale.
}

// UserIdentity maps an external auth provider subject to an internal user.
type UserIdentity struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	Provider        string    `json:"provider"`
	ProviderSubject string    `json:"provider_subject"`
	ProviderEmail   string    `json:"provider_email,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Product struct {
	ID          int64     `json:"id"`
	StoreID     int64     `json:"store_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Order struct {
	ID         int64       `json:"id"`
	StoreID    int64       `json:"store_id"`
	UserID     uuid.UUID   `json:"user_id"`
	TotalPrice float64     `json:"total_price"`
	Status     string      `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	Items      []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"order_id"`
	ProductID int64   `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type CartPayload struct {
	StoreID int64      `json:"store_id"`
	Lines   []CartLine `json:"lines"`
}

type CartLine struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}
