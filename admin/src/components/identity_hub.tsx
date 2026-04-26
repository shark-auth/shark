// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { useAPI } from './api'

// ─── helpers ────────────────────────────────────────────────────────────────

function relTime(iso: string): string {
  if (!iso) return '—';
  const d = Date.parse(iso);
  if (isNaN(d)) return iso;
  const diff = (Date.now() - d) / 1000;
  if (diff < 60) return 'just now';
  if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
  if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
  return Math.floor(diff / 86400) + 'd ago';
}

function truncateMiddle(s: string, max = 24): string {
  if (!s || s.length <= max) return s || '—';
  const half = Math.floor((max - 3) / 2);
  return s.slice(0, half) + '…' + s.slice(-half);
}

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

// Section card wrapper — matches impeccable card style
function Section({ title, icon, children, actions }: {
  title: string; icon?: React.ReactNode; children: React.ReactNode; actions?: React.ReactNode;
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
        <span style={{ fontSize: 12, fontWeight: 600, letterSpacing: '0.02em' }}>{title}</span>
        <div style={{ flex: 1 }}/>
        {actions}
      </div>
      <div style={{ padding: '12px 14px' }}>
        {children}
      </div>
    </div>
  );
}

// Config row — label + input side by side
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

// Drawer component
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
        {/* header */}
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

// ─── main component ───────────────────────────────────────────────────────────

