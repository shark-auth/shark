// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'

// Identity Hub — single config surface for the auth posture.
// Sessions and JWTs are ONE concept here: every authenticated request rides
// a session, and JWTs (when minted) are derived from that same session.
// Tabs slice the (large) config surface into focused panes; everything saves
// through PATCH /admin/config with optimistic patching of unsaved fields.

const HAIRLINE = '1px solid var(--hairline)';
const HAIRLINE_STRONG = '1px solid var(--hairline-strong)';

// Set of dotted paths that have been modified locally — used to render the
// "unsaved" affordance and to power discard/save.
function diffPaths(a: any, b: any, prefix = ''): string[] {
  if (a === b) return [];
  if (a == null || b == null || typeof a !== typeof b) return [prefix];
  if (typeof a !== 'object') return a === b ? [] : [prefix];
  if (Array.isArray(a) || Array.isArray(b)) {
    return JSON.stringify(a) === JSON.stringify(b) ? [] : [prefix];
  }
  const keys = new Set([...Object.keys(a || {}), ...Object.keys(b || {})]);
  const out: string[] = [];
  keys.forEach(k => {
    const next = prefix ? `${prefix}.${k}` : k;
    out.push(...diffPaths(a?.[k], b?.[k], next));
  });
  return out;
}

function get(obj: any, path: string) {
  return path.split('.').reduce((o, k) => (o == null ? o : o[k]), obj);
}

function set(obj: any, path: string, value: any) {
  const parts = path.split('.');
  const next = Array.isArray(obj) ? [...obj] : { ...(obj || {}) };
  let cur = next;
  for (let i = 0; i < parts.length - 1; i++) {
    const k = parts[i];
    cur[k] = cur[k] == null ? {} : (Array.isArray(cur[k]) ? [...cur[k]] : { ...cur[k] });
    cur = cur[k];
  }
  cur[parts[parts.length - 1]] = value;
  return next;
}

const TABS = [
  { id: 'overview', label: 'Overview' },
  { id: 'methods', label: 'Auth methods' },
  { id: 'session', label: 'Session & JWT' },
  { id: 'oauth', label: 'OAuth server' },
  { id: 'mfa', label: 'MFA' },
  { id: 'social', label: 'Social' },
];

export function IdentityHub() {
  const { data, loading, refresh } = useAPI('/admin/config');
  const [form, setForm] = React.useState<any>(null);
  const [busy, setBusy] = React.useState(false);
  const [tab, setTab] = React.useState(() => {
    const m = window.location.pathname.match(/^\/admin\/auth\/([^\/]+)/);
    return m && TABS.find(t => t.id === m[1]) ? m[1] : 'overview';
  });
  const [drawer, setDrawer] = React.useState<null | { kind: string; payload?: any }>(null);
  const toast = useToast();

  React.useEffect(() => {
    if (data && !form) setForm(JSON.parse(JSON.stringify(data)));
  }, [data]);

  const updateTab = (t: string) => {
    setTab(t);
    const url = `/admin/auth${t === 'overview' ? '' : '/' + t}`;
    window.history.replaceState(null, '', url);
  };

  if (loading || !form || !data) {
    return (
      <div style={{ padding: 20 }}>
        <div className="faint" style={{ fontSize: 13 }}>Loading identity configuration…</div>
      </div>
    );
  }

  const dirtyPaths = diffPaths(data, form);
  const dirty = dirtyPaths.length > 0;
  const patch = (path: string, value: any) => setForm((f: any) => set(f, path, value));

  const save = async () => {
    setBusy(true);
    try {
      await API.patch('/admin/config', form);
      toast.success(`Saved ${dirtyPaths.length} change${dirtyPaths.length !== 1 ? 's' : ''}`);
      refresh();
    } catch (e: any) {
      toast.error('Save failed: ' + (e?.message || 'unknown'));
    } finally {
      setBusy(false);
    }
  };

  const discard = () => {
    setForm(JSON.parse(JSON.stringify(data)));
  };

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Header */}
        <div style={{
          padding: '10px 16px',
          borderBottom: HAIRLINE,
          background: 'var(--surface-0)',
          display: 'flex', alignItems: 'center', gap: 10,
        }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
            <h1 style={{
              margin: 0,
              fontSize: 14, fontWeight: 600,
              fontFamily: 'var(--font-display)',
              letterSpacing: '-0.005em',
            }}>Identity</h1>
            <span className="faint mono" style={{ fontSize: 11 }}>auth posture · GET /admin/config</span>
          </div>
          <div style={{ flex: 1 }} />
          <PostureBadge form={form} />
          <span style={{ width: 1, height: 18, background: 'var(--hairline-strong)' }} />
          {dirty ? (
            <>
              <span className="mono" style={{ fontSize: 11, color: 'var(--warn)' }}>
                {dirtyPaths.length} unsaved
              </span>
              <button className="btn ghost sm" onClick={discard} disabled={busy}>Discard</button>
              <button className="btn primary sm" onClick={save} disabled={busy}>
                {busy ? 'Saving…' : 'Apply changes'}
              </button>
            </>
          ) : (
            <span className="faint mono" style={{ fontSize: 11 }}>in sync</span>
          )}
        </div>

        {/* Tabs */}
        <div style={{
          display: 'flex', gap: 0,
          padding: '0 16px',
          borderBottom: HAIRLINE,
          background: 'var(--surface-0)',
        }}>
          {TABS.map(t => {
            const dirtyHere = dirtyPaths.some(p => tabOwns(t.id, p));
            return (
              <button key={t.id} onClick={() => updateTab(t.id)} style={{
                padding: '9px 12px',
                fontSize: 11,
                textTransform: 'uppercase',
                letterSpacing: '0.06em',
                color: tab === t.id ? 'var(--fg)' : 'var(--fg-dim)',
                fontWeight: tab === t.id ? 600 : 400,
                borderBottom: tab === t.id ? '1px solid var(--fg)' : '1px solid transparent',
                marginBottom: -1,
                position: 'relative',
              }}>
                {t.label}
                {dirtyHere && <span style={{
                  position: 'absolute', top: 7, right: 4,
                  width: 5, height: 5, borderRadius: '50%',
                  background: 'var(--warn)',
                }} />}
              </button>
            );
          })}
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: 'auto', background: 'var(--surface-0)' }}>
          {tab === 'overview' && <OverviewTab form={form} data={data} patch={patch} openDrawer={setDrawer} />}
          {tab === 'methods' && <MethodsTab form={form} patch={patch} />}
          {tab === 'session' && <SessionTab form={form} patch={patch} openDrawer={setDrawer} />}
          {tab === 'oauth' && <OAuthTab form={form} patch={patch} />}
          {tab === 'mfa' && <MFATab form={form} patch={patch} />}
          {tab === 'social' && <SocialTab form={form} patch={patch} openDrawer={setDrawer} />}
        </div>
      </div>

      {drawer && (
        <Drawer
          drawer={drawer}
          form={form}
          patch={patch}
          onClose={() => setDrawer(null)}
        />
      )}
    </div>
  );
}

