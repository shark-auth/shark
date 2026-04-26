// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { CLIFooter } from './CLIFooter'
import { usePageActions } from './useKeyboardShortcuts'

// Settings — global config command center.
// Sibling page to users.tsx: dense, monochrome, hairline borders, 13px base,
// editable inputs bound to GET/PATCH /admin/config, drawer for confirm/test
// flows. No YAML, no read-only "info" sections, no JWT/session mode toggle.

// ─── Section registry ────────────────────────────────────────────────────────
const SECTIONS = [
  { id: 'server',       label: 'Server',         desc: 'Base URL, CORS, port' },
  { id: 'tokens',       label: 'Sessions & Tokens', desc: 'Cookie + JWT lifetimes, signing key' },
  { id: 'auth_policy',  label: 'Auth Policy',    desc: 'Password rules, magic links, password reset' },
  { id: 'oauth_sso',    label: 'OAuth & SSO',    desc: 'OAuth server, SSO connections, MFA enforcement' },
  { id: 'email',        label: 'Email Delivery', desc: 'Provider, sender, test send' },
  { id: 'oauth',        label: 'OAuth Providers', desc: 'Per-provider credentials' },
  { id: 'audit',        label: 'Audit & Data',   desc: 'Retention, purge, export' },
  { id: 'maintenance',  label: 'Maintenance',    desc: 'Sessions + audit cleanup' },
];

// ─── Helpers ─────────────────────────────────────────────────────────────────
function deepClone(o) { return JSON.parse(JSON.stringify(o ?? {})); }
function fmtUptime(s) {
  if (!s || s < 0) return '—';
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}
function fingerprint(s) {
  if (!s) return '—';
  return s.length > 12 ? s.slice(0, 6) + '…' + s.slice(-4) : s;
}

// ─── Atoms ───────────────────────────────────────────────────────────────────
function Field({ label, hint, children, span = 1 }) {
  return (
    <div style={{ gridColumn: `span ${span}` }}>
      {label && (
        <div style={{
          fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
          color: 'var(--fg-muted)', marginBottom: 4, lineHeight: 1.5,
        }}>{label}</div>
      )}
      {children}
      {hint && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>{hint}</div>}
    </div>
  );
}

function Input({ value, onChange, placeholder, mono, type = 'text', width, disabled }) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={(e) => onChange && onChange(type === 'number' ? (e.target.value === '' ? '' : Number(e.target.value)) : e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      style={{
        width: width || '100%', height: 28, padding: '0 8px',
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        fontSize: 13, color: 'var(--fg)',
        fontFamily: mono ? 'var(--font-mono)' : 'inherit',
        opacity: disabled ? 0.55 : 1,
      }}
    />
  );
}

function Select({ value, onChange, options, width }) {
  return (
    <div style={{ position: 'relative', display: 'inline-flex', width: width || '100%' }}>
      <select
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        style={{
          appearance: 'none', WebkitAppearance: 'none',
          width: '100%', height: 28, padding: '0 26px 0 8px',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 4, color: 'var(--fg)',
          fontSize: 13, cursor: 'pointer', colorScheme: 'dark',
        }}
      >
        {options.map((o) => <option key={o.v} value={o.v}>{o.l}</option>)}
      </select>
      <Icon.ChevronDown width={10} height={10} style={{
        position: 'absolute', right: 8, top: '50%', transform: 'translateY(-50%)',
        pointerEvents: 'none', opacity: 0.5,
      }}/>
    </div>
  );
}

function MaskedInput({ value, onChange, placeholder, hasValue }) {
  const [shown, setShown] = React.useState(false);
  return (
    <div style={{ position: 'relative' }}>
      <input
        type={shown ? 'text' : 'password'}
        value={value ?? ''}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete="new-password"
        style={{
          width: '100%', height: 28, padding: '0 56px 0 8px',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 4, fontSize: 13, color: 'var(--fg)',
          fontFamily: 'var(--font-mono)',
        }}
      />
      <button type="button" onClick={() => setShown(s => !s)} style={{
        position: 'absolute', right: 4, top: '50%', transform: 'translateY(-50%)',
        height: 22, padding: '0 6px', background: 'transparent', border: 'none',
        color: 'var(--fg-muted)', fontSize: 11, cursor: 'pointer',
      }}>{shown ? 'Hide' : 'Show'}</button>
    </div>
  );
}

// Section block — anchored target for left-rail nav.
function Section({ id, title, desc, children, onSection }) {
  const ref = React.useRef(null);
  React.useEffect(() => {
    if (!ref.current || !onSection) return;
    const obs = new IntersectionObserver(
      (entries) => {
        const visible = entries.find(e => e.isIntersecting);
        if (visible) onSection(id);
      },
      { rootMargin: '-30% 0px -60% 0px', threshold: 0 }
    );
    obs.observe(ref.current);
    return () => obs.disconnect();
  }, [id, onSection]);
  return (
    <section ref={ref} id={id} style={{
      borderBottom: '1px solid var(--hairline)',
      padding: '20px 24px',
    }}>
      <div style={{ marginBottom: 14 }}>
        <div style={{
          fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
          color: 'var(--fg-muted)', fontWeight: 600,
        }}>{title}</div>
        {desc && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>{desc}</div>}
      </div>
      {children}
    </section>
  );
}

// Tag list — used by CORS origins. Compact, hairline, "+ add" inline input.
function TagList({ values, onChange, placeholder = 'https://example.com' }) {
  const [draft, setDraft] = React.useState('');
  const list = Array.isArray(values) ? values : [];
  const add = () => {
    const v = draft.trim();
    if (!v) return;
    if (list.includes(v)) { setDraft(''); return; }
    onChange([...list, v]);
    setDraft('');
  };
  const remove = (i) => onChange(list.filter((_, ix) => ix !== i));
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {list.length === 0 && (
        <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>No origins · all cross-origin requests blocked</span>
      )}
      {list.map((v, i) => (
        <div key={i} className="row" style={{
          gap: 6, padding: '4px 8px',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline)',
          borderRadius: 4,
        }}>
          <span className="mono" style={{ fontSize: 12, flex: 1, color: 'var(--fg)' }}>{v}</span>
          <button type="button"
            onClick={() => remove(i)}
            style={{
              height: 20, padding: '0 6px', background: 'transparent',
              border: 'none', color: 'var(--fg-muted)', fontSize: 11, cursor: 'pointer',
            }}>
            <Icon.X width={10} height={10}/>
          </button>
        </div>
      ))}
      <div className="row" style={{ gap: 6 }}>
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); add(); } }}
          placeholder={placeholder}
          style={{
            flex: 1, height: 28, padding: '0 8px',
            background: 'var(--surface-1)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4, fontSize: 13, color: 'var(--fg)',
            fontFamily: 'var(--font-mono)',
          }}
        />
        <button type="button" className="btn sm" onClick={add} disabled={!draft.trim()}>
          <Icon.Plus width={10} height={10}/>Add
        </button>
      </div>
    </div>
  );
}

