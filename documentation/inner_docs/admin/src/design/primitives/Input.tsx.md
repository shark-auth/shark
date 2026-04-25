# Input.tsx

**Path:** `admin/src/design/primitives/Input.tsx`
**Type:** React component (primitive)
**LOC:** 120+

## Purpose
Reusable input field with optional leading icon, trailing adornment, error state, and focus styling.

## Exports
- `Input` (forwardRef) — input element
- `InputProps` (interface)

## Props
- `error?: boolean` — error state styling (red border)
- `leadingIcon?: React.ReactNode` — icon on left
- `trailingAdornment?: React.ReactNode` — icon/element on right
- `disabled?: boolean`
- Standard input HTML attributes

## Features
- **Focus state** — primary color border, subtle shadow
- **Error state** — danger color border
- **Icons** — positioned absolutely, no pointer events
- **Smooth transitions** — border-color, shadow animations
- **Accessible** — aria-invalid on error

## Styling
- Height: 34px
- Background: surface2 color
- Border radius: medium
- Padding: 8px (3px × space unit), adjusted for icons

## Notes
- Uses design system tokens (color, space, motion, type)
- Supports standard HTML input attributes
- Trailing adornment allows pointer events (e.g., visibility toggle)
