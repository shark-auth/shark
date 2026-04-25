# useKeyboardShortcuts.tsx

**Path:** `admin/src/components/useKeyboardShortcuts.tsx`
**Type:** React hooks & components
**LOC:** ~200

## Purpose
Global keyboard shortcut handling. Shortcuts: /=search, r=refresh, n=new, ?=help, Cmd+K=command palette.

## Exports
- `useKeyboardShortcuts(setPage)` — setup shortcuts, returns cheatsheet state
- `usePageActions({ onRefresh?, onNew? })` — per-page refresh/new actions
- `KeyboardCheatsheet({ onClose })` — help modal component

## Global shortcuts
- `/` — focus search in current page
- `r` — refresh current data
- `n` — new item (create modal)
- `?` — open keyboard cheatsheet modal
- `Cmd+K` or `Ctrl+K` — open command palette (handled in App.tsx)

## useKeyboardShortcuts hook
- Returns `{ cheatsheetOpen, setCheatsheetOpen }`
- Takes `setPage` callback for navigation
- Listens globally on window

## usePageActions hook
- Typical page adoption:
  ```
  usePageActions({
    onRefresh: refresh,
    onNew: () => setCreating(true),
  });
  ```
- Handles r/n shortcuts at page level

## KeyboardCheatsheet component
- Modal listing all available shortcuts
- Shows command palette trigger
- Pages can have custom shortcuts shown

## Notes
- Shortcuts only active when appropriate context (not in input field)
- Prevents default browser behavior (e.g., Cmd+K)
