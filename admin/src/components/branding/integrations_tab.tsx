// @ts-nocheck
// Branding > Integrations subtab.
//
// Per-application integration_mode picker + a "Get snippet" modal that
// fetches /admin/apps/{id}/snippet?framework=react and exposes three
// copy-pasteable React snippets (install, provider setup, page usage).
//
// Route contract (A8):
//   GET    /admin/apps                   → { applications: [...] }
//   PATCH  /admin/apps/{id}              → update integration_mode / fallbacks
//   GET    /admin/apps/{id}/snippet      → { framework, snippets: [{label,lang,code}] }
//
// Apps do not carry a slug yet, so the hosted-login shortcut uses client_id
// as the stable public identifier.
import React from 'react'
import { API } from '../api'
import { useToast } from '../toast'

const MODES = [
  { value: 'hosted', label: 'Hosted' },
  { value: 'components', label: 'Components' },
  { value: 'proxy', label: 'Proxy' },
  { value: 'custom', label: 'Custom' },
]

const FRAMEWORKS = [
  { value: 'react', label: 'React', enabled: true },
  { value: 'vue', label: 'Vue (coming soon)', enabled: false },
  { value: 'svelte', label: 'Svelte (coming soon)', enabled: false },
  { value: 'solid', label: 'Solid (coming soon)', enabled: false },
  { value: 'angular', label: 'Angular (coming soon)', enabled: false },
]

const thStyle: React.CSSProperties = {
  textAlign: 'left',
  padding: '8px 14px',
  fontSize: 10,
  fontWeight: 500,
  color: 'var(--fg-dim)',
  borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-0)',
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
}
const tdStyle: React.CSSProperties = {
  padding: '10px 14px',
  borderBottom: '1px solid var(--hairline)',
  verticalAlign: 'middle',
  fontSize: 12,
}
const inputStyle: React.CSSProperties = {
  padding: '5px 8px',
  fontSize: 12,
  background: 'var(--surface-1)',
  border: '1px solid var(--hairline)',
  borderRadius: 'var(--radius)',
  color: 'var(--fg)',
  boxSizing: 'border-box',
}

