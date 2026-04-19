# EcomHub — build cheatsheets

Quick reference: evolve a **working monolith** into a **multi-tenant hub** without premature complexity.

---

## 1) Evolution at a glance

| Phase | You add | Stack shape |
|-------|---------|-------------|
| **1** MVP | Auth, stores, products, simple storefront, orders | `User → Go monolith → PostgreSQL` |
| **2** Multi-tenant | `store_id` everywhere, resolve store from host/path, scoped queries | Same + **strict isolation** |
| **3** Hub | Global `/products`, `/stores`, `/search` | Tenant DB + **aggregated / indexed global view** |
| **4** Scale | Indexes, Redis, CDN | `… → PG (+ replicas later) + Redis + CDN` |
| **5–8** Later | Split services (if needed), analytics DB, search engine, horizontal scale | `CDN → LB → stateless services → DBs + cache + search + events` |

**Golden path:** validate → isolate tenants → add hub read models → measure → cache/CDN → search/events → split only when justified.

---

## 2) Stage cheatsheet

### Stage 1 — Monolith MVP (single server)

| Item | Choice |
|------|--------|
| **Goal** | Ship fast; learn domain |
| **Backend** | One Go service |
| **DB** | PostgreSQL |
| **Frontend** | Basic (SSR or SPA) |
| **Features** | Auth, store creation, product CRUD, storefront (subdomain or path), simple orders |
| **Host** | Single VPS / Render / Fly / similar |
| **Why** | Idea validation; avoid premature architecture |

### Stage 2 — Multi-tenant design

| Item | Rule |
|------|------|
| **Goal** | Many users, one platform, safe isolation |
| **Schema** | `store_id` on all tenant-owned rows |
| **Queries** | Always `WHERE store_id = ?` (plus authz) |
| **Middleware** | `Request → resolve store → attach context` |
| **Resolve store** | Subdomain `store1.app.com` **or** path `/store/store1` |
| **Mental model** | **Isolation first** — no cross-store leaks in tenant APIs |

```text
resolveStore(host or path) → store_id
SELECT … FROM products WHERE store_id = ? AND …
```

### Stage 3 — Hub (global layer)

| Item | Detail |
|------|--------|
| **Goal** | Marketplace-style discovery |
| **Routes (examples)** | `/products`, `/stores`, `/search?q=` |
| **Data** | **Tenant tables** (source of truth) + **global index / projections** (hub reads) |
| **Design** | Do not mash hub SQL into every store query; keep hub reads explicit |

### Stage 4 — Performance scaling

| Area | Action |
|------|--------|
| **PostgreSQL** | Indexes on `store_id`, `created_at`, FKs; explain slow queries |
| **Replicas** | Read replicas when read load dominates (later) |
| **Redis** | Sessions, hot keys, storefront fragments / rate limits |
| **CDN** | Static assets + product images |

### Stage 5 — Microservices (optional)

| Trigger | When a boundary hurts (team scale, deploy risk, different SLO) |
|---------|------------------------------------------------------------------|
| **Splits (examples)** | Auth, Store, Product, Order, Analytics |
| **Sync** | REST or gRPC |
| **Async** | Queue / log (Kafka-like) for notifications, indexing, analytics |

### Stage 6 — Analytics & data scaling

| Piece | Role |
|-------|------|
| **Events** | `order_placed`, `product_viewed`, … |
| **Pipeline** | Queue → batch → warehouse / OLAP |
| **Store** | Separate analytics DB (not OLTP) |

### Stage 7 — Search infrastructure

| When | Hub / marketplace search outgrows SQL `ILIKE` |
|------|-----------------------------------------------|
| **Options** | Elasticsearch, Meilisearch, Typesense |
| **Pattern** | OLTP → indexer (event or cron) → search index |

### Stage 8 — Horizontal scaling

| Layer | Pattern |
|-------|---------|
| **App** | Stateless Go instances behind load balancer |
| **Session** | Redis or short-lived tokens (today: Supabase access token in HttpOnly cookie, re-verified per request) |
| **DB** | Replicas first; sharding only with clear pain |

---

## 3) Architecture string summary

```text
Start:     User → Go app → PostgreSQL
Then:      User → LB → N× Go → PostgreSQL + Redis
Mature:    User → CDN → LB → services → PostgreSQL + Redis + Search + analytics store
```

---

## 4) Scaling principles (do)

1. **Stateless services** — state in PostgreSQL / Redis / object storage, not local memory.
2. **Multi-tenancy early** — every tenant table row tied to `store_id`; APIs enforce scope.
3. **Index for real queries** — composite indexes matching `WHERE store_id … ORDER BY created_at`.
4. **Separate concerns gradually** — monolith modules before microservices.
5. **Optimize after measuring** — traces, slow query logs, load tests.

---

## 5) Common mistakes (don’t)

| Mistake | Why it hurts |
|---------|----------------|
| Microservices on day one | Slow delivery, distributed debugging |
| Ignoring tenant isolation | Data leaks, legal/reputational risk |
| Missing `store_id` indexes | Table scans at scale |
| Hub logic mixed into every store query | Coupling, bugs, slow evolution |
| Overengineering before MVP | No users, complex system |

