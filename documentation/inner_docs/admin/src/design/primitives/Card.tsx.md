# Card.tsx

**Path:** `admin/src/design/primitives/Card.tsx`
**Type:** React component (primitive)
**LOC:** 50

## Purpose
Simple card container with optional header, border, and rounded corners.

## Exports
- `Card` — function component
- `CardProps` (interface)

## Props
- `header?: React.ReactNode` — header content
- `children?: React.ReactNode` — body content
- `style?: React.CSSProperties` — card-level styles
- `bodyStyle?: React.CSSProperties` — padding/styling of body
- `className?: string` — CSS class

## Structure
- Outer div: bordered, rounded, with background
- Header (if provided): separated by bottom border, semi-bold text
- Body: padded content area

## Default styling
- Background: surface2 color
- Border: 1px hairline
- Border radius: large
- Padding: 24px (6px × space unit)

## Notes
- Minimal component, primarily layout and spacing
- Header is optional; body is required
