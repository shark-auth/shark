// @ts-nocheck
// Branding > Visuals subtab.
//
// Left pane: editable form (logo upload, two color pickers, font select,
// footer textarea, email from-name + from-address).
// Right pane: live preview of the magic_link email template rendered by the
// backend and dropped into a sandboxed iframe via srcDoc. Preview is
// debounced 300ms against the in-progress draft so every keystroke doesn't
// round-trip to the server.
//
// State model:
//   cfg    — last canonical branding config (from GET /admin/branding)
//   draft  — in-progress edits, mutated on every field change
//   dirty  — draft != cfg; gates Save + Discard
//   previewHTML — latest rendered HTML body, iframe srcDoc
//
// Save PATCHes the draft, swaps cfg = draft, clears dirty. Discard resets
// draft = cfg. Logo upload + remove hit their own endpoints and update
// draft + dirty in place (so the user can still discard a just-uploaded
// logo before persisting the rest of the form).
import React from 'react'
import { HexColorPicker } from 'react-colorful'
import { API } from '../api'
import { useToast } from '../toast'

const FONT_OPTIONS = [
  { value: 'manrope', label: 'Manrope' },
  { value: 'inter', label: 'Inter' },
  { value: 'ibm_plex', label: 'IBM Plex' },
]

// errorCodeMessage maps the backend error codes for logo upload to
// user-facing toast copy. Falls back to a generic message so we never show
// the raw JSON body to the admin.
function errorCodeMessage(code: string): string {
  switch (code) {
    case 'file_too_large':
      return 'Logo must be 1 MB or smaller'
    case 'invalid_file_type':
      return 'Logo must be .png, .svg, .jpg, or .jpeg'
    case 'invalid_svg':
      return 'SVG contains disallowed scripting content'
    default:
      return 'Logo upload failed'
  }
}

// shallowEqual is enough to diff cfg vs draft — the branding row is flat.
function shallowEqual(a: any, b: any): boolean {
  if (a === b) return true
  if (!a || !b) return false
  const ak = Object.keys(a)
  const bk = Object.keys(b)
  if (ak.length !== bk.length) return false
  for (const k of ak) {
    if (a[k] !== b[k]) return false
  }
  return true
}

