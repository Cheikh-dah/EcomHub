# 🚀 EcomHub MVP Roadmap

## 🧭 Goal
Build and launch the smallest working version of EcomHub where users can:
- Create a store
- Add products
- Share a public store link
- Let others view the store

---

## ✅ Current status snapshot
- Backend monolith scaffold exists in `ecomhub/` (Go + Gin + PostgreSQL).
- Core domain structure is defined for users, stores, products, orders, and order_items.
- API plus HTML storefront/dashboard routes are scaffolded and ready for MVP hardening.
- Auth direction is locked for MVP: Supabase Auth + internal user mapping.

---

# 🗓️ Phase 0 — Setup (Day 1–2)

## ✅ Tasks
- Keep backend (Go API) deployable to Render (or similar)
- Use managed auth for MVP: Supabase Auth
- Keep app ownership/authorization in internal DB via `users` + `user_identities`
- Set environment variables:
  - `DATABASE_URL`
  - `SUPABASE_URL`
  - `SUPABASE_ANON_KEY`
  - `SUPABASE_SERVICE_KEY` (server only)
  - `PORT`
  - `ENVIRONMENT`
  - Optional: `BASE_HOST`

## Deliverable
- App runs locally end to end:
  - Postgres up
  - Go backend running
  - Dashboard and public store pages accessible

---

# 🧱 Phase 1 — Core Backend (Day 3–5)

## 📦 Database Schema (MVP)

### users
- id
- email
- created_at

### user_identities
- id
- user_id
- provider
- provider_subject
- provider_email
- created_at

### stores
- id
- user_id
- name
- subdomain
- description
- created_at

### products
- id
- store_id
- name
- description
- price
- stock
- image_url
- created_at

## 🔌 API Endpoints (Go)

### Auth
- Provider-auth routes (Supabase-based)
- Internal auth middleware maps provider identity -> internal `userID`

### Stores
- `GET /api/stores` → List my stores
- `POST /api/stores` → Create store
- `PUT /api/stores/:id` → Update store

### Products
- `GET /api/products` → List products in my store scope
- `POST /api/products` → Add product
- `PUT /api/products/:id` → Update product
- `DELETE /api/products/:id` → Delete product

### Cart / Orders
- `GET /api/cart`
- `POST /api/cart/add`
- `POST /api/cart/remove`
- `POST /api/cart/clear`
- `POST /api/orders`
- `GET /api/orders`

## 🔐 Authentication
- Supabase Auth for sign-in/session flows
- Backend middleware verifies auth token and resolves internal user identity
- Route-level protection for owner dashboard and API endpoints

## Deliverable
- Seller can register, create store, create products, and place a test order.

---

# 🎨 Phase 2 — Frontend Core (Day 6–9)

## 📄 Pages

### 1. Authentication Page
- Sign up / login forms (Supabase Auth)
- Session persistence via provider SDK/token flow

### 2. Dashboard
- Create and edit store
- Add/edit/delete products
- View basic order list

### 3. Public Store Page
- Route: `/s/{subdomain}`
- Product detail route: `/s/{subdomain}/products/{id}`
- Cart page: `/s/{subdomain}/cart`

## Deliverable
- Public can browse store catalog and create a checkout flow with minimal friction.

---

# 🧩 Phase 3 — Hub Layer (Day 10–12)

## Add global discovery
- `GET /products` (all public products)
- `GET /stores` (all public stores)
- `GET /search?q=...` (basic SQL-backed search)

## Deliverable
- Users can discover products and stores outside a single tenant storefront.

---

# 🚢 Phase 4 — Deployment and launch (Day 13–14)

## Infrastructure
- Backend deploy (Render/Fly/Railway/VPS)
- PostgreSQL managed instance
- Frontend deploy (Vercel)
- DNS and domain setup

## Reliability and security checklist
- HTTPS enabled
- CORS and secure cookie settings
- Error logging and health checks (`/health`)
- Basic backup strategy for database

## Deliverable
- Public beta launch with core MVP flow fully usable.

---

# 🧪 MVP acceptance checklist
- [ ] User can register/login with managed auth
- [ ] User can create at least one store
- [ ] User can add/edit/delete products
- [ ] Public can open store URL and view products
- [ ] Public can add items to cart and checkout (basic flow)
- [ ] Hub pages (`/products`, `/stores`, `/search`) are functional

---

# 📌 Post-MVP (next)
- Redis for caching/session improvements
- Object storage for product images
- Advanced search (Meilisearch/Typesense/Elasticsearch)
- Analytics event pipeline
- Service decomposition only if operationally required