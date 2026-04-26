// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { useAPI } from './api'

// IdentityHub — WHO can authenticate.
// Scope: Authentication methods only (Password, Magic Link, Passkeys, Social providers).
// Sessions, Tokens, SSO, OAuth Server, MFA enforcement → Settings tab (source of truth).

// ─── helpers ────────────────────────────────────────────────────────────────

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

  // Form state — only auth-method fields (password, magic link, passkeys, social)
  const [form, setForm] = React.useState<any>(null);
  const [original, setOriginal] = React.useState<any>(null);
  const [busy, setBusy] = React.useState(false);
  const [saveMsg, setSaveMsg] = React.useState('');

  // Social provider drawer
  const [socialDrawer, setSocialDrawer] = React.useState<null | { provider: string; cfg: any }>(null);

  const adminKey = localStorage.getItem('shark_admin_key') || '';
  const headers = { Authorization: `Bearer ${adminKey}`, 'Content-Type': 'application/json' };

  // ── shape helpers ──────────────────────────────────────────────────────────
  // Only auth-method fields — Sessions/Tokens/OAuth/SSO/MFA live in Settings.
  const toForm = (d: any) => ({
    auth: {
      password_min_length: d?.auth?.password_min_length ?? 8,
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
  });

  React.useEffect(() => {
    if (!rawConfig) return;
    const f = toForm(rawConfig);
    setForm(f);
    setOriginal(JSON.stringify(f));
  }, [rawConfig]);

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
        social: form.social,
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

  // ── loading / not ready ────────────────────────────────────────────────────
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
          1. Authentication Methods — WHO can authenticate (source of truth here)
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
          Cross-link: Sessions, Tokens, OAuth Server, SSO, MFA → Settings
          (source of truth for HOW the system operates)
      ═══════════════════════════════════════════════════════════════════ */}
      <div style={{
        padding: '10px 14px',
        border: '1px solid var(--hairline)',
        borderRadius: 5,
        background: 'var(--surface-1)',
        display: 'flex', alignItems: 'center', gap: 12,
        marginBottom: 12,
      }}>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 12.5, fontWeight: 500, color: 'var(--fg)', marginBottom: 3 }}>
            Sessions, Tokens, OAuth Server, SSO &amp; MFA
          </div>
          <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            Lifetimes, signing keys, DPoP enforcement, SSO connections, and MFA policy
            are configured in <strong>Settings</strong> (source of truth for system operation).
          </div>
        </div>
        <a href="/admin/settings" className="btn ghost sm" style={{ textDecoration: 'none', flexShrink: 0 }}>
          Open Settings →
        </a>
      </div>

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
    </div>
  );
}

export default IdentityHub;