// Drawer — right-side fixed panel. Used for Test email + confirm purge.
// NEVER a modal — backdrop dims, panel slides from right, Esc closes.
function Drawer({ title, subtitle, onClose, children, footer, width = 480 }) {
  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);
  return (
    <>
      <div
        onClick={onClose}
        style={{ position: 'fixed', inset: 0, zIndex: 40, background: 'rgba(0,0,0,0.45)' }}
      />
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          position: 'fixed', top: 0, right: 0, bottom: 0,
          width, maxWidth: '100vw',
          zIndex: 41,
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          display: 'flex', flexDirection: 'column',
          animation: 'slideIn 140ms ease-out',
        }}
      >
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 8 }}>
            <button className="btn ghost sm" onClick={onClose}>
              <Icon.X width={12} height={12}/>Close
            </button>
          </div>
          <div style={{
            fontSize: 18, fontWeight: 500, letterSpacing: '-0.01em',
            fontFamily: 'var(--font-display)', lineHeight: 1.2,
          }}>{title}</div>
          {subtitle && <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>{subtitle}</div>}
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>{children}</div>
        {footer && (
          <div style={{
            padding: 12, borderTop: '1px solid var(--hairline)',
            display: 'flex', justifyContent: 'flex-end', gap: 8,
            background: 'var(--surface-0)',
          }}>{footer}</div>
        )}
      </div>
    </>
  );
}

