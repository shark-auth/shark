// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'

// IdentityHub — auth-flow policy + methods.
// Owns: HOW users authenticate. Password / magic link / passkeys / social,
// session + JWT lifetimes, OAuth authorization server, MFA enforcement,
// SSO connections, device-code approval queue.
// Server identity, CORS, email, audit retention, maintenance, danger zone
// → Settings tab.

// ─── helpers ────────────────────────────────────────────────────────────────

function truncateMiddle(s: string, max = 24): string {
  if (!s || s.length <= max) return s || '—';
  const half = Math.floor((max - 3) / 2);
  return s.slice(0, half) + '…' + s.slice(-half);
}

function deepClone(o) { return JSON.parse(JSON.stringify(o ?? {})); }

function Chip({ label, variant }: { label: string; variant?: 'success' | 'warn' | 'danger' | 'dim' }) {
  const colors: Record<string, string> = {
    success: 'var(--success, #22c55e)',
    warn: 'var(--warn, #f59e0b)',
    danger: 'var(--danger)',
    dim: 'var(--fg-dim)',
  };
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center',
      height: 16, padding: '0 5px', borderRadius: 3,
      fontSize: 10, fontWeight: 500, letterSpacing: '0.04em',
      background: 'var(--surface-2)',
      color: variant ? colors[variant] : 'var(--fg-dim)',
      border: '1px solid var(--hairline)',
    }}>{label}</span>
  );
}

function Section({ title, desc, icon, children, actions }: {
  title: string; desc?: string; icon?: React.ReactNode; children: React.ReactNode; actions?: React.ReactNode;
}) {
  return (
    <div style={{
      background: 'var(--surface-1)',
      border: '1px solid var(--hairline)',
      borderRadius: 5,
      marginBottom: 12,
    }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8,
        padding: '10px 14px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        borderRadius: '5px 5px 0 0',
      }}>
        {icon && <span style={{ color: 'var(--fg-dim)' }}>{icon}</span>}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 12, fontWeight: 600, letterSpacing: '0.02em' }}>{title}</div>
          {desc && <div className="faint" style={{ fontSize: 11, lineHeight: 1.4, marginTop: 2 }}>{desc}</div>}
        </div>
        {actions}
      </div>
      <div style={{ padding: '12px 14px' }}>
        {children}
      </div>
    </div>
  );
}

function ConfigRow({ label, sublabel, children }: {
  label: string; sublabel?: string; children: React.ReactNode;
}) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center',
      padding: '7px 0',
      borderBottom: '1px solid var(--hairline)',
      gap: 12,
      minHeight: 36,
    }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 12.5, fontWeight: 500 }}>{label}</div>
        {sublabel && <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 1 }}>{sublabel}</div>}
      </div>
      <div style={{ flexShrink: 0 }}>
        {children}
      </div>
    </div>
  );
}

function Inp({ value, onChange, type = 'text', placeholder = '', style = {} }: any) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      style={{
        height: 28, padding: '0 8px', fontSize: 12,
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        background: 'var(--surface-0)',
        color: 'var(--fg)',
        width: 180,
        ...style,
      }}
    />
  );
}

function Sel({ value, onChange, options }: { value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }) {
  return (
    <select
      value={value ?? ''}
      onChange={e => onChange(e.target.value)}
      style={{
        height: 28, padding: '0 8px', fontSize: 12,
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        background: 'var(--surface-0)',
        color: 'var(--fg)',
        width: 180,
      }}
    >
      {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
    </select>
  );
}

function Drawer({ open, onClose, title, width = 420, children }: {
  open: boolean; onClose: () => void; title: string; width?: number; children: React.ReactNode;
}) {
  React.useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [open]);

  if (!open) return null;
  return (
    <>
      <div onClick={onClose} style={{
        position: 'fixed', inset: 0, zIndex: 49,
        background: 'rgba(0,0,0,0.18)',
      }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0,
        width, zIndex: 50,
        background: 'var(--surface-0)',
        borderLeft: '1px solid var(--hairline)',
        display: 'flex', flexDirection: 'column',
        boxShadow: '-8px 0 24px rgba(0,0,0,0.08)',
      }}>
        <div style={{
          display: 'flex', alignItems: 'center',
          padding: '0 16px', height: 48,
          borderBottom: '1px solid var(--hairline)',
          flexShrink: 0,
          gap: 10,
        }}>
          <span style={{ fontWeight: 600, fontSize: 13, flex: 1 }}>{title}</span>
          <button className="btn ghost icon sm" onClick={onClose}>
            <Icon.X width={12} height={12}/>
          </button>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '14px 16px' }}>
          {children}
        </div>
      </div>
    </>
  );
}

