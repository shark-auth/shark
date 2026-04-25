// @ts-nocheck
import React from 'react'

// React only catches render-phase errors via class components — hooks have no
// equivalent. Without this boundary, any unhandled exception in a child
// component blanks the whole admin SPA with no recovery UI. Mount once at
// the root in main.tsx.
export class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { error: null, info: null, expanded: false }
  }
  static getDerivedStateFromError(error) {
    return { error }
  }
  componentDidCatch(error, info) {
    this.setState({ info })
    if (typeof console !== 'undefined' && console.error) {
      console.error('[ErrorBoundary]', error, info)
    }
  }
  reset = () => this.setState({ error: null, info: null, expanded: false })
  reload = () => window.location.reload()
  render() {
    if (!this.state.error) return this.props.children
    const { error, info, expanded } = this.state
    return (
      <div style={{
        minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: 40, background: 'var(--bg, #0a0a0a)', color: 'var(--fg, #e4e4e4)',
        fontFamily: 'system-ui, sans-serif',
      }}>
        <div style={{ maxWidth: 560, width: '100%', border: '1px solid var(--hairline-strong, #333)', padding: '32px 36px' }}>
          <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.12em', color: 'var(--fg-muted, #888)', fontWeight: 600 }}>
            Something broke
          </div>
          <div style={{ fontSize: 18, fontWeight: 600, marginTop: 10 }}>
            The admin UI hit an unexpected error.
          </div>
          <div style={{ fontSize: 13, color: 'var(--fg-dim, #888)', marginTop: 12, lineHeight: 1.6 }}>
            Your data is safe on the server. Try a reload — if the error persists, copy the
            details below and report it.
          </div>
          <div style={{ marginTop: 18, display: 'flex', gap: 8 }}>
            <button onClick={this.reload} style={{
              padding: '8px 16px', fontSize: 12, fontWeight: 600, cursor: 'pointer',
              background: 'var(--fg, #e4e4e4)', color: 'var(--bg, #0a0a0a)',
              border: '1px solid var(--hairline-strong, #333)',
            }}>Reload</button>
            <button onClick={this.reset} style={{
              padding: '8px 16px', fontSize: 12, cursor: 'pointer',
              background: 'transparent', color: 'var(--fg-muted, #888)',
              border: '1px solid var(--hairline-strong, #333)',
            }}>Try again</button>
            <button onClick={() => this.setState({ expanded: !expanded })} style={{
              padding: '8px 16px', fontSize: 12, cursor: 'pointer', marginLeft: 'auto',
              background: 'transparent', color: 'var(--fg-muted, #888)',
              border: '1px solid var(--hairline-strong, #333)',
            }}>{expanded ? 'Hide details' : 'Show details'}</button>
          </div>
          {expanded && (
            <pre style={{
              marginTop: 18, padding: 12, fontSize: 11, lineHeight: 1.5,
              fontFamily: 'ui-monospace, monospace', overflow: 'auto', maxHeight: 280,
              background: 'var(--surface-1, #141414)', border: '1px solid var(--hairline, #222)',
              color: 'var(--fg-dim, #888)', whiteSpace: 'pre-wrap', wordBreak: 'break-word',
            }}>{String(error?.stack || error)}{info?.componentStack ? '\n\n' + info.componentStack : ''}</pre>
          )}
        </div>
      </div>
    )
  }
}
