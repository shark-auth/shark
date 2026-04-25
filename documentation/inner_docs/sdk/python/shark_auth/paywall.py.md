# paywall.py

**Path:** `sdk/python/shark_auth/paywall.py`
**Module:** `shark_auth.paywall`
**LOC:** 106

## Purpose
Helpers for the public paywall page at `GET /paywall/{app_slug}` — URL builder (no network) plus HTML fetchers for rendering and previewing.

## Public API
- `class PaywallClient`
  - `__init__(base_url, token=None, *, session=None)` — `token` is optional (paywall is unauthenticated); pass it only when sharing the client with authed endpoints
  - `.paywall_url(app_slug, tier, return_url=None) -> str` — pure URL builder; query string `?tier=…[&return=…]`
  - `.render_paywall(app_slug, tier, return_url=None) -> str` — `GET` + returns the HTML body string
  - `.preview_paywall(app_slug, tier, return_url=None, format="html") -> str` — when `format="url"` returns the URL only; otherwise delegates to `render_paywall`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `proxy_rules.SharkAPIError`, `proxy_rules._raise`
- `urllib.parse.urlencode`

## Notes
- The paywall endpoint is **public** — `Authorization` header is only attached when `token` is non-None.
- `paywall_url()` makes no network call; safe for use in templates and redirects.
- `render_paywall()` raises `SharkAPIError` on 400/404 (e.g. unknown `app_slug`, invalid `tier`).
- `tier` is a free-form string here (not `Literal`) because the server evolves the tier set faster than the SDK.
- This is the back-end for the v1.5 dashboard "Preview paywall" pane (Lane H5).
- `format` parameter shadows the built-in but is consistent with the TS SDK; `# noqa: A002` suppresses the lint.