// ─── Main ────────────────────────────────────────────────────────────────────
export function Settings() {
  const { data: health } = useAPI('/admin/health');
  const { data: config, loading, refresh } = useAPI('/admin/config');
  const { data: modeData } = useAPI('/admin/system/mode');
  const toast = useToast();

  const [form, setForm] = React.useState(null);
  const [saving, setSaving] = React.useState(false);
  const [active, setActive] = React.useState('server');
  const [drawer, setDrawer] = React.useState(null);
  const [rotating, setRotating] = React.useState(false);

  // OAuth & SSO section state (moved from Identity Hub — source of truth here)
  const adminKey = typeof localStorage !== 'undefined' ? localStorage.getItem('shark_admin_key') || '' : '';
  const authHeaders = { Authorization: `Bearer ${adminKey}`, 'Content-Type': 'application/json' };

  const [ssoConns, setSsoConns] = React.useState([]);
  const [ssoLoading, setSsoLoading] = React.useState(false);
  const [ssoTick, setSsoTick] = React.useState(0);
  const [ssoDrawer, setSsoDrawer] = React.useState(null);
  const [deletingSSO, setDeletingSSO] = React.useState(null);

  const [deviceCodes, setDeviceCodes] = React.useState([]);
  const [dcLoading, setDcLoading] = React.useState(false);
  const [dcTick, setDcTick] = React.useState(0);
  const [dcActing, setDcActing] = React.useState(null);

  // SSO fetch
  React.useEffect(() => {
    setSsoLoading(true);
    fetch('/api/v1/sso/connections', { headers: authHeaders })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setSsoConns(j?.data || j?.connections || []); setSsoLoading(false); })
      .catch(() => { setSsoConns([]); setSsoLoading(false); });
  }, [ssoTick]);

  // Device codes fetch
  React.useEffect(() => {
    setDcLoading(true);
    fetch('/api/v1/admin/oauth/device-codes', { headers: authHeaders })
      .then(r => r.ok ? r.json() : { data: [] })
      .then(j => { setDeviceCodes(j?.data || j?.codes || []); setDcLoading(false); })
      .catch(() => { setDeviceCodes([]); setDcLoading(false); });
  }, [dcTick]);

  const deleteSSOConn = async (id) => {
    setDeletingSSO(id);
    await fetch(`/api/v1/sso/connections/${id}`, { method: 'DELETE', headers: authHeaders }).catch(() => {});
    setDeletingSSO(null);
    setSsoDrawer(null);
    setSsoTick(t => t + 1);
  };

  const deviceCodeAction = async (userCode, action) => {
    setDcActing(userCode);
    await fetch(`/api/v1/admin/oauth/device-codes/${userCode}/${action}`, { method: 'POST', headers: authHeaders }).catch(() => {});
    setDcActing(null);
    setDcTick(t => t + 1);
  };

  // Initialize editable form from config payload. Sentinels:
  //  - email.api_key === '********' means "set, masked".
  React.useEffect(() => {
    if (!config) return;
    setForm({
      server: {
        base_url:     config.server?.base_url ?? config.base_url ?? '',
        port:         config.server?.port ?? '',
        cors_origins: config.server?.cors_origins ?? config.cors_origins ?? [],
        cors_relaxed: config.server?.cors_relaxed ?? false,
      },
      auth: {
        session_lifetime:    config.auth?.session_lifetime || '30d',
        password_min_length: config.auth?.password_min_length ?? 8,
      },
      jwt: {
        // NO mode toggle. Cookie + JWT both always on.
        lifetime: config.jwt?.lifetime || config.jwt?.access_token_ttl || '15m',
      },
      magic_link: {
        ttl: config.magic_link?.ttl || '15m',
      },
      password_reset: {
        ttl: config.password_reset?.ttl || '1h',
      },
      social: {
        redirect_url: config.social?.redirect_url || '',
      },
      email: {
        provider:          config.email?.provider || 'shark',
        api_key:           config.email?.api_key || '',
        from:              config.email?.from || '',
        from_name:         config.email?.from_name || '',
        previous_provider: config.email?.previous_provider || '',
      },
      audit: {
        retention:        config.audit?.retention || '30d',
        cleanup_interval: config.audit?.cleanup_interval || '1h',
      },
      oauth_server: {
        enabled:                config.oauth_server?.enabled                ?? true,
        issuer:                 config.oauth_server?.issuer                 ?? '',
        access_token_lifetime:  config.oauth_server?.access_token_lifetime  ?? '15m',
        refresh_token_lifetime: config.oauth_server?.refresh_token_lifetime ?? '30d',
        require_dpop:           config.oauth_server?.require_dpop           ?? false,
      },
      mfa: {
        enforcement:    config.mfa?.enforcement    ?? 'optional',
        issuer:         config.mfa?.issuer         ?? '',
        recovery_codes: config.mfa?.recovery_codes ?? 10,
      },
    });
  }, [config]);

  const set = (path, value) => {
    setForm(prev => {
      const next = deepClone(prev);
      const parts = path.split('.');
      let cur = next;
      for (let i = 0; i < parts.length - 1; i++) {
        cur[parts[i]] = cur[parts[i]] || {};
        cur = cur[parts[i]];
      }
      cur[parts[parts.length - 1]] = value;
      return next;
    });
  };

  const isDirty = React.useMemo(() => {
    if (!form || !config) return false;
    const baseline = {
      server: {
        base_url: config.server?.base_url ?? config.base_url ?? '',
        port: config.server?.port ?? '',
        cors_origins: config.server?.cors_origins ?? config.cors_origins ?? [],
        cors_relaxed: config.server?.cors_relaxed ?? false,
      },
      auth:           { session_lifetime: config.auth?.session_lifetime || '30d', password_min_length: config.auth?.password_min_length ?? 8 },
      jwt:            { lifetime: config.jwt?.lifetime || config.jwt?.access_token_ttl || '15m' },
      magic_link:     { ttl: config.magic_link?.ttl || '15m' },
      password_reset: { ttl: config.password_reset?.ttl || '1h' },
      social:         { redirect_url: config.social?.redirect_url || '' },
      email:  {
        provider:          config.email?.provider || 'shark',
        api_key:           config.email?.api_key || '',
        from:              config.email?.from || '',
        from_name:         config.email?.from_name || '',
        previous_provider: config.email?.previous_provider || '',
      },
      audit:  {
        retention: config.audit?.retention || '30d',
        cleanup_interval: config.audit?.cleanup_interval || '1h',
      },
      oauth_server: {
        enabled:                config.oauth_server?.enabled                ?? true,
        issuer:                 config.oauth_server?.issuer                 ?? '',
        access_token_lifetime:  config.oauth_server?.access_token_lifetime  ?? '15m',
        refresh_token_lifetime: config.oauth_server?.refresh_token_lifetime ?? '30d',
        require_dpop:           config.oauth_server?.require_dpop           ?? false,
      },
      mfa: {
        enforcement:    config.mfa?.enforcement    ?? 'optional',
        issuer:         config.mfa?.issuer         ?? '',
        recovery_codes: config.mfa?.recovery_codes ?? 10,
      },
    };
    return JSON.stringify(baseline) !== JSON.stringify(form);
  }, [form, config]);

  const handleSave = async () => {
    if (!form || saving) return;
    setSaving(true);
    try {
      const payload = deepClone(form);
      // Don't echo the masked sentinel back — backend interprets unchanged.
      if (payload.email?.api_key === '********') delete payload.email.api_key;
      // Remove empty social.redirect_url to avoid overwriting with blank.
      if (payload.social?.redirect_url === '') delete payload.social;
      // Coerce numeric fields
      if (payload.mfa?.recovery_codes) payload.mfa.recovery_codes = Number(payload.mfa.recovery_codes);
      await API.patch('/admin/config', payload);
      toast.success('Configuration saved');
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const handleDiscard = () => { if (config) { setForm(null); refresh(); } };

  const handleRotate = async () => {
    if (rotating) return;
    if (!confirm('Rotate JWT signing key? Existing tokens remain valid until expiry.')) return;
    setRotating(true);
    try {
      const res = await API.post('/admin/auth/rotate-signing-key', {});
      toast.success(`Signing key rotated · kid ${res?.kid ? res.kid.slice(0, 8) : 'new'}`);
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Rotation failed');
    } finally {
      setRotating(false);
    }
  };

  // r refreshes
  usePageActions({ onRefresh: refresh });

  // Anchor scroll on left-rail click. ScrollIntoView smooth.
  const goTo = (id) => {
    setActive(id);
    const el = document.getElementById(id);
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  if (loading || !form) {
    return (
      <div style={{ padding: 32, color: 'var(--fg-muted)', fontSize: 13 }}>Loading configuration…</div>
    );
  }

  // Status strip data — derived only, never editable.
  const version = health?.version || 'dev';
  const uptime  = fmtUptime(health?.uptime_seconds);
  const dbDriver = health?.db?.driver || 'sqlite';
  const dbSize   = health?.db?.size_mb != null ? `${health.db.size_mb.toFixed(1)} MB` : '—';
  const jwtKid   = config.jwt?.active_kid || config.jwt?.kid || '';
  const activeKeys = health?.jwt?.active_keys ?? config.jwt?.active_keys ?? 0;

  // Email config "set" — sentinel from server: api_key="********" means present.
  const emailKeySet = (config.email?.api_key || '') !== '';

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {/* ── Left rail nav ── */}
      <aside style={{
        width: 200, flexShrink: 0,
        borderRight: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        padding: '12px 0',
        overflowY: 'auto',
      }}>
        <div style={{ padding: '0 16px 8px', fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', fontWeight: 600 }}>
          Settings
        </div>
        {SECTIONS.map(s => {
          const on = active === s.id;
          return (
            <button key={s.id} onClick={() => goTo(s.id)} style={{
              display: 'block', width: '100%', textAlign: 'left',
              padding: '7px 16px', fontSize: 13,
              color: on ? 'var(--fg)' : 'var(--fg-muted)',
              fontWeight: on ? 500 : 400,
              borderLeft: on ? '2px solid var(--fg)' : '2px solid transparent',
              cursor: 'pointer',
              background: on ? 'var(--surface-1)' : 'transparent',
            }}>
              {s.label}
            </button>
          );
        })}
      </aside>

      {/* ── Right pane ── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, position: 'relative' }}>
        {/* Status strip — derived facts only, NOT a section */}
        <div style={{
          padding: '8px 24px',
          borderBottom: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          display: 'flex', alignItems: 'center', gap: 16,
          fontSize: 11, lineHeight: 1.5,
        }}>
          <span className="row" style={{ gap: 6 }}>
            <span className={'dot' + (health?.status === 'ok' ? ' success' : '')}/>
            <span style={{ color: 'var(--fg-muted)' }}>v</span>
            <span className="mono" style={{ color: 'var(--fg)' }}>{version}</span>
          </span>
          <span className="faint">·</span>
          <span><span style={{ color: 'var(--fg-muted)' }}>uptime </span><span className="mono">{uptime}</span></span>
          <span className="faint">·</span>
          <span><span style={{ color: 'var(--fg-muted)' }}>{dbDriver} </span><span className="mono">{dbSize}</span></span>
          <span className="faint">·</span>
          <span><span style={{ color: 'var(--fg-muted)' }}>migration </span><span className="mono">{health?.migrations?.current ?? '—'}</span></span>
          <div style={{ flex: 1 }}/>
          <span className="faint mono">GET /admin/config</span>
        </div>

        {/* Sections — anchored scroll */}
        <div style={{ flex: 1, overflowY: 'auto', paddingBottom: 80 }}>
          {/* Server */}
          <Section id="server" title="Server" desc="HTTP listener, public base URL, allowed origins" onSection={setActive}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="Base URL" hint="Public URL clients use to reach this server" span={2}>
                <Input value={form.server.base_url}
                  onChange={(v) => set('server.base_url', v)}
                  mono
                  placeholder="https://auth.example.com"/>
              </Field>
              <Field label="Port" hint="HTTP listener port (restart required)">
                <Input value={form.server.port} type="number" mono
                  onChange={(v) => set('server.port', v)} placeholder="8080" width={120}/>
              </Field>
              <Field label="Environment">
                <Input value={config.environment || (config.dev_mode ? 'development' : 'production')} disabled/>
              </Field>
              <Field label="CORS origins" hint="Browser origins permitted to call the API · empty = same-origin only" span={2}>
                <TagList values={form.server.cors_origins} onChange={(v) => set('server.cors_origins', v)}/>
              </Field>
              <Field label="Relaxed CORS" hint="" span={2}>
                <div className="row" style={{ gap: 10, alignItems: 'center' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', userSelect: 'none' }}>
                    <input
                      type="checkbox"
                      checked={!!form.server.cors_relaxed}
                      onChange={(e) => set('server.cors_relaxed', e.target.checked)}
                      style={{ width: 14, height: 14, accentColor: 'var(--danger)' }}
                    />
                    <span style={{ fontSize: 12 }}>Accept any origin (<code>*</code>)</span>
                  </label>
                  <span style={{ fontSize: 11, color: 'var(--danger)' }}>
                    Insecure — for local development only
                  </span>
                </div>
              </Field>
            </div>
          </Section>

          {/* Sessions & Tokens — single section. Cookie + JWT both ON, no mode toggle. */}
          <Section id="tokens" title="Sessions & Tokens" desc="Cookie sessions and JWT access tokens are both always active. Configure lifetimes and rotate the signing key." onSection={setActive}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="Cookie session lifetime" hint="How long a sign-in remains valid · accepts 30m, 24h, 30d">
                <Input value={form.auth.session_lifetime} mono
                  onChange={(v) => set('auth.session_lifetime', v)} placeholder="30d" width={140}/>
              </Field>
              <Field label="JWT access token lifetime" hint="Short-lived bearer tokens for SDKs and APIs">
                <Input value={form.jwt.lifetime} mono
                  onChange={(v) => set('jwt.lifetime', v)} placeholder="15m" width={140}/>
              </Field>
              <Field label="JWT signing key" hint={`${activeKeys} active key${activeKeys === 1 ? '' : 's'} · keys retire after 2× access TTL`} span={2}>
                <div className="row" style={{ gap: 8 }}>
                  <div style={{
                    flex: 1, height: 28, padding: '0 10px',
                    display: 'inline-flex', alignItems: 'center', gap: 8,
                    background: 'var(--surface-1)',
                    border: '1px solid var(--hairline)',
                    borderRadius: 4,
                  }}>
                    <Icon.Shield width={11} height={11} style={{ color: 'var(--fg-muted)' }}/>
                    <span className="mono" style={{ fontSize: 12, color: 'var(--fg)' }}>{fingerprint(jwtKid) || (activeKeys > 0 ? 'active' : 'no key')}</span>
                    <span className="faint" style={{ fontSize: 11, marginLeft: 'auto' }}>{health?.jwt?.algorithm || 'RS256'}</span>
                  </div>
                  <button type="button" className="btn sm" onClick={handleRotate} disabled={rotating}>
                    {rotating ? 'Rotating…' : 'Rotate key'}
                  </button>
                </div>
              </Field>
            </div>
          </Section>

          {/* Auth Policy */}
          <Section id="auth_policy" title="Auth Policy" desc="Password requirements, magic-link expiry, password-reset window" onSection={setActive}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="Min password length" hint="Characters required for new passwords (min 6)">
                <Input value={form.auth.password_min_length} type="number" mono
                  onChange={(v) => set('auth.password_min_length', v)} placeholder="8" width={100}/>
              </Field>
              <Field label="Magic-link TTL" hint="How long a magic-link sign-in link stays valid">
                <Input value={form.magic_link.ttl} mono
                  onChange={(v) => set('magic_link.ttl', v)} placeholder="15m" width={120}/>
              </Field>
              <Field label="Password-reset TTL" hint="How long a reset-password link stays valid">
                <Input value={form.password_reset.ttl} mono
                  onChange={(v) => set('password_reset.ttl', v)} placeholder="1h" width={120}/>
              </Field>
            </div>
          </Section>

          {/* OAuth & SSO — moved from Identity Hub. Source of truth for server-level OAuth config. */}
          <Section id="oauth_sso" title="OAuth & SSO" desc="OAuth authorization server config, SSO connections, and MFA enforcement policy" onSection={setActive}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>

              {/* OAuth Server sub-section */}
              <div>
                <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-muted)', letterSpacing: '0.08em', marginBottom: 10, textTransform: 'uppercase' }}>
                  OAuth Authorization Server
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                  <Field label="Enabled">
                    <Select value={form.oauth_server.enabled ? 'true' : 'false'}
                      onChange={v => set('oauth_server.enabled', v === 'true')}
                      options={[{ v: 'true', l: 'On' }, { v: 'false', l: 'Off' }]}
                      width={140}/>
                  </Field>
                  <Field label="Require DPoP" hint="Enforce Demonstrating Proof of Possession on all tokens">
                    <Select value={form.oauth_server.require_dpop ? 'true' : 'false'}
                      onChange={v => set('oauth_server.require_dpop', v === 'true')}
                      options={[{ v: 'false', l: 'Off' }, { v: 'true', l: 'Required' }]}
                      width={140}/>
                  </Field>
                  <Field label="Issuer URL" hint="Authorization server issuer (iss claim)" span={2}>
                    <Input value={form.oauth_server.issuer} mono
                      onChange={v => set('oauth_server.issuer', v)}
                      placeholder="https://auth.example.com"/>
                  </Field>
                  <Field label="Access token lifetime">
                    <Input value={form.oauth_server.access_token_lifetime} mono
                      onChange={v => set('oauth_server.access_token_lifetime', v)}
                      placeholder="15m" width={140}/>
                  </Field>
                  <Field label="Refresh token lifetime">
                    <Input value={form.oauth_server.refresh_token_lifetime} mono
                      onChange={v => set('oauth_server.refresh_token_lifetime', v)}
                      placeholder="30d" width={140}/>
                  </Field>
                </div>

                {/* Device code queue */}
                <div style={{ marginTop: 14 }}>
                  <div style={{
                    display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8,
                  }}>
                    <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-muted)', letterSpacing: '0.08em', textTransform: 'uppercase' }}>Device Code Queue</div>
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
                          <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>User code</th>
                          <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Client</th>
                          <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Scopes</th>
                          <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'right' }}>Action</th>
                        </tr>
                      </thead>
                      <tbody>
                        {deviceCodes.map(dc => (
                          <tr key={dc.user_code}
                            onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                          >
                            <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}><span className="mono" style={{ fontSize: 11 }}>{dc.user_code}</span></td>
                            <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}><span className="faint" style={{ fontSize: 11 }}>{dc.client_id ? dc.client_id.slice(0, 18) + (dc.client_id.length > 18 ? '…' : '') : '—'}</span></td>
                            <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}><span className="faint" style={{ fontSize: 11 }}>{Array.isArray(dc.scopes) ? dc.scopes.join(' ') : (dc.scope || '—')}</span></td>
                            <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)', textAlign: 'right' }}>
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
              </div>

              {/* MFA sub-section */}
              <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 14 }}>
                <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-muted)', letterSpacing: '0.08em', marginBottom: 10, textTransform: 'uppercase' }}>
                  MFA Enforcement
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
                  <Field label="Enforcement" hint="Whether MFA is required for admin users">
                    <Select value={form.mfa.enforcement}
                      onChange={v => set('mfa.enforcement', v)}
                      options={[
                        { v: 'off',      l: 'Off' },
                        { v: 'optional', l: 'Optional' },
                        { v: 'required', l: 'Required' },
                      ]} width={160}/>
                  </Field>
                  <Field label="Recovery code count" hint="Backup codes per user">
                    <Input value={form.mfa.recovery_codes} type="number" mono
                      onChange={v => set('mfa.recovery_codes', v)} placeholder="10" width={80}/>
                  </Field>
                  <Field label="TOTP issuer" hint="Shown in authenticator apps" span={2}>
                    <Input value={form.mfa.issuer}
                      onChange={v => set('mfa.issuer', v)} placeholder="MyApp"/>
                  </Field>
                </div>
              </div>

              {/* SSO Connections sub-section */}
              <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 14 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
                  <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-muted)', letterSpacing: '0.08em', textTransform: 'uppercase' }}>SSO Connections</div>
                  <div style={{ flex: 1 }}/>
                  <button className="btn ghost sm" onClick={() => setSsoTick(t => t + 1)}>
                    <Icon.Refresh width={11} height={11}/>
                    Refresh
                  </button>
                </div>
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
                        <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Name / ID</th>
                        <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Type</th>
                        <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Domain</th>
                        <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'left' }}>Status</th>
                        <th style={{ padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-1)', textAlign: 'right' }}>Action</th>
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
                          <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}>
                            <div>{conn.name || conn.id}</div>
                            <div className="mono faint" style={{ fontSize: 10 }}>{conn.id ? conn.id.slice(0, 24) : ''}</div>
                          </td>
                          <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}><span className="mono faint" style={{ fontSize: 11 }}>{conn.type || conn.provider || '—'}</span></td>
                          <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}><span className="faint" style={{ fontSize: 11 }}>{conn.domain || '—'}</span></td>
                          <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)' }}>
                            <span style={{
                              display: 'inline-flex', alignItems: 'center',
                              height: 16, padding: '0 5px', borderRadius: 3,
                              fontSize: 10, fontWeight: 500,
                              background: 'var(--surface-2)',
                              color: conn.enabled !== false ? 'var(--success)' : 'var(--fg-dim)',
                              border: '1px solid var(--hairline)',
                            }}>{conn.enabled !== false ? 'active' : 'disabled'}</span>
                          </td>
                          <td style={{ padding: '7px 14px', fontSize: 12.5, borderBottom: '1px solid var(--hairline)', textAlign: 'right' }}>
                            <button className="btn ghost sm" onClick={e => { e.stopPropagation(); setSsoDrawer(conn); }}>Inspect</button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </div>
          </Section>

          {/* Email Delivery */}
          <Section id="email" title="Email Delivery" desc="Outbound mail for verifications, magic links, and password resets" onSection={setActive}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="Provider">
                <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                  <Select value={form.email.provider} onChange={(v) => set('email.provider', v)}
                    options={[
                      { v: 'shark',  l: 'Shark (managed)' },
                      { v: 'resend', l: 'Resend' },
                      { v: 'smtp',   l: 'SMTP' },
                      { v: 'dev',    l: 'Dev inbox (capture only)' },
                    ]} width={240}/>
                  {form.email.provider === 'dev' ? (
                    <button
                      type="button"
                      className="btn sm ghost"
                      style={{ alignSelf: 'flex-start' }}
                      onClick={() => {
                        const prev = config?.email?.previous_provider || form.email.previous_provider || 'shark';
                        set('email.provider', prev);
                        set('email.previous_provider', '');
                      }}
                    >
                      Switch back to {config?.email?.previous_provider || 'production provider'}
                    </button>
                  ) : (
                    <button
                      type="button"
                      className="btn sm ghost"
                      style={{ alignSelf: 'flex-start' }}
                      onClick={() => {
                        set('email.previous_provider', form.email.provider);
                        set('email.provider', 'dev');
                      }}
                    >
                      Switch to dev inbox (testing)
                    </button>
                  )}
                </div>
              </Field>
              <Field label={form.email.provider === 'smtp' ? 'SMTP password' : 'API key'} hint={emailKeySet ? 'Set · paste a new value to replace' : 'Required for non-dev providers'}>
                <MaskedInput
                  value={form.email.api_key}
                  hasValue={emailKeySet}
                  onChange={(v) => set('email.api_key', v)}
                  placeholder={emailKeySet ? '••••••••' : 're_••• or smtp password'}/>
              </Field>
              <Field label="From address" hint="Verified sender · must match your provider domain">
                <Input value={form.email.from} mono
                  onChange={(v) => set('email.from', v)} placeholder="auth@example.com"/>
              </Field>
              <Field label="From name">
                <Input value={form.email.from_name}
                  onChange={(v) => set('email.from_name', v)} placeholder="Acme Auth"/>
              </Field>
              <Field span={2}>
                <div className="row" style={{ gap: 8, marginTop: 4 }}>
                  <button type="button" className="btn sm" onClick={() => setDrawer('test_email')}>
                    <Icon.Mail width={11} height={11}/>Send test email
                  </button>
                  <span className="faint" style={{ fontSize: 11 }}>Save changes first if you edited provider settings</span>
                </div>
              </Field>
            </div>
          </Section>

          {/* OAuth Providers — quick-link to Identity Hub for social provider credentials */}
          <Section id="oauth" title="OAuth Providers" desc="Social provider credentials (Google, GitHub, Apple, Discord) are configured in Identity Hub" onSection={setActive}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <Field label="OAuth redirect URL" hint="Where users land after completing OAuth — leave blank to use server default">
                <Input value={form.social.redirect_url} mono
                  onChange={(v) => set('social.redirect_url', v)} placeholder="/dashboard"/>
              </Field>
              <div style={{
                padding: 14,
                border: '1px solid var(--hairline)',
                borderRadius: 5,
                background: 'var(--surface-1)',
                display: 'grid', gridTemplateColumns: '1fr auto', gap: 10, alignItems: 'center',
              }}>
                <div>
                  <div style={{ fontSize: 13, color: 'var(--fg)', marginBottom: 4 }}>
                    {(config.oauth_providers || []).length} provider{(config.oauth_providers || []).length === 1 ? '' : 's'} configured
                  </div>
                  <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
                    Google, GitHub, Apple, Discord — client IDs, secrets, and per-provider scopes are managed in Identity Hub.
                  </div>
                  <div className="row" style={{ gap: 4, flexWrap: 'wrap', marginTop: 8 }}>
                    {(config.oauth_providers || []).map((p) => (
                      <span key={p} className="chip" style={{ height: 18 }}>{p}</span>
                    ))}
                    {(!config.oauth_providers || config.oauth_providers.length === 0) && (
                      <span className="faint" style={{ fontSize: 11 }}>No providers active</span>
                    )}
                  </div>
                </div>
                <a href="/admin/identity-hub" className="btn sm" style={{ textDecoration: 'none' }}>
                  Open Identity Hub →
                </a>
              </div>
            </div>
          </Section>

          {/* Audit & Data */}
          <Section id="audit" title="Audit & Data" desc="How long audit logs are kept, how often expired records are cleaned, and CSV export" onSection={setActive}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 14 }}>
              <Field label="Retention" hint="Audit logs older than this are eligible for purge">
                <Input value={form.audit.retention} mono
                  onChange={(v) => set('audit.retention', v)} placeholder="30d" width={140}/>
              </Field>
              <Field label="Cleanup interval" hint="How often the background sweeper runs">
                <Input value={form.audit.cleanup_interval} mono
                  onChange={(v) => set('audit.cleanup_interval', v)} placeholder="1h" width={140}/>
              </Field>
              <Field span={2}>
                <div className="row" style={{ gap: 8 }}>
                  <button type="button" className="btn sm danger" onClick={() => setDrawer('purge_audit')}>
                    Purge audit logs older than retention
                  </button>
                  <button type="button" className="btn sm" onClick={() => setDrawer('export_audit')}>
                    Export CSV
                  </button>
                </div>
              </Field>
            </div>
          </Section>

          {/* Maintenance */}
          <Section id="maintenance" title="Maintenance" desc="Manual cleanup operations. Use carefully." onSection={setActive}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div className="row" style={{
                padding: 12, gap: 12,
                border: '1px solid var(--hairline)',
                borderRadius: 5,
                background: 'var(--surface-1)',
              }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>Purge expired sessions</div>
                  <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>Removes session rows past their expiry. Active users keep their sessions.</div>
                </div>
                <button type="button" className="btn sm danger" onClick={() => setDrawer('purge_sessions')}>
                  Purge expired
                </button>
              </div>
              <div className="row" style={{
                padding: 12, gap: 12,
                border: '1px solid var(--hairline)',
                borderRadius: 5,
                background: 'var(--surface-1)',
              }}>
                <div style={{ flex: 1 }}>
                  <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>Purge audit logs</div>
                  <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>Manual one-shot cleanup of audit rows older than the retention window above.</div>
                </div>
                <button type="button" className="btn sm danger" onClick={() => setDrawer('purge_audit')}>
                  Purge older
                </button>
              </div>
            </div>
          </Section>

        {/* ─── Danger Zone ─────────────────────────────────────────────── */}
        {/* W18: moved INSIDE scroll container so it flows with the rest of */}
        {/* settings instead of overlapping above as a fixed-height frame.  */}
        <div style={{ padding: '0 24px 24px' }}>
          <div style={{
            border: '1px solid color-mix(in oklch, var(--danger) 30%, var(--hairline))',
            borderRadius: 5,
            overflow: 'hidden',
          }}>
            {/* Header */}
            <div style={{
              padding: '10px 16px',
              borderBottom: '1px solid color-mix(in oklch, var(--danger) 20%, var(--hairline))',
              background: 'color-mix(in oklch, var(--danger) 5%, var(--surface-1))',
              fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
              color: 'var(--danger)', fontWeight: 600,
            }}>
              Danger Zone
            </div>

            {/* Mode row */}
            <DangerRow
              label="Mode"
              desc={`Currently: ${modeData?.mode ?? '…'}`}
              action={<button type="button" className="btn sm ghost" style={{ borderColor: 'var(--danger)', color: 'var(--danger)' }}
                onClick={() => setDrawer('swap_mode')}>Switch to {modeData?.mode === 'dev' ? 'prod' : 'dev'} mode</button>}
            />

            {/* Reset dev DB */}
            <DangerRow
              label="Reset dev DB"
              desc="Wipes dev.db, regenerates secrets"
              action={<button type="button" className="btn sm ghost" style={{ borderColor: 'var(--danger)', color: 'var(--danger)' }}
                onClick={() => setDrawer('reset_dev')}>Reset dev DB</button>}
            />

            {/* Reset prod DB */}
            <DangerRow
              label="Reset prod DB"
              desc="Wipes shark.db, regenerates secrets"
              action={<button type="button" className="btn sm danger"
                onClick={() => setDrawer('reset_prod')}>Reset prod DB…</button>}
              last={false}
            />

            {/* Rotate admin key */}
            <DangerRow
              label="Rotate admin key"
              desc="Invalidates current admin key"
              action={<button type="button" className="btn sm ghost" style={{ borderColor: 'var(--warn)', color: 'var(--warn)' }}
                onClick={() => setDrawer('rotate_key')}>Rotate key</button>}
              last
            />
          </div>
        </div>
        </div>{/* end scroll container — Danger Zone now scrolls with sections */}

        <CLIFooter command="shark admin config dump"/>

        {/* Floating dirty bar — pinned to bottom of right pane */}
        {isDirty && (
          <div style={{
            position: 'absolute',
            left: '50%',
            bottom: 56,
            transform: 'translateX(-50%)',
            background: 'var(--surface-2)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 5,
            padding: '8px 12px',
            display: 'flex', gap: 10, alignItems: 'center',
            zIndex: 30,
            boxShadow: '0 12px 32px -8px rgba(0,0,0,0.6), 0 0 0 1px var(--hairline)',
          }}>
            <span className="dot" style={{ background: 'var(--warn)', boxShadow: '0 0 0 3px color-mix(in oklch, var(--warn) 20%, transparent)' }}/>
            <span style={{ fontSize: 12, color: 'var(--fg)' }}>Unsaved changes</span>
            <button className="btn ghost sm" onClick={handleDiscard} disabled={saving}>Discard</button>
            <button className="btn primary sm" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving…' : 'Save changes'}
            </button>
          </div>
        )}
      </div>

      {/* Drawers */}
      {drawer === 'test_email' && (
        <TestEmailDrawer
          fromAddress={form.email.from}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer === 'purge_audit' && (
        <PurgeAuditDrawer
          retention={form.audit.retention}
          onClose={() => setDrawer(null)}
          onDone={refresh}
        />
      )}
      {drawer === 'purge_sessions' && (
        <PurgeSessionsDrawer
          onClose={() => setDrawer(null)}
          onDone={refresh}
        />
      )}
      {drawer === 'export_audit' && (
        <ExportAuditDrawer onClose={() => setDrawer(null)}/>
      )}
      {drawer === 'swap_mode' && (
        <SwapModeDrawer currentMode={modeData?.mode} onClose={() => setDrawer(null)} onDone={() => { setDrawer(null); refresh(); }}/>
      )}
      {drawer === 'reset_dev' && (
        <ResetDBDrawer target="dev" onClose={() => setDrawer(null)}/>
      )}
      {drawer === 'reset_prod' && (
        <ResetDBDrawer target="prod" onClose={() => setDrawer(null)}/>
      )}
      {drawer === 'rotate_key' && (
        <RotateKeyDrawer onClose={() => setDrawer(null)}/>
      )}

      {/* SSO inspect drawer — from OAuth & SSO section */}
      {ssoDrawer && (
        <Drawer
          title={ssoDrawer.name || 'SSO Connection'}
          subtitle={ssoDrawer.type || ssoDrawer.provider || ''}
          onClose={() => setSsoDrawer(null)}
          footer={(
            <>
              <button className="btn sm ghost" style={{ color: 'var(--danger)', borderColor: 'var(--danger)' }}
                disabled={deletingSSO === ssoDrawer.id}
                onClick={() => deleteSSOConn(ssoDrawer.id)}>
                {deletingSSO === ssoDrawer.id ? 'Deleting…' : 'Delete connection'}
              </button>
              <button className="btn sm ghost" onClick={() => setSsoDrawer(null)}>Close</button>
            </>
          )}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            {Object.entries(ssoDrawer).map(([k, v]) => (
              <div key={k}>
                <div style={{ fontSize: 10, color: 'var(--fg-muted)', fontWeight: 500, marginBottom: 2, letterSpacing: '0.06em', textTransform: 'uppercase' }}>{k}</div>
                <span className="mono" style={{ fontSize: 11, wordBreak: 'break-all', color: 'var(--fg)' }}>{String(v)}</span>
              </div>
            ))}
          </div>
        </Drawer>
      )}
    </div>
  );
}

