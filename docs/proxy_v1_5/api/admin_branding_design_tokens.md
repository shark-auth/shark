# Admin API — Branding design tokens

## Purpose

Stores a free-form design-token JSON object on the global branding row so the dashboard can evolve its token tree (colors, typography, spacing, motion) without requiring a schema migration per field. The tokens are returned to any consumer that calls `GET /api/v1/admin/branding` or the hosted-page resolver, letting the paywall and hosted-login surfaces render with the operator's palette + type scale.

See `migration/00025_branding_design_tokens.md` for the underlying schema change that introduced the `design_tokens TEXT` column.

## Route

| Method | Path | Handler symbol |
|---|---|---|
| PATCH | `/api/v1/admin/branding/design-tokens` | `Server.handleSetDesignTokens` |

## Auth required

Admin API key.

## Request shape

```json
{
  "design_tokens": {
    "colors": {
      "primary": "#6366f1",
      "secondary": "#4f46e5",
      "background": "#ffffff",
      "text": "#111827"
    },
    "typography": {
      "font_family": "Inter, system-ui, sans-serif",
      "scale": { "base": "16px", "h1": "2rem", "h2": "1.5rem" }
    },
    "spacing": { "unit": "4px" },
    "motion": { "duration_ms": 150 }
  }
}
```

- `design_tokens` (object, required): any JSON object. Nested maps and arrays are preserved verbatim. An explicit `null` is replaced with `{}` at the handler level so storage never writes a literal `null` blob.

## Response shape

### Success (200)

```json
{
  "data": {
    "branding":      { ...full branding row with all columns... },
    "design_tokens": { ...the same object echoed back... }
  }
}
```

### Error

```json
{ "error": { "code": "invalid_request", "message": "Invalid JSON body" } }
```

## Status codes

- `200 OK` — tokens persisted.
- `400 Bad Request` — bad JSON body.
- `401 Unauthorized` — missing/invalid admin key.
- `500 Internal Server Error` — DB write or read-back failed.

## Side effects

- DB write: `branding.design_tokens` column (global row) is overwritten with the JSON-encoded map. This is a full replace, not a deep merge — callers that want partial updates must GET the current tokens, merge client-side, and PATCH the result.
- Consumption path: `Store.ResolveBranding` returns the design tokens alongside the existing primary/secondary/font columns; the paywall + hosted-login renderers embed them as CSS custom properties.
- Audit log: `branding.design_tokens.set` entry with `TargetType=branding`, `TargetID=global`.

## Frontend hint

A design-tokens editor belongs on the Branding page. A good split: left column is a JSON editor (CodeMirror or Monaco in JSON mode) with schema validation; right column is a live preview pane that renders a mini hosted-page fragment using the current token values. Pair with "Reset to defaults" and "Copy as CSS vars" actions — the server stores whatever shape you send, so the UI can enforce its own schema on the way in without coordinating with the backend.
