# error.tsx

**Path:** `admin/src/hosted/routes/error.tsx`
**Type:** React route shell
**LOC:** 47

## Purpose
Hosted-login error page route. Wraps the `ErrorPage` design surface and supports two modes: static props (used by the App.tsx 404 fallback) and dynamic props sourced from the URL query string (`?code=...&msg=...`).

## Exports
- `ErrorPage` — `{code?, message?, config?}`.

## Props / hooks
- `code?` (number|string), `message?` — static overrides.
- `config?: HostedConfig` — used to derive the "Back to sign in" link via `config.app.slug`.
- `useState` initialized once from `URLSearchParams(window.location.search)`.

## API calls
- None.

## Composed by
- `admin/src/hosted/App.tsx` — fallback route (`<ErrorPage code={404} message="Page not found"/>`) and `/error` route (dynamic mode).

## Notes
- Static props win over query string when both present.
- "Back to sign in" only rendered when `config.app.slug` is available — otherwise the design's `actions` prop is left undefined and the surface renders without CTAs.
