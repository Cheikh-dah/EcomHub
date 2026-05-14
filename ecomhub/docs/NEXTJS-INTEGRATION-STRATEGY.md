# Next.js Integration Strategy

This document describes how EcomHub can move from the current Go SSR MVP toward a separated Next.js frontend without forcing a rewrite before the product surface is stable.

## Current Go SSR State

EcomHub currently runs as a Go/Gin application that owns:

- public hub pages
- public storefront pages
- merchant dashboard pages
- JSON API routes
- Clerk-backed auth/session handling
- PostgreSQL reads and writes
- storefront theme rendering

This is the correct MVP shape. It keeps product behavior, database access, auth, and templates in one deployable system while the marketplace model is still being stabilized.

## Target Split

The future split should be:

- Next.js owns public frontend rendering and frontend routing.
- Go/Gin remains the source of truth for API, auth verification, business logic, and database access.
- PostgreSQL remains the source of truth for stores, products, themes, users, and future order data.

Next.js should not become a second backend. It should consume stable public and authenticated Go APIs.

## Same-Origin Rewrite Strategy

The first Next.js integration should use same-origin rewrites instead of browser cross-origin requests.

In production, the browser should call:

```text
/api/public/...
```

Next.js/Vercel can rewrite those requests to the Go backend on Render.

Benefits:

- browser requests remain same-origin
- no public CORS policy is needed for the first migration phase
- frontend code can use relative API paths
- the Go backend remains hidden behind a stable API path
- the strategy works for root-domain hub pages and store subdomains

The exact Vercel rewrite configuration should be added when the Next.js app exists.

## Public API Routes Available Now

The current public API foundation is:

```text
GET /api/public/stores/:subdomain
GET /api/public/stores/:subdomain/products?limit=24&offset=0
GET /api/public/stores/:subdomain/products/:id
GET /api/public/hub/products?limit=24&offset=0&search=
GET /api/public/hub/stores?limit=24&offset=0&search=
```

These routes are the initial contract for the future Next.js public UI.

The public API intentionally:

- returns DTOs instead of raw database models
- exposes only active stores
- omits private merchant fields such as `user_id`
- keeps product `store_id` out of public product DTOs
- includes store attribution on hub product responses
- normalizes nullable public text fields before JSON serialization
- uses bounded pagination

## Subdomain Routing Plan

The target production routing model is:

```text
ecomhub.com                 -> public hub
www.ecomhub.com             -> public hub
{store}.ecomhub.com         -> public storefront
```

For store subdomains, Next.js should:

1. Read the request host.
2. Extract the subdomain.
3. Render the storefront route.
4. Fetch store data from:

```text
GET /api/public/stores/:subdomain
GET /api/public/stores/:subdomain/products
```

Product detail pages should fetch:

```text
GET /api/public/stores/:subdomain/products/:id
```

The Go backend should continue to enforce active-store visibility. Next.js routing must not be treated as a security boundary.

## Localhost And www Behavior

Local development should avoid pretending that every environment supports wildcard subdomains.

Recommended local behavior:

```text
localhost:3000              -> hub
localhost:3000/s/:subdomain -> storefront fallback during development
```

Production behavior can use wildcard subdomains once Vercel/domain configuration is in place.

`www` should be treated as the public hub, not as a store subdomain.

Reserved host labels should not resolve as stores:

```text
www
api
admin
dashboard
app
```

This reserved-name policy should be enforced by store creation validation before depending on it in Next.js.

## CORS Decision

CORS is deferred deliberately.

The first separated frontend should use same-origin rewrites. That keeps the browser-facing contract simple and avoids opening a broad cross-origin API surface before there is a real external-client requirement.

Add an explicit CORS policy only when one of these becomes necessary:

- a separate domain must call the Go API directly from the browser
- third-party clients need browser access
- mobile or partner apps need a documented public API origin policy

When CORS is added, it should be narrow and environment-specific.

## Migration Phases

### Phase 1: Keep Go SSR Stable

Continue shipping the MVP through Go SSR while public API contracts mature.

### Phase 2: Prototype Next.js Storefront

Build a small Next.js storefront prototype against:

```text
/api/public/stores/:subdomain
/api/public/stores/:subdomain/products
/api/public/stores/:subdomain/products/:id
```

Keep the Go storefront available during this phase.

### Phase 3: Move Public Hub Pages

Move hub product/store discovery after storefront reads are proven.

Use:

```text
/api/public/hub/products
/api/public/hub/stores
```

### Phase 4: Move Merchant Dashboard Later

Dashboard migration should wait until public pages are stable because dashboard work involves authenticated APIs, Clerk session behavior, forms, destructive actions, and merchant workflows.

### Phase 5: Retire Duplicated SSR Surfaces Carefully

Only remove Go SSR templates after the equivalent Next.js pages have:

- matching behavior
- passing tests
- working production routing
- clear rollback path

## Rollback Triggers

Pause or roll back a Next.js migration phase if:

- storefront pages cannot resolve subdomains reliably
- same-origin API rewrites are unstable
- public API responses need breaking changes
- auth/session handling becomes unclear
- page load performance regresses noticeably
- merchant dashboard workflows become blocked
- production debugging becomes harder than the Go SSR baseline

Rollback should mean routing traffic back to Go SSR while keeping the Go API intact.

## Non-Goals For Now

Do not include these in the first Next.js migration:

- full frontend rewrite
- dashboard rewrite
- auth/session redesign
- cart or checkout redesign
- WhatsApp ordering
- media upload pipeline
- CDN/image transformation system
- crop/reposition tooling
- broad CORS policy
- public API versioning

These are separate product and architecture decisions.

## Next Safety Slice

Before building the Next.js storefront prototype, harden merchant media URL policy.

The public API currently allows merchant-owned remote image URLs. That is the right MVP direction, but the platform should add validation before the frontend depends on these values heavily.

Recommended next safety work:

- reject oversized `data:` image URLs
- define allowed URL schemes
- trim and length-limit image URLs
- preserve remote merchant image URLs for now
- defer upload/CDN/media-library work
