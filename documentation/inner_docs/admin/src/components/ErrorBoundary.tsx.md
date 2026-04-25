# ErrorBoundary.tsx

**Path:** `admin/src/components/ErrorBoundary.tsx`

React class component that wraps the entire admin SPA in `main.tsx`. Catches render-phase errors that hooks cannot intercept — without it, an unhandled exception blanks the whole app with no recovery UI.

## Exports

- `ErrorBoundary` — named + default export. React class component.

## State

| Key | Type | Purpose |
|-----|------|---------|
| `error` | `Error \| null` | Caught error |
| `info` | `React.ErrorInfo \| null` | Component stack from `componentDidCatch` |
| `expanded` | `boolean` | Toggle for the stack-trace details pane |

## Methods

| Method | Behaviour |
|--------|-----------|
| `getDerivedStateFromError(error)` | Sets `state.error`; triggers error UI |
| `componentDidCatch(error, info)` | Stores `info`; `console.error`s the pair |
| `reset()` | Clears state — stays on current page |
| `reload()` | `window.location.reload()` |

## Error UI

Monochrome inline styles using CSS vars (`--bg`, `--fg`, `--hairline-strong`, etc.) so it renders correctly even if the stylesheet has not loaded. Shows:
- Fixed descriptive header ("Something broke")
- **Reload** / **Try again** / **Show/Hide details** buttons
- Collapsible `<pre>` with `error.stack` + `info.componentStack`

## Mount point

`admin/src/main.tsx` — wraps `<App />`:
```tsx
<ErrorBoundary><App /></ErrorBoundary>
```

## Notes

- Uses `// @ts-nocheck` (project-wide pattern for admin components).
- Does not log to any remote error tracker — extend `componentDidCatch` to add Sentry/etc.
