# Notifications.tsx

**Path:** `admin/src/components/Notifications.tsx`
**Type:** React component
**LOC:** ~150

## Purpose
Notification bell widget—shows count of unread notifications, opens popover with list, mark as read.

## Exports
- `NotificationBell()` (default) — function component

## Features
- **Bell icon** — with red badge showing unread count
- **Popover** — list of recent notifications
- **Mark as read** — individual or all
- **Auto-dismiss** — can auto-close on read

## Hooks used
- `useAPI('/admin/notifications')` — fetch notifications

## Composed by
- TopBar (layout.tsx)

## Notes
- Typically shows system alerts, events, or user actions
