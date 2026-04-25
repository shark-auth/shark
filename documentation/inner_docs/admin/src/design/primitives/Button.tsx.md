# Button.tsx

**Path:** `admin/src/design/primitives/Button.tsx`
**Type:** React component (primitive)
**LOC:** 180+

## Purpose
Reusable button component with variants, sizes, loading state, and comprehensive keyboard/mouse interaction.

## Exports
- `Button` (forwardRef) — button element
- `ButtonProps` (interface)
- `ButtonVariant` (type union: primary|ghost|danger|icon)
- `ButtonSize` (type union: sm|md|lg)

## Props
- `variant?: ButtonVariant` — visual style (default: ghost)
- `size?: ButtonSize` — sm(28h)|md(32h)|lg(40h) (default: md)
- `loading?: boolean` — show spinner, disable interaction
- `disabled?: boolean`
- `children?: React.ReactNode`
- Standard button HTML attributes

## Variants
- **primary** — solid colored, high emphasis
- **ghost** — transparent, bordered
- **danger** — red, destructive actions
- **icon** — square, no text padding

## States
- **Hover** — lift animation (translateY(-1px)), shadow
- **Press** — no lift, shadow removed
- **Focus** — focus ring outline
- **Disabled** — opacity 0.5, cursor not-allowed
- **Loading** — spinner, width constraint

## Sizing
- sm: 28px height, 12px font, 10px horizontal padding
- md: 32px height, 13px font, 14px horizontal padding
- lg: 40px height, 14px font, 18px horizontal padding

## Spinner
- Animated SVG rotation (700ms linear)
- Uses currentColor for theming
- Size adapts to button size

## Notes
- Uses tokens for consistent theming (colors, motion, shadows)
- Proper focus management for accessibility
