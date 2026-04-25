# proxy_config.tsx

**Path:** `admin/src/components/proxy_config.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)

## Purpose
Proxy management: lifecycle toggle, status metrics, topology, rules CRUD, simulate. Replaces both prior `proxy_config.tsx` AND `proxy_wizard.tsx` chrome. Wizard component still loaded only when lifecycle is `missing`.

## Exports
- `Proxy()` — function component

## Sections
- **Header** — title, rule count chip, last-error inline, Simulate button, New rule button, **Power toggle**
- **Power toggle** — small square button (28px tall, 4px radius, hairline-strong border, surface-1 fill). Label flips Start/Stop. Leading 7px dot — `--success` running, `--warn` reloading, `--fg-dim` stopped. NO pulse, NO color fill.
- **Status strip** — single hairline-bordered row of 8 monochrome tiles: state, listeners, rules, uptime, breaker, failures, cache, last sim. Color only on state-tile leading dot.
- **Topology** — per-app `public_domain → protected_url` rows w/ small Reload button
- **Rules table** — sticky uppercase header, 7px row padding, click → drawer, inline square slide-toggle for `enabled`, disabled rows fade 0.55
- **Rule editor drawer** — 420px right-side fixed, square, hairline left border. Editable: app, public_domain, protected_url, pattern, methods, priority, require, allow_override, tier, scopes, enabled, m2m. Save/Delete/Cancel. Esc + backdrop close.
- **Simulate drawer** — 440px. Method + path + identity JSON; result panel: decision dot, reason, matched rule, injected headers table, eval microseconds (bubbles into Last sim tile).

## Removed (vs prior)
- YAML import drawer entirely (no Import button, no `ImportModal`, no `/admin/proxy/rules/import` UI)
- All centered modals (rule editor, delete confirm, import) — replaced by drawers + native confirm for delete
- Colored pulsing power toggle
- Loud `LifecycleBar` chip strip
- Bulky colored status chip line

## API calls (preserved contract)
- `GET/POST /admin/proxy/lifecycle`
- `GET /admin/proxy/status`
- `GET/POST/PATCH/DELETE /admin/proxy/rules/db`
- `POST /admin/proxy/simulate`

## Composed by
- `App.tsx` — proxy route
- `ProxyWizard` still rendered when `lifecycle.missing` (initial setup)

## Visual contract (per .impeccable.md v3)
- Monochrome — status dots only carry color
- Radii: 5px outer, 4px inputs/buttons
- 13px base, hairline borders, 7-10px row padding
- All editor surfaces are right-side drawers, square, hairline left border

## Notes
- Polls lifecycle + status every 5s
- ProxyWizard.tsx still imported (used by get_started + applications) but not by main proxy page chrome