export function BrandingVisualsTab() {
  const [cfg, setCfg] = React.useState<any>(null)
  const [draft, setDraft] = React.useState<any>(null)
  const [previewHTML, setPreviewHTML] = React.useState('')
  const [saving, setSaving] = React.useState(false)
  const [uploading, setUploading] = React.useState(false)
  const toast = useToast()

  const dirty = !!(cfg && draft && !shallowEqual(cfg, draft))

  // Initial load.
  React.useEffect(() => {
    let cancelled = false
    API.get('/admin/branding')
      .then((r) => {
        if (cancelled) return
        setCfg(r.branding)
        setDraft(r.branding)
      })
      .catch((e) => {
        if (!cancelled) toast.error(e?.message || 'Failed to load branding')
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Debounced live preview. Runs any time the draft changes. On failure
  // we keep the last good HTML so the iframe doesn't flash empty.
  React.useEffect(() => {
    if (!draft) return
    const h = setTimeout(async () => {
      try {
        const r = await API.post('/admin/email-templates/magic_link/preview', {
          config: draft,
          sample_data: {
            AppName: 'Acme',
            Link: 'https://example.com/login?token=sample',
            UserEmail: 'user@example.com',
          },
        })
        setPreviewHTML(r.html || '')
      } catch (err) {
        // Don't clobber last good preview; just log.
        // eslint-disable-next-line no-console
        console.warn('preview render failed', err)
      }
    }, 300)
    return () => clearTimeout(h)
  }, [draft])

  const patch = (patch: any) => setDraft((d: any) => ({ ...d, ...patch }))

  const save = async () => {
    setSaving(true)
    try {
      const r = await API.patch('/admin/branding', draft)
      setCfg(r.branding)
      setDraft(r.branding)
      toast.success('Branding saved')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const discard = () => {
    if (cfg) setDraft(cfg)
  }

  const uploadLogo = async (file: File) => {
    setUploading(true)
    try {
      const form = new FormData()
      form.append('logo', file)
      const res = await fetch('/api/v1/admin/branding/logo', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${sessionStorage.getItem('shark_admin_key')}`,
        },
        body: form,
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        toast.error(errorCodeMessage(body?.code || ''))
        return
      }
      const j = await res.json()
      // Upload already persists server-side, so update both cfg + draft so
      // the Save button doesn't light up for a change we already committed.
      const next = { ...draft, logo_url: j.logo_url, logo_sha: j.logo_sha }
      setDraft(next)
      setCfg((c: any) => ({ ...(c || {}), logo_url: j.logo_url, logo_sha: j.logo_sha }))
      toast.success('Logo uploaded')
    } catch (e: any) {
      toast.error(e?.message || 'Logo upload failed')
    } finally {
      setUploading(false)
    }
  }

  const removeLogo = async () => {
    try {
      await API.del('/admin/branding/logo')
      const next = { ...draft, logo_url: '', logo_sha: '' }
      setDraft(next)
      setCfg((c: any) => ({ ...(c || {}), logo_url: '', logo_sha: '' }))
      toast.success('Logo removed')
    } catch (e: any) {
      toast.error(e?.message || 'Remove failed')
    }
  }

  if (!draft) {
    return (
      <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        Loading branding…
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', gap: 24, padding: 20, alignItems: 'flex-start' }}>
      {/* ── Left: form ─────────────────────────────────────────────── */}
      <div style={{ flex: '0 0 400px', maxWidth: 400, display: 'flex', flexDirection: 'column', gap: 18 }}>
        {/* Logo */}
        <Field label="Logo">
          <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
            {draft.logo_url ? (
              <img
                src={draft.logo_url}
                alt="Current logo"
                style={{
                  maxHeight: 48,
                  maxWidth: 140,
                  padding: 4,
                  background: 'var(--surface-1)',
                  border: '1px solid var(--hairline)',
                  borderRadius: 'var(--radius)',
                }}
              />
            ) : (
              <div style={{ fontSize: 11, color: 'var(--fg-muted)' }}>No logo uploaded</div>
            )}
          </div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <label
              className="btn sm"
              style={{ cursor: uploading ? 'wait' : 'pointer', opacity: uploading ? 0.6 : 1 }}
            >
              {uploading ? 'Uploading…' : 'Upload'}
              <input
                type="file"
                accept=".png,.svg,.jpg,.jpeg"
                style={{ display: 'none' }}
                disabled={uploading}
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) uploadLogo(f)
                  e.target.value = ''
                }}
              />
            </label>
            {draft.logo_url && (
              <button
                type="button"
                className="btn sm"
                onClick={removeLogo}
                disabled={uploading}
              >
                Remove logo
              </button>
            )}
          </div>
          <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 6 }}>
            .png .svg .jpg .jpeg · max 1&nbsp;MB
          </div>
        </Field>

        {/* Primary color */}
        <ColorField
          label="Primary color"
          value={draft.primary_color || '#000000'}
          onChange={(c) => patch({ primary_color: c })}
        />

        {/* Secondary color */}
        <ColorField
          label="Secondary color"
          value={draft.secondary_color || '#000000'}
          onChange={(c) => patch({ secondary_color: c })}
        />

        {/* Font */}
        <Field label="Font family">
          <select
            value={draft.font_family || 'inter'}
            onChange={(e) => patch({ font_family: e.target.value })}
            style={inputStyle}
          >
            {FONT_OPTIONS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
        </Field>

        {/* Footer */}
        <Field label="Footer text">
          <textarea
            value={draft.footer_text || ''}
            onChange={(e) => patch({ footer_text: e.target.value })}
            rows={2}
            placeholder="© Acme 2026 · support@acme.com"
            style={{ ...inputStyle, resize: 'vertical', minHeight: 56, fontFamily: 'inherit' }}
          />
        </Field>

        {/* Email from name */}
        <Field label="Email from name">
          <input
            type="text"
            value={draft.email_from_name || ''}
            onChange={(e) => patch({ email_from_name: e.target.value })}
            placeholder="Acme Security"
            style={inputStyle}
          />
        </Field>

        {/* Email from address */}
        <Field label="Email from address">
          <input
            type="email"
            value={draft.email_from_address || ''}
            onChange={(e) => patch({ email_from_address: e.target.value })}
            placeholder="no-reply@acme.com"
            style={inputStyle}
          />
        </Field>

        {/* Save / discard — sticky to bottom of scroll container so it's
            always reachable without scrolling on tall forms. */}
        <div
          style={{
            display: 'flex',
            gap: 8,
            marginTop: 8,
            position: 'sticky',
            bottom: 0,
            background: 'var(--surface-1, #0e0e10)',
            padding: '12px 0',
            borderTop: '1px solid var(--hairline)',
            zIndex: 1,
          }}
        >
          <button
            type="button"
            className="btn primary"
            onClick={save}
            disabled={!dirty || saving}
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            className="btn"
            onClick={discard}
            disabled={!dirty || saving}
          >
            Discard
          </button>
          {dirty && (
            <span style={{ fontSize: 11, color: 'var(--warn, #b58900)', alignSelf: 'center' }}>
              unsaved changes
            </span>
          )}
        </div>
      </div>

      {/* ── Right: live preview ────────────────────────────────────── */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'flex', borderBottom: '1px solid var(--hairline)', gap: 10 }}>
           <button className="btn sm ghost" style={{ borderBottom: '2px solid var(--accent)', borderRadius: 0 }}>Login Page</button>
           <button className="btn sm ghost" style={{ borderRadius: 0 }}>Magic Link Email</button>
        </div>

        <div style={{ flex: 1, position: 'relative' }}>
          <div style={{ position: 'absolute', top: 12, right: 12, zIndex: 10, pointerEvents: 'none' }}>
            <span className="chip agent sm">LIVE PREVIEW</span>
          </div>
          
          <iframe
            title="Hosted Login Preview"
            src={`/hosted/default/login?preview=true&primary=${encodeURIComponent(draft.primary_color)}&secondary=${encodeURIComponent(draft.secondary_color)}&font=${encodeURIComponent(draft.font_family)}`}
            style={{
              width: '100%',
              height: 600,
              border: '1px solid var(--hairline)',
              borderRadius: 'var(--radius)',
              background: '#fff',
            }}
          />
        </div>

        <div style={{ fontSize: 11, color: 'var(--fg-dim)', textAlign: 'center' }}>
          Real-time preview of your <b>Hosted Login Page</b>. Changes are applied via query params before saving.
        </div>
      </div>
    </div>
  )
}

// ── helpers ────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '6px 8px',
  fontSize: 12,
  background: 'var(--surface-1)',
  border: '1px solid var(--hairline)',
  borderRadius: 'var(--radius)',
  color: 'var(--fg)',
  boxSizing: 'border-box',
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
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
        {label}
      </label>
      {children}
    </div>
  )
}

// ColorField pairs a HexColorPicker with a text input. Editing either
// updates the draft; the hex input is the escape hatch for paste-in values.
function ColorField({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (c: string) => void
}) {
  return (
    <Field label={label}>
      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
        <div style={{ width: 140 }}>
          <HexColorPicker
            color={value}
            onChange={onChange}
            style={{ width: 140, height: 120 }}
          />
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 6 }}>
          <input
            type="text"
            value={value}
            onChange={(e) => onChange(e.target.value)}
            style={{ ...inputStyle, fontFamily: 'var(--font-mono)' }}
            spellCheck={false}
          />
          <div
            style={{
              width: '100%',
              height: 32,
              background: value,
              border: '1px solid var(--hairline)',
              borderRadius: 'var(--radius)',
            }}
            aria-label={`${label} swatch`}
          />
        </div>
      </div>
    </Field>
  )
}
