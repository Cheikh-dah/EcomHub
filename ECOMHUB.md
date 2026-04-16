# EcomHub — project reference

Single source of truth for MVP product, stack, routing, data model, roadmap, and checklist.

---

## 1. Overview

EcomHub helps sellers create a store quickly, share a public storefront link, and get light marketplace discovery. **MVP core loop:** create account → create store → add products → share storefront → receive basic orders.

**One-line summary:** Launch a simple online store and get discovered through a shared hub.

---

## 2. MVP stack

| Layer | Choice |
|--------|--------|
| Backend | Go + Gin (modular monolith) |
| Database | PostgreSQL (`pgxpool`) |
| Auth | Supabase Auth + internal `users` / `user_identities` mapping |
| UI (MVP) | Server-rendered HTML from the Go app |
| Local dev | Docker Compose (Postgres) + `go run ./cmd/server` from `ecomhub/` |
| Deploy | Backend: Render / Fly / Railway; frontend later: Vercel |
| Ops | `/health`, HTTPS, secure cookies |

---

## 3. Product

### Problem

Small sellers lack easy storefronts and built-in discovery; marketplaces vs DIY tools each solve only half.

### Goals (MVP)

- Seller register / login (managed auth)
- Create and manage stores
- CRUD products
- Public storefront + cart + basic order capture
- Hub: list products/stores + simple search

### Non-goals (MVP)

- Payment gateway, advanced ranking, native app, microservices

### Users

- **Sellers:** small businesses, students, social sellers  
- **Buyers:** browsers of public stores and hub search

### Scope by area

**Auth:** Supabase Auth (sessions, email verification); backend resolves provider identity → internal `userID`; owner routes protected.

**Stores:** name, description, unique **subdomain** (public URL); reserved names blocked at create/update.

**Products:** name, description, price, stock, image_url.

**Public storefront:** see §4 Storefront routing.

**Discovery:** public products/stores lists + basic search.

### User flows

- **Seller:** login (apex) → dashboard → create store → add products → share `https://{subdomain}.ecomhub.com` → review orders  
- **Buyer:** discover → open storefront → browse → cart → submit order

### Success metrics

Registered sellers, stores, products, orders, discovery traffic.

### Risks (short)

Cold start (seed content), spam (validation + moderation backlog), scope creep (enforce non-goals), reliability (health, logs, backups).

### Post-MVP

Payments, reviews, better search, seller analytics, image CDN/storage.

---

## 4. Storefront routing (MVP)

**Decision:** Option 1 — **dual routing, host-first** (storefront only on subdomains; dashboard + hub on apex).

| Environment | Buyer URL | Seller / hub |
|---------------|-----------|----------------|
| Production | `https://{subdomain}.ecomhub.com` | `https://ecomhub.com` |
| Dev | `http://{subdomain}.localhost:{port}` with `BASE_HOST=localhost` | same host apex paths |

**Fallback:** `/s/{subdomain}` (and nested paths) when `BASE_HOST` is empty or for tests.

**Rules**

- Parse `Host` without port; lowercase; **single-level** label only (reject extra dots in the tenant label).
- Strict match against `BASE_HOST` (e.g. `ecomhub.com`); avoid naive suffix bugs on lookalike domains.
- **Reserved** (never tenant): `www`, `api`, `admin`, `app`, `dashboard`, `mail`, `support` (extend as needed).
- **Store `status`:** only `active` serves public storefront; unknown or non-active → branded **Store not found** (optional subdomain in copy + CTAs: browse stores, create store).
- **Cache:** subdomain → store in-memory + TTL; invalidate on store writes; Redis later.
- **Logs:** host → subdomain → store id or not-found for debugging.

**Infra:** DNS wildcard `*.ecomhub.com` → app; TLS for apex + wildcard.

---

## 5. Data model (MVP)

**users:** id, email, created_at  

**user_identities:** id, user_id, provider, provider_subject, provider_email, created_at  

**stores:** id, user_id, name, subdomain (unique), description, **status** (`active` \| `suspended` \| `deleted`), created_at  

**products:** id, store_id, name, description, price, stock, image_url, created_at  

**orders / order_items:** as implemented for MVP checkout.

---

## 6. API surface (target)

**Auth:** Supabase-backed routes; middleware maps provider → internal `userID`.

**Stores:** `GET/POST /api/stores`, `PUT /api/stores/:id`  

**Products:** `GET/POST /api/products`, `PUT/DELETE /api/products/:id`  

**Cart / orders:** `GET /api/cart`, `POST /api/cart/add|remove|clear`, `POST /api/orders`, `GET /api/orders`

**Hub HTML:** `/products`, `/stores`, `/search?q=...`

---

## 7. Environment variables

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | Postgres connection |
| `SUPABASE_URL` | Supabase project URL |
| `SUPABASE_ANON_KEY` | Client-safe key where applicable |
| `SUPABASE_SERVICE_KEY` | Server only |
| `PORT` | HTTP port |
| `ENVIRONMENT` | e.g. development / production |
| `BASE_HOST` | Production: `ecomhub.com`; dev: `localhost`; empty → path-only `/s/{subdomain}` |

---

## 8. Roadmap (phased)

**Phase 0 — Setup:** Deployable backend; Supabase Auth + `users` / `user_identities`; env vars set; Postgres + app running; dashboard + store pages reachable.

**Phase 1 — Core backend:** Schema as §5; APIs as §6; auth middleware; storefront routing §4; seller can register, store, products, test order.

**Phase 2 — UI:** Auth screens (Supabase); dashboard (store CRUD, products, orders); public storefront on subdomain + fallback paths.

**Phase 3 — Hub:** Global discovery routes live.

**Phase 4 — Launch:** DNS/TLS, CORS/cookies, logging, `/health`, DB backups; public beta.

---

## 9. MVP acceptance checklist

- [ ] Register / login with managed auth  
- [ ] Create at least one store  
- [ ] Add / edit / delete products  
- [ ] Open storefront via `{store}.ecomhub.com` or `/s/{store}` and view products  
- [ ] Unknown or inactive subdomain → branded Store not found  
- [ ] Cart + basic checkout  
- [ ] Hub pages `/products`, `/stores`, `/search` functional  

---

## 10. Current status (repo)

- Scaffold under `ecomhub/` (Go + Gin + Postgres); routes and HTML in progress.  
- Auth direction: Supabase + internal mapping.  
- Next: implement managed auth, `user_identities`, store **status**, host-first routing, and schema alignment in code.
