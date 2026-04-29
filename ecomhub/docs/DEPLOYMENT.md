# Deployment Guide

EcomHub is a Go application with a PostgreSQL database and Clerk authentication.

## Recommended: Render (Easiest)

Render is the simplest path for this stack.

### 1. Create a PostgreSQL Database
- Go to [dashboard.render.com](https://dashboard.render.com)
- New → Database
- Name: `ecomhub-db`
- Copy the **Internal Database URL** for later.

### 2. Create a Web Service
- New → Web Service
- Connect your GitHub repository.
- **Language**: `Go`
- **Build Command**: `go build -o ecomhub ./cmd/server`
- **Start Command**: `./ecomhub`

### 3. Environment Variables
Add these in the Render dashboard:
- `DATABASE_URL`: (Paste your Internal Database URL)
- `CLERK_SECRET_KEY`: (Your Clerk Secret Key)
- `CLERK_PUBLISHABLE_KEY`: (Your Clerk Publishable Key)
- `ENVIRONMENT`: `production`
- `APP_URL`: `https://your-app-name.onrender.com`
- `PORT`: `8080` (Render will use this)

---

## Alternative: Fly.io (Docker-based)

If you prefer Fly.io, use the provided `Dockerfile`.

1. **Install Fly CLI** and run `fly launch`.
2. It will detect the `Dockerfile`.
3. Create a **Postgres cluster** when prompted.
4. Set secrets:
   ```bash
   fly secrets set CLERK_SECRET_KEY=... CLERK_PUBLISHABLE_KEY=... ENVIRONMENT=production APP_URL=...
   ```

---

## Production Checklist

### 1. Database Migrations
The app currently doesn't run migrations automatically on start. 
**Action**: You should run the `.sql` files in `migrations/` against your production database using a tool like `psql` or a Go migration library.

### 2. Clerk Configuration
In the [Clerk Dashboard](https://dashboard.clerk.com):
- Add your production URL (`https://your-app-name.onrender.com`) to **Authorized Origins**.
- Ensure the **JWT template** matches the one used in development (standard Clerk session).

### 3. Environment Variables
Ensure `ENVIRONMENT` is set to `production`. This enables secure cookie settings (SameSite=Lax, Secure=true).
