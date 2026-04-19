// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Consents page — per-user OAuth consent grants
// Lists the authenticated user's active consent grants with revoke action.
// NOTE: backend endpoint /auth/consents is SESSION-authenticated, not admin-key
// authenticated. When called from the admin UI (Bearer sk_live_*), it will
// typically 401. We degrade gracefully into an explanatory empty state.

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };

export function Consents() {
  const [filter, setFilter] = React.useState('all'); // all | hasExpiry | noExpiry
  const [query, setQuery] = React.useState('');
  const [revokeModal, setRevokeModal] = React.useState(null);

  const { data: raw, loading, error, refresh } = useAPI('/auth/consents');
  const consents = raw?.data || [];

  // Treat 401/403 or any auth-shaped error as "endpoint not available to admin key"
  const authMismatch = !!error && /unauthorized|forbidden|401|403/i.test(error);
  const showFallback = authMismatch || (!loading && !error && consents.length === 0);

  const filtered = consents.filter(c => {
    if (filter === 'hasExpiry' && !c.expires_at) return false;
    if (filter === 'noExpiry' && c.expires_at) return false;
    if (query) {
      const q = query.toLowerCase();
      const name = (c.agent_name || '').toLowerCase();
      const cid = (c.client_id || '').toLowerCase();
      if (!name.includes(q) && !cid.includes(q)) return false;
    }
    return true;
  });

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden', minWidth: 0 }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Consents</h1>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              Active OAuth consent grants · {consents.length} total
            </p>
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="row" style={{ padding: '10px 20px', gap: 8, borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{
          flex: 1, gap: 6, padding: '0 8px', height: 28, maxWidth: 320,
          background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 4,
        }}>
          <Icon.Search width={12} height={12} style={{opacity:0.6}}/>
          <input placeholder="Filter by agent name or client_id…"
            value={query} onChange={e => setQuery(e.target.value)}
            style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
        </div>
        <div className="row" style={{ gap: 2, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', padding: 2, height: 28 }}>
          {[['all', 'All'], ['hasExpiry', 'Has expiry'], ['noExpiry', 'No expiry']].map(([v, l]) => (
            <button key={v} onClick={() => setFilter(v)} style={{
              padding: '0 10px', fontSize: 11, height: 22,
              background: filter === v ? 'var(--surface-3)' : 'transparent',
              color: filter === v ? 'var(--fg)' : 'var(--fg-muted)',
              border: 0, borderRadius: 3, cursor: 'pointer', fontWeight: filter === v ? 500 : 400,
            }}>{l}</button>
          ))}
        </div>
        <div style={{flex:1}}/>
        <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {consents.length}</span>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflow: 'auto' }}>
        {loading ? (
          <div className="faint" style={{padding: 20, fontSize: 12}}>Loading consents…</div>
        ) : showFallback ? (
          <FallbackEmptyState authMismatch={authMismatch}/>
        ) : filtered.length === 0 ? (
          <div className="faint" style={{padding: 40, fontSize: 12, textAlign: 'center'}}>
            No consents match your filters.
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={appThStyle}>Agent Name</th>
                <th style={appThStyle}>Client ID</th>
                <th style={appThStyle}>Scopes</th>
                <th style={appThStyle}>Granted</th>
                <th style={appThStyle}>Expires</th>
                <th style={{...appThStyle, textAlign: 'right'}}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(c => {
                const scopes = splitScope(c.scope);
                return (
                  <tr key={c.id}>
                    <td style={appTdStyle}>
                      <div style={{fontWeight: 500}}>{c.agent_name || '—'}</div>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {truncateMiddle(c.client_id, 24)}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <div className="row" style={{gap: 4, flexWrap: 'wrap', maxWidth: 320}}>
                        {scopes.length === 0 ? (
                          <span className="faint mono" style={{fontSize: 10.5}}>—</span>
                        ) : scopes.map(s => (
                          <span key={s} className="chip mono" style={{height: 18, fontSize: 10}}>{s}</span>
                        ))}
                      </div>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(c.granted_at)}</span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {c.expires_at ? relativeTime(c.expires_at) : 'never'}
                      </span>
                    </td>
                    <td style={{...appTdStyle, textAlign: 'right'}}>
                      <button className="btn danger sm" onClick={() => setRevokeModal(c)}>Revoke</button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* CLI parity footer */}
      <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
        <Icon.Debug width={11} height={11} style={{opacity:0.5}}/>
        <span className="faint">CLI parity:</span>
        <span className="mono faint">shark consent list</span>
        <div style={{flex:1}}/>
        <span className="faint mono">DELETE /api/v1/auth/consents/{'{id}'}</span>
      </div>

      {revokeModal && (
        <RevokeConsentModal
          consent={revokeModal}
          onClose={() => setRevokeModal(null)}
          onSuccess={() => {
            setRevokeModal(null);
            refresh();
          }}
        />
      )}
    </div>
  );
}

function FallbackEmptyState({ authMismatch }) {
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      minHeight: '100%', padding: 40, textAlign: 'center',
    }}>
      <div style={{ maxWidth: 520 }}>
        <div style={{
          width: 44, height: 44, margin: '0 auto 14px',
          borderRadius: 8, background: 'var(--surface-2)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          border: '1px solid var(--hairline-strong)',
          color: 'var(--fg-dim)',
        }}>
          <Icon.Consent width={22} height={22}/>
        </div>
        {authMismatch ? (
          <>
            <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--fg)' }}>
              Consent management endpoint is session-authenticated
            </div>
            <div style={{ marginTop: 10, fontSize: 12, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
              Self-service UI for users can be found at their account dashboard.
              Admin-wide consent viewing is not yet available — track via Audit Log
              (<span className="mono">action=consent.granted</span> / <span className="mono">consent.revoked</span>).
            </div>
            <div className="row" style={{ justifyContent: 'center', gap: 8, marginTop: 16 }}>
              <button className="btn ghost sm" onClick={() => { /* placeholder — no cross-page nav */ }}>
                <Icon.Audit width={11} height={11}/> Audit log
              </button>
            </div>
          </>
        ) : (
          <>
            <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--fg)' }}>
              No consents granted yet
            </div>
            <div style={{ marginTop: 10, fontSize: 12, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
              Users will authorize agents via the OAuth flow.
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function RevokeConsentModal({ consent, onClose, onSuccess }) {
  const toast = useToast();
  const [working, setWorking] = React.useState(false);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape' && !working) onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose, working]);

  const doRevoke = async () => {
    setWorking(true);
    setError(null);
    try {
      await API.del('/auth/consents/' + consent.id);
      toast.undo(`Revoked consent for "${consent.agent_name || consent.client_id}"`, () => { /* action is irreversible; undo window just lets toast dismiss */ });
      onSuccess();
    } catch (e) {
      setError(e.message);
      toast.error(`Failed to revoke: ${e.message}`);
    } finally {
      setWorking(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={() => !working && onClose()}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Revoke consent</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Revoking consent for <b style={{color:'var(--fg)'}}>{consent.agent_name || consent.client_id}</b> will also revoke all active tokens issued to that agent for this user. They will need to re-authorize to regain access.
        </p>
        <div style={{background:'var(--surface-1)', border:'1px solid var(--hairline)', padding: 10, borderRadius: 4, marginTop: 14, fontSize: 11.5}}>
          <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'6px 14px'}}>
            <span className="faint">Client ID</span>
            <span className="mono" style={{wordBreak: 'break-all'}}>{consent.client_id}</span>
            <span className="faint">Scopes</span>
            <span className="mono">{consent.scope || '—'}</span>
            <span className="faint">Granted</span>
            <span className="mono">{relativeTime(consent.granted_at)}</span>
          </div>
        </div>
        {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
        <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
          <button className="btn danger" onClick={doRevoke} disabled={working}>
            {working ? 'Revoking…' : 'Revoke'}
          </button>
        </div>
      </div>
    </div>
  );
}

// Format ISO timestamp or epoch ms to relative string
function relativeTime(val) {
  if (!val) return '—';
  const ms = typeof val === 'string' ? new Date(val).getTime() : val;
  if (!Number.isFinite(ms)) return '—';
  const diff = Date.now() - ms;
  const abs = Math.abs(diff);
  const fmt = (n, u) => Math.floor(n) + u + (diff < 0 ? ' from now' : ' ago');
  if (abs < 60e3) return diff < 0 ? 'soon' : 'just now';
  if (abs < 3600e3) return fmt(abs / 60e3, 'm');
  if (abs < 86400e3) return fmt(abs / 3600e3, 'h');
  return fmt(abs / 86400e3, 'd');
}

function truncateMiddle(s, n) {
  if (!s) return '';
  if (s.length <= n) return s;
  const keep = Math.floor((n - 1) / 2);
  return s.slice(0, keep) + '…' + s.slice(-keep);
}

function splitScope(scope) {
  if (!scope) return [];
  return String(scope).split(/[\s,]+/).map(s => s.trim()).filter(Boolean);
}
