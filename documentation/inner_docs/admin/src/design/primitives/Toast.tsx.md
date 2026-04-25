# Toast.tsx

**Path:** `admin/src/design/primitives/Toast.tsx`
**Type:** React primitive (context + provider + view)
**LOC:** 224

## Purpose
Toast notification system — provides a context with `info` / `success` / `warn` / `danger` methods, plus per-toast view that renders an icon, message, and dismiss button. Includes a slide-in keyframe animation.

## Exports
- `ToastContext` — React context value `{info, success, warn, danger, dismiss}`.
- `ToastProvider` (later in file) — installs context + viewport.
- `useToast()` hook (later in file).
- Types: `ToastVariant`, `ToastItem`, `ToastContextValue`.

## Props / hooks
- `ToastItemView` — `{item, onDismiss}`. `aria-live="polite"`, `role="status"`.
- Variant color tables map to oklch tones (info / success 160h / warn 85h / danger 25h).
- Provider internally manages a list of `ToastItem`s with auto-dismiss timers (continued past line 150).

## API calls
- None.

## Composed by
- App root (`hosted-entry.tsx`, `App.tsx`) wraps children in `<ToastProvider>`.
- All hosted route shells call `useToast()` to surface success/error.

## Notes
- Variant SVG icons inlined per-variant for accessibility (`aria-hidden`).
- Slide-in animation injected via static keyframes string (`shark-toast-slide-in`) — needs to be mounted into `document.head` by the provider.
- Min/max widths 260/360px keep notifications consistent with the dashboard surface widths.
