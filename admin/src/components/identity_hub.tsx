// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'

// IdentityHub — single scrollable identity configuration surface.
// Monochrome, square, dense. Sibling page to users.tsx — same row patterns,
// same hairlines, same input styling. Cookie sessions and JWT issuance are
// BOTH always on; no "validation mode" toggle is exposed. Persists to the
// backend through PATCH /admin/config; signing-key rotate goes to
// POST /admin/auth/rotate-signing-key.

const SECTIONS = [
  { id: 'methods',  label: 'Authentication methods' },
  { id: 'tokens',   label: 'Sessions & tokens' },
  { id: 'mfa',      label: 'MFA' },
  { id: 'oauth',    label: 'OAuth server' },
];

// ── Inline form primitives — match users.tsx Field/Input ────────────────────

function FieldLabel({ children, hint }) {
  return (
    <div style={{ marginBottom: 4 }}>
      <span style={{
        fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
        color: 'var(--fg-muted)', lineHeight: 1.5,
      }}>{children}</span>
      {hint && (
        <span className="faint" style={{ fontSize: 11, marginLeft: 8 }}>{hint}</span>
      )}
    </div>
  );
}

function TextInput({ value, onChange, placeholder, type = 'text', mono, width, disabled }) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={e => onChange(type === 'number' ? (parseInt(e.target.value) || 0) : e.target.value)}
      placeholder={placeholder}
      disabled={disabled}
      className={mono ? 'mono' : undefined}
      style={{
        width: width ?? '100%', height: 28, padding: '0 8px',
        background: disabled ? 'var(--surface-0)' : 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4, fontSize: 13, color: 'var(--fg)',
        opacity: disabled ? 0.6 : 1,
      }}
    />
  );
}

function SelectInput({ value, onChange, options, width }) {
  return (
    <div style={{ position: 'relative', display: 'inline-flex', width: width ?? 'auto' }}>
      <select
        value={value ?? ''}
        onChange={e => onChange(e.target.value)}
        style={{
          appearance: 'none', WebkitAppearance: 'none',
          height: 28, padding: '0 24px 0 8px', minWidth: 120,
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 4, color: 'var(--fg)',
          fontSize: 13, cursor: 'pointer',
          colorScheme: 'dark',
        }}
      >
        {options.map(o => <option key={o.v} value={o.v}>{o.l}</option>)}
      </select>
      <Icon.ChevronDown width={10} height={10} style={{
        position: 'absolute', right: 6, top: '50%', transform: 'translateY(-50%)',
        pointerEvents: 'none', opacity: 0.5,
      }}/>
    </div>
  );
}

// Toggle — square, monochrome. Active = solid fg, inactive = ghost.
function Toggle({ on, onChange, disabled }) {
  return (
    <button
      type="button"
      onClick={() => !disabled && onChange(!on)}
      disabled={disabled}
      style={{
        height: 22, padding: '0 8px',
        background: on ? 'var(--fg)' : 'transparent',
        border: '1px solid ' + (on ? 'var(--fg)' : 'var(--hairline-strong)'),
        borderRadius: 3,
        color: on ? 'var(--bg)' : 'var(--fg-muted)',
        fontSize: 11, fontWeight: 600,
        textTransform: 'uppercase', letterSpacing: '0.06em',
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
      }}
    >
      {on ? 'On' : 'Off'}
    </button>
  );
}

// ConfigRow — label-on-left, control-on-right. Mirrors users.tsx density.
function ConfigRow({ label, hint, children, last, sub }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'flex-start',
      padding: '9px 14px',
      borderBottom: last ? 'none' : '1px solid var(--hairline)',
      gap: 14, minHeight: 38,
    }}>
      <div style={{ width: 200, flexShrink: 0, paddingTop: 5 }}>
        <div style={{
          fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.06em',
          color: 'var(--fg-muted)', lineHeight: 1.5,
        }}>{label}</div>
        {hint && (
          <div className="faint" style={{ fontSize: 11, marginTop: 2, lineHeight: 1.5 }}>{hint}</div>
        )}
      </div>
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 4, paddingTop: 1 }}>
        {children}
        {sub && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{sub}</div>}
      </div>
    </div>
  );
}

function SectionCard({ id, title, sub, right, children }) {
  return (
    <section id={id} style={{ marginBottom: 24 }}>
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '10px 14px',
        background: 'var(--surface-0)',
        border: '1px solid var(--hairline)',
        borderBottom: 'none',
        borderRadius: '5px 5px 0 0',
      }}>
        <div>
          <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--fg)', lineHeight: 1.4 }}>{title}</div>
          {sub && <div className="faint" style={{ fontSize: 11, marginTop: 1, lineHeight: 1.5 }}>{sub}</div>}
        </div>
        {right}
      </div>
      <div style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline)',
        borderRadius: '0 0 5px 5px',
      }}>
        {children}
      </div>
    </section>
  );
}

