# EcomHub Cheatsheets

Quick reference for the current MVP and the path toward a future separated frontend.

## Current Stack

```text
Browser -> Go SSR/API -> PostgreSQL
                 |
                 +-> Clerk identity/session JWTs
```

| Area | Current choice |
| --- | --- |
| Backend | Go + Gin |
| Frontend | Go SSR templates + scoped CSS |
| Database | PostgreSQL |
| Auth | Clerk session JWTs verified by Go |
| Deployment | Render is the current simplest path |
| Future frontend | Next.js on Vercel after API boundaries are ready |

## Product Model

EcomHub is a platform where merchants create their own storefronts while customers discover stores and products through a shared hub.

Current customer purchase direction:

```text
Customer visits store
-> sees products
-> opens product
-> orders through merchant-managed channel
```

Do not expand checkout/cart behavior until product direction is intentionally revisited.

## Core Data Model

| Entity | Key fields |
| --- | --- |
| `users` | internal user profile |
| `user_identities` | maps Clerk `sub` to internal `users.id` |
| `stores` | `user_id`, `name`, `subdomain`, `description`, `status`, `theme_config` |
| `products` | `store_id`, `name`, `description`, `price`, `stock`, `image_url` |
| `orders` | `store_id`, total/status fields |

Rules:

- Tenant-owned rows must be scoped through `store_id` or `user_id`.
- Store ownership checks must use `WHERE id = ? AND user_id = ?`.
- Product deletion and updates must verify ownership through the product's store.

## Route Map

Current public SSR routes:

```text
GET /
GET /products
GET /stores
GET /search?q=<term>
GET /s/:subdomain
GET /s/:subdomain/products/:id
GET /s/:subdomain/cart
```

Current dashboard SSR routes:

```text
GET  /dashboard
GET  /dashboard/stores
POST /dashboard/stores
GET  /dashboard/stores/:id/theme
GET  /dashboard/products
POST /dashboard/products
```

Current API references live in [REST-API-REFERENCE.md](./REST-API-REFERENCE.md).

## Auth Cheatsheet

See [AUTH-BRIDGE.md](./AUTH-BRIDGE.md) for full details.

Short version:

```text
Clerk browser session -> Clerk JWT -> POST /dashboard/session
Go verifies JWT -> maps Clerk subject to users.id
Go sets HttpOnly auth_token cookie
Dashboard/API requests use cookie or Bearer token
```

Go remains the authorization authority. Frontend checks are UX only.

## UI Architecture

Use semantic classes that map cleanly to future React components.

| Current class | Future component idea |
| --- | --- |
| `.product-media` | `<ProductImage />` wrapper |
| `.product-media-img` | internal image element |
| `.product-media-img--contain` | `fit="contain"` |
| `.store-logo` | `<StoreLogo />` |
| `.site-nav` | `<SiteNav />` |
| `.dashboard-body .card` | dashboard-scoped card |
| `.hub-card` | hub-scoped card |

Guardrails:

- Do not make `.card` global until dashboard, hub, and storefront areas are reviewed.
- Do not rely on global input styles for color/range/radio controls.
- Do not let dashboard polish leak into storefront themes.
- Keep storefront theme variables scoped.

## Media Policy

Now:

- Product images are remote merchant URLs.
- Store logos are remote merchant URLs.
- Use semantic media wrappers.
- Use `cover` for cards and `contain` for detail when edges/details matter.

Later:

- Upload pipeline.
- CDN/image optimization.
- Focal point/crop metadata.
- Advanced media manager.

Do not import merchant product/logo images as frontend assets. Imported images are for static platform-owned assets only.

## Theme Policy

Theme config is stored in `stores.theme_config`.

Current high-value fields:

- `primary_color`
- `accent_color`
- `page_bg`
- `text_color`
- `card_bg`
- `footer_bg`
- `logo_url`
- `layout_preset`
- `rounding`

Rules:

- No arbitrary CSS.
- Validate colors and URLs server-side.
- Storefront pages read theme through scoped CSS variables.
- Theme editor preview should mirror real storefront behavior.

See [THEME-CUSTOMIZATION.md](./THEME-CUSTOMIZATION.md).

## Future Next.js/Vercel Direction

Do this gradually:

1. Keep Go SSR stable.
2. Add public JSON endpoints for storefront data.
3. Build Next.js storefront first.
4. Use Vercel for frontend/subdomain routing.
5. Move dashboard later only if needed.

Recommended future public endpoints:

```text
GET /api/public/stores/:subdomain
GET /api/public/stores/:subdomain/products
GET /api/public/stores/:subdomain/products/:id
GET /api/public/hub/products
GET /api/public/hub/stores
GET /api/public/search?q=<term>
```

## Priority Order

Now:

1. Stabilize storefront/product UX.
2. Keep dashboard store/product management reliable.
3. Harden auth/session behavior.
4. Clean API boundaries.
5. Improve visual consistency with scoped styles.

Later:

1. CSRF tokens for destructive dashboard forms.
2. Public API endpoints for Next.js.
3. Host/subdomain routing.
4. Store logo upload and media infrastructure.
5. Next.js storefront migration.
6. Dashboard migration if SSR becomes limiting.

## Non-Goals For MVP

- Full page builder.
- Arbitrary merchant CSS.
- Crop tool.
- Upload/CDN pipeline.
- Microservices.
- Full frontend rewrite before API contracts exist.

## Principle

Ship the reliable core first. Keep class names, API shapes, and theme semantics clean enough that the future Next.js frontend can reuse the same product language.
