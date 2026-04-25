# toast.tsx

**Path:** `admin/src/components/toast.tsx`
**Type:** Toast notification system
**LOC:** 106

## Purpose
Global toast notification provider and hook. Supports success|error|info|warn|undo with auto-dismiss and countdown.

## Exports
- `ToastProvider({ children })` — context provider
- `ToastContext` — React Context
- `useToast()` — hook to access toast API

## Toast API
- `toast.success(msg)` — 5s auto-dismiss
- `toast.error(msg)` — 8s auto-dismiss
- `toast.info(msg)` — 5s auto-dismiss
- `toast.warn(msg)` — 5s auto-dismiss
- `toast.undo(msg, onAction)` — undo action with countdown, returns ID
- `toast.cancel(id)` — cancel undo toast before action fires

## Undo example
```
const id = toast.undo('User deleted', () => API.put('/users/' + id + '/restore'));
```

## Features
- Max 3 toasts on screen at once (FIFO)
- Countdown timer for undo actions
- Styled by type (success=green, error=red, warn=orange, info=gray, undo=surface)
- Fixed position (bottom right)
- Animation on enter (`slideIn` keyframe)

## Notes
- Uses `// @ts-nocheck`
- Timers stored in ref to prevent leaks
