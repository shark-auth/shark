# Tabs.tsx

**Path:** `admin/src/design/primitives/Tabs.tsx`
**Type:** React primitive
**LOC:** 139

## Purpose
Accessible horizontal tab strip with optional badge counts. Implements full WAI-ARIA `tablist` keyboard semantics (Arrow keys / Home / End) and renders an optional `tabpanel` body for the active tab.

## Exports
- `Tabs` — `{tabs, activeTab, onTabChange, children?, style?, listStyle?}`.
- `TabsProps`, `Tab` (interfaces).

## Props / hooks
- `tabs: Tab[]` — each `{id, label, badge?, disabled?}`.
- `activeTab: string`, `onTabChange(id)` — controlled state.
- `children?` — rendered inside the active tab panel; if omitted the component is just the strip.
- `listStyle?` — overrides on the tablist container.

## API calls
- None.

## Composed by
- Admin pages with subtab navigation (e.g. branding / settings sections, when migrated to this primitive).

## Notes
- Keyboard handling: ArrowRight/Left wrap; Home jumps to first enabled, End to last enabled.
- Disabled tabs skipped in keyboard navigation (`enabledTabs` filter).
- Active tab gets a 2px primary underline (`marginBottom: -1` aligns with the bottom hairline).
- Tab panel itself receives `tabIndex={0}` so screen readers can land on it.
- `displayName = 'Tabs'` set explicitly.