// SystemSettings alias is exported from system_settings.tsx (compat shim).

// ─── DangerRow ───────────────────────────────────────────────────────────────
function DangerRow({ label, desc, action, last = false }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 16,
      padding: '12px 16px',
      borderBottom: last ? 'none' : '1px solid color-mix(in oklch, var(--danger) 15%, var(--hairline))',
    }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 13, fontWeight: 500, color: 'var(--fg)' }}>{label}</div>
        <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 2, lineHeight: 1.5 }}>{desc}</div>
      </div>
      <div style={{ flexShrink: 0 }}>{action}</div>
    </div>
  );
}

// ─── SwapModeDrawer ───────────────────────────────────────────────────────────
function SwapModeDrawer({ currentMode, onClose, onDone }) {
  const toast = useToast();
  const [busy, setBusy] = React.useState(false);
  const newMode = currentMode === 'dev' ? 'prod' : 'dev';

  const run = async () => {
    if (busy) return;
    setBusy(true);
    try {
      await API.post('/admin/system/swap-mode', { mode: newMode });
      toast.success(`Mode set to ${newMode} — restart the server to activate`);
      onDone && onDone();
    } catch (e) {
      toast.error(e?.message || 'Swap failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title={`Switch to ${newMode} mode`}
      subtitle={`Currently running in ${currentMode ?? '…'} mode. Takes effect on next server restart.`}
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn sm ghost" style={{ borderColor: 'var(--danger)', color: 'var(--danger)' }}
            onClick={run} disabled={busy}>
            {busy ? 'Switching…' : `Switch to ${newMode}`}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{
          padding: 10, borderRadius: 4, fontSize: 12, lineHeight: 1.6,
          border: '1px solid color-mix(in oklch, var(--warn) 35%, var(--hairline))',
          background: 'color-mix(in oklch, var(--warn) 6%, var(--surface-1))',
          color: 'var(--fg)',
        }}>
          The server will continue running in <strong>{currentMode}</strong> mode until restarted.
          After restart it will open <span style={{ fontFamily: 'var(--font-mono)' }}>{newMode}.db</span>.
        </div>
        <div className="faint mono" style={{ fontSize: 11 }}>POST /admin/system/swap-mode</div>
      </div>
    </Drawer>
  );
}

// ─── ResetDBDrawer ─────────────────────────────────────────────────────────────
const RESET_PROD_PHRASE = 'RESET PROD';

function ResetDBDrawer({ target, onClose }) {
  const toast = useToast();
  const [confirmText, setConfirmText] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [newKey, setNewKey] = React.useState(null);
  const [copied, setCopied] = React.useState(false);
  const isProd = target === 'prod';

  const run = async () => {
    if (busy) return;
    if (isProd && confirmText !== RESET_PROD_PHRASE) return;
    setBusy(true);
    try {
      const res = await API.post('/admin/system/reset', {
        target,
        ...(isProd ? { confirmation: confirmText } : {}),
      });
      if (res?.admin_key) setNewKey(res.admin_key);
      toast.success(`${target} DB reset complete`);
    } catch (e) {
      toast.error(e?.message || 'Reset failed');
      setBusy(false);
    }
  };

  const copy = () => {
    if (!newKey) return;
    navigator.clipboard.writeText(newKey).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  if (newKey) {
    return (
      <Drawer
        title="Reset complete"
        subtitle="New admin key — shown once only"
        onClose={onClose}
        footer={<button className="btn sm primary" onClick={onClose}>Done</button>}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{
            padding: 10, borderRadius: 4, fontSize: 12, lineHeight: 1.6,
            border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline))',
            background: 'color-mix(in oklch, var(--danger) 6%, var(--surface-1))',
            color: 'var(--fg)',
          }}>
            Store this key now — it will never be shown again.
          </div>
          <div style={{
            display: 'flex', gap: 6, alignItems: 'center',
            padding: '8px 10px',
            background: 'var(--surface-1)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4,
          }}>
            <span style={{ flex: 1, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak: 'break-all', color: 'var(--fg)' }}>
              {newKey}
            </span>
            <button className="btn ghost sm" onClick={copy} style={{ flexShrink: 0 }}>
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>
      </Drawer>
    );
  }

  return (
    <Drawer
      title={`Reset ${target} DB`}
      subtitle={isProd ? 'Wipes shark.db and regenerates all secrets' : 'Wipes dev.db and regenerates all secrets'}
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn sm danger"
            onClick={run}
            disabled={busy || (isProd && confirmText !== RESET_PROD_PHRASE)}>
            {busy ? 'Resetting…' : `Reset ${target} DB`}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{
          padding: 10, borderRadius: 4, fontSize: 12, lineHeight: 1.6,
          border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline))',
          background: 'color-mix(in oklch, var(--danger) 6%, var(--surface-1))',
          color: 'var(--fg)',
        }}>
          This will permanently delete all users, sessions, and secrets in the <strong>{target}</strong> database. This cannot be undone.
        </div>
        {isProd && (
          <Field label={`Type "${RESET_PROD_PHRASE}" to confirm`}>
            <input
              autoFocus
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder={RESET_PROD_PHRASE}
              style={{
                width: '100%', height: 28, padding: '0 8px',
                background: 'var(--surface-1)',
                border: '1px solid var(--hairline-strong)',
                borderRadius: 4, fontSize: 13, color: 'var(--fg)',
                fontFamily: 'var(--font-mono)',
              }}/>
          </Field>
        )}
        <div className="faint mono" style={{ fontSize: 11 }}>POST /admin/system/reset</div>
      </div>
    </Drawer>
  );
}

