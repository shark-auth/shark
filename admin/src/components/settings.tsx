// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'
import { CLIFooter } from './CLIFooter'

// Settings — unified, dense configuration command center.
// Replaces the old templated panel layout. Sessions and JWTs are now ONE concept.

const SECTIONS = [
  { id: 'server',     label: 'Server',          desc: 'Runtime info' },
  { id: 'tokens',     label: 'Session Tokens',  desc: 'Lifetimes, rotation, signing' },
  { id: 'auth',       label: 'Authentication',  desc: 'Passwords, magic links, reset' },
  { id: 'passkeys',   label: 'Passkeys',        desc: 'WebAuthn relying party' },
  { id: 'social',     label: 'OAuth Providers', desc: 'Google, GitHub, Apple, Discord' },
  { id: 'email',      label: 'Email Delivery',  desc: 'Provider, sender, templates' },
  { id: 'security',   label: 'CORS & Security', desc: 'Origins, dev mode' },
  { id: 'audit',      label: 'Audit & Data',    desc: 'Retention, cleanup' },
  { id: 'telemetry',  label: 'Telemetry',       desc: 'Anonymous usage data' },
  { id: 'maintenance',label: 'Maintenance',     desc: 'Manual cleanup operations' },
];

// ─── Atomic primitives ────────────────────────────────────────────────────────

function ConfigRow({ label, hint, system, children }) {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'flex-start',
      padding: '10px 14px',
      borderBottom: '1px solid var(--hairline)',
      gap: 14,
      minHeight: 40,
    }}>
      <div style={{ width: 168, flexShrink: 0, paddingTop: 4 }}>
        <div className="row" style={{ gap: 6, alignItems: 'center' }}>
          <span style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', fontWeight: 500 }}>{label}</span>
          {system && <span className="chip" style={{height: 14, fontSize: 9, padding: '0 4px', color: 'var(--fg-dim)'}}>READ ONLY</span>}
        </div>
        {hint && <div style={{ fontSize: 11, color: 'var(--fg-faint)', marginTop: 3, lineHeight: 1.4 }}>{hint}</div>}
      </div>
      <div style={{ flex: 1, fontSize: 12.5, minWidth: 0, paddingTop: 2 }}>{children}</div>
    </div>
  );
}

function Section({ id, title, desc, children, accessory }) {
  return (
    <section id={id} style={{ scrollMarginTop: 16 }}>
      <div style={{ display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between', marginBottom: 10 }}>
        <div>
          <div style={{ fontSize: 13, fontWeight: 600, fontFamily: 'var(--font-display)', letterSpacing: '-0.005em' }}>{title}</div>
          {desc && <div style={{ fontSize: 11.5, color: 'var(--fg-dim)', marginTop: 2 }}>{desc}</div>}
        </div>
        {accessory}
      </div>
      <div className="card">{children}</div>
    </section>
  );
}

const inputStyle = {
  background: 'var(--surface-2)',
  border: '1px solid var(--hairline-strong)',
  borderRadius: 4,
  padding: '5px 8px',
  color: 'var(--fg)',
  fontSize: 12.5,
  outline: 'none',
  width: '100%',
  height: 26,
  fontFamily: 'inherit',
  transition: 'border-color 80ms ease-out',
};

const monoInputStyle = { ...inputStyle, fontFamily: 'var(--font-mono)' };

function TextInput({ value, onChange, placeholder, mono = false, type = 'text', width, disabled }) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      style={{
        ...(mono ? monoInputStyle : inputStyle),
        width: width || '100%',
        opacity: disabled ? 0.5 : 1,
        cursor: disabled ? 'not-allowed' : 'text',
      }}
      onFocus={e => { e.target.style.borderColor = 'var(--hairline-bright)'; }}
      onBlur={e => { e.target.style.borderColor = 'var(--hairline-strong)'; }}
    />
  );
}

function NumberInput({ value, onChange, min, max, suffix, width = 72 }) {
  return (
    <div className="row" style={{ gap: 6 }}>
      <input
        type="number"
        value={value ?? ''}
        onChange={e => onChange(parseInt(e.target.value) || 0)}
        min={min}
        max={max}
        style={{ ...monoInputStyle, width }}
      />
      {suffix && <span style={{ fontSize: 11, color: 'var(--fg-dim)' }}>{suffix}</span>}
    </div>
  );
}

