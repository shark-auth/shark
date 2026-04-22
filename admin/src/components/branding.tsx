// @ts-nocheck
import React from 'react'
import { BrandingVisualsTab } from './branding/visuals_tab'
import { BrandingEmailTab } from './branding/email_tab'
import { BrandingIntegrationsTab } from './branding/integrations_tab'

export function Branding() {
  const [subtab, setSubtab] = React.useState<'visuals' | 'email' | 'integrations'>('visuals')

  return (
    <div style={{ padding: '16px 20px', height: '100%', overflow: 'auto', background: 'var(--bg)' }}>
      <div style={{ marginBottom: 16 }}>
        <h1 style={{ 
          fontFamily: 'Hanken Grotesk, var(--font-sans)',
          fontSize: 18, 
          fontWeight: 600,
          letterSpacing: '-0.01em',
          margin: 0,
          color: 'var(--fg)'
        }}>
          Branding
        </h1>
        <p className="faint" style={{ margin: '2px 0 0', fontSize: 11, fontWeight: 400 }}>
          Precision identity & communication control.
        </p>
      </div>
      
      <div style={{ 
        display: 'flex', 
        gap: 0, 
        borderBottom: '1px solid var(--hairline)', 
        marginBottom: 20,
        padding: '0'
      }}>
        {([
          { key: 'visuals', label: 'Visual Identity' },
          { key: 'email', label: 'Email Communication' },
          { key: 'integrations', label: 'Integrations' },
        ] as const).map(t => {
          const isSel = subtab === t.key
          return (
            <button
              key={t.key}
              onClick={() => setSubtab(t.key)}
              style={{
                padding: '6px 14px',
                height: 32,
                fontSize: 11.5,
                fontWeight: isSel ? 600 : 400,
                color: isSel ? 'var(--fg)' : 'var(--fg-dim)',
                background: 'transparent',
                borderBottom: isSel ? '2px solid var(--fg)' : '2px solid transparent',
                marginBottom: -1,
                cursor: 'pointer',
                transition: 'all 60ms ease-out',
                display: 'flex',
                alignItems: 'center',
                borderRadius: 0,
                border: 'none',
                outline: 'none'
              }}
            >
              {t.label}
            </button>
          )
        })}
      </div>

      <div style={{ marginTop: 4 }}>
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
    </div>
  )
}
