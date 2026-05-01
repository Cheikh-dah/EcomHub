# Store Theme Customization

## Overview

EcomHub allows merchants to customize the appearance of their storefronts through a live theme editor on the merchant dashboard. Phase 1 focuses on core customization: primary color, accent color, logo, and layout presets.

---

## Phase 1: MVP Features

### What merchants can customize

- **Primary Color**: Main brand color (CTA buttons, links, accents) — hex value
- **Accent Color**: Secondary highlight color (alerts, badges, decorative elements) — hex value
- **Logo**: Store logo displayed in storefront header — HTTPS image URL
- **Layout Preset**: `default` (full product cards) or `compact` (condensed cards)

### User Experience

1. Merchant navigates to **Dashboard** → click "theme editor" link next to store name
2. Theme Editor page loads with:
   - **Controls panel** (left): Color pickers (HTML5 `<input type="color">`) + hex input fields, logo URL input, layout toggle
   - **Preview panel** (right): Real storefront iframe with live CSS variable injection, plus a local preview surface for quick UI feedback
3. Merchant adjusts colors/logo/layout, sees preview update instantly
4. Clicks "Save theme" button
5. Changes persist to database and apply immediately on storefront

### Technical Implementation

#### Route & Handler

```
GET /dashboard/stores/:id/theme
```

Handled by `Server.dashboardStoreThemeGet()` in `handlers_html.go`:
- Parses store ID from URL parameter
- Validates ownership via `loadOwnedStore(ctx, userID, storeID)` (404 if not owner)
- Loads current theme via `loadStoreThemeByID(ctx, storeID)`
- Renders `theme_editor.html` template with store + theme data

#### API Endpoint

```
PUT /api/stores/:id/theme
```

Requires authenticated user. Handled by `Server.apiUpdateStoreTheme()` in `handlers_api.go`:

**Request body:**

```json
{
  "primary_color": "#1d9bf0",
  "accent_color": "#00ba7c",
  "page_bg": "#ffffff",
  "text_color": "#111111",
  "card_bg": "#f9fafb",
  "footer_bg": "#ffffff",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default",
  "rounding": 0.4
}
```

