# branding.ts

**Path:** `sdk/typescript/src/branding.ts`
**Type:** Admin namespace — branding design tokens
**LOC:** 139

## Purpose
Read and write the free-form JSON design-token blob that drives paywall and hosted-login color/typography palette.

## Public API
- `class BrandingClient`
  - `constructor(opts: BrandingClientOptions)`
  - `getBranding(appSlug?: string): Promise<BrandingRow>` — GET `/api/v1/admin/branding[?app_slug=…]`
  - `setBranding(tokens: DesignTokens): Promise<SetBrandingResult>` — PATCH `/api/v1/admin/branding/design-tokens` (full replace; no deep-merge)

## DesignTokens
`Record<string, unknown>` — server stores any JSON object. Recommended shape: `{ colors, typography, spacing, motion }`.

## BrandingRow shape
`id, app_id?, primary_color?, secondary_color?, font_family?, logo_url?, design_tokens, created_at, updated_at` (open index signature for forward-compat).

## SetBrandingResult
`{ branding: BrandingRow, design_tokens: DesignTokens }` — full row plus echoed token blob.

## Constructor options
- `baseUrl: string`
- `adminKey: string` — Bearer token

## Error mapping
- Non-200 → `SharkAPIError` parsed from `{error:{code,message}}` envelope.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- `setBranding` is a full replace — for partial updates do GET → merge client-side → PATCH.
- Passing `null` or `{}` clears the token blob.
- `appSlug` is the path scope (e.g. per-tenant); omit for the global row.
