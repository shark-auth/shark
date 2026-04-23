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
      const res = await API.postFormData('/admin/branding/logo', form)
      const j = res
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
    <div style={{ display: 'flex', gap: 32, padding: 0, alignItems: 'flex-start' }}>
      {/* ── Left: form ─────────────────────────────────────────────── */}
      <div style={{ flex: '0 0 400px', maxWidth: 400, display: 'flex', flexDirection: 'column', gap: 20 }}>
        <div style={{ marginBottom: 4 }}>
          <h2 style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 600, margin: '0 0 2px' }}>Identity</h2>
          <p style={{ fontSize: 11.5, color: 'var(--fg-dim)', margin: 0 }}>Visual branding & identifiers.</p>
        </div>

        {/* Logo */}
        <Field label="Brand Logo">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 12 }}>
            {draft.logo_url ? (
              <div style={{ 
                padding: 12, 
                background: 'var(--surface-0)', 
                border: '1px solid var(--hairline-strong)', 
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}>
                <img
                  src={draft.logo_url}
                  alt="Current logo"
                  style={{ maxHeight: 60, maxWidth: 180 }}
                />
              </div>
            ) : (
              <div style={{ fontSize: 12, color: 'var(--fg-muted)', padding: '12px', border: '1px dashed var(--hairline-strong)', borderRadius: '12px', width: '100%', textAlign: 'center' }}>
                No logo uploaded
              </div>
            )}
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'center' }}>
            <label
              className="btn"
              style={{ padding: '0 20px', height: 36, cursor: uploading ? 'wait' : 'pointer', opacity: uploading ? 0.6 : 1 }}
            >
              {uploading ? 'Uploading…' : 'Upload Logo'}
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
                className="btn danger"
                style={{ height: 36, padding: '0 16px' }}
                onClick={removeLogo}
                disabled={uploading}
              >
                Remove
              </button>
            )}
          </div>
          <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 8 }}>
            Supports PNG, SVG, JPG (Max 1MB). Transparent background recommended.
          </div>
        </Field>

        <div style={{ height: 1, background: 'var(--hairline)', margin: '4px 0' }} />

        <div style={{ marginBottom: 4 }}>
          <h2 style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 600, margin: '0 0 2px' }}>Visual Style</h2>
          <p style={{ fontSize: 11.5, color: 'var(--fg-dim)', margin: 0 }}>Colors & typography.</p>
        </div>

        {/* Primary color */}
        <ColorField
          label="Primary Accent"
          value={draft.primary_color || '#000000'}
          onChange={(c) => patch({ primary_color: c })}
        />

        {/* Secondary color */}
        <ColorField
          label="Secondary Surface"
          value={draft.secondary_color || '#000000'}
          onChange={(c) => patch({ secondary_color: c })}
        />

        {/* Font */}
        <Field label="Typography Interface">
          <select
            value={draft.font_family || 'inter'}
            onChange={(e) => patch({ font_family: e.target.value })}
            style={{ ...inputStyle, height: 40 }}
          >
            {FONT_OPTIONS.map((f) => (
              <option key={f.value} value={f.value}>
                {f.label}
              </option>
            ))}
          </select>
        </Field>

        <div style={{ height: 1, background: 'var(--hairline)', margin: '4px 0' }} />

        <div style={{ marginBottom: 4 }}>
          <h2 style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 600, margin: '0 0 2px' }}>Communication</h2>
          <p style={{ fontSize: 11.5, color: 'var(--fg-dim)', margin: 0 }}>Sender metadata & signatures.</p>
        </div>

        {/* Footer */}
        <Field label="System Footer Note">
          <textarea
            value={draft.footer_text || ''}
            onChange={(e) => patch({ footer_text: e.target.value })}
            rows={3}
            placeholder="© Acme 2026 · support@acme.com"
            style={{ ...inputStyle, resize: 'vertical', minHeight: 80, fontFamily: 'inherit' }}
          />
        </Field>

        {/* Email from name */}
        <Field label="Sender Display Name">
          <input
            type="text"
            value={draft.email_from_name || ''}
            onChange={(e) => patch({ email_from_name: e.target.value })}
            placeholder="Acme Security"
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        {/* Email from address */}
        <Field label="Sender Reply-To Address">
          <input
            type="email"
            value={draft.email_from_address || ''}
            onChange={(e) => patch({ email_from_address: e.target.value })}
            placeholder="no-reply@acme.com"
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        {/* Save / discard */}
        <div
          style={{
            display: 'flex',
            gap: 12,
            marginTop: 8,
            position: 'sticky',
            bottom: -32,
            background: 'var(--bg)',
            padding: '20px 0 24px',
            borderTop: '1px solid var(--hairline)',
            zIndex: 1,
            alignItems: 'center'
          }}
        >
          <button
            type="button"
            className="btn primary sm"
            style={{ height: 32, padding: '0 20px', fontSize: 12, fontWeight: 600 }}
            onClick={save}
            disabled={!dirty || saving}
          >
            {saving ? 'Saving…' : 'Publish'}
          </button>
          <button
            type="button"
            className="btn ghost sm"
            style={{ height: 32, padding: '0 16px', fontSize: 12 }}
            onClick={discard}
            disabled={!dirty || saving}
          >
            Discard
          </button>
          {dirty && (
            <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginLeft: 'auto' }}>
               <div style={{ width: 6, height: 6, borderRadius: 1, background: 'var(--warn)' }} />
               <span style={{ fontSize: 11, fontWeight: 500, color: 'var(--fg-dim)' }}>
                 Draft edits
               </span>
            </div>
          )}
        </div>
      </div>

      {/* ── Right: live preview ────────────────────────────────────── */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
        <div style={{ width: '100%', maxWidth: 460 }}>
          <div style={{ display: 'flex', borderBottom: '1px solid var(--hairline)', gap: 0, marginBottom: 16 }}>
             <div style={{ 
               borderBottom: '2px solid var(--fg-dim)', 
               borderRadius: 0, 
               color: 'var(--fg)', 
               height: 36,
               fontSize: 11,
               fontWeight: 600,
               padding: '0 4px',
               display: 'flex',
               alignItems: 'center'
             }}>
               Live Page Preview
             </div>
          </div>

          <div style={{ 
            width: '100%', 
            aspectRatio: '1 / 1',
            background: 'var(--surface-1)',
            borderRadius: 4,
            border: '1px solid var(--hairline-strong)',
            overflow: 'hidden',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundSize: '20px 20px',
            backgroundImage: 'radial-gradient(var(--hairline) 1px, transparent 0)',
          }}>
            <div style={{ 
              width: '125%', 
              height: '125%', 
              transform: 'scale(0.8)',
              transformOrigin: 'center',
              boxShadow: '0 30px 60px rgba(0,0,0,0.4)',
              borderRadius: 8,
              overflow: 'hidden',
              border: '1px solid var(--hairline-bright)'
            }}>
              <iframe
                title="Hosted Login Preview"
                src={`/hosted/default/login?preview=true&primary=${encodeURIComponent(draft.primary_color)}&secondary=${encodeURIComponent(draft.secondary_color)}&font=${encodeURIComponent(draft.font_family)}`}
                style={{
                  width: '100%',
                  height: '100%',
                  border: 'none',
                  background: '#fff',
                }}
              />
            </div>
          </div>

          <div style={{ fontSize: 10.5, color: 'var(--fg-dim)', textAlign: 'center', marginTop: 16, maxWidth: 360, margin: '16px auto 0', lineHeight: 1.5 }}>
            Real-time reconciliation through optimistic rendering.
            Page state tracks draft in-buffer.
          </div>
        </div>
      </div>
    </div>
  )
}

// ── helpers ────────────────────────────────────────────────────────

  const inputStyle: React.CSSProperties = {
    width: '100%',
    padding: '6px 10px',
    fontSize: 13,
    fontFamily: 'var(--font-sans)',
    background: 'var(--surface-1)',
    border: '1px solid var(--hairline-strong)',
    borderRadius: 2,
    color: 'var(--fg)',
    boxSizing: 'border-box',
    outline: 'none',
    transition: 'border-color 60ms ease-out'
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
          color: 'var(--fg-dim)',
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
            style={{ ...inputStyle, fontFamily: 'var(--font-mono)', fontSize: 11.5 }}
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
