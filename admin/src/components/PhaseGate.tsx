// @ts-nocheck
import React from 'react'

// Current shipped phase. Anything > CURRENT_PHASE renders gated.
// Phase 6 = proxy + flow builder (shipped). Phase 7 = SDK (in flight).
export const CURRENT_PHASE = 6;

// PhaseGate — wrap a button/control that depends on a phase not yet shipped.
// Renders the children with reduced opacity and disabled cursor, plus a
// title tooltip explaining the gating. Click is suppressed.
//
// Usage:
//   <PhaseGate phase={7}>
//     <button>...</button>
//   </PhaseGate>
//
// If `phase <= CURRENT_PHASE` it just returns children unchanged (no wrapper).
export function PhaseGate({ phase, children, label }) {
  if (!phase || phase <= CURRENT_PHASE) return children;
  const tip = label
    ? `${label} ships in Phase ${phase}`
    : `Available in Phase ${phase}`;
  return (
    <span
      title={tip}
      onClickCapture={(e) => { e.preventDefault(); e.stopPropagation(); }}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 4,
        opacity: 0.45, cursor: 'not-allowed',
        pointerEvents: 'auto',
      }}
    >
      <span style={{ pointerEvents: 'none' }}>{children}</span>
    </span>
  );
}

// PhaseChip — small label rendered alongside gated nav items.
export function PhaseChip({ phase }) {
  if (!phase || phase <= CURRENT_PHASE) return null;
  return (
    <span
      title={`Available in Phase ${phase}`}
      className="mono"
      style={{ fontSize: 9, color: 'var(--fg-faint)' }}
    >
      P{phase}
    </span>
  );
}
