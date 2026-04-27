// @ts-nocheck
import React from 'react'
import { BrandingVisualsTab } from './branding/visuals_tab'
import { BrandingEmailTab } from './branding/email_tab'
import { BrandingIntegrationsTab } from './branding/integrations_tab'

const TABS = [
  ['visuals', 'Visuals'],
  ['email', 'Email templates'],
  ['integrations', 'Integrations'],
] as const

export function Branding() {
  const [tab, setTab] = React.useState<string>('visuals')

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Tab bar */}
      <div
        style={{
          display: 'flex',
          gap: 0,
          padding: '0 20px',
          borderBottom: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          flexShrink: 0,
        }}
      >
        {TABS.map(([v, l]) => (
          <button
            key={v}
            onClick={() => setTab(v)}
            style={{
              padding: '10px 12px 8px',
              fontSize: 12,
              color: tab === v ? 'var(--fg)' : 'var(--fg-muted)',
              borderBottom: tab === v ? '2px solid var(--fg)' : '2px solid transparent',
              fontWeight: tab === v ? 500 : 400,
              background: 'none',
              border: 'none',
              cursor: 'pointer',
            }}
          >
            {l}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 24 }}>
        {tab === 'visuals' && <BrandingVisualsTab />}
        {tab === 'email' && <BrandingEmailTab />}
        {tab === 'integrations' && <BrandingIntegrationsTab />}
      </div>
    </div>
  )
}

export default Branding