// Map dotted-path prefixes to the tab that "owns" them so we can show
// per-tab unsaved markers. Anything unmatched falls through silently.
function tabOwns(tabId: string, path: string): boolean {
  const map: Record<string, string[]> = {
    methods: ['passkey', 'magic_link', 'password_policy'],
    session: ['session_mode', 'session_lifetime', 'jwt_mode', 'jwt'],
    oauth: ['oauth_server'],
    mfa: ['mfa'],
    social: ['social_providers', 'social'],
  };
  const prefixes = map[tabId] || [];
  return prefixes.some(p => path === p || path.startsWith(p + '.'));
}

// =====================================================================
// Reusable atoms — kept tight & local to this file.
// =====================================================================

function Pane({ children }) {
  return <div style={{ padding: '14px 16px 80px', maxWidth: 980 }}>{children}</div>;
}

function Section({ title, hint, action, children, dense = false }) {
  return (
    <section style={{ marginBottom: dense ? 16 : 24 }}>
      <header style={{
        display: 'flex', alignItems: 'baseline', gap: 8,
        marginBottom: 8, paddingBottom: 6,
        borderBottom: HAIRLINE,
      }}>
        <h2 style={{
          margin: 0, fontSize: 11, fontWeight: 600,
          textTransform: 'uppercase', letterSpacing: '0.08em',
          color: 'var(--fg)',
        }}>{title}</h2>
        {hint && <span className="faint" style={{ fontSize: 11 }}>{hint}</span>}
        <div style={{ flex: 1 }} />
        {action}
      </header>
      {children}
    </section>
  );
}

function Row({ label, hint, children, mono = false }) {
  return (
    <div style={{
      display: 'grid', gridTemplateColumns: '180px 1fr',
      alignItems: 'center', gap: 12,
      padding: '7px 0',
      borderBottom: HAIRLINE,
      minHeight: 32,
    }}>
      <div style={{ minWidth: 0 }}>
        <div style={{ fontSize: 12, color: 'var(--fg)', fontWeight: 500 }}>{label}</div>
        {hint && <div className="faint" style={{ fontSize: 11, lineHeight: 1.4, marginTop: 1 }}>{hint}</div>}
      </div>
      <div className={mono ? 'mono' : ''} style={{ fontSize: 12 }}>{children}</div>
    </div>
  );
}

function TextField({ value, onChange, placeholder, type = 'text', width, mono = false, disabled = false }) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={e => onChange(type === 'number' ? Number(e.target.value) : e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      className={mono ? 'mono' : ''}
      style={{
        width: width || '100%',
        height: 26, padding: '0 8px',
        background: disabled ? 'var(--surface-0)' : 'var(--surface-1)',
        border: HAIRLINE_STRONG, borderRadius: 4,
        fontSize: 12, color: 'var(--fg)',
        outline: 'none',
      }}
    />
  );
}

function Toggle({ value, onChange, disabled = false, label = null }) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onChange(!value)}
      style={{
        height: 18, width: 32,
        padding: 2, position: 'relative',
        background: value ? 'var(--success)' : 'var(--surface-3)',
        border: '1px solid ' + (value ? 'transparent' : 'var(--hairline-strong)'),
        borderRadius: 3, cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.4 : 1,
        display: 'inline-flex', alignItems: 'center',
      }}
      aria-label={label || ''}
    >
      <span style={{
        position: 'absolute', top: 2, left: value ? 16 : 2,
        width: 12, height: 12,
        background: value ? '#000' : 'var(--fg)',
        borderRadius: 2,
        transition: 'left 100ms ease-out',
      }} />
    </button>
  );
}

function SegBtn({ items, value, onChange, size = 'md' }) {
  return (
    <div className="seg" style={{ display: 'inline-flex', border: HAIRLINE_STRONG, borderRadius: 4, overflow: 'hidden' }}>
      {items.map((it, i) => {
        const on = value === it.value;
        return (
          <button
            key={it.value}
            type="button"
            onClick={() => onChange(it.value)}
            style={{
              padding: size === 'sm' ? '3px 8px' : '4px 10px',
              fontSize: size === 'sm' ? 11 : 12,
              fontWeight: on ? 600 : 400,
              background: on ? 'var(--fg)' : 'transparent',
              color: on ? 'var(--bg)' : 'var(--fg-dim)',
              borderRight: i < items.length - 1 ? HAIRLINE_STRONG : 'none',
              cursor: 'pointer',
            }}
          >{it.label}</button>
        );
      })}
    </div>
  );
}

