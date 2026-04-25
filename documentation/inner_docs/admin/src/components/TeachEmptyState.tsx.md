# TeachEmptyState.tsx

**Path:** `admin/src/components/TeachEmptyState.tsx`
**Type:** React component
**LOC:** ~120

## Purpose
Empty state component—shown when list is empty, with icon, message, create CTA, and CLI hint.

## Exports
- `TeachEmptyState({ icon, title, description, createLabel, onCreate, cliSnippet })`

## Props
- `icon: string` — icon name (from Icon library)
- `title: string` — heading
- `description: string` — explanation
- `createLabel?: string` — button text (e.g., "New Agent")
- `onCreate?: () => void` — create button callback
- `cliSnippet?: string` — CLI command to run

## Features
- **Centered layout** — nice visual hierarchy
- **Icon** — large, faded icon from shared library
- **Call to action** — primary button
- **CLI hint** — shows equivalent CLI command

## Composed by
- Multiple pages (Users, Agents, API Keys, Organizations, etc.)

## Example
```
<TeachEmptyState
  icon="Users"
  title="No users yet"
  description="Invite your first user to get started."
  createLabel="Invite User"
  onCreate={() => setCreating(true)}
  cliSnippet="shark users invite --email user@example.com"
/>
```
