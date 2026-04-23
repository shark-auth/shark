// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { useAPI } from './api'

// Notification bell + panel — backed by /admin/health.
// Health endpoint returns components keyed by name with status:"warning"|"error".
// We surface those as notifications. No data → no badge.

function deriveNotifications(health) {
  if (!health) return [];
  const out = [];
  // Health response shape varies — try common keys.
  // 1) flat warnings array
  if (Array.isArray(health.warnings)) {
    for (const w of health.warnings) {
      out.push({
        id: w.id || w.code || w.message,
        severity: w.severity || 'warning',
        title: w.title || w.code || 'Warning',
        message: w.message || w.detail || '',
      });
    }
  }
  // 2) components map
  if (health.components && typeof health.components === 'object') {
    for (const [name, c] of Object.entries(health.components)) {
      if (!c || typeof c !== 'object') continue;
      const status = String(c.status || '').toLowerCase();
      if (status === 'warning' || status === 'warn' || status === 'error' || status === 'down') {
        out.push({
          id: name,
          severity: status === 'error' || status === 'down' ? 'error' : 'warning',
          title: name.replace(/_/g, ' '),
          message: c.message || c.detail || '',
        });
      }
    }
  }
  // 3) flat keys we know about
  const flatKeys = [
    ['smtp_unconfigured', 'SMTP not configured', 'Email delivery is disabled until SMTP is set.'],
    ['jwt_no_keys', 'No JWT signing keys', 'Token issuance is unavailable until a key is generated.'],
    ['no_oauth_providers', 'No OAuth providers', 'No upstream identity providers are configured.'],
    ['expiring_keys', 'Signing keys expiring soon', 'One or more JWT keys expire within 7 days.'],
    ['db_status', 'Database degraded', 'Database health check returned a non-OK status.'],
  ];
  for (const [key, title, message] of flatKeys) {
    const v = health[key];
    if (v === true || v === 'warning' || v === 'error' || (typeof v === 'string' && v && v !== 'ok' && v !== 'healthy')) {
      out.push({ id: key, severity: 'warning', title, message });
    }
  }
  // De-dupe by id, prefer first
  const seen = new Set();
  return out.filter(n => {
    if (seen.has(n.id)) return false;
    seen.add(n.id);
    return true;
  });
}

export function NotificationBell({ setPage }) {
  const [open, setOpen] = React.useState(false);
  const btnRef = React.useRef(null);
  const panelRef = React.useRef(null);

  const { data: health } = useAPI('/admin/health');
  const notifications = deriveNotifications(health);
  const count = notifications.length;

  React.useEffect(() => {
    if (!open) return;
    const click = (e) => {
      if (panelRef.current?.contains(e.target)) return;
      if (btnRef.current?.contains(e.target)) return;
      setOpen(false);
    };
    const esc = (e) => { if (e.key === 'Escape') setOpen(false); };
    window.addEventListener('mousedown', click);
    window.addEventListener('keydown', esc);
    return () => {
      window.removeEventListener('mousedown', click);
      window.removeEventListener('keydown', esc);
    };
  }, [open]);

  return (
    <>
      <button
        ref={btnRef}
        className="btn ghost icon"
        title={count ? `${count} notification${count !== 1 ? 's' : ''}` : 'Notifications'}
        style={{ position: 'relative' }}
        onClick={() => setOpen(v => !v)}
      >
        <Icon.Bell width={14} height={14}/>
        {count > 0 && (
          <span style={{
            position: 'absolute', top: 4, right: 4,
            minWidth: 12, height: 12, padding: '0 3px',
            borderRadius: 99,
            background: 'var(--danger)', color: '#fff',
            fontSize: 9, fontWeight: 600,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            lineHeight: 1,
          }}>{count > 9 ? '9+' : count}</span>
        )}
      </button>

      {open && (
        <div ref={panelRef} style={{
          position: 'absolute', top: 'calc(var(--topbar-h) + 4px)', right: 16,
          width: 320, maxHeight: 420, overflowY: 'auto',
          background: 'var(--surface-1)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 'var(--radius)',
          boxShadow: 'var(--shadow-lg)',
          zIndex: 55,
          animation: 'fadeIn 80ms ease-out',
        }}>
          <div style={{
            padding: '10px 12px', borderBottom: '1px solid var(--hairline)',
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <span style={{ fontSize: 12, fontWeight: 600, fontFamily: 'var(--font-display)' }}>Notifications</span>
            {setPage && (
              <button
                className="btn ghost sm"
                onClick={() => { setOpen(false); setPage('overview'); }}
                style={{ fontSize: 11, color: 'var(--fg-dim)' }}
              >View all</button>
            )}
          </div>
          {count === 0 ? (
            <div style={{ padding: '24px 16px', textAlign: 'center', fontSize: 12, color: 'var(--fg-dim)' }}>
              All systems healthy
            </div>
          ) : notifications.map(n => (
            <div key={n.id} style={{
              padding: '10px 12px', borderBottom: '1px solid var(--hairline)',
              display: 'flex', gap: 9,
            }}>
              <span style={{
                width: 6, height: 6, borderRadius: 99, marginTop: 6,
                background: n.severity === 'error' ? 'var(--danger)' : 'var(--warn)',
                flexShrink: 0,
              }}/>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 500, color: 'var(--fg)', textTransform: 'capitalize' }}>
                  {n.title}
                </div>
                {n.message && (
                  <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 2, lineHeight: 1.45 }}>
                    {n.message}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
