# EcomHub REST API Reference

This document describes the current JSON API surface and the API direction needed for future frontend separation.

For auth internals, see [AUTH-BRIDGE.md](./AUTH-BRIDGE.md).
For product architecture, see [ECOMHUB-CHEATSHEETS.md](./ECOMHUB-CHEATSHEETS.md).

## Scope

- Current authenticated merchant APIs are served by the Go backend.
- Current public storefront and hub pages are mostly Go SSR HTML.
- Future Next.js storefront pages should consume stable public JSON endpoints from Go.

## Base URL

Local development:

```text
http://localhost:8080
```

Production:

```text
https://your-app-origin
```

## Authentication

Protected endpoints accept either:

- `Authorization: Bearer <clerk_session_jwt>`
- `auth_token` HttpOnly cookie set by `POST /dashboard/session`

`GET /api/me` is the fastest auth check.

```json
{
  "user_id": "a682e5da-7df4-474f-ac83-71cf374e13a9"
}
```

## Error Format

Current errors use a simple object:

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

## Status Codes

| Status | Meaning |
| --- | --- |
| `200 OK` | Successful read/update/delete |
| `201 Created` | Resource created |
| `400 Bad Request` | Invalid body, id, query, or validation error |
| `401 Unauthorized` | Missing or invalid auth |
| `403 Forbidden` | User does not own the resource |
| `404 Not Found` | Resource does not exist or is intentionally hidden |
| `409 Conflict` | Unique constraint conflict |
| `500 Internal Server Error` | Unexpected server/database error |

## Auth Endpoints

### `POST /api/logout`

Auth: none

Clears the `auth_token` cookie.

```json
{ "ok": true }
```

### `GET /api/me`

Auth: required

```json
{ "user_id": "<uuid>" }
```

## Merchant Store Endpoints

### `GET /api/stores`

Auth: required

Returns stores owned by the authenticated merchant.

```json
[
  {
    "id": 11,
    "user_id": "a682e5da-7df4-474f-ac83-71cf374e13a9",
    "name": "My Store",
    "subdomain": "my-store",
    "description": "Optional",
    "status": "active",
    "created_at": "2026-04-25T09:00:00Z"
  }
]
```

### `POST /api/stores`

Auth: required

```json
{
  "name": "My Store",
  "subdomain": "my-store",
  "description": "Optional"
}
```

Validation:

- `name` is required.
- `subdomain` is required.
- `subdomain` is normalized/lowercased server-side.
- `subdomain` must be unique.

Response:

```json
{
  "id": 12,
  "subdomain": "my-store"
}
```

### `PUT /api/stores/:id`

Auth: required

Caller must own the store.

```json
{
  "name": "Updated Store Name",
  "subdomain": "updated-store",
  "description": "Updated description"
}
```

Response:

```json
{ "ok": true }
```

## Store Theme Endpoints

### `GET /api/stores/:id/theme`

Auth: required

Caller must own the store.

Returns the current theme or a normalized default theme.

```json
{
  "primary_color": "#1d9bf0",
  "accent_color": "#00ba7c",
  "page_bg": "#ffffff",
  "text_color": "#111111",
  "card_bg": "#f9fafb",
  "footer_bg": "#ffffff",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default",
  "rounding": 0.4,
  "preset": "minimal",
  "version": 1
}
```

### `PUT /api/stores/:id/theme`

Auth: required

Patch semantics: include only the fields to change.

```json
{
  "primary_color": "#111827",
  "accent_color": "#16a34a",
  "logo_url": "https://example.com/logo.png",
  "layout_preset": "default"
}
```

Validation:

- Color fields must be valid `#RRGGBB` hex values or omitted.
- `logo_url` must be an absolute HTTP(S) URL, empty, or omitted.
- `layout_preset` must be `default` or `compact`.
- Caller must own the store.

Response: full normalized theme.

## Product Endpoints

### `GET /api/products?store_id=<id>`

Auth: required

Caller must own the store.

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

### `POST /api/products`

Auth: required

```json
{
  "store_id": 11,
  "name": "New Product",
  "description": "Optional",
  "price": 12.5,
  "stock": 7,
  "image_url": "https://example.com/product.jpg"
}
```

Validation:

- `store_id` is required.
- `name` is required.
- `price >= 0`.
- `stock >= 0`.
- Caller must own the store.

Response:

```json
{ "id": 3 }
```

### `PUT /api/products/:id`

Auth: required

Partial update.

```json
{
  "name": "New Name",
  "description": "Updated",
  "price": 15.99,
  "stock": 4,
  "image_url": "https://example.com/new.jpg"
}
```

Response:

```json
{ "ok": true }
```

### `DELETE /api/products/:id`

Auth: required

Caller must own the product's store.

```json
{ "ok": true }
```

## Cart And Orders

These endpoints still exist, but current MVP customer purchasing direction is WhatsApp/manual merchant completion rather than a full checkout system.

Do not expand cart/checkout behavior until the product direction is confirmed.

Current endpoints:

- `GET /api/cart`
- `POST /api/cart/add`
- `POST /api/cart/remove`
- `POST /api/cart/clear`
- `POST /api/orders`
- `GET /api/orders`

## Current Public HTML Endpoints

These are rendered by Go templates today:

- `GET /`
- `GET /products`
- `GET /stores`
- `GET /search?q=<term>`
- `GET /s/:subdomain`
- `GET /s/:subdomain/products/:id`
- `GET /s/:subdomain/cart`

## Future Public JSON Endpoints

These are the recommended API boundary for the future Next.js storefront:

```text
GET /api/public/stores/:subdomain
GET /api/public/stores/:subdomain/products
GET /api/public/stores/:subdomain/products/:id
GET /api/public/hub/products
GET /api/public/hub/stores
GET /api/public/search?q=<term>
```

Guidelines:

- Return stable JSON independent of Go templates.
- Include `theme`, `store`, and `products` shapes needed by storefront pages.
- Keep merchant-owned media as remote URLs (`image_url`, `logo_url`).
- Do not expose private merchant fields.
- Use the same semantic rendering intent as the current frontend: product cards use cover images, product detail can use contain images.

## Browser Console Example

```js
const me = await fetch('/api/me', {
  credentials: 'include'
}).then(r => r.json());
```

## PowerShell Example

```powershell
$base = "http://localhost:8080"
$token = "PASTE_SESSION_TOKEN"
$auth = @{ Authorization = "Bearer $token" }

Invoke-RestMethod -Uri "$base/api/me" -Headers $auth
```

## Notes

- Avoid sharing live JWTs in chats, screenshots, or logs.
- API errors are not versioned yet.
- Add CSRF protection before serious production use of destructive cookie-authenticated POST actions.
