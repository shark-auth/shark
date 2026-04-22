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

export function SystemSettings() {
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
      toast.success('System settings updated');
      refresh();
    } catch (e) {
      toast.error('Save failed: ' + e.message);
    } finally {
      setBusy(false);
    }
  };

  if (loading || !form) return <div className="faint" style={{ padding: 20 }}>Loading system settings…</div>;

  const changed = JSON.stringify(data) !== JSON.stringify(form);

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
        <div className="row" style={{ justifyContent: 'space-between' }}>
          <h1 style={{ fontSize: 18, fontWeight: 600, fontFamily: 'var(--font-display)', margin: 0 }}>System</h1>
          {changed && (
            <button className="btn primary sm" onClick={save} disabled={busy}>
              {busy ? 'Saving…' : 'Save Changes'}
            </button>
          )}
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: '20px auto', width: '100%', padding: '0 20px 100px' }}>
        
        {/* Outbound Email */}
        <div style={SECTION_LABEL}>Outbound Email</div>
        <div className="card" style={{ padding: 20, marginBottom: 32 }}>
          <div className="row" style={{ gap: 20, marginBottom: 20 }}>
             <label className="col" style={{ flex: 1, gap: 4 }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontWeight: 500 }}>Provider</span>
                <select 
                  value={form.email.provider} 
                  onChange={e => setForm({ ...form, email: { ...form.email, provider: e.target.value } })}
                  style={{
                    height: 34, padding: '0 8px', background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)',
                    borderRadius: 5, color: 'var(--fg)', fontSize: 13, outline: 'none'
                  }}
                >
                  <option value="dev">Dev Inbox (Local)</option>
                  <option value="shark">Shark Native (Internal)</option>
                  <option value="resend">Resend</option>
                  <option value="smtp">Custom SMTP</option>
                </select>
             </label>
             <div style={{ flex: 1 }}>
               <ConfigInput 
                 label="API Key / Password" 
                 value={form.email.api_key} 
                 onChange={v => setForm({ ...form, email: { ...form.email, api_key: v } })}
                 type="password"
                 placeholder="sk_live_..."
               />
             </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <ConfigInput 
              label="From Email" 
              value={form.email.from} 
              onChange={v => setForm({ ...form, email: { ...form.email, from: v } })}
              placeholder="auth@yourdomain.com"
            />
            <ConfigInput 
              label="From Name" 
              value={form.email.from_name} 
              onChange={v => setForm({ ...form, email: { ...form.email, from_name: v } })}
              placeholder="SharkAuth"
            />
          </div>
        </div>

        {/* Audit Retention */}
        <div style={SECTION_LABEL}>Audit & Data Retention</div>
        <div className="card" style={{ padding: 20, marginBottom: 32 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <ConfigInput 
              label="Retention Period" 
              value={form.audit.retention} 
              onChange={v => setForm({ ...form, audit: { ...form.audit, retention: v } })}
              sub="e.g. 30d, 90d, 0 (forever)"
            />
            <ConfigInput 
              label="Cleanup Interval" 
              value={form.audit.cleanup_interval} 
              onChange={v => setForm({ ...form, audit: { ...form.audit, cleanup_interval: v } })}
              sub="e.g. 1h, 24h"
            />
          </div>
        </div>

        {/* Infrastructure (Read Only) */}
        <div style={SECTION_LABEL}>Infrastructure (Read-Only)</div>
        <div className="card" style={{ padding: 20, background: 'var(--surface-0)', opacity: 0.8 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
            <ConfigInput label="Environment" value={form.environment} onChange={() => {}} placeholder="production" sub="Set via ENV" />
            <ConfigInput label="Base URL" value={form.base_url} onChange={() => {}} sub="Restart required to change" />
          </div>
        </div>

      </div>
    </div>
  );
}
