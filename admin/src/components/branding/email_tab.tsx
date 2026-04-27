// @ts-nocheck
// Branding > Email Templates subtab.
//
// Three-column layout:
//   Left  (200px)  — template list, 5 rows seeded by the backend.
//   Middle (0.4fr) — structured field editor (subject, preheader, header text,
//                    body paragraphs array, CTA text/URL, footer).
//   Right  (0.6fr) — iframe live preview, debounced 300ms against the draft.
//
// Spec decision #4A: no rich text / HTML injection — admins type template
// strings like `{{.AppName}}` directly into plain text inputs. Preview calls
// POST /admin/email-templates/{id}/preview with an empty body so the backend
// uses global resolved branding + default sample_data (A7).
//
// State model:
//   templates   — list from GET /admin/email-templates
//   selected    — id of the currently focused template (defaults to first)
//   draft       — full template object being edited in the middle pane
//   previewHTML — most recent rendered body (iframe srcDoc)
//
// Save PATCHes all 7 editable fields + body_paragraphs. Send-test prompts for
// an address and POSTs {to_email}. Reset confirm()s, POSTs /reset, and swaps
// the draft for the fresh seed echoed back by the server. Template selection
// is a no-op if the user clicks the already-selected row (we don't nuke the
// in-progress draft).
import React from 'react'
import { API } from '../api'
import { useToast } from '../toast'

// ── Redirect URL config ──────────────────────────────────────────────────────
// Lets developers point post-action flows at their own hosted pages without
// needing to ship shark's built-in hosted/* pages first.
function EmailRedirectConfig() {
  const [cfg, setCfg] = React.useState({
    verify_redirect_url: '',
    reset_redirect_url: '',
    magic_link_redirect_url: '',
  })
  const [saving, setSaving] = React.useState(false)
  const toast = useToast()

  React.useEffect(() => {
    API.get('/admin/email-config')
      .then((r) => {
        setCfg({
          verify_redirect_url: r.verify_redirect_url || '',
          reset_redirect_url: r.reset_redirect_url || '',
          magic_link_redirect_url: r.magic_link_redirect_url || '',
        })
      })
      .catch(() => {}) // 404 on fresh install is fine — defaults to empty
  }, [])

  const save = async () => {
    setSaving(true)
    try {
      await API.patch('/admin/email-config', cfg)
      toast.success('Redirect URLs saved')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const redirectInputStyle: React.CSSProperties = {
    width: '100%',
    padding: '6px 10px',
    fontSize: 12,
    fontFamily: 'var(--font-mono)',
    background: 'var(--surface-1)',
    border: '1px solid var(--hairline-strong)',
    borderRadius: 2,
    color: 'var(--fg)',
    boxSizing: 'border-box' as const,
    outline: 'none',
    height: 34,
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div>
        <h2 style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 600, margin: '0 0 2px' }}>Redirect URLs</h2>
        <p style={{ fontSize: 11.5, color: 'var(--fg-dim)', margin: 0 }}>
          Where shark sends users after consuming email tokens. Leave blank to use the default hosted page (when live) or the token-embedded URL.
        </p>
      </div>

      <div>
        <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 5, fontWeight: 500 }}>
          Verify-email redirect
        </label>
        <input
          type="url"
          value={cfg.verify_redirect_url}
          onChange={(e) => setCfg(c => ({ ...c, verify_redirect_url: e.target.value }))}
          placeholder="https://yourapp.com/email-verified"
          style={redirectInputStyle}
          spellCheck={false}
        />
        <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
          Called after the verify-email token is consumed. Token appended as <code style={{ fontSize: 10 }}>?token=...</code>
        </div>
      </div>

      <div>
        <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 5, fontWeight: 500 }}>
          Password-reset redirect
        </label>
        <input
          type="url"
          value={cfg.reset_redirect_url}
          onChange={(e) => setCfg(c => ({ ...c, reset_redirect_url: e.target.value }))}
          placeholder="https://yourapp.com/reset-password"
          style={redirectInputStyle}
          spellCheck={false}
        />
        <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
          Your page collects the new password and POSTs to <code style={{ fontSize: 10 }}>POST /auth/reset-password</code>
        </div>
      </div>

      <div>
        <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 5, fontWeight: 500 }}>
          Magic-link redirect (default)
        </label>
        <input
          type="url"
          value={cfg.magic_link_redirect_url}
          onChange={(e) => setCfg(c => ({ ...c, magic_link_redirect_url: e.target.value }))}
          placeholder="https://yourapp.com/dashboard"
          style={redirectInputStyle}
          spellCheck={false}
        />
        <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
          Where to redirect after magic-link login. Per-request override via <code style={{ fontSize: 10 }}>redirect_uri</code> param.
        </div>
      </div>

      <button
        type="button"
        className="btn primary sm"
        style={{ alignSelf: 'flex-start', height: 32, padding: '0 20px', fontSize: 12, fontWeight: 600 }}
        onClick={save}
        disabled={saving}
      >
        {saving ? 'Saving…' : 'Save redirect URLs'}
      </button>
    </div>
  )
}

