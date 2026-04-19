// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'

// Settings page — server info, email, session management, audit retention, danger zone

export function Settings() {
  const { data: health, loading: healthLoading } = useAPI('/admin/health');
  const { data: config, loading: configLoading } = useAPI('/admin/config');

  const [toast, setToast] = React.useState(null);
  const [purgingSession, setPurgingSession] = React.useState(false);
  const [purgingAudit, setPurgingAudit] = React.useState(false);
  const [auditBefore, setAuditBefore] = React.useState('');
  const [deleteConfirmInput, setDeleteConfirmInput] = React.useState('');
  const [deleteConfirmOpen, setDeleteConfirmOpen] = React.useState(false);

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

  function handleDeleteAllUsers() {
    setDeleteConfirmOpen(false);
    setDeleteConfirmInput('');
    showToast('Not implemented yet', 'warn');
  }

  const baseUrl = config?.base_url || window.location.origin;
  const corsOrigins = config?.cors_origins || [];
  const isDevMode = config?.dev_mode === true || config?.environment === 'development';
  const smtpConfigured = !!(config?.smtp_host || health?.smtp?.host);

  // Row helper for read-only config display
  function ConfigRow({ label, children }) {
    return (
      <div style={{
        display: 'flex', alignItems: 'center',
        padding: '9px 14px',
        borderBottom: '1px solid var(--hairline)',
        gap: 12,
        minHeight: 38,
      }}>
        <span style={{ width: 160, flexShrink: 0, fontSize: 11.5, color: 'var(--fg-dim)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{label}</span>
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

  return (
    <div style={{ height: '100%', overflowY: 'auto', padding: 20 }}>
      <div style={{ maxWidth: 760, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 20 }}>

        {/* ── Server ── */}
        <section>
          <SectionHeader title="Server" sub="Read-only system configuration and runtime info"/>
          <div className="card">
            <ConfigRow label="Base URL">
              {configLoading
                ? <Skeleton w={240}/>
                : <CopyField value={baseUrl} truncate={48}/>
              }
            </ConfigRow>
            <ConfigRow label="Version">
              {healthLoading
                ? <Skeleton w={80}/>
                : <span className="mono" style={{ fontSize: 12 }}>{health?.version ?? '—'}</span>
              }
            </ConfigRow>
            <ConfigRow label="Uptime">
              {healthLoading
                ? <Skeleton w={100}/>
                : <span className="mono" style={{ fontSize: 12 }}>{formatUptime(health?.uptime_seconds)}</span>
              }
            </ConfigRow>
            <ConfigRow label="DB Size">
              {healthLoading
                ? <Skeleton w={80}/>
                : (
                  <div className="row" style={{ gap: 8 }}>
                    <span className="mono" style={{ fontSize: 12 }}>{formatBytes(health?.db?.size_mb)}</span>
                    {health?.db?.status && (
                      <span className={"chip" + (health.db.status === 'healthy' ? ' success' : ' warn')} style={{ height: 16, fontSize: 10 }}>
                        <span className={"dot" + (health.db.status === 'healthy' ? ' success' : ' warn')} style={{ width: 5, height: 5 }}/>
                        {health.db.status}
                      </span>
                    )}
                  </div>
                )
              }
            </ConfigRow>
            <ConfigRow label="CORS Origins">
              {configLoading
                ? <Skeleton w={200}/>
                : corsOrigins.length > 0
                  ? (
                    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 5 }}>
                      {corsOrigins.map((o, i) => (
                        <span key={i} className="chip mono" style={{ height: 18, fontSize: 10.5 }}>{o}</span>
                      ))}
                    </div>
                  )
                  : <span className="faint" style={{ fontSize: 12 }}>None configured</span>
              }
            </ConfigRow>
            <ConfigRow label="Environment">
              {configLoading
                ? <Skeleton w={80}/>
                : (
                  <div className="row" style={{ gap: 8 }}>
                    <span className="mono" style={{ fontSize: 12 }}>{config?.environment ?? '—'}</span>
                    {isDevMode && (
                      <span className="chip warn" style={{ height: 16, fontSize: 10 }}>dev mode</span>
                    )}
                  </div>
                )
              }
            </ConfigRow>
            <ConfigRow label="Session Mode">
              {configLoading
                ? <Skeleton w={100}/>
                : (
                  <div className="row" style={{ gap: 8 }}>
                    <span className={"chip" + (config?.jwt_mode ? '' : ' success')} style={{ height: 18, fontSize: 10 }}>
                      {config?.jwt_mode ? 'JWT' : 'Cookie'}
                    </span>
                    {config?.jwt_mode && (
                      <span className="mono faint" style={{ fontSize: 11 }}>{config?.jwt_algorithm || 'ES256'}</span>
                    )}
                    {!config?.jwt_mode && (
                      <span className="faint" style={{ fontSize: 11 }}>server-side sessions</span>
                    )}
                  </div>
                )
              }
            </ConfigRow>
          </div>
        </section>

        {/* ── Email ── */}
        <section>
          <SectionHeader title="Email" sub="Outbound email provider configuration"/>
          <div className="card">
            <ConfigRow label="Provider">
              {healthLoading
                ? <Skeleton w={160}/>
                : smtpConfigured
                  ? (
                    <div className="row" style={{ gap: 8 }}>
                      <span className="chip success" style={{ height: 16, fontSize: 10 }}>
                        <span className="dot success" style={{ width: 5, height: 5 }}/>
                        SMTP configured
                      </span>
                      {health?.smtp?.host && (
                        <span className="mono faint" style={{ fontSize: 11 }}>{health.smtp.host}</span>
                      )}
                    </div>
                  )
                  : (
                    <span className="chip" style={{ height: 16, fontSize: 10, color: 'var(--fg-dim)' }}>
                      <span className="dot" style={{ width: 5, height: 5 }}/>
                      Not configured
                    </span>
                  )
              }
            </ConfigRow>
            {(health?.smtp?.tier || health?.smtp?.sent_today != null) && (
              <ConfigRow label="Email Tier">
                <div className="row" style={{ gap: 8 }}>
                  <span className={"chip" + (health.smtp.tier === 'testing' ? ' warn' : ' success')} style={{ height: 18, fontSize: 10 }}>
                    {health.smtp.tier || 'default'}
                  </span>
                  {health.smtp.sent_today != null && (
                    <span className="mono" style={{ fontSize: 11, color: 'var(--fg-muted)' }}>
                      {health.smtp.sent_today}/{health.smtp.daily_limit ?? '∞'} today
                    </span>
                  )}
                  {health.smtp.tier === 'testing' && (
                    <span className="faint" style={{ fontSize: 10.5 }}>switch before prod: <span className="mono">shark email setup</span></span>
                  )}
                </div>
              </ConfigRow>
            )}
            <div style={{ padding: '10px 14px', display: 'flex', alignItems: 'center', gap: 12 }}>
              <div title="Endpoint not available yet" style={{ display: 'inline-block' }}>
                <button
                  className="btn sm"
                  disabled
                  style={{ cursor: 'not-allowed', opacity: 0.45 }}
                >
                  <Icon.Mail width={11} height={11}/>
                  Send test email
                </button>
              </div>
              <span className="faint" style={{ fontSize: 11 }}>Endpoint not available yet</span>
            </div>
          </div>
        </section>

        {/* ── Session Management ── */}
        <section>
          <SectionHeader title="Session Management" sub="Clean up expired sessions from the database"/>
          <div className="card">
            <div style={{ padding: '12px 14px' }}>
              <div style={{ marginBottom: 10 }}>
                <div style={{ fontSize: 12.5, color: 'var(--fg-muted)', lineHeight: 1.5 }}>
                  Expired sessions accumulate over time and are not automatically removed. Purging frees up database space and may improve query performance.
                </div>
              </div>
              <div className="row" style={{ gap: 10 }}>
                <button
                  className="btn sm"
                  onClick={handlePurgeSessions}
                  disabled={purgingSession}
                  style={{ opacity: purgingSession ? 0.65 : 1 }}
                >
                  <Icon.Refresh width={11} height={11} style={purgingSession ? { animation: 'spin 0.8s linear infinite' } : {}}/>
                  {purgingSession ? 'Purging…' : 'Purge expired sessions'}
                </button>
              </div>
            </div>
          </div>
        </section>

        {/* ── Audit Log Retention ── */}
        <section>
          <SectionHeader title="Audit Log Retention" sub="Permanently delete audit log entries older than a given date"/>
          <div className="card">
            <div style={{ padding: '12px 14px' }}>
              <div style={{ marginBottom: 10 }}>
                <div style={{ fontSize: 12.5, color: 'var(--fg-muted)', lineHeight: 1.5 }}>
                  Audit log entries before the selected date will be permanently deleted. This action cannot be undone.
                </div>
              </div>
              <div className="row" style={{ gap: 8, flexWrap: 'wrap' }}>
                <input
                  type="date"
                  value={auditBefore}
                  onChange={e => setAuditBefore(e.target.value)}
                  style={{
                    height: 28, padding: '0 8px',
                    background: 'var(--surface-2)',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 5,
                    color: 'var(--fg)',
                    fontSize: 12,
                    colorScheme: 'dark',
                  }}
                />
                <button
                  className="btn sm danger"
                  onClick={handlePurgeAudit}
                  disabled={!auditBefore || purgingAudit}
                  style={{ opacity: (!auditBefore || purgingAudit) ? 0.55 : 1 }}
                >
                  <Icon.Refresh width={11} height={11} style={purgingAudit ? { animation: 'spin 0.8s linear infinite' } : {}}/>
                  {purgingAudit ? 'Purging…' : 'Purge logs before this date'}
                </button>
              </div>
            </div>
          </div>
        </section>

        {/* ── Danger Zone ── */}
        <section style={{ marginBottom: 8 }}>
          <SectionHeader title="Danger Zone" sub="Irreversible operations — proceed with caution"/>
          <div className="card" style={{
            borderColor: 'color-mix(in oklch, var(--danger) 40%, var(--hairline))',
            background: 'color-mix(in oklch, var(--danger) 4%, var(--surface-1))',
          }}>
            <div style={{ padding: '12px 14px' }}>
              <div className="row" style={{ justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
                <div>
                  <div style={{ fontSize: 12.5, fontWeight: 500, color: 'var(--fg)' }}>Delete all users</div>
                  <div style={{ fontSize: 11.5, color: 'var(--fg-dim)', marginTop: 2 }}>
                    Permanently removes every user account, session, and associated data from the database.
                  </div>
                </div>
                <button
                  className="btn sm danger"
                  onClick={() => setDeleteConfirmOpen(true)}
                >
                  <Icon.X width={11} height={11}/>
                  Delete all users
                </button>
              </div>
            </div>
          </div>
        </section>

      </div>

      {/* ── Confirm modal — Delete all users ── */}
      {deleteConfirmOpen && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 50,
          background: 'rgba(0,0,0,0.7)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }} onClick={() => { setDeleteConfirmOpen(false); setDeleteConfirmInput(''); }}>
          <div
            className="card"
            style={{ width: 400, padding: 20, animation: 'slideIn 120ms ease-out' }}
            onClick={e => e.stopPropagation()}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 14 }}>
              <div style={{
                width: 28, height: 28, borderRadius: 6, flexShrink: 0,
                background: 'var(--danger-bg)',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }}>
                <Icon.Warn width={14} height={14} style={{ color: 'var(--danger)' }}/>
              </div>
              <div>
                <div style={{ fontWeight: 600, fontSize: 13 }}>Delete all users</div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 1 }}>This action is permanent and cannot be undone.</div>
              </div>
            </div>

            <div style={{ fontSize: 12, color: 'var(--fg-muted)', marginBottom: 12, lineHeight: 1.5 }}>
              Type <span className="mono" style={{ color: 'var(--danger)', background: 'var(--danger-bg)', padding: '1px 5px', borderRadius: 3, fontSize: 11 }}>DELETE ALL USERS</span> to confirm.
            </div>

            <input
              autoFocus
              type="text"
              value={deleteConfirmInput}
              onChange={e => setDeleteConfirmInput(e.target.value)}
              placeholder="DELETE ALL USERS"
              style={{
                width: '100%', height: 32, padding: '0 10px',
                background: 'var(--surface-2)',
                border: '1px solid var(--hairline-strong)',
                borderRadius: 5,
                color: 'var(--fg)',
                fontSize: 12,
                marginBottom: 14,
              }}
            />

            <div className="row" style={{ justifyContent: 'flex-end', gap: 8 }}>
              <button className="btn sm ghost" onClick={() => { setDeleteConfirmOpen(false); setDeleteConfirmInput(''); }}>Cancel</button>
              <button
                className="btn sm danger"
                disabled={deleteConfirmInput !== 'DELETE ALL USERS'}
                onClick={handleDeleteAllUsers}
                style={{ opacity: deleteConfirmInput !== 'DELETE ALL USERS' ? 0.45 : 1 }}
              >
                Delete all users
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Toast ── */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: 20, right: 20, zIndex: 60,
          padding: '9px 14px', borderRadius: 6,
          display: 'flex', alignItems: 'center', gap: 8,
          fontSize: 12.5, fontWeight: 500,
          animation: 'slideIn 160ms ease-out',
          background: toast.type === 'danger' ? 'var(--danger-bg)' : toast.type === 'warn' ? 'var(--warn-bg)' : 'var(--success-bg)',
          border: '1px solid ' + (toast.type === 'danger' ? 'var(--danger)' : toast.type === 'warn' ? 'var(--warn)' : 'var(--success)'),
          color: toast.type === 'danger' ? 'var(--danger)' : toast.type === 'warn' ? 'var(--warn)' : 'var(--success)',
          boxShadow: 'var(--shadow-lg)',
        }}>
          {toast.type === 'danger'
            ? <Icon.X width={13} height={13}/>
            : toast.type === 'warn'
              ? <Icon.Warn width={13} height={13}/>
              : <Icon.Check width={13} height={13}/>
          }
          {toast.msg}
        </div>
      )}

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </div>
  );
}

