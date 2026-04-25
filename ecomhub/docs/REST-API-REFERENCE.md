# EcomHub REST API Reference

This document describes the current authenticated JSON API surface for EcomHub.

For auth/session internals, see [AUTH-BRIDGE.md](./AUTH-BRIDGE.md).
For product and architecture context, see [ECOMHUB-CHEATSHEETS.md](./ECOMHUB-CHEATSHEETS.md).

---

## Documentation Scope

- This file is the **source of truth** for REST request/response contracts and status codes.
- If an endpoint path, method, payload, auth requirement, or response changes, update this file in the same PR.
- Keep architecture decisions in `ECOMHUB-CHEATSHEETS.md` and auth internals in `AUTH-BRIDGE.md`.

---

## Base URL

- Local development: `http://localhost:8080`

---

## Auth Quickstart (Local)

1. Sign in from `http://localhost:8080/dashboard`.
2. Verify auth is working:
   - browser: `GET /api/me` with `credentials: "include"`, or
   - terminal: `GET /api/me` with `Authorization: Bearer <token>`.
3. Call protected endpoints only after `/api/me` returns `200`.

---

## Authentication

Protected endpoints accept either:

- `Authorization: Bearer <clerk_session_jwt>`
- `auth_token` HttpOnly cookie (set by `POST /dashboard/session`)

### Quick auth check

- `GET /api/me` returns the resolved internal user id for a valid token/cookie.

Example response:

```json
{
  "user_id": "a682e5da-7df4-474f-ac83-71cf374e13a9"
}
```

---

## Error Format

Errors currently return a simple JSON object:

```json
{
  "error": "message"
}
```

Common values:

- `missing or invalid authorization`
- `invalid token`
- `forbidden`
- `not found`
- `invalid body`

---

## Common Status Codes

- `200 OK`: successful read/update/delete.
- `201 Created`: resource created successfully.
- `400 Bad Request`: invalid query/body/id format.
- `401 Unauthorized`: missing or invalid token/cookie.
- `403 Forbidden`: authenticated user does not own target store/resource.
- `404 Not Found`: resource id does not exist.
- `409 Conflict`: unique constraint conflict (for example, duplicate subdomain).
- `500 Internal Server Error`: unexpected server/database error.

---

## Endpoints

### Auth

#### `POST /api/logout`

- Auth: none
- Behavior: clears `auth_token` cookie
- Response: `200 OK`

```json
{ "ok": true }
```

#### `GET /api/me`

- Auth: required
- Response: `200 OK`

```json
{ "user_id": "<uuid>" }
```

---

### Stores

#### `GET /api/stores`

- Auth: required
- Response: list of stores owned by the authenticated user

```json
[
  {
    "id": 11,
    "user_id": "a682e5da-7df4-474f-ac83-71cf374e13a9",
    "name": "cch",
    "subdomain": "cch",
    "description": "cccc",
    "status": "active",
    "created_at": "2026-04-25T09:00:00Z"
  }
]
```

#### `POST /api/stores`

- Auth: required
- Body:

```json
{
  "name": "My Store",
  "subdomain": "my-store",
  "description": "Optional"
}
```

- Validation:
  - `name` required
  - `subdomain` required and normalized/lowercased server-side
  - subdomain must be unique
- Response: `201 Created`

```json
{
  "id": 12,
  "subdomain": "my-store"
}
```

#### `PUT /api/stores/:id`

- Auth: required
- Body:

```json
{
  "name": "Updated Store Name",
  "subdomain": "updated-subdomain",
  "description": "Updated description"
}
```

- Response: `200 OK`

```json
{ "ok": true }
```

---

### Products

#### `GET /api/products?store_id=<id>`

- Auth: required
- Query:
  - `store_id` required, positive integer
- Authorization:
  - caller must own the target store
- Response: list of products for the store

```json
[
  {
    "id": 1,
    "store_id": 11,
    "name": "Black Hoodie",
    "description": "Premium cotton hoodie",
    "price": 49.99,
    "stock": 25,
    "image_url": "https://example.com/hoodie.jpg",
    "created_at": "2026-04-25T09:10:00Z"
  }
]
```

#### `POST /api/products`

- Auth: required
- Body:

```json
{
  "store_id": 11,
  "name": "New Product",
  "description": "Optional",
  "price": 12.5,
  "stock": 7,
  "image_url": ""
}
```

- Validation:
  - `store_id` required
  - `name` required
  - `price >= 0`
  - `stock >= 0`
- Authorization:
  - caller must own `store_id`
- Response: `201 Created`

```json
{ "id": 3 }
```

#### `PUT /api/products/:id`

- Auth: required
- Body: partial update; include only fields you want to change