const TEMPLATE_LABELS: Record<string, string> = {
  magic_link: 'Magic link',
  password_reset: 'Password reset',
  verify_email: 'Verify email',
  organization_invitation: 'Organization invitation',
  welcome: 'Welcome',
}

function templateLabel(id: string): string {
  return TEMPLATE_LABELS[id] || id
}

export function BrandingEmailTab() {
  const [templates, setTemplates] = React.useState<any[]>([])
  const [selected, setSelected] = React.useState<string>('')
  const [draft, setDraft] = React.useState<any>(null)
  const [previewHTML, setPreviewHTML] = React.useState('')
  const [saving, setSaving] = React.useState(false)
  const [sending, setSending] = React.useState(false)
  const [resetting, setResetting] = React.useState(false)
  const toast = useToast()

  // Initial load — pull all templates, default to the first one.
  React.useEffect(() => {
    let cancelled = false
    API.get('/admin/email-templates')
      .then((r) => {
        if (cancelled) return
        const list = r.data || []
        setTemplates(list)
        if (list.length > 0) {
          setSelected(list[0].id)
          setDraft(list[0])
        }
      })
      .catch((e) => {
        if (!cancelled) toast.error(e?.message || 'Failed to load email templates')
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Debounced live preview against the in-progress draft. On failure we keep
  // the last good HTML so the iframe doesn't flash empty mid-edit.
  React.useEffect(() => {
    if (!draft || !selected) return
    const h = setTimeout(async () => {
      try {
        const r = await API.post(`/admin/email-templates/${selected}/preview`, {})
        setPreviewHTML(r.html || '')
      } catch (err) {
        // eslint-disable-next-line no-console
        console.warn('email preview render failed', err)
      }
    }, 300)
    return () => clearTimeout(h)
  }, [draft, selected])

  const selectTemplate = (id: string) => {
    if (id === selected) return // don't nuke an in-progress draft on re-click
    const row = templates.find((t) => t.id === id)
    if (!row) return
    setSelected(id)
    setDraft(row)
  }

  const patch = (p: any) => setDraft((d: any) => ({ ...d, ...p }))

  const setParagraph = (idx: number, value: string) => {
    setDraft((d: any) => {
      const next = [...(d.body_paragraphs || [])]
      next[idx] = value
      return { ...d, body_paragraphs: next }
    })
  }

  const addParagraph = () => {
    setDraft((d: any) => ({ ...d, body_paragraphs: [...(d.body_paragraphs || []), ''] }))
  }

  const removeParagraph = (idx: number) => {
    setDraft((d: any) => {
      const next = [...(d.body_paragraphs || [])]
      next.splice(idx, 1)
      return { ...d, body_paragraphs: next }
    })
  }

  const save = async () => {
    if (!draft || !selected) return
    setSaving(true)
    try {
      const body = {
        subject: draft.subject || '',
        preheader: draft.preheader || '',
        header_text: draft.header_text || '',
        body_paragraphs: draft.body_paragraphs || [],
        cta_text: draft.cta_text || '',
        cta_url_template: draft.cta_url_template || '',
        footer_text: draft.footer_text || '',
      }
      const r = await API.patch(`/admin/email-templates/${selected}`, body)
      setDraft(r)
      setTemplates((prev) => prev.map((t) => (t.id === r.id ? r : t)))
      toast.success('Template saved')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const sendTest = async () => {
    if (!selected) return
    // eslint-disable-next-line no-alert
    const to = window.prompt('Send test email to:')
    if (!to) return
    setSending(true)
    try {
      await API.post(`/admin/email-templates/${selected}/send-test`, { to_email: to })
      toast.success(`Test email sent to ${to}`)
    } catch (e: any) {
      toast.error(e?.message || 'Send test failed')
    } finally {
      setSending(false)
    }
  }

  const reset = async () => {
    if (!selected) return
    // eslint-disable-next-line no-alert
    if (!window.confirm('Reset this template to its default? All edits will be lost.')) return
    setResetting(true)
    try {
      const r = await API.post(`/admin/email-templates/${selected}/reset`, {})
      setDraft(r)
      setTemplates((prev) => prev.map((t) => (t.id === r.id ? r : t)))
      toast.success('Template reset to default')
    } catch (e: any) {
      toast.error(e?.message || 'Reset failed')
    } finally {
      setResetting(false)
    }
  }

  if (!draft) {
    return (
      <div style={{ padding: 40, textAlign: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        Loading email templates…
      </div>
    )
  }

  const paragraphs: string[] = draft.body_paragraphs || []

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 32, padding: 0 }}>
    <div style={{ display: 'flex', gap: 24, padding: 0, alignItems: 'flex-start' }}>
      {/* ── Left: template list ────────────────────────────────────── */}
      <div
        style={{
          flex: '0 0 180px',
          display: 'flex',
          flexDirection: 'column',
          gap: 2,
          border: '1px solid var(--hairline)',
          borderRadius: 4,
          padding: '6px 2px',
          background: 'var(--surface-1)',
        }}
      >
        <div
          style={{
            fontSize: 9,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--fg-dim)',
            padding: '4px 10px 4px',
            fontWeight: 500
          }}
        >
          Communication
        </div>
        {templates.map((t) => {
          const isSel = t.id === selected
          return (
            <button
              key={t.id}
              type="button"
              onClick={() => selectTemplate(t.id)}
              style={{
                textAlign: 'left',
                padding: '6px 10px',
                fontSize: 12,
                background: isSel ? 'var(--surface-2)' : 'transparent',
                color: isSel ? 'var(--fg)' : 'var(--fg-muted)',
                borderRadius: 2,
                cursor: 'pointer',
                fontWeight: isSel ? 600 : 400,
                transition: 'all 60ms ease-out',
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                border: 'none'
              }}
            >
              <div style={{ width: 4, height: 4, borderRadius: 1, background: isSel ? 'var(--fg-dim)' : 'transparent' }} />
              {templateLabel(t.id)}
            </button>
          )
        })}
      </div>

      {/* ── Middle: editor ─────────────────────────────────────────── */}
      <div
        style={{
          flex: '0.45 1 0',
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 20,
        }}
      >
        <div style={{ marginBottom: 4 }}>
          <h2 style={{ fontFamily: 'var(--font-sans)', fontSize: 15, fontWeight: 600, margin: '0 0 2px' }}>Template Content</h2>
          <p style={{ fontSize: 11.5, color: 'var(--fg-dim)', margin: 0 }}>Editable strings for <b>{templateLabel(selected)}</b>.</p>
        </div>

        <Field label="Message Subject">
          <input
            type="text"
            value={draft.subject || ''}
            onChange={(e) => patch({ subject: e.target.value })}
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        <Field label="System Preheader">
          <input
            type="text"
            value={draft.preheader || ''}
            onChange={(e) => patch({ preheader: e.target.value })}
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        <Field label="Primary Header Text">
          <input
            type="text"
            value={draft.header_text || ''}
            onChange={(e) => patch({ header_text: e.target.value })}
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        <Field label="Message Body Paragraphs">
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {paragraphs.map((p, i) => (
              <div key={i} style={{ display: 'flex', gap: 8, alignItems: 'flex-start' }}>
                <textarea
                  value={p}
                  onChange={(e) => setParagraph(i, e.target.value)}
                  rows={3}
                  style={{ ...inputStyle, resize: 'vertical', minHeight: 60, fontFamily: 'inherit', flex: 1 }}
                />
                <button
                  type="button"
                  className="btn danger"
                  onClick={() => removeParagraph(i)}
                  title="Remove paragraph"
                  style={{ flex: '0 0 auto', width: 32, height: 32, padding: 0, justifyContent: 'center' }}
                >
                  ×
                </button>
              </div>
            ))}
            <button
              type="button"
              className="btn"
              onClick={addParagraph}
              style={{ alignSelf: 'flex-start', height: 32, padding: '0 16px' }}
            >
              + Add new paragraph
            </button>
          </div>
        </Field>

        <Field label="Call to Action Label">
          <input
            type="text"
            value={draft.cta_text || ''}
            onChange={(e) => patch({ cta_text: e.target.value })}
            style={{ ...inputStyle, height: 40 }}
          />
        </Field>

        <Field label="Action Target Template">
          <input
            type="text"
            value={draft.cta_url_template || ''}
            onChange={(e) => patch({ cta_url_template: e.target.value })}
            placeholder="{{.Link}}"
            style={{ ...inputStyle, fontFamily: 'var(--font-mono)', height: 40 }}
            spellCheck={false}
          />
        </Field>

        <Field label="Footer Signature">
          <textarea
            value={draft.footer_text || ''}
            onChange={(e) => patch({ footer_text: e.target.value })}
            rows={2}
            style={{ ...inputStyle, resize: 'vertical', minHeight: 60, fontFamily: 'inherit' }}
          />
        </Field>

        <div style={{ display: 'flex', gap: 10, marginTop: 12, flexWrap: 'wrap', alignItems: 'center' }}>
          <button
            type="button"
            className="btn primary sm"
            style={{ height: 32, padding: '0 20px', fontSize: 12, fontWeight: 600 }}
            onClick={save}
            disabled={saving}
          >
            {saving ? 'Saving…' : 'Publish'}
          </button>
          <button
            type="button"
            className="btn ghost sm"
            style={{ height: 32, padding: '0 16px', fontSize: 12 }}
            onClick={sendTest}
            disabled={sending}
          >
            {sending ? 'Sending…' : 'Send test'}
          </button>
          <div style={{ flex: 1 }} />
          <button
            type="button"
            className="btn danger sm"
            style={{ height: 32, padding: '0 16px', fontSize: 12 }}
            onClick={reset}
            disabled={resetting}
          >
            {resetting ? 'Resetting…' : 'Reset'}
          </button>
        </div>
      </div>

      {/* ── Right: iframe preview ──────────────────────────────────── */}
      <div style={{ flex: '1 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center' }}>
        <div style={{ width: '100%', maxWidth: 460 }}>
          <div
            style={{
              fontSize: 10,
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
              color: 'var(--fg-dim)',
              fontWeight: 500,
              marginBottom: 8,
              borderBottom: '1px solid var(--hairline)',
              paddingBottom: 4
            }}
          >
            Visual Preview — {templateLabel(selected)}
          </div>
          <div style={{
            width: '100%',
            aspectRatio: '1 / 1',
            background: 'var(--surface-1)',
            borderRadius: 4,
            border: '1px solid var(--hairline-strong)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundSize: '20px 20px',
            backgroundImage: 'radial-gradient(var(--hairline) 1px, transparent 0)',
            overflow: 'hidden'
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
                title={`Email preview — ${selected}`}
                srcDoc={previewHTML}
                sandbox=""
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
            Template state rendered via backend resolver.
            Dynamic tokens injected into buffer.
          </div>
        </div>
      </div>
    </div>
    {/* ── Redirect URL config ─────────────────────────────────────── */}
    <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 24 }}>
      <EmailRedirectConfig />
    </div>
    </div>
  )
}

// ── helpers ────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  width: '100%',
  padding: '6px 10px',
  fontSize: 13,
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
