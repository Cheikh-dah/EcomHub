# Store Theme Customization

EcomHub themes let merchants make their storefront feel owned without creating an unsafe custom-CSS system.

The current theme model is intentionally small:

- Brand colors.
- Page/card/footer colors.
- Logo URL.
- Layout preset.
- Rounding.

Merchant media remains URL-based for now. Do not add upload, crop, CDN, or image transformation logic during the MVP theme phase.

## Current Flow

```text
Dashboard -> Store card -> Edit Theme -> Theme editor -> Storefront preview
```

The theme editor is a dashboard page:

```text
GET /dashboard/stores/:id/theme
```

The authenticated API update endpoint is:

```text
PUT /api/stores/:id/theme
```

Both paths must enforce store ownership.

## Theme Shape

Theme config is stored in `stores.theme_config` as JSONB and normalized into `StoreTheme`.

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

## Validation Rules

- Colors must be `#RRGGBB` hex values.
- `logo_url` must be empty or an absolute HTTP(S) URL.
- `layout_preset` must be `default` or `compact`.
- `rounding` must stay in the supported numeric range.
- Invalid or missing theme JSON should fall back to a valid renderable default.

## Storefront Scoping

Storefront pages use a dedicated body scope:

```html
<body class="store-page storefront preset-{{.Theme.Preset}}" style="...">
```

Theme variables should stay scoped to storefront contexts so dashboard, hub, and public site styles do not leak into merchant themes.

Important variables:

```css
--store-primary
--store-accent
--store-bg
--store-text
--store-card
--store-footer
--rounding
```

## Logo Rendering

Logo support is URL-based.

Storefront header behavior:

- If `Theme.LogoURL` exists, render the logo image.
- If missing, fall back to the store name.

Current semantic contract:

```html
<a class="brand" href="/s/{{.Store.Subdomain}}" data-store-name="{{.Store.Name}}">
  {{if and .Theme .Theme.LogoURL}}
    <img class="store-logo" src="{{.Theme.LogoURL}}" alt="{{.Store.Name}}">
  {{else}}
    {{.Store.Name}}
  {{end}}
</a>
```

CSS:

```css
.store-logo {
  max-height: 44px;
  max-width: 180px;
  object-fit: contain;
  display: block;
}
```

This maps cleanly to a future Next.js component:

```tsx
<StoreLogo src={theme.logoUrl} fallback={store.name} />
```

## Product Media Contract

Product images use shared semantic classes instead of page-specific image hacks.

Card/list images:

```html
<div class="product-media">
  <img class="product-media-img" src="{{.ImageURL}}" alt="{{.Name}}">
</div>
```

Detail images:

```html
<div class="product-media product-media-detail">
  <img class="product-media-img product-media-img--contain" src="{{.ImageURL}}" alt="{{.Name}}">
</div>
```

Policy:

- Marketplace cards use `cover`.
- Storefront grids use `cover`.
- Product detail can use `contain` when edges/details matter.
- Wrapper controls layout/aspect ratio.
- Image controls rendering fit.

Future Next.js mapping:

```tsx
<ProductImage src={product.imageUrl} alt={product.name} fit="cover" />
```

## Theme Editor Preview

The theme editor should update the preview without requiring a full page reload.

Expected live behavior:

- Color controls update CSS variables.
- Logo URL updates the `.site-header .brand` content inside the preview iframe.
- Layout preset updates the visible product-card presentation.
- Save persists normalized JSON through `PUT /api/stores/:id/theme`.

## API Contract

### `GET /api/stores/:id/theme`

Auth: required.

Caller must own the store.

Returns the normalized theme.

### `PUT /api/stores/:id/theme`

Auth: required.

Caller must own the store.

Patch body:

```json
{
  "primary_color": "#111827",
  "accent_color": "#16a34a",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default"
}
```

Response: full normalized theme.

## Security Rules

- Ownership check must happen before theme reads or writes.
- Do not allow arbitrary CSS.
- Validate color values before writing CSS variables.
- Validate logo URLs to prevent `javascript:` and invalid schemes.
- Do not expose another merchant's private store/theme data through dashboard APIs.

## File Manifest

| File | Purpose |
| --- | --- |
| `internal/web/templates/theme_editor.html` | Merchant theme editor and preview scripting |
| `internal/web/templates/store_layout.html` | Storefront header/logo rendering and scoped theme variables |
| `internal/web/static/style.css` | Storefront, product media, logo, and theme editor styles |
| `internal/httpserver/handlers_html.go` | Dashboard theme editor handler and ownership helpers |
| `internal/httpserver/handlers_api.go` | Theme API handlers and normalization |
| `internal/models/models.go` | `StoreTheme` struct |
| `migrations/` | Schema including `stores.theme_config` |

## Roadmap

Now:

- Keep logo URL support.
- Keep product media semantics.
- Keep storefront theme isolation.
- Avoid crop/upload/CDN complexity.

Later:

- Store logo upload.
- Image storage and lifecycle.
- CDN/image optimization.
- Focal point or crop metadata.
- Theme gallery and reusable presets.
