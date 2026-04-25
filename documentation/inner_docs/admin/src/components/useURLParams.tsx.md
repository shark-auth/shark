# useURLParams.tsx

**Path:** `admin/src/components/useURLParams.tsx`
**Type:** React hooks
**LOC:** ~80

## Purpose
Sync component state with URL query parameters for pagination, filters, search.

## Exports
- `useURLParam(key, defaultValue)` — single param hook
- `useTabParam(defaultValue)` — tab selection hook

## useURLParam behavior
- Reads initial value from URL query
- Updates URL on state change
- Returns `[value, setValue]` tuple

## useTabParam behavior
- Wrapper for `useURLParam('tab', defaultValue)`
- Syncs active tab to URL

## Examples
```
const [search, setSearch] = useURLParam('search', '');
const [filter, setFilter] = useURLParam('role', 'all');
const [tab, setTab] = useTabParam('overview');
```

## Notes
- Uses `window.location.search` and `window.history.replaceState`
- Useful for:
  - Deep linking (`/users?search=john&verified=yes`)
  - Browser back/forward button sync
  - Bookmarkable filtered views
