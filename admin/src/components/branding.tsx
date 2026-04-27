// @ts-nocheck
import React from 'react'

// Branding sub-tabs (visuals / email / integrations) are implemented under
// ./branding/* but NOT battle-tested for v0.1. Keep stub until v0.2.
// Re-enable by importing the tabbed container from git history at
// commit b482cf9 (Wave C admin polish, 2026-04-27).

export function Branding() {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      height: '100%', textAlign: 'center', padding: 40,
    }}>
      <div style={{ maxWidth: 460, border: '1px solid var(--hairline-strong)', padding: '28px 32px', background: 'var(--surface-1)' }}>
        <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.12em', color: 'var(--fg-muted)', fontWeight: 600 }}>Coming soon</div>
        <div style={{ fontSize: 17, fontWeight: 600, color: 'var(--fg)', marginTop: 10 }}>Branding</div>
        <div style={{ marginTop: 12, fontSize: 13, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
          Logo, colors, typography, and email-template styling for hosted login pages and customer-facing emails.
          Lands in v0.2 once the sub-tabs (visuals / email / integrations) are battle-tested.
        </div>
      </div>
    </div>
  )
}

export default Branding
