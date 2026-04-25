# PhaseGate.tsx

**Path:** `admin/src/components/PhaseGate.tsx`
**Type:** React utility
**LOC:** ~50

## Purpose
Feature gating by phase number—some features only available after specific milestone.

## Exports
- `CURRENT_PHASE` (number) — current development phase
- `isGated(phase)` — check if feature is gated

## Constants
- CURRENT_PHASE — typically 3-5 depending on milestone
- Features with `ph: N` in NAV only show if CURRENT_PHASE >= N
- Can be overridden with `showPreview` tweak in App

## Used by
- layout.tsx (NAV filtering)
- App.tsx (phase gating logic)

## Notes
- Allows gradual feature rollout
- Useful for communicating coming soon features
