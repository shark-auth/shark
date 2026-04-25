# get_started.tsx

**Path:** `admin/src/components/get_started.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)

## Purpose
First-experience onboarding checklist. Shows the two SharkAuth integration paths (Proxy + hosted UI vs SDK) as parallel task lists, switchable at any time, with auto-verification probing real backend state.

## Exports
- `GetStarted()` — function component

## Layout
- **Toolbar** — segmented tab strip (Proxy + hosted UI / SDK), progress fraction `done/total`, 4px progress bar, Recheck button, Skip-to-dashboard
- **Body** — `<table class="tbl">` with sticky head. Columns: index, Task title + summary, Status dot + label (+ "auto" tag), Goes-to (mono path), chevron
- **Drawer** — 420px right-side fixed, hairline left border, square. Contents: task id (mono), title, status dot, "What this does" prose, deep-link card (Open → setPage), CLI snippet, code snippet, Copy buttons. Footer: Mark done / Skip / Reset / Go.

## Path A — Proxy + hosted UI (6 tasks)
1. Configure proxy upstream
2. Add a protect rule
3. Pick login methods
4. Set post-login redirect
5. Brand the hosted login page
6. Test the proxy URL

## Path B — SDK (6 tasks)
1. Create an Application
2. Install SDK package
3. Mount SharkProvider
4. Wire Sign-in button
5. Mount callback route
6. Read the session

## Auto-verify
Probes `/admin/proxy/status`, `/admin/proxy/rules`, `/admin/apps`, `/admin/branding`. Pending tasks with matching `autoCheck` flag flip to `done` with "auto" tag.

## Persistence
`shark_get_started_state_v1` localStorage — active tab + per-task status (pending/done/skipped).

## API calls
- `GET /admin/proxy/status`
- `GET /admin/proxy/rules`
- `GET /admin/apps`
- `GET /admin/branding`

## Composed by
- `App.tsx` — route `get-started`

## Visual contract (per .impeccable.md v3)
- Monochrome — status dots only carry color (`--success`/`--warn`/`--danger`/`--fg-faint`)
- 13px base, hairline borders, table rhythm matches users.tsx exactly
- 10/16px toolbar padding, 11px uppercase column headers, surface-0 head
- Drawer (right-side fixed) NOT modal
- No path cards, no hero, no recommended badge, no emoji, no colored tabs, no YAML

## Notes
- No "session vs JWT" framing — tasks are about wiring auth, not picking modes
- Cookie + JWT both always on; SDK path uses both transparently
