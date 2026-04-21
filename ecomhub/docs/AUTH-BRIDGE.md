# EcomHub — Auth bridge (Clerk + Go + browser)

This document is the **single reference** for how authentication works today: identity (Clerk), the **session bridge** (browser ↔ server), and **backend authority** (JWT verification + Postgres).

For product and scaling context, see [ECOMHUB-CHEATSHEETS.md](./ECOMHUB-CHEATSHEETS.md).

---

## 1) Three layers

```text
┌─────────────────────────────────────────────────────────────┐
│ 1. Identity (Clerk)                                         │
│    Browser: Clerk JS + session JWT from Clerk              │
└───────────────────────────┬─────────────────────────────────┘
                            │ POST /dashboard/session
                            │ JSON: session_token (or access_token)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Session bridge                                            │
│    Server sets HttpOnly cookie `auth_token` = Clerk JWT    │
│    Max-Age derived from JWT `exp`                            │
└───────────────────────────┬─────────────────────────────────┘
                            │ Cookie or Authorization: Bearer
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. Backend authority                                         │
│    Verify JWT (Clerk JWKS) → `sub` (Clerk user id)         │
│    Resolve → `user_identities` (`provider=clerk`) →       │
│    internal `users.id` → APIs + HTML enforce ownership      │
└─────────────────────────────────────────────────────────────┘
```

**Rules of thumb**

- **Clerk** decides *who signed in* in the browser.
- **Go** decides *whether this request is authenticated* and *which internal `user_id`* it maps to.
- **Postgres** is the source of truth for stores, products, and `(provider, provider_subject)` identity links.

---

## 2) Environment variables

| Variable | Required | Role |
|----------|----------|------|
| `CLERK_SECRET_KEY` | Yes | `sk_test_…` / `sk_live_…`. Set via `clerk.SetKey` in `cmd/server/main.go` for Backend API (JWKS fetch, `user.Get` on JIT). |
| `CLERK_PUBLISHABLE_KEY` | Yes | `pk_test_…` / `pk_live_…`. Embedded in dashboard bootstrap JSON for Clerk JS only. |
| `CLERK_FRONTEND_API` | No | Full origin from Clerk **API keys** (Frontend API), e.g. `https://YOUR_INSTANCE.clerk.accounts.dev`. If unset, the dashboard derives the origin from the publishable key (Clerk’s documented pattern). |
| `CLERK_AUTHORIZED_PARTIES` | No | Comma-separated **exact** origins allowed in the JWT `azp` claim. If unset and `APP_URL` is set, `APP_URL` (trimmed, no trailing slash) is added as a single allowed party. |
| `APP_URL` | Recommended | Public base URL of this app (e.g. `http://localhost:8080`). Used for `azp` when parties list is empty. |
| `ENVIRONMENT` | Yes | `development` \| `staging` \| `production`. Cookie `Secure` is on when `production`. |

Copy from [.env.example](../.env.example). Never commit real `.env`.

---

## 3) Server components

| Piece | Location | Behavior |
|-------|----------|----------|
| Clerk API key | `cmd/server/main.go` | `clerk.SetKey(cfg.ClerkSecretKey)` before serving traffic. |
| JWT verification | `internal/auth/clerk_session.go` | `jwt.Verify` (RS256 via JWKS). Optional `AuthorizedPartyHandler` when `ClerkAuthorizedParties` is non-empty. |
| User resolution | `internal/auth/identity.go` | `ResolveClerkUser`: lookup `user_identities` for `provider=clerk` + `provider_subject=sub`; on miss, `user.Get` then JIT `users` + identity row. |
| Auth middleware | `internal/middleware/auth.go` | `RequireAuth` / `OptionalAuth`: read `Authorization: Bearer` or `auth_token` cookie, verify + resolve, set `userID` on Gin context. |
| Session endpoint | `internal/httpserver/handlers_html.go` | `POST /dashboard/session`: verify token, resolve user, `setAuthCookie`, JSON `{ ok, redirect }`. |
| Logout | `POST /dashboard/logout` + `POST /api/logout` | Clear `auth_token`. HTML logout redirects to `/dashboard?signed_out=1` so the dashboard can run `Clerk.signOut()` and avoid an instant Clerk→server re-sync loop. |

---

## 4) HTTP contract (auth-related)

| Method | Path | Auth | Body / notes |
|--------|------|------|----------------|
| `POST` | `/dashboard/session` | No | JSON: `session_token` **or** `access_token` (Clerk session JWT), optional `next` (internal path only; server validates). Response: `{ "ok": true, "redirect": "/dashboard" }`. |
| `POST` | `/dashboard/logout` | No | Clears cookie; redirect `See Other` to `/dashboard?signed_out=1`. |
| `POST` | `/api/logout` | No | Clears cookie; JSON `{ "ok": true }`. |
| `GET` | `/api/me` | Yes | Cookie or `Authorization: Bearer` with same Clerk JWT. Response: `{ "user_id": "<uuid>" }`. |

Protected API routes live under `/api/…` with `RequireAuth`. Dashboard POST `/dashboard/stores` uses the same cookie.

---

## 5) Dashboard frontend (`internal/web/templates/dashboard.html`)

**Config delivery**

- Server renders `<script type="application/json" id="ecomhub-clerk-bootstrap">` with `ClerkBootstrapJSON` (`publishableKey`, `frontendAPI`).
- The module script `JSON.parse`s that blob with validation so templating never injects raw strings into executable JS.

**Lifecycle (order matters)**