// ─── RotateKeyDrawer ───────────────────────────────────────────────────────────
function RotateKeyDrawer({ onClose }) {
  const toast = useToast();
  const [busy, setBusy] = React.useState(false);
  const [newKey, setNewKey] = React.useState(null);
  const [copied, setCopied] = React.useState(false);

  const run = async () => {
    if (busy) return;
    setBusy(true);
    try {
      const res = await API.post('/admin/system/reset', { target: 'key' });
      if (res?.admin_key) setNewKey(res.admin_key);
      toast.success('Admin key rotated');
    } catch (e) {
      toast.error(e?.message || 'Rotation failed');
      setBusy(false);
    }
  };

  const copy = () => {
    if (!newKey) return;
    navigator.clipboard.writeText(newKey).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  if (newKey) {
    return (
      <Drawer
        title="Key rotated"
        subtitle="New admin key — shown once only"
        onClose={onClose}
        footer={<button className="btn sm primary" onClick={onClose}>Done</button>}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div style={{
            padding: 10, borderRadius: 4, fontSize: 12, lineHeight: 1.6,
            border: '1px solid color-mix(in oklch, var(--warn) 35%, var(--hairline))',
            background: 'color-mix(in oklch, var(--warn) 6%, var(--surface-1))',
            color: 'var(--fg)',
          }}>
            The old key is now invalid. Store the new key — it will not be shown again.
          </div>
          <div style={{
            display: 'flex', gap: 6, alignItems: 'center',
            padding: '8px 10px',
            background: 'var(--surface-1)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4,
          }}>
            <span style={{ flex: 1, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak: 'break-all', color: 'var(--fg)' }}>
              {newKey}
            </span>
            <button className="btn ghost sm" onClick={copy} style={{ flexShrink: 0 }}>
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>
      </Drawer>
    );
  }

  return (
    <Drawer
      title="Rotate admin key"
      subtitle="Generates a new admin API key and invalidates the current one"
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn sm ghost" style={{ borderColor: 'var(--warn)', color: 'var(--warn)' }}
            onClick={run} disabled={busy}>
            {busy ? 'Rotating…' : 'Rotate key'}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{
          padding: 10, borderRadius: 4, fontSize: 12, lineHeight: 1.6,
          border: '1px solid color-mix(in oklch, var(--warn) 35%, var(--hairline))',
          background: 'color-mix(in oklch, var(--warn) 6%, var(--surface-1))',
          color: 'var(--fg)',
        }}>
          All existing admin sessions will be invalidated. You will need to log in again with the new key.
        </div>
        <div className="faint mono" style={{ fontSize: 11 }}>POST /admin/system/reset {"{"} target: "key" {"}"}</div>
      </div>
    </Drawer>
  );
}

