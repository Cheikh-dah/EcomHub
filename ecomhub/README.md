# EcomHub

Multi-tenant e-commerce platform built with Go, Gin, and Clerk.

## Features

- **Merchant Dashboard**: Create and manage multiple stores.
- **DNA Theme Engine**: Designer-level storefronts with zero effort using Brand Presets (Minimal, Luxury, Playful) and Corner Rounding DNA.
- **Storefront Customization**: Dynamic colors, logos, and layout presets (Default/Compact).
- **Lightweight Marketplace**: Search products across all active stores.
- **Storefront Search**: Built-in search functionality for individual stores.
- **Secure Authentication**: Integrated with Clerk for user identity and session management.
- **Hot Reloading**: Developer-friendly setup with Air.

## Tech Stack

- **Backend**: Go (Golang) with Gin Gonic.
- **Database**: PostgreSQL (pgx driver).
- **Auth**: Clerk (Server-side JWT verification + Browser cookie bridge).
- **Frontend**: Server-rendered Go HTML templates + Vanilla CSS.

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

- [Theme Engine Strategy (v2)](docs/STORE-DESIGN-STRATEGY.md)
- [Shippable Theme Plan](docs/SHIPPABLE-THEME-PLAN.md)
- [Theme Customization (Legacy)](docs/THEME-CUSTOMIZATION.md)
- [Auth Bridge Implementation](docs/AUTH-BRIDGE.md)
- [REST API Reference](docs/REST-API-REFERENCE.md)
- [Cheatsheets](docs/ECOMHUB-CHEATSHEETS.md)
