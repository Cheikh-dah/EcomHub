# EcomHub

MVP platform: sellers create stores and share public storefronts; buyers browse on **subdomains** (`https://{store}.ecomhub.com`) while dashboard and hub stay on the apex domain.

## Documentation

All product, stack, routing, schema, env, roadmap, and acceptance criteria live in **[ECOMHUB.md](./ECOMHUB.md)**.

## Quick start

1. Postgres: `docker compose -f ecomhub/docker-compose.yml up -d`  
2. Env: copy `ecomhub/.env.example` → `ecomhub/.env` and fill values (see `ECOMHUB.md` §7).  
3. Run: from `ecomhub/`, `go run ./cmd/server`

## Repo layout

| Path | Purpose |
|------|---------|
| `ECOMHUB.md` | Full MVP reference |
| `ecomhub/` | Application source |
| `Get idea/` | Exploratory / legacy notes |
