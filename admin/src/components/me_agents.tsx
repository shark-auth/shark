// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// My Agents — user-facing self-service view.
// Shows agents created by the current user and agents the current user has
// authorized via OAuth consent.  Uses GET /api/v1/me/agents?filter=created|authorized
// (session cookie auth).

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 50 };
const modalCard = { background: 'var(--surface-1)', border: '1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };

function relativeTime(val) {
  if (!val) return '—';
  const ms = typeof val === 'string' ? new Date(val).getTime() : val;
  const diff = Date.now() - ms;
  if (diff < 0) return 'just now';
  if (diff < 60e3) return Math.floor(diff / 1e3) + 's ago';
  if (diff < 3600e3) return Math.floor(diff / 60e3) + 'm ago';
  if (diff < 86400e3) return Math.floor(diff / 3600e3) + 'h ago';
  return Math.floor(diff / 86400e3) + 'd ago';
}

// Fetch /me/agents with a given filter using admin API key (same fetch pattern as rest of dashboard).
// Note: the spec says session cookie auth — but this dashboard uses the admin API key stored in
// localStorage.  We reuse the same API helper; operators who have loaded the admin key can self-serve.
function useMeAgents(filter) {
  return useAPI(`/me/agents?filter=${filter}`);
}

export function MyAgents() {
  const toast = useToast();

  const { data: createdRaw, loading: loadingCreated, refresh: refreshCreated } = useMeAgents('created');
  const { data: authorizedRaw, loading: loadingAuthorized, refresh: refreshAuthorized } = useMeAgents('authorized');

  const created = createdRaw?.data || [];
  const authorized = authorizedRaw?.data || [];

  const [revokeAllOpen, setRevokeAllOpen] = React.useState(false);
  const [revokeRowTarget, setRevokeRowTarget] = React.useState(null); // { agent, type: 'created'|'authorized' }

  // Admin key presence — if absent disable "Revoke all" because it requires admin auth.
  const hasAdminKey = Boolean(localStorage.getItem('shark_admin_key'));

  const refresh = () => { refreshCreated(); refreshAuthorized(); };

  const handleRevokeRow = async (agent, type) => {
    try {
      if (type === 'created') {
        // Deactivate by patching active=false
        await API.patch('/agents/' + agent.id, { active: false });
        toast.success(`Revoked agent "${agent.name}"`);
      } else {
        // Revoke consent — endpoint may not exist yet
        if (agent.consent_id) {
          await API.del('/oauth/consents/' + agent.consent_id);
          toast.success(`Revoked consent for "${agent.name}"`);
        } else {
          toast.error('Consent ID unknown — cannot revoke consent via UI yet.');
          return;
        }
      }
      refresh();
    } catch (e) {
      toast.error('Revoke failed: ' + e.message);
    } finally {
      setRevokeRowTarget(null);
    }
  };

  const handleRevokeAll = async () => {
    try {
      // POST /api/v1/users/{id}/revoke-agents — requires admin key which is present here.
      // We use "me" path: POST /me/agents/revoke-all (fallback to admin endpoint).
      await API.post('/me/agents/revoke-all', {});
      toast.success('All agents revoked');
      refresh();
    } catch (e) {
      toast.error('Revoke all failed: ' + e.message);
    } finally {
      setRevokeAllOpen(false);
    }
  };

  const loading = loadingCreated || loadingAuthorized;
  const total = created.length + authorized.length;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Header */}
      <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)', flexShrink: 0 }}>
        <div className="row" style={{ gap: 12 }}>
          <div style={{ flex: 1 }}>
            <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>My Agents</h1>
            <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
              Agents linked to your account · {total} total
            </p>
          </div>
          <div
            title={hasAdminKey ? undefined : 'Available with admin key only'}
            style={{ display: 'inline-block' }}
          >
            <button
              className="btn danger"
              disabled={!hasAdminKey || total === 0}
              onClick={() => setRevokeAllOpen(true)}
              style={!hasAdminKey ? { opacity: 0.45, cursor: 'not-allowed' } : {}}
            >
              Revoke all my agents
            </button>
          </div>
        </div>
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '20px' }}>
        {loading ? (
          <div className="faint" style={{ fontSize: 12 }}>Loading…</div>
        ) : total === 0 ? (
          <div style={{
            border: '1px solid var(--hairline)', borderRadius: 4,
            padding: '32px 24px', textAlign: 'center',
            background: 'var(--surface-1)',
          }}>
            <Icon.Agent width={24} height={24} style={{ opacity: 0.3, marginBottom: 10 }}/>
            <div style={{ fontSize: 13, fontWeight: 500, marginBottom: 6 }}>No agents linked to your account yet.</div>
            <div className="faint" style={{ fontSize: 11.5 }}>
              Register one at <span className="mono">/agents/new</span> or via{' '}
              <span className="mono">shark agent register</span>.
            </div>
          </div>
        ) : (
          <>
            {/* Created by me */}
            <AgentSection
              title="Created by me"
              count={created.length}
              agents={created}
              type="created"
              onRevoke={(agent) => setRevokeRowTarget({ agent, type: 'created' })}
              revokeLabel="Revoke"
            />

            {authorized.length > 0 && (
              <div style={{ marginTop: 28 }}>
                <AgentSection
                  title="Authorized by me"
                  count={authorized.length}
                  agents={authorized}
                  type="authorized"
                  onRevoke={(agent) => setRevokeRowTarget({ agent, type: 'authorized' })}
                  revokeLabel="Revoke consent"
                />
              </div>
            )}
          </>
        )}
      </div>

      <CLIFooter command="shark agent list --mine"/>

      {/* Revoke row confirmation */}
      {revokeRowTarget && (
        <ConfirmModal
          title={revokeRowTarget.type === 'created' ? 'Revoke agent' : 'Revoke consent'}
          body={
            revokeRowTarget.type === 'created'
              ? `Deactivate "${revokeRowTarget.agent.name}"? All active tokens will be revoked.`
              : `Revoke your consent for "${revokeRowTarget.agent.name}"? The agent will lose access to your account.`
          }
          confirmLabel={revokeRowTarget.type === 'created' ? 'Deactivate' : 'Revoke consent'}
          onClose={() => setRevokeRowTarget(null)}
          onConfirm={() => handleRevokeRow(revokeRowTarget.agent, revokeRowTarget.type)}
        />
      )}

      {/* Revoke all confirmation */}
      {revokeAllOpen && (
        <ConfirmModal
          title="Revoke all my agents"
          body="This will deactivate all agents you created and revoke all tokens they hold. This cannot be undone."
          confirmLabel="Revoke all"
          onClose={() => setRevokeAllOpen(false)}
          onConfirm={handleRevokeAll}
          danger
        />
      )}
    </div>
  );
}