export function BrandingIntegrationsTab() {
  const [apps, setApps] = React.useState<any[] | null>(null)
  const [snippetModal, setSnippetModal] = React.useState<any>(null)
  const toast = useToast()

  React.useEffect(() => {
    let cancelled = false
    API.get('/admin/apps')
      .then((r) => {
        if (cancelled) return
        setApps(r?.applications || [])
      })
      .catch((e) => {
        if (!cancelled) toast.error(e?.message || 'Failed to load applications')
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const update = async (app: any, patch: any) => {
    try {
      await API.patch(`/admin/apps/${app.id}`, patch)
      setApps((list) =>
        (list || []).map((a) => (a.id === app.id ? { ...a, ...patch } : a)),
      )
    } catch (e: any) {
      toast.error(e?.message || 'Update failed')
    }
  }

  const openSnippet = async (app: any) => {
    try {
      const r = await API.get(`/admin/apps/${app.id}/snippet?framework=react`)
      setSnippetModal({ app, snippets: r?.snippets || [] })
    } catch (e: any) {
      toast.error(e?.message || 'Failed to load snippet')
    }
  }

  if (!apps) {
    return (
      <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        Loading applications…
      </div>
    )
  }

  if (apps.length === 0) {
    return (
      <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        No applications registered. Create one under Applications first.
      </div>
    )
  }

  return (
    <div style={{ padding: 20 }}>
      <div style={{ marginBottom: 12, fontSize: 11, color: 'var(--fg-muted)' }}>
        Pick how each application authenticates users. Per-app overrides take precedence over
        the tenant default.
      </div>
      <div
        style={{
          border: '1px solid var(--hairline)',
          borderRadius: 'var(--radius)',
          overflow: 'hidden',
          background: 'var(--surface-1)',
        }}
      >
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={thStyle}>Application</th>
              <th style={{ ...thStyle, width: 160 }}>Integration mode</th>
              <th style={thStyle}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {apps.map((app) => (
              <AppRow
                key={app.id}
                app={app}
                onMode={(m) => update(app, { integration_mode: m })}
                onFallback={(m) => update(app, { proxy_login_fallback: m })}
                onFallbackURL={(u) => update(app, { proxy_login_fallback_url: u })}
                onSnippet={() => openSnippet(app)}
              />
            ))}
          </tbody>
        </table>
      </div>

      {snippetModal && (
        <SnippetModal
          data={snippetModal}
          onClose={() => setSnippetModal(null)}
          toast={toast}
        />
      )}
    </div>
  )
}

function AppRow({
  app,
  onMode,
  onFallback,
  onFallbackURL,
  onSnippet,
}: {
  app: any
  onMode: (m: string) => void
  onFallback: (m: string) => void
  onFallbackURL: (u: string) => void
  onSnippet: () => void
}) {
  const mode = app.integration_mode || 'hosted'
  const slug = app.slug || app.client_id || app.id
  const [urlDraft, setUrlDraft] = React.useState(app.proxy_login_fallback_url || '')

  React.useEffect(() => {
    setUrlDraft(app.proxy_login_fallback_url || '')
  }, [app.proxy_login_fallback_url])

  return (
    <tr>
      <td style={tdStyle}>
        <div style={{ fontWeight: 500 }}>{app.name}</div>
        <div style={{ fontSize: 10, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>
          {app.client_id}
        </div>
      </td>
      <td style={tdStyle}>
        <select
          value={mode}
          onChange={(e) => onMode(e.target.value)}
          style={{ ...inputStyle, width: '100%' }}
        >
          {MODES.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </select>
      </td>
      <td style={tdStyle}>
        {mode === 'hosted' && (
          <a
            href={`/hosted/${slug}/login`}
            target="_blank"
            rel="noopener noreferrer"
            className="btn sm"
          >
            Open hosted login ↗
          </a>
        )}
        {mode === 'components' && (
          <button type="button" className="btn sm primary" onClick={onSnippet}>
            Get snippet
          </button>
        )}
        {mode === 'proxy' && (
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
            <span style={{ fontSize: 11, color: 'var(--fg-muted)' }}>Fallback:</span>
            <select
              value={app.proxy_login_fallback || 'hosted'}
              onChange={(e) => onFallback(e.target.value)}
              style={inputStyle}
            >
              <option value="hosted">hosted</option>
              <option value="custom_url">custom_url</option>
            </select>
            {app.proxy_login_fallback === 'custom_url' && (
              <input
                type="url"
                value={urlDraft}
                placeholder="https://app.example.com/login"
                onChange={(e) => setUrlDraft(e.target.value)}
                onBlur={() => {
                  if (urlDraft !== (app.proxy_login_fallback_url || '')) {
                    onFallbackURL(urlDraft)
                  }
                }}
                style={{ ...inputStyle, minWidth: 240 }}
              />
            )}
          </div>
        )}
        {mode === 'custom' && (
          <span style={{ fontSize: 11, color: 'var(--fg-muted)' }}>
            API-only. See <code>/api/v1/auth/*</code>
          </span>
        )}
      </td>
    </tr>
  )
}

function SnippetModal({
  data,
  onClose,
  toast,
}: {
  data: { app: any; snippets: any[] }
  onClose: () => void
  toast: any
}) {
  const { app, snippets } = data

  const copy = async (code: string) => {
    try {
      await navigator.clipboard.writeText(code)
      toast.success('Copied')
    } catch (e: any) {
      toast.error('Copy failed — select the code and copy manually')
    }
  }

  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 50,
      }}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-bright)',
          borderRadius: 6,
          padding: 20,
          width: 'min(720px, 92vw)',
          maxHeight: '88vh',
          overflow: 'auto',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            marginBottom: 14,
          }}
        >
          <h2 style={{ margin: 0, fontSize: 15, fontWeight: 600, flex: 1 }}>
            Integration snippets: {app.name}
          </h2>
          <button type="button" className="btn sm" onClick={onClose}>
            Close
          </button>
        </div>

        <div style={{ marginBottom: 14 }}>
          <label
            style={{
              display: 'block',
              fontSize: 10,
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
              color: 'var(--fg-muted)',
              marginBottom: 6,
              fontWeight: 500,
            }}
          >
            Framework
          </label>
          <select value="react" onChange={() => {}} style={inputStyle}>
            {FRAMEWORKS.map((f) => (
              <option key={f.value} value={f.value} disabled={!f.enabled}>
                {f.label}
              </option>
            ))}
          </select>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {snippets.map((s, i) => (
            <div key={i}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  marginBottom: 6,
                }}
              >
                <div
                  style={{
                    flex: 1,
                    fontSize: 10,
                    textTransform: 'uppercase',
                    letterSpacing: '0.08em',
                    color: 'var(--fg-muted)',
                    fontWeight: 500,
                  }}
                >
                  {s.label}
                </div>
                <button
                  type="button"
                  className="btn sm"
                  onClick={() => copy(s.code)}
                >
                  Copy
                </button>
              </div>
              <pre
                style={{
                  margin: 0,
                  padding: 12,
                  fontSize: 11.5,
                  fontFamily: 'var(--font-mono)',
                  background: 'var(--surface-0)',
                  border: '1px solid var(--hairline)',
                  borderRadius: 'var(--radius)',
                  overflow: 'auto',
                  whiteSpace: 'pre',
                }}
              >
                {s.code}
              </pre>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