// ─── Audit CSV Export Drawer ─────────────────────────────────────────────────
function ExportAuditDrawer({ onClose }) {
  const toast = useToast();
  const now = new Date();
  const thirtyDaysAgo = new Date(now - 30 * 86400000);
  const fmt = (d) => d.toISOString().slice(0, 10);
  const [from, setFrom] = React.useState(fmt(thirtyDaysAgo));
  const [to, setTo]     = React.useState(fmt(now));
  const [busy, setBusy] = React.useState(false);

  const run = async () => {
    if (busy) return;
    setBusy(true);
    try {
      // Backend streams CSV; use raw fetch with auth header so we can get a Blob.
      const key = localStorage.getItem('shark_admin_key');
      const res = await fetch('/api/v1/admin/audit-logs/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${key}`,
        },
        body: JSON.stringify({ from: new Date(from).toISOString(), to: new Date(to + 'T23:59:59Z').toISOString() }),
      });
      if (!res.ok) throw new Error(`Export failed (${res.status})`);
      const blob = await res.blob();
      const dlUrl = URL.createObjectURL(blob);
      const a     = document.createElement('a');
      a.href      = dlUrl;
      a.download  = `audit-logs-${from}-to-${to}.csv`;
      a.click();
      URL.revokeObjectURL(dlUrl);
      toast.success('CSV downloaded');
      onClose();
    } catch (e) {
      toast.error(e?.message || 'Export failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title="Export audit logs"
      subtitle="Downloads a CSV of all audit records in the selected date range"
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn primary sm" onClick={run} disabled={busy || !from || !to}>
            {busy ? 'Exporting…' : 'Download CSV'}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Field label="From date">
          <Input type="text" mono value={from} onChange={setFrom} placeholder="YYYY-MM-DD"/>
        </Field>
        <Field label="To date">
          <Input type="text" mono value={to} onChange={setTo} placeholder="YYYY-MM-DD"/>
        </Field>
        <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>POST /admin/audit-logs/export</div>
      </div>
    </Drawer>
  );
}
function TestEmailDrawer({ fromAddress, onClose }) {
  const toast = useToast();
  const [to, setTo] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  const [result, setResult] = React.useState(null);

  const send = async () => {
    if (!to.trim() || busy) return;
    setBusy(true);
    setResult(null);
    try {
      const res = await API.post('/admin/test-email', { to: to.trim() });
      setResult({ ok: true, msg: `Delivered to ${res?.to || to.trim()}` });
      toast.success('Test email sent');
    } catch (e) {
      setResult({ ok: false, msg: e?.message || 'Send failed' });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title="Send test email"
      subtitle={`Routes through your active email provider · From ${fromAddress || '(not configured)'}`}
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn primary sm" onClick={send} disabled={busy || !to.trim()}>
            {busy ? 'Sending…' : 'Send test'}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Field label="Send to" hint="An inbox you own — message subject is 'SharkAuth Test Email'">
          <input
            type="email"
            autoFocus
            value={to}
            onChange={(e) => setTo(e.target.value)}
            placeholder="you@example.com"
            style={{
              width: '100%', height: 28, padding: '0 8px',
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, fontSize: 13, color: 'var(--fg)',
            }}/>
        </Field>
        {result && (
          <div style={{
            padding: 10,
            border: '1px solid ' + (result.ok ? 'color-mix(in oklch, var(--success) 35%, var(--hairline))' : 'color-mix(in oklch, var(--danger) 35%, var(--hairline))'),
            borderRadius: 4,
            background: result.ok
              ? 'color-mix(in oklch, var(--success) 8%, var(--surface-1))'
              : 'color-mix(in oklch, var(--danger) 8%, var(--surface-1))',
            fontSize: 12, lineHeight: 1.5,
            color: result.ok ? 'var(--success)' : 'var(--danger)',
          }}>
            {result.msg}
          </div>
        )}
        <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>POST /admin/test-email</div>
      </div>
    </Drawer>
  );
}

function PurgeAuditDrawer({ retention, onClose, onDone }) {
  const toast = useToast();
  const [confirmText, setConfirmText] = React.useState('');
  const [busy, setBusy] = React.useState(false);
  // Use the configured retention to compute a "before" timestamp client-side.
  // The endpoint accepts {before: <RFC3339 timestamp>}.
  const beforeISO = React.useMemo(() => {
    const m = (retention || '30d').match(/^(\d+)([smhd])$/);
    if (!m) return null;
    const n = Number(m[1]);
    const unit = m[2];
    const ms = unit === 's' ? n*1000 : unit === 'm' ? n*60000 : unit === 'h' ? n*3600000 : n*86400000;
    return new Date(Date.now() - ms).toISOString();
  }, [retention]);

  const run = async () => {
    if (busy || confirmText !== 'PURGE') return;
    setBusy(true);
    try {
      await API.post('/admin/audit-logs/purge', beforeISO ? { before: beforeISO } : {});
      toast.success('Audit logs purged');
      onDone && onDone();
      onClose();
    } catch (e) {
      toast.error(e?.message || 'Purge failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title="Purge audit logs"
      subtitle="Permanently delete audit rows older than the retention window"
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn sm danger" onClick={run} disabled={busy || confirmText !== 'PURGE'}>
            {busy ? 'Purging…' : 'Purge logs'}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{
          padding: 10,
          border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline))',
          borderRadius: 4,
          background: 'color-mix(in oklch, var(--danger) 6%, var(--surface-1))',
          fontSize: 12, lineHeight: 1.5, color: 'var(--fg)',
        }}>
          This permanently removes audit rows older than <span className="mono" style={{ color: 'var(--danger)' }}>{retention}</span>{beforeISO && <> (before <span className="mono">{beforeISO}</span>)</>}. This action cannot be undone.
        </div>
        <Field label="Type PURGE to confirm">
          <input
            autoFocus
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder="PURGE"
            style={{
              width: '100%', height: 28, padding: '0 8px',
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, fontSize: 13, color: 'var(--fg)',
              fontFamily: 'var(--font-mono)',
            }}/>
        </Field>
        <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>POST /admin/audit-logs/purge</div>
      </div>
    </Drawer>
  );
}

