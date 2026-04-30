// @ts-nocheck
import React from 'react'
import { API } from './api'
import { useToast } from './toast'
import { Icon } from './shared'

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

export function MailEditor() {
  const [templates, setTemplates] = React.useState<any[]>([])
  const [selected, setSelected] = React.useState<string>('appearance')
  const [draft, setDraft] = React.useState<any>(null)
  const [branding, setBranding] = React.useState<any>(null)
  const [previewHTML, setPreviewHTML] = React.useState('')
  const [previewMode, setPreviewMode] = React.useState<'light' | 'dark'>('light')
  const [saving, setSaving] = React.useState(false)
  const [sending, setSending] = React.useState(false)
  const [resetting, setResetting] = React.useState(false)
  const [uploading, setUploading] = React.useState(false)
  const toast = useToast()

  const loadAll = async () => {
    try {
      const [tResp, bResp] = await Promise.all([
        API.get('/admin/email-templates'),
        API.get('/admin/branding')
      ])
      const list = tResp.data || []
      setTemplates(list)
      setBranding(bResp.branding || {})
      if (selected !== 'appearance' && selected !== 'redirects') {
        const row = list.find(t => t.id === selected)
        if (row) setDraft(row)
        else setSelected('appearance')
      }
    } catch (e: any) {
      toast.error(e?.message || 'Failed to load data')
    }
  }

  React.useEffect(() => {
    loadAll()
  }, [])

  React.useEffect(() => {
    if (selected === 'redirects' || !selected) return
    const previewTarget = selected === 'appearance' ? 'magic_link' : selected
    const h = setTimeout(async () => {
      try {
        const r = await API.post(`/admin/email-templates/${previewTarget}/preview`, {})
        setPreviewHTML(r.html || '')
      } catch (err) {
        console.warn('email preview render failed', err)
      }
    }, 300)
    return () => clearTimeout(h)
  }, [draft, selected, branding])

  const uploadLogo = async (file: File) => {
    setUploading(true)
    const form = new FormData()
    form.append('logo', file)
    try {
      const res = await API.postFormData('/admin/branding/logo', form)
      setBranding((b: any) => ({ ...b, logo_url: res.logo_url }))
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
      setBranding((b: any) => ({ ...b, logo_url: '' }))
      toast.success('Logo removed')
    } catch (e: any) {
      toast.error(e?.message || 'Failed to remove logo')
    }
  }

  const updateBranding = async (fields: any) => {
    try {
      await API.patch('/admin/branding', fields)
      setBranding((b: any) => ({ ...b, ...fields }))
      toast.success('Appearance updated')
    } catch (e: any) {
      toast.error(e?.message || 'Branding update failed')
    }
  }

  const selectTab = (id: string) => {
    setSelected(id)
    if (id === 'appearance' || id === 'redirects') {
      setDraft(null)
    } else {
      const row = templates.find((t) => t.id === id)
      setDraft(row || null)
    }
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
    if (!draft || selected === 'appearance' || selected === 'redirects') return
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
      toast.success('Template published successfully')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const sendTest = async () => {
    if (selected === 'appearance' || selected === 'redirects') return
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
    if (selected === 'appearance' || selected === 'redirects') return
    if (!window.confirm('Reset this template to its default? All edits will be lost.')) return
    setResetting(true)
    try {
      const r = await API.post(`/admin/email-templates/${selected}/reset`, {})
      setDraft(r)
      setTemplates((prev) => prev.map((t) => (t.id === r.id ? r : t)))
      toast.success('Template reset to factory default')
    } catch (e: any) {
      toast.error(e?.message || 'Reset failed')
    } finally {
      setResetting(false)
    }
  }

  if (!branding) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <div className="faint" style={{ fontSize: 13 }}>Initializing workspace...</div>
      </div>
    )
  }

  const finalPreviewHTML = `
    <style>
      :root { color-scheme: ${previewMode}; }
      body { transition: background 0.2s ease-in-out; }
    </style>
    ${previewHTML}
  `

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden', background: 'var(--bg)' }}>
      {/* ── Sidebar ────────────────────────────────────────────── */}
      <div style={{ 
        width: 220, 
        borderRight: '1px solid var(--hairline)', 
        display: 'flex', 
        flexDirection: 'column',
        background: 'var(--surface-0)'
      }}>
        <div style={{ padding: '24px 20px 12px', fontSize: 10, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-dim)' }}>
          General
        </div>
        <div className="col" style={{ gap: 2, padding: '0 8px' }}>
          <NavButton id="appearance" label="Appearance" icon="Brand" active={selected === 'appearance'} onClick={selectTab} />
          <NavButton id="redirects" label="Redirects" icon="Globe" active={selected === 'redirects'} onClick={selectTab} />
        </div>

        <div style={{ padding: '24px 20px 12px', fontSize: 10, fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.1em', color: 'var(--fg-dim)' }}>
          Transactional
        </div>
        <div className="col" style={{ gap: 2, padding: '0 8px' }}>
          {templates.map(t => (
            <NavButton key={t.id} id={t.id} label={templateLabel(t.id)} icon="Mail" active={selected === t.id} onClick={selectTab} />
          ))}
        </div>
      </div>

      {/* ── Middle: Editor ─────────────────────────────────────────── */}
      <div style={{ 
        flex: '1', 
        minWidth: 400, 
        maxWidth: 500,
        display: 'flex', 
        flexDirection: 'column', 
        borderRight: '1px solid var(--hairline)',
        background: 'var(--bg)'
      }}>
        <div style={{ 
          padding: '20px 24px', 
          borderBottom: '1px solid var(--hairline)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          height: 64,
          flexShrink: 0
        }}>
          <div className="col">
            <h1 style={{ margin: 0, fontSize: 14, fontWeight: 600 }}>
              {selected === 'appearance' ? 'Appearance' : selected === 'redirects' ? 'Redirects' : templateLabel(selected)}
            </h1>
            <span style={{ fontSize: 11, color: 'var(--fg-dim)' }}>
              {selected === 'appearance' ? 'Brand identity settings' : selected === 'redirects' ? 'Callback URL configuration' : 'Email template editor'}
            </span>
          </div>
          {selected !== 'appearance' && selected !== 'redirects' && (
            <div className="row" style={{ gap: 8 }}>
              <button className="btn ghost sm" onClick={reset} disabled={resetting}>Reset</button>
              <button className="btn primary sm" onClick={save} disabled={saving}>{saving ? 'Saving...' : 'Publish'}</button>
            </div>
          )}
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: '24px' }} className="stack">
          {selected === 'appearance' && (
            <AppearanceEditor branding={branding} uploading={uploading} onUpload={uploadLogo} onRemove={removeLogo} onUpdate={updateBranding} />
          )}
          {selected === 'redirects' && <EmailRedirectConfig />}
          {draft && (
            <>
              <Field label="Subject line" sub="Admin-only reference for internal logs">
                <input type="text" className="mono" style={inputStyle} value={draft.subject || ''} onChange={e => patch({ subject: e.target.value })} />
              </Field>
              <Field label="Email Header">
                 <input type="text" style={inputStyle} value={draft.header_text || ''} onChange={e => patch({ header_text: e.target.value })} />
              </Field>
              <Field label="Preheader text">
                 <input type="text" style={inputStyle} value={draft.preheader || ''} onChange={e => patch({ preheader: e.target.value })} />
              </Field>
              <Field label="Body Paragraphs">
                <div className="col" style={{ gap: 12 }}>
                  {(draft.body_paragraphs || []).map((p, i) => (
                    <div key={i} style={{ position: 'relative' }}>
                      <textarea style={{ ...inputStyle, minHeight: 80, resize: 'vertical', paddingRight: 32 }} value={p} onChange={e => setParagraph(i, e.target.value)} />
                      <button onClick={() => removeParagraph(i)} style={{ position: 'absolute', top: 8, right: 8, color: 'var(--fg-dim)', fontSize: 16, width: 20, height: 20 }}>×</button>
                    </div>
                  ))}
                  <button className="btn ghost sm" style={{ alignSelf: 'flex-start' }} onClick={addParagraph}>+ Add paragraph</button>
                </div>
              </Field>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                 <Field label="CTA Button Label">
                    <input type="text" style={inputStyle} value={draft.cta_text || ''} onChange={e => patch({ cta_text: e.target.value })} />
                 </Field>
                 <Field label="Target Template">
                    <input type="text" className="mono" style={inputStyle} value={draft.cta_url_template || ''} onChange={e => patch({ cta_url_template: e.target.value })} placeholder="{{.Link}}" />
                 </Field>
              </div>
              <Field label="Footer Signature">
                 <textarea style={{ ...inputStyle, minHeight: 60, resize: 'vertical' }} value={draft.footer_text || ''} onChange={e => patch({ footer_text: e.target.value })} />
              </Field>
            </>
          )}
        </div>
      </div>

      {/* ── Right: Preview Stage ───────────────────────────────────── */}
      <div style={{ 
        flex: '1.2', 
        background: 'var(--surface-0)', 
        display: 'flex', 
        flexDirection: 'column',
        position: 'relative'
      }}>
        {selected !== 'redirects' ? (
          <>
            <div style={{ 
              position: 'absolute', top: 20, left: '50%', transform: 'translateX(-50%)', 
              zIndex: 10, display: 'flex', gap: 4, background: 'var(--surface-2)', 
              padding: 4, borderRadius: 6, border: '1px solid var(--hairline-strong)',
              boxShadow: 'var(--shadow-lg)'
            }}>
              <button className={`btn sm ${previewMode === 'light' ? 'primary' : 'ghost'}`} style={{ height: 24, fontSize: 10 }} onClick={() => setPreviewMode('light')}>LIGHT</button>
              <button className={`btn sm ${previewMode === 'dark' ? 'primary' : 'ghost'}`} style={{ height: 24, fontSize: 10 }} onClick={() => setPreviewMode('dark')}>DARK</button>
              {selected !== 'appearance' && (
                <>
                  <div style={{ width: 1, background: 'var(--hairline-strong)', margin: '4px 2px' }}/>
                  <button className="btn ghost sm" style={{ height: 24, fontSize: 10 }} onClick={sendTest} disabled={sending}>{sending ? '...' : 'SEND TEST'}</button>
                </>
              )}
            </div>
            <div style={{ 
              flex: 1, 
              display: 'flex', 
              alignItems: 'center', 
              justifyContent: 'center',
              padding: 40,
              backgroundSize: '24px 24px',
              backgroundImage: 'radial-gradient(var(--hairline-strong) 1px, transparent 0)',
            }}>
              <div style={{ 
                width: '100%', maxWidth: 600, height: '90%', 
                background: previewMode === 'light' ? '#fff' : '#000',
                borderRadius: 8, overflow: 'hidden', boxShadow: '0 40px 80px rgba(0,0,0,0.5)',
                border: '1px solid var(--hairline-bright)', transition: 'all 0.3s ease-in-out'
              }}>
                <iframe title="Preview" srcDoc={finalPreviewHTML} style={{ width: '100%', height: '100%', border: 'none' }} sandbox="allow-same-origin" />
              </div>
            </div>
          </>
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--surface-0)' }}>
            <div className="col" style={{ alignItems: 'center', gap: 16 }}>
               <Icon.Brand width={40} height={40} style={{ opacity: 0.1 }} />
               <div style={{ color: 'var(--fg-dim)', fontSize: 13 }}>Settings workspace active</div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function NavButton({ id, label, icon, active, onClick }) {
  const IconEl = Icon[icon] || Icon.Mail
  return (
    <button
      onClick={() => onClick(id)}
      className={`btn ghost ${active ? 'active' : ''}`}
      style={{ 
        justifyContent: 'flex-start', padding: '8px 12px', background: active ? 'var(--surface-2)' : 'transparent',
        color: active ? 'var(--fg)' : 'var(--fg-muted)', border: 'none', textAlign: 'left', height: 'auto',
        fontSize: 12.5, gap: 10
      }}
    >
      <IconEl width={14} height={14} style={{ opacity: active ? 1 : 0.6 }} />
      {label}
    </button>
  )
}

function AppearanceEditor({ branding, uploading, onUpload, onRemove, onUpdate }) {
  const [draft, setDraft] = React.useState(branding)
  const [saving, setSaving] = React.useState(false)

  React.useEffect(() => {
    setDraft(branding)
  }, [branding])

  const patch = (fields: any) => {
    setDraft((d: any) => ({ ...d, ...fields }))
  }

  const handleSave = async () => {
    setSaving(true)
    await onUpdate(draft)
    setSaving(false)
  }

  return (
    <div className="col" style={{ gap: 24 }}>
      <Field label="Brand Logo" sub="Displayed at the top of all transactional emails">
        <div className="row" style={{ gap: 16, alignItems: 'center', padding: 16, background: 'var(--surface-1)', borderRadius: 6, border: '1px solid var(--hairline)' }}>
          <div style={{ width: 64, height: 64, background: 'var(--surface-2)', borderRadius: 6, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', border: '1px solid var(--hairline-strong)' }}>
            {branding.logo_url ? (
              <img src={branding.logo_url} style={{ maxWidth: '80%', maxHeight: '80%', objectFit: 'contain' }} alt="Logo" />
            ) : (
              <Icon.Brand width={24} style={{ opacity: 0.2 }} />
            )}
          </div>
          <div className="col" style={{ gap: 8 }}>
            <div className="row" style={{ gap: 8 }}>
              <button className="btn sm primary" style={{ position: 'relative' }} disabled={uploading}>
                {uploading ? 'Uploading...' : 'Upload new'}
                <input type="file" accept="image/*" style={{ position: 'absolute', inset: 0, opacity: 0, cursor: 'pointer' }} onChange={e => {
                  if (e.target.files?.[0]) {
                    onUpload(e.target.files[0])
                  }
                  e.target.value = ''
                }} />
              </button>
              {branding.logo_url && <button className="btn sm ghost" onClick={onRemove}>Remove</button>}
            </div>
            <span style={{ fontSize: 11, color: 'var(--fg-dim)' }}>SVG, PNG, or JPG. Max 1MB.</span>
          </div>
        </div>
      </Field>
      <Field label="Primary Brand Color" sub="Used for action buttons and header highlights">
        <div className="row" style={{ gap: 12, alignItems: 'center' }}>
          <input type="color" style={{ width: 40, height: 40, padding: 0, border: 'none', background: 'none', cursor: 'pointer' }} value={draft.primary_color || '#000000'} onChange={e => patch({ primary_color: e.target.value })} />
          <input type="text" className="mono" style={{ ...inputStyle, width: 120 }} value={draft.primary_color || ''} onChange={e => patch({ primary_color: e.target.value })} />
        </div>
      </Field>
      <Field label="Email From Name" sub="The sender name displayed in user inboxes">
        <input type="text" style={inputStyle} value={draft.email_from_name || ''} placeholder="SharkAuth" onChange={e => patch({ email_from_name: e.target.value })} />
      </Field>
      <Field label="Email From Address" sub="The reply-to and sender address">
        <input type="email" style={inputStyle} value={draft.email_from_address || ''} placeholder="noreply@sharkauth.com" onChange={e => patch({ email_from_address: e.target.value })} />
      </Field>
      <button className="btn primary sm" style={{ alignSelf: 'flex-start', height: 32, padding: '0 20px' }} onClick={handleSave} disabled={saving}>
        {saving ? 'Saving...' : 'Save Appearance'}
      </button>
    </div>
  )
}

function EmailRedirectConfig() {
  const [cfg, setCfg] = React.useState({
    verify_redirect_url: '',
    reset_redirect_url: '',
    magic_link_redirect_url: '',
    invite_redirect_url: '',
  })
  const [saving, setSaving] = React.useState(false)
  const toast = useToast()

  React.useEffect(() => {
    API.get('/admin/email-config').then((r) => {
      setCfg({
        verify_redirect_url: r.verify_redirect_url || '',
        reset_redirect_url: r.reset_redirect_url || '',
        magic_link_redirect_url: r.magic_link_redirect_url || '',
        invite_redirect_url: r.invite_redirect_url || '',
      })
    }).catch(() => {})
  }, [])

  const save = async () => {
    setSaving(true)
    try {
      await API.patch('/admin/email-config', cfg)
      toast.success('Redirect URLs updated')
    } catch (e: any) {
      toast.error(e?.message || 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="col" style={{ gap: 24 }}>
      <Field label="Verify Email Redirect" sub="Users land here after confirming their email">
        <input type="url" style={inputStyle} value={cfg.verify_redirect_url} placeholder="https://app.com/verified" onChange={e => setCfg(c => ({ ...c, verify_redirect_url: e.target.value }))} spellCheck={false} />
      </Field>
      <Field label="Password Reset Redirect" sub="Where users are sent to choose a new password">
        <input type="url" style={inputStyle} value={cfg.reset_redirect_url} placeholder="https://app.com/reset-password" onChange={e => setCfg(c => ({ ...c, reset_redirect_url: e.target.value }))} spellCheck={false} />
      </Field>
      <Field label="Magic Link Redirect" sub="Default landing page after magic link sign-in">
        <input type="url" style={inputStyle} value={cfg.magic_link_redirect_url} placeholder="https://app.com/dashboard" onChange={e => setCfg(c => ({ ...c, magic_link_redirect_url: e.target.value }))} spellCheck={false} />
      </Field>
      <Field label="Organization Invite Redirect" sub="Where users accept their team invitations">
        <input type="url" style={inputStyle} value={cfg.invite_redirect_url} placeholder="https://app.com/invites" onChange={e => setCfg(c => ({ ...c, invite_redirect_url: e.target.value }))} spellCheck={false} />
      </Field>
      <button className="btn primary sm" style={{ alignSelf: 'flex-start', height: 32, padding: '0 20px' }} onClick={save} disabled={saving}>
        {saving ? 'Saving...' : 'Save Redirects'}
      </button>
    </div>
  )
}

function Field({ label, sub, children }) {
  return (
    <div className="col" style={{ gap: 6 }}>
      <div className="col" style={{ gap: 1 }}>
        <label style={{ fontSize: 10.5, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--fg-muted)' }}>{label}</label>
        {sub && <span style={{ fontSize: 10.5, color: 'var(--fg-dim)' }}>{sub}</span>}
      </div>
      {children}
    </div>
  )
}

const inputStyle = {
  width: '100%',
  background: 'var(--surface-1)',
  border: '1px solid var(--hairline-strong)',
  borderRadius: 4,
  padding: '10px 12px',
  color: 'var(--fg)',
  fontSize: 13,
  outline: 'none',
  transition: 'all 60ms ease-out'
}
