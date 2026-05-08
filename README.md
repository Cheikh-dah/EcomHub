# EcomHub

MVP platform where merchants create online storefronts and customers discover stores/products through a shared ecosystem.

## Documentation

The current product, stack, routing, schema, environment, roadmap, and architecture notes live in [ECOMHUB.md](./ECOMHUB.md).

## Quick Start

1. Start Postgres:

   ```powershell
   docker compose -f ecomhub/docker-compose.yml up -d
   ```

2. Copy environment variables:

   ```powershell
   Copy-Item ecomhub/.env.example ecomhub/.env
   ```

3. Fill Clerk/Postgres values in `ecomhub/.env`.

4. Run from `ecomhub/`:

   ```powershell
   go run ./cmd/server
   ```

## Repo Layout

| Path | Purpose |
|------|---------|
| `ECOMHUB.md` | Current MVP reference |
| `ecomhub/` | Go application source |
| `ecomhub/docs/` | Focused implementation docs |
| `Get idea/` | Exploratory / legacy notes |