function Stat({ label, value, valueColor = null, icon = null, mono = true }) {
  return (
    <div style={{
      padding: '10px 12px',
      background: 'var(--surface-1)',
      border: HAIRLINE,
      borderRadius: 4,
      minWidth: 0,
    }}>
      <div className="row" style={{ gap: 6, marginBottom: 4 }}>
        {icon}
        <div style={{
          fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
          color: 'var(--fg-muted)', fontWeight: 600,
        }}>{label}</div>
      </div>
      <div className={mono ? 'mono' : ''} style={{
        fontSize: 14, fontWeight: 500, color: valueColor || 'var(--fg)',
        whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis',
      }}>{value}</div>
    </div>
  );
}

// =====================================================================
// Posture badge — single-glance summary of the auth strength.
// =====================================================================

function computePosture(form: any) {
  let score = 0;
  let max = 6;
  const items: { ok: boolean; label: string }[] = [];

  // 1. Strong password policy
  const pp = form.password_policy || {};
  const ppOk = (pp.min_length || 0) >= 12 && (pp.require_upper || pp.require_lower || pp.require_digit || pp.require_symbol);
  items.push({ ok: ppOk, label: ppOk ? 'Strong password policy' : 'Weak password policy' });
  if (ppOk) score++;

  // 2. Passkey enabled
  const pk = (form.passkey || {}).enabled;
  items.push({ ok: !!pk, label: pk ? 'Passkeys enabled' : 'Passkeys disabled' });
  if (pk) score++;

  // 3. MFA configured
  const mfa = form.mfa && (form.mfa.totp_enabled || form.mfa.recovery_codes);
  items.push({ ok: !!mfa, label: mfa ? 'MFA configured' : 'MFA not configured' });
  if (mfa) score++;

  // 4. JWT signing alg is asymmetric
  const alg = (form.jwt || {}).algorithm || '';
  const algOk = /^(ES|RS|PS|EdDSA)/i.test(alg);
  items.push({ ok: algOk, label: algOk ? 'Asymmetric JWT signing' : 'Symmetric / unset JWT signing' });
  if (algOk) score++;

  // 5. DPoP available
  const dpop = (form.oauth_server || {}).require_dpop || (form.oauth_server || {}).dpop_supported;
  items.push({ ok: !!dpop, label: dpop ? 'DPoP enabled' : 'DPoP not required' });
  if (dpop) score++;

  // 6. Session lifetime sane (≤30d)
  const lifetime = String(form.session_lifetime || '');
  const sane = /^([1-9]|[12]\d|30)d$|^\d+h$|^\d+m$/.test(lifetime);
  items.push({ ok: sane, label: sane ? `Session ≤30d (${lifetime})` : `Long-lived sessions (${lifetime || 'unset'})` });
  if (sane) score++;

  let level: 'strong' | 'fair' | 'weak' = 'weak';
  if (score >= 5) level = 'strong';
  else if (score >= 3) level = 'fair';

  return { score, max, items, level };
}

function PostureBadge({ form }) {
  const p = computePosture(form);
  const color = p.level === 'strong' ? 'var(--success)' : p.level === 'fair' ? 'var(--warn)' : 'var(--danger)';
  return (
    <div style={{
      display: 'inline-flex', alignItems: 'center', gap: 6,
      padding: '2px 8px', height: 22,
      background: 'var(--surface-1)',
      border: HAIRLINE_STRONG, borderRadius: 3,
    }}>
      <span style={{
        width: 6, height: 6, borderRadius: '50%', background: color,
      }} />
      <span style={{ fontSize: 11, fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.04em' }}>
        Posture {p.level}
      </span>
      <span className="mono faint" style={{ fontSize: 11 }}>
        {p.score}/{p.max}
      </span>
    </div>
  );
}

// =====================================================================
// Tabs
// =====================================================================

function OverviewTab({ form, data, patch, openDrawer }) {
  const posture = computePosture(form);
  const passkey = form.passkey || {};
  const jwt = form.jwt || {};
  const oauth = form.oauth_server || {};
  const social = form.social_providers || [];
  const enabledSocial = social.filter((p: any) => p && p.enabled).map((p: any) => p.id || p.name);

  return (
    <Pane>
      <Section title="Auth posture" hint="6 dimensions of identity strength">
        <div style={{
          display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))',
          gap: 8, marginBottom: 12,
        }}>
          <Stat label="Score" value={`${posture.score} / ${posture.max}`}
                valueColor={posture.level === 'strong' ? 'var(--success)' : posture.level === 'fair' ? 'var(--warn)' : 'var(--danger)'} />
          <Stat label="Session mode" value={form.session_mode || 'cookie'} mono />
          <Stat label="JWT algorithm" value={jwt.algorithm || '—'} mono />
          <Stat label="Active keys" value={String((jwt.active_keys || []).length || jwt.active_keys || 0)} mono />
          <Stat label="Lifetime" value={form.session_lifetime || '—'} mono />
          <Stat label="Smtp" value={form.smtp_configured ? 'configured' : 'unset'}
                valueColor={form.smtp_configured ? 'var(--success)' : 'var(--warn)'} />
        </div>

        <div style={{
          display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 6,
          padding: 10, background: 'var(--surface-1)', border: HAIRLINE, borderRadius: 4,
        }}>
          {posture.items.map((it, i) => (
            <div key={i} className="row" style={{ gap: 6, fontSize: 12 }}>
              {it.ok ? (
                <Icon.Check width={11} height={11} style={{ color: 'var(--success)' }}/>
              ) : (
                <Icon.X width={11} height={11} style={{ color: 'var(--warn)' }}/>
              )}
              <span style={{ color: it.ok ? 'var(--fg)' : 'var(--fg-muted)' }}>{it.label}</span>
            </div>
          ))}
        </div>
      </Section>

      <Section title="Active surface" hint="What end-users will see">
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 8 }}>
          <SurfaceCard
            on={true}
            title="Password"
            sub={`min ${form.password_policy?.min_length || 8} chars`}
          />
          <SurfaceCard
            on={!!passkey.enabled}
            title="Passkeys"
            sub={passkey.enabled ? `rp ${passkey.rp_id || 'unset'}` : 'disabled'}
          />
          <SurfaceCard
            on={!!form.magic_link?.ttl}
            title="Magic link"
            sub={form.magic_link?.ttl ? `ttl ${form.magic_link.ttl}` : 'disabled'}
          />
          <SurfaceCard
            on={enabledSocial.length > 0}
            title={`Social (${enabledSocial.length})`}
            sub={enabledSocial.slice(0, 3).join(', ') || 'none configured'}
          />
          <SurfaceCard
            on={!!oauth.enabled}
            title="OAuth server"
            sub={oauth.enabled ? `${oauth.signing_algorithm || 'ES256'} · ${oauth.access_token_lifetime || '15m'}` : 'disabled'}
          />
          <SurfaceCard
            on={!!form.mfa?.totp_enabled}
            title="MFA / TOTP"
            sub={form.mfa?.totp_enabled ? `issuer ${form.mfa.issuer || '—'}` : 'optional'}
          />
        </div>
      </Section>

      <Section title="Issuer / base URL" hint="Used in JWT iss claim, OAuth metadata, callback URLs">
        <Row label="Base URL" hint="Public origin of this Shark instance">
          <TextField value={form.base_url} onChange={v => patch('base_url', v)} mono placeholder="https://auth.example.com" />
        </Row>
        <Row label="Dev mode" hint="Bypasses TLS / cookie-secure flags. Never enable in production.">
          <div className="row" style={{ gap: 8 }}>
            <Toggle value={!!form.dev_mode} onChange={() => {}} disabled />
            <span className="faint mono" style={{ fontSize: 11 }}>read-only · set via config file</span>
          </div>
        </Row>
      </Section>
    </Pane>
  );
}

