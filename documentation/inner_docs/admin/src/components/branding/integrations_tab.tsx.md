# integrations_tab.tsx

**Path:** `admin/src/components/branding/integrations_tab.tsx`
**Type:** React component (subtab)
**LOC:** 396

## Purpose
Branding > Integrations subtab. Per-application picker for `integration_mode` (hosted / components / proxy / custom) plus a snippet modal that surfaces framework-specific React install + provider + page-usage code.

## Exports
- `BrandingIntegrationsTab` (named).
- `AppRow`, `SnippetModal` (internal helpers, also live in this file beyond line 200).

## Props / hooks
- props: `{}`.
- State: `apps`, `snippetModal`.
- `useToast()` for update/snippet errors.

## API calls
- GET `/api/v1/admin/apps` → `{applications: [...]}` (also tolerates `data` / `items`).
- PATCH `/api/v1/admin/apps/{id}` (fields: `integration_mode`, `proxy_login_fallback`, `proxy_login_fallback_url`).
- GET `/api/v1/admin/apps/{id}/snippet?framework=react` → `{framework, snippets: [{label, lang, code}]}`.

## Composed by
- `admin/src/components/branding.tsx` (Branding tab router).

## Notes
- Modes: `hosted`, `components`, `proxy`, `custom`. Frameworks: only `react` enabled; vue/svelte/solid/angular reserved disabled options.
- App slug fallback: `app.slug || app.client_id || app.id` because backend doesn't yet stamp slugs.
- Empty state ("No applications registered") prompts the admin to create one under Applications first.
- Hosted-login shortcut keys off `client_id` until slugs ship.
