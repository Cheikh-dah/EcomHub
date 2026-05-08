# Store Design Strategy

EcomHub storefront design should feel merchant-owned without asking merchants to become designers.

## Strategy

```text
Brand intent -> safe theme JSON -> scoped storefront UI
```

The platform should offer controlled choices that produce professional storefronts by default.

## Current MVP Direction

Build:

- Theme tokens.
- Store logo URL.
- Product media semantics.
- Scoped storefront CSS.
- Live theme preview.

Avoid for now:

- Full crop tools.
- Upload/storage pipelines.
- Arbitrary CSS.
- Drag-and-drop page builder.
- AI-generated CSS.
- Banner/gallery systems.

## Theme DNA

The long-term theme model can grow into:

```go
type StoreTheme struct {
    Preset       string
    Rounding     float64
    PrimaryColor string
    AccentColor  string
    LogoURL      string
    LayoutPreset string
    Version      int
}
```

Later fields can include font pair, spacing scale, surface style, and button style, but they should stay bounded enums.

## Design Guardrails

- Curated options, not infinite controls.
- Server-side validation for every theme field.
- CSS variables and classes, not raw CSS input.
- Storefront styles scoped under `.storefront`.
- Defaults must always render a complete store.
- Existing stores must remain valid after theme changes.

## Media Semantics

Product images and store logos are merchant-owned remote URLs.

Current rendering contract:

```html
<div class="product-media">
  <img class="product-media-img" src="{{.ImageURL}}" alt="{{.Name}}">
</div>
```

Product detail can opt into contain:

```html
<img class="product-media-img product-media-img--contain" src="{{.ImageURL}}" alt="{{.Name}}">
```

Store logos use:

```html
<img class="store-logo" src="{{.Theme.LogoURL}}" alt="{{.Store.Name}}">
```

These semantics are intentionally framework-neutral and map cleanly to future Next.js components.

## Priority

Now:

1. Stable storefront/product layout.
2. Responsive behavior.
3. Logo URL support.
4. Theme preview accuracy.
5. Security and ownership checks.

Later:

1. Logo upload.
2. Product image upload.
3. CDN/image optimization.
4. Focal point/crop metadata.
5. Theme gallery and presets.
6. AI-assisted theme selection.

## Measurement

The theme system should improve:

- Time to first live store.
- Merchant confidence after previewing their storefront.
- Storefront polish without manual support.
- Migration readiness for a future Next.js storefront.

## Principle

Do not let merchants design from scratch. Give them a few strong levers and make the system protect the final result.