function SurfaceCard({ on, title, sub }) {
  return (
    <div style={{
      padding: 10,
      background: 'var(--surface-1)',
      border: on ? HAIRLINE_STRONG : '1px dashed var(--hairline-strong)',
      borderRadius: 4,
      opacity: on ? 1 : 0.55,
    }}>
      <div className="row" style={{ gap: 6, marginBottom: 4 }}>
        <span style={{
          width: 6, height: 6, borderRadius: '50%',
          background: on ? 'var(--success)' : 'var(--fg-muted)',
        }} />
        <span style={{ fontSize: 12, fontWeight: 600 }}>{title}</span>
      </div>
      <div className="faint mono" style={{ fontSize: 11 }}>{sub}</div>
    </div>
  );
}

function MethodsTab({ form, patch }) {
  const pp = form.password_policy || {};
  const pk = form.passkey || {};
  const ml = form.magic_link || {};

  return (
    <Pane>
      <Section title="Password policy" hint="Applies to new passwords and resets">
        <Row label="Minimum length" hint="NIST recommends ≥ 12 characters">
          <div className="row" style={{ gap: 8 }}>
            <TextField value={pp.min_length} onChange={v => patch('password_policy.min_length', v)} type="number" width={80} mono />
            <span className="faint" style={{ fontSize: 11 }}>characters</span>
          </div>
        </Row>
        <Row label="Composition" hint="Each requirement increases brute-force cost">
          <div className="row" style={{ gap: 6, flexWrap: 'wrap' }}>
            <CheckChip on={!!pp.require_upper} onChange={v => patch('password_policy.require_upper', v)} label="A-Z" />
            <CheckChip on={!!pp.require_lower} onChange={v => patch('password_policy.require_lower', v)} label="a-z" />
            <CheckChip on={!!pp.require_digit} onChange={v => patch('password_policy.require_digit', v)} label="0-9" />
            <CheckChip on={!!pp.require_symbol} onChange={v => patch('password_policy.require_symbol', v)} label="!@#" />
          </div>
        </Row>
        <Row label="Reset link TTL" hint="How long a password-reset email link stays valid">
          <TextField value={form.password_reset?.token_lifetime} onChange={v => patch('password_reset.token_lifetime', v)} width={120} mono placeholder="30m" />
        </Row>
        <Row label="Reset redirect" hint="Where the reset email sends users">
          <TextField value={form.password_reset?.redirect_url} onChange={v => patch('password_reset.redirect_url', v)} mono />
        </Row>
      </Section>

      <Section title="Passkeys (WebAuthn)" hint="Phishing-resistant, public-key auth">
        <Row label="Enabled">
          <Toggle value={!!pk.enabled} onChange={v => patch('passkey.enabled', v)} />
        </Row>
        <Row label="Relying-party ID" hint="Public domain serving auth — usually equals base URL host">
          <TextField value={pk.rp_id} onChange={v => patch('passkey.rp_id', v)} disabled={!pk.enabled} mono placeholder="auth.example.com" />
        </Row>
        <Row label="Relying-party name" hint="Shown in the platform credential picker">
          <TextField value={pk.rp_name} onChange={v => patch('passkey.rp_name', v)} disabled={!pk.enabled} placeholder="SharkAuth" />
        </Row>
        <Row label="Origin" hint="Full origin URL — must match exactly, scheme included">
          <TextField value={pk.origin} onChange={v => patch('passkey.origin', v)} disabled={!pk.enabled} mono placeholder="https://auth.example.com" />
        </Row>
        <Row label="User verification" hint="Whether the authenticator must verify the user (PIN/biometric)">
          <SegBtn
            value={pk.user_verification || 'preferred'}
            onChange={v => patch('passkey.user_verification', v)}
            items={[
              { value: 'discouraged', label: 'discouraged' },
              { value: 'preferred', label: 'preferred' },
              { value: 'required', label: 'required' },
            ]}
          />
        </Row>
        <Row label="Attestation" hint="Whether to require attestation statements from authenticators">
          <SegBtn
            value={pk.attestation || 'none'}
            onChange={v => patch('passkey.attestation', v)}
            items={[
              { value: 'none', label: 'none' },
              { value: 'indirect', label: 'indirect' },
              { value: 'direct', label: 'direct' },
            ]}
          />
        </Row>
      </Section>

      <Section title="Magic link" hint="Email-only sign-in. Requires SMTP.">
        <Row label="Token lifetime" hint="How long the email link is valid">
          <TextField value={ml.ttl || ml.token_lifetime} onChange={v => { patch('magic_link.ttl', v); patch('magic_link.token_lifetime', v); }} width={120} mono placeholder="10m" />
        </Row>
        <Row label="Redirect URL" hint="Where the magic link lands the user after success">
          <TextField value={ml.redirect_url} onChange={v => patch('magic_link.redirect_url', v)} mono />
        </Row>
        {!form.smtp_configured && (
          <div style={{
            marginTop: 10, padding: 10,
            background: 'color-mix(in oklch, var(--warn) 10%, var(--surface-1))',
            border: '1px solid color-mix(in oklch, var(--warn) 35%, var(--hairline))',
            borderRadius: 4,
            display: 'flex', gap: 8, alignItems: 'center',
            fontSize: 12,
          }}>
            <Icon.Warn width={14} height={14} style={{ color: 'var(--warn)' }} />
            <span>SMTP is unconfigured — magic links cannot be delivered. Configure SMTP under <span className="mono">Settings → Email</span>.</span>
          </div>
        )}
      </Section>
    </Pane>
  );
}