- All fields are optional (patch semantics)
- `primary_color`, `accent_color`, `page_bg`, `text_color`, `card_bg`, `footer_bg` must be valid hex (#RRGGBB) or omitted
- `logo_url` must be an absolute HTTP(S) URL or omitted
- `layout_preset` must be `default` or `compact` or omitted
- `rounding` must be a number between `0.0` and `1.0` or omitted

**Response (200 OK):**

```json
{
  "primary_color": "#1d9bf0",
  "accent_color": "#00ba7c",
  "page_bg": "#ffffff",
  "text_color": "#111111",
  "card_bg": "#f9fafb",
  "footer_bg": "#ffffff",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default",
  "rounding": 0.4,
  "preset": "minimal",
  "version": 1
}
```

#### Data Model

`StoreTheme` struct in `models.go`:

```go
type StoreTheme struct {
	PrimaryColor string  `json:"primary_color"`
	AccentColor  string  `json:"accent_color"`
	LogoURL      string  `json:"logo_url,omitempty"`
	LayoutPreset string  `json:"layout_preset"`
	Preset       string  `json:"preset"`
	Rounding     float64 `json:"rounding"`
	Version      int     `json:"version"`
	PageBg       string  `json:"page_bg"`
	TextColor    string  `json:"text_color"`
	CardBg       string  `json:"card_bg"`
	FooterBg     string  `json:"footer_bg"`
}
```

Stored as JSONB in `stores.theme_config` column.

#### Database Schema

```sql
ALTER TABLE stores ADD COLUMN theme_config JSONB NOT NULL DEFAULT '{"primary_color":"#1d9bf0","accent_color":"#00ba7c","logo_url":"","layout_preset":"default"}';
```

(Already present in migrations.)

#### Storefront CSS Application

Store pages use a storefront-specific body class and inline CSS variables via template context:

```html
<body class="store-page storefront preset-{{.Theme.Preset}}" style="--page-bg: {{.Theme.PageBg}}; --text-color: {{.Theme.TextColor}}; --card-bg: {{.Theme.CardBg}}; --footer-bg: {{.Theme.FooterBg}}; --accent-color: {{.Theme.AccentColor}}; --store-primary: {{.Theme.PrimaryColor}}; --store-accent: {{.Theme.AccentColor}}; --store-bg: {{.Theme.PageBg}}; --store-text: {{.Theme.TextColor}}; --store-card: {{.Theme.CardBg}}; --store-footer: {{.Theme.FooterBg}}; --rounding: {{.Theme.Rounding}}">
```

This keeps storefront theme styles scoped to the `.storefront` body context and prevents dashboard header leakage.

#### Storefront header scoping

The storefront header is styled using store-scoped variables:

```css
body.storefront .site-header,
body.storefront .site-header.themed {
  background: var(--store-primary, var(--primary-color, #111111));
}
```

The theme editor's live preview injection writes both scoped and fallback variables into the iframe root:

```js
root.style.setProperty("--store-primary", primary);
root.style.setProperty("--primary-color", primary);
```

That makes the storefront header update instantly in the preview while keeping dashboard styles separate.

#### Dynamic Header Branding

The storefront header (`layout.html`) displays the store logo when available, otherwise it renders the store name as text. Theme colors are applied through scoped CSS variables rather than inline background styling.

This is implemented using Go template `{{with}}` blocks to safely handle missing data:
```html
{{with .Store}}
  {{with .Theme.LogoURL}}
    <img src="{{.}}" alt="{{$.Store.Name}}">
  {{else}}
    <span class="brand">{{.Name}}</span>
  {{end}}
{{else}}
  <a class="brand" href="/">EcomHub</a>
{{end}}
```

#### Helper Functions

**`loadOwnedStore(ctx, userID, storeID)`** — `handlers_html.go`
- Single query with ownership filter: `WHERE id = $1 AND user_id = $2`
- Returns `pgx.ErrNoRows` if store not found OR not owned (intentional — don't leak store existence)

**`loadStoreThemeByID(ctx, storeID)`** — `handlers_api.go`
- Fetches theme config from `stores.theme_config`
- Returns default theme if JSON parsing fails

**Validation functions** — `handlers_api.go`
- `normalizeColor(value, fallback)` — validates hex format, returns fallback if empty
- `normalizeLogoURL(value)` — validates absolute HTTPS/HTTP URLs
- `normalizeLayoutPreset(value)` — validates `default` or `compact`
- `normalizeStoreTheme(body)` — full validation pipeline for POST body
- `normalizeStoreThemePatch(current, patch)` — partial validation for PATCH updates

---

## Phase 2: Enhanced Customization (Roadmap)

- **Fonts**: Select from preset font families (sans/serif/mono) + scale presets
- **Advanced Layout**: Sidebar position, product grid columns, spacing/padding
- **Typography**: Adjust heading/body font sizes, weights, line-height
- **Sections**: Customize colors for specific sections (header, footer, cards)
- **Behavior-linked layout** (optional): Layout preset affects product query density (compact preset fetches minimal fields; default fetches rich data with ratings/descriptions)

#### Implementation notes
- Extend `StoreTheme` struct with new fields
- Add font selection UI to theme editor
- Update storefront CSS to read new variables
- Database migration: add new fields to `theme_config` JSON
- For behavior-linked layout: coordinate with product fetch layer in `handlers_html.go`

---

## Phase 3: Social & Gallery (Roadmap)

- **Theme Gallery**: Browse and apply pre-built theme templates by category
- **Snapshots**: Share theme snapshots (read-only preview link)
- **Recommendations**: Suggest themes based on store category
- **Community Themes**: One-click apply community-created themes

---

## Testing

### Manual Testing

1. **Happy path:**
   - Sign in to dashboard
   - Click "theme editor" on a store
   - Adjust color picker (verify preview updates in real time)
   - Adjust hex input manually (verify color picker syncs)
   - Enter logo URL (verify preview logo appears)
   - Toggle layout preset (verify preview card layout changes)
   - Click "Save theme"
   - Verify success message appears
   - Refresh page — verify theme persists
   - Navigate to store front — verify colors/logo/layout applied

2. **Edge cases:**
   - Invalid hex color (non-hex text in hex input) → "Save failed" message
   - Invalid logo URL (not http/https, not absolute) → "Save failed" message
   - Non-owner accessing other's theme editor → 404
   - Missing theme_config column → returns default theme, still saveable

### Unit Tests

- `normalizeColor()` validates hex, returns fallback if empty
- `normalizeLogoURL()` rejects relative URLs, non-http(s) schemes
- `normalizeLayoutPreset()` allows `default`/`compact` only
- `loadOwnedStore()` returns `ErrNoRows` for non-owner
- `apiUpdateStoreTheme()` returns 403 for non-owner, 200 for owner

### Integration Tests

- POST new store, GET theme editor — returns default theme
- PUT theme via API, GET theme editor — persists and loads correctly
- Multiple stores — themes don't cross-contaminate

---

## Invariants (Enforced)

These guarantees are built into the system:

1. **Renderability**: Every store always has a valid, renderable theme.
   - Missing `theme_config` → defaults applied
   - Invalid JSON → parsed with fallbacks
   - Storefront never 500s due to theme

2. **Ownership**: Only the store owner can view or modify theme.
   - Enforced at API boundary: `loadOwnedStore(ctx, userID, storeID)` gate
   - Non-owners receive 404 (not 403) — don't leak store existence
   - Theme is tamper-proof

3. **Consistency**: Theme changes apply atomically.
   - Single `UPDATE stores SET theme_config = $1` query
   - All pages see same theme within request
   - No partial updates visible to user

4. **Boundaries**: Theme affects presentation only within its store.
   - CSS variables scoped to store pages
   - No cross-store theme leakage
   - Multi-tenant safety guaranteed

---

## Security Considerations

- **Ownership validation**: `loadOwnedStore` ensures only store owner can view/edit theme
- **Color validation**: Prevents CSS injection via color input (hexadecimal only)
- **Logo URL validation**: Must be absolute HTTPS/HTTP URL; prevents javascript: URLs
- **Error messaging**: Non-owners receive 404 (not 403) to avoid leaking store existence
- **CORS**: Theme editor is dashboard-only (not cross-origin accessible)
- **Session Persistence**: The theme editor includes a background sync script (`clerk_sync`) to prevent logout while designing (see [AUTH-BRIDGE.md](AUTH-BRIDGE.md)).

---

## API Contracts

### GET /api/stores/:id/theme

**Authentication:** Required

**Response:** Returns current theme (200 OK) or 404 if store not found or not owner

```json
{
  "primary_color": "#1d9bf0",
  "accent_color": "#00ba7c",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default"
}
```

### PUT /api/stores/:id/theme

**Authentication:** Required

**Request body:** (all fields optional)

```json
{
  "primary_color": "#1d9bf0",
  "accent_color": "#00ba7c",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default"
}
```

**Responses:**

- `200 OK`: Theme updated, returns full theme object
- `400 Bad Request`: Invalid color/URL/layout format
- `403 Forbidden`: User does not own this store
- `404 Not Found`: Store not found
- `500 Internal Server Error`: Database error

---

## File Manifest

| File | Purpose |
|------|---------|
| `internal/web/templates/theme_editor.html` | Merchant theme editor page (controls + live preview) |
| `internal/httpserver/handlers_html.go` | `dashboardStoreThemeGet()` handler, `loadOwnedStore()` helper |
| `internal/httpserver/handlers_api.go` | `apiUpdateStoreTheme()`, `loadStoreThemeByID()`, validation fns |
| `internal/models/models.go` | `StoreTheme` struct |
| `internal/web/static/style.css` | `.theme-editor`, `.theme-preview`, `.theme-preview-card` styles |
| `internal/web/templates/dashboard.html` | "theme editor" link in store list |
| `internal/db/migrations/` | Migration adding `theme_config` JSONB column |

---

## Future Enhancements

- **Import/Export**: JSON export of theme for backups / sharing
- **Presets**: Save custom themes as named presets
- **A/B Testing**: Show different themes to different visitor cohorts, track conversion
- **Analytics**: See which colors/layouts drive more engagement
- **Figma Integration**: Design system sync with Figma
