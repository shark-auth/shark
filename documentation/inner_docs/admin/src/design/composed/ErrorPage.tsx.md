# ErrorPage.tsx

**Path:** `admin/src/design/composed/ErrorPage.tsx`
**Type:** React component (composed surface)
**LOC:** 155

## Purpose
Generic full-screen error page primitive — title, message, optional code badge, and a stack of action buttons. First action is rendered as primary; remaining actions are ghost variants.

## Exports
- `ErrorPage` — `{code?, title, message, actions?}`.
- `ErrorPageProps`, `ErrorPageAction` (interfaces).

## Props / hooks
- `code?: string` — short identifier rendered above the title (e.g. `404`).
- `title: string`, `message: string`.
- `actions?: ErrorPageAction[]` — each `{label, href?, onClick?}`. Items with `href` render an anchor wrapping the Button; click-only items render a button.
- No internal state.

## API calls
- None.

## Composed by
- `admin/src/hosted/routes/error.tsx` (`ErrorPage` route wrapper).

## Notes
- Single 56px warning SVG styled with `tokens.color.danger`.
- Card body padding is asymmetric (`space[8]` / `space[6]`) so the surface feels centered on viewports.
- `displayName = 'ErrorPage'` set explicitly.
