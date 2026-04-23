// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'

// Settings page — server info, email, session management, audit retention, danger zone
// Refactored to support interactive configuration updates (Phase Wave 2).

export function Settings() {
  const { data: health, loading: healthLoading } = useAPI('/admin/health');
  const { data: config, loading: configLoading, refresh: refreshConfig } = useAPI('/admin/config');

  const [toast, setToast] = React.useState(null);
  const [purgingSession, setPurgingSession] = React.useState(false);
  const [purgingAudit, setPurgingAudit] = React.useState(false);
  const [auditBefore, setAuditBefore] = React.useState('');

  const [isDirty, setIsDirty] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [formState, setFormState] = React.useState(null);

  // Initialize form state when config loads
  React.useEffect(() => {
    if (config) {
      setFormState({
        auth: {
          session_lifetime: config.auth?.session_lifetime || '30d',
          password_min_length: config.auth?.password_min_length || 8,
        },
        passkeys: {
          rp_name: config.passkeys?.rp_name || 'SharkAuth',
          rp_id: config.passkeys?.rp_id || '',
          user_verification: config.passkeys?.user_verification || 'preferred',
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
        }
      });
      setIsDirty(false);
    }
  }, [config]);

  const updateForm = (path, val) => {
    const parts = path.split('.');
    setFormState(prev => {
      const next = { ...prev };
      let curr = next;
      for (let i = 0; i < parts.length - 1; i++) {
        curr[parts[i]] = { ...curr[parts[i]] };
        curr = curr[parts[i]];
      }
      curr[parts[parts.length - 1]] = val;
      return next;
    });
    setIsDirty(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await API.patch('/admin/config', formState);
      showToast('Configuration updated and persisted');
      setIsDirty(false);
      refreshConfig();
    } catch (e) {
      showToast(e.message || 'Save failed', 'danger');
    } finally {
      setSaving(false);
    }
  };

  function showToast(msg, type = 'success') {
    setToast({ msg, type });
    setTimeout(() => setToast(null), 3500);
  }

  function formatUptime(s) {
    if (s == null) return '—';
    const d = Math.floor(s / 86400);
    const hh = Math.floor((s % 86400) / 3600).toString().padStart(2, '0');
    const mm = Math.floor((s % 3600) / 60).toString().padStart(2, '0');
    return `${d}d ${hh}:${mm}`;
  }

  function formatBytes(mb) {
    if (mb == null) return '—';
    if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
    return `${mb} MB`;
  }

  async function handlePurgeSessions() {
    setPurgingSession(true);
    try {
      const res = await API.post('/admin/sessions/purge-expired');
      showToast(`Purged ${res?.deleted ?? 0} expired session${res?.deleted !== 1 ? 's' : ''}`);
    } catch (e) {
      showToast(e?.message || 'Purge failed', 'danger');
    } finally {
      setPurgingSession(false);
    }
  }

  async function handlePurgeAudit() {
    if (!auditBefore) return;
    setPurgingAudit(true);
    try {
      const isoDate = new Date(auditBefore).toISOString();
      const res = await API.post('/admin/audit-logs/purge', { before: isoDate });
      showToast(`Purged ${res?.deleted ?? 0} audit log entr${res?.deleted !== 1 ? 'ies' : 'y'}`);
      setAuditBefore('');
    } catch (e) {
      showToast(e?.message || 'Purge failed', 'danger');
    } finally {
      setPurgingAudit(false);
    }
  }

  async function handleTestEmail() {
    const to = prompt('Send test email to:');
    if (!to) return;
    try {
      await API.post('/admin/test-email', { to });
      showToast('Test email sent to ' + to);
    } catch (e) {
      showToast(e?.message || 'Failed to send', 'danger');
    }
  }

  const baseUrl = config?.base_url || window.location.origin;
  const corsOrigins = config?.cors_origins || [];
  const isDevMode = config?.dev_mode === true || config?.environment === 'development';
  const smtpConfigured = !!(config?.smtp_configured || health?.smtp?.configured);

  function ConfigRow({ label, children, system }) {
    return (
      <div style={{
        display: 'flex', alignItems: 'center',
        padding: '9px 14px',
        borderBottom: '1px solid var(--hairline)',
        gap: 12,
        minHeight: 38,
      }}>
        <div style={{ width: 160, flexShrink: 0, display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 11.5, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{label}</span>
          {system && <span className="chip faint sm" style={{fontSize: 8, height: 14, padding: '0 4px'}}>SYSTEM</span>}
        </div>
        <div style={{ flex: 1, fontSize: 12.5 }}>{children}</div>
      </div>
    );
  }

  function SectionHeader({ title, sub }) {
    return (
      <div style={{ marginBottom: 8 }}>
        <div style={{ fontSize: 12.5, fontWeight: 600, color: 'var(--fg)' }}>{title}</div>
        {sub && <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 2 }}>{sub}</div>}
      </div>
    );
  }

  function Skeleton({ w = '60%', h = 13 }) {
    return <div style={{ width: w, height: h, background: 'var(--surface-3)', borderRadius: 3 }}/>;
  }

  if (configLoading && !formState) {
    return <div className="faint" style={{ padding: 24, fontSize: 12 }}>Loading configuration…</div>;
  }

  return (
    <div style={{ height: '100%', overflowY: 'auto', padding: 20, paddingBottom: 100 }}>
      <div style={{ maxWidth: 760, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 24 }}>

        {/* ── Server ── */}
        <section>
          <SectionHeader title="Server" sub="Runtime info and immutable system-level configuration"/>
          <div className="card">
            <ConfigRow label="Base URL" system>
              <CopyField value={baseUrl} truncate={48}/>
            </ConfigRow>
            <ConfigRow label="Version" system>
              <span className="mono" style={{ fontSize: 12 }}>{health?.version ?? '—'}</span>
            </ConfigRow>
            <ConfigRow label="Uptime" system>
              <span className="mono" style={{ fontSize: 12 }}>{formatUptime(health?.uptime_seconds)}</span>
            </ConfigRow>
            <ConfigRow label="DB Size" system>
              <div className="row" style={{ gap: 8 }}>
                <span className="mono" style={{ fontSize: 12 }}>{formatBytes(health?.db?.size_mb)}</span>
                {health?.db?.status && (
                  <span className={"chip" + (health.db.status === 'ok' ? ' success' : ' warn')} style={{ height: 16, fontSize: 10 }}>
                    <span className={"dot" + (health.db.status === 'ok' ? ' success' : ' warn')} style={{ width: 5, height: 5 }}/>
                    {health.db.status}
                  </span>
                )}
              </div>
            </ConfigRow>
            <ConfigRow label="Environment" system>
              <div className="row" style={{ gap: 8 }}>
                <span className="mono" style={{ fontSize: 12 }}>{config?.environment ?? '—'}</span>
                {isDevMode && (
                  <span className="chip warn" style={{ height: 16, fontSize: 10 }}>dev mode</span>
                )}
              </div>
            </ConfigRow>
          </div>
        </section>

        {/* ── Authentication ── */}
        <section>
          <SectionHeader title="Authentication" sub="Policies for sessions and password security"/>
          <div className="card">
            <ConfigRow label="Session Lifetime">
              <input
                value={formState?.auth?.session_lifetime}
                onChange={e => updateForm('auth.session_lifetime', e.target.value)}
                className="input-inline mono"
                placeholder="e.g. 30d, 12h"
              />
            </ConfigRow>
            <ConfigRow label="Min Password Length">
              <input
                type="number"
                value={formState?.auth?.password_min_length}
                onChange={e => updateForm('auth.password_min_length', parseInt(e.target.value))}
                className="input-inline mono"
                style={{ width: 60 }}
              />
            </ConfigRow>
          </div>
        </section>

        {/* ── Passkeys ── */}
        <section>
          <SectionHeader title="Passkeys" sub="WebAuthn rely-party and verification settings"/>
          <div className="card">
            <ConfigRow label="RP Name">
              <input
                value={formState?.passkeys?.rp_name}
                onChange={e => updateForm('passkeys.rp_name', e.target.value)}
                className="input-inline"
                placeholder="Company Name"
              />
            </ConfigRow>
            <ConfigRow label="RP ID">
              <input
                value={formState?.passkeys?.rp_id}
                onChange={e => updateForm('passkeys.rp_id', e.target.value)}
                className="input-inline mono"
                placeholder="domain.com"
              />
            </ConfigRow>
            <ConfigRow label="User Verification">
              <select
                value={formState?.passkeys?.user_verification}
                onChange={e => updateForm('passkeys.user_verification', e.target.value)}
                className="input-inline"
              >
                <option value="required">Required (biometrics only)</option>
                <option value="preferred">Preferred (biometrics if avail)</option>
                <option value="discouraged">Discouraged</option>
              </select>
            </ConfigRow>
          </div>
        </section>

        {/* ── Email ── */}
        <section>
          <SectionHeader title="Email Provider" sub="Update delivery credentials for outbound mail"/>
          <div className="card">
            <ConfigRow label="Provider">
              <select
                value={formState?.email?.provider}
                onChange={e => updateForm('email.provider', e.target.value)}
                className="input-inline"
              >
                <option value="shark">Shark Managed (Native)</option>
                <option value="resend">Resend (API Key)</option>
                <option value="smtp">Custom SMTP Server</option>
                <option value="dev">Dev Mode (Local Inbox)</option>
              </select>
            </ConfigRow>
            {(formState?.email?.provider === 'resend' || formState?.email?.provider === 'shark') && (
              <ConfigRow label="API Key">
                <input
                  type="password"
                  value={formState?.email?.api_key}
                  onChange={e => updateForm('email.api_key', e.target.value)}
                  className="input-inline mono"
                  placeholder="re_..."
                />
              </ConfigRow>
            )}
            <ConfigRow label="From Name">
              <input
                value={formState?.email?.from_name}
                onChange={e => updateForm('email.from_name', e.target.value)}
                className="input-inline"
                placeholder="SharkAuth Notifications"
              />
            </ConfigRow>
            <ConfigRow label="From Email">
              <input
                value={formState?.email?.from}
                onChange={e => updateForm('email.from', e.target.value)}
                className="input-inline mono"
                placeholder="no-reply@yourdomain.com"
              />
            </ConfigRow>
            <div style={{ padding: '10px 14px', display: 'flex', alignItems: 'center', gap: 12 }}>
              <button className="btn sm" onClick={handleTestEmail}>
                <Icon.Mail width={11} height={11}/>
                Send test email
              </button>
              <span className="faint" style={{ fontSize: 11 }}>Verify credentials before saving</span>
            </div>
          </div>
        </section>

        {/* ── Governance ── */}
        <section>
          <SectionHeader title="Governance" sub="Audit retention and maintenance intervals"/>
          <div className="card">
            <ConfigRow label="Audit Retention">
              <input
                value={formState?.audit?.retention}
                onChange={e => updateForm('audit.retention', e.target.value)}
                className="input-inline mono"
                placeholder="e.g. 90d, 0 (forever)"
              />
            </ConfigRow>
            <ConfigRow label="Cleanup Interval">
              <input
                value={formState?.audit?.cleanup_interval}
                onChange={e => updateForm('audit.cleanup_interval', e.target.value)}
                className="input-inline mono"
                placeholder="e.g. 1h, 24h"
              />
            </ConfigRow>
          </div>
        </section>

        {/* ── Maintenance ── */}
        <section>
          <SectionHeader title="Maintenance" sub="Manual cleanup operations"/>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
            <div className="card" style={{ padding: 14 }}>
              <div style={{ fontSize: 12.5, fontWeight: 600, marginBottom: 4 }}>Purge Sessions</div>
              <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 12, lineHeight: 1.5 }}>Remove all expired sessions from the database now.</div>
              <button className="btn sm" onClick={handlePurgeSessions} disabled={purgingSession}>
                <Icon.Refresh width={11} height={11} style={purgingSession ? { animation: 'spin 0.8s linear infinite' } : {}}/>
                Purge now
              </button>
            </div>
            <div className="card" style={{ padding: 14 }}>
              <div style={{ fontSize: 12.5, fontWeight: 600, marginBottom: 4 }}>Manual Audit Purge</div>
              <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 12, lineHeight: 1.5 }}>Delete entries older than the selected date.</div>
              <div className="row" style={{ gap: 6 }}>
                <input type="date" value={auditBefore} onChange={e => setAuditBefore(e.target.value)} className="mono" style={{ flex: 1, fontSize: 11, height: 26, background: 'var(--surface-2)', border: '1px solid var(--hairline)', borderRadius: 3, color: 'var(--fg)', colorScheme: 'dark', padding: '0 4px' }}/>
                <button className="btn sm danger" onClick={handlePurgeAudit} disabled={!auditBefore || purgingAudit}>Purge</button>
              </div>
            </div>
          </div>
        </section>

      </div>

      {/* ── Dirty Bar ── */}
      {isDirty && (
        <div style={{
          position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
          background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)',
          padding: '10px 16px', borderRadius: 8, boxShadow: 'var(--shadow-lg)',
          display: 'flex', alignItems: 'center', gap: 16, zIndex: 100,
          animation: 'slideUp 200ms ease-out',
        }}>
          <div style={{ fontSize: 13, fontWeight: 500 }}>You have unsaved changes</div>
          <div className="row" style={{ gap: 8 }}>
            <button className="btn ghost sm" onClick={() => refreshConfig()} disabled={saving}>Reset</button>
            <button className="btn primary sm" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving…' : 'Save Changes'}
            </button>
          </div>
        </div>
      )}

      {/* ── Toast ── */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 20, right: 20, zIndex: 110,
          padding: '9px 14px', borderRadius: 6,
          display: 'flex', alignItems: 'center', gap: 8,
          fontSize: 12.5, fontWeight: 500,
          background: toast.type === 'danger' ? 'var(--danger-bg)' : toast.type === 'warn' ? 'var(--warn-bg)' : 'var(--success-bg)',
          border: '1px solid ' + (toast.type === 'danger' ? 'var(--danger)' : toast.type === 'warn' ? 'var(--warn)' : 'var(--success)'),
          color: toast.type === 'danger' ? 'var(--danger)' : toast.type === 'warn' ? 'var(--warn)' : 'var(--success)',
          boxShadow: 'var(--shadow-lg)',
        }}>
          {toast.type === 'danger' ? <Icon.X width={13} height={13}/> : toast.type === 'warn' ? <Icon.Warn width={13} height={13}/> : <Icon.Check width={13} height={13}/>}
          {toast.msg}
        </div>
      )}

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
        @keyframes slideUp { from { transform: translate(-50%, 20px); opacity: 0; } to { transform: translate(-50%, 0); opacity: 1; } }
        .input-inline {
          background: transparent; border: 1px solid transparent; border-radius: 4px;
          padding: 4px 8px; color: var(--fg); width: 100%; outline: none; transition: all 0.1s;
        }
        .input-inline:hover { background: var(--surface-2); border-color: var(--hairline); }
        .input-inline:focus { background: var(--surface-2); border-color: var(--fg-muted); }
      `}</style>
      <CLIFooter command="shark admin config dump"/>
    </div>
  );
}
