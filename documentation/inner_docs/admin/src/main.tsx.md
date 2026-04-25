# main.tsx

**Path:** `admin/src/main.tsx`
**Type:** Entry point
**LOC:** 11

## Purpose
Bootstrap React app with Toast provider and render App component into root DOM element.

## Exports
- None (side effects only)

## Structure
- Imports React, ReactDOM
- Wraps `App` with `ToastProvider`
- Mounts to `#root` element in HTML

## Dependencies
- `./components/App` — main admin dashboard component
- `./components/toast` — global toast notification provider
- `./styles.css` — global styles

## Notes
- Uses `@ts-nocheck` to skip type checking for this bootstrap file
- Standard Vite + React entry pattern