function Select({ value, onChange, options, width }) {
  return (
    <select
      value={value || ''}
      onChange={e => onChange(e.target.value)}
      style={{
        ...inputStyle,
        width: width || 'auto',
        minWidth: 160,
        appearance: 'none',
        backgroundImage: `url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 10 10'><path d='M2 4l3 3 3-3' stroke='%236b6b6b' fill='none' stroke-width='1.5'/></svg>")`,
        backgroundRepeat: 'no-repeat',
        backgroundPosition: 'right 6px center',
        paddingRight: 22,
        cursor: 'pointer',
      }}
    >
      {options.map(o => <option key={o.v} value={o.v}>{o.l}</option>)}
    </select>
  );
}

function Toggle({ checked, onChange, label }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className="row"
      style={{
        gap: 8,
        height: 24,
        padding: '0 4px',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        background: checked ? 'var(--surface-3)' : 'var(--surface-2)',
        cursor: 'pointer',
      }}
    >
      <span style={{
        width: 22, height: 12,
        borderRadius: 2,
        background: checked ? 'var(--fg)' : 'var(--hairline-bright)',
        position: 'relative',
        transition: 'background 80ms ease-out',
      }}>
        <span style={{
          position: 'absolute',
          top: 1, left: checked ? 11 : 1,
          width: 10, height: 10,
          borderRadius: 1,
          background: checked ? '#000' : 'var(--fg-muted)',
          transition: 'left 100ms ease-out',
        }}/>
      </span>
      <span style={{ fontSize: 11.5, color: checked ? 'var(--fg)' : 'var(--fg-dim)' }}>
        {label || (checked ? 'Enabled' : 'Disabled')}
      </span>
    </button>
  );
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatUptime(s) {
  if (s == null) return '—';
  const d = Math.floor(s / 86400);
  const hh = Math.floor((s % 86400) / 3600).toString().padStart(2, '0');
  const mm = Math.floor((s % 3600) / 60).toString().padStart(2, '0');
  return `${d}d ${hh}:${mm}`;
}

function formatBytes(mb) {
  if (mb == null) return '—';
  if (mb >= 1024) return `${(mb / 1024).toFixed(2)} GB`;
  return `${mb.toFixed(2)} MB`;
}

function deepClone(o) {
  return JSON.parse(JSON.stringify(o ?? {}));
}

// ─── Main Component ───────────────────────────────────────────────────────────

