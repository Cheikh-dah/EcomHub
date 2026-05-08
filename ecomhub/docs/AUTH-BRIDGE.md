# EcomHub Auth Bridge

This document explains the current Clerk + Go authentication model.

For API payloads, see [REST-API-REFERENCE.md](./REST-API-REFERENCE.md).
For product architecture, see [ECOMHUB-CHEATSHEETS.md](./ECOMHUB-CHEATSHEETS.md).

## Mental Model

```text
1. Clerk identifies the browser user.
2. Browser gets a Clerk session JWT.
3. Browser POSTs that JWT to /dashboard/session.
4. Go verifies the JWT with Clerk.
5. Go maps Clerk sub -> internal users.id through user_identities.
6. Go sets an HttpOnly auth_token cookie.
7. Dashboard/API requests use the cookie or a Bearer token.
```

Clerk decides who signed in. Go decides whether a request is authorized and which internal user owns the requested data.

## Environment Variables

| Variable | Required | Purpose |
| --- | --- | --- |
| `CLERK_SECRET_KEY` | Yes | Backend secret key used by Go for Clerk API/JWKS work |
| `CLERK_PUBLISHABLE_KEY` | Yes | Browser publishable key used by Clerk JS |
| `CLERK_FRONTEND_API` | No | Explicit Clerk frontend origin override |
| `CLERK_AUTHORIZED_PARTIES` | No | Exact browser origins allowed in JWT `azp` |
| `APP_URL` | Recommended | Public app origin; can seed authorized parties |
| `ENVIRONMENT` | Yes | `development`, `staging`, or `production` |

Never commit real `.env` values.

## Server Components

| Component | Location | Responsibility |
| --- | --- | --- |
| Clerk setup | `cmd/server/main.go` | Calls `clerk.SetKey` |
| JWT verification | `internal/auth/clerk_session.go` | Verifies Clerk session JWTs |
| Identity resolution | `internal/auth/identity.go` | Maps Clerk users to internal users |
| Auth middleware | `internal/middleware/auth.go` | Reads cookie/Bearer token and sets `userID` |
| Session bridge | `internal/httpserver/handlers_html.go` | Handles `POST /dashboard/session` |
| Logout | HTML/API handlers | Clears `auth_token` |

## HTTP Contract

### `POST /dashboard/session`

Auth: none.

Body:

```json
{
  "session_token": "<clerk-session-jwt>",
  "next": "/dashboard"
}
```

`access_token` is also accepted for compatibility.

Behavior:

- Verifies the Clerk JWT.
- Resolves/creates the internal user identity.
- Sets `auth_token` as an HttpOnly cookie.
- Returns a safe internal redirect path.

Response:

```json
{
  "ok": true,
  "redirect": "/dashboard"
}
```

### `GET /api/me`

Auth: required.

Accepts either `auth_token` cookie or `Authorization: Bearer <clerk_session_jwt>`.

```json
{
  "user_id": "<uuid>"
}
```

### `POST /dashboard/logout`

Clears `auth_token` and redirects to `/dashboard?signed_out=1`.

### `POST /api/logout`

Clears `auth_token`.

```json
{ "ok": true }
```

## Cookie Policy

`auth_token` should be:

- HttpOnly.
- SameSite=Lax.
- Secure in production.
- Max age derived from the Clerk JWT expiry.

CSRF protection is not complete yet. SameSite=Lax helps, but destructive dashboard forms should get CSRF tokens before serious production use.

## Dashboard Frontend Lifecycle

The dashboard must:

1. Load Clerk JS using the server-provided publishable key.
2. Wait for `Clerk.load`.
3. Check `Clerk.isSignedIn` and `Clerk.session`.
4. Call `Clerk.session.getToken()`.
5. POST the token to `/dashboard/session`.
6. Continue to the intended dashboard page.

Do not send `document.cookie` as the session token. Clerk cookies are not the same as the Clerk session JWT returned by `Clerk.session.getToken()`.

## Background Session Sync

Dashboard pages can include the `clerk_sync` partial to keep the Go cookie aligned with Clerk while the merchant works.

This is useful for pages like the theme editor, where the merchant may stay active for a while without leaving the page.

## Database

`user_identities` is the bridge between Clerk and EcomHub:

```text
provider = "clerk"
provider_subject = Clerk JWT sub
user_id = internal users.id
```

Authorization must use internal ownership fields after identity resolution.

Example:

```sql
SELECT * FROM stores WHERE id = $1 AND user_id = $2;
```

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `window.Clerk` is undefined | Clerk JS did not load or bootstrap data is missing |
| `session_token required` | Request body was empty or token variable was undefined |
| `invalid token` | Sent Clerk cookie text instead of `Clerk.session.getToken()` result, wrong Clerk instance, or `azp` mismatch |
| `/api/me` returns 401 | Missing/expired cookie or Bearer token |
| User can sign in but dashboard loops | Server cookie not being created or cleared/synced correctly |

## Future Next.js Direction

When a Next.js frontend is introduced, Go should remain the authorization authority.

Good migration options:

- Next.js gets a Clerk session token and calls Go APIs with `Authorization: Bearer`.
- Or Next.js routes complete the same session bridge and rely on the Go cookie.

Do not combine all of these in one step:

- frontend rewrite,
- auth rewrite,
- API contract redesign.

Move one boundary at a time.

## Production Checklist

- Use HTTPS.
- Set `ENVIRONMENT=production`.
- Set exact `CLERK_AUTHORIZED_PARTIES`.
- Keep database credentials private.
- Avoid logging JWTs.
- Add CSRF tokens for destructive POST actions.
- Use separate Clerk keys for development and production.
