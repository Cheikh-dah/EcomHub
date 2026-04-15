# Product Requirements Document (PRD)

## Product Name
EcomHub

## 1. Overview
EcomHub is an e-commerce platform that helps sellers create a store quickly and share a public storefront link, while also supporting a lightweight discovery layer for buyers.

For MVP, the focus is the core loop:
Create account -> create store -> add products -> share store -> receive basic orders.

## 2. Problem Statement
Small businesses and new sellers often struggle to launch an online store because setup is technical and discovery is hard. Existing options usually optimize either:
- Store ownership/customization, or
- Marketplace discovery.

EcomHub aims to offer both in a simple, fast-to-launch experience.

## 3. Goals and Non-Goals
### Goals (MVP)
- Let a seller register and log in.
- Let a seller create and manage one or more stores.
- Let a seller create, update, and delete products.
- Provide a public store page where buyers can browse products.
- Support a basic cart and order creation flow.
- Expose simple discovery endpoints for stores/products/search.

### Non-Goals (MVP)
- Payment gateway integration.
- Advanced ranking/recommendation algorithms.
- Mobile app.
- Multi-service decomposition or microservices.

## 4. Target Users
### Primary Users (Sellers)
- Small business owners
- Student entrepreneurs
- Social media sellers moving to a storefront model

### Secondary Users (Buyers)
- Visitors browsing public stores
- Users searching for products/stores across the hub

## 5. MVP Scope
### 5.1 Authentication
- Register, login, logout via managed auth provider
- Supabase Auth for session and email verification flows
- Protected owner routes using internal user identity mapping

### 5.2 Store Management
- Create store
- Update store profile (name, description, subdomain/slug as implemented)
- List current user's stores

### 5.3 Product Management
- Add, edit, delete products
- Product fields:
  - name
  - description
  - price
  - stock
  - image_url

### 5.4 Public Storefront
- Public route to view store catalog
- Product detail view (if implemented in current routes)
- Basic cart operations and checkout/order creation

### 5.5 Discovery (Hub Layer)
- List public products
- List public stores
- Basic text search endpoint

## 6. User Flows
### Seller Flow
1. Register/login
2. Create store
3. Add products
4. Share store link
5. Review incoming orders

### Buyer Flow
1. Discover store/product
2. Open public storefront
3. Browse products
4. Add to cart
5. Submit order

## 7. Functional Requirements
- API endpoints for auth, stores, products, cart, and orders are available and documented.
- Owner-only actions require authentication/authorization.
- Public storefront routes are accessible without authentication.
- Orders are persisted with enough data to review basic purchase activity.

## 8. Technical Constraints (MVP)
- Backend: Go monolith (Gin + internal modules)
- Database: PostgreSQL
- Auth: Supabase Auth + internal `users`/`user_identities` mapping
- Deployment target: backend on Render/Fly/Railway/VPS, frontend on Vercel (when frontend is wired)
- Health check endpoint available (`/health`)

## 9. Success Metrics
- Number of registered sellers
- Number of stores created
- Number of products listed
- Number of orders created
- Discovery usage (search/list endpoint traffic)

## 10. Risks and Mitigations
- Cold start discovery risk: seed with demo stores/products.
- Abuse/spam risk: add minimal validation and moderation backlog.
- Scope creep risk: enforce MVP non-goals until core loop is stable.
- Reliability risk: keep health checks, logs, and backup basics in place.

## 11. Post-MVP Backlog
- Payments integration
- Reviews/ratings
- Better search relevance
- Seller analytics
- Image storage optimization

## One-Line Summary
EcomHub helps anyone launch a simple online store and get discovered through a shared hub.