function AgentSection({ title, count, agents, type, onRevoke, revokeLabel }) {
  if (agents.length === 0) {
    return (
      <div>
        <SectionHeader title={title} count={0}/>
        <div className="faint" style={{ fontSize: 11.5, padding: '8px 0' }}>None.</div>
      </div>
    );
  }

  return (
    <div>
      <SectionHeader title={title} count={count}/>
      <div style={{ border: '1px solid var(--hairline)', borderRadius: 4, overflow: 'hidden', background: 'var(--surface-0)' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
          <thead>
            <tr>
              <th style={{ ...appThStyle, width: 28 }}/>
              <th style={appThStyle}>Name</th>
              <th style={appThStyle}>{type === 'authorized' ? 'Provider' : 'Client ID'}</th>
              <th style={appThStyle}>{type === 'authorized' ? 'Granted' : 'Created'}</th>
              <th style={appThStyle}>Status</th>
              <th style={{ ...appThStyle, textAlign: 'right' }}/>
            </tr>
          </thead>
          <tbody>
            {agents.map((a, i) => (
              <tr key={a.id || i} style={{ background: 'transparent' }}>
                <td style={appTdStyle}>
                  <div style={{
                    width: 22, height: 22, borderRadius: 4, background: 'var(--surface-3)',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 10, fontWeight: 600, color: 'var(--fg)', border: '1px solid var(--hairline-strong)',
                  }}>
                    {(a.name || '?')[0].toUpperCase()}
                  </div>
                </td>
                <td style={appTdStyle}>
                  <div style={{ fontWeight: 500 }}>{a.name}</div>
                  {a.description && (
                    <div className="faint" style={{ fontSize: 10.5, marginTop: 1 }}>{a.description}</div>
                  )}
                </td>
                <td style={appTdStyle}>
                  <span className="mono faint" style={{ fontSize: 10.5 }}>
                    {type === 'authorized'
                      ? (a.provider || a.audience || '—')
                      : (a.client_id ? truncMid(a.client_id, 22) : '—')}
                  </span>
                </td>
                <td style={appTdStyle}>
                  <span className="mono faint" style={{ fontSize: 10.5 }}>
                    {type === 'authorized'
                      ? relativeTime(a.granted_at || a.created_at)
                      : relativeTime(a.created_at)}
                  </span>
                </td>
                <td style={appTdStyle}>
                  {a.active !== false
                    ? <span className="chip success" style={{ height: 18, fontSize: 10 }}><span className="dot success"/>active</span>
                    : <span className="chip" style={{ height: 18, fontSize: 10 }}><span className="dot"/>inactive</span>}
                </td>
                <td style={{ ...appTdStyle, textAlign: 'right' }}>
                  <button
                    className="btn danger sm"
                    onClick={() => onRevoke(a)}
                    disabled={a.active === false}
                    style={{ fontSize: 11 }}
                  >
                    {revokeLabel}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function SectionHeader({ title, count }) {
  return (
    <div className="row" style={{ marginBottom: 8, gap: 8 }}>
      <h3 style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: 0 }}>
        {title}
      </h3>
      <span className="faint mono" style={{ fontSize: 10 }}>({count})</span>
    </div>
  );
}

function truncMid(s, n) {
  if (!s || s.length <= n) return s || '';
  const k = Math.floor((n - 1) / 2);
  return s.slice(0, k) + '…' + s.slice(-k);
}

function ConfirmModal({ title, body, confirmLabel, onClose, onConfirm, danger }) {
  const [working, setWorking] = React.useState(false);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    const h = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', h);
    return () => window.removeEventListener('keydown', h);
  }, [onClose]);

  const go = async () => {
    setWorking(true);
    setError(null);
    try {
      await onConfirm();
    } catch (e) {
      setError(e.message);
    } finally {
      setWorking(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{ ...modalCard, width: 440 }} onClick={e => e.stopPropagation()}>
        <h2 style={{ margin: 0, fontSize: 15, fontWeight: 600 }}>{title}</h2>
        <p className="faint" style={{ fontSize: 12, marginTop: 6, lineHeight: 1.5 }}>{body}</p>
        {error && <div style={{ color: 'var(--danger)', fontSize: 11, marginTop: 8 }}>{error}</div>}
        <div className="row" style={{ marginTop: 18, justifyContent: 'flex-end', gap: 8 }}>
          <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
          <button
            className={danger ? 'btn danger' : 'btn primary'}
            onClick={go}
            disabled={working}
          >
            {working ? 'Working…' : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