function SessionTab({ form, patch, openDrawer }) {
  const jwt = form.jwt || {};
  const activeKeys = Array.isArray(jwt.active_keys) ? jwt.active_keys : [];

  return (
    <Pane>
      <Section title="Unified session model" hint="Sessions and JWTs are one — JWTs are derived from sessions">
        <div style={{
          padding: 10, marginBottom: 10,
          background: 'var(--surface-1)', border: HAIRLINE, borderRadius: 4,
          fontSize: 12, lineHeight: 1.55, color: 'var(--fg-muted)',
        }}>
          <div className="row" style={{ gap: 6, marginBottom: 4 }}>
            <Icon.Info width={12} height={12} style={{ opacity: 0.7 }} />
            <span style={{ color: 'var(--fg)', fontWeight: 500 }}>How this works</span>
          </div>
          Every authenticated request rides a session. When a JWT is issued, it inherits the session's identity, scopes, and lifetime — revoking the session revokes its tokens. There is no "JWT-only mode": tokens are always backed by session state.
        </div>

        <Row label="Session lifetime" hint="Maximum time a session stays valid before re-auth">
          <TextField value={form.session_lifetime} onChange={v => patch('session_lifetime', v)} width={120} mono placeholder="30d" />
        </Row>
        <Row label="Transport" hint="Default mechanism for browser sessions — cookie is recommended for web apps">
          <SegBtn
            value={form.session_mode || 'cookie'}
            onChange={v => patch('session_mode', v)}
            items={[
              { value: 'cookie', label: 'HTTP-only cookie' },
              { value: 'jwt', label: 'JWT in Authorization header' },
            ]}
          />
        </Row>
        <Row label="Token shape" hint="Single bearer token vs. short-lived access + long refresh — both still backed by the same session">
          <SegBtn
            value={form.jwt_mode || jwt.mode || 'session'}
            onChange={v => { patch('jwt_mode', v); patch('jwt.mode', v); }}
            items={[
              { value: 'session', label: 'Single (session-bound)' },
              { value: 'access_refresh', label: 'Access + refresh' },
            ]}
          />
        </Row>
      </Section>

      <Section title="JWT issuance" hint="Same session, different surface">
        <Row label="Issuer (iss)" hint="Set in token claims — defaults to base URL">
          <TextField value={jwt.issuer} onChange={v => patch('jwt.issuer', v)} mono placeholder={form.base_url} />
        </Row>
        <Row label="Audience (aud)" hint="Default audience claim — services check this on verification">
          <TextField value={jwt.audience} onChange={v => patch('jwt.audience', v)} mono placeholder="shark" />
        </Row>
        <Row label="Algorithm" hint="Asymmetric (ES256 / EdDSA / RS256) is required for clients to verify offline via JWKS">
          <SegBtn
            value={jwt.algorithm || 'ES256'}
            onChange={v => patch('jwt.algorithm', v)}
            items={[
              { value: 'ES256', label: 'ES256' },
              { value: 'EdDSA', label: 'EdDSA' },
              { value: 'RS256', label: 'RS256' },
              { value: 'HS256', label: 'HS256' },
            ]}
          />
        </Row>
        {(form.jwt_mode === 'access_refresh' || jwt.mode === 'access_refresh') && (
          <>
            <Row label="Access token TTL" hint="Short — services revalidate often">
              <TextField value={jwt.access_token_ttl || jwt.lifetime} onChange={v => patch('jwt.access_token_ttl', v)} width={120} mono placeholder="15m" />
            </Row>
            <Row label="Refresh token TTL" hint="Long — used to mint new access tokens without re-auth">
              <TextField value={jwt.refresh_token_ttl} onChange={v => patch('jwt.refresh_token_ttl', v)} width={120} mono placeholder="30d" />
            </Row>
          </>
        )}
        <Row label="Clock skew tolerance" hint="Grace period for nbf/exp on token verification">
          <TextField value={jwt.clock_skew} onChange={v => patch('jwt.clock_skew', v)} width={120} mono placeholder="30s" />
        </Row>
        <Row label="Per-request revocation" hint="Check session state on every JWT request — slower but instant logout">
          <Toggle value={!!jwt.revocation?.check_per_request} onChange={v => patch('jwt.revocation.check_per_request', v)} />
        </Row>
      </Section>

      <Section
        title="Signing keys"
        hint={`${activeKeys.length || 0} active key${activeKeys.length === 1 ? '' : 's'}`}
        action={
          <button className="btn ghost sm" onClick={() => openDrawer({ kind: 'rotate-keys' })}>
            <Icon.Refresh width={11} height={11}/> Rotate
          </button>
        }
      >
        {activeKeys.length === 0 ? (
          <div className="faint" style={{ fontSize: 12, padding: '8px 0' }}>
            No active signing keys exposed in config. View JWKS at <span className="mono">{form.base_url}/.well-known/jwks.json</span>.
          </div>
        ) : (
          <div style={{ border: HAIRLINE, borderRadius: 4, overflow: 'hidden' }}>
            {activeKeys.map((k: any, i: number) => (
              <div key={i} className="row" style={{
                padding: '8px 12px',
                borderBottom: i < activeKeys.length - 1 ? HAIRLINE : 'none',
                fontSize: 12, gap: 12,
              }}>
                <span className="mono" style={{ flex: 1, color: 'var(--fg)' }}>{k.kid || k.id || `key-${i}`}</span>
                <span className="chip" style={{ height: 18 }}>{k.alg || jwt.algorithm}</span>
                <span className="faint mono" style={{ fontSize: 11 }}>{k.use || 'sig'}</span>
              </div>
            ))}
          </div>
        )}
      </Section>
    </Pane>
  );
}

