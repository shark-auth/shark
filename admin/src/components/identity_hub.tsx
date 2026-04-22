// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { useToast } from './toast'

const SECTION_LABEL = {
  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
  color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 12
};

const HAIRLINE = '1px solid var(--hairline)';

function ConfigInput({ label, value, onChange, placeholder, type = "text", sub }) {
  return (
    <div className="col" style={{ gap: 4, marginBottom: 16 }}>
      <div style={{ fontSize: 12, fontWeight: 500 }}>{label}</div>
      <input
        type={type}
        value={value || ''}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        style={{
          height: 34, padding: '0 10px', background: 'var(--surface-2)',
          border: '1px solid var(--hairline-strong)', borderRadius: 5,
          color: 'var(--fg)', fontSize: 13, outline: 'none'
        }}
      />
      {sub && <div className="faint" style={{ fontSize: 11 }}>{sub}</div>}
    </div>
  );
}

export function IdentityHub() {
  const { data, loading, refresh } = useAPI('/admin/config');
  const [form, setForm] = React.useState(null);
  const [busy, setBusy] = React.useState(false);
  const toast = useToast();

  React.useEffect(() => {
    if (data) setForm(JSON.parse(JSON.stringify(data)));
  }, [data]);

  const save = async () => {
    setBusy(true);
    try {
      await API.patch('/admin/config', form);
      toast.success('Identity policies updated');
      refresh();
    } catch (e) {
      toast.error('Save failed: ' + e.message);
    } finally {
      setBusy(false);
    }
  };

  if (loading || !form) return <div className="faint" style={{ padding: 20 }}>Loading identity hub…</div>;

  const changed = JSON.stringify(data) !== JSON.stringify(form);

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between' }}>
          <h1 style={{ fontSize: 18, fontWeight: 600, fontFamily: 'var(--font-display)', margin: 0 }}>Identity Hub</h1>
          {changed && (
            <button className="btn primary sm" onClick={save} disabled={busy}>
              {busy ? 'Saving…' : 'Save Changes'}
            </button>
          )}
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: '20px auto', width: '100%', padding: '0 20px 100px' }}>
        
        {/* JWT & Session Mode */}
        <div style={SECTION_LABEL}>Token & Session Strategy</div>
        <div className="card" style={{ padding: 20, marginBottom: 32 }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 20 }}>
            <div className="col" style={{ gap: 4 }}>
              <div style={{ fontSize: 13, fontWeight: 600 }}>Enable JWT Issuance</div>
              <div className="faint" style={{ fontSize: 11.5 }}>Mint OIDC-compliant tokens alongside sessions.</div>
            </div>
            <button 
              className={"btn sm " + (form.jwt.enabled ? 'primary' : 'ghost')}
              onClick={() => setForm({ ...form, jwt: { ...form.jwt, enabled: !form.jwt.enabled } })}
            >
              {form.jwt.enabled ? 'Enabled' : 'Disabled'}
            </button>
          </div>

          <div className="col" style={{ gap: 10, opacity: form.jwt.enabled ? 1 : 0.5, pointerEvents: form.jwt.enabled ? 'auto' : 'none' }}>
             <div style={{ fontSize: 12, fontWeight: 500 }}>Issuance Mode</div>
             <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <button 
                  className={"card " + (form.jwt.mode === 'session' ? 'active-border' : '')} 
                  style={{ padding: 12, textAlign: 'left', background: 'var(--surface-1)', border: form.jwt.mode === 'session' ? '1px solid var(--accent)' : '1px solid var(--hairline)' }}
                  onClick={() => setForm({ ...form, jwt: { ...form.jwt, mode: 'session' } })}
                >
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4 }}>Session Mode</div>
                  <div className="faint" style={{ fontSize: 10.5, lineHeight: 1.4 }}>One long-lived JWT matching the session. Best for simple SPAs.</div>
                </button>
                <button 
                  className={"card " + (form.jwt.mode === 'access_refresh' ? 'active-border' : '')} 
                  style={{ padding: 12, textAlign: 'left', background: 'var(--surface-1)', border: form.jwt.mode === 'access_refresh' ? '1px solid var(--accent)' : '1px solid var(--hairline)' }}
                  onClick={() => setForm({ ...form, jwt: { ...form.jwt, mode: 'access_refresh' } })}
                >
                  <div style={{ fontSize: 12, fontWeight: 600, marginBottom: 4 }}>Access / Refresh</div>
                  <div className="faint" style={{ fontSize: 10.5, lineHeight: 1.4 }}>Short-lived access tokens + rotation. Best for mobile/security-first apps.</div>
                </button>
             </div>
             
             {form.jwt.mode === 'access_refresh' && (
               <div className="col" style={{ gap: 12, marginTop: 12, padding: 16, background: 'var(--surface-2)', borderRadius: 6 }}>
                  <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--fg-dim)', textTransform: 'uppercase' }}>Advanced Settings</div>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                     <ConfigInput 
                       label="Issuer" 
                       value={form.jwt.issuer} 
                       onChange={v => setForm({ ...form, jwt: { ...form.jwt, issuer: v } })}
                       placeholder={data.base_url}
                     />
                     <ConfigInput 
                       label="Audience" 
                       value={form.jwt.audience} 
                       onChange={v => setForm({ ...form, jwt: { ...form.jwt, audience: v } })}
                       placeholder="shark"
                     />
                     <ConfigInput 
                       label="Clock Skew" 
                       value={form.jwt.clock_skew} 
                       onChange={v => setForm({ ...form, jwt: { ...form.jwt, clock_skew: v } })}
                       placeholder="30s"
                       sub="Grace period for expired tokens"
                     />
                  </div>
               </div>
             )}
          </div>
        </div>

        {/* Auth Methods */}
        <div style={SECTION_LABEL}>Authentication Methods</div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 32 }}>
          <ProviderCard 
            id="google" 
            name="Google" 
            icon={<Icon.Globe width={16}/>} 
            config={form.social.google}
            onChange={v => setForm({ ...form, social: { ...form.social, google: v } })}
          />
          <ProviderCard 
            id="github" 
            name="GitHub" 
            icon={<Icon.Users width={16}/>} 
            config={form.social.github}
            onChange={v => setForm({ ...form, social: { ...form.social, github: v } })}
          />
        </div>

        {/* Password Policy */}
        <div style={SECTION_LABEL}>Password Security</div>
        <div className="card" style={{ padding: 20, marginBottom: 32 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <ConfigInput 
              label="Minimum Length" 
              value={form.auth.password_min_length} 
              onChange={v => setForm({ ...form, auth: { ...form.auth, password_min_length: parseInt(v) || 0 } })}
              type="number"
            />
            <ConfigInput 
              label="Session Lifetime" 
              value={form.auth.session_lifetime} 
              onChange={v => setForm({ ...form, auth: { ...form.auth, session_lifetime: v } })}
              sub="e.g. 24h, 7d, 30d"
            />
          </div>
          
          <div style={{ marginTop: 24, paddingTop: 20, borderTop: HAIRLINE }}>
            <div style={{ fontSize: 12, fontWeight: 500, marginBottom: 12 }}>Password Recovery</div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
               <ConfigInput 
                 label="Reset Link TTL" 
                 value={form.password_reset.ttl} 
                 onChange={v => setForm({ ...form, password_reset: { ...form.password_reset, ttl: v } })}
                 placeholder="30m"
                 sub="e.g. 15m, 1h"
               />
            </div>
          </div>
        </div>

        {/* Passkeys */}
        <div style={SECTION_LABEL}>Passkeys (WebAuthn)</div>
        <div className="card" style={{ padding: 20, marginBottom: 32 }}>
           <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <ConfigInput 
              label="RP Name" 
              value={form.passkey.rp_name} 
              onChange={v => setForm({ ...form, passkey: { ...form.passkey, rp_name: v } })}
              placeholder="e.g. SharkAuth"
            />
            <ConfigInput 
              label="RP ID" 
              value={form.passkey.rp_id} 
              onChange={v => setForm({ ...form, passkey: { ...form.passkey, rp_id: v } })}
              placeholder="e.g. localhost"
            />
          </div>
        </div>

      </div>
    </div>
  );
}

function ProviderCard({ name, icon, config, onChange }) {
  const [expanded, setExpanded] = React.useState(!!config.client_id);

  return (
    <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
      <div className="row" style={{ padding: '12px 16px', background: 'var(--surface-1)', borderBottom: HAIRLINE, justifyContent: 'space-between' }}>
        <div className="row" style={{ gap: 10 }}>
          {icon}
          <span style={{ fontSize: 13, fontWeight: 600 }}>{name}</span>
        </div>
        <button className="btn ghost sm" onClick={() => setExpanded(!expanded)}>
          {expanded ? 'Hide Config' : 'Configure'}
        </button>
      </div>
      {expanded && (
        <div style={{ padding: 16 }}>
          <ConfigInput 
            label="Client ID" 
            value={config.client_id} 
            onChange={v => onChange({ ...config, client_id: v })}
          />
          <ConfigInput 
            label="Client Secret" 
            value={config.client_secret} 
            onChange={v => onChange({ ...config, client_secret: v })}
            type="password"
            placeholder="********"
            sub="Leave as ******** to keep existing secret"
          />
        </div>
      )}
    </div>
  );
}