export function IdentityHub() {
  const { data: rawConfig, loading } = useAPI('/admin/config');

  // Form state — mirroring the shape returned by GET /api/v1/admin/config
  const [form, setForm] = React.useState<any>(null);
  const [original, setOriginal] = React.useState<any>(null);
  const [busy, setBusy] = React.useState(false);
  const [saveMsg, setSaveMsg] = React.useState('');

  // Active sessions state
  const [sessions, setSessions] = React.useState<any[]>([]);
  const [sessionsLoading, setSessionsLoading] = React.useState(false);
  const [sessionsTick, setSessionsTick] = React.useState(0);
  const [revokeConfirm, setRevokeConfirm] = React.useState(false);
  const [revoking, setRevoking] = React.useState<string | null>(null);

  // JWKS / signing keys
  const [keys, setKeys] = React.useState<any[]>([]);
  const [keysLoading, setKeysLoading] = React.useState(false);
  const [keyDrawer, setKeyDrawer] = React.useState<any>(null);

  // SSO connections
  const [ssoConns, setSsoConns] = React.useState<any[]>([]);
  const [ssoLoading, setSsoLoading] = React.useState(false);
  const [ssoTick, setSsoTick] = React.useState(0);
  const [ssoDrawer, setSsoDrawer] = React.useState<any>(null);
  const [deletingSSO, setDeletingSSO] = React.useState<string | null>(null);

  // Social provider drawer
  const [socialDrawer, setSocialDrawer] = React.useState<null | { provider: string; cfg: any }>(null);

  // OAuth device codes
  const [deviceCodes, setDeviceCodes] = React.useState<any[]>([]);
  const [dcLoading, setDcLoading] = React.useState(false);
  const [dcTick, setDcTick] = React.useState(0);
  const [dcActing, setDcActing] = React.useState<string | null>(null);

  const adminKey = localStorage.getItem('shark_admin_key') || '';
  const headers = { Authorization: `Bearer ${adminKey}`, 'Content-Type': 'application/json' };

  // ── shape helpers ──────────────────────────────────────────────────────────
  const toForm = (d: any) => ({
    auth: {
      password_min_length: d?.auth?.password_min_length ?? 8,
      session_lifetime:    d?.auth?.session_lifetime    ?? '24h',
      password_reset_ttl:  d?.auth?.password_reset_ttl  ?? '1h',
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
      google:  { client_id: d?.social?.google?.client_id  ?? '', client_secret: d?.social?.google?.client_secret  ?? '', callback: d?.social?.google?.callback  ?? '' },
      github:  { client_id: d?.social?.github?.client_id  ?? '', client_secret: d?.social?.github?.client_secret  ?? '', callback: d?.social?.github?.callback  ?? '' },
      apple:   { client_id: d?.social?.apple?.client_id   ?? '', client_secret: d?.social?.apple?.client_secret   ?? '', callback: d?.social?.apple?.callback   ?? '' },
      discord: { client_id: d?.social?.discord?.client_id ?? '', client_secret: d?.social?.discord?.client_secret ?? '', callback: d?.social?.discord?.callback ?? '' },
    },
    jwt: {
      lifetime: d?.jwt?.lifetime ?? '15m',
      issuer:   d?.jwt?.issuer   ?? '',
      audience: d?.jwt?.audience ?? '',
    },
    mfa: {
      enforcement:    d?.mfa?.enforcement    ?? 'optional',
      issuer:         d?.mfa?.issuer         ?? '',
      recovery_codes: d?.mfa?.recovery_codes ?? 10,
    },
    oauth_server: {
      enabled:                d?.oauth_server?.enabled                ?? true,
      issuer:                 d?.oauth_server?.issuer                 ?? '',
      access_token_lifetime:  d?.oauth_server?.access_token_lifetime  ?? '15m',
      refresh_token_lifetime: d?.oauth_server?.refresh_token_lifetime ?? '30d',
      require_dpop:           d?.oauth_server?.require_dpop           ?? false,
    },
  });

  React.useEffect(() => {
    if (!rawConfig) return;
    const f = toForm(rawConfig);
    setForm(f);
    setOriginal(JSON.stringify(f));
  }, [rawConfig]);

  // Active sessions fetch
  React.useEffect(() => {
    setSessionsLoading(true);
    fetch('/api/v1/admin/sessions?limit=50', { headers })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setSessions(j?.data || j?.sessions || []); setSessionsLoading(false); })
      .catch(() => { setSessions([]); setSessionsLoading(false); });
  }, [sessionsTick]);

  // JWKS / signing keys
  React.useEffect(() => {
    setKeysLoading(true);
    fetch('/.well-known/jwks.json')
      .then(r => r.ok ? r.json() : { keys: [] })
      .then(j => { setKeys(j?.keys || []); setKeysLoading(false); })
      .catch(() => { setKeys([]); setKeysLoading(false); });
  }, []);

  // SSO connections
  React.useEffect(() => {
    setSsoLoading(true);
    fetch('/api/v1/sso/connections', { headers })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setSsoConns(j?.data || j?.connections || []); setSsoLoading(false); })
      .catch(() => { setSsoConns([]); setSsoLoading(false); });
  }, [ssoTick]);

  // Device codes
  React.useEffect(() => {
    setDcLoading(true);
    fetch('/api/v1/admin/oauth/device-codes', { headers })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setDeviceCodes(j?.data || j?.codes || []); setDcLoading(false); })
      .catch(() => { setDeviceCodes([]); setDcLoading(false); });
  }, [dcTick]);

  // ── form setters ───────────────────────────────────────────────────────────
  const set = (path: string[], value: any) => {
    setForm(prev => {
      const next = JSON.parse(JSON.stringify(prev));
      let cur = next;
      for (let i = 0; i < path.length - 1; i++) cur = cur[path[i]];
      cur[path[path.length - 1]] = value;
      return next;
    });
  };

  const dirty = form && original !== JSON.stringify(form);

  // ── save ───────────────────────────────────────────────────────────────────
  const save = async () => {
    if (!form) return;
    setBusy(true);
    setSaveMsg('');
    try {
      const payload = {
        auth: {
          session_lifetime:    form.auth.session_lifetime,
          password_min_length: Number(form.auth.password_min_length),
          password_reset_ttl:  form.auth.password_reset_ttl,
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
        jwt: {
          lifetime: form.jwt.lifetime,
          issuer:   form.jwt.issuer,
          audience: form.jwt.audience,
        },
        social: form.social,
        mfa: {
          enforcement:    form.mfa.enforcement,
          issuer:         form.mfa.issuer,
          recovery_codes: Number(form.mfa.recovery_codes),
        },
        oauth_server: {
          enabled:                form.oauth_server.enabled,
          issuer:                 form.oauth_server.issuer,
          access_token_lifetime:  form.oauth_server.access_token_lifetime,
          refresh_token_lifetime: form.oauth_server.refresh_token_lifetime,
          require_dpop:           form.oauth_server.require_dpop,
        },
      };
      const r = await fetch('/api/v1/admin/config', {
        method: 'PUT',
        headers,
        body: JSON.stringify(payload),
      });
      if (!r.ok) {
        const err = await r.json().catch(() => ({}));
        setSaveMsg('Error: ' + (err?.error || r.status));
      } else {
        setOriginal(JSON.stringify(form));
        setSaveMsg('Saved');
        setTimeout(() => setSaveMsg(''), 2000);
      }
    } catch (e) {
      setSaveMsg('Network error');
    } finally {
      setBusy(false);
    }
  };

  // ── session actions ────────────────────────────────────────────────────────
  const revokeSession = async (id: string) => {
    setRevoking(id);
    await fetch(`/api/v1/admin/sessions/${id}`, { method: 'DELETE', headers }).catch(() => {});
    setRevoking(null);
    setSessionsTick(t => t + 1);
  };

  const revokeAll = async () => {
    await fetch('/api/v1/admin/sessions', { method: 'DELETE', headers }).catch(() => {});
    setRevokeConfirm(false);
    setSessionsTick(t => t + 1);
  };

  const purgeExpired = async () => {
    await fetch('/api/v1/admin/sessions/purge-expired', { method: 'POST', headers }).catch(() => {});
    setSessionsTick(t => t + 1);
  };

  // ── SSO actions ────────────────────────────────────────────────────────────
  const deleteSSOConn = async (id: string) => {
    setDeletingSSO(id);
    await fetch(`/api/v1/sso/connections/${id}`, { method: 'DELETE', headers }).catch(() => {});
    setDeletingSSO(null);
    setSsoDrawer(null);
    setSsoTick(t => t + 1);
  };

  // ── OAuth device code actions ──────────────────────────────────────────────
  const deviceCodeAction = async (userCode: string, action: 'approve' | 'deny') => {
    setDcActing(userCode);
    await fetch(`/api/v1/admin/oauth/device-codes/${userCode}/${action}`, { method: 'POST', headers }).catch(() => {});
    setDcActing(null);
    setDcTick(t => t + 1);
  };

  // ── loading / not ready ────────────────────────────────────────────────────
  if (loading || !form) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
        <span className="faint" style={{ fontSize: 12 }}>Loading identity…</span>
      </div>
    );
  }

  const socialProviders = [
    { key: 'google',  label: 'Google',  icon: '𝙂' },
    { key: 'github',  label: 'GitHub',  icon: '𝙂𝙃' },
    { key: 'apple',   label: 'Apple',   icon: '𝘼' },
    { key: 'discord', label: 'Discord', icon: '𝘿' },
  ] as const;

  const thStyle: React.CSSProperties = {
    padding: '8px 14px', fontSize: 10, fontWeight: 500,
    color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)',
    background: 'var(--surface-1)', textAlign: 'left',
  };
  const tdStyle: React.CSSProperties = {
    padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)',
  };

  return (
    <div style={{
      height: '100%', overflowY: 'auto',
      padding: '16px',
      display: 'flex', flexDirection: 'column', gap: 0,
    }}>
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
      <Section title="Authentication Methods" icon={<Icon.Lock width={13} height={13}/>}>
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
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5 }}>
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
      <Section title="Sessions & Tokens" icon={<Icon.Token width={13} height={13}/>}>
        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
            COOKIE SESSION
            <Chip label="always on" variant="dim"/>
          </div>
          <ConfigRow label="Session lifetime" sublabel="Duration before cookie sessions expire">
            <Inp value={form.auth.session_lifetime} onChange={v => set(['auth', 'session_lifetime'], v)} placeholder="24h"/>
          </ConfigRow>
        </div>

        <div style={{ marginBottom: 12 }}>
          <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em', marginBottom: 6, display: 'flex', alignItems: 'center', gap: 6 }}>
            JWT ACCESS TOKEN
            <Chip label="always on" variant="dim"/>
          </div>
          <ConfigRow label="Access token lifetime" sublabel="Duration of JWT access tokens">
            <Inp value={form.jwt.lifetime} onChange={v => set(['jwt', 'lifetime'], v)} placeholder="15m"/>
          </ConfigRow>
          <ConfigRow label="Issuer (iss)" sublabel="JWT iss claim value">
            <Inp value={form.jwt.issuer} onChange={v => set(['jwt', 'issuer'], v)} placeholder="https://auth.example.com"/>
          </ConfigRow>
          <ConfigRow label="Audience (aud)" sublabel="JWT aud claim value">
            <Inp value={form.jwt.audience} onChange={v => set(['jwt', 'audience'], v)} placeholder="https://api.example.com"/>
          </ConfigRow>
        </div>

        {/* Signing keys */}
        <div>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 8,
            marginBottom: 6,
          }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em' }}>SIGNING KEYS</div>
            <div style={{ flex: 1 }}/>
            <span className="faint" style={{ fontSize: 11 }}>{keys.length} key{keys.length !== 1 ? 's' : ''}</span>
          </div>
          {keysLoading ? (
            <span className="faint" style={{ fontSize: 11 }}>Loading…</span>
          ) : keys.length === 0 ? (
            <span className="faint" style={{ fontSize: 11 }}>No keys found in JWKS</span>
          ) : (
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5 }}>
              <thead>
                <tr>
                  <th style={thStyle}>KID</th>
                  <th style={thStyle}>Algorithm</th>
                  <th style={thStyle}>Use</th>
                  <th style={thStyle}>Status</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
                </tr>
              </thead>
              <tbody>
                {keys.map((k, i) => (
                  <tr key={k.kid || i}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={tdStyle}><span className="mono" style={{ fontSize: 10.5 }}>{truncateMiddle(k.kid, 24)}</span></td>
                    <td style={tdStyle}><span className="mono" style={{ fontSize: 11 }}>{k.alg || '—'}</span></td>
                    <td style={tdStyle}><span className="mono faint" style={{ fontSize: 11 }}>{k.use || '—'}</span></td>
                    <td style={tdStyle}><Chip label={i === 0 ? 'active' : 'rotated'} variant={i === 0 ? 'success' : 'dim'}/></td>
                    <td style={{ ...tdStyle, textAlign: 'right' }}>
                      <button className="btn ghost sm" onClick={() => setKeyDrawer(k)}>Inspect</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          3. Active Sessions
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="Active Sessions"
        icon={<Icon.Session width={13} height={13}/>}
        actions={
          <div className="row" style={{ gap: 6 }}>
            <button className="btn ghost sm" onClick={() => setSessionsTick(t => t + 1)}>
              <Icon.Refresh width={11} height={11}/>
              Refresh
            </button>
            <button className="btn ghost sm" onClick={purgeExpired}>
              Purge expired
            </button>
            {!revokeConfirm ? (
              <button className="btn ghost sm" style={{ color: 'var(--danger)' }} onClick={() => setRevokeConfirm(true)}>
                Revoke all
              </button>
            ) : (
              <div className="row" style={{ gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--danger)' }}>Revoke all?</span>
                <button className="btn ghost sm" onClick={() => setRevokeConfirm(false)}>Cancel</button>
                <button className="btn sm" style={{ background: 'var(--danger)', color: '#fff', border: 'none' }} onClick={revokeAll}>
                  Confirm
                </button>
              </div>
            )}
          </div>
        }
      >
        {sessionsLoading ? (
          <span className="faint" style={{ fontSize: 11 }}>Loading…</span>
        ) : sessions.length === 0 ? (
          <span className="faint" style={{ fontSize: 11 }}>No active sessions</span>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12.5 }}>
            <thead>
              <tr>
                <th style={thStyle}>User</th>
                <th style={thStyle}>Auth method</th>
                <th style={thStyle}>MFA</th>
                <th style={thStyle}>Created</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map(s => (
                <tr key={s.id || s.session_id}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td style={tdStyle}>
                    <div style={{ fontSize: 12.5 }}>{s.user_email || s.email || '—'}</div>
                  </td>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{ fontSize: 11 }}>{s.auth_method || s.method || '—'}</span>
                  </td>
                  <td style={tdStyle}>
                    {s.mfa_passed ? <Chip label="MFA" variant="success"/> : <span className="faint" style={{ fontSize: 11 }}>—</span>}
                  </td>
                  <td style={tdStyle}>
                    <span className="faint" style={{ fontSize: 11 }}>{relTime(s.created_at || s.created)}</span>
                  </td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>
                    <button
                      className="btn ghost sm"
                      style={{ color: 'var(--danger)' }}
                      disabled={revoking === (s.id || s.session_id)}
                      onClick={() => revokeSession(s.id || s.session_id)}
                    >
                      {revoking === (s.id || s.session_id) ? '…' : 'Revoke'}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          4. MFA
      ═══════════════════════════════════════════════════════════════════ */}
      <Section title="Multi-Factor Authentication" icon={<Icon.Shield width={13} height={13}/>}>
        <ConfigRow label="Enforcement" sublabel="Whether MFA is required for users">
          <Sel value={form.mfa.enforcement} onChange={v => set(['mfa', 'enforcement'], v)} options={[
            { value: 'off',      label: 'Off' },
            { value: 'optional', label: 'Optional' },
            { value: 'required', label: 'Required' },
          ]}/>
        </ConfigRow>
        <ConfigRow label="TOTP issuer" sublabel="Displayed in authenticator apps">
          <Inp value={form.mfa.issuer} onChange={v => set(['mfa', 'issuer'], v)} placeholder="MyApp"/>
        </ConfigRow>
        <ConfigRow label="Recovery code count" sublabel="Backup codes generated per user">
          <Inp value={form.mfa.recovery_codes} onChange={v => set(['mfa', 'recovery_codes'], v)} type="number" style={{ width: 80 }}/>
        </ConfigRow>
      </Section>

      {/* ═══════════════════════════════════════════════════════════════════
          5. SSO Connections
      ═══════════════════════════════════════════════════════════════════ */}
      <Section
        title="SSO Connections"
        icon={<Icon.SSO width={13} height={13}/>}
        actions={
          <button className="btn ghost sm" onClick={() => setSsoTick(t => t + 1)}>
            <Icon.Refresh width={11} height={11}/>
            Refresh
          </button>
        }
      >
        {ssoLoading ? (
          <span className="faint" style={{ fontSize: 11 }}>Loading…</span>
        ) : ssoConns.length === 0 ? (
          <div style={{ padding: '12px 0' }}>
            <div className="faint" style={{ fontSize: 12, marginBottom: 6 }}>No SSO connections configured.</div>
            <div className="faint" style={{ fontSize: 11 }}>Create connections via the Admin API: <span className="mono" style={{ fontSize: 10.5 }}>POST /api/v1/admin/sso/connections</span></div>
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
              {ssoConns.map(conn => (
                <tr key={conn.id}
                  style={{ cursor: 'pointer' }}
                  onClick={() => setSsoDrawer(conn)}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td style={tdStyle}>
                    <div style={{ fontSize: 12.5 }}>{conn.name || conn.id}</div>
                    <div className="mono faint" style={{ fontSize: 10 }}>{truncateMiddle(conn.id, 24)}</div>
                  </td>
                  <td style={tdStyle}><span className="mono faint" style={{ fontSize: 11 }}>{conn.type || conn.provider || '—'}</span></td>
                  <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{conn.domain || '—'}</span></td>
                  <td style={tdStyle}><Chip label={conn.enabled !== false ? 'active' : 'disabled'} variant={conn.enabled !== false ? 'success' : 'dim'}/></td>
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
          6. OAuth Server
      ═══════════════════════════════════════════════════════════════════ */}
      <Section title="OAuth Server" icon={<Icon.Globe width={13} height={13}/>}>
        <ConfigRow label="Enabled">
          <Sel value={form.oauth_server.enabled ? 'true' : 'false'} onChange={v => set(['oauth_server', 'enabled'], v === 'true')} options={[{ value: 'true', label: 'On' }, { value: 'false', label: 'Off' }]}/>
        </ConfigRow>
        <ConfigRow label="Issuer URL" sublabel="Authorization server issuer (iss)">
          <Inp value={form.oauth_server.issuer} onChange={v => set(['oauth_server', 'issuer'], v)} placeholder="https://auth.example.com" style={{ width: 240 }}/>
        </ConfigRow>
        <ConfigRow label="Access token lifetime">
          <Inp value={form.oauth_server.access_token_lifetime} onChange={v => set(['oauth_server', 'access_token_lifetime'], v)} placeholder="15m"/>
        </ConfigRow>
        <ConfigRow label="Refresh token lifetime">
          <Inp value={form.oauth_server.refresh_token_lifetime} onChange={v => set(['oauth_server', 'refresh_token_lifetime'], v)} placeholder="30d"/>
        </ConfigRow>
        <ConfigRow label="Require DPoP" sublabel="Enforce Demonstrating Proof of Possession">
          <Sel value={form.oauth_server.require_dpop ? 'true' : 'false'} onChange={v => set(['oauth_server', 'require_dpop'], v === 'true')} options={[{ value: 'false', label: 'Off' }, { value: 'true', label: 'Required' }]}/>
        </ConfigRow>

        {/* Device code queue */}
        <div style={{ marginTop: 14 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
            <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', letterSpacing: '0.08em' }}>DEVICE CODE APPROVAL QUEUE</div>
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
                  <th style={thStyle}>Expires</th>
                  <th style={{ ...thStyle, textAlign: 'right' }}>Action</th>
                </tr>
              </thead>
              <tbody>
                {deviceCodes.map(dc => (
                  <tr key={dc.user_code}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={tdStyle}><span className="mono" style={{ fontSize: 11 }}>{dc.user_code}</span></td>
                    <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{dc.client_id ? truncateMiddle(dc.client_id, 18) : '—'}</span></td>
                    <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{Array.isArray(dc.scopes) ? dc.scopes.join(' ') : (dc.scope || '—')}</span></td>
                    <td style={tdStyle}><span className="faint" style={{ fontSize: 11 }}>{relTime(dc.expires_at)}</span></td>
                    <td style={{ ...tdStyle, textAlign: 'right' }}>
                      <div className="row" style={{ gap: 4, justifyContent: 'flex-end' }}>
                        <button
                          className="btn ghost sm"
                          disabled={dcActing === dc.user_code}
                          onClick={() => deviceCodeAction(dc.user_code, 'approve')}
                        >
                          Approve
                        </button>
                        <button
                          className="btn ghost sm"
                          style={{ color: 'var(--danger)' }}
                          disabled={dcActing === dc.user_code}
                          onClick={() => deviceCodeAction(dc.user_code, 'deny')}
                        >
                          Deny
                        </button>
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

      {/* Key inspect drawer */}
      <Drawer open={!!keyDrawer} onClose={() => setKeyDrawer(null)} title="Signing Key">
        {keyDrawer && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {Object.entries(keyDrawer).map(([k, v]) => (
              <div key={k}>
                <div style={{ fontSize: 10, color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 2, letterSpacing: '0.06em' }}>{k.toUpperCase()}</div>
                <div className="row" style={{ gap: 6 }}>
                  <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{String(v)}</span>
                  {String(v).length > 6 && <CopyField value={String(v)}/>}
                </div>
              </div>
            ))}
          </div>
        )}
      </Drawer>

      {/* SSO inspect drawer */}
      <Drawer open={!!ssoDrawer} onClose={() => setSsoDrawer(null)} title={ssoDrawer?.name || 'SSO Connection'}>
        {ssoDrawer && (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {Object.entries(ssoDrawer).map(([k, v]) => (
              <div key={k}>
                <div style={{ fontSize: 10, color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 2, letterSpacing: '0.06em' }}>{k.toUpperCase()}</div>
                <div className="row" style={{ gap: 6 }}>
                  <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{String(v)}</span>
                  {String(v).length > 6 && <CopyField value={String(v)}/>}
                </div>
              </div>
            ))}
            <div style={{ marginTop: 8, paddingTop: 12, borderTop: '1px solid var(--hairline)' }}>
              <button
                className="btn ghost sm"
                style={{ color: 'var(--danger)' }}
                disabled={deletingSSO === ssoDrawer.id}
                onClick={() => deleteSSOConn(ssoDrawer.id)}
              >
                {deletingSSO === ssoDrawer.id ? 'Deleting…' : 'Delete connection'}
              </button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}

export default IdentityHub;