const thStyle: React.CSSProperties = {
  padding: '8px 14px', fontSize: 10, fontWeight: 500,
  color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-1)', textAlign: 'left',
};
const tdStyle: React.CSSProperties = {
  padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)',
};

// ─── main component ───────────────────────────────────────────────────────────

export function IdentityHub() {
  const { data: rawConfig, loading, refresh } = useAPI('/admin/config');
  const toast = useToast();

  const [form, setForm] = React.useState<any>(null);
  const [original, setOriginal] = React.useState<any>(null);
  const [busy, setBusy] = React.useState(false);
  const [saveMsg, setSaveMsg] = React.useState('');

  const [socialDrawer, setSocialDrawer] = React.useState<null | { provider: string; cfg: any }>(null);

  const adminKey = typeof localStorage !== 'undefined' ? localStorage.getItem('shark_admin_key') || '' : '';
  const headers = { Authorization: `Bearer ${adminKey}`, 'Content-Type': 'application/json' };

  // SSO + device-code state (live via direct fetch — separate endpoints)
  const [ssoConns, setSsoConns] = React.useState<any[]>([]);
  const [ssoLoading, setSsoLoading] = React.useState(false);
  const [ssoTick, setSsoTick] = React.useState(0);
  const [ssoDrawer, setSsoDrawer] = React.useState<any>(null);
  const [deletingSSO, setDeletingSSO] = React.useState<any>(null);

  const [deviceCodes, setDeviceCodes] = React.useState<any[]>([]);
  const [dcLoading, setDcLoading] = React.useState(false);
  const [dcTick, setDcTick] = React.useState(0);
  const [dcActing, setDcActing] = React.useState<any>(null);

  React.useEffect(() => {
    setSsoLoading(true);
    fetch('/api/v1/sso/connections', { headers })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setSsoConns(j?.data || j?.connections || []); setSsoLoading(false); })
      .catch(() => { setSsoConns([]); setSsoLoading(false); });
  }, [ssoTick]);

  React.useEffect(() => {
    setDcLoading(true);
    fetch('/api/v1/admin/oauth/device-codes', { headers })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setDeviceCodes(j?.data || j?.codes || []); setDcLoading(false); })
      .catch(() => { setDeviceCodes([]); setDcLoading(false); });
  }, [dcTick]);

  const deleteSSOConn = async (id: string) => {
    setDeletingSSO(id);
    await fetch(`/api/v1/sso/connections/${id}`, { method: 'DELETE', headers }).catch(() => {});
    setDeletingSSO(null);
    setSsoDrawer(null);
    setSsoTick(t => t + 1);
  };

  const deviceCodeAction = async (userCode: string, action: string) => {
    setDcActing(userCode);
    await fetch(`/api/v1/admin/oauth/device-codes/${userCode}/${action}`, { method: 'POST', headers }).catch(() => {});
    setDcActing(null);
    setDcTick(t => t + 1);
  };

  // ── shape helpers ──────────────────────────────────────────────────────────
  const toForm = (d: any) => ({
    auth: {
      session_lifetime:    d?.auth?.session_lifetime    ?? '30d',
      password_min_length: d?.auth?.password_min_length ?? 8,
      password_reset_ttl:  d?.auth?.password_reset_ttl  ?? d?.password_reset?.ttl ?? '1h',
    },
    jwt: {
      lifetime: d?.jwt?.lifetime || d?.jwt?.access_token_ttl || '15m',
    },
    magic_link: {
      enabled: d?.magic_link?.ttl ? true : (d?.magic_link?.enabled ?? false),
      ttl:     d?.magic_link?.ttl     ?? '15m',
    },
    passkey: {
      rp_name:           d?.passkeys?.rp_name           ?? d?.passkey?.rp_name           ?? '',
      rp_id:             d?.passkeys?.rp_id             ?? d?.passkey?.rp_id             ?? '',
      user_verification: d?.passkeys?.user_verification ?? d?.passkey?.user_verification ?? 'preferred',
    },
    social: {
      redirect_url: d?.social?.redirect_url ?? '',
      google:  { client_id: d?.social?.google?.client_id  ?? '', client_secret: d?.social?.google?.client_secret  ?? '', callback: d?.social?.google?.callback  ?? '' },
      github:  { client_id: d?.social?.github?.client_id  ?? '', client_secret: d?.social?.github?.client_secret  ?? '', callback: d?.social?.github?.callback  ?? '' },
      apple:   { client_id: d?.social?.apple?.client_id   ?? '', client_secret: d?.social?.apple?.client_secret   ?? '', callback: d?.social?.apple?.callback   ?? '' },
      discord: { client_id: d?.social?.discord?.client_id ?? '', client_secret: d?.social?.discord?.client_secret ?? '', callback: d?.social?.discord?.callback ?? '' },
    },
    oauth_server: {
      enabled:                d?.oauth_server?.enabled                ?? true,
      issuer:                 d?.oauth_server?.issuer                 ?? '',
      access_token_lifetime:  d?.oauth_server?.access_token_lifetime  ?? '15m',
      refresh_token_lifetime: d?.oauth_server?.refresh_token_lifetime ?? '30d',
      require_dpop:           d?.oauth_server?.require_dpop           ?? false,
    },
    mfa: {
      enforcement:    d?.mfa?.enforcement    ?? 'optional',
      issuer:         d?.mfa?.issuer         ?? '',
      recovery_codes: d?.mfa?.recovery_codes ?? 10,
    },
  });

  React.useEffect(() => {
    if (!rawConfig) return;
    const f = toForm(rawConfig);
    setForm(f);
    setOriginal(JSON.stringify(f));
  }, [rawConfig]);

  const set = (path: string[], value: any) => {
    setForm(prev => {
      const next = deepClone(prev);
      let cur = next;
      for (let i = 0; i < path.length - 1; i++) cur = cur[path[i]];
      cur[path[path.length - 1]] = value;
      return next;
    });
  };

  const dirty = form && original !== JSON.stringify(form);

  const save = async () => {
    if (!form) return;
    setBusy(true);
    setSaveMsg('');
    try {
      const payload: any = {
        auth: {
          session_lifetime:    form.auth.session_lifetime,
          password_min_length: Number(form.auth.password_min_length),
          password_reset_ttl:  form.auth.password_reset_ttl,
        },
        jwt: {
          lifetime: form.jwt.lifetime,
        },
        passkeys: {
          rp_name:           form.passkey.rp_name,
          rp_id:             form.passkey.rp_id,
          user_verification: form.passkey.user_verification,
        },
        magic_link: {
          enabled: form.magic_link.enabled,
          ttl:     form.magic_link.ttl,
        },
        social: form.social,
        oauth_server: form.oauth_server,
        mfa: {
          enforcement:    form.mfa.enforcement,
          issuer:         form.mfa.issuer,
          recovery_codes: Number(form.mfa.recovery_codes),
        },
      };
      if (!payload.social.redirect_url) delete payload.social.redirect_url;
      const r = await fetch('/api/v1/admin/config', {
        method: 'PUT',
        headers,
        body: JSON.stringify(payload),
      });
      if (!r.ok) {
        const err = await r.json().catch(() => ({}));
        setSaveMsg('Error: ' + (err?.error || r.status));
        toast?.error?.(err?.error || `HTTP ${r.status}`);
      } else {
        setOriginal(JSON.stringify(form));
        setSaveMsg('Saved');
        toast?.success?.('Auth config saved');
        refresh && refresh();
        setTimeout(() => setSaveMsg(''), 2000);
      }
    } catch (e: any) {
      setSaveMsg('Network error');
      toast?.error?.(e?.message || 'Save failed');
    } finally {
      setBusy(false);
    }
  };

  if (loading || !form) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <span className="faint" style={{ fontSize: 12 }}>Loading identity…</span>
      </div>
    );
  }

  const socialProviders = [
    { key: 'google',  label: 'Google' },
    { key: 'github',  label: 'GitHub' },
    { key: 'apple',   label: 'Apple' },
    { key: 'discord', label: 'Discord' },
  ] as const;

  return (
    <div style={{
      height: '100%', overflowY: 'auto',
      padding: '16px',
      display: 'flex', flexDirection: 'column', gap: 0,
    }}>
      {/* ───── header ───── */}
      <div style={{ marginBottom: 14, padding: '0 2px' }}>
        <div style={{ fontSize: 14, fontWeight: 600 }}>Identity</div>
        <div className="faint" style={{ fontSize: 11.5, lineHeight: 1.5, marginTop: 3 }}>
          Auth-flow policy and methods — how users authenticate, how sessions and tokens behave,
          OAuth/SSO/MFA. For server identity, CORS, email, audit, and operator controls, see <strong>System</strong>.
        </div>
      </div>

      {/* ───── sticky save bar ───── */}
      {dirty && (
        <div style={{
          position: 'sticky', top: 0, zIndex: 20,
          background: 'var(--surface-0)',
          borderBottom: '1px solid var(--hairline)',
          padding: '8px 14px',
          display: 'flex', alignItems: 'center', gap: 10,
          marginBottom: 12, borderRadius: 5,
          border: '1px solid var(--hairline-strong)',
        }}>
          <span className="faint" style={{ fontSize: 12, flex: 1 }}>You have unsaved changes</span>
          {saveMsg && <span style={{ fontSize: 12, color: saveMsg.startsWith('Error') ? 'var(--danger)' : 'var(--fg-dim)' }}>{saveMsg}</span>}
          <button className="btn ghost sm" onClick={() => { const f = toForm(rawConfig); setForm(f); setOriginal(JSON.stringify(f)); }}>
            Discard
          </button>
          <button className="btn sm" onClick={save} disabled={busy}>
            {busy ? 'Saving…' : 'Save changes'}
          </button>
        </div>
      )}

      {/* ═══════════════════════════════════════════════════════════════════
          1. Authentication Methods
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="Authentication Methods"
        desc="Which credentials users can sign in with"
        icon={<Icon.Lock width={13} height={13}/>}
      >
        {/* Password */}
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
            PASSWORD
            <Chip label="always on" variant="dim"/>
          </div>
          <ConfigRow label="Minimum length" sublabel="Characters required at sign-up">
            <Inp value={form.auth.password_min_length} onChange={v => set(['auth', 'password_min_length'], v)} type="number" style={{ width: 80 }}/>
          </ConfigRow>
          <ConfigRow label="Reset link TTL" sublabel="How long a password-reset email link stays valid">
            <Inp value={form.auth.password_reset_ttl} onChange={v => set(['auth', 'password_reset_ttl'], v)} placeholder="1h"/>
          </ConfigRow>
        </div>

        {/* Magic Link */}
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
            MAGIC LINK
            <Chip label={form.magic_link.enabled ? 'enabled' : 'disabled'} variant={form.magic_link.enabled ? 'success' : 'dim'}/>
          </div>
          <ConfigRow label="Enabled">
            <Sel value={form.magic_link.enabled ? 'true' : 'false'} onChange={v => set(['magic_link', 'enabled'], v === 'true')} options={[{ value: 'true', label: 'On' }, { value: 'false', label: 'Off' }]}/>
          </ConfigRow>
          <ConfigRow label="Token TTL" sublabel="Expiry for magic link tokens">
            <Inp value={form.magic_link.ttl} onChange={v => set(['magic_link', 'ttl'], v)} placeholder="15m"/>
          </ConfigRow>
        </div>

        {/* Passkeys */}
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6 }}>PASSKEYS (WEBAUTHN)</div>
          <ConfigRow label="Relying Party name" sublabel="Displayed to users during registration">
            <Inp value={form.passkey.rp_name} onChange={v => set(['passkey', 'rp_name'], v)} placeholder="My App"/>
          </ConfigRow>
          <ConfigRow label="Relying Party ID" sublabel="Domain (e.g. example.com)">
            <Inp value={form.passkey.rp_id} onChange={v => set(['passkey', 'rp_id'], v)} placeholder="example.com"/>
          </ConfigRow>
          <ConfigRow label="User verification">
            <Sel value={form.passkey.user_verification} onChange={v => set(['passkey', 'user_verification'], v)} options={[
              { value: 'required',    label: 'Required' },
              { value: 'preferred',   label: 'Preferred' },
              { value: 'discouraged', label: 'Discouraged' },
            ]}/>
          </ConfigRow>
        </div>

        {/* Social providers */}
        <div>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6 }}>SOCIAL PROVIDERS</div>
          <ConfigRow label="Post-OAuth redirect URL" sublabel="Where users land after social sign-in (blank = server default)">
            <Inp value={form.social.redirect_url} onChange={v => set(['social', 'redirect_url'], v)} placeholder="/dashboard"/>
          </ConfigRow>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5, marginTop: 8 }}>
            <thead>
              <tr>
                <th style={thStyle}>Provider</th>
                <th style={thStyle}>Client ID</th>
                <th style={thStyle}>Status</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
              </tr>
            </thead>
            <tbody>
              {socialProviders.map(({ key, label }) => {
                const cfg = form.social[key];
                const on = !!cfg.client_id;
                return (
                  <tr key={key}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setSocialDrawer({ provider: key, cfg: { ...cfg } })}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={tdStyle}>{label}</td>
                    <td style={tdStyle}>
                      {cfg.client_id
                        ? <span className="mono faint" style={{ fontSize: 10.5 }}>{truncateMiddle(cfg.client_id, 28)}</span>
                        : <span className="faint" style={{ fontSize: 11 }}>—</span>}
                    </td>
                    <td style={tdStyle}><Chip label={on ? 'configured' : 'not set'} variant={on ? 'success' : 'dim'}/></td>
                    <td style={{ ...tdStyle, textAlign: 'right' }}>
                      <button className="btn ghost sm" onClick={e => { e.stopPropagation(); setSocialDrawer({ provider: key, cfg: { ...cfg } }); }}>
                        Edit
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          2. Sessions & Tokens
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="Sessions & Tokens"
        desc="How long a sign-in stays valid and how short-lived bearer tokens behave"
        icon={<Icon.Shield width={13} height={13}/>}
      >
        <ConfigRow label="Cookie session lifetime" sublabel="How long a sign-in remains valid · accepts 30m, 24h, 30d">
          <Inp value={form.auth.session_lifetime} onChange={v => set(['auth', 'session_lifetime'], v)} placeholder="30d" style={{ width: 140 }}/>
        </ConfigRow>
        <ConfigRow label="JWT access token lifetime" sublabel="Short-lived bearer tokens for SDKs and APIs">
          <Inp value={form.jwt.lifetime} onChange={v => set(['jwt', 'lifetime'], v)} placeholder="15m" style={{ width: 140 }}/>
        </ConfigRow>
        <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 8, lineHeight: 1.5 }}>
          Signing-key rotation lives in <a href="/admin/settings" style={{ color: 'var(--fg)' }}>System → Server</a>.
        </div>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          3. OAuth Authorization Server
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="OAuth Authorization Server"
        desc="Issuer, token lifetimes, and DPoP enforcement for OAuth-based clients"
      >
        <ConfigRow label="Enabled">
          <Sel value={form.oauth_server.enabled ? 'true' : 'false'}
            onChange={v => set(['oauth_server', 'enabled'], v === 'true')}
            options={[{ value: 'true', label: 'On' }, { value: 'false', label: 'Off' }]}/>
        </ConfigRow>
        <ConfigRow label="Issuer URL" sublabel="Authorization server issuer (iss claim)">
          <Inp value={form.oauth_server.issuer}
            onChange={v => set(['oauth_server', 'issuer'], v)}
            placeholder="https://auth.example.com" style={{ width: 260 }}/>
        </ConfigRow>
        <ConfigRow label="Access token lifetime">
          <Inp value={form.oauth_server.access_token_lifetime}
            onChange={v => set(['oauth_server', 'access_token_lifetime'], v)}
            placeholder="15m" style={{ width: 140 }}/>
        </ConfigRow>
        <ConfigRow label="Refresh token lifetime">
          <Inp value={form.oauth_server.refresh_token_lifetime}
            onChange={v => set(['oauth_server', 'refresh_token_lifetime'], v)}
            placeholder="30d" style={{ width: 140 }}/>
        </ConfigRow>
        <ConfigRow label="Require DPoP" sublabel="Demonstrating Proof of Possession on all tokens">
          <Sel value={form.oauth_server.require_dpop ? 'true' : 'false'}
            onChange={v => set(['oauth_server', 'require_dpop'], v === 'true')}
            options={[{ value: 'false', label: 'Off' }, { value: 'true', label: 'Required' }]}/>
        </ConfigRow>

        {/* Device code queue */}
        <div style={{ marginTop: 14 }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8,
          }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', textTransform: 'uppercase' }}>Device Code Queue</div>
            <div style={{ flex: 1 }}/>
            <button className="btn ghost sm" onClick={() => setDcTick(t => t + 1)}>
              <Icon.Refresh width={11} height={11}/>
            </button>
          </div>
          {dcLoading ? (
            <span className="faint" style={{ fontSize: 11 }}>Loading…</span>
          ) : deviceCodes.length === 0 ? (
            <span className="faint" style={{ fontSize: 11 }}>No pending device codes</span>
          ) : (
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5 }}>
              <thead>
                <tr>
                  <th style={thStyle}>User code</th>
                  <th style={thStyle}>Client</th>
                  <th style={thStyle}>Scopes</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
                </tr>
              </thead>
              <tbody>
                {deviceCodes.map((dc: any) => (
                  <tr key={dc.user_code}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={tdStyle}><span className="mono" style={{ fontSize: 11 }}>{dc.user_code}</span></td>
                    <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{dc.client_id ? dc.client_id.slice(0, 18) + (dc.client_id.length > 18 ? '…' : '') : '—'}</span></td>
                    <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{Array.isArray(dc.scopes) ? dc.scopes.join(' ') : (dc.scope || '—')}</span></td>
                    <td style={{ ...tdStyle, textAlign: 'right' }}>
                      <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
                        <button className="btn ghost sm" disabled={dcActing === dc.user_code}
                          onClick={() => deviceCodeAction(dc.user_code, 'approve')}>Approve</button>
                        <button className="btn ghost sm" style={{ color: 'var(--danger)' }}
                          disabled={dcActing === dc.user_code}
                          onClick={() => deviceCodeAction(dc.user_code, 'deny')}>Deny</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          4. MFA Enforcement
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="MFA Enforcement"
        desc="Whether multi-factor is optional, required, or off — plus TOTP issuer label and recovery code count"
      >
        <ConfigRow label="Enforcement" sublabel="Whether MFA is required for admin users">
          <Sel value={form.mfa.enforcement}
            onChange={v => set(['mfa', 'enforcement'], v)}
            options={[
              { value: 'off',      label: 'Off' },
              { value: 'optional', label: 'Optional' },
              { value: 'required', label: 'Required' },
            ]}/>
        </ConfigRow>
        <ConfigRow label="TOTP issuer" sublabel="Shown in authenticator apps">
          <Inp value={form.mfa.issuer}
            onChange={v => set(['mfa', 'issuer'], v)}
            placeholder="MyApp"/>
        </ConfigRow>
        <ConfigRow label="Recovery code count" sublabel="Backup codes per user">
          <Inp value={form.mfa.recovery_codes} type="number"
            onChange={v => set(['mfa', 'recovery_codes'], v)}
            placeholder="10" style={{ width: 80 }}/>
        </ConfigRow>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          5. SSO Connections
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="SSO Connections"
        desc="External identity providers that federate sign-in (SAML, OIDC)"
        actions={(
          <button className="btn ghost sm" onClick={() => setSsoTick(t => t + 1)}>
            <Icon.Refresh width={11} height={11}/>
            Refresh
          </button>
        )}
      >
        {ssoLoading ? (
          <span className="faint" style={{ fontSize: 11 }}>Loading…</span>
        ) : ssoConns.length === 0 ? (
          <div>
            <div className="faint" style={{ fontSize: 12, marginBottom: 4 }}>No SSO connections configured.</div>
            <div className="faint" style={{ fontSize: 11 }}>
              Create via Admin API: <span className="mono" style={{ fontSize: 10.5 }}>POST /api/v1/admin/sso/connections</span>
            </div>
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5 }}>
            <thead>
              <tr>
                <th style={thStyle}>Name / ID</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Domain</th>
                <th style={thStyle}>Status</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
              </tr>
            </thead>
            <tbody>
              {ssoConns.map((conn: any) => (
                <tr key={conn.id}
                  style={{ cursor: 'pointer' }}
                  onClick={() => setSsoDrawer(conn)}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td style={tdStyle}>
                    <div>{conn.name || conn.id}</div>
                    <div className="mono faint" style={{ fontSize: 10 }}>{conn.id ? conn.id.slice(0, 24) : ''}</div>
                  </td>
                  <td style={tdStyle}><span className="mono faint" style={{ fontSize: 11 }}>{conn.type || conn.provider || '—'}</span></td>
                  <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{conn.domain || '—'}</span></td>
                  <td style={tdStyle}>
                    <Chip label={conn.enabled !== false ? 'active' : 'disabled'} variant={conn.enabled !== false ? 'success' : 'dim'}/>
                  </td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>
                    <button className="btn ghost sm" onClick={e => { e.stopPropagation(); setSsoDrawer(conn); }}>Inspect</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          Drawers
      ═══════════════════════════════════════════════════════════════════ */}

      {/* Social provider drawer */}
      <Drawer
        open={!!socialDrawer}
        onClose={() => setSocialDrawer(null)}
        title={socialDrawer ? `${socialDrawer.provider.charAt(0).toUpperCase() + socialDrawer.provider.slice(1)} OAuth` : ''}
      >
        {socialDrawer && (() => {
          const pd = { ...socialDrawer };
          const update = (field: string, val: string) => {
            setSocialDrawer(prev => ({ ...prev, cfg: { ...prev.cfg, [field]: val } }));
          };
          return (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Client ID</div>
                <div className="row" style={{ gap: 6 }}>
                  <input
                    value={pd.cfg.client_id || ''}
                    onChange={e => update('client_id', e.target.value)}
                    placeholder="client_id"
                    style={{ flex: 1, height: 28, padding: '0 8px', fontSize: 12, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)', color: 'var(--fg)' }}
                  />
                  {pd.cfg.client_id && <CopyField value={pd.cfg.client_id}/>}
                </div>
              </div>
              <div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Client Secret</div>
                <input
                  type="password"
                  value={pd.cfg.client_secret || ''}
                  onChange={e => update('client_secret', e.target.value)}
                  placeholder="client_secret"
                  style={{ width: '100%', height: 28, padding: '0 8px', fontSize: 12, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)', color: 'var(--fg)', boxSizing: 'border-box' }}
                />
              </div>
              <div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 4 }}>Callback URL</div>
                <div className="row" style={{ gap: 6 }}>
                  <input
                    value={pd.cfg.callback || `${window.location.origin}/api/v1/auth/social/${pd.provider}/callback`}
                    onChange={e => update('callback', e.target.value)}
                    placeholder="https://..."
                    style={{ flex: 1, height: 28, padding: '0 8px', fontSize: 12, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-0)', color: 'var(--fg)' }}
                  />
                  <CopyField value={pd.cfg.callback || `${window.location.origin}/api/v1/auth/social/${pd.provider}/callback`}/>
                </div>
              </div>
              <div style={{ display: 'flex', gap: 8, marginTop: 4 }}>
                <button className="btn ghost sm" onClick={() => setSocialDrawer(null)}>Cancel</button>
                <button className="btn sm" onClick={() => {
                  set(['social', pd.provider], pd.cfg);
                  setSocialDrawer(null);
                }}>Apply</button>
              </div>
            </div>
          );
        })()}
      </Drawer>

      {/* SSO inspect drawer */}
      <Drawer
        open={!!ssoDrawer}
        onClose={() => setSsoDrawer(null)}
        title={ssoDrawer ? (ssoDrawer.name || 'SSO Connection') : ''}
      >
        {ssoDrawer && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {Object.entries(ssoDrawer).map(([k, v]) => (
              <div key={k}>
                <div style={{ fontSize: 10, color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 2, letterSpacing: '0.06em', textTransform: 'uppercase' }}>{k}</div>
                <span className="mono" style={{ fontSize: 11, wordBreak: 'break-all', color: 'var(--fg)' }}>{String(v)}</span>
              </div>
            ))}
            <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
              <button className="btn sm ghost" style={{ color: 'var(--danger)', borderColor: 'var(--danger)' }}
                disabled={deletingSSO === ssoDrawer.id}
                onClick={() => deleteSSOConn(ssoDrawer.id)}>
                {deletingSSO === ssoDrawer.id ? 'Deleting…' : 'Delete connection'}
              </button>
              <button className="btn sm ghost" onClick={() => setSsoDrawer(null)}>Close</button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}

export default IdentityHub;