1. Parse and validate bootstrap (publishable key non-empty).
2. Resolve Frontend API URL (override or derive).
3. Load `@clerk/ui` then `@clerk/clerk-js` from that origin.
4. `await Clerk.load({ ui: { ClerkUI: window.__internal_ClerkUICtor } })`.
5. Only then read `Clerk.isSignedIn`, `Clerk.session`, mount SignIn, or sync the server session.

**Session sync (idempotent bridge)**

- `syncServerSessionIfNewToken(token)` dedupes with `lastSyncedToken`, serializes with `syncing`, and clears `lastSyncedToken` on failed `POST /dashboard/session` so Clerk listeners can retry.
- `establishSession` uses an `establishing` guard to avoid overlapping fetches for the same logical operation.

**Logout UX**

- After server clears the cookie, `?signed_out=1` triggers `Clerk.signOut()` before mounting SignIn so a lingering Clerk session does not immediately re-POST `/dashboard/session`.

---

## 6) Database

- **`users`**: internal profile; `password_hash` may be null for Clerk-only users.
- **`user_identities`**: `provider = 'clerk'`, `provider_subject` = Clerk user id (JWT `sub`, e.g. `user_…`). Unique `(provider, provider_subject)`.
- Legacy rows with other `provider` values are not used by the current code path.

---

## 7) Clerk Dashboard checklist

- **Allowed origins / redirect URLs** must include your app origin (e.g. `http://localhost:8080`) so session tokens and `azp` align with `CLERK_AUTHORIZED_PARTIES` / `APP_URL`.
- Use **separate** Clerk instances or keys for production vs development when you go live.

---

## 8) Production checklist

- [ ] `ENVIRONMENT=production`, `APP_URL` and Clerk URLs use **https**.
- [ ] `CLERK_AUTHORIZED_PARTIES` explicitly lists every browser origin that may obtain a session JWT (do not rely on defaults alone in prod).
- [ ] Postgres is **not** exposed on the public internet; only the app tier can connect.
- [ ] Secrets only in env / secret manager, not in git.
- [ ] Plan **backups** and a tested restore for Postgres.

---

## 9) Troubleshooting

| Symptom | Things to check |
|---------|------------------|
| `401` on `/api/me` or APIs | Cookie missing or expired; `Authorization: Bearer` wrong token; Clerk JWT secret path OK (JWKS + `SetKey`). |
| `invalid token` on `/dashboard/session` | Clock skew; wrong Clerk instance keys; `azp` rejected — set `CLERK_AUTHORIZED_PARTIES` / `APP_URL` to match the browser origin Clerk uses. |
| Dashboard blank / “Invalid auth configuration” | `CLERK_PUBLISHABLE_KEY` empty in env; server not restarted; malformed JSON in bootstrap (should not happen if config loads). |
| `.env` has keys but Go sees them **empty** (Windows) | A **User or System** environment variable with the same name (sometimes set to nothing) takes precedence over `.env` with default `godotenv.Load`. This project uses **`godotenv.Overload()`** in `config.Load()` so **`.env` wins** after all. If you still see issues, remove stray `CLERK_*` / `DATABASE_URL` entries from Windows “Environment variables”. |
| PowerShell `KEY=value` fails | That is **bash** syntax. Set vars in **`.env`** or use `$env:KEY = "value"` for the current session only. |
| Clerk scripts fail to load | Set `CLERK_FRONTEND_API` explicitly; check ad blockers; confirm Frontend API URL in Clerk **API keys** page. |
| “Could not resolve user” | Clerk `user.Get` failed (network, key, or user deleted in Clerk). |

---

## 10) Optional future work (not implemented)

| Item | Purpose |
|------|---------|
| **Auth bridge spec versioning** | If mobile or third-party clients also sync sessions, document their `POST` contract alongside the dashboard. |
| **Clerk Organizations → RBAC** | Map org roles / permissions from JWT claims to route-level authorization in Go. |
| **Refresh / rotation** | Today the browser relies on Clerk’s session lifecycle; explicit refresh policies can be added if session length becomes a product requirement. |

---

## 10.1) Dashboard scope guidance

Given the current codebase shape (Go SSR templates + backend route guards), treat these as separate workstreams:

- **Now:** strengthen current dashboard SSR flows and auth/session reliability.
- **Later:** introduce a separate client-side dashboard route architecture *only if needed*.

This avoids mixing three high-risk changes at once:

1. identity/session migration,
2. frontend architecture migration,
3. API envelope redesign.

Keep backend middleware as security authority in both models.

---

## 11) Host-based subdomains (SaaS routing)

Current storefront URLs support `/s/{subdomain}`. To match production SaaS routing (`store.example.com`) add host-first resolution and keep path fallback during rollout.

### Local development

- Keep `BASE_HOST=localhost`.
- Use `store1.localhost:8080`, `store2.localhost:8080` where supported by your browser/OS.
- Keep `/s/{subdomain}` as fallback if local host resolution is unavailable.

### Render (single service) baseline

- One Render web service can host many tenant subdomains.
- Configure DNS for apex plus wildcard (`*.yourdomain.com`) to the same service.
- Set `BASE_HOST=yourdomain.com` in Render environment variables.
- Ensure TLS/certificate coverage for wildcard and apex based on Render's current domain features.

### Middleware contract (target)

1. Read `Host` (strip port).
2. If host matches `{subdomain}.{BASE_HOST}`, resolve store by `subdomain` and `status='active'`.
3. Attach resolved store/tenant id to request context.
4. If not resolvable, continue existing hub/dashboard routes and `/s/{subdomain}` fallback.

This gives local and production the same tenant-identification model while preserving backward compatibility.

---

*Last aligned with:* Clerk session JWT verification in Go, dashboard bootstrap JSON pattern, and `user_identities.provider = clerk`.
