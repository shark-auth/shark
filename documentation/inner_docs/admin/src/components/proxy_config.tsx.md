# proxy_config.tsx

**Path:** `admin/src/components/proxy_config.tsx`
**Type:** React component (page)
**LOC:** ~500

## Purpose
Proxy configuration—reverse proxy setup for protecting upstream services without app changes.

## Exports
- `Proxy()` (default) — function component

## Tabs
- **Routes** — protected routes, path patterns, backend URLs
- **Settings** — CORS, redirect behavior, rate limiting, auth mode
- **Status** — active proxies, request stats, health checks
- **Wizard** — step-by-step setup for new proxies

## Features
- **Route creation** — `/api/*` → `https://backend/api/*`, with auth required
- **CORS config** — allowed origins, headers, methods
- **Rate limiting** — requests/second per IP or user
- **Auth modes** — require login, specific roles, RBAC rules
- **SSL/TLS** — upstream certificate validation
- **Logging** — request/response logging for debugging

## Hooks used
- `useAPI('/admin/proxy/config')` — fetch config
- `useToast()` — feedback

## State
- `tab` — current tab
- `routes` — defined routes
- `settings` — global proxy settings

## API calls
- `GET /api/v1/admin/proxy/status` — health check (404 if disabled)
- `GET /api/v1/admin/proxy/config` — fetch configuration
- `POST /api/v1/admin/proxy/routes` — create route
- `PATCH /api/v1/admin/proxy/settings` — update settings

## Composed by
- App.tsx

## Notes
- Zero-code auth integration for any service
- Can be combined with branding for seamless UX
- Transparent to upstream service (no code changes needed)
