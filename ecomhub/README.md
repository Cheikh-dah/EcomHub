# EcomHub App

Multi-tenant commerce platform built with Go, Gin, PostgreSQL, and Clerk.

The current app is a Go SSR + JSON API monolith. The future frontend direction is a gradual Next.js/Vercel migration, starting with storefront pages after the Go API boundary is ready.

## Current Features

- **Merchant Dashboard**: Create and manage multiple stores.
- **Theme Engine**: Storefront colors, logo URL, layout preset, rounding, and live preview.
- **Lightweight Marketplace**: Search products across all active stores.
- **Storefront Search**: Built-in search functionality for individual stores.
- **Secure Authentication**: Integrated with Clerk for user identity and session management.
- **Storefront Media Semantics**: Product cards/details use stable `product-media` classes that map cleanly to a future `ProductImage` component.
- **Hot Reloading**: Developer-friendly setup with Air.

## Tech Stack

- **Backend**: Go (Golang) with Gin Gonic.
- **Database**: PostgreSQL (pgx driver).
- **Auth**: Clerk (Server-side JWT verification + Browser cookie bridge).
- **Frontend Now**: Server-rendered Go HTML templates + vanilla CSS.
- **Frontend Later**: Next.js on Vercel, backed by the Go JSON API.

## Getting Started

### Prerequisites
- Go 1.26+
- Docker (for PostgreSQL)
- Clerk account and API keys

### Setup

1. **Clone the repo**
2. **Setup environment variables**
   ```bash
   cp .env.example .env
   # Update CLERK_SECRET_KEY, etc.
   ```
3. **Start the database**
   ```bash
   docker-compose up -d
   ```
4. **Run with hot reload**
   ```bash
   air
   ```

## Documentation

- [Project Reference](../ECOMHUB.md)
- [Theme Engine Strategy (v2)](docs/STORE-DESIGN-STRATEGY.md)
- [Shippable Theme Plan](docs/SHIPPABLE-THEME-PLAN.md)
- [Theme Customization](docs/THEME-CUSTOMIZATION.md)
- [Auth Bridge Implementation](docs/AUTH-BRIDGE.md)
- [REST API Reference](docs/REST-API-REFERENCE.md)
- [Cheatsheets](docs/ECOMHUB-CHEATSHEETS.md)

## Migration Direction

Do not rewrite the whole frontend at once.

1. Keep current Go SSR working.
2. Make public storefront APIs clean and documented.
3. Build the Next.js storefront first.
4. Move dashboard/theme editor later.
5. Keep Go as backend authority for auth, ownership, and database writes.