function OAuthTab({ form, patch }) {
  const o = form.oauth_server || {};

  return (
    <Pane>
      <Section title="OAuth 2.1 server" hint="Issue tokens to third-party agents and applications">
        <Row label="Enabled" hint="Master switch for /oauth/* endpoints">
          <Toggle value={!!o.enabled} onChange={v => patch('oauth_server.enabled', v)} />
        </Row>
        <Row label="Issuer" hint="iss claim — used in /.well-known/oauth-authorization-server">
          <TextField value={o.issuer} onChange={v => patch('oauth_server.issuer', v)} disabled={!o.enabled} mono placeholder={form.base_url} />
        </Row>
        <Row label="Signing algorithm" hint="Used for tokens issued through this server">
          <SegBtn
            value={o.signing_algorithm || 'ES256'}
            onChange={v => patch('oauth_server.signing_algorithm', v)}
            items={[
              { value: 'ES256', label: 'ES256' },
              { value: 'EdDSA', label: 'EdDSA' },
              { value: 'RS256', label: 'RS256' },
            ]}
          />
        </Row>
      </Section>

      <Section title="Token lifetimes">
        <Row label="Access token" hint="Short-lived bearer used to call APIs">
          <TextField value={o.access_token_lifetime} onChange={v => patch('oauth_server.access_token_lifetime', v)} disabled={!o.enabled} width={120} mono placeholder="15m" />
        </Row>
        <Row label="Refresh token" hint="Used by the agent to renew without user interaction">
          <TextField value={o.refresh_token_lifetime} onChange={v => patch('oauth_server.refresh_token_lifetime', v)} disabled={!o.enabled} width={120} mono placeholder="30d" />
        </Row>
        <Row label="Authorization code" hint="One-time code exchanged for tokens — keep this short">
          <TextField value={o.auth_code_lifetime} onChange={v => patch('oauth_server.auth_code_lifetime', v)} disabled={!o.enabled} width={120} mono placeholder="60s" />
        </Row>
        <Row label="Device code" hint="For headless device flow (CLI tools, smart devices)">
          <TextField value={o.device_code_lifetime} onChange={v => patch('oauth_server.device_code_lifetime', v)} disabled={!o.enabled} width={120} mono placeholder="15m" />
        </Row>
      </Section>

      <Section title="DPoP — proof-of-possession" hint="Binds tokens to a client-held key. Stops bearer-token theft.">
        <Row label="Require DPoP" hint="When on, all OAuth tokens must be presented with a DPoP proof">
          <Toggle value={!!o.require_dpop} onChange={v => patch('oauth_server.require_dpop', v)} disabled={!o.enabled} />
        </Row>
        <div style={{
          marginTop: 6, padding: 10,
          background: 'var(--surface-1)', border: HAIRLINE, borderRadius: 4,
          fontSize: 11, lineHeight: 1.55, color: 'var(--fg-muted)',
        }}>
          DPoP (RFC 9449) is recommended for agent + mobile clients. Requires client SDK support — most libraries adopted it in 2024. Enable per-client first, then make global.
        </div>
      </Section>

      <Section title="Consent">
        <Row label="Custom consent template" hint="Path to an HTML template — leave blank for the default screen">
          <TextField value={o.consent_template} onChange={v => patch('oauth_server.consent_template', v)} disabled={!o.enabled} mono placeholder="(default)" />
        </Row>
      </Section>
    </Pane>
  );
}

