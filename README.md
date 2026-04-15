# EcomHub

EcomHub is an MVP e-commerce platform for creating simple online stores and sharing public storefront links.

## MVP Stack
- Backend: Go + Gin
- Database: PostgreSQL (`pgxpool`)
- Auth: Supabase Auth + internal user mapping
- UI (MVP): server-rendered pages from the Go app
- Local development: Docker Compose + local Go server

## Repository Structure
- `PRD.md` - Product requirements for MVP scope
- `MVP-ROADMAP.md` - phased implementation roadmap
- `STACK.md` - locked MVP technology choices
- `ecomhub/` - application source code
- `Get idea/` - exploratory/legacy notes and prototypes

## Local Setup (MVP)
1. Start PostgreSQL:
   - `docker compose -f ecomhub/docker-compose.yml up -d`
2. Configure environment:
   - copy `ecomhub/.env.example` to `ecomhub/.env`
   - set required values (`DATABASE_URL`, Supabase variables, etc.)
3. Run the backend:
   - from `ecomhub/`: `go run ./cmd/server`

## Current Status
- Core MVP docs are aligned and committed:
  - `PRD.md`
  - `MVP-ROADMAP.md`
  - `STACK.md`
- Next step is implementing the managed-auth flow and MVP hardening in `ecomhub/`.