---

## 6) PRD — MVP one-pager

| Area | In scope |
|------|----------|
| **Product** | Multi-tenant stores + hosted storefront + hub-lite discovery |
| **Auth** | Supabase Auth (dashboard JS); `POST /dashboard/session` → HttpOnly cookie; `GET /api/me`; middleware verifies Supabase JWT + `user_identities` |
| **Stores** | CRUD metadata; subdomain/slug; user owns 1+ stores |
| **Products** | CRUD; name, description, price, stock, images |
| **Storefront** | Public home + product detail (subdomain) |
| **Commerce** | Cart, checkout, order create; **mock payment OK** |
| **Hub** | Global product list, store list, basic search |
| **NFRs** | HTTPS; validation; typical API under 300 ms; stateless; backups |

**Out of scope (MVP):** theme marketplace, heavy analytics UI, commission engine, native apps, reco engine, deep shipping integrations.

---

## 7) Data model quick reference

| Entity | Key fields |
|--------|------------|
| **User** | `id`, `email`, `password_hash` (nullable for Supabase-only), `created_at` |
| **UserIdentity** | `user_id`, `provider` (`supabase`), `provider_subject` (JWT `sub`), `provider_email`, unique `(provider, provider_subject)` |
| **Store** | `id`, `user_id`, `name`, `subdomain`, `description`, `status` (`active` \| `suspended` \| `deleted`), `created_at` |
| **Product** | `id`, `store_id`, `name`, `description`, `price`, `stock`, `image_url`, `created_at` |
| **Order** | `id`, `store_id`, `total_price`, `status`, `created_at` |
| **OrderItem** | `id`, `order_id`, `product_id`, `quantity`, `price` |

**Rule:** any row that “belongs to a store” includes `store_id` and is queried with it.

---

## 8) API surface (examples)

| Domain | Examples |
|--------|----------|
| **Auth** | `POST /dashboard/session` (JSON `access_token`, optional `next`); `POST /api/logout`; `GET /api/me` (cookie or `Authorization: Bearer` Supabase access token) |
| **Stores** | `POST /api/stores`, `GET /api/stores` |
| **Products** | `POST /api/products`, `GET /api/products`, `PUT /api/products/{id}`, `DELETE /api/products/{id}` |
| **Orders** | `POST /api/orders`, `GET /api/orders` |
| **Public** | `GET /s/{subdomain}`, `GET /s/{subdomain}/products`, … |
| **Hub** | `GET /products`, `GET /stores`, `GET /search?q=` |

*(Adjust paths to match your router; keep **tenant** vs **hub** routes mentally separate.)*

---

## 9) Implementation milestones (checklist)

- [x] **Foundation:** Go app, Postgres migrations (`001` init + `002` identities / `stores.status`), Supabase JWT auth + JIT `user_identities`
- [x] **Store + products:** CRUD + `store_id` scoping + owner checks; partial product `PUT` uses single atomic `UPDATE`
- [ ] **Storefront:** host-first subdomain middleware (optional `BASE_HOST`); `/s/{subdomain}` fallback exists
- [x] **Commerce:** cookie cart → `placeOrder` (transaction + `FOR UPDATE` stock); HTML + API paths
- [x] **Hub:** global listings + `ILIKE` search (`/products`, `/stores`, `/search`)

**Notes:** Migrations re-run on each boot (keep SQL idempotent). Cart cookie is unsigned JSON; **checkout always re-validates** store, lines, and stock, so tampering cannot bypass server rules.

**Optional later (not blocking MVP):**

| Item | When it matters |
|------|-------------------|
| **`schema_migrations` ledger** | When you add migrations that are **not** safe to re-`EXEC` on every boot. |
| **Cart HMAC / signed payload** | Extra tamper-evidence; **not** a substitute for checkout validation (already required). |
| **`pg_trgm` or FTS** | When hub `ILIKE '%q%'` gets **slow** or tables are **large**; until then Postgres is fine. |

Cart HTML, `resolveCartLines`, and `placeOrder` already use **batched** `SELECT … WHERE id = ANY($ids)` (plus `FOR UPDATE` in `placeOrder`) — **no N+1** on those paths.

---

## 10) Next artifacts (pick one)

| Deliverable | Use when |
|-------------|----------|
| **Go project layout** | Standardizing packages: `cmd/`, `internal/{httpserver,auth,db,tenant,hub}` |
| **SQL schema** | Indexes + FKs + hub projection tables or materialized views |
| **Deployment** | `Dockerfile`, `docker-compose`, env matrix, reverse proxy + TLS + wildcard DNS |
| **Full PRD** | Stakeholder sign-off; expand acceptance criteria per feature |

---

*Principle to remember:* **correct tenancy + schema beats premature infra.** Scaling gets easier when every request has an explicit `store_id` and hub reads are a deliberate layer.