// Status dot — circular, semantic color. Only place color is allowed.
function Dot({ tone = 'fg-dim' }) {
  const map = {
    success: 'var(--success)',
    warn:    'var(--warn)',
    danger:  'var(--danger)',
    'fg-dim': 'var(--fg-dim)',
  };
  return (
    <span style={{
      width: 6, height: 6, borderRadius: '50%',
      background: map[tone] || map['fg-dim'],
      flexShrink: 0, display: 'inline-block',
    }}/>
  );
}

// ── Main component ─────────────────────────────────────────────────────────

export function IdentityHub() {
  const { data, loading, refresh } = useAPI('/admin/config');
  const { data: jwks } = useAPI(null); // load below via raw fetch — JWKS isn't behind /admin
  const [form, setForm] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const [drawer, setDrawer] = React.useState(null); // { kind: 'provider'|'key', payload }
  const [keys, setKeys] = React.useState([]);
  const [keysLoading, setKeysLoading] = React.useState(true);
  const [keysTick, setKeysTick] = React.useState(0);
  const toast = useToast();

  // Initialise form once /admin/config resolves. Deep-clone so edits stay
  // local until the operator hits Save.
  React.useEffect(() => {
    if (!data) return;
    setForm({
      auth: {
        session_lifetime:    data.auth?.session_lifetime    ?? '30d',
        password_min_length: data.auth?.password_min_length ?? 8,
      },
      passkey: {
        rp_name:           data.passkey?.rp_name           ?? '',
        rp_id:             data.passkey?.rp_id             ?? '',
        user_verification: data.passkey?.user_verification ?? 'preferred',
      },
      magic_link: {
        ttl: data.magic_link?.ttl ?? '10m',
      },
      password_reset: {
        ttl: data.password_reset?.ttl ?? '30m',
      },
      jwt: {
        // jwt.enabled is informational — JWT issuance is always available
        // alongside cookie sessions. Operators tune lifetime, not "mode".
        lifetime:   data.jwt?.lifetime  ?? '15m',
        issuer:     data.jwt?.issuer    ?? '',
        audience:   data.jwt?.audience  ?? 'shark',
      },
      social: {
        redirect_url: data.social?.redirect_url ?? '',
        google: {
          client_id:     data.social?.google?.client_id     ?? '',
          client_secret: data.social?.google?.client_secret ?? '',
        },
        github: {
          client_id:     data.social?.github?.client_id     ?? '',
          client_secret: data.social?.github?.client_secret ?? '',
        },
        apple: {
          client_id:     data.social?.apple?.client_id     ?? '',
          client_secret: data.social?.apple?.client_secret ?? '',
        },
        discord: {
          client_id:     data.social?.discord?.client_id     ?? '',
          client_secret: data.social?.discord?.client_secret ?? '',
        },
      },
      mfa: {
        enforcement:    data.mfa?.enforcement    ?? 'optional', // off | optional | required
        issuer:         data.mfa?.issuer         ?? 'SharkAuth',
        recovery_codes: data.mfa?.recovery_codes ?? 10,
      },
      oauth_server: {
        enabled:                data.oauth_server?.enabled                 ?? true,
        issuer:                 data.oauth_server?.issuer                  ?? '',
        access_token_lifetime:  data.oauth_server?.access_token_lifetime   ?? '15m',
        refresh_token_lifetime: data.oauth_server?.refresh_token_lifetime  ?? '30d',
        require_dpop:           data.oauth_server?.require_dpop            ?? false,
      },
    });
  }, [data]);

  // Signing keys — pulled straight from the public JWKS endpoint, same
  // pattern as signing_keys.tsx. The /admin/config summary surfaces a
  // count, but operators want to see KIDs and rotation state here.
  React.useEffect(() => {
    setKeysLoading(true);
    fetch('/.well-known/jwks.json')
      .then(r => r.ok ? r.json() : { keys: [] })
      .then(j => { setKeys(j.keys || []); setKeysLoading(false); })
      .catch(() => { setKeys([]); setKeysLoading(false); });
  }, [keysTick]);

  if (loading || !form) {
    return <div className="faint" style={{ padding: 24, fontSize: 13 }}>Loading identity…</div>;
  }

  const dirty = (() => {
    try { return JSON.stringify(originalShape(data)) !== JSON.stringify(form); }
    catch { return true; }
  })();

  const passwordProviderOn = true; // password is always available
  const magicLinkOn = !!form.magic_link?.ttl;
  const passkeysOn = !!(form.passkey?.rp_id || form.passkey?.rp_name);
  const googleOn   = !!form.social.google.client_id;
  const githubOn   = !!form.social.github.client_id;
  const appleOn    = !!form.social.apple.client_id;
  const discordOn  = !!form.social.discord.client_id;

  const save = async () => {
    setBusy(true);
    try {
      // Map our form shape onto the PATCH /admin/config contract. The
      // backend ignores unknown keys, so it's safe to send everything.
      const payload = {
        auth: {
          session_lifetime:    form.auth.session_lifetime,
          password_min_length: form.auth.password_min_length,
        },
        passkeys: {
          rp_name:           form.passkey.rp_name,
          rp_id:             form.passkey.rp_id,
          user_verification: form.passkey.user_verification,
        },
        magic_link: {
          ttl: form.magic_link.ttl,
        },
        password_reset: {
          ttl: form.password_reset.ttl,
        },
        jwt: {
          issuer:   form.jwt.issuer,
          audience: form.jwt.audience,
        },
        social: {
          redirect_url: form.social.redirect_url,
          google:  toProviderPayload(form.social.google,  data.social?.google),
          github:  toProviderPayload(form.social.github,  data.social?.github),
          apple:   toProviderPayload(form.social.apple,   data.social?.apple),
          discord: toProviderPayload(form.social.discord, data.social?.discord),
        },
        mfa: {
          enforcement:    form.mfa.enforcement,
          issuer:         form.mfa.issuer,
          recovery_codes: form.mfa.recovery_codes,
        },
        oauth_server: {
          enabled:                form.oauth_server.enabled,
          issuer:                 form.oauth_server.issuer,
          access_token_lifetime:  form.oauth_server.access_token_lifetime,
          refresh_token_lifetime: form.oauth_server.refresh_token_lifetime,
          require_dpop:           form.oauth_server.require_dpop,
        },
      };
      await API.patch('/admin/config', payload);
      toast.success('Identity configuration saved');
      refresh();
    } catch (e) {
      toast.error('Save failed: ' + (e?.message || 'unknown'));
    } finally {
      setBusy(false);
    }
  };

  const reset = () => {
    if (!dirty) return;
    if (!window.confirm('Discard unsaved changes?')) return;
    refresh();
  };

  const rotateKey = async () => {
    if (!window.confirm('Rotate JWT signing key? Clients caching JWKS may see transient failures within cache TTL.')) return;
    try {
      const res = await API.post('/admin/auth/rotate-signing-key');
      toast.success('Key rotated · KID ' + (res?.kid || '—'));
      setKeysTick(t => t + 1);
    } catch (e) {
      toast.error('Rotation failed: ' + (e?.message || 'unknown'));
    }
  };

  // Patch a nested path of `form` immutably.
  const upd = (path, val) => {
    setForm(prev => {
      const next = { ...prev };
      const parts = path.split('.');
      let cur = next;
      for (let i = 0; i < parts.length - 1; i++) {
        cur[parts[i]] = { ...cur[parts[i]] };
        cur = cur[parts[i]];
      }
      cur[parts[parts.length - 1]] = val;
      return next;
    });
  };

  const openProvider = (id, label) => setDrawer({ kind: 'provider', id, label });

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>

        {/* Toolbar — mirrors users.tsx 10/16 padding + hairline border */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 500, fontFamily: 'var(--font-display)', letterSpacing: '-0.01em', lineHeight: 1.2 }}>
              Identity
            </div>
            <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
              Authentication methods, tokens, MFA, OAuth server
            </div>
          </div>
          <div style={{ flex: 1 }}/>
          <span className="faint mono" style={{ fontSize: 11 }}>PATCH /admin/config</span>
          {dirty && (
            <button className="btn ghost sm" onClick={reset} disabled={busy}>Discard</button>
          )}
          <button className="btn primary sm" onClick={save} disabled={!dirty || busy}>
            {busy ? 'Saving…' : 'Save changes'}
          </button>
        </div>

        <div style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>

          {/* Left rail — section anchors. 11px uppercase, hairline divider. */}
          <nav style={{
            width: 200, flexShrink: 0,
            borderRight: '1px solid var(--hairline)',
            padding: '14px 8px',
            background: 'var(--surface-0)',
            overflow: 'auto',
          }}>
            {SECTIONS.map(s => (
              <a key={s.id} href={'#' + s.id}
                 onClick={(e) => {
                   e.preventDefault();
                   document.getElementById(s.id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
                 }}
                 style={{
                   display: 'block',
                   padding: '7px 10px',
                   fontSize: 12,
                   color: 'var(--fg-muted)',
                   borderRadius: 3,
                   textDecoration: 'none',
                 }}>
                {s.label}
              </a>
            ))}
            <div style={{ height: 1, background: 'var(--hairline)', margin: '8px 4px' }}/>
            <div style={{ padding: '4px 10px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>
              Always on
            </div>
            <div style={{ padding: '4px 10px', fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5 }}>
              <div className="row" style={{ gap: 6 }}><Dot tone="success"/>Cookie sessions</div>
              <div className="row" style={{ gap: 6, marginTop: 4 }}><Dot tone="success"/>JWT issuance</div>
            </div>
          </nav>

          {/* Body — scrollable column of section cards */}
          <div style={{ flex: 1, overflow: 'auto', padding: '20px 24px 100px', background: 'var(--bg)' }}>
            <div style={{ maxWidth: 820 }}>

              {/* ── Authentication methods ─────────────────────────────── */}
              <SectionCard
                id="methods"
                title="Authentication methods"
                sub="Toggle the channels operators may use to sign in. Settings inline."
              >
                {/* Password */}
                <ConfigRow label="Password" hint="Argon2id-hashed credentials">
                  <div className="row" style={{ gap: 12, alignItems: 'center', flexWrap: 'wrap' }}>
                    <Toggle on={passwordProviderOn} onChange={() => {}} disabled/>
                    <span className="faint" style={{ fontSize: 11 }}>built-in</span>
                  </div>
                  <div className="row" style={{ gap: 10, marginTop: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 11, color: 'var(--fg-muted)', minWidth: 110 }}>Min length</span>
                    <TextInput
                      type="number"
                      width={80}
                      value={form.auth.password_min_length}
                      onChange={v => upd('auth.password_min_length', v)}
                    />
                    <span style={{ fontSize: 11, color: 'var(--fg-muted)', minWidth: 110, marginLeft: 8 }}>Reset link TTL</span>
                    <TextInput
                      width={100}
                      mono
                      value={form.password_reset.ttl}
                      onChange={v => upd('password_reset.ttl', v)}
                      placeholder="30m"
                    />
                  </div>
                </ConfigRow>

                {/* Magic link */}
                <ConfigRow label="Magic link" hint="Email-delivered single-use sign-in">
                  <div className="row" style={{ gap: 12, alignItems: 'center', flexWrap: 'wrap' }}>
                    <Toggle on={magicLinkOn} onChange={(on) => upd('magic_link.ttl', on ? '10m' : '')}/>
                  </div>
                  <div className="row" style={{ gap: 10, marginTop: 8, flexWrap: 'wrap' }}>
                    <span style={{ fontSize: 11, color: 'var(--fg-muted)', minWidth: 110 }}>Token TTL</span>
                    <TextInput
                      width={100}
                      mono
                      value={form.magic_link.ttl}
                      onChange={v => upd('magic_link.ttl', v)}
                      placeholder="10m"
                      disabled={!magicLinkOn}
                    />
                  </div>
                </ConfigRow>

                {/* Passkeys */}
                <ConfigRow label="Passkeys" hint="WebAuthn / FIDO2 credentials">
                  <div className="row" style={{ gap: 12, alignItems: 'center', flexWrap: 'wrap' }}>
                    <Toggle
                      on={passkeysOn}
                      onChange={(on) => {
                        if (!on) {
                          upd('passkey.rp_id', '');
                          upd('passkey.rp_name', '');
                        } else if (!form.passkey.rp_name) {
                          upd('passkey.rp_name', 'SharkAuth');
                        }
                      }}
                    />
                  </div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 8, marginTop: 8 }}>
                    <div>
                      <FieldLabel>RP name</FieldLabel>
                      <TextInput
                        value={form.passkey.rp_name}
                        onChange={v => upd('passkey.rp_name', v)}
                        placeholder="SharkAuth"
                        disabled={!passkeysOn}
                      />
                    </div>
                    <div>
                      <FieldLabel>RP ID</FieldLabel>
                      <TextInput
                        mono
                        value={form.passkey.rp_id}
                        onChange={v => upd('passkey.rp_id', v)}
                        placeholder="example.com"
                        disabled={!passkeysOn}
                      />
                    </div>
                    <div>
                      <FieldLabel>User verification</FieldLabel>
                      <SelectInput
                        value={form.passkey.user_verification}
                        onChange={v => upd('passkey.user_verification', v)}
                        options={[
                          { v: 'discouraged', l: 'Discouraged' },
                          { v: 'preferred',   l: 'Preferred' },
                          { v: 'required',    l: 'Required' },
                        ]}
                      />
                    </div>
                  </div>
                </ConfigRow>

                {/* Social providers */}
                <ConfigRow label="Social" hint="OAuth identity providers" last>
                  <div className="row" style={{ gap: 10, marginBottom: 6 }}>
                    <span style={{ fontSize: 11, color: 'var(--fg-muted)', minWidth: 110 }}>Redirect URL</span>
                    <TextInput
                      mono
                      value={form.social.redirect_url}
                      onChange={v => upd('social.redirect_url', v)}
                      placeholder="https://app.example.com/auth/callback"
                    />
                  </div>
                  <div style={{ display: 'flex', flexDirection: 'column', border: '1px solid var(--hairline)', borderRadius: 4, marginTop: 6 }}>
                    <ProviderRow
                      label="Google" id="google" on={googleOn}
                      clientId={form.social.google.client_id}
                      onOpen={() => openProvider('google', 'Google')}
                    />
                    <ProviderRow
                      label="GitHub" id="github" on={githubOn}
                      clientId={form.social.github.client_id}
                      onOpen={() => openProvider('github', 'GitHub')}
                    />
                    <ProviderRow
                      label="Apple" id="apple" on={appleOn}
                      clientId={form.social.apple.client_id}
                      onOpen={() => openProvider('apple', 'Apple')}
                    />
                    <ProviderRow
                      label="Discord" id="discord" on={discordOn}
                      clientId={form.social.discord.client_id}
                      onOpen={() => openProvider('discord', 'Discord')}
                      last
                    />
                  </div>
                </ConfigRow>
              </SectionCard>

              {/* ── Sessions & tokens ──────────────────────────────────── */}
              <SectionCard
                id="tokens"
                title="Sessions & tokens"
                sub="Cookie sessions and JWT issuance run side-by-side. Tune lifetimes and signing keys."
                right={
                  <span className="row" style={{ gap: 6, fontSize: 11, color: 'var(--fg-muted)' }}>
                    <Dot tone="success"/>cookie · <Dot tone="success"/>jwt
                  </span>
                }
              >
                <ConfigRow label="Cookie session lifetime" hint="Idle plus absolute window">
                  <TextInput
                    mono width={140}
                    value={form.auth.session_lifetime}
                    onChange={v => upd('auth.session_lifetime', v)}
                    placeholder="30d"
                  />
                </ConfigRow>

                <ConfigRow label="JWT access lifetime" hint="Reissued on session refresh">
                  <TextInput
                    mono width={140}
                    value={form.jwt.lifetime}
                    onChange={v => upd('jwt.lifetime', v)}
                    placeholder="15m"
                  />
                </ConfigRow>

                <ConfigRow label="JWT issuer / audience">
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                    <div>
                      <FieldLabel>Issuer</FieldLabel>
                      <TextInput
                        mono
                        value={form.jwt.issuer}
                        onChange={v => upd('jwt.issuer', v)}
                        placeholder={data.base_url || 'https://auth.example.com'}
                      />
                    </div>
                    <div>
                      <FieldLabel>Audience</FieldLabel>
                      <TextInput
                        mono
                        value={form.jwt.audience}
                        onChange={v => upd('jwt.audience', v)}
                        placeholder="shark"
                      />
                    </div>
                  </div>
                </ConfigRow>

                <ConfigRow label="Signing keys" hint="Public keys live at /.well-known/jwks.json" last>
                  <div className="row" style={{ gap: 8, marginBottom: 6 }}>
                    <span className="mono" style={{ fontSize: 11, color: 'var(--fg-muted)' }}>
                      {keysLoading ? 'loading…' : `${keys.length} key${keys.length === 1 ? '' : 's'}`}
                    </span>
                    <span style={{ flex: 1 }}/>
                    <button className="btn ghost sm" onClick={() => setKeysTick(t => t + 1)}>
                      <Icon.Refresh width={11} height={11}/>Refresh
                    </button>
                    <button className="btn sm" onClick={rotateKey}>
                      <Icon.Signing width={11} height={11}/>Rotate
                    </button>
                  </div>
                  <div style={{ border: '1px solid var(--hairline)', borderRadius: 4, overflow: 'hidden' }}>
                    {keysLoading && (
                      <div className="faint" style={{ padding: 12, fontSize: 12 }}>Loading JWKS…</div>
                    )}
                    {!keysLoading && keys.length === 0 && (
                      <div className="faint" style={{ padding: 12, fontSize: 12 }}>No active signing keys</div>
                    )}
                    {!keysLoading && keys.map((k, i) => (
                      <div key={k.kid || i}
                           className="row"
                           style={{
                             padding: '8px 10px', gap: 10,
                             borderBottom: i === keys.length - 1 ? 'none' : '1px solid var(--hairline)',
                             background: i === 0 ? 'var(--surface-2)' : 'transparent',
                           }}>
                        <Dot tone={i === 0 ? 'success' : 'fg-dim'}/>
                        <span className="mono" style={{ fontSize: 11, color: 'var(--fg)', minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>{k.kid || '—'}</span>
                        <span className="chip" style={{ height: 17, fontSize: 10 }}>{k.alg || k.kty || '—'}</span>
                        {i === 0 && <span className="chip" style={{ height: 17, fontSize: 10 }}>active</span>}
                        <span style={{ flex: 1 }}/>
                        <button
                          className="btn ghost sm"
                          onClick={() => setDrawer({ kind: 'key', payload: k })}
                        >
                          Inspect
                        </button>
                      </div>
                    ))}
                  </div>
                </ConfigRow>
              </SectionCard>

              {/* ── MFA ────────────────────────────────────────────────── */}
              <SectionCard
                id="mfa"
                title="Multi-factor authentication"
                sub="Second-factor enrolment for human accounts."
              >
                <ConfigRow label="Enforcement">
                  <SelectInput
                    width={200}
                    value={form.mfa.enforcement}
                    onChange={v => upd('mfa.enforcement', v)}
                    options={[
                      { v: 'off',      l: 'Off — never prompt' },
                      { v: 'optional', l: 'Optional — opt-in' },
                      { v: 'required', l: 'Required — all users' },
                    ]}
                  />
                </ConfigRow>
                <ConfigRow label="TOTP issuer" hint="Shown in authenticator apps">
                  <TextInput
                    value={form.mfa.issuer}
                    onChange={v => upd('mfa.issuer', v)}
                    placeholder="SharkAuth"
                  />
                </ConfigRow>
                <ConfigRow label="Recovery codes" hint="Printed once at enrolment" last>
                  <TextInput
                    type="number"
                    width={80}
                    value={form.mfa.recovery_codes}
                    onChange={v => upd('mfa.recovery_codes', v)}
                  />
                </ConfigRow>
              </SectionCard>

              {/* ── OAuth server ──────────────────────────────────────── */}
              <SectionCard
                id="oauth"
                title="OAuth server"
                sub="Issue tokens to third-party clients. Disabled by default outside of dev."
                right={
                  <Toggle
                    on={!!form.oauth_server.enabled}
                    onChange={v => upd('oauth_server.enabled', v)}
                  />
                }
              >
                <ConfigRow label="Issuer URL" hint="Embedded in iss claim">
                  <TextInput
                    mono
                    value={form.oauth_server.issuer}
                    onChange={v => upd('oauth_server.issuer', v)}
                    placeholder={data.base_url || 'https://auth.example.com'}
                    disabled={!form.oauth_server.enabled}
                  />
                </ConfigRow>
                <ConfigRow label="Access token lifetime">
                  <TextInput
                    mono width={140}
                    value={form.oauth_server.access_token_lifetime}
                    onChange={v => upd('oauth_server.access_token_lifetime', v)}
                    placeholder="15m"
                    disabled={!form.oauth_server.enabled}
                  />
                </ConfigRow>
                <ConfigRow label="Refresh token lifetime">
                  <TextInput
                    mono width={140}
                    value={form.oauth_server.refresh_token_lifetime}
                    onChange={v => upd('oauth_server.refresh_token_lifetime', v)}
                    placeholder="30d"
                    disabled={!form.oauth_server.enabled}
                  />
                </ConfigRow>
                <ConfigRow label="DPoP requirement" hint="Bind tokens to client key" last>
                  <Toggle
                    on={!!form.oauth_server.require_dpop}
                    onChange={v => upd('oauth_server.require_dpop', v)}
                    disabled={!form.oauth_server.enabled}
                  />
                </ConfigRow>
              </SectionCard>

            </div>
          </div>
        </div>
      </div>

      {/* Right-side drawer — provider edit OR key inspect */}
      {drawer?.kind === 'provider' && (
        <ProviderDrawer
          providerId={drawer.id}
          label={drawer.label}
          form={form}
          onChange={upd}
          onClose={() => setDrawer(null)}
        />
      )}
      {drawer?.kind === 'key' && (
        <KeyDrawer
          jwk={drawer.payload}
          onClose={() => setDrawer(null)}
        />
      )}
    </div>
  );
}

// ── Helpers ────────────────────────────────────────────────────────────────

// Map a provider sub-form back onto the PATCH wire shape. Don't ship the
// "********" placeholder back — the backend treats it as "no change".
function toProviderPayload(local, source) {
  const out = { client_id: local.client_id || '' };
  if (local.client_secret && local.client_secret !== (source?.client_secret || '')) {
    out.client_secret = local.client_secret;
  }
  return out;
}

// Build the same form-shape from the server response so dirty-checks are
// stable across save/reload cycles.
function originalShape(d) {
  if (!d) return null;
  return {
    auth: {
      session_lifetime:    d.auth?.session_lifetime    ?? '30d',
      password_min_length: d.auth?.password_min_length ?? 8,
    },
    passkey: {
      rp_name:           d.passkey?.rp_name           ?? '',
      rp_id:             d.passkey?.rp_id             ?? '',
      user_verification: d.passkey?.user_verification ?? 'preferred',
    },
    magic_link: { ttl: d.magic_link?.ttl ?? '10m' },
    password_reset: { ttl: d.password_reset?.ttl ?? '30m' },
    jwt: {
      lifetime: d.jwt?.lifetime ?? '15m',
      issuer:   d.jwt?.issuer   ?? '',
      audience: d.jwt?.audience ?? 'shark',
    },
    social: {
      redirect_url: d.social?.redirect_url ?? '',
      google:  { client_id: d.social?.google?.client_id  ?? '', client_secret: d.social?.google?.client_secret  ?? '' },
      github:  { client_id: d.social?.github?.client_id  ?? '', client_secret: d.social?.github?.client_secret  ?? '' },
      apple:   { client_id: d.social?.apple?.client_id   ?? '', client_secret: d.social?.apple?.client_secret   ?? '' },
      discord: { client_id: d.social?.discord?.client_id ?? '', client_secret: d.social?.discord?.client_secret ?? '' },
    },
    mfa: {
      enforcement:    d.mfa?.enforcement    ?? 'optional',
      issuer:         d.mfa?.issuer         ?? 'SharkAuth',
      recovery_codes: d.mfa?.recovery_codes ?? 10,
    },
    oauth_server: {
      enabled:                d.oauth_server?.enabled                 ?? true,
      issuer:                 d.oauth_server?.issuer                  ?? '',
      access_token_lifetime:  d.oauth_server?.access_token_lifetime   ?? '15m',
      refresh_token_lifetime: d.oauth_server?.refresh_token_lifetime  ?? '30d',
      require_dpop:           d.oauth_server?.require_dpop            ?? false,
    },
  };
}

// Provider row inside the social card — lookup-style row matching users.tsx
// table cadence. No color, just dot + label + click-through.
function ProviderRow({ label, id, on, clientId, onOpen, last }) {
  return (
    <div className="row" style={{
      padding: '8px 10px', gap: 10,
      borderBottom: last ? 'none' : '1px solid var(--hairline)',
      cursor: 'pointer',
    }} onClick={onOpen}>
      <Dot tone={on ? 'success' : 'fg-dim'}/>
      <span style={{ fontSize: 13, fontWeight: 500, minWidth: 80 }}>{label}</span>
      {clientId ? (
        <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5, flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {clientId}
        </span>
      ) : (
        <span className="faint" style={{ fontSize: 11, flex: 1 }}>not configured</span>
      )}
      <span className="chip" style={{ height: 17, fontSize: 10 }}>{on ? 'on' : 'off'}</span>
      <button className="btn ghost sm" onClick={(e) => { e.stopPropagation(); onOpen(); }}>
        Configure
      </button>
    </div>
  );
}

// ── Drawers — right-side, 480px, slideIn 140ms (matches users.tsx) ─────────

function ProviderDrawer({ providerId, label, form, onChange, onClose }) {
  const cfg = form.social[providerId] || { client_id: '', client_secret: '' };
  const callbackURL = (form.social.redirect_url || '').replace(/\/?$/, '') + '/' + providerId;

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  return (
    <>
      <div onClick={onClose}
           style={{ position: 'fixed', inset: 0, zIndex: 40, background: 'rgba(0,0,0,0.35)' }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0,
        width: 480, maxWidth: '100vw',
        zIndex: 41,
        borderLeft: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
            <button className="btn ghost sm" onClick={onClose}><Icon.X width={12} height={12}/>Close</button>
            <span className="faint mono" style={{ fontSize: 11 }}>social.{providerId}</span>
          </div>
          <div style={{ fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em', fontFamily: 'var(--font-display)', lineHeight: 1.2 }}>
            {label}
          </div>
          <div className="faint" style={{ fontSize: 13, marginTop: 4, lineHeight: 1.5 }}>
            OAuth client credentials. Empty client ID disables this provider.
          </div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: 16, display: 'flex', flexDirection: 'column', gap: 14 }}>
          <div>
            <FieldLabel>Client ID</FieldLabel>
            <TextInput
              mono
              value={cfg.client_id}
              onChange={v => onChange('social.' + providerId + '.client_id', v)}
              placeholder="client-id-from-provider"
            />
          </div>

          <div>
            <FieldLabel hint="Leave the masked value to keep the existing secret">Client secret</FieldLabel>
            <TextInput
              type="password"
              mono
              value={cfg.client_secret}
              onChange={v => onChange('social.' + providerId + '.client_secret', v)}
              placeholder="••••••••"
            />
          </div>

          <div>
            <FieldLabel>Callback URL</FieldLabel>
            <div className="row" style={{
              gap: 8, padding: '6px 8px',
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4, height: 28,
            }}>
              <span className="mono" style={{ fontSize: 11, color: 'var(--fg)', flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>{callbackURL}</span>
              <CopyField value={callbackURL} truncate={0}/>
            </div>
            <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 4 }}>
              Register this exact URL in the {label} OAuth app. Edit the redirect base in the social section.
            </div>
          </div>
        </div>

        <div style={{
          padding: 12, borderTop: '1px solid var(--hairline)',
          display: 'flex', justifyContent: 'flex-end', gap: 8,
          background: 'var(--surface-0)',
        }}>
          <button className="btn sm" onClick={onClose}>Done</button>
        </div>
      </div>
    </>
  );
}

function KeyDrawer({ jwk, onClose }) {
  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const fields = [
    ['kid', jwk?.kid],
    ['kty', jwk?.kty],
    ['alg', jwk?.alg],
    ['use', jwk?.use],
    ['crv', jwk?.crv],
    ['n',   jwk?.n],
    ['x',   jwk?.x],
    ['y',   jwk?.y],
  ].filter(([, v]) => v != null && v !== '');

  return (
    <>
      <div onClick={onClose}
           style={{ position: 'fixed', inset: 0, zIndex: 40, background: 'rgba(0,0,0,0.35)' }}/>
      <div style={{
        position: 'fixed', top: 0, right: 0, bottom: 0,
        width: 480, maxWidth: '100vw',
        zIndex: 41,
        borderLeft: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
        display: 'flex', flexDirection: 'column',
        animation: 'slideIn 140ms ease-out',
      }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
            <button className="btn ghost sm" onClick={onClose}><Icon.X width={12} height={12}/>Close</button>
            <span className="faint mono" style={{ fontSize: 11 }}>JWKS entry</span>
          </div>
          <div style={{ fontSize: 20, fontWeight: 500, letterSpacing: '-0.02em', fontFamily: 'var(--font-display)', lineHeight: 1.2 }}>
            Signing key
          </div>
          <div className="row" style={{ gap: 6, marginTop: 6 }}>
            <span className="chip">{jwk?.alg || '—'}</span>
            <span className="chip">{jwk?.kty || '—'}</span>
          </div>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
          <div style={{
            border: '1px solid var(--hairline)',
            borderRadius: 4,
            background: 'var(--surface-1)',
          }}>
            {fields.map(([k, v], i) => (
              <div key={k} className="row" style={{
                padding: '7px 10px', gap: 10,
                borderBottom: i === fields.length - 1 ? 'none' : '1px solid var(--hairline)',
              }}>
                <span style={{
                  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                  color: 'var(--fg-muted)', minWidth: 36,
                }}>{k}</span>
                <span className="mono" style={{ fontSize: 11, color: 'var(--fg)', flex: 1, minWidth: 0, wordBreak: 'break-all' }}>{String(v)}</span>
                <CopyField value={String(v)} truncate={0}/>
              </div>
            ))}
          </div>
          <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 12 }}>
            Public key only. Private material is AES-GCM encrypted at rest and never leaves the server.
          </div>
        </div>
        <div style={{
          padding: 12, borderTop: '1px solid var(--hairline)',
          display: 'flex', justifyContent: 'flex-end', gap: 8,
          background: 'var(--surface-0)',
        }}>
          <button className="btn sm" onClick={onClose}>Close</button>
        </div>
      </div>
    </>
  );
}
