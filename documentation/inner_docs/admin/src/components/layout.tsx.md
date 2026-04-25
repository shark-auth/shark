# layout.tsx

**Path:** `admin/src/components/layout.tsx`
**Type:** React components
**LOC:** 500+

## Purpose
Sidebar + TopBar shell components. NAV constant maps page IDs to navigation items.

## Exports
- `NAV` (array) — hierarchical navigation config
- `Sidebar(props)` — left navigation panel
- `TopBar(props)` — top banner with logo, title, buttons

## Sidebar props
- `page: string` — current page
- `setPage: (id, extra?) => void` — navigation callback
- `collapsed: boolean` — sidebar width state
- `setCollapsed: (v) => void` — toggle sidebar
- `devMode: boolean` — show dev-only items
- `showPreview: boolean` — show phase-gated items

## TopBar props
- `page: string` — current page
- `setTweaksOpen: (v) => void` — open tweaks panel
- `onOpenPalette: () => void` — open command palette
- `setPage: (id, extra?) => void` — navigation

## NAV structure
Grouped by category (HUMANS, AGENTS, ACCESS, OPERATIONS, DEVELOPERS, INFRASTRUCTURE, ENTERPRISE):
- Each item has id, label, icon, optional badge, phase gating (ph)
- Some items marked devOnly (dev-inbox)
- Some items marked live (device-flow)

## Sidebar features
- Collapsible (shows glyph icon when collapsed, wordmark when expanded)
- Health indicator (DB status: healthy|unhealthy)
- Version badge
- Keyboard shortcuts visible on navigation items
- Active page highlighted with left border

## TopBar features
- Quick-create menu (QuickCreateMenu)
- Notification bell (NotificationBell)
- Opens command palette (Cmd+K)
- Tweaks panel toggle

## API calls
- `GET /admin/health` — health status, version, DB info

## Composed by
- App.tsx
