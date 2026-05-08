# EcomHub - project reference

Single source of truth for the current MVP direction, architecture, data model, roadmap, and guardrails.

---

## 1. Overview

EcomHub is a multi-tenant commerce platform where merchants create online storefronts and customers discover stores/products through a shared ecosystem.

Current MVP loop:

```text
Create account -> create store -> add products -> share storefront -> receive basic orders
```

One-line summary:

```text
Launch a simple online store and get discovered through a shared hub.
```

---

## 2. Current Stack

| Layer | Current choice |
|-------|----------------|
| Backend | Go + Gin modular monolith |
| Database | PostgreSQL via pgxpool |
| Auth | Clerk session JWTs + Go verification + internal users/user_identities mapping |
| UI now | Server-rendered Go HTML templates + CSS |
| Frontend later | Next.js on Vercel, starting with storefront pages |
| Deploy now | Go app on Render/Fly/Railway-style service |
| Ops | `/health`, HTTPS in production, secure HttpOnly auth cookie |

Architecture direction:

```text
Now:
Browser -> Go SSR/API -> PostgreSQL

Later:
Next.js/Vercel frontend -> Go API -> PostgreSQL
```

The current Go SSR frontend should remain working while the backend becomes API-ready for a gradual Next.js migration.

---

## 3. Product Scope

### MVP Goals

- Merchant login with Clerk.
- Create/manage stores.
- Add/delete products.
- Public storefront pages.
- Product detail pages.
- Hub pages for products/stores/search.
- Basic cart/order flow.
- Lightweight theme customization: colors, logo URL, layout preset, rounding.

### Current Commerce Positioning

EcomHub is moving toward a lightweight merchant storefront model. For the near-term MVP, avoid building a heavy checkout/media/editor system too early.

Recommended near-term ordering flow:

```text
Customer visits store
-> sees products
-> opens product
-> orders through the merchant workflow
```

Do not add advanced payment, crop tools, upload pipelines, or media libraries until the core storefront/dashboard flows are stable.

### Non-goals For Current MVP

- Full payment gateway.
- Advanced analytics.
- Native app.
- Microservices.
- Drag-and-drop storefront builder.
- Image crop/focal-point editor.
- Upload/CDN/image transformation pipeline.
- Full Next.js rewrite in one pass.

---

## 4. Routing

### Current

The current storefront route is path-based:

```text
/s/{subdomain}
/s/{subdomain}/products/{id}
/s/{subdomain}/cart
```

Hub/dashboard routes live on the same Go app:

```text
/
/products
/stores
/search
/dashboard
```

### Future

Subdomain storefronts are a strong reason to introduce Next.js on Vercel later:

```text
{store}.ecomhub.com -> Next.js storefront route
ecomhub.com         -> hub/dashboard routes
api.ecomhub.com     -> Go API, or same backend origin behind rewrites
```

Migration should be gradual:

1. Keep Go SSR working.
2. Add/standardize JSON APIs for public storefront data.
3. Build Next.js storefront first.
4. Move dashboard/theme editor later.
5. Retire Go templates only when replacement routes are stable.

---

## 5. Data Model

Key tables:

- `users`: internal user profile.
- `user_identities`: maps Clerk user subject to internal `users.id`.
- `stores`: merchant-owned stores with `name`, `subdomain`, `description`, `status`, `theme_config`.
- `products`: store-owned products with `image_url`.
- `orders` / `order_items`: current basic order tables.

Important relationships:

- `products.store_id` references `stores(id)` with `ON DELETE CASCADE`.
- `orders.store_id` references `stores(id)` with `ON DELETE RESTRICT`.
- Store deletion must always be owner-scoped:

```sql
DELETE FROM stores
WHERE id = $1 AND user_id = $2
```

---

## 6. API Surface

Current protected API examples:

```text
GET    /api/me
POST   /api/logout
GET    /api/stores
POST   /api/stores
PUT    /api/stores/:id
GET    /api/stores/:id/theme
PUT    /api/stores/:id/theme
GET    /api/products?store_id=<id>
POST   /api/products
PUT    /api/products/:id
DELETE /api/products/:id
GET    /api/cart
POST   /api/cart/add
POST   /api/cart/remove
POST   /api/cart/clear
POST   /api/orders
GET    /api/orders
```