export function Settings() {
  const { data: health } = useAPI('/admin/health');
  const { data: config, loading, refresh } = useAPI('/admin/config');
  const toast = useToast();

  const [form, setForm] = React.useState(null);
  const [saving, setSaving] = React.useState(false);
  const [activeAnchor, setActiveAnchor] = React.useState('server');
  const [purging, setPurging] = React.useState({ session: false, audit: false });
  const [auditBefore, setAuditBefore] = React.useState('');
  const [showSecret, setShowSecret] = React.useState({});
  const scrollRef = React.useRef(null);

  // Initialize form when config loads
  React.useEffect(() => {
    if (config) {
      setForm({
        auth: {
          session_lifetime: config.auth?.session_lifetime || '30d',
          password_min_length: config.auth?.password_min_length || 8,
        },
        jwt: {
          enabled: config.jwt?.enabled ?? true,
          mode: config.jwt?.mode || 'session',
          issuer: config.jwt?.issuer || '',
          audience: config.jwt?.audience || 'shark',
          clock_skew: config.jwt?.clock_skew || '30s',
        },
        passkeys: {
          rp_name: config.passkey?.rp_name || 'SharkAuth',
          rp_id: config.passkey?.rp_id || '',
          user_verification: config.passkey?.user_verification || 'preferred',
        },
        email: {
          provider: config.email?.provider || 'shark',
          api_key: config.email?.api_key || '',
          from: config.email?.from || '',
          from_name: config.email?.from_name || 'SharkAuth',
        },
        audit: {
          retention: config.audit?.retention || '0',
          cleanup_interval: config.audit?.cleanup_interval || '1h',
        },
        password_reset: {
          ttl: config.password_reset?.ttl || '30m',
        },
        social: {
          redirect_url: config.social?.redirect_url || '',
          google: {
            client_id: config.social?.google?.client_id || '',
            client_secret: config.social?.google?.client_secret || '',
          },
          github: {
            client_id: config.social?.github?.client_id || '',
            client_secret: config.social?.github?.client_secret || '',
          },
        },
      });
    }
  }, [config]);

  // Track which anchor is in viewport (for sub-nav highlight)
  React.useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const handler = () => {
      const top = el.scrollTop;
      let cur = SECTIONS[0].id;
      for (const s of SECTIONS) {
        const node = el.querySelector(`#${s.id}`);
        if (node && node.offsetTop - 80 <= top) cur = s.id;
      }
      setActiveAnchor(cur);
    };
    el.addEventListener('scroll', handler);
    handler();
    return () => el.removeEventListener('scroll', handler);
  }, [form]);

  const original = React.useMemo(() => {
    if (!config) return null;
    return {
      auth: { session_lifetime: config.auth?.session_lifetime, password_min_length: config.auth?.password_min_length },
      jwt: { enabled: config.jwt?.enabled, mode: config.jwt?.mode, issuer: config.jwt?.issuer, audience: config.jwt?.audience, clock_skew: config.jwt?.clock_skew },
      passkeys: { rp_name: config.passkey?.rp_name, rp_id: config.passkey?.rp_id, user_verification: config.passkey?.user_verification },
      email: { provider: config.email?.provider, api_key: config.email?.api_key, from: config.email?.from, from_name: config.email?.from_name },
      audit: { retention: config.audit?.retention, cleanup_interval: config.audit?.cleanup_interval },
      password_reset: { ttl: config.password_reset?.ttl },
      social: deepClone(config.social),
    };
  }, [config]);

  const isDirty = form && original && JSON.stringify(form) !== JSON.stringify({
    ...original,
    auth: { session_lifetime: original.auth.session_lifetime || '30d', password_min_length: original.auth.password_min_length || 8 },
    jwt: { enabled: original.jwt.enabled ?? true, mode: original.jwt.mode || 'session', issuer: original.jwt.issuer || '', audience: original.jwt.audience || 'shark', clock_skew: original.jwt.clock_skew || '30s' },
    passkeys: { rp_name: original.passkeys.rp_name || 'SharkAuth', rp_id: original.passkeys.rp_id || '', user_verification: original.passkeys.user_verification || 'preferred' },
    email: { provider: original.email.provider || 'shark', api_key: original.email.api_key || '', from: original.email.from || '', from_name: original.email.from_name || 'SharkAuth' },
    audit: { retention: original.audit.retention || '0', cleanup_interval: original.audit.cleanup_interval || '1h' },
    password_reset: { ttl: original.password_reset.ttl || '30m' },
    social: {
      redirect_url: original.social?.redirect_url || '',
      google: {
        client_id: original.social?.google?.client_id || '',
        client_secret: original.social?.google?.client_secret || '',
      },
      github: {
        client_id: original.social?.github?.client_id || '',
        client_secret: original.social?.github?.client_secret || '',
      },
    },
  });

  const set = (path, val) => {
    setForm(prev => {
      const next = deepClone(prev);
      const parts = path.split('.');
      let curr = next;
      for (let i = 0; i < parts.length - 1; i++) {
        curr[parts[i]] = curr[parts[i]] ?? {};
        curr = curr[parts[i]];
      }
      curr[parts[parts.length - 1]] = val;
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      // Filter out unchanged secret placeholders (********) so they don't overwrite
      const payload = deepClone(form);
      if (payload.social?.google?.client_secret === '********') delete payload.social.google.client_secret;
      if (payload.social?.github?.client_secret === '********') delete payload.social.github.client_secret;
      await API.patch('/admin/config', payload);
      toast.success('Configuration saved and reloaded');
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setForm(null);
    refresh();
  };

  const handleTestEmail = async () => {
    const to = window.prompt('Send test email to:');
    if (!to) return;
    try {
      await API.post('/admin/test-email', { to });
      toast.success('Test email queued for ' + to);
    } catch (e) {
      toast.error(e?.message || 'Send failed');
    }
  };

  const handlePurgeSessions = async () => {
    setPurging(p => ({ ...p, session: true }));
    try {
      const res = await API.post('/admin/sessions/purge-expired');
      toast.success(`Purged ${res?.deleted ?? 0} expired session${res?.deleted === 1 ? '' : 's'}`);
    } catch (e) {
      toast.error(e?.message || 'Purge failed');
    } finally {
      setPurging(p => ({ ...p, session: false }));
    }
  };

  const handlePurgeAudit = async () => {
    if (!auditBefore) return;
    setPurging(p => ({ ...p, audit: true }));
    try {
      const iso = new Date(auditBefore).toISOString();
      const res = await API.post('/admin/audit-logs/purge', { before: iso });
      toast.success(`Purged ${res?.deleted ?? 0} audit ${res?.deleted === 1 ? 'entry' : 'entries'}`);
      setAuditBefore('');
    } catch (e) {
      toast.error(e?.message || 'Purge failed');
    } finally {
      setPurging(p => ({ ...p, audit: false }));
    }
  };

  const handleRotateSigningKey = async () => {
    if (!window.confirm('Rotate JWT signing key? Existing tokens stay valid until they expire.')) return;
    try {
      const res = await API.post('/admin/auth/rotate-signing-key');
      toast.success('Signing key rotated' + (res?.kid ? ` (${res.kid.slice(0, 8)}…)` : ''));
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Rotation failed');
    }
  };

  const scrollTo = (id) => {
    const node = scrollRef.current?.querySelector(`#${id}`);
    if (node) node.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  if (loading || !form) {
    return (
      <div className="row" style={{ height: '100%', justifyContent: 'center', color: 'var(--fg-dim)', fontSize: 12 }}>
        <Icon.Refresh width={11} height={11} style={{ animation: 'spin 0.8s linear infinite', marginRight: 6 }}/>
        Loading configuration…
      </div>
    );
  }

  const baseUrl = config?.base_url || window.location.origin;
  const corsOrigins = config?.cors_origins || [];
  const isDevMode = config?.dev_mode === true;
  const smtpConfigured = !!(health?.smtp?.configured);
  const jwtEnabled = form.jwt.enabled;
  const sessionMode = form.jwt.mode === 'jwt' ? 'JWT' : 'Session';
  const oauthProviders = config?.oauth_providers || [];

  return (
    <div className="col" style={{ height: '100%', overflow: 'hidden' }}>
      {/* ── Topbar ── */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        height: 44, padding: '0 16px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        flexShrink: 0,
      }}>
        <div className="row" style={{ gap: 10 }}>
          <h1 style={{ fontSize: 14, fontWeight: 600, fontFamily: 'var(--font-display)', margin: 0, letterSpacing: '-0.005em' }}>Settings</h1>
          <span className="chip" style={{ height: 18, fontSize: 10 }}>
            <span className="dot success" style={{ width: 5, height: 5 }}/>
            {sessionMode} mode
          </span>
          {isDevMode && <span className="chip warn" style={{ height: 18, fontSize: 10 }}>dev mode</span>}
        </div>
        <div className="row" style={{ gap: 6 }}>
          <span className="faint mono" style={{ fontSize: 11 }}>v{health?.version ?? '—'}</span>
        </div>
      </div>

      {/* ── Body: sub-nav + scrollable content ── */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}>
        {/* Sub-nav */}
        <nav style={{
          width: 200,
          borderRight: '1px solid var(--hairline)',
          flexShrink: 0,
          padding: '12px 0',
          overflowY: 'auto',
          background: 'var(--surface-0)',
        }}>
          {SECTIONS.map(s => (
            <button
              key={s.id}
              onClick={() => scrollTo(s.id)}
              style={{
                display: 'block',
                width: '100%',
                textAlign: 'left',
                padding: '6px 14px',
                fontSize: 12,
                color: activeAnchor === s.id ? 'var(--fg)' : 'var(--fg-dim)',
                background: activeAnchor === s.id ? 'var(--surface-2)' : 'transparent',
                borderLeft: activeAnchor === s.id ? '1px solid var(--fg)' : '1px solid transparent',
                fontWeight: activeAnchor === s.id ? 500 : 400,
                cursor: 'pointer',
                transition: 'color 80ms, background 80ms',
              }}
              onMouseEnter={e => { if (activeAnchor !== s.id) e.currentTarget.style.color = 'var(--fg-muted)'; }}
              onMouseLeave={e => { if (activeAnchor !== s.id) e.currentTarget.style.color = 'var(--fg-dim)'; }}
            >
              {s.label}
            </button>
          ))}
        </nav>

        {/* Content */}
        <div ref={scrollRef} style={{ flex: 1, overflowY: 'auto', padding: '20px 24px', paddingBottom: 120 }}>
          <div style={{ maxWidth: 820, display: 'flex', flexDirection: 'column', gap: 24 }}>

            {/* ── Server ── */}
            <Section id="server" title="Server" desc="Runtime info and immutable system-level configuration">
              <ConfigRow label="Base URL" system hint="Configured at boot via sharkauth.yaml">
                <CopyField value={baseUrl} truncate={56}/>
              </ConfigRow>
              <ConfigRow label="Version" system>
                <span className="mono" style={{ fontSize: 12 }}>{health?.version ?? '—'}</span>
              </ConfigRow>
              <ConfigRow label="Uptime" system>
                <span className="mono" style={{ fontSize: 12 }}>{formatUptime(health?.uptime_seconds)}</span>
              </ConfigRow>
              <ConfigRow label="Database" system>
                <div className="row" style={{ gap: 8 }}>
                  <span className="mono" style={{ fontSize: 12 }}>{health?.db?.driver ?? 'sqlite'}</span>
                  <span className="faint" style={{ fontSize: 11 }}>·</span>
                  <span className="mono" style={{ fontSize: 12 }}>{formatBytes(health?.db?.size_mb)}</span>
                  {health?.db?.status && (
                    <span className={'chip' + (health.db.status === 'ok' ? ' success' : ' danger')} style={{ height: 16, fontSize: 10 }}>
                      <span className={'dot' + (health.db.status === 'ok' ? ' success' : ' danger')} style={{ width: 4, height: 4 }}/>
                      {health.db.status}
                    </span>
                  )}
                </div>
              </ConfigRow>
              <ConfigRow label="Migrations" system>
                <span className="mono" style={{ fontSize: 12 }}>v{health?.migrations?.current ?? '—'}</span>
              </ConfigRow>
              <ConfigRow label="Environment" system>
                <div className="row" style={{ gap: 6 }}>
                  <span className="mono" style={{ fontSize: 12 }}>{config?.environment ?? 'production'}</span>
                  {isDevMode && <span className="chip warn" style={{ height: 16, fontSize: 10 }}>dev mode active</span>}
                </div>
              </ConfigRow>
            </Section>

            {/* ── Session Tokens (UNIFIED) ── */}
            <Section
              id="tokens"
              title="Session Tokens"
              desc="One unified concept. Sessions are JWTs internally; choose how they're validated."
              accessory={
                jwtEnabled && (
                  <button className="btn sm" onClick={handleRotateSigningKey}>
                    <Icon.Refresh width={11} height={11}/>
                    Rotate signing key
                  </button>
                )
              }
            >
              <ConfigRow label="Lifetime" hint="How long a session stays valid before re-auth">
                <TextInput
                  value={form.auth.session_lifetime}
                  onChange={v => set('auth.session_lifetime', v)}
                  mono
                  placeholder="e.g. 30d, 12h, 4w"
                  width={120}
                />
              </ConfigRow>
              <ConfigRow label="Validation Mode" hint="Stateful = DB lookup per request. Stateless = signature-only (faster, harder to revoke).">
                <Select
                  value={form.jwt.mode}
                  onChange={v => set('jwt.mode', v)}
                  options={[
                    { v: 'session', l: 'Stateful — DB-backed sessions' },
                    { v: 'jwt',     l: 'Stateless — JWT signature only' },
                  ]}
                  width={300}
                />
              </ConfigRow>
              <ConfigRow label="Issuer" hint="iss claim. Defaults to base URL when empty.">
                <TextInput
                  value={form.jwt.issuer}
                  onChange={v => set('jwt.issuer', v)}
                  mono
                  placeholder={baseUrl}
                />
              </ConfigRow>
              <ConfigRow label="Audience" hint="aud claim — your application identifier">
                <TextInput
                  value={form.jwt.audience}
                  onChange={v => set('jwt.audience', v)}
                  mono
                  placeholder="shark"
                  width={200}
                />
              </ConfigRow>
              <ConfigRow label="Clock Skew" hint="Tolerance for nbf/exp validation across machines">
                <TextInput
                  value={form.jwt.clock_skew}
                  onChange={v => set('jwt.clock_skew', v)}
                  mono
                  placeholder="30s"
                  width={100}
                />
              </ConfigRow>
              <ConfigRow label="Signing" system hint="Algorithm and active key count from JWKS">
                <div className="row" style={{ gap: 8 }}>
                  <span className="mono" style={{ fontSize: 12 }}>{config?.jwt?.algorithm || 'ES256'}</span>
                  <span className="faint" style={{ fontSize: 11 }}>·</span>
                  <span className="mono" style={{ fontSize: 12 }}>{config?.jwt?.active_keys ?? 0} active key{config?.jwt?.active_keys === 1 ? '' : 's'}</span>
                </div>
              </ConfigRow>
            </Section>

            {/* ── Authentication ── */}
            <Section id="auth" title="Authentication" desc="Password rules and short-lived token lifetimes">
              <ConfigRow label="Min Password Length" hint="Minimum characters required for new passwords">
                <NumberInput
                  value={form.auth.password_min_length}
                  onChange={v => set('auth.password_min_length', v)}
                  min={6}
                  max={128}
                  suffix="characters"
                />
              </ConfigRow>
              <ConfigRow label="Password Reset TTL" hint="How long reset links stay valid">
                <TextInput
                  value={form.password_reset.ttl}
                  onChange={v => set('password_reset.ttl', v)}
                  mono
                  placeholder="30m"
                  width={100}
                />
              </ConfigRow>
              <ConfigRow label="Magic Link TTL" system hint="Edit magic_link.token_lifetime in sharkauth.yaml">
                <span className="mono" style={{ fontSize: 12 }}>{config?.magic_link?.ttl || '10m'}</span>
              </ConfigRow>
            </Section>

            {/* ── Passkeys ── */}
            <Section id="passkeys" title="Passkeys" desc="WebAuthn relying party — must match your domain exactly">
              <ConfigRow label="RP Name" hint="Shown in browser passkey prompts">
                <TextInput
                  value={form.passkeys.rp_name}
                  onChange={v => set('passkeys.rp_name', v)}
                  placeholder="Acme Inc"
                />
              </ConfigRow>
              <ConfigRow label="RP ID" hint="Effective domain (no scheme/port). e.g. example.com — must match origin">
                <TextInput
                  value={form.passkeys.rp_id}
                  onChange={v => set('passkeys.rp_id', v)}
                  mono
                  placeholder="example.com"
                />
              </ConfigRow>
              <ConfigRow label="User Verification" hint="Whether biometrics/PIN are required at sign-in">
                <Select
                  value={form.passkeys.user_verification}
                  onChange={v => set('passkeys.user_verification', v)}
                  options={[
                    { v: 'required',    l: 'Required — biometrics every time' },
                    { v: 'preferred',   l: 'Preferred — biometrics if available' },
                    { v: 'discouraged', l: 'Discouraged — fastest UX' },
                  ]}
                  width={300}
                />
              </ConfigRow>
            </Section>

            {/* ── OAuth Providers ── */}
            <Section
              id="social"
              title="OAuth Providers"
              desc={`Active: ${oauthProviders.length ? oauthProviders.join(', ') : 'none'}. Configure client credentials below.`}
            >
              <ConfigRow label="Redirect URL" hint="Default callback for all providers when not overridden">
                <TextInput
                  value={form.social.redirect_url}
                  onChange={v => set('social.redirect_url', v)}
                  mono
                  placeholder={`${baseUrl}/auth/callback`}
                />
              </ConfigRow>

              <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--hairline)' }}>
                <div className="row" style={{ gap: 8 }}>
                  <span style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', fontWeight: 500 }}>Google</span>
                  {form.social.google.client_id ? (
                    <span className="chip success" style={{ height: 16, fontSize: 10 }}>configured</span>
                  ) : (
                    <span className="chip" style={{ height: 16, fontSize: 10 }}>not set</span>
                  )}
                </div>
              </div>
              <ConfigRow label="Client ID">
                <TextInput
                  value={form.social.google.client_id}
                  onChange={v => set('social.google.client_id', v)}
                  mono
                  placeholder="xxxxxxxxx.apps.googleusercontent.com"
                />
              </ConfigRow>
              <ConfigRow label="Client Secret">
                <div className="row" style={{ gap: 6 }}>
                  <TextInput
                    value={form.social.google.client_secret}
                    onChange={v => set('social.google.client_secret', v)}
                    mono
                    type={showSecret.google ? 'text' : 'password'}
                    placeholder="GOCSPX-..."
                  />
                  <button
                    className="btn sm icon"
                    onClick={() => setShowSecret(s => ({ ...s, google: !s.google }))}
                    title={showSecret.google ? 'Hide' : 'Show'}
                  >
                    <Icon.Eye width={11} height={11}/>
                  </button>
                </div>
              </ConfigRow>

              <div style={{ padding: '6px 14px', borderBottom: '1px solid var(--hairline)', borderTop: '1px solid var(--hairline)', background: 'var(--surface-0)' }}>
                <div className="row" style={{ gap: 8 }}>
                  <span style={{ fontSize: 11, color: 'var(--fg-muted)', textTransform: 'uppercase', letterSpacing: '0.06em', fontWeight: 500 }}>GitHub</span>
                  {form.social.github.client_id ? (
                    <span className="chip success" style={{ height: 16, fontSize: 10 }}>configured</span>
                  ) : (
                    <span className="chip" style={{ height: 16, fontSize: 10 }}>not set</span>
                  )}
                </div>
              </div>
              <ConfigRow label="Client ID">
                <TextInput
                  value={form.social.github.client_id}
                  onChange={v => set('social.github.client_id', v)}
                  mono
                  placeholder="Iv1.xxxxxxxxxx"
                />
              </ConfigRow>
              <ConfigRow label="Client Secret">
                <div className="row" style={{ gap: 6 }}>
                  <TextInput
                    value={form.social.github.client_secret}
                    onChange={v => set('social.github.client_secret', v)}
                    mono
                    type={showSecret.github ? 'text' : 'password'}
                    placeholder="ghp_..."
                  />
                  <button
                    className="btn sm icon"
                    onClick={() => setShowSecret(s => ({ ...s, github: !s.github }))}
                    title={showSecret.github ? 'Hide' : 'Show'}
                  >
                    <Icon.Eye width={11} height={11}/>
                  </button>
                </div>
              </ConfigRow>
            </Section>

            {/* ── Email ── */}
            <Section
              id="email"
              title="Email Delivery"
              desc="Outbound email for magic links, verification, password reset"
              accessory={
                <button className="btn sm" onClick={handleTestEmail}>
                  <Icon.Mail width={11} height={11}/>
                  Send test
                </button>
              }
            >
              <ConfigRow label="Provider">
                <div className="row" style={{ gap: 8 }}>
                  <Select
                    value={form.email.provider}
                    onChange={v => set('email.provider', v)}
                    options={[
                      { v: 'shark',  l: 'Shark Managed' },
                      { v: 'resend', l: 'Resend (API)' },
                      { v: 'smtp',   l: 'Custom SMTP' },
                      { v: 'dev',    l: 'Dev Inbox (local only)' },
                    ]}
                    width={240}
                  />
                  {smtpConfigured ? (
                    <span className="chip success" style={{ height: 18, fontSize: 10 }}>
                      <span className="dot success" style={{ width: 4, height: 4 }}/>
                      ready
                    </span>
                  ) : (
                    <span className="chip warn" style={{ height: 18, fontSize: 10 }}>not configured</span>
                  )}
                </div>
              </ConfigRow>
              {(form.email.provider === 'resend' || form.email.provider === 'shark' || form.email.provider === 'smtp') && (
                <ConfigRow label={form.email.provider === 'smtp' ? 'SMTP Password' : 'API Key'}>
                  <div className="row" style={{ gap: 6 }}>
                    <TextInput
                      value={form.email.api_key}
                      onChange={v => set('email.api_key', v)}
                      mono
                      type={showSecret.email ? 'text' : 'password'}
                      placeholder={form.email.provider === 'resend' ? 're_...' : '••••••••••'}
                    />
                    <button
                      className="btn sm icon"
                      onClick={() => setShowSecret(s => ({ ...s, email: !s.email }))}
                    >
                      <Icon.Eye width={11} height={11}/>
                    </button>
                  </div>
                </ConfigRow>
              )}
              <ConfigRow label="From Name" hint="Sender name shown to recipients">
                <TextInput
                  value={form.email.from_name}
                  onChange={v => set('email.from_name', v)}
                  placeholder="SharkAuth"
                />
              </ConfigRow>
              <ConfigRow label="From Address" hint="Must be a verified sender on your provider">
                <TextInput
                  value={form.email.from}
                  onChange={v => set('email.from', v)}
                  mono
                  placeholder="auth@yourdomain.com"
                />
              </ConfigRow>
            </Section>

            {/* ── CORS & Security ── */}
            <Section
              id="security"
              title="CORS & Security"
              desc="Allowed origins for browser-based clients"
            >
              <ConfigRow label="Allowed Origins" system hint="Edit server.cors_origins in sharkauth.yaml">
                <div className="col" style={{ gap: 4 }}>
                  {corsOrigins.length === 0 ? (
                    <span className="faint" style={{ fontSize: 11.5, fontStyle: 'italic' }}>None — same-origin only</span>
                  ) : (
                    corsOrigins.map(o => (
                      <div key={o} className="row" style={{
                        gap: 6,
                        padding: '3px 8px',
                        background: 'var(--surface-2)',
                        border: '1px solid var(--hairline)',
                        borderRadius: 3,
                        width: 'fit-content',
                        maxWidth: '100%',
                      }}>
                        <Icon.Globe width={10} height={10} style={{ opacity: 0.5 }}/>
                        <span className="mono" style={{ fontSize: 11.5 }}>{o}</span>
                      </div>
                    ))
                  )}
                </div>
              </ConfigRow>
              <ConfigRow label="Server Port" system>
                <span className="mono" style={{ fontSize: 12 }}>{config?.server?.port ?? '—'}</span>
              </ConfigRow>
              <ConfigRow label="Dev Mode" system hint="Disables certain production safety checks">
                <span className={'chip' + (isDevMode ? ' warn' : '')} style={{ height: 16, fontSize: 10 }}>
                  {isDevMode ? 'enabled' : 'disabled'}
                </span>
              </ConfigRow>
            </Section>

            {/* ── Audit ── */}
            <Section id="audit" title="Audit & Data Retention" desc="How long audit logs and expired sessions are kept">
              <ConfigRow label="Audit Retention" hint="0 = forever. Older entries are purged on the cleanup interval.">
                <TextInput
                  value={form.audit.retention}
                  onChange={v => set('audit.retention', v)}
                  mono
                  placeholder="0 or e.g. 90d"
                  width={140}
                />
              </ConfigRow>
              <ConfigRow label="Cleanup Interval" hint="How often expired sessions and old audit logs are swept">
                <TextInput
                  value={form.audit.cleanup_interval}
                  onChange={v => set('audit.cleanup_interval', v)}
                  mono
                  placeholder="1h"
                  width={140}
                />
              </ConfigRow>
            </Section>

            {/* ── Telemetry ── */}
            <Section id="telemetry" title="Telemetry" desc="Anonymous usage data (ping count, version) — no PII">
              <ConfigRow label="Status" system>
                <div className="row" style={{ gap: 8 }}>
                  <span className={'chip' + (config?.telemetry?.enabled === false ? '' : ' success')} style={{ height: 18, fontSize: 10 }}>
                    {config?.telemetry?.enabled === false ? 'disabled' : 'enabled'}
                  </span>
                  <span className="faint" style={{ fontSize: 11 }}>Edit telemetry.enabled in sharkauth.yaml to toggle</span>
                </div>
              </ConfigRow>
              <ConfigRow label="Endpoint" system>
                <span className="mono" style={{ fontSize: 11.5, color: 'var(--fg-muted)' }}>
                  {config?.telemetry?.endpoint || 'https://telemetry.shark-auth.com/v1/ping'}
                </span>
              </ConfigRow>
            </Section>

            {/* ── Maintenance ── */}
            <Section id="maintenance" title="Maintenance" desc="Manual cleanup operations. Use carefully.">
              <ConfigRow label="Expired Sessions" hint="Remove sessions whose expiry has already passed">
                <button
                  className="btn sm"
                  onClick={handlePurgeSessions}
                  disabled={purging.session}
                >
                  <Icon.Refresh width={11} height={11} style={purging.session ? { animation: 'spin 0.8s linear infinite' } : {}}/>
                  Purge expired sessions
                </button>
              </ConfigRow>
              <ConfigRow label="Audit Logs" hint="Delete entries older than the chosen date — irreversible">
                <div className="row" style={{ gap: 6 }}>
                  <input
                    type="date"
                    value={auditBefore}
                    onChange={e => setAuditBefore(e.target.value)}
                    style={{
                      ...monoInputStyle,
                      colorScheme: 'dark',
                      width: 160,
                    }}
                  />
                  <button
                    className="btn sm danger"
                    onClick={handlePurgeAudit}
                    disabled={!auditBefore || purging.audit}
                  >
                    {purging.audit ? 'Purging…' : 'Purge older'}
                  </button>
                </div>
              </ConfigRow>
            </Section>

          </div>
          <CLIFooter command="shark admin config dump"/>
        </div>
      </div>

      {/* ── Floating dirty bar ── */}
      {isDirty && (
        <div style={{
          position: 'absolute',
          bottom: 20,
          left: '50%',
          transform: 'translateX(-50%)',
          background: 'var(--surface-2)',
          border: '1px solid var(--hairline-bright)',
          borderRadius: 5,
          padding: '8px 10px 8px 14px',
          boxShadow: 'var(--shadow-lg)',
          display: 'flex',
          alignItems: 'center',
          gap: 14,
          zIndex: 100,
          animation: 'slideUp 160ms ease-out',
        }}>
          <span className="dot warn"/>
          <span style={{ fontSize: 12, fontWeight: 500 }}>Unsaved changes</span>
          <div className="row" style={{ gap: 6 }}>
            <button className="btn ghost sm" onClick={handleReset} disabled={saving}>Reset</button>
            <button className="btn primary sm" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving…' : 'Save & reload'}
            </button>
          </div>
        </div>
      )}

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
        @keyframes slideUp { from { transform: translate(-50%, 12px); opacity: 0; } to { transform: translate(-50%, 0); opacity: 1; } }
      `}</style>
    </div>
  );
}
