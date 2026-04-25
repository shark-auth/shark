# CommandPalette.tsx

**Path:** `admin/src/components/CommandPalette.tsx`
**Type:** React component (modal)
**LOC:** ~250

## Purpose
Command palette modal (Cmd+K)—fuzzy search navigation, quick actions, recent pages, theme switching.

## Exports
- `CommandPalette({ open, onClose, setPage })` (default) — function component

## Props
- `open: boolean` — modal visibility
- `onClose: () => void` — close callback
- `setPage: (id, extra?) => void` — navigate

## Features
- **Search input** — live fuzzy filter over commands/pages
- **Groups** — navigation, actions, settings
- **Navigation** — link to all main pages
- **Actions** — refresh, export, toggle theme (light/dark)
- **Recent** — last visited pages (from history state)
- **Keyboard** — arrow keys to select, enter to execute, esc to close

## Composed by
- App.tsx

## Notes
- Modal is dismissible on escape or clicking backdrop
- Search highlights matching text
