# Coming-soon placeholders (v0.1 launch)

For the v0.1 launch, three dashboard surfaces ship as placeholders to keep the
product surface honest — the nav entries remain visible so users know what's
coming, but they route to a "Coming in v0.2" card instead of half-finished
implementations.

## Surfaces

| Surface | Nav label | Component | Real impl preserved? |
|---------|-----------|-----------|----------------------|
| Proxy | "Proxy" | `proxy_config.tsx` | ✅ behind `VITE_FEATURE_PROXY=true`, real impl is `proxy_config_real.tsx` |
| Compliance → Exporting logs | "Exporting logs" | `compliance.tsx` | ❌ git history only |
| Branding | "Branding" | `branding.tsx` | ❌ git history only |

## Enabling the real Proxy implementation

The proxy is feature-flagged because it carries non-trivial setup (DPoP rebinding,
upstream config, paywall rules). Self-hosted users can enable it at build time:

```bash
VITE_FEATURE_PROXY=true pnpm --filter admin build
```

The conditional lives in `admin/src/components/App.tsx`:

```tsx
import { Proxy as ProxyPlaceholder } from './proxy_config'
import { Proxy as ProxyReal } from './proxy_config_real'
const Proxy = (import.meta as any).env?.VITE_FEATURE_PROXY === 'true' ? ProxyReal : ProxyPlaceholder
```

## What still works while these surfaces are placeholders

- **Compliance / Exporting logs** — the placeholder hints users at the existing
  `/audit` page and `/api/v1/audit` endpoints. Audit log query, filter, and SSE
  live stream all ship in v0.1.
- **Branding** — there is no v0.1 substitute. Users get the default monochrome
  shell. Custom branding ships in v0.2.
- **Proxy** — self-hosted users can opt in via the env flag above. Hosted users
  see the placeholder.

## Shared placeholder component

All three surfaces use `admin/src/components/coming_soon.tsx`. The component
takes `title`, `message`, optional `hint`, and an optional `githubUrl` for the
"Track this on GitHub" link.

## Smoke coverage

`tests/smoke/test_w17_coming_soon_routes.py` verifies the backend endpoints
referenced by the placeholders (audit, agents) still respond. UI mounting is
verified visually during release rehearsal.
