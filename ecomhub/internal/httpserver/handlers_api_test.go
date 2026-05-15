package httpserver

import (
	"errors"
	"strings"
	"testing"
)

func strPtr(v string) *string {
	return &v
}

func floatPtr(v float64) *float64 {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func TestNormalizeProductUpdateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		body productUpdateBody
	}{
		{
			name: "blank name",
			body: productUpdateBody{Name: strPtr("   ")},
		},
		{
			name: "negative price",
			body: productUpdateBody{Price: floatPtr(-0.01)},
		},
		{
			name: "negative stock",
			body: productUpdateBody{Stock: intPtr(-1)},
		},
		{
			name: "invalid image url",
			body: productUpdateBody{ImageURL: strPtr("data:image/png;base64,abc")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeProductUpdate(tt.body); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeProductCreateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		product  string
		price    float64
		stock    int
		imageURL string
	}{
		{name: "blank name", product: "   ", price: 1, stock: 1},
		{name: "negative price", product: "Jacket", price: -0.01, stock: 1},
		{name: "negative stock", product: "Jacket", price: 1, stock: -1},
		{name: "invalid image url", product: "Jacket", price: 1, stock: 1, imageURL: "data:image/png;base64,abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeProductCreate(tt.product, "desc", tt.price, tt.stock, tt.imageURL); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeProductCreateTrimsValidValues(t *testing.T) {
	product, err := normalizeProductCreate("  Jacket  ", "  Warm  ", 12.5, 3, "  https://example.com/jacket.jpg  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if product.Name != "Jacket" {
		t.Fatalf("name not normalized: %#v", product)
	}
	if product.Description != "Warm" {
		t.Fatalf("description not normalized: %#v", product)
	}
	if product.Price != 12.5 {
		t.Fatalf("price not normalized: %#v", product)
	}
	if product.Stock != 3 {
		t.Fatalf("stock not normalized: %#v", product)
	}
	if product.ImageURL != "https://example.com/jacket.jpg" {
		t.Fatalf("image URL not normalized: %#v", product)
	}
}

func TestNormalizeProductImageURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "whitespace", in: "   ", want: ""},
		{name: "http", in: "http://example.com/image.jpg", want: "http://example.com/image.jpg"},
		{name: "https", in: "https://example.com/image.jpg", want: "https://example.com/image.jpg"},
		{name: "trims", in: "  https://example.com/image.jpg  ", want: "https://example.com/image.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeProductImageURL(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestNormalizeProductImageURLRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "data image", in: "data:image/png;base64,abc"},
		{name: "data", in: "data:text/plain,abc"},
		{name: "javascript", in: "javascript:alert(1)"},
		{name: "file", in: "file:///C:/tmp/image.jpg"},
		{name: "blob", in: "blob:https://example.com/id"},
		{name: "ftp", in: "ftp://example.com/image.jpg"},
		{name: "relative", in: "/image.jpg"},
		{name: "hostless", in: "https:///image.jpg"},
		{name: "too long", in: "https://example.com/" + strings.Repeat("a", maxProductImageURLLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeProductImageURL(tt.in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeProductUpdateTrimsValidValues(t *testing.T) {
	update, err := normalizeProductUpdate(productUpdateBody{
		Name:        strPtr("  Jacket  "),
		Description: strPtr("  Warm  "),
		Price:       floatPtr(12.5),
		Stock:       intPtr(3),
		ImageURL:    strPtr("  https://example.com/jacket.jpg  "),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !update.SetName || update.Name != "Jacket" {
		t.Fatalf("name not normalized: %#v", update)
	}
	if !update.SetDesc || update.Desc != "Warm" {
		t.Fatalf("description not normalized: %#v", update)
	}
	if !update.SetPrice || update.Price != 12.5 {
		t.Fatalf("price not normalized: %#v", update)
	}
	if !update.SetStock || update.Stock != 3 {
		t.Fatalf("stock not normalized: %#v", update)
	}
	if !update.SetImage || update.ImageURL != "https://example.com/jacket.jpg" {
		t.Fatalf("image URL not normalized: %#v", update)
	}
}

func TestNormalizeStoreNameRejectsBlankName(t *testing.T) {
	if _, err := normalizeStoreName("   "); err == nil {
		t.Fatal("expected blank store name to fail")
	}
}

func TestNormalizeStoreNameTrimsName(t *testing.T) {
	name, err := normalizeStoreName("  My Store  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "My Store" {
		t.Fatalf("expected trimmed store name, got %q", name)
	}
}

func TestNormalizeStoreSubdomainNormalizesValidSubdomain(t *testing.T) {
	sub, err := normalizeStoreSubdomain("  My-Store  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub != "my-store" {
		t.Fatalf("expected normalized subdomain, got %q", sub)
	}
}

func TestNormalizeStoreSubdomainRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty", in: ""},
		{name: "starts with hyphen", in: "-store"},
		{name: "ends with hyphen", in: "store-"},
		{name: "underscore", in: "my_store"},
		{name: "too long", in: strings.Repeat("a", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeStoreSubdomain(tt.in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeStoreSubdomainRejectsReservedValues(t *testing.T) {
	for _, in := range []string{"www", "api", "admin", "dashboard", "app", "  API  "} {
		t.Run(in, func(t *testing.T) {
			if _, err := normalizeStoreSubdomain(in); !errors.Is(err, errReservedSubdomain) {
				t.Fatalf("expected reserved subdomain error, got %v", err)
			}
		})
	}
}

func TestNormalizeStoreThemePatchRejectsInvalidSurfaceColors(t *testing.T) {
	curr := defaultStoreTheme()
	tests := []struct {
		name  string
		patch storeThemeUpdateBody
	}{
		{name: "invalid page_bg", patch: storeThemeUpdateBody{PageBg: strPtr("white")}},
		{name: "invalid text_color", patch: storeThemeUpdateBody{TextColor: strPtr("#12345g")}},
		{name: "invalid card_bg", patch: storeThemeUpdateBody{CardBg: strPtr("rgb(0,0,0)")}},
		{name: "invalid footer_bg", patch: storeThemeUpdateBody{FooterBg: strPtr("none")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeStoreThemePatch(curr, tt.patch); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeStoreThemePatchNormalizesSurfaceColors(t *testing.T) {
	curr := defaultStoreTheme()
	theme, err := normalizeStoreThemePatch(curr, storeThemeUpdateBody{
		PrimaryColor: strPtr("  #ABCDEF  "),
		AccentColor:  strPtr("#123456"),
		PageBg:       strPtr("#FAFAFA"),
		TextColor:    strPtr("#0F172A"),
		CardBg:       strPtr("#F8FAFC"),
		FooterBg:     strPtr(" TRANSPARENT "),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if theme.PrimaryColor != "#abcdef" {
		t.Fatalf("primary color not normalized: %q", theme.PrimaryColor)
	}
	if theme.AccentColor != "#123456" {
		t.Fatalf("accent color not normalized: %q", theme.AccentColor)
	}
	if theme.PageBg != "#fafafa" {
		t.Fatalf("page bg not normalized: %q", theme.PageBg)
	}
	if theme.TextColor != "#0f172a" {
		t.Fatalf("text color not normalized: %q", theme.TextColor)
	}
	if theme.CardBg != "#f8fafc" {
		t.Fatalf("card bg not normalized: %q", theme.CardBg)
	}
	if theme.FooterBg != "transparent" {
		t.Fatalf("footer bg not normalized: %q", theme.FooterBg)
	}
}
