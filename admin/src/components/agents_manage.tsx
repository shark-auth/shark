// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { usePageActions } from './useKeyboardShortcuts'
import { TeachEmptyState } from './TeachEmptyState'
import { useTabParam } from './useURLParams'
import { DelegationCanvasWithProvider, toEgoLayout } from './delegation_canvas'

// Agents page — OAuth 2.1 client registrations for autonomous/agent workloads
// Table view → slide-over detail → create flow with one-time secret reveal
// Mirrors applications.tsx layout conventions.

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };

const ALL_GRANTS = ['authorization_code', 'client_credentials', 'refresh_token', 'urn:ietf:params:oauth:grant-type:device_code', 'urn:ietf:params:oauth:grant-type:token-exchange'];

export function Agents({ setPage }) {
  const [selected, setSelected] = React.useState(null);
  const [tab, setTab] = useTabParam('config');
  const [filter, setFilter] = React.useState('all'); // all | active | inactive
  const [query, setQuery] = React.useState('');
  const [creatorFilter, setCreatorFilter] = React.useState(''); // '' = all, or user ID
  const [createOpen, setCreateOpen] = React.useState(false);
  const [deactivateModal, setDeactivateModal] = React.useState(null);
  const [bulkRevokeOpen, setBulkRevokeOpen] = React.useState(false);
  const [bulkRevokePattern, setBulkRevokePattern] = React.useState('');
  const [bulkRevokeReason, setBulkRevokeReason] = React.useState('');
  const [bulkRevokeLoading, setBulkRevokeLoading] = React.useState(false);
  const toast = useToast();

  // Fetch users for creator filter dropdown
  const { data: usersRaw } = useAPI('/users?limit=200');
  const allUsers = usersRaw?.data || usersRaw?.users || [];

  const agentsPath = creatorFilter
    ? `/agents?limit=200&created_by_user_id=${encodeURIComponent(creatorFilter)}`
    : '/agents?limit=200';
  const { data: listRaw, loading, refresh } = useAPI(agentsPath, [creatorFilter]);
  const agents = listRaw?.data || [];

  // Build user lookup map for "Created by" column
  const userMap = React.useMemo(() => {
    const m = {};
    for (const u of allUsers) { m[u.id] = u; }
    return m;
  }, [allUsers]);

  // Auto-open create modal when arriving with ?new=1, then strip the param.
  React.useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    if (p.get('new') === '1') {
      setCreateOpen(true);
      p.delete('new');
      const s = p.toString();
      window.history.replaceState(null, '', window.location.pathname + (s ? '?' + s : ''));
    }
    const s = p.get('search');
    if (s && !query) setQuery(s);
  }, []);

  // n=new, r=refresh
  usePageActions({ onNew: () => setCreateOpen(true), onRefresh: refresh });

  const filtered = agents.filter(a => {
    if (filter === 'active' && !a.active) return false;
    if (filter === 'inactive' && a.active) return false;
    if (query) {
      const q = query.toLowerCase();
      const name = (a.name || '').toLowerCase();
      const cid = (a.client_id || '').toLowerCase();
      if (!name.includes(q) && !cid.includes(q)) return false;
    }
    return true;
  });

  const activeCount = agents.filter(a => a.active).length;

  // Keep selected in sync after refresh
  React.useEffect(() => {
    if (selected) {
      const fresh = agents.find(a => a.id === selected.id);
      if (fresh) setSelected(fresh);
    }
  }, [agents, selected?.id]);

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 560px' : '1fr', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
        {/* Header */}
        <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ gap: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Agents</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                OAuth 2.1 clients · {agents.length} registered · {activeCount} active
              </p>
            </div>
            <button className="btn" onClick={() => setBulkRevokeOpen(true)} style={{ fontSize: 11 }}>
              Bulk Revoke by Pattern
            </button>
            <button className="btn primary" onClick={() => setCreateOpen(true)}>
              <Icon.Plus width={11} height={11}/> New agent
            </button>
          </div>
        </div>

        {/* Toolbar */}
        <div className="row" style={{ padding: '10px 20px', gap: 8, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{
            flex: 1, gap: 6, padding: '0 8px', height: 28, maxWidth: 320,
            background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 4,
          }}>
            <Icon.Search width={12} height={12} style={{opacity:0.6}}/>
            <input placeholder="Filter by name or client_id…"
              value={query} onChange={e => setQuery(e.target.value)}
              style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
          </div>
          <div className="row" style={{ gap: 2, border: '1px solid var(--hairline-strong)', borderRadius: 4, background: 'var(--surface-1)', padding: 2, height: 28 }}>
            {[['all', 'All'], ['active', 'Active'], ['inactive', 'Inactive']].map(([v, l]) => (
              <button key={v} onClick={() => setFilter(v)} style={{
                padding: '0 10px', fontSize: 11, height: 22,
                background: filter === v ? 'var(--surface-3)' : 'transparent',
                color: filter === v ? 'var(--fg)' : 'var(--fg-muted)',
                border: 0, borderRadius: 3, cursor: 'pointer', fontWeight: filter === v ? 500 : 400,
              }}>{l}</button>
            ))}
          </div>
          {/* Creator filter */}
          <select
            value={creatorFilter}
            onChange={e => setCreatorFilter(e.target.value)}
            style={{
              fontSize: 11, padding: '0 6px', height: 28, borderRadius: 4,
              border: '1px solid var(--hairline-strong)', background: 'var(--surface-1)',
              color: 'var(--fg)', outline: 'none', cursor: 'pointer',
            }}
          >
            <option value="">All creators</option>
            {allUsers.map(u => (
              <option key={u.id} value={u.id}>
                {u.email || u.username || u.name || u.id}
              </option>
            ))}
          </select>
          <div style={{flex:1}}/>
          <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {agents.length}</span>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          {loading ? (
            <div className="faint" style={{padding: 20, fontSize: 12}}>Loading agents…</div>
          ) : filtered.length === 0 && agents.length === 0 ? (
            <TeachEmptyState
              icon="Agent"
              title="No agents registered"
              description="Agents are OAuth 2.1 clients for autonomous workloads. Create one to issue tokens with scoped, time-bound access."
              createLabel="New Agent"
              onCreate={() => setCreateOpen(true)}
              cliSnippet="shark agent create --name 'my-agent'"
            />
          ) : filtered.length === 0 ? (
            <div className="faint" style={{padding: 40, fontSize: 12, textAlign: 'center'}}>
              No agents match your filters.
            </div>
          ) : (
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr>
                  <th style={{...appThStyle, width: 28}}/>
                  <th style={appThStyle}>Name</th>
                  <th style={appThStyle}>Client ID</th>
                  <th style={appThStyle}>Grants</th>
                  <th style={appThStyle}>Scopes</th>
                  <th style={appThStyle}>Created</th>
                  <th style={appThStyle}>Created by</th>
                  <th style={appThStyle}>Status</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map(a => (
                  <tr key={a.id}
                    onClick={() => { setSelected(a); setTab('config'); }}
                    style={{
                      cursor:'pointer',
                      background: selected?.id === a.id ? 'var(--surface-2)' : 'transparent',
                    }}>
                    <td style={appTdStyle}>
                      <div style={{
                        width:22, height:22, borderRadius: 4, background: 'var(--surface-3)',
                        display:'flex', alignItems:'center', justifyContent:'center',
                        fontSize: 10, fontWeight: 600, color: 'var(--fg)', border:'1px solid var(--hairline-strong)',
                      }}>{(a.name || '?')[0].toUpperCase()}</div>
                    </td>
                    <td style={appTdStyle}>
                      <div style={{fontWeight: 500}}>{a.name}</div>
                      {a.description && (
                        <div className="faint" style={{ fontSize: 10.5, marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 280 }}>
                          {a.description}
                        </div>
                      )}
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {truncateMiddle(a.client_id, 24)}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {(a.grant_types || []).length} grant{(a.grant_types || []).length !== 1 && 's'}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {(a.scopes || []).length} scope{(a.scopes || []).length !== 1 && 's'}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(a.created_at)}</span>
                    </td>
                    <td style={appTdStyle}>
                      {(() => {
                        const uid = a.created_by_user_id || a.created_by;
                        if (!uid) return <span className="faint mono" style={{fontSize:10.5}}>—</span>;
                        const u = userMap[uid];
                        const label = u ? (u.email || u.username || u.name || uid) : uid;
                        return (
                          <button
                            className="btn ghost sm"
                            onClick={e => { e.stopPropagation(); setPage && setPage('users', { search: uid }); }}
                            style={{ fontSize: 10.5, padding: '2px 4px', fontFamily: 'var(--font-mono)' }}
                            title={uid}
                          >
                            {label.length > 22 ? label.slice(0, 20) + '…' : label}
                          </button>
                        );
                      })()}
                    </td>
                    <td style={appTdStyle}>
                      {a.active
                        ? <span className="chip success" style={{height:18, fontSize:10}}><span className="dot success"/>active</span>
                        : <span className="chip" style={{height:18, fontSize:10}}><span className="dot"/>inactive</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* CLI parity footer */}
        <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
          <Icon.Debug width={11} height={11} style={{opacity:0.5}}/>
          <span className="faint">CLI parity:</span>
          <span className="mono faint">shark agent list</span>
          <div style={{flex:1}}/>
          <span className="faint mono">GET /api/v1/agents</span>
        </div>
      </div>

      {selected && (
        <AgentDetail
          agent={selected}
          tab={tab} setTab={setTab}
          setPage={setPage}
          onClose={() => setSelected(null)}
          onDeactivate={() => setDeactivateModal(selected)}
          onUpdate={async (updates) => {
            const updated = await API.patch('/agents/' + selected.id, updates);
            refresh();
            return updated;
          }}
        />
      )}

      {createOpen && (
        <CreateAgentSlideOver
          onClose={() => setCreateOpen(false)}
          onCreate={async (payload) => {
            const result = await API.post('/agents', payload);
            refresh();
            return result;
          }}
        />
      )}

      {deactivateModal && (
        <DeactivateModal
          agent={deactivateModal}
          onClose={() => setDeactivateModal(null)}
          onConfirm={async () => {
            await API.del('/agents/' + deactivateModal.id);
            setDeactivateModal(null);
            setSelected(null);
            refresh();
          }}
        />
      )}

      {bulkRevokeOpen && (
        <BulkRevokeModal
          pattern={bulkRevokePattern}
          reason={bulkRevokeReason}
          loading={bulkRevokeLoading}
          onPatternChange={setBulkRevokePattern}
          onReasonChange={setBulkRevokeReason}
          onClose={() => { setBulkRevokeOpen(false); setBulkRevokePattern(''); setBulkRevokeReason(''); }}
          onConfirm={async () => {
            setBulkRevokeLoading(true);
            try {
              const res = await API.post('/admin/oauth/revoke-by-pattern', {
                client_id_pattern: bulkRevokePattern,
                reason: bulkRevokeReason,
              });
              toast.success(`Revoked ${res.revoked_count} token(s) matching "${res.pattern_matched}"`);
              setBulkRevokeOpen(false);
              setBulkRevokePattern('');
              setBulkRevokeReason('');
              refresh();
            } catch (e) {
              toast.error(e?.message || 'Bulk revoke failed');
            } finally {
              setBulkRevokeLoading(false);
            }
          }}
        />
      )}
    </div>
  );
}

// BulkRevokeModal — pattern-based token revocation for admin use
function BulkRevokeModal({ pattern, reason, loading, onPatternChange, onReasonChange, onClose, onConfirm }) {
  return (
    <div style={modalBackdrop} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div style={{ ...modalCard, width: 420, display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span style={{ fontSize: 13, fontWeight: 600 }}>Bulk Revoke Tokens by Pattern</span>
          <button className="btn ghost sm" onClick={onClose} style={{ padding: '2px 6px' }}>✕</button>
        </div>
        <p style={{ margin: 0, fontSize: 11.5, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
          Revokes all active tokens whose <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>client_id</code> matches the given SQLite GLOB pattern (<code style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>*</code> = any sequence, <code style={{ fontFamily: 'var(--font-mono)', fontSize: 11 }}>?</code> = one char).
        </p>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <p style={sectionLabelStyle}>Client ID Pattern</p>
          <input
            value={pattern}
            onChange={e => onPatternChange(e.target.value)}
            placeholder="shark_agent_v3.2_*"
            style={{
              fontSize: 12, padding: '6px 10px', borderRadius: 4,
              border: '1px solid var(--hairline-strong)', background: 'var(--surface-0)',
              color: 'var(--fg)', outline: 'none', fontFamily: 'var(--font-mono)',
            }}
          />
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <p style={sectionLabelStyle}>Reason (optional)</p>
          <input
            value={reason}
            onChange={e => onReasonChange(e.target.value)}
            placeholder="Compromised credentials, rotating keys…"
            style={{
              fontSize: 12, padding: '6px 10px', borderRadius: 4,
              border: '1px solid var(--hairline-strong)', background: 'var(--surface-0)',
              color: 'var(--fg)', outline: 'none',
            }}
          />
        </div>
        <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
          <button className="btn" onClick={onClose} disabled={loading} style={{ fontSize: 12 }}>Cancel</button>
          <button
            className="btn danger"
            onClick={onConfirm}
            disabled={loading || !pattern.trim()}
            style={{ fontSize: 12 }}
          >
            {loading ? 'Revoking…' : 'Revoke'}
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
  const diff = Date.now() - ms;
  if (diff < 0) return 'just now';
  if (diff < 60e3) return Math.floor(diff / 1e3) + 's ago';
  if (diff < 3600e3) return Math.floor(diff / 60e3) + 'm ago';
  if (diff < 86400e3) return Math.floor(diff / 3600e3) + 'h ago';
  return Math.floor(diff / 86400e3) + 'd ago';
}

function truncateMiddle(s, n) {
  if (!s) return '';
  if (s.length <= n) return s;
  const keep = Math.floor((n - 1) / 2);
  return s.slice(0, keep) + '…' + s.slice(-keep);
}

function shortGrant(g) {
  if (!g) return '';
  if (g === 'urn:ietf:params:oauth:grant-type:token-exchange') return 'Token Exchange (RFC 8693)';
  if (g.startsWith('urn:ietf:params:oauth:grant-type:')) return g.replace('urn:ietf:params:oauth:grant-type:', '');
  return g;
}

function AgentDetail({ agent, tab, setTab, setPage, onClose, onDeactivate, onUpdate }) {
  const toast = useToast();
  const [revoking, setRevoking] = React.useState(false);
  const [tokensVersion, setTokensVersion] = React.useState(0);
  const [rotateModalOpen, setRotateModalOpen] = React.useState(false);

  const revokeAll = async () => {
    setRevoking(true);
    try {
      const r = await API.post('/agents/' + agent.id + '/tokens/revoke-all');
      toast.success(r?.message || `Revoked ${r?.count ?? 0} tokens`);
      // Bump version so AgentTokens re-fetches.
      setTokensVersion(v => v + 1);
    } catch (e) {
      toast.error(`Failed to revoke: ${e.message}`);
    } finally {
      setRevoking(false);
    }
  };

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)', background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 10 }}>
          <div style={{
            width: 32, height: 32, borderRadius: 5, background: 'var(--surface-3)',
            display:'flex', alignItems:'center', justifyContent:'center',
            fontWeight: 600, border:'1px solid var(--hairline-strong)',
          }}>{(agent.name || '?')[0].toUpperCase()}</div>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight: 500, fontSize: 14}}>{agent.name}</div>
            <div className="row" style={{gap: 6, fontSize: 10.5, marginTop: 2}}>
              <span className="mono faint">{truncateMiddle(agent.client_id, 30)}</span>
              <CopyField value={agent.client_id} truncate={0}/>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn ghost sm" onClick={revokeAll} disabled={revoking}>
            {revoking ? 'Revoking…' : 'Revoke all tokens'}
          </button>
          <button className="btn danger sm" onClick={onDeactivate} disabled={!agent.active}>
            Deactivate
          </button>
          <div style={{flex:1}}/>
          {agent.active
            ? <span className="chip success" style={{height:18, fontSize:10}}><span className="dot success"/>active</span>
            : <span className="chip" style={{height:18, fontSize:10}}><span className="dot"/>inactive</span>}
        </div>
      </div>

      {/* Tabs */}
      <div className="row" style={{ borderBottom: '1px solid var(--hairline)', padding: '0 10px', gap: 2 }}>
        {[
          ['config', 'Config'],
          ['tokens', 'Tokens'],
          ['consents', 'Consents'],
          ['audit', 'Audit'],
          ['security', 'Security'],
          ['delegation', 'Delegation Policies'],
          ['delegations', 'Delegations'],
        ].map(([v, l]) => (
          <button key={v} onClick={() => setTab(v)}
            style={{
              padding: '10px 10px 8px', fontSize: 11.5,
              color: tab === v ? 'var(--fg)' : 'var(--fg-muted)',
              borderBottom: tab === v ? '2px solid var(--fg)' : '2px solid transparent',
              fontWeight: tab === v ? 500 : 400,
            }}>{l}</button>
        ))}
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {tab === 'config' && <AgentConfig agent={agent} onUpdate={onUpdate} onRotateSecret={() => setRotateModalOpen(true)}/>}
        {tab === 'tokens' && <AgentTokens agent={agent} tokensVersion={tokensVersion}/>}
        {tab === 'consents' && <AgentConsents agent={agent}/>}
        {tab === 'audit' && <AgentAudit agent={agent}/>}
        {tab === 'security' && <AgentSecurity agent={agent}/>}
        {tab === 'delegation' && <AgentDelegationPolicies agent={agent}/>}
        {tab === 'delegations' && <DelegationsTab agent={agent} setPage={setPage || (() => {})}/>}
      </div>
      {rotateModalOpen && (
        <RotateSecretModal
          agent={agent}
          onClose={() => setRotateModalOpen(false)}
        />
      )}
      <CLIFooter command={`shark agent show ${agent.client_id || agent.id}`}/>
    </aside>
  );
}

function AgentConfig({ agent, onUpdate, onRotateSecret }) {
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState(null);
  const [saveOk, setSaveOk] = React.useState(false);
  const [redirectInput, setRedirectInput] = React.useState('');
  const [scopeInput, setScopeInput] = React.useState('');
  const [name, setName] = React.useState(agent.name || '');
  const [description, setDescription] = React.useState(agent.description || '');

  const redirectUris = agent.redirect_uris || [];
  const grantTypes = agent.grant_types || [];
  const scopes = agent.scopes || [];

  React.useEffect(() => {
    setName(agent.name || '');
    setDescription(agent.description || '');
  }, [agent.id]);

  const save = async (updates) => {
    setSaving(true);
    setSaveError(null);
    setSaveOk(false);
    try {
      await onUpdate(updates);
      setSaveOk(true);
      setTimeout(() => setSaveOk(false), 2000);
    } catch (e) {
      setSaveError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const saveName = () => {
    if (name.trim() && name !== agent.name) save({ name: name.trim() });
  };
  const saveDescription = () => {
    if (description !== agent.description) save({ description: description });
  };

  const removeRedirect = (uri) => save({ redirect_uris: redirectUris.filter(u => u !== uri) });
  const addRedirect = () => {
    const val = redirectInput.trim();
    if (!val) return;
    save({ redirect_uris: [...redirectUris, val] });
    setRedirectInput('');
  };

  const toggleGrant = (g) => {
    const has = grantTypes.includes(g);
    save({ grant_types: has ? grantTypes.filter(x => x !== g) : [...grantTypes, g] });
  };

  const removeScope = (s) => save({ scopes: scopes.filter(x => x !== s) });
  const addScope = () => {
    const val = scopeInput.trim();
    if (!val) return;
    save({ scopes: [...scopes, val] });
    setScopeInput('');
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>
      {saveError && <div style={{color:'var(--danger)', fontSize: 11, padding: '6px 8px', background:'var(--surface-1)', borderRadius: 3}}>{saveError}</div>}
      {saveOk && <div style={{color:'var(--success)', fontSize: 11}}>Saved.</div>}

      <Section label="Name">
        <input
          value={name}
          onChange={e => setName(e.target.value)}
          onBlur={saveName}
          onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
          disabled={saving}
          style={{width:'100%', boxSizing:'border-box', fontSize:12, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
        />
      </Section>

      <Section label="Description">
        <textarea
          value={description}
          onChange={e => setDescription(e.target.value)}
          onBlur={saveDescription}
          disabled={saving}
          rows={2}
          style={{width:'100%', boxSizing:'border-box', fontSize:11.5, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none', resize:'vertical'}}
        />
      </Section>

      <Section label="Redirect URIs" count={redirectUris.length}>
        <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
          {redirectUris.length === 0 ? (
            <div className="faint" style={{padding: '8px 10px', fontSize: 11}}>None configured.</div>
          ) : redirectUris.map((uri, i) => (
            <div key={uri} className="row" style={{ padding: '7px 10px', gap: 8, borderBottom: i < redirectUris.length - 1 ? '1px solid var(--hairline)' : 0 }}>
              <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{uri}</span>
              {uri.includes('localhost') && <span className="chip" style={{height:15, fontSize:9}}>dev</span>}
              <button className="btn ghost icon sm" onClick={() => removeRedirect(uri)} disabled={saving}><Icon.X width={10} height={10}/></button>
            </div>
          ))}
        </div>
        <div className="row" style={{gap: 6, marginTop: 6}}>
          <input
            placeholder="https://…"
            value={redirectInput}
            onChange={e => setRedirectInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addRedirect()}
            style={{flex:1, fontSize:11, padding:'4px 7px', border:'1px solid var(--hairline)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
          />
          <button className="btn ghost sm" onClick={addRedirect} disabled={saving || !redirectInput.trim()}><Icon.Plus width={10} height={10}/> Add</button>
        </div>
      </Section>

      <Section label="Grant types" count={grantTypes.length}>
        <div style={{display:'flex', flexDirection:'column', gap: 4}}>
          {ALL_GRANTS.map(g => (
            <label key={g} className="row" style={{ gap: 8, fontSize: 11.5, padding: '5px 8px', background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 3, cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={grantTypes.includes(g)}
                onChange={() => toggleGrant(g)}
                disabled={saving}
              />
              <span className="mono" style={{fontSize: 11}}>{shortGrant(g)}</span>
              <span className="faint mono" style={{fontSize: 10, marginLeft: 'auto'}}>{g}</span>
            </label>
          ))}
        </div>
      </Section>

      <Section label="Scopes" count={scopes.length}>
        {scopes.length === 0 ? (
          <div className="faint" style={{fontSize: 11.5, padding: '6px 0'}}>No scopes configured.</div>
        ) : (
          <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
            {scopes.map(s => (
              <span key={s} className="chip mono" style={{height: 20, fontSize: 10.5, gap: 4}}>
                {s}
                <button className="btn ghost icon sm" onClick={() => removeScope(s)} disabled={saving} style={{padding: 0, marginLeft: 2}}>
                  <Icon.X width={8} height={8}/>
                </button>
              </span>
            ))}
          </div>
        )}
        <div className="row" style={{gap: 6, marginTop: 6}}>
          <input
            placeholder="e.g. read:files"
            value={scopeInput}
            onChange={e => setScopeInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addScope()}
            style={{flex:1, fontSize:11, padding:'4px 7px', border:'1px solid var(--hairline)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
          />
          <button className="btn ghost sm" onClick={addScope} disabled={saving || !scopeInput.trim()}><Icon.Plus width={10} height={10}/> Add</button>
        </div>
      </Section>

      <Section label="Client secret">
        <div className="row" style={{gap: 8}}>
          <span className="faint" style={{fontSize: 11.5, flex: 1}}>
            Rotate the client secret to invalidate the current one and reveal a new plaintext secret once.
          </span>
          <button className="btn ghost sm" onClick={onRotateSecret} style={{flexShrink: 0}}>
            Rotate secret…
          </button>
        </div>
      </Section>

      <Section label="Metadata">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">Agent ID</span>
          <span className="mono">{agent.id}</span>
          <span className="faint">Client type</span>
          <span className="mono">{agent.client_type || 'confidential'}</span>
          <span className="faint">Auth method</span>
          <span className="mono">{agent.auth_method || '—'}</span>
          <span className="faint">Token lifetime</span>
          <span className="mono">{agent.token_lifetime ? `${agent.token_lifetime}s` : '—'}</span>
          <span className="faint">Created</span>
          <span className="mono">{relativeTime(agent.created_at)}</span>
          <span className="faint">Last updated</span>
          <span className="mono">{relativeTime(agent.updated_at)}</span>
        </div>
      </Section>
    </div>
  );
}

function AgentTokens({ agent, tokensVersion }) {
  const { data, loading, refresh } = useAPI('/agents/' + agent.id + '/tokens?limit=50', [agent.id, tokensVersion]);
  const tokens = data?.data || [];
  const toast = useToast();
  const [revoking, setRevoking] = React.useState(false);

  const revokeAll = async () => {
    setRevoking(true);
    try {
      const r = await API.post('/agents/' + agent.id + '/tokens/revoke-all');
      toast.success(r?.message || `Revoked ${r?.count ?? 0} tokens`);
      refresh();
    } catch (e) {
      toast.error(`Failed to revoke: ${e.message}`);
    } finally {
      setRevoking(false);
    }
  };

  return (
    <div>
      <div className="row" style={{ padding: '10px 16px', borderBottom: '1px solid var(--hairline)', gap: 8 }}>
        <span style={{ fontSize: 12 }}>{tokens.length} active token{tokens.length !== 1 && 's'}</span>
        <div style={{ flex: 1 }}/>
        <button className="btn ghost sm" onClick={refresh} disabled={loading}>Refresh</button>
        <button className="btn danger sm" onClick={revokeAll} disabled={revoking || tokens.length === 0}>
          {revoking ? 'Revoking…' : 'Revoke all'}
        </button>
      </div>
      {loading ? (
        <div className="faint" style={{padding: 16, fontSize: 11}}>Loading…</div>
      ) : tokens.length === 0 ? (
        <div className="faint" style={{padding: 20, fontSize: 11.5, textAlign: 'center'}}>No active tokens for this agent.</div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11.5 }}>
          <thead>
            <tr>
              <th style={appThStyle}>Token ID</th>
              <th style={appThStyle}>Subject</th>
              <th style={appThStyle}>Scope</th>
              <th style={appThStyle}>Expires</th>
            </tr>
          </thead>
          <tbody>
            {tokens.map((t, i) => (
              <tr key={t.id || i}>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize: 10.5}}>{truncateMiddle(t.id || t.jti || '—', 16)}</span></td>
                <td style={appTdStyle}><span className="mono" style={{fontSize: 11}}>{t.subject || t.user_id || '—'}</span></td>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize: 10.5, maxWidth: 180, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t.scope || '—'}</span></td>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(t.expires_at || t.expiry)}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function AgentConsents({ agent }) {
  const clientId = agent.client_id;
  const { data, loading } = useAPI(
    clientId ? `/admin/oauth/consents?client_id=${encodeURIComponent(clientId)}&limit=50` : null,
    [clientId]
  );
  const consents = data?.data || [];

  return (
    <div style={{padding: 16}}>
      {loading ? (
        <div className="faint" style={{fontSize: 11}}>Loading consents…</div>
      ) : consents.length === 0 ? (
        <div className="faint" style={{fontSize: 11.5, textAlign: 'center', padding: 20}}>
          No users have authorized <b style={{color:'var(--fg)'}}>{agent.name}</b> yet.
        </div>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11.5 }}>
          <thead>
            <tr>
              <th style={appThStyle}>User ID</th>
              <th style={appThStyle}>Scope</th>
              <th style={appThStyle}>Granted</th>
            </tr>
          </thead>
          <tbody>
            {consents.map((c, i) => (
              <tr key={c.id || i}>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize:10.5}}>{c.user_id || '—'}</span></td>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize:10.5, maxWidth:180, display:'inline-block', overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap'}}>{c.scope || '—'}</span></td>
                <td style={appTdStyle}><span className="mono faint" style={{fontSize:10.5}}>{relativeTime(c.created_at || c.granted_at)}</span></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function SevDot({ sev }) {
  const c = sev === 'danger' ? 'var(--danger)' : sev === 'warn' ? 'var(--warn)' : 'var(--fg-dim)';
  return <span style={{width:6,height:6,borderRadius:1,background:c,flexShrink:0}}/>;
}

function AgentAudit({ agent }) {
  const { data, loading } = useAPI('/agents/' + agent.id + '/audit?limit=30', [agent.id]);
  const events = data?.data || [];
  return (
    <div style={{padding: 16}}>
      {loading ? (
        <div className="faint" style={{fontSize: 11}}>Loading…</div>
      ) : events.length === 0 ? (
        <div className="faint" style={{fontSize: 11.5, textAlign: 'center', padding: 20}}>No audit events for this agent yet.</div>
      ) : events.map((e, i) => (
        <div key={e.id || i} className="row" style={{padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11.5}}>
          <SevDot sev={e.severity}/>
          <span className="mono" style={{flex: 1}}>{e.action}</span>
          <span className="faint" style={{fontSize: 10.5}}>{e.actor || e.actor_id || '—'}</span>
          <span className="faint mono" style={{fontSize: 10}}>{relativeTime(e.created_at || e.t)}</span>
        </div>
      ))}
    </div>
  );
}

function DeactivateModal({ agent, onClose, onConfirm }) {
  const [working, setWorking] = React.useState(false);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
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
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Deactivate agent</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Deactivating <b style={{color:'var(--fg)'}}>{agent.name}</b> will prevent any new tokens from being issued and will revoke all active tokens.
        </p>
        <div style={{background:'var(--surface-1)', border:'1px solid var(--hairline)', padding: 10, borderRadius: 4, marginTop: 14, fontSize: 11.5}}>
          <div className="row" style={{gap: 8, marginBottom: 4}}>
            <span style={{color:'var(--warn)'}}>⚠</span>
            <b style={{color:'var(--fg)'}}>This cannot be undone from the UI.</b>
          </div>
          <div className="faint" style={{fontSize: 11}}>
            All currently valid tokens issued to this agent will be immediately invalidated.
          </div>
        </div>
        {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
        <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
          <button className="btn danger" onClick={go} disabled={working}>
            {working ? 'Deactivating…' : 'Deactivate agent'}
          </button>
        </div>
      </div>
    </div>
  );
}

function CreateAgentSlideOver({ onClose, onCreate }) {
  const [name, setName] = React.useState('');
  const [description, setDescription] = React.useState('');
  const [redirectUris, setRedirectUris] = React.useState('');
  const [grantTypes, setGrantTypes] = React.useState(['authorization_code', 'refresh_token']);
  const [scopes, setScopes] = React.useState('openid profile');
  const [creating, setCreating] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [created, setCreated] = React.useState(null); // holds response with secret

  React.useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape' && !created?.client_secret) onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose, created?.client_secret]);

  const handleBackdropClick = () => {
    if (!created?.client_secret) onClose();
  };

  const toggleGrant = (g) => {
    setGrantTypes(prev => prev.includes(g) ? prev.filter(x => x !== g) : [...prev, g]);
  };

  const handleCreate = async () => {
    if (!name.trim()) { setError('Name is required.'); return; }
    setCreating(true);
    setError(null);
    try {
      const payload = {
        name: name.trim(),
        description: description.trim(),
        redirect_uris: redirectUris.split('\n').map(s => s.trim()).filter(Boolean),
        grant_types: grantTypes,
        scopes: scopes.split(/\s+/).map(s => s.trim()).filter(Boolean),
      };
      const result = await onCreate(payload);
      setCreated(result);
    } catch (e) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={handleBackdropClick}>
      <div style={{
        position:'fixed', top: 0, right: 0, bottom: 0, width: 520,
        background:'var(--surface-0)', borderLeft:'1px solid var(--hairline-bright)',
        display:'flex', flexDirection:'column', boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        <div className="row" style={{padding: '14px 16px', borderBottom:'1px solid var(--hairline)'}}>
          <h2 style={{margin:0, fontSize: 14, fontWeight: 600, flex:1}}>
            {created ? 'Agent created' : 'New agent'}
          </h2>
          <button className="btn ghost icon sm" onClick={handleBackdropClick}><Icon.X width={11} height={11}/></button>
        </div>

        <div style={{flex:1, overflowY:'auto', padding: 20}}>
          {created ? (
            <div style={{display:'flex', flexDirection:'column', gap: 16}}>
              <div className="faint" style={{fontSize: 12}}>
                Agent <b style={{color:'var(--fg)'}}>{created.name || name}</b> has been created.
                {created.client_secret && ' Copy your client secret now — it won\'t be shown again.'}
              </div>
              <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
                <span className="faint">Client ID</span>
                <span className="mono" style={{wordBreak: 'break-all'}}>{created.client_id || created.id}</span>
              </div>
              {created.client_secret && (
                <div>
                  <div className="faint" style={{fontSize: 11, marginBottom: 6}}>Client Secret (shown once)</div>
                  <div style={{background:'var(--surface-3)', color:'var(--fg)', padding: 14, borderRadius: 0, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative', paddingRight: 80}}>
                    {created.client_secret}
                    <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'var(--fg)', borderColor:'var(--hairline-strong)'}}
                      onClick={() => navigator.clipboard.writeText(created.client_secret)}>
                      <Icon.Copy width={10} height={10}/> Copy
                    </button>
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div style={{display:'flex', flexDirection:'column', gap: 16}}>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Agent name <span style={{color:'var(--danger)'}}>*</span></label>
                <input
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="My Agent"
                  style={{width:'100%', boxSizing:'border-box', fontSize:12, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
                />
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Description</label>
                <textarea
                  value={description}
                  onChange={e => setDescription(e.target.value)}
                  placeholder="What does this agent do?"
                  rows={2}
                  style={{width:'100%', boxSizing:'border-box', fontSize:11.5, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none', resize:'vertical'}}
                />
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Redirect URIs <span className="faint">(one per line)</span></label>
                <textarea
                  value={redirectUris}
                  onChange={e => setRedirectUris(e.target.value)}
                  placeholder={"https://app.example.com/callback\nhttp://localhost:3000/callback"}
                  rows={3}
                  style={{width:'100%', boxSizing:'border-box', fontSize:11.5, fontFamily:'var(--font-mono)', padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none', resize:'vertical'}}
                />
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Grant types</label>
                <div style={{display:'flex', flexDirection:'column', gap: 4}}>
                  {ALL_GRANTS.map(g => (
                    <label key={g} className="row" style={{ gap: 8, fontSize: 11.5, padding: '5px 8px', background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 3, cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={grantTypes.includes(g)}
                        onChange={() => toggleGrant(g)}
                      />
                      <span className="mono" style={{fontSize: 11}}>{shortGrant(g)}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Scopes <span className="faint">(space-separated)</span></label>
                <textarea
                  value={scopes}
                  onChange={e => setScopes(e.target.value)}
                  placeholder="openid profile email"
                  rows={2}
                  style={{width:'100%', boxSizing:'border-box', fontSize:11.5, fontFamily:'var(--font-mono)', padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none', resize:'vertical'}}
                />
              </div>
              {error && <div style={{color:'var(--danger)', fontSize: 11}}>{error}</div>}
            </div>
          )}
        </div>

        <div className="row" style={{padding: 12, borderTop: '1px solid var(--hairline)', gap: 8, justifyContent: 'flex-end'}}>
          {created ? (
            <button className="btn primary" onClick={onClose}>Done</button>
          ) : (
            <>
              <button className="btn ghost" onClick={onClose}>Cancel</button>
              <button className="btn primary" onClick={handleCreate} disabled={creating || !name.trim()}>
                {creating ? 'Creating…' : 'Create agent'}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function RotateSecretModal({ agent, onClose }) {
  const toast = useToast();
  const [confirm, setConfirm] = React.useState('');
  const [working, setWorking] = React.useState(false);
  const [newSecret, setNewSecret] = React.useState(null);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape' && !newSecret) onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose, newSecret]);

  const go = async () => {
    setWorking(true);
    setError(null);
    try {
      const r = await API.post('/agents/' + agent.id + '/rotate-secret', {});
      setNewSecret(r.client_secret);
      toast.success('Secret rotated successfully');
    } catch (e) {
      setError(e.message);
    } finally {
      setWorking(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={!newSecret ? onClose : undefined}>
      <div style={{...modalCard, width: 500}} onClick={e => e.stopPropagation()}>
        {newSecret ? (
          <>
            <h2 style={{margin: 0, fontSize: 15, fontWeight: 600}}>New client secret</h2>
            <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
              Copy it now — it won't be shown again.
            </p>
            <div style={{background:'var(--surface-3)', color:'var(--fg)', padding: 14, borderRadius: 0, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative', paddingRight: 80, marginTop: 12}}>
              {newSecret}
              <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'var(--fg)', borderColor:'var(--hairline-strong)'}}
                onClick={() => navigator.clipboard.writeText(newSecret)}>
                <Icon.Copy width={10} height={10}/> Copy
              </button>
            </div>
            <div className="row" style={{marginTop: 16, justifyContent:'flex-end'}}>
              <button className="btn primary" onClick={onClose}>Done</button>
            </div>
          </>
        ) : (
          <>
            <h2 style={{margin: 0, fontSize: 15, fontWeight: 600}}>Rotate client secret</h2>
            <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
              This will invalidate the current secret for <b style={{color:'var(--fg)'}}>{agent.name}</b> immediately. Any services using the old secret will fail to authenticate.
            </p>
            <div style={{marginTop: 14}}>
              <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                Type <b style={{fontFamily:'var(--font-mono)'}}>{agent.name}</b> to confirm
              </label>
              <input
                value={confirm}
                onChange={e => setConfirm(e.target.value)}
                placeholder={agent.name}
                style={{width:'100%', boxSizing:'border-box', fontSize:12, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
              />
            </div>
            {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
            <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
              <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
              <button className="btn danger" onClick={go} disabled={working || confirm !== agent.name}>
                {working ? 'Rotating…' : 'Rotate secret'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// Truncate a string: first 8 chars + "..." + last 4 chars
function jktShort(s) {
  if (!s) return '—';
  if (s.length <= 12) return s;
  return s.slice(0, 8) + '...' + s.slice(-4);
}

function AgentSecurity({ agent }) {
  const toast = useToast();
  const [rotHistOpen, setRotHistOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  // Pull DPoP fields from the agent object (populated by GET /api/v1/agents/:id)
  const keyId   = agent.dpop_key_id   || agent.key_id   || null;
  const jkt     = agent.dpop_jkt      || agent.jkt      || null;

  // Pull rotation history from the agent's audit log (last 5 key-rotation events)
  const { data: auditData, loading: auditLoading } = useAPI(
    '/agents/' + agent.id + '/audit?limit=50',
    [agent.id]
  );
  const allEvents = auditData?.data || [];
  const rotations = allEvents
    .filter(e => e.action && (
      e.action.includes('key_rotation') ||
      e.action.includes('dpop') ||
      e.action.includes('rotate_key')
    ))
    .slice(0, 5);

  const copyJkt = () => {
    if (!jkt) return;
    navigator.clipboard.writeText(jkt).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
      toast.success('Thumbprint copied');
    }).catch(() => toast.error('Copy failed'));
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>

      {/* DPoP keypair section */}
      <Section label="DPoP keypair">
        <div style={{
          background: 'var(--surface-1)', border: '1px solid var(--hairline)',
          borderRadius: 3, padding: '10px 12px', display: 'flex', flexDirection: 'column', gap: 8,
        }}>
          {/* Algorithm + key_id row */}
          <div className="row" style={{ gap: 10, fontSize: 12 }}>
            <Icon.Key width={12} height={12} style={{ opacity: 0.6, flexShrink: 0 }} />
            <span className="mono" style={{ flex: 1 }}>
              ECDSA P-256
              {keyId ? (
                <span className="faint"> · key_id: {keyId}</span>
              ) : (
                <span className="faint"> · key_id: —</span>
              )}
            </span>
          </div>

          {/* Thumbprint (jkt) row */}
          <div className="row" style={{ gap: 10, fontSize: 12 }}>
            <Icon.Shield width={12} height={12} style={{ opacity: 0.6, flexShrink: 0 }} />
            <span className="faint" style={{ flexShrink: 0, fontSize: 11 }}>Thumbprint (jkt):</span>
            <span className="mono" style={{ flex: 1, fontSize: 11, wordBreak: 'break-all' }}>
              {jkt ? jktShort(jkt) : '—'}
            </span>
            {jkt && (
              <button
                className="btn ghost icon sm"
                onClick={copyJkt}
                title="Copy full thumbprint"
                style={{ flexShrink: 0 }}
              >
                {copied
                  ? <Icon.Check width={11} height={11} style={{ color: 'var(--success)' }} />
                  : <Icon.Copy width={11} height={11} />}
              </button>
            )}
          </div>

          {/* Empty state when no DPoP data */}
          {!keyId && !jkt && (
            <div className="faint" style={{ fontSize: 11, marginTop: 2 }}>
              No DPoP keypair registered for this agent.
            </div>
          )}
        </div>
      </Section>

      {/* Rotation history section */}
      <Section label="Rotation history">
        <div style={{
          border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)', overflow: 'hidden',
        }}>
          {/* Collapsible header */}
          <button
            onClick={() => setRotHistOpen(o => !o)}
            style={{
              width: '100%', display: 'flex', alignItems: 'center', gap: 8,
              padding: '8px 12px', background: 'transparent', border: 0, cursor: 'pointer',
              fontSize: 11.5, color: 'var(--fg-muted)',
            }}
          >
            {rotHistOpen
              ? <Icon.ChevronDown width={11} height={11} />
              : <Icon.ChevronRight width={11} height={11} />}
            <span style={{ flex: 1, textAlign: 'left' }}>Last 5 key rotations</span>
            {!auditLoading && (
              <span className="mono faint" style={{ fontSize: 10 }}>{rotations.length} event{rotations.length !== 1 && 's'}</span>
            )}
          </button>

          {rotHistOpen && (
            <div style={{ borderTop: '1px solid var(--hairline)' }}>
              {auditLoading ? (
                <div className="faint" style={{ padding: '10px 12px', fontSize: 11 }}>Loading…</div>
              ) : rotations.length === 0 ? (
                <div className="faint" style={{ padding: '10px 12px', fontSize: 11 }}>
                  No key rotation events found in audit log.
                </div>
              ) : rotations.map((e, i) => (
                <div
                  key={e.id || i}
                  className="row"
                  style={{
                    padding: '7px 12px', gap: 8, fontSize: 11,
                    borderBottom: i < rotations.length - 1 ? '1px solid var(--hairline)' : 0,
                  }}
                >
                  <SevDot sev={e.severity} />
                  <span className="mono" style={{ flex: 1 }}>{e.action}</span>
                  <span className="faint" style={{ fontSize: 10.5 }}>{e.actor || e.actor_id || '—'}</span>
                  <span className="faint mono" style={{ fontSize: 10 }}>{relativeTime(e.created_at || e.t)}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </Section>
    </div>
  );
}

function AgentDelegationPolicies({ agent }) {
  const toast = useToast();
  const { data: allAgentsRaw, loading: agentsLoading } = useAPI('/agents?limit=200');
  const { data: policiesRaw, loading: policiesLoading, refresh: refreshPolicies } = useAPI(
    '/agents/' + agent.id + '/policies',
    [agent.id]
  );

  const allAgents = (allAgentsRaw?.data || []).filter(a => a.id !== agent.id);
  const savedPolicies = policiesRaw?.data || policiesRaw?.policies || [];

  // editMode: false = read-only summary, true = checkbox grid
  const [editMode, setEditMode] = React.useState(false);
  const [saving, setSaving] = React.useState(false);

  // Local draft: map of agentId -> { enabled: bool, scope: string }
  const [draft, setDraft] = React.useState({});

  // When entering edit mode, seed draft from saved policies
  React.useEffect(() => {
    if (!editMode) return;
    const initial = {};
    for (const a of allAgents) {
      const saved = savedPolicies.find(p => p.delegate_to_id === a.id || p.agent_id === a.id);
      initial[a.id] = {
        enabled: !!saved,
        scope: saved?.scope || '',
      };
    }
    setDraft(initial);
  }, [editMode, agent.id, policiesLoading]);

  const delegatedNames = savedPolicies
    .map(p => {
      const match = allAgents.find(a => a.id === (p.delegate_to_id || p.agent_id));
      return match ? match.name : p.delegate_to_id || p.agent_id || '?';
    })
    .filter(Boolean);

  const toggleAgent = (id) => {
    setDraft(d => ({ ...d, [id]: { ...d[id], enabled: !d[id]?.enabled } }));
  };

  const setScopeFor = (id, scope) => {
    setDraft(d => ({ ...d, [id]: { ...d[id], scope } }));
  };

  const save = async () => {
    setSaving(true);
    try {
      const policies = Object.entries(draft)
        .filter(([, v]) => v.enabled)
        .map(([id, v]) => ({ delegate_to_id: id, scope: v.scope || '' }));
      await API.post('/agents/' + agent.id + '/policies', { policies });
      toast.success('Delegation policies saved');
      refreshPolicies();
      setEditMode(false);
    } catch (e) {
      toast.error('Save failed: ' + e.message);
    } finally {
      setSaving(false);
    }
  };

  const loading = agentsLoading || policiesLoading;

  if (loading) {
    return <div className="faint" style={{ padding: 20, fontSize: 11 }}>Loading…</div>;
  }

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>

      {/* Read-only summary / empty state */}
      {!editMode && (
        <Section label="Delegation Policies">
          {savedPolicies.length === 0 ? (
            <div style={{
              background: 'var(--surface-1)', border: '1px solid var(--hairline)',
              borderRadius: 3, padding: '14px 16px', display: 'flex', flexDirection: 'column',
              gap: 10, alignItems: 'flex-start',
            }}>
              <div style={{ fontSize: 12, color: 'var(--fg-muted)' }}>
                No delegation policies set. This agent cannot delegate to any other agent.
              </div>
              <button className="btn ghost sm" onClick={() => setEditMode(true)}>
                Configure
              </button>
            </div>
          ) : (
            <div style={{
              background: 'var(--surface-1)', border: '1px solid var(--hairline)',
              borderRadius: 3, padding: '10px 12px', fontSize: 12,
            }}>
              <div className="row" style={{ gap: 6, flexWrap: 'wrap', alignItems: 'baseline' }}>
                <span style={{ color: 'var(--fg-muted)', flexShrink: 0 }}>This agent can delegate to:</span>
                {delegatedNames.map(name => (
                  <span key={name} className="chip mono" style={{ height: 20, fontSize: 10.5 }}>{name}</span>
                ))}
                <button
                  className="btn ghost sm"
                  onClick={() => setEditMode(true)}
                  style={{ marginLeft: 4, fontSize: 11 }}
                >
                  edit
                </button>
              </div>
            </div>
          )}
        </Section>
      )}

      {/* Edit mode: checkbox grid */}
      {editMode && (
        <Section label="Delegation Policies">
          {allAgents.length === 0 ? (
            <div className="faint" style={{ fontSize: 11.5, padding: '8px 0' }}>
              No other agents in this workspace to delegate to.
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
              {allAgents.map(a => {
                const row = draft[a.id] || { enabled: false, scope: '' };
                return (
                  <div
                    key={a.id}
                    style={{
                      display: 'grid', gridTemplateColumns: '20px 1fr auto',
                      gap: 8, alignItems: 'center',
                      padding: '6px 10px', background: 'var(--surface-1)',
                      border: '1px solid var(--hairline)', borderRadius: 3,
                    }}
                  >
                    <input
                      type="checkbox"
                      checked={row.enabled}
                      onChange={() => toggleAgent(a.id)}
                      disabled={saving}
                    />
                    <span style={{ fontSize: 12, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {a.name}
                      {a.description && (
                        <span className="faint" style={{ fontWeight: 400, marginLeft: 6, fontSize: 11 }}>
                          {a.description}
                        </span>
                      )}
                    </span>
                    <input
                      placeholder="scope (optional)"
                      value={row.scope}
                      onChange={e => setScopeFor(a.id, e.target.value)}
                      disabled={!row.enabled || saving}
                      style={{
                        fontSize: 11, padding: '3px 7px', width: 160,
                        border: '1px solid var(--hairline)', borderRadius: 3,
                        background: row.enabled ? 'var(--surface-0)' : 'var(--surface-2)',
                        color: 'var(--fg)', outline: 'none', opacity: row.enabled ? 1 : 0.5,
                        fontFamily: 'var(--font-mono)',
                      }}
                    />
                  </div>
                );
              })}
            </div>
          )}

          <div className="row" style={{ gap: 8, marginTop: 12, justifyContent: 'flex-end' }}>
            <button className="btn ghost sm" onClick={() => setEditMode(false)} disabled={saving}>
              Cancel
            </button>
            <button className="btn primary sm" onClick={save} disabled={saving}>
              {saving ? 'Saving…' : 'Save policies'}
            </button>
          </div>
        </Section>
      )}
    </div>
  );
}

function Section({ label, count, children }) {
  return (
    <div>
      <div className="row" style={{marginBottom: 6, gap: 6}}>
        <h4 style={{...sectionLabelStyle, margin: 0}}>{label}</h4>
        {count != null && <span className="faint mono" style={{fontSize:10}}>· {count}</span>}
      </div>
      {children}
    </div>
  );
}

// ── Delegations Tab ──────────────────────────────────────────────────────────

function groupByCounterpart(items, agentId, direction) {
  const map = {};
  for (const ev of items) {
    const meta = ev.metadata || ev.meta || {};
    const chain = ev.act_chain || meta.act_chain || [];
    let counterpartId = null;
    let counterpartName = null;
    if (direction === 'outbound') {
      // actor is this agent; target is the subject
      counterpartId = meta.target_agent_id || meta.subject_agent_id || meta.target_sub || ev.subject_id || null;
      counterpartName = meta.target_agent_name || meta.subject_name || counterpartId;
    } else {
      // find this agent in act_chain; the issuer of the chain is the counterpart
      const node = chain.find(n => n.sub === agentId || n.agent_id === agentId);
      if (node) {
        counterpartId = ev.actor_id || ev.actor?.id || ev.subject_id || null;
        counterpartName = ev.actor?.name || counterpartId;
      } else {
        counterpartId = ev.actor_id || ev.actor?.id || null;
        counterpartName = ev.actor?.name || counterpartId;
      }
    }
    if (!counterpartId) continue;
    if (!map[counterpartId]) {
      map[counterpartId] = {
        id: counterpartId,
        name: counterpartName || counterpartId,
        jkt: (ev.jkt || meta.jkt || counterpartId || '????'),
        lastScope: meta.scope || meta.scopes || ev.scope || '',
        lastTs: ev.created_at || ev.timestamp || '',
        count: 0,
      };
    }
    map[counterpartId].count++;
    // keep latest timestamp
    const ts = ev.created_at || ev.timestamp || '';
    if (ts > map[counterpartId].lastTs) {
      map[counterpartId].lastTs = ts;
      map[counterpartId].lastScope = meta.scope || meta.scopes || ev.scope || '';
    }
  }
  return Object.values(map).slice(0, 8);
}

function fmtTs(ts) {
  if (!ts) return '';
  try {
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch { return ts.slice(11, 16) || ''; }
}

function fmtScope(s) {
  if (!s) return '';
  const str = Array.isArray(s) ? s.join(' ') : String(s);
  return str.length > 18 ? str.slice(0, 16) + '…' : str;
}

function DelegationsTab({ agent, setPage }) {
  const [inbound, setInbound] = React.useState(null);   // null = loading
  const [outbound, setOutbound] = React.useState(null);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    if (!agent?.id) return;
    let cancelled = false;
    const headers = { Authorization: `Bearer ${localStorage.getItem('shark_admin_key')}` };

    const load = async () => {
      try {
        const [outRes, inRes] = await Promise.all([
          fetch(`/api/v1/audit-logs?action=oauth.token.exchanged&actor_id=${agent.id}&limit=100`, { headers }),
          fetch(`/api/v1/audit-logs?action=oauth.token.exchanged&limit=200`, { headers }),
        ]);
        const [outData, inData] = await Promise.all([outRes.json(), inRes.json()]);
        if (cancelled) return;

        const outItems = outData?.data || outData?.items || [];
        const inItems = (inData?.data || inData?.items || []).filter(ev => {
          let parsedMeta: Record<string, any> = {};
          if (typeof ev?.metadata === 'string') {
            try { parsedMeta = JSON.parse(ev.metadata) || {}; } catch { parsedMeta = {}; }
          } else if (ev?.metadata && typeof ev.metadata === 'object') {
            parsedMeta = ev.metadata;
          }
          const chain = ev.act_chain || parsedMeta.act_chain || ev?.meta?.act_chain || [];
          return chain.some((n: any) => n.sub === agent.id || n.agent_id === agent.id);
        });

        setOutbound(groupByCounterpart(outItems, agent.id, 'outbound'));
        setInbound(groupByCounterpart(inItems, agent.id, 'inbound'));
      } catch (e) {
        if (!cancelled) setError(String(e));
      }
    };
    load();
    return () => { cancelled = true; };
  }, [agent?.id]);

  // ── Loading / error ───────────────────────────────────────────────────────
  if (inbound === null || outbound === null) {
    return (
      <div style={{ padding: 24, textAlign: 'center', fontFamily: 'monospace', fontSize: 12, color: 'var(--fg-dim)' }}>
        {error ? `Error: ${error}` : 'Loading…'}
      </div>
    );
  }

  if (inbound.length === 0 && outbound.length === 0) {
    return (
      <div style={{ padding: 24, textAlign: 'center', fontFamily: 'monospace', fontSize: 12, color: 'var(--fg-dim)', lineHeight: 1.7 }}>
        No delegations recorded.<br/>
        Run the demo or use tools/agent_demo_tester.py<br/>
        to populate the chain.
      </div>
    );
  }

  // ── Build react-flow ego graph ────────────────────────────────────────────
  const centerNode = {
    id: agent.id,
    label: agent.name || agent.client_id || agent.id,
    isUser: false,
    isCenter: true,
    layer: 1,
    slotInLayer: 0,
    jkt: agent.jkt || agent.client_id,
  };

  const inboundNodes = inbound.map((item, i) => ({
    id: item.id,
    label: item.name || item.id,
    isUser: false,
    layer: 0,
    slotInLayer: i,
    jkt: item.jkt,
  }));

  const outboundNodes = outbound.map((item, i) => ({
    id: item.id,
    label: item.name || item.id,
    isUser: false,
    layer: 2,
    slotInLayer: i,
    jkt: item.jkt,
  }));

  const inboundEdges = inbound.map(item => ({
    id: `in-${item.id}->${agent.id}`,
    from: item.id,
    to: agent.id,
    label: [fmtTs(item.lastTs), fmtScope(item.lastScope)].filter(Boolean).join(' ') || undefined,
    eventId: item.lastEventId,
  }));

  const outboundEdges = outbound.map(item => ({
    id: `out-${agent.id}->${item.id}`,
    from: agent.id,
    to: item.id,
    label: [fmtTs(item.lastTs), fmtScope(item.lastScope)].filter(Boolean).join(' ') || undefined,
    eventId: item.lastEventId,
  }));

  const { rfNodes, rfEdges } = toEgoLayout(
    centerNode,
    inboundNodes,
    outboundNodes,
    inboundEdges,
    outboundEdges,
  );

  return (
    <div style={{ padding: '12px 0' }}>
      <div style={{ padding: '0 14px 6px', fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500 }}>
        Delegations
      </div>
      <DelegationCanvasWithProvider
        rfNodes={rfNodes}
        rfEdges={rfEdges}
        height={500}
        onNodeClick={(nodeId, nodeData) => {
          if (nodeId !== agent.id) setPage('agents', { agentId: nodeId });
        }}
        fitView
      />
    </div>
  );
}
