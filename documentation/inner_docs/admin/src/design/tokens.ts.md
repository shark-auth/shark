# tokens.ts

**Path:** `admin/src/design/tokens.ts`
**Type:** Design token definitions
**LOC:** ~300

## Purpose
Centralized design system tokens—colors, spacing, typography, shadows, motion, border radius.

## Exports
- `tokens` (object) — design system constants

## Token structure
```
{
  color: { primary, danger, success, warning, ... },
  space: [0, 4, 8, 12, 16, 24, 32, ...], // px values
  type: { body, display, mono; size, weight },
  shadow: { sm, md, lg },
  radius: { sm, md, lg },
  motion: { fast, normal, slow },
}
```

## Usage
All primitives reference tokens for consistency:
```
color: tokens.color.primary
padding: tokens.space[4]
fontSize: tokens.type.size.base
borderRadius: tokens.radius.lg
```

## Notes
- Single source of truth for visual consistency
- Supports theming by swapping token values
- Type-safe with TypeScript
