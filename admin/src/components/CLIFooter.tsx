// @ts-nocheck
import React from 'react'
import { Icon } from './shared'

export function CLIFooter({ command }) {
  const [copied, setCopied] = React.useState(false);
  return (
    <div style={{
      borderTop: '1px solid var(--hairline)',
      padding: '8px 12px',
      background: 'var(--surface-2)',
      display: 'flex', alignItems: 'center', gap: 8,
    }}>
      <Icon.Terminal width={11} height={11} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
      <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', flexShrink: 0 }}>CLI</span>
      <span style={{
        flex: 1, fontSize: 11, fontFamily: 'var(--font-mono)', color: 'var(--fg-muted)',
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
      }}>{command}</span>
      <button
        className="btn ghost icon sm"
        onClick={() => { navigator.clipboard?.writeText(command); setCopied(true); setTimeout(() => setCopied(false), 900); }}
        title="Copy command"
      >
        {copied
          ? <Icon.Check width={10} height={10} style={{ color: 'var(--success)' }}/>
          : <Icon.Copy width={10} height={10}/>}
      </button>
    </div>
  );
}
