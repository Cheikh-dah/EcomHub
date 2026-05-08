# Shippable Theme Plan

This is the practical theme plan for the current EcomHub MVP.

## Goal

Give merchants visible branding value without building a full design or media subsystem.

## Ship Now

### 1. Theme JSON

Persist and normalize:

```json
{
  "primary_color": "#111827",
  "accent_color": "#16a34a",
  "page_bg": "#ffffff",
  "text_color": "#111111",
  "card_bg": "#f9fafb",
  "footer_bg": "#ffffff",
  "logo_url": "",
  "layout_preset": "default",
  "rounding": 0.4,
  "preset": "minimal",
  "version": 1
}
```

### 2. Theme Editor

Dashboard editor controls:

- Primary color.
- Accent color.
- Page/card/footer colors.
- Logo URL.
- Layout preset.
- Rounding.

The preview should update instantly and save through the authenticated theme API.

### 3. Storefront Header Logo

If `logo_url` exists, show logo.

If missing, show store name.

No upload logic yet.

### 4. Product Media

Use semantic wrappers:

```text
Cards/detail wrapper: .product-media
Image: .product-media-img
Detail contain variant: .product-media-img--contain
```

Cards use `cover`; detail may use `contain`.

## Do Not Ship Yet

- Upload API.
- Crop UI.
- CDN transformation pipeline.
- Image library.
- Arbitrary merchant CSS.
- Theme marketplace.
- AI theme generation.

## Implementation Rules

- Backend validates all theme input.
- Storefront styling stays scoped to `.storefront`.
- Dashboard styling does not leak into storefront themes.
- Theme data must always normalize to a renderable default.
- No business logic should depend on visual theme fields.

## Future Next.js Mapping

The current classes should become simple components later:

```tsx
<StoreLogo src={theme.logoUrl} fallback={store.name} />
<ProductImage src={product.imageUrl} alt={product.name} fit="cover" />
```

This keeps the Go SSR version useful today and migration-ready later.

## Done When

- A merchant can paste a logo URL and see it in the storefront header.
- Product cards and product detail images render consistently.
- Theme editor preview and real storefront match.
- Invalid theme data cannot break storefront rendering.
- Existing stores without theme data still render cleanly.

## Principle

Ship small, make the store feel real, and keep the media pipeline simple until demand proves it needs to grow.
