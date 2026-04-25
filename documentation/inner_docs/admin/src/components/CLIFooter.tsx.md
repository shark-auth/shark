# CLIFooter.tsx

**Path:** `admin/src/components/CLIFooter.tsx`
**Type:** React component
**LOC:** ~80

## Purpose
Footer note showing equivalent CLI command for current page action.

## Exports
- `CLIFooter({ snippet })` (default) — function component

## Props
- `snippet: string` — shell command example (e.g., "shark users list")

## Features
- **Monospace display** — shows the CLI command
- **Copy button** — copy command to clipboard
- **Faint styling** — secondary info appearance

## Example
Used in Users page:
```
<CLIFooter snippet="shark users list --limit 25 --search 'john'" />
```

## Composed by
- Pages that have CLI equivalents

## Notes
- Helps users understand CLI alternatives to UI actions
