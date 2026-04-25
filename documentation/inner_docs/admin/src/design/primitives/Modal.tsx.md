# Modal.tsx

**Path:** `admin/src/design/primitives/Modal.tsx`
**Type:** React primitive
**LOC:** 187

## Purpose
Accessible modal dialog primitive — renders a backdrop + dialog with focus trap, Escape-to-close, body-scroll lock, and an optional default header (title + close button). Caller controls visibility via `open` + `onClose`.

## Exports
- `Modal` — `{open, onClose, titleId?, title?, children?, style?, maxWidth?}`.
- `ModalProps` (interface).
- Internal `useFocusTrap` hook (not exported — could be promoted later).

## Props / hooks
- `open` — visibility flag.
- `onClose()` — fired on Escape, backdrop click (lower in file), and close button.
- `titleId?` (default `shark-modal-title`) — `aria-labelledby` target.
- `title?` — when present, rendered in default header with close button.
- `maxWidth?` (default 480).
- `useFocusTrap` cycles Tab/Shift+Tab through focusable descendants and focuses the first one on mount.

## API calls
- None.

## Composed by
- Multiple admin pages (e.g. revoke-consent dialog, snippet modal in branding integrations).

## Notes
- Body scroll lock saves and restores `document.body.style.overflow`.
- Focus trap selector: `a[href]`, enabled `button`/`input`/`select`/`textarea`, and any `[tabindex]` ≥ 0.
- Backdrop uses `oklch(0% 0 0 / 60%)` instead of an rgba so it tracks the design-system color space.
- z-index from `tokens.zIndex.modal`.