function PurgeSessionsDrawer({ onClose, onDone }) {
  const toast = useToast();
  const [confirmText, setConfirmText] = React.useState('');
  const [busy, setBusy] = React.useState(false);

  const run = async () => {
    if (busy || confirmText !== 'PURGE') return;
    setBusy(true);
    try {
      await API.post('/admin/sessions/purge-expired', {});
      toast.success('Expired sessions purged');
      onDone && onDone();
      onClose();
    } catch (e) {
      toast.error(e?.message || 'Purge failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <Drawer
      title="Purge expired sessions"
      subtitle="Remove session rows whose expiry has passed"
      onClose={onClose}
      footer={(
        <>
          <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn sm danger" onClick={run} disabled={busy || confirmText !== 'PURGE'}>
            {busy ? 'Purging…' : 'Purge expired'}
          </button>
        </>
      )}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{
          padding: 10,
          border: '1px solid color-mix(in oklch, var(--warn) 35%, var(--hairline))',
          borderRadius: 4,
          background: 'color-mix(in oklch, var(--warn) 6%, var(--surface-1))',
          fontSize: 12, lineHeight: 1.5, color: 'var(--fg)',
        }}>
          Active users are not affected — only sessions whose <span className="mono">expires_at</span> is in the past are deleted.
        </div>
        <Field label="Type PURGE to confirm">
          <input
            autoFocus
            value={confirmText}
            onChange={(e) => setConfirmText(e.target.value)}
            placeholder="PURGE"
            style={{
              width: '100%', height: 28, padding: '0 8px',
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, fontSize: 13, color: 'var(--fg)',
              fontFamily: 'var(--font-mono)',
            }}/>
        </Field>
        <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>POST /admin/sessions/purge-expired</div>
      </div>
    </Drawer>
  );
}
