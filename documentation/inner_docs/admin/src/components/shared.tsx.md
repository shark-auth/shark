# shared.tsx

**Path:** `admin/src/components/shared.tsx`
**Type:** React utility components & icons
**LOC:** 300+

## Purpose
Shared UI primitives and icon library exported to all components.

## Exports
### Icons
- `Icon.<name>` — 40+ SVG icon components (Home, Users, Session, Org, Agent, Lock, Shield, Key, Audit, Webhook, Settings, etc.)

### Components
- `SharkIcon(size?)` — branded icon
- `SharkFullLogo(width?)` — full logo image
- `Avatar({ name, email, size?, agent? })` — user avatar with initials
- `CopyField({ value, mono?, truncate? })` — copyable text field with visual feedback
- `Sparkline({ data, height?, color? })` — mini line chart
- `Donut({ segments, size?, thickness? })` — donut/pie chart
- `Kbd({ keys })` — keyboard shortcut badge

### Utils
- `hashColor(str)` — deterministic color from string (oklch color space)

## Asset imports
- `sharkyGlyphPng` — glyph icon
- `sharkyWordmarkPng` — text logo
- `sharkyFullPng` — full logo
- `sharkyIconPng` — square icon

## Usage examples
- Avatar: `<Avatar name="John" email="john@..." />`
- CopyField: `<CopyField value="sk_live_..." />`
- Icon: `<Icon.Users width={14} height={14} />`

## Notes
- Icons use SVG with `currentColor` for easy theming
- Sparkline and Donut use Canvas-less SVG rendering
- hashColor uses oklch color space for consistent hue distribution
