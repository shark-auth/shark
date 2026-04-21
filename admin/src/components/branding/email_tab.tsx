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
    <div style={{ display: 'flex', gap: 20, padding: 20, alignItems: 'flex-start' }}>
      {/* ── Left: template list ────────────────────────────────────── */}
      <div
        style={{
          flex: '0 0 200px',
          display: 'flex',
          flexDirection: 'column',
          gap: 4,
          border: '1px solid var(--hairline)',
          borderRadius: 'var(--radius)',
          padding: 6,
          background: 'var(--surface-1)',
        }}
      >
        <div
          style={{
            fontSize: 10,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--fg-muted)',
            padding: '4px 6px 6px',
          }}
        >
          Templates
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
                padding: '6px 8px',
                fontSize: 12,
                background: isSel ? 'var(--surface-2, #e8ecff)' : 'transparent',
                color: isSel ? 'var(--fg)' : 'var(--fg)',
                border: '1px solid ' + (isSel ? 'var(--accent, #4f6bed)' : 'transparent'),
                borderRadius: 'var(--radius)',
                cursor: 'pointer',
                fontWeight: isSel ? 600 : 400,
              }}
            >
              {templateLabel(t.id)}
            </button>
          )
        })}
      </div>

      {/* ── Middle: editor ─────────────────────────────────────────── */}
      <div
        style={{
          flex: '0.4 1 0',
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        <Field label="Subject">
          <input
            type="text"
            value={draft.subject || ''}
            onChange={(e) => patch({ subject: e.target.value })}
            style={inputStyle}
          />
        </Field>

        <Field label="Preheader">
          <input
            type="text"
            value={draft.preheader || ''}
            onChange={(e) => patch({ preheader: e.target.value })}
            style={inputStyle}
          />
        </Field>

        <Field label="Header text">
          <input
            type="text"
            value={draft.header_text || ''}
            onChange={(e) => patch({ header_text: e.target.value })}
            style={inputStyle}
          />
        </Field>

        <Field label="Body paragraphs">
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {paragraphs.map((p, i) => (
              <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'flex-start' }}>
                <textarea
                  value={p}
                  onChange={(e) => setParagraph(i, e.target.value)}
                  rows={2}
                  style={{ ...inputStyle, resize: 'vertical', minHeight: 48, fontFamily: 'inherit', flex: 1 }}
                />
                <button
                  type="button"
                  className="btn sm"
                  onClick={() => removeParagraph(i)}
                  title="Remove paragraph"
                  style={{ flex: '0 0 auto' }}
                >
                  ×
                </button>
              </div>
            ))}
            <button
              type="button"
              className="btn sm"
              onClick={addParagraph}
              style={{ alignSelf: 'flex-start' }}
            >
              + Add paragraph
            </button>
          </div>
        </Field>

        <Field label="CTA button text">
          <input
            type="text"
            value={draft.cta_text || ''}
            onChange={(e) => patch({ cta_text: e.target.value })}
            style={inputStyle}
          />
        </Field>

        <Field label="CTA URL template">
          <input
            type="text"
            value={draft.cta_url_template || ''}
            onChange={(e) => patch({ cta_url_template: e.target.value })}
            placeholder="{{.Link}}"
            style={{ ...inputStyle, fontFamily: 'var(--font-mono)' }}
            spellCheck={false}
          />
        </Field>

        <Field label="Footer">
          <textarea
            value={draft.footer_text || ''}
            onChange={(e) => patch({ footer_text: e.target.value })}
            rows={2}
            style={{ ...inputStyle, resize: 'vertical', minHeight: 56, fontFamily: 'inherit' }}
          />
        </Field>

        <div style={{ display: 'flex', gap: 8, marginTop: 4, flexWrap: 'wrap' }}>
          <button
            type="button"
            className="btn primary"
            onClick={save}
            disabled={saving}
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            className="btn"
            onClick={sendTest}
            disabled={sending}
          >
            {sending ? 'Sending…' : 'Send test'}
          </button>
          <button
            type="button"
            className="btn"
            onClick={reset}
            disabled={resetting}
          >
            {resetting ? 'Resetting…' : 'Reset to default'}
          </button>
        </div>
      </div>

      {/* ── Right: iframe preview ──────────────────────────────────── */}
      <div style={{ flex: '0.6 1 0', minWidth: 0, display: 'flex', flexDirection: 'column', gap: 6 }}>
        <div
          style={{
            fontSize: 10,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--fg-dim)',
          }}
        >
          Live preview — {templateLabel(selected)}
        </div>
        <iframe
          title={`Email preview — ${selected}`}
          srcDoc={previewHTML}
          sandbox=""
          style={{
            width: '100%',
            height: 600,
            border: '1px solid var(--hairline)',
            borderRadius: 'var(--radius)',
            background: '#fff',
          }}
        />
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
