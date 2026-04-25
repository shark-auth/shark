# QuickCreate.tsx

**Path:** `admin/src/components/QuickCreate.tsx`
**Type:** React component (popover)
**LOC:** 79

## Purpose
Topbar "+ Quick create" dropdown menu. Selecting an item navigates to the matching page with `?new=1`, which the destination page detects on mount to auto-open its create flow.

## Exports
- `QuickCreateMenu` — `{open, onClose, setPage, anchorRef}`.

## Props / hooks
- `open` — visibility flag.
- `onClose()` — called on outside-click or Escape.
- `setPage(page, query)` — router setter; receives `{new: '1'}` query alongside the page id.
- `anchorRef` — ref to the topbar trigger button used to position the popover.
- `useRef` for the menu, `useEffect` for the click-outside + escape listeners.

## API calls
- None.

## Composed by
- `admin/src/topbar.tsx` (or equivalent) — the "+ Quick create" button mounts this and passes its `ref`.

## Notes
- Hard-coded `ITEMS` list: New User, Agent, Application, Organization, API Key, Webhook, Role, SSO Connection — each pairs a label with a destination page key and an `Icon` name (looked up in `Icon[it.icon] || Icon.Plus`).
- Positioning is fixed to viewport using the anchor's bounding rect; falls back to `top: 56, right: 16`.
- `fadeIn` 80ms animation is referenced but defined elsewhere (global CSS).