function MFATab({ form, patch }) {
  const m = form.mfa || {};
  return (
    <Pane>
      <Section title="TOTP (authenticator apps)" hint="6-digit codes from Google Authenticator, 1Password, etc.">
        <Row label="TOTP enabled" hint="Lets users enroll an authenticator app on their account">
          <Toggle value={!!m.totp_enabled} onChange={v => patch('mfa.totp_enabled', v)} />
        </Row>
        <Row label="Issuer label" hint="Shown inside authenticator apps as the account issuer">
          <TextField value={m.issuer} onChange={v => patch('mfa.issuer', v)} placeholder="SharkAuth" />
        </Row>
        <Row label="Recovery codes" hint="One-time codes issued at enrollment for account recovery">
          <div className="row" style={{ gap: 8 }}>
            <TextField value={m.recovery_codes} onChange={v => patch('mfa.recovery_codes', v)} type="number" width={80} mono />
            <span className="faint" style={{ fontSize: 11 }}>codes per user</span>
          </div>
        </Row>
      </Section>

      <Section title="Enforcement" hint="Who must complete MFA on sign-in">
        <Row label="Policy">
          <SegBtn
            value={m.enforcement || 'optional'}
            onChange={v => patch('mfa.enforcement', v)}
            items={[
              { value: 'optional', label: 'Optional' },
              { value: 'risk_based', label: 'Risk-based' },
              { value: 'required', label: 'Required' },
            ]}
          />
        </Row>
        {m.enforcement === 'risk_based' && (
          <>
            <Row label="Trigger on new device" hint="Prompt for MFA when a session starts from an unrecognized fingerprint">
              <Toggle value={!!m.risk_new_device} onChange={v => patch('mfa.risk_new_device', v)} />
            </Row>
            <Row label="Trigger on geo change" hint="Prompt when the login country differs from the last successful login">
              <Toggle value={!!m.risk_geo_change} onChange={v => patch('mfa.risk_geo_change', v)} />
            </Row>
            <Row label="Trigger on impossible travel" hint="Distance/time between logins implies physical impossibility">
              <Toggle value={!!m.risk_impossible_travel} onChange={v => patch('mfa.risk_impossible_travel', v)} />
            </Row>
          </>
        )}
        <Row label="Require for admin actions" hint="Even on an active session, force MFA before sensitive admin operations">
          <Toggle value={!!m.require_for_admin} onChange={v => patch('mfa.require_for_admin', v)} />
        </Row>
      </Section>
    </Pane>
  );
}

function SocialTab({ form, patch, openDrawer }) {
  const social = form.social || {};
  const providers = [
    { id: 'google', label: 'Google', cfg: social.google },
    { id: 'github', label: 'GitHub', cfg: social.github },
    { id: 'apple', label: 'Apple', cfg: social.apple },
    { id: 'discord', label: 'Discord', cfg: social.discord },
  ];

  return (
    <Pane>
      <Section title="Redirect" hint="Where social callbacks land the user after success">
        <Row label="Redirect URL">
          <TextField value={social.redirect_url} onChange={v => patch('social.redirect_url', v)} mono placeholder="https://app.example.com/auth/callback" />
        </Row>
      </Section>

      <Section title="Providers" hint="OIDC / OAuth2 social sign-in">
        <div style={{ border: HAIRLINE, borderRadius: 4, overflow: 'hidden' }}>
          <div style={{
            display: 'grid', gridTemplateColumns: '180px 1fr 100px 80px',
            padding: '6px 12px', background: 'var(--surface-1)',
            fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.06em',
            color: 'var(--fg-muted)', fontWeight: 600,
            borderBottom: HAIRLINE,
          }}>
            <span>Provider</span>
            <span>Client ID</span>
            <span>Status</span>
            <span></span>
          </div>
          {providers.map((p, i) => {
            const configured = !!(p.cfg?.client_id);
            return (
              <div key={p.id} style={{
                display: 'grid', gridTemplateColumns: '180px 1fr 100px 80px',
                padding: '8px 12px',
                borderBottom: i < providers.length - 1 ? HAIRLINE : 'none',
                alignItems: 'center', fontSize: 12, gap: 12,
              }}>
                <span style={{ fontWeight: 500 }}>{p.label}</span>
                <span className="mono faint" style={{
                  fontSize: 11, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                }}>
                  {p.cfg?.client_id ? p.cfg.client_id : '— not configured'}
                </span>
                <span className="chip" style={{
                  height: 18,
                  background: configured ? 'color-mix(in oklch, var(--success) 15%, var(--surface-1))' : 'var(--surface-2)',
                  color: configured ? 'var(--success)' : 'var(--fg-muted)',
                  borderColor: configured ? 'color-mix(in oklch, var(--success) 35%, var(--hairline))' : 'var(--hairline-strong)',
                }}>
                  {configured ? 'live' : 'inactive'}
                </span>
                <button
                  className="btn ghost sm"
                  onClick={() => openDrawer({ kind: 'social-provider', payload: p })}
                >Configure</button>
              </div>
            );
          })}
        </div>
      </Section>
    </Pane>
  );
}

// =====================================================================
// Drawer — used for inspecting / editing dense per-item config (social
// providers, key rotation, etc.). Right-side, 420px, hairline-bordered.
// =====================================================================

