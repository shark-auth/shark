# branding.py

**Path:** `sdk/python/shark_auth/branding.py`
**Module:** `shark_auth.branding`
**LOC:** 67

## Purpose
Admin client for the global branding row — fetches the current row and writes design tokens used by the paywall and any branded UI.

## Public API
- `class BrandingClient`
  - `__init__(base_url, token, *, session=None)`
  - `.get_branding(app_slug=None) -> dict` — `GET /api/v1/admin/branding`
  - `.set_branding(app_slug=None, tokens=None) -> dict` — `PATCH /api/v1/admin/branding/design-tokens`; body `{design_tokens: tokens or {}}`; returns `{branding, design_tokens}`

## Constructor params
- `base_url: str` — required
- `token: str` — admin API key (`sk_live_…`)
- `session: object | None` — optional shared `requests.Session`

## Internal dependencies
- `_http.new_session`, `_http.request`
- `proxy_rules._raise` — admin error envelope unwrapper

## Notes
- `app_slug` is accepted for API symmetry with the TS SDK but is **ignored server-side today** — there is exactly one global branding row.
- `tokens` is a free-form JSON object of any depth; passing `{}` (or omitting) clears the design tokens.
- Returns plain `dict` — no model wrapping. Callers usually consume `result["design_tokens"]`.
- Both methods route non-2xx through `_raise()` → `SharkAPIError`.
- This is the back-end for the v1.5 dashboard "Design tokens" editor (Lane H4).
