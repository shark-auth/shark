# Admin API — Set user tier

## Purpose

Persists a caller-supplied tier (`free` or `pro` — anything else rejects with 400) into `users.metadata` and returns the freshly-read user. The tier feeds two downstream consumers: the JWT Claims baker (so the tier travels in every access token issued after the change) and the proxy `ReqTier` predicate (so paywall redirects fire without extra DB lookups on the hot path).

Only `free` and `pro` are currently recognised. Adding a new tier is a three-step change: extend the validation here, update the Claims baker to accept it, and document the new value in `contracts/require_grammar.md`.

## Route

| Method | Path | Handler symbol |
|---|---|---|
| PATCH | `/api/v1/admin/users/{id}/tier` | `Server.handleSetUserTier` |

## Auth required

Admin API key.

## Request shape

```json
{ "tier": "pro" }
```

- `tier` (string, required): one of `free`, `pro`. Case-insensitive — the handler lowercases before compare.

## Response shape

### Success (200)

```json
{
  "data": {
    "user": { ...full user object... },
    "tier": "pro"
  }
}
```

### Error

```json
{ "error": { "code": "invalid_tier", "message": "tier must be \"free\" or \"pro\"" } }
```

Error codes: `invalid_request` (bad JSON or missing user id), `invalid_tier` (unknown tier), `not_found` (404 — no user with that id).

## Status codes

- `200 OK` — tier persisted, fresh user returned.
- `400 Bad Request` — bad JSON, missing user id param, unrecognised tier string.
- `401 Unauthorized` — missing/invalid admin key.
- `404 Not Found` — user id does not exist. Distinguished from a write failure so dashboards can show "user deleted?" rather than "retry".
- `500 Internal Server Error` — DB write or read-back failed.

## Side effects

- DB write: updates `users.metadata` (JSON blob) with the new tier via `Store.SetUserTier`. Other metadata fields are preserved.
- JWT Claims propagation: the tier is baked into access tokens at issue time, so existing tokens retain their old tier until refresh. If you need an immediate cutover, revoke the user's refresh tokens separately.
- Proxy behaviour: `ReqTier`-gated rules re-evaluate on the next request using the new tier from the Identity struct. Mismatch triggers a `DecisionPaywallRedirect` (see `contracts/decision_kinds.md`).
- Audit log: `user.tier.set` entry keyed on the acting admin session; metadata carries the new tier value so the timeline reads as a tier-flip history.

## Frontend hint

Render a compact tier selector (two-button segmented control: Free / Pro) on the user detail page. Optimistic update is fine — the response round-trips quickly and the error surface is narrow (unknown tier or 404). Pair with a "tokens expire in N min" tooltip so operators understand that the old tier may still appear in access tokens until the user's next refresh; an inline "force logout" secondary action can call into the existing sessions-revoke endpoint if immediate cutover matters.
