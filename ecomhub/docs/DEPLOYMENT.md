# Deployment Guide

EcomHub currently deploys as a Go/Gin application with PostgreSQL and Clerk authentication.

The recommended MVP deployment is:

```text
Browser -> Go SSR/API on Render -> PostgreSQL
                         |
                         +-> Clerk for identity/session tokens
```

The future frontend direction is:

```text
Browser -> Next.js on Vercel -> Go API on Render -> PostgreSQL
                                |
                                +-> Clerk for identity/session tokens
```

Do not move everything at once. Keep the Go SSR app working while API boundaries are made ready for a gradual Next.js migration.

## Render

Render is the simplest current production path.

### 1. Create PostgreSQL

1. Open [Render](https://dashboard.render.com).
2. Create a new PostgreSQL database.
3. Copy the internal database URL.

### 2. Create Web Service

1. Create a new web service from the GitHub repository.
2. Use Go as the runtime.
3. Set the build command:

```bash
go build -o ecomhub ./cmd/server
```

4. Set the start command:

```bash
./ecomhub
```

### 3. Environment Variables

Set these in Render:

| Variable | Purpose |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection string |
| `CLERK_SECRET_KEY` | Clerk backend secret key |
| `CLERK_PUBLISHABLE_KEY` | Clerk browser publishable key |
| `CLERK_FRONTEND_API` | Optional Clerk frontend origin override |
| `CLERK_AUTHORIZED_PARTIES` | Optional exact browser origins allowed in Clerk JWT `azp` |
| `ENVIRONMENT` | Use `production` in production |
| `APP_URL` | Public app URL, for example `https://ecomhub-wd00.onrender.com` |
| `PORT` | Render provides this automatically; `8080` is fine locally |

## Database Migrations

The app does not currently run migrations automatically at startup.

Run the SQL files in `migrations/` against the production database before serving traffic. Use `psql`, a migration tool, or a controlled deployment step.

## Clerk Checklist

In the Clerk dashboard:

- Add local and production origins to allowed origins/redirect URLs.
- Use separate Clerk instances or keys for development and production.
- Make sure `APP_URL` and `CLERK_AUTHORIZED_PARTIES` match the browser origin that receives Clerk session JWTs.
- In production, use HTTPS only.

## Future Vercel / Next.js Deployment

Vercel is attractive for the future storefront because it supports frontend routing and wildcard subdomain patterns.

Recommended migration path:

1. Keep Go SSR on Render working.
2. Add stable public JSON endpoints in Go for storefront data.
3. Build the Next.js storefront first.
4. Route storefront traffic through Vercel.
5. Move dashboard pages later if the API boundary is clean.

Do not start with a full rewrite. Subdomain routing is a good reason to introduce Vercel later, but the backend API contract should come first.

## Production Hardening

- Add CSRF tokens for destructive dashboard POST forms.
- Keep `auth_token` as HttpOnly and Secure in production.
- Avoid logging live Clerk JWTs.
- Keep merchant media as remote URLs for now; upload/CDN/image processing can be a later subsystem.