```json
{
  "name": "New Name",
  "description": "Updated",
  "price": 15.99,
  "stock": 4,
  "image_url": ""
}
```

- Validation:
  - if provided, `price >= 0`
  - if provided, `stock >= 0`
- Authorization:
  - caller must own the store for the product id
- Response: `200 OK`

```json
{ "ok": true }
```

#### `DELETE /api/products/:id`

- Auth: required
- Authorization:
  - caller must own the store for the product id
- Response: `200 OK`

```json
{ "ok": true }
```

---

### Cart

#### `GET /api/cart`

- Auth: required
- Response:

```json
{
  "store_id": 11,
  "lines": [
    {
      "product_id": 1,
      "name": "Black Hoodie",
      "quantity": 2,
      "unit_price": 49.99,
      "line_total": 99.98
    }
  ],
  "total": 99.98
}
```

#### `POST /api/cart/add`

- Auth: required
- Body:

```json
{
  "product_id": 1,
  "quantity": 1
}
```

- Behavior:
  - enforces single-store cart
  - validates stock
- Response: `200 OK`

```json
{ "ok": true }
```

#### `POST /api/cart/remove`

- Auth: required
- Body:

```json
{
  "product_id": 1
}
```

- Response: `200 OK`

```json
{ "ok": true }
```

#### `POST /api/cart/clear`

- Auth: required
- Response: `200 OK`

```json
{ "ok": true }
```

---

### Orders

#### `POST /api/orders`

- Auth: required
- Body (optional store assertion):

```json
{
  "store_id": 11
}
```

- Behavior:
  - validates cart ownership and stock
  - places order transactionally
  - decrements stock
  - clears cart cookie on success
- Response: `201 Created`

```json
{
  "order_id": 42,
  "total": 99.98
}
```

#### `GET /api/orders`

- Auth: required
- Response: list of orders for authenticated user

---

## Public HTML Endpoints (non-JSON)

These are useful for manual product/store verification:

- `GET /products` (hub products page)
- `GET /stores` (hub stores page)
- `GET /search?q=<term>` (hub search page)
- `GET /s/:subdomain` (storefront home)
- `GET /s/:subdomain/products/:id` (storefront product detail)
- `GET /s/:subdomain/cart` (store cart page)

---

## Manual Smoke Test Checklist

1. `GET /health` returns `200`.
2. `GET /api/me` with auth returns `200`.
3. Create store (if needed): `POST /api/stores`.
4. Product CRUD:
   - `POST /api/products`
   - `PUT /api/products/:id`
   - `DELETE /api/products/:id`
5. Confirm product list: `GET /api/products?store_id=<id>`.
6. Cart/order flow:
   - `POST /api/cart/add`
   - `GET /api/cart`
   - `POST /api/orders`
   - `GET /api/orders`

---

## Product CRUD Examples

### Browser console (cookie auth)

```js
const storeId = 11;
const created = await fetch('/api/products', {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    store_id: storeId,
    name: 'CRUD Test Product',
    description: 'Initial',
    price: 12.5,
    stock: 7,
    image_url: ''
  })
}).then(r => r.json());

const productId = created.id;

await fetch(`/api/products/${productId}`, {
  method: 'PUT',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'CRUD Test Product (Edited)', price: 15.99, stock: 4 })
});

await fetch(`/api/products/${productId}`, { method: 'DELETE', credentials: 'include' });
```

### PowerShell (Bearer auth)

```powershell
$base = "http://localhost:8080"
$token = "PASTE_SESSION_TOKEN"
$auth = @{ Authorization = "Bearer $token" }

$created = Invoke-RestMethod -Uri "$base/api/products" -Method Post `
  -Headers ($auth + @{ "Content-Type" = "application/json" }) `
  -Body (@{
    store_id    = 11
    name        = "CRUD Test Product"
    description = "Initial"
    price       = 12.50
    stock       = 7
    image_url   = ""
  } | ConvertTo-Json -Compress)

$productId = $created.id

Invoke-RestMethod -Uri "$base/api/products/$productId" -Method Put `
  -Headers ($auth + @{ "Content-Type" = "application/json" }) `
  -Body (@{ name = "CRUD Test Product (Edited)"; price = 15.99; stock = 4 } | ConvertTo-Json -Compress)

Invoke-RestMethod -Uri "$base/api/products/$productId" -Method Delete -Headers $auth
```

---

## Notes

- Token and cookie issues are the most common local-dev failures; see troubleshooting in [AUTH-BRIDGE.md](./AUTH-BRIDGE.md).
- API currently uses a simple `{"error":"..."}` contract rather than a versioned error envelope.
- Avoid sharing live JWTs in chat, screenshots, or logs; use short-lived local tokens and refresh after debugging.
