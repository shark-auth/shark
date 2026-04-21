// @ts-nocheck
import React from 'react'
import { BrandingVisualsTab } from './branding/visuals_tab'
import { BrandingEmailTab } from './branding/email_tab'
import { BrandingIntegrationsTab } from './branding/integrations_tab'

export function Branding() {
  const [subtab, setSubtab] = React.useState<'visuals' | 'email' | 'integrations'>('visuals')

  return (
    <div style={{ padding: 24, height: '100%', overflow: 'auto' }}>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>Branding</h1>
      <div style={{ display: 'flex', gap: 4, borderBottom: '1px solid var(--hairline)', marginBottom: 20 }}>
        {(['visuals', 'email', 'integrations'] as const).map(t => (
          <button
            key={t}
            onClick={() => setSubtab(t)}
            style={{
              padding: '10px 14px',
              background: subtab === t ? 'var(--surface-2)' : 'transparent',
              borderBottom: subtab === t ? '2px solid var(--accent)' : '2px solid transparent',
              border: 'none',
              cursor: 'pointer',
              textTransform: 'capitalize',
            }}
          >
            {t === 'email' ? 'Email Templates' : t}
          </button>
        ))}
      </div>

      {/* Keep all subtabs mounted so in-progress drafts survive tab switches.
          Hide inactive ones via display:none instead of unmounting. */}
      <div style={{ display: subtab === 'visuals' ? 'block' : 'none' }}>
        <BrandingVisualsTab/>
      </div>
      <div style={{ display: subtab === 'email' ? 'block' : 'none' }}>
        <BrandingEmailTab/>
      </div>
      <div style={{ display: subtab === 'integrations' ? 'block' : 'none' }}>
        <BrandingIntegrationsTab/>
      </div>
    </div>
  )
}
