# Paywall route

## Purpose

The paywall is a public, branded HTML page served when the proxy denies a request because the caller's tier doesn't match the rule's required tier (`DecisionPaywallRedirect` — see `contracts/decision_kinds.md`). The page renders inline HTML keyed on the calling application's branding and carries the required tier + a return URL so the upgrade CTA can loop the caller back after payment.

It is the one page in the v1.5 surface that is intentionally unauthenticated — the caller is, by definition, someone the proxy just refused.

## Route

| Method | Path | Handler symbol |
|---|---|---|
| GET | `/paywall/{app_slug}` | `Server.handlePaywallPage` |

## Auth required

**None.** Public. The `{app_slug}` path param identifies which branding/palette to render; the `tier` query param is what the proxy populated from `Decision.RequiredTier`; the `return` param is the original URL so the upgrade CTA can return the caller after payment.

## Request shape

Query parameters (all read from `r.URL.Query()`):

- `tier` (string, required) — the tier the denied rule wanted. Sanitized via an allowlist (`[a-zA-Z0-9 _-]`) before embedding in attribute/text context so a URL-param injection can't break out.
- `return` (string, optional) — absolute or relative URL to return the caller to after payment. Rejected if the scheme is anything other than `http(s):` or a bare relative path — `javascript:`, `data:`, `file:` are stripped via `sanitizeReturnURL`. Empty or unsafe values fall back to `/`.

Example:

```
GET /paywall/my-app?tier=pro&return=https%3A%2F%2Fapp.example.com%2Fdashboard
```

## Response shape

### Success (200)

`Content-Type: text/html; charset=utf-8`. Rendered HTML including:

- The app's name (from the application row).
- The required tier label.
- Primary/secondary/font CSS variables sourced from `Store.ResolveBranding` for that app.
- An "Upgrade" CTA pointing at `<return>?upgrade=<tier>`.

### Error

- Plain-text `tier query param required` body on missing tier.
- Standard 404 `Not Found` on unknown app slug.

## Status codes

- `200 OK` — page rendered successfully.
- `400 Bad Request` — `tier` query param missing or empty.
- `404 Not Found` — unknown `app_slug`. Also returned when branding lookup fails, so an attacker can't enumerate which apps exist via differential responses.

## Side effects

- Read-only: `GetApplicationBySlug` + `ResolveBranding`. No DB writes, no audit entries.
- Template is rendered inline — no filesystem reads, no external fetches, no user sessions consulted.

## Frontend hint

The paywall is rendered server-side, so the dashboard's only job is a *preview* surface: a "Preview paywall" action on the Branding page opens `/paywall/{current_app_slug}?tier=pro&return=/` in an iframe or a new tab. Pair with a tier dropdown so designers can preview each tier's copy. No SDK method is needed — the paywall is exclusively reached via proxy redirect; treat this as a documentation-only endpoint for dashboard integrators.
