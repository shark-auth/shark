# hosted-entry.tsx

**Path:** `admin/src/hosted-entry.tsx`
**Type:** Entry point (hosted auth)
**LOC:** 25

## Purpose
Bootstrap hosted authentication flows (separate from admin dashboard)—renders HostedApp with config from window object.

## Features
- **Config injection** — reads `window.__SHARK_HOSTED` config
- **Renders to** — `#hosted-root` DOM element
- **HostedConfig** — app info, branding, auth methods, OAuth config

## HostedConfig interface
```typescript
{
  app: { slug, name, logo_url }
  branding: { primary_color, secondary_color, font_family, logo_url }
  authMethods: Array<'password'|'magic_link'|'passkey'|'oauth'>
  oauthProviders: Array<{ id, name, iconUrl }>
  oauth: { client_id, redirect_uri, state, scope }
}
```

## Used by
- Hosted auth pages (login, signup, MFA, etc.)
- Typically loaded via iframe or separate domain

## Notes
- Separate entry point from admin dashboard (main.tsx)
- Config passed from server via global window var
- Error handling: shows message if config missing
