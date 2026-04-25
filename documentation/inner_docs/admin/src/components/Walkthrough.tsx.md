# Walkthrough.tsx

**Path:** `admin/src/components/Walkthrough.tsx`
**Type:** React component (modal)
**LOC:** ~250

## Purpose
Onboarding walkthrough modal—step-by-step tour of key features for first-time users.

## Exports
- `Walkthrough({ onComplete })` (default) — function component

## Props
- `onComplete: () => void` — callback when walkthrough finished

## Features
- **Sequential steps** — overview → users → sessions → configs → finish
- **Each step has** — title, description, diagram, navigation buttons (next/skip)
- **Skip option** — dismiss walkthrough at any time
- **Progress** — step indicator

## Triggers
- Shown on app load if first time (`shark_admin_onboarded` === '1' AND `shark_walkthrough_seen` not set)

## Composed by
- App.tsx

## Notes
- User can disable walkthrough via localStorage flag
- Often shows on cold start for new orgs
