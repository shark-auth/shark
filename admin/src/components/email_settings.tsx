// @ts-nocheck
// Admin > Email — standalone page for email redirect URL configuration.
//
// Extracted from branding/email_tab.tsx (EmailRedirectConfig) and extended
// with a fourth field: org-invite redirect URL.
import React from 'react'
import { API } from './api'
import { useToast } from './toast'

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

function HelpTip({ text }: { text: string }) {
  return (
    <span
      title={text}
      style={{
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: 14, height: 14, borderRadius: '50%',
        border: '1px solid var(--fg-dim)', fontSize: 9,
        color: 'var(--fg-dim)', cursor: 'default', flexShrink: 0, lineHeight: 1,
      }}
    >?</span>
  )
}

export function EmailSettings() {
  const [cfg, setCfg] = React.useState({
    verify_redirect_url: '',
    reset_redirect_url: '',
    magic_link_redirect_url: '',
    invite_redirect_url: '',
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
          invite_redirect_url: r.invite_redirect_url || '',
        })
      })
      .catch(() => {})
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

  return (
    <div style={{ padding: '28px 32px', maxWidth: 640, display: 'flex', flexDirection: 'column', gap: 28 }}>
      {/* Page header */}
      <div>
        <h1 style={{ fontFamily: 'var(--font-sans)', fontSize: 18, fontWeight: 700, margin: '0 0 6px' }}>Email</h1>
        <p style={{ fontSize: 12.5, color: 'var(--fg-dim)', margin: 0, lineHeight: 1.5 }}>
          Configure callback URLs for magic-link, password-reset, email-verification, and org-invite emails.
          Leave blank to use the built-in hosted page (token-embedded fallback).
        </p>
      </div>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>

        {/* Verify-email redirect */}
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 5 }}>
            <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: 0 }}>
              Verify-email redirect
            </label>
            <HelpTip text="Shark appends ?token=<token> to this URL before sending. Your page reads the token and POSTs it to /api/v1/auth/verify-email to complete verification." />
          </div>
          <input
            type="url"
            value={cfg.verify_redirect_url}
            onChange={(e) => setCfg(c => ({ ...c, verify_redirect_url: e.target.value }))}
            onBlur={save}
            placeholder="https://yourapp.com/email-verified"
            style={redirectInputStyle}
            spellCheck={false}
          />
          <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
            Called after verify-email token is consumed. Token appended as <code style={{ fontSize: 10 }}>?token=...</code>
          </div>
        </div>

        {/* Password-reset redirect */}
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 5 }}>
            <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: 0 }}>
              Password-reset redirect
            </label>
            <HelpTip text="Shark appends ?token=<token> to this URL before sending. Your page collects the new password and POSTs {token, password} to /api/v1/auth/reset-password." />
          </div>
          <input
            type="url"
            value={cfg.reset_redirect_url}
            onChange={(e) => setCfg(c => ({ ...c, reset_redirect_url: e.target.value }))}
            onBlur={save}
            placeholder="https://yourapp.com/reset-password"
            style={redirectInputStyle}
            spellCheck={false}
          />
          <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
            Your page collects the new password and POSTs to <code style={{ fontSize: 10 }}>POST /auth/reset-password</code>
          </div>
        </div>

        {/* Magic-link redirect */}
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 5 }}>
            <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: 0 }}>
              Magic-link redirect (default)
            </label>
            <HelpTip text="Shark appends ?token=<token> to this URL before sending. The token is already verified server-side — redirect the user straight to their destination (dashboard, etc.)." />
          </div>
          <input
            type="url"
            value={cfg.magic_link_redirect_url}
            onChange={(e) => setCfg(c => ({ ...c, magic_link_redirect_url: e.target.value }))}
            onBlur={save}
            placeholder="https://yourapp.com/dashboard"
            style={redirectInputStyle}
            spellCheck={false}
          />
          <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
            Where to redirect after magic-link login. Per-request override via <code style={{ fontSize: 10 }}>redirect_uri</code> param.
          </div>
        </div>

        {/* Org-invite redirect */}
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 5 }}>
            <label style={{ display: 'block', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: 0 }}>
              Org-invite redirect
            </label>
            <HelpTip text="Base URL for org-invitation accept links. Shark appends /organizations/invitations/{token}/accept. Leave blank to use server.base_url." />
          </div>
          <input
            type="url"
            value={cfg.invite_redirect_url}
            onChange={(e) => setCfg(c => ({ ...c, invite_redirect_url: e.target.value }))}
            onBlur={save}
            placeholder="https://yourapp.com"
            style={redirectInputStyle}
            spellCheck={false}
          />
          <div style={{ fontSize: 10, color: 'var(--fg-faint)', marginTop: 3 }}>
            Overrides <code style={{ fontSize: 10 }}>server.base_url</code> for org invitation accept links.
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
    </div>
  )
}

export default EmailSettings