Future public API readiness for Next.js storefront:

```text
GET /api/public/stores/:subdomain
GET /api/public/stores/:subdomain/products
GET /api/public/stores/:subdomain/products/:id
GET /api/public/stores/:subdomain/theme
```

Those endpoints should return stable JSON shapes that a future Next.js storefront can consume without depending on Go templates.

---

## 7. Environment Variables

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string |
| `CLERK_SECRET_KEY` | Clerk backend secret key |
| `CLERK_PUBLISHABLE_KEY` | Clerk browser publishable key |
| `CLERK_FRONTEND_API` | Optional Clerk frontend origin override |
| `CLERK_AUTHORIZED_PARTIES` | Optional exact origins for Clerk JWT `azp` validation |
| `APP_URL` | Public app origin; used for auth origin defaults |
| `PORT` | HTTP port |
| `ENVIRONMENT` | `development`, `staging`, or `production` |

---

## 8. UI Architecture Decisions

### Component-minded SSR

The current frontend is Go SSR, but class naming should stay component-minded so it can migrate cleanly to Next.js later.

Current semantic contracts:

```text
.product-media              -> future <ProductImage />
.product-media-img          -> internal image element
.product-media-img--contain -> fit="contain"
.product-card               -> future <ProductCard />
.hub-card                   -> future <HubCard />
.store-card                 -> future <StoreCard />
.site-nav                   -> future navigation primitive
.store-logo                 -> future <StoreLogo />
```

### Product Media Policy

Do now:

- Product cards use `cover`.
- Product detail uses `contain`.
- Wrapper controls aspect ratio/layout.
- Image controls rendering fit.
- Merchant-owned media stays as remote URLs.

Do later:

- Upload-to-storage.
- CDN/image optimization.
- Crop/focal-point tooling.

### Store Logo Policy

Do now:

- Store logo is `logo_url` in `theme_config`.
- Theme Editor accepts a remote HTTP(S) logo URL.
- Storefront header renders `.store-logo` when present.
- Fallback to store name when missing.

Do not do now:

- Upload UI.
- Crop editor.
- Media library.
- Banner system.

### Mobile Navigation Policy

Current simple stacked navigation is enough for MVP.

A hamburger/drawer should be a later reusable interface system, not a one-off toggle. Build it when navigation grows enough to justify:

- overlay state
- focus management
- body scroll locking
- Escape key behavior
- z-index coordination
- accessibility labels/states

---

## 9. Security Notes

Current:

- Clerk session JWTs are verified by Go.
- Backend middleware is the security authority.
- Dashboard HTML routes use auth middleware.
- Store/product writes perform owner checks.
- Store delete uses POST and owner-scoped SQL.
- Auth cookie uses `SameSite=Lax`.

Known follow-up:

- Add CSRF tokens for destructive dashboard forms before serious production usage.

---

## 10. Roadmap

### Phase 1 - Stabilize Current MVP

- Keep Go SSR working.
- Polish storefront/product/dashboard UX.
- Harden auth/session and destructive actions.
- Keep media and navigation semantics migration-ready.

### Phase 2 - API Readiness

- Audit current JSON endpoints.
- Add missing public storefront JSON endpoints.
- Standardize response shapes.
- Document auth requirements and error behavior.

### Phase 3 - Next.js Storefront

- Deploy Next.js on Vercel.
- Use subdomain routing for stores.
- Consume Go public APIs.
- Keep Go as backend authority.

### Phase 4 - Dashboard Migration

- Move merchant dashboard/theme editor only after storefront is stable.
- Preserve Go API and ownership checks.

### Phase 5 - Retire Go SSR Views

- Remove old templates only after Next.js replacements are live and verified.

---

## 11. Current Status Snapshot

- Clerk auth bridge is active.
- Dashboard/store/product SSR flows exist.
- Hub pages exist.
- Path-based storefront exists.
- Theme config includes colors, logo URL, layout preset, rounding, and surface colors.
- Product media semantics are in place.
- Storefront logo URL rendering is polished.
- Mobile nav foundation uses semantic `.site-nav` classes, without drawer infrastructure.

---

## 12. Principle

```text
Stabilize the product core first.
Define clean semantic boundaries now.
Move to Next.js gradually when the API boundary is ready.
```
