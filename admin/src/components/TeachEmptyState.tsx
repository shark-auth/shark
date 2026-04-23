// @ts-nocheck
import React from 'react'
import { Icon } from './shared'

export function TeachEmptyState({ icon, title, description, createLabel, onCreate, cliSnippet }) {
  const [copied, setCopied] = React.useState(false);
  const IconEl = Icon[icon] || Icon.Info;

  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '48px 24px', textAlign: 'center',
    }}>
      <div style={{ maxWidth: 400 }}>
        <IconEl width={24} height={24} style={{ color: 'var(--fg-dim)', marginBottom: 12 }}/>
        <div style={{ fontSize: 14, fontWeight: 600, fontFamily: 'var(--font-display)', color: 'var(--fg)' }}>
          {title}
        </div>
        <div style={{ fontSize: 12, color: 'var(--fg-dim)', lineHeight: 1.6, marginTop: 6 }}>
          {description}
        </div>

        <div style={{ display: 'flex', gap: 10, justifyContent: 'center', marginTop: 16, alignItems: 'center' }}>
          {onCreate && (
            <button className="btn primary sm" onClick={onCreate}>
              <Icon.Plus width={11} height={11}/>{createLabel || 'Create'}
            </button>
          )}
          {onCreate && cliSnippet && (
            <span className="faint" style={{ fontSize: 11 }}>or</span>
          )}
          {cliSnippet && (
            <button
              className="mono"
              onClick={() => { navigator.clipboard?.writeText(cliSnippet); setCopied(true); setTimeout(() => setCopied(false), 1200); }}
              style={{
                display: 'inline-flex', alignItems: 'center', gap: 6,
                padding: '5px 10px', borderRadius: 'var(--radius-sm)',
                background: 'var(--surface-2)', border: '1px solid var(--hairline)',
                fontSize: 11, color: 'var(--fg-muted)', cursor: 'pointer',
              }}
              title="Copy to clipboard"
            >
              <span style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{cliSnippet}</span>
              {copied
                ? <Icon.Check width={10} height={10} style={{ color: 'var(--success)' }}/>
                : <Icon.Copy width={10} height={10} style={{ opacity: 0.5 }}/>}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