function Drawer({ drawer, form, patch, onClose }) {
  React.useEffect(() => {
    const onKey = (e: any) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  return (
    <>
      <div
        onClick={onClose}
        style={{
          position: 'fixed', inset: 0, zIndex: 40,
          background: 'rgba(0,0,0,0.4)',
        }}
      />
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0,
        width: 420, maxWidth: '100vw', zIndex: 41,
        background: 'var(--surface-0)',
        borderLeft: HAIRLINE,
        display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        {drawer.kind === 'social-provider' && (
          <SocialProviderDrawer provider={drawer.payload} form={form} patch={patch} onClose={onClose} />
        )}
        {drawer.kind === 'rotate-keys' && (
          <RotateKeysDrawer onClose={onClose} />
        )}
      </div>
    </>
  );
}

function SocialProviderDrawer({ provider, form, patch, onClose }) {
  const path = `social.${provider.id}`;
  const cfg = (form.social || {})[provider.id] || {};
  const callback = `${form.base_url || ''}/auth/${provider.id}/callback`;

  return (
    <>
      <div style={{ padding: 14, borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 10 }}>
          <button className="btn ghost sm" onClick={onClose}>
            <Icon.X width={12} height={12} /> Close
          </button>
          <span className="faint mono" style={{ fontSize: 11 }}>social.{provider.id}</span>
        </div>
        <div style={{
          fontSize: 17, fontWeight: 500,
          fontFamily: 'var(--font-display)',
          letterSpacing: '-0.01em',
        }}>{provider.label}</div>
        <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
          OIDC / OAuth2 client credentials
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: 14 }}>
        <Row label="Client ID">
          <TextField value={cfg.client_id} onChange={v => patch(`${path}.client_id`, v)} mono />
        </Row>
        <Row label="Client secret" hint="Stored encrypted at rest">
          <TextField value={cfg.client_secret} onChange={v => patch(`${path}.client_secret`, v)} type="password" mono placeholder="••••••••" />
        </Row>
        {provider.id === 'apple' && (
          <>
            <Row label="Team ID">
              <TextField value={cfg.team_id} onChange={v => patch(`${path}.team_id`, v)} mono />
            </Row>
            <Row label="Key ID">
              <TextField value={cfg.key_id} onChange={v => patch(`${path}.key_id`, v)} mono />
            </Row>
            <Row label="Private key path">
              <TextField value={cfg.private_key_path} onChange={v => patch(`${path}.private_key_path`, v)} mono placeholder="/var/secrets/apple.p8" />
            </Row>
          </>
        )}
        <Row label="Scopes" hint="Comma-separated">
          <TextField
            value={Array.isArray(cfg.scopes) ? cfg.scopes.join(',') : (cfg.scopes || '')}
            onChange={v => patch(`${path}.scopes`, v.split(',').map(s => s.trim()).filter(Boolean))}
            mono placeholder="openid,email,profile"
          />
        </Row>

        <div style={{
          marginTop: 16, padding: 10,
          background: 'var(--surface-1)', border: HAIRLINE, borderRadius: 4,
        }}>
          <div style={{
            fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
            color: 'var(--fg-muted)', fontWeight: 600, marginBottom: 6,
          }}>Add this to your provider</div>
          <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginBottom: 4 }}>Authorized redirect URI</div>
          <CopyField value={callback} truncate={0} />
        </div>
      </div>

      <div style={{
        padding: 12, borderTop: HAIRLINE,
        display: 'flex', justifyContent: 'flex-end', gap: 8,
      }}>
        <button className="btn sm ghost" onClick={onClose}>Done</button>
      </div>
    </>
  );
}

function RotateKeysDrawer({ onClose }) {
  const toast = useToast();
  const [busy, setBusy] = React.useState(false);

  const rotate = async () => {
    setBusy(true);
    try {
      await API.post('/admin/jwt/rotate', {});
      toast.success('Signing keys rotated. Old key kept for verification.');
      onClose();
    } catch (e: any) {
      toast.error('Rotation failed: ' + (e?.message || 'unknown'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div style={{ padding: 14, borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 10 }}>
          <button className="btn ghost sm" onClick={onClose}>
            <Icon.X width={12} height={12} /> Close
          </button>
          <span className="faint mono" style={{ fontSize: 11 }}>POST /admin/jwt/rotate</span>
        </div>
        <div style={{
          fontSize: 17, fontWeight: 500,
          fontFamily: 'var(--font-display)',
          letterSpacing: '-0.01em',
        }}>Rotate signing keys</div>
        <div className="faint" style={{ fontSize: 12, marginTop: 2 }}>
          New tokens will be signed with a fresh key
        </div>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: 14, fontSize: 12, lineHeight: 1.55 }}>
        <p style={{ marginTop: 0 }}>Generates a new keypair, marks it active, and demotes the previous active key to verify-only. Old tokens remain valid until their natural expiry.</p>
        <ol style={{ paddingLeft: 18, color: 'var(--fg-muted)' }}>
          <li>New key becomes active for signing.</li>
          <li>Previous key is retained in JWKS for verification.</li>
          <li>Tokens issued before rotation continue to verify.</li>
          <li>After all old tokens expire, drop the previous key.</li>
        </ol>
        <div style={{
          marginTop: 12, padding: 10,
          background: 'color-mix(in oklch, var(--warn) 8%, var(--surface-1))',
          border: '1px solid color-mix(in oklch, var(--warn) 30%, var(--hairline))',
          borderRadius: 4,
        }}>
          <div className="row" style={{ gap: 6, marginBottom: 4 }}>
            <Icon.Warn width={12} height={12} style={{ color: 'var(--warn)' }} />
            <span style={{ fontWeight: 600, color: 'var(--warn)' }}>Effects propagation</span>
          </div>
          <div style={{ color: 'var(--fg-muted)' }}>
            Clients that cache JWKS for &gt; 60s may briefly fail to verify new tokens. Bump cache-control / refresh JWKS in your services before rotation.
          </div>
        </div>
      </div>

      <div style={{
        padding: 12, borderTop: HAIRLINE,
        display: 'flex', justifyContent: 'flex-end', gap: 8,
      }}>
        <button className="btn sm ghost" onClick={onClose} disabled={busy}>Cancel</button>
        <button className="btn primary sm" onClick={rotate} disabled={busy}>
          {busy ? 'Rotating…' : 'Rotate now'}
        </button>
      </div>
    </>
  );
}

// =====================================================================
// Misc atoms
// =====================================================================

function CheckChip({ on, onChange, label }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!on)}
      className={'chip' + (on ? ' solid' : '')}
      style={{
        height: 22, padding: '0 8px',
        cursor: 'pointer',
        border: '1px solid ' + (on ? 'transparent' : 'var(--hairline-strong)'),
        background: on ? 'var(--fg)' : 'transparent',
        color: on ? 'var(--bg)' : 'var(--fg-muted)',
        fontSize: 11, fontFamily: 'var(--font-mono)',
        display: 'inline-flex', alignItems: 'center', gap: 4,
        borderRadius: 3,
      }}
    >
      {on && <Icon.Check width={9} height={9} />}
      {label}
    </button>
  );
}
