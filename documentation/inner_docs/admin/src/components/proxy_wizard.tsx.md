# proxy_wizard.tsx

**Path:** `admin/src/components/proxy_wizard.tsx`
**Type:** React component
**LOC:** 314

## Purpose
Onboarding wizard for the proxy gateway. Two-step flow (upstream URL → path/methods/rule kind) that creates an `application` (when needed) and posts initial proxy rules — including auto-bypass rules for static assets so Next.js / Vite frontends don't break.

## Exports
- `ProxyWizard` (named) — `{appId?, onComplete?, autofocus?}`.
- Internal `StepHeader`, `isValidURL`.

## Props / hooks
- `appId` — when present, skips Step 1 (upstream creation) and just adds rules to the existing app.
- `onComplete({ upstream, path, appId })` — fired after launch.
- `autofocus` — focuses the upstream input on mount.
- State: `isLocal`, `upstream`, `path`, `methods` (Set of HTTP verbs), `ruleKind`, `step`, `busy`, `error`, `launched`.
- `useToast()` (defensive wrapper since some hosts mount without provider).

## API calls
- POST `/api/v1/admin/apps` (when no `appId`, creates with `integration_mode: 'proxy'`).
- POST `/api/v1/admin/proxy/rules/db` (multiple): `Static Assets Bypass` for `/_next/*` (priority 100), `Common Assets Bypass` for `/*.{css,js,png,jpg,svg,woff2}` (priority 90), and the user's `Main Route Protection` rule.

## Composed by
- Empty-state on the proxy page (`proxy_rules.tsx` / proxy dashboard) when no rules exist.

## Notes
- "Champion Tier — The Frontend First Rule Pattern" auto-injects asset bypasses when the user accepts the default `/*` path, preventing the most common dev-loop foot-gun (proxy intercepting static assets).
- `isLocal` toggle defaults true and auto-flips when the URL contains `localhost` / `127.0.0.1`.
- Rule kinds: `require` (authenticated), `anonymous` (allow), `optional`.
- HTTP verb Set: when all 5 selected, the payload sends `methods: []` (server treats empty as "all").
