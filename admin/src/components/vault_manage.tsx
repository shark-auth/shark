// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { useTabParam } from './useURLParams'
import { usePageActions } from './useKeyboardShortcuts'

// Token Vault — manage third-party OAuth providers (Google, Slack, GitHub, …)
// whose tokens are refreshed on behalf of users so agents can request them.
// Split grid: providers table on the left, slide-over detail on the right.
// Mirrors agents_manage.tsx / applications.tsx conventions.

const thStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const tdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };
const inputStyle = { width:'100%', boxSizing:'border-box', fontSize:12, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none' };

export function Vault() {
  const [selected, setSelected] = React.useState(null);
  const [tab, setTab] = useTabParam('config');
  const [filter, setFilter] = React.useState('all'); // all | active | inactive
  const [query, setQuery] = React.useState('');
  const [createOpen, setCreateOpen] = React.useState(false);
  const [deleteModal, setDeleteModal] = React.useState(null);
  const toast = useToast();

  const { data: listRaw, loading, refresh } = useAPI('/vault/providers');
  const providers = listRaw?.data || [];

  React.useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    if (p.get('new') === '1') {
      setCreateOpen(true);
      p.delete('new');
      const s = p.toString();
      window.history.replaceState(null, '', window.location.pathname + (s ? '?' + s : ''));
    }
  }, []);
  usePageActions({ onNew: () => setCreateOpen(true), onRefresh: refresh });

  const filtered = providers.filter(p => {
    if (filter === 'active' && !p.active) return false;
    if (filter === 'inactive' && p.active) return false;
    if (query) {
      const q = query.toLowerCase();
      const name = (p.name || '').toLowerCase();
      const dn = (p.display_name || '').toLowerCase();
      const cid = (p.client_id || '').toLowerCase();
      if (!name.includes(q) && !dn.includes(q) && !cid.includes(q)) return false;
    }
    return true;
  });

  const activeCount = providers.filter(p => p.active).length;

  // Keep selected in sync after refresh
  React.useEffect(() => {
    if (selected) {
      const fresh = providers.find(p => p.id === selected.id);
      if (fresh) setSelected(fresh);
    }
  }, [providers, selected?.id]);

  const noProviders = !loading && providers.length === 0;

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 560px' : '1fr', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
        {/* Header */}
        <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ gap: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Vault</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                Third-party OAuth providers · {providers.length} configured · {activeCount} active
              </p>
            </div>
            <button className="btn primary" onClick={() => setCreateOpen(true)}>
              <Icon.Plus width={11} height={11}/> Add provider
            </button>
          </div>
        </div>

        {noProviders ? (
          <EmptyProviders onAdd={() => setCreateOpen(true)}/>
        ) : (
          <>
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
              <div style={{flex:1}}/>
              <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {providers.length}</span>
            </div>

            {/* Table */}
            <div style={{ flex: 1, overflow: 'auto' }}>
              {loading ? (
                <div className="faint" style={{padding: 20, fontSize: 12}}>Loading providers…</div>
              ) : filtered.length === 0 ? (
                <div className="faint" style={{padding: 40, fontSize: 12, textAlign: 'center'}}>
                  No providers match your filters.
                </div>
              ) : (
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                  <thead>
                    <tr>
                      <th style={{...thStyle, width: 28}}/>
                      <th style={thStyle}>Display Name</th>
                      <th style={thStyle}>Name</th>
                      <th style={thStyle}>Client ID</th>
                      <th style={thStyle}>Scopes</th>
                      <th style={thStyle}>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filtered.map(p => (
                      <tr key={p.id}
                        onClick={() => { setSelected(p); setTab('config'); }}
                        style={{
                          cursor:'pointer',
                          background: selected?.id === p.id ? 'var(--surface-2)' : 'transparent',
                        }}>
                        <td style={tdStyle}>
                          <ProviderIcon provider={p} size={22}/>
                        </td>
                        <td style={tdStyle}>
                          <div style={{fontWeight: 500}}>{p.display_name || p.name}</div>
                        </td>
                        <td style={tdStyle}>
                          <span className="mono faint" style={{fontSize: 10.5}}>{p.name}</span>
                        </td>
                        <td style={tdStyle}>
                          <span className="mono faint" style={{fontSize: 10.5}}>
                            {truncateMiddle(p.client_id, 24)}
                          </span>
                        </td>
                        <td style={tdStyle}>
                          <span className="mono faint" style={{fontSize: 10.5}}>
                            {(p.scopes || []).length} scope{(p.scopes || []).length !== 1 && 's'}
                          </span>
                        </td>
                        <td style={tdStyle}>
                          {p.active
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
              <span className="mono faint">shark vault provider list</span>
              <div style={{flex:1}}/>
              <span className="faint mono">GET /api/v1/vault/providers</span>
            </div>
          </>
        )}
      </div>

      {selected && (
        <ProviderDetail
          provider={selected}
          tab={tab} setTab={setTab}
          onClose={() => setSelected(null)}
          onDelete={() => setDeleteModal(selected)}
          onUpdate={async (updates) => {
            const updated = await API.patch('/vault/providers/' + selected.id, updates);
            refresh();
            return updated;
          }}
        />
      )}

      {createOpen && (
        <CreateProviderWizard
          onClose={() => setCreateOpen(false)}
          onCreate={async (payload) => {
            const result = await API.post('/vault/providers', payload);
            refresh();
            return result;
          }}
          onCreated={(p) => { setSelected(p); setTab('config'); }}
        />
      )}

      {deleteModal && (
        <DeleteProviderModal
          provider={deleteModal}
          onClose={() => setDeleteModal(null)}
          onConfirm={async () => {
            try {
              await API.del('/vault/providers/' + deleteModal.id);
              toast.success(`Provider "${deleteModal.name || deleteModal.id}" deleted`);
              setDeleteModal(null);
              setSelected(null);
              refresh();
            } catch (e) {
              toast.error(e?.message || 'Delete failed');
            }
          }}
        />
      )}
    </div>
  );
}

// --- Empty state -----------------------------------------------------------

function EmptyProviders({ onAdd }) {
  return (
    <div style={{
      flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: 40, textAlign: 'center',
    }}>
      <div style={{ maxWidth: 440 }}>
        <div style={{
          width: 56, height: 56, borderRadius: 10,
          background: 'var(--surface-2)', border: '1px solid var(--hairline-strong)',
          display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
          color: 'var(--fg-muted)', marginBottom: 18,
        }}>
          <Icon.Vault width={26} height={26}/>
        </div>
        <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--fg)' }}>No providers configured</div>
        <div style={{ marginTop: 8, fontSize: 12.5, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
          Add Google, Slack, GitHub, and more — agents request tokens through Shark.
        </div>
        <button className="btn primary" onClick={onAdd} style={{ marginTop: 20 }}>
          <Icon.Plus width={11} height={11}/> Add provider
        </button>
      </div>
    </div>
  );
}

// --- Provider icon helper --------------------------------------------------

function ProviderIcon({ provider, size = 22 }) {
  const [failed, setFailed] = React.useState(false);
  const showImg = provider?.icon_url && !failed;
  return (
    <div style={{
      width: size, height: size, borderRadius: 4, background: 'var(--surface-3)',
      display:'flex', alignItems:'center', justifyContent:'center',
      fontSize: Math.max(9, Math.floor(size / 2.2)), fontWeight: 600,
      color: 'var(--fg)', border:'1px solid var(--hairline-strong)',
      overflow: 'hidden',
    }}>
      {showImg
        ? <img src={provider.icon_url} onError={() => setFailed(true)} alt=""
            style={{ width: size - 4, height: size - 4, objectFit: 'contain' }}/>
        : ((provider?.display_name || provider?.name || '?')[0] || '?').toUpperCase()}
    </div>
  );
}

// --- Provider detail slide-over --------------------------------------------

function ProviderDetail({ provider, tab, setTab, onClose, onDelete, onUpdate }) {
  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)', background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 10 }}>
          <ProviderIcon provider={provider} size={32}/>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight: 500, fontSize: 14}}>{provider.display_name || provider.name}</div>
            <div className="row" style={{gap: 6, fontSize: 10.5, marginTop: 2}}>
              <span className="mono faint">{provider.name}</span>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn danger sm" onClick={onDelete}>
            Delete provider
          </button>
          <div style={{flex:1}}/>
          {provider.active
            ? <span className="chip success" style={{height:18, fontSize:10}}><span className="dot success"/>active</span>
            : <span className="chip" style={{height:18, fontSize:10}}><span className="dot"/>inactive</span>}
        </div>
      </div>

      {/* Tabs */}
      <div className="row" style={{ borderBottom: '1px solid var(--hairline)', padding: '0 10px', gap: 2 }}>
        {[
          ['config', 'Config'],
          ['connections', 'Connections'],
          ['audit', 'Audit'],
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
        {tab === 'config' && <ProviderConfig provider={provider} onUpdate={onUpdate}/>}
        {tab === 'connections' && <ProviderConnections provider={provider}/>}
        {tab === 'audit' && <ProviderAudit provider={provider}/>}
      </div>
      <CLIFooter command={`shark vault provider show ${provider.name}`}/>
    </aside>
  );
}

function ProviderConfig({ provider, onUpdate }) {
  const toast = useToast();
  const [saving, setSaving] = React.useState(false);
  const [displayName, setDisplayName] = React.useState(provider.display_name || '');
  const [iconURL, setIconURL] = React.useState(provider.icon_url || '');
  const [authURL, setAuthURL] = React.useState(provider.auth_url || '');
  const [tokenURL, setTokenURL] = React.useState(provider.token_url || '');
  const [clientID, setClientID] = React.useState(provider.client_id || '');
  const [scopeInput, setScopeInput] = React.useState('');
  const [rotateOpen, setRotateOpen] = React.useState(false);

  const scopes = provider.scopes || [];

  React.useEffect(() => {
    setDisplayName(provider.display_name || '');
    setIconURL(provider.icon_url || '');
    setAuthURL(provider.auth_url || '');
    setTokenURL(provider.token_url || '');
    setClientID(provider.client_id || '');
  }, [provider.id]);

  const save = async (updates, successMsg) => {
    setSaving(true);
    try {
      await onUpdate(updates);
      if (successMsg) toast.success(successMsg);
    } catch (e) {
      toast.error(`Save failed: ${e.message}`);
    } finally {
      setSaving(false);
    }
  };

  const saveDisplayName = () => {
    if (displayName !== provider.display_name) save({ display_name: displayName }, 'Display name updated');
  };
  const saveIconURL = () => {
    if (iconURL !== (provider.icon_url || '')) save({ icon_url: iconURL }, 'Icon updated');
  };
  const saveAuthURL = () => {
    if (authURL.trim() && authURL !== provider.auth_url) save({ auth_url: authURL.trim() }, 'Auth URL updated');
  };
  const saveTokenURL = () => {
    if (tokenURL.trim() && tokenURL !== provider.token_url) save({ token_url: tokenURL.trim() }, 'Token URL updated');
  };
  const saveClientID = () => {
    if (clientID.trim() && clientID !== provider.client_id) save({ client_id: clientID.trim() }, 'Client ID updated');
  };
  const toggleActive = () => save({ active: !provider.active }, provider.active ? 'Provider deactivated' : 'Provider activated');

  const removeScope = (s) => save({ scopes: scopes.filter(x => x !== s) }, 'Scopes updated');
  const addScope = () => {
    const val = scopeInput.trim();
    if (!val) return;
    if (scopes.includes(val)) { setScopeInput(''); return; }
    save({ scopes: [...scopes, val] }, 'Scopes updated');
    setScopeInput('');
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>
      <Section label="Display name">
        <input
          value={displayName}
          onChange={e => setDisplayName(e.target.value)}
          onBlur={saveDisplayName}
          onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
          disabled={saving}
          style={inputStyle}
        />
      </Section>

      <Section label="Name">
        <div className="row" style={{ gap: 6 }}>
          <span className="mono" style={{ fontSize: 12, padding: '6px 9px', background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 3, flex: 1 }}>
            {provider.name}
          </span>
          <span className="faint" style={{ fontSize: 10.5 }}>read-only</span>
        </div>
      </Section>

      <Section label="Client ID">
        <div className="row" style={{ gap: 6 }}>
          <input
            value={clientID}
            onChange={e => setClientID(e.target.value)}
            onBlur={saveClientID}
            onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
            disabled={saving}
            className="mono"
            style={{...inputStyle, fontSize: 11.5, flex: 1}}
          />
          <CopyField value={provider.client_id} truncate={0}/>
        </div>
      </Section>

      <Section label="Client secret">
        <div className="row" style={{ gap: 8 }}>
          <span className="mono faint" style={{ fontSize: 11.5, padding: '6px 9px', background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 3, flex: 1 }}>
            •••••••••••••••••••• <span className="faint" style={{ fontSize: 10, marginLeft: 6 }}>encrypted at rest</span>
          </span>
          <button className="btn ghost sm" onClick={() => setRotateOpen(true)} disabled={saving}>
            <Icon.Key width={10} height={10}/> Rotate
          </button>
        </div>
      </Section>

      <Section label="Auth URL">
        <input
          value={authURL}
          onChange={e => setAuthURL(e.target.value)}
          onBlur={saveAuthURL}
          onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
          disabled={saving}
          placeholder="https://provider.example.com/oauth/authorize"
          className="mono"
          style={{...inputStyle, fontSize: 11}}
        />
      </Section>

      <Section label="Token URL">
        <input
          value={tokenURL}
          onChange={e => setTokenURL(e.target.value)}
          onBlur={saveTokenURL}
          onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
          disabled={saving}
          placeholder="https://provider.example.com/oauth/token"
          className="mono"
          style={{...inputStyle, fontSize: 11}}
        />
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
            placeholder="e.g. https://www.googleapis.com/auth/calendar"
            value={scopeInput}
            onChange={e => setScopeInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addScope()}
            style={{flex:1, fontSize:11, padding:'4px 7px', border:'1px solid var(--hairline)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
          />
          <button className="btn ghost sm" onClick={addScope} disabled={saving || !scopeInput.trim()}><Icon.Plus width={10} height={10}/> Add</button>
        </div>
      </Section>

      <Section label="Icon URL">
        <input
          value={iconURL}
          onChange={e => setIconURL(e.target.value)}
          onBlur={saveIconURL}
          onKeyDown={e => e.key === 'Enter' && e.currentTarget.blur()}
          placeholder="https://example.com/icon.png"
          disabled={saving}
          style={inputStyle}
        />
      </Section>

      <Section label="Status">
        <div className="row" style={{ gap: 8 }}>
          <button
            className={provider.active ? 'btn ghost sm' : 'btn primary sm'}
            onClick={toggleActive}
            disabled={saving}
          >
            {provider.active ? 'Deactivate' : 'Activate'}
          </button>
          <span className="faint" style={{ fontSize: 11 }}>
            Inactive providers can't start new connections.
          </span>
        </div>
      </Section>

      <Section label="Metadata">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">Provider ID</span>
          <span className="mono">{provider.id}</span>
          <span className="faint">Created</span>
          <span className="mono">{relativeTime(provider.created_at)}</span>
          <span className="faint">Last updated</span>
          <span className="mono">{relativeTime(provider.updated_at)}</span>
        </div>
      </Section>

      {rotateOpen && (
        <RotateSecretModal
          provider={provider}
          onClose={() => setRotateOpen(false)}
          onConfirm={async (newSecret) => {
            await onUpdate({ client_secret: newSecret });
            toast.success('Client secret rotated');
            setRotateOpen(false);
          }}
        />
      )}
    </div>
  );
}

function ProviderConnections({ provider }) {
  const toast = useToast();
  // Pass provider.id as dep so the hook re-fetches when the selected provider
  // changes, and use the ?provider_id filter to avoid a full-table scan.
  const { data, loading, error, refresh } = useAPI(
    '/admin/vault/connections?provider_id=' + encodeURIComponent(provider.id),
    [provider.id],
  );
  const conns = data?.data || [];
  const [testModal, setTestModal] = React.useState(null); // {userId, connId} | null

  const handleDisconnect = async (id) => {
    if (!confirm('Disconnect this user from ' + (provider.display_name || provider.name) + '? They will need to re-authorise.')) return;
    try {
      await API.del('/admin/vault/connections/' + id);
      toast.success('Connection disconnected');
      refresh();
    } catch (e) {
      toast.error(e.message || 'Disconnect failed');
    }
  };

  const statusChip = (c) => {
    if (c.needs_reauth) return <span className="chip warn" style={{ height: 16, fontSize: 10 }}>needs_reauth</span>;
    if (c.expires_at) {
      const ms = new Date(c.expires_at).getTime() - Date.now();
      if (ms < 0) return <span className="chip danger" style={{ height: 16, fontSize: 10 }}>expired</span>;
      if (ms < 60 * 60 * 1000) return <span className="chip warn" style={{ height: 16, fontSize: 10 }}>expiring</span>;
    }
    return <span className="chip success" style={{ height: 16, fontSize: 10 }}>active</span>;
  };

  const handleConnect = () => {
    // Open the OAuth connect flow in a new tab. When the callback completes
    // the dashboard will re-fetch on next refresh (or user clicks Refresh).
    const win = window.open('/api/v1/vault/connect/' + encodeURIComponent(provider.name), '_blank', 'noopener,noreferrer');
    // Poll after the new tab closes to auto-refresh the connection list.
    if (win) {
      const poll = setInterval(() => {
        if (win.closed) { clearInterval(poll); refresh(); }
      }, 800);
    }
  };

  return (
    <div style={{ padding: 16 }}>
      {testModal && (
        <VaultTestTokenModal
          provider={provider}
          userId={testModal.userId}
          onClose={() => setTestModal(null)}
        />
      )}
      <div className="row" style={{ marginBottom: 10, gap: 8 }}>
        <div style={{ fontSize: 12, fontWeight: 500 }}>
          {conns.length} connection{conns.length !== 1 ? 's' : ''} to {provider.display_name || provider.name}
        </div>
        <div style={{ flex: 1 }}/>
        <button className="btn ghost sm" onClick={handleConnect} disabled={!provider.active} title={!provider.active ? 'Provider is inactive' : 'Start OAuth flow'}>
          <Icon.Plus width={11} height={11}/>Connect
        </button>
        <button className="btn ghost sm" onClick={refresh} disabled={loading}>
          <Icon.Refresh width={11} height={11}/>Refresh
        </button>
      </div>

      {loading ? (
        <div className="faint" style={{ padding: 24, textAlign: 'center', fontSize: 12 }}>Loading…</div>
      ) : error ? (
        <div style={{ padding: 12, color: 'var(--danger)', fontSize: 12 }}>Failed to load: {error}</div>
      ) : conns.length === 0 ? (
        <div className="faint" style={{ padding: 24, textAlign: 'center', fontSize: 12 }}>
          No users have connected to {provider.display_name || provider.name} yet.
        </div>
      ) : (
        <table className="tbl" style={{ fontSize: 12 }}>
          <thead>
            <tr>
              <th style={{ textAlign: 'left' }}>User</th>
              <th style={{ textAlign: 'left' }}>Status</th>
              <th style={{ textAlign: 'left' }}>Scopes</th>
              <th style={{ textAlign: 'left' }}>Expires</th>
              <th style={{ textAlign: 'left' }}>Last refresh</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {conns.map(c => (
              <tr key={c.id}>
                <td className="mono" style={{ fontSize: 11 }}>{c.user_id}</td>
                <td>{statusChip(c)}</td>
                <td className="mono" style={{ fontSize: 10.5, color: 'var(--fg-muted)' }}>
                  {(c.scopes || []).slice(0, 3).join(' ')}
                  {c.scopes && c.scopes.length > 3 ? ` +${c.scopes.length - 3}` : ''}
                </td>
                <td className="mono faint" style={{ fontSize: 11 }}>
                  {c.expires_at ? new Date(c.expires_at).toLocaleString() : '—'}
                </td>
                <td className="mono faint" style={{ fontSize: 11 }}>
                  {c.last_refreshed_at ? new Date(c.last_refreshed_at).toLocaleString() : '—'}
                </td>
                <td>
                  <div className="row" style={{ gap: 4 }}>
                    <button className="btn ghost sm" onClick={() => setTestModal({ userId: c.user_id, connId: c.id })} title="Test token retrieval">
                      Test
                    </button>
                    <button className="btn ghost sm danger" onClick={() => handleDisconnect(c.id)}>
                      Disconnect
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function ProviderAudit({ provider }) {
  // Filter the platform audit log by target_id for this provider, and fall back
  // to action prefix `vault.` since older events may not always carry target_id.
  const { data, loading } = useAPI(
    '/audit-logs?target_id=' + encodeURIComponent(provider.id) + '&limit=30',
    [provider.id]
  );
  const raw = data?.items || data?.audit_logs || data?.data || (Array.isArray(data) ? data : []);
  const events = raw.filter(e => (e.action || '').startsWith('vault.'));

  return (
    <div style={{padding: 16}}>
      {loading ? (
        <div className="faint" style={{fontSize: 11}}>Loading…</div>
      ) : events.length === 0 ? (
        <div className="faint" style={{fontSize: 11.5, textAlign: 'center', padding: 20}}>
          No audit events for this provider yet.
        </div>
      ) : events.map((e, i) => (
        <div key={e.id || i} className="row" style={{padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11.5}}>
          <span style={{width:6,height:6,borderRadius:1,background:'var(--fg-dim)',flexShrink:0}}/>
          <span className="mono" style={{flex: 1}}>{e.action}</span>
          <span className="faint" style={{fontSize: 10.5}}>{e.actor_id || e.actor || '—'}</span>
          <span className="faint mono" style={{fontSize: 10}}>{relativeTime(e.created_at || e.t)}</span>
        </div>
      ))}
    </div>
  );
}

// --- Rotate client_secret modal --------------------------------------------

function RotateSecretModal({ provider, onClose, onConfirm }) {
  const [secret, setSecret] = React.useState('');
  const [working, setWorking] = React.useState(false);
  const [error, setError] = React.useState(null);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const go = async () => {
    if (!secret.trim()) { setError('Secret is required.'); return; }
    setWorking(true);
    setError(null);
    try {
      await onConfirm(secret.trim());
    } catch (e) {
      setError(e.message);
    } finally {
      setWorking(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Rotate client secret</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Paste the new client secret for{' '}
          <b style={{color:'var(--fg)'}}>{provider.display_name || provider.name}</b>.
          Issue a new secret in the provider console first, then update here.
        </p>
        <div style={{ marginTop: 14 }}>
          <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
            New client secret
          </label>
          <input
            type="password"
            autoFocus
            value={secret}
            onChange={e => setSecret(e.target.value)}
            placeholder="••••••••••••••"
            style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
          />
        </div>
        <div style={{
          marginTop: 12, background:'var(--surface-0)', border:'1px solid var(--hairline)',
          padding: 10, borderRadius: 3, fontSize: 11, lineHeight: 1.5,
        }}>
          <b style={{color:'var(--fg)'}}>Heads up.</b>
          {' '}Existing connections keep working; the new secret is used on the next token refresh.
          Revoke the old secret in the provider console once you've verified a refresh succeeds.
        </div>
        {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
        <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
          <button className="btn primary" onClick={go} disabled={working || !secret.trim()}>
            {working ? 'Rotating…' : 'Rotate secret'}
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Test token modal ---------------------------------------------------------
// Shows metadata (expiry) for a user's vault connection token without
// revealing the raw access token value. Admin-only diagnostic tool.

function VaultTestTokenModal({ provider, userId, onClose }) {
  const [result, setResult] = React.useState(null); // {ok, exp, error}
  const [loading, setLoading] = React.useState(false);

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const handleTest = async () => {
    setLoading(true);
    setResult(null);
    try {
      // Use admin connection list to get the connection; the token endpoint
      // itself requires a Bearer token so we just check the connection state.
      const conns = await API.get('/admin/vault/connections?provider_id=' + encodeURIComponent(provider.id));
      const data = conns?.data || [];
      const conn = data.find(c => c.user_id === userId);
      if (!conn) {
        setResult({ ok: false, error: 'Connection not found for this user.' });
        return;
      }
      const expired = conn.expires_at && new Date(conn.expires_at) < new Date();
      const expiring = conn.expires_at && !expired && (new Date(conn.expires_at).getTime() - Date.now()) < 60 * 60 * 1000;
      setResult({
        ok: !expired && !conn.needs_reauth,
        exp: conn.expires_at,
        needsReauth: conn.needs_reauth,
        expired,
        expiring,
      });
    } catch (e) {
      setResult({ ok: false, error: e.message || 'Request failed' });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 440}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin: 0, fontSize: 15, fontWeight: 600}}>Test token — {provider.display_name || provider.name}</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Checks the stored connection for user <span className="mono">{userId}</span>. No token value is shown.
        </p>
        <div style={{marginTop: 14}}>
          {!result && !loading && (
            <button className="btn primary sm" onClick={handleTest}>Run check</button>
          )}
          {loading && <div className="faint" style={{fontSize: 12}}>Checking…</div>}
          {result && (
            <div style={{display:'flex', flexDirection:'column', gap: 10}}>
              {result.ok ? (
                <div className="chip success" style={{height: 24, fontSize: 12, padding: '0 10px'}}>
                  Token valid
                  {result.exp && <span className="faint" style={{marginLeft: 8, fontSize: 11}}>
                    exp: {new Date(result.exp).toLocaleString()}
                  </span>}
                  {result.expiring && <span className="chip warn" style={{height: 16, fontSize: 10, marginLeft: 6}}>expiring soon</span>}
                </div>
              ) : (
                <div className="chip danger" style={{height: 24, fontSize: 12, padding: '0 10px'}}>
                  {result.needsReauth ? 'Needs re-auth' : result.expired ? 'Token expired' : result.error || 'Invalid'}
                </div>
              )}
              <button className="btn ghost sm" onClick={handleTest} disabled={loading}>Re-run</button>
            </div>
          )}
        </div>
        <div className="row" style={{marginTop: 18, justifyContent:'flex-end'}}>
          <button className="btn ghost" onClick={onClose}>Close</button>
        </div>
      </div>
    </div>
  );
}

// --- Delete confirmation ---------------------------------------------------

function DeleteProviderModal({ provider, onClose, onConfirm }) {
  const [working, setWorking] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [confirmText, setConfirmText] = React.useState('');

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

  const canDelete = confirmText === provider.name;

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Delete provider</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Deleting <b style={{color:'var(--fg)'}}>{provider.display_name || provider.name}</b> will remove the provider
          and cascade-delete every user connection tied to it. Agents will no longer be able to request tokens.
        </p>
        <div style={{background:'var(--surface-1)', border:'1px solid var(--hairline)', padding: 10, borderRadius: 4, marginTop: 14, fontSize: 11.5}}>
          <div className="row" style={{gap: 8, marginBottom: 4}}>
            <span style={{color:'var(--warning)'}}>⚠</span>
            <b style={{color:'var(--fg)'}}>This cannot be undone.</b>
          </div>
          <div className="faint" style={{fontSize: 11}}>
            Type <span className="mono" style={{color:'var(--fg)'}}>{provider.name}</span> to confirm.
          </div>
        </div>
        <input
          value={confirmText}
          onChange={e => setConfirmText(e.target.value)}
          placeholder={provider.name}
          style={{...inputStyle, marginTop: 10}}
        />
        {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
        <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={working}>Cancel</button>
          <button className="btn danger" onClick={go} disabled={working || !canDelete}>
            {working ? 'Deleting…' : 'Delete provider'}
          </button>
        </div>
      </div>
    </div>
  );
}

// --- Create wizard ---------------------------------------------------------

function CreateProviderWizard({ onClose, onCreate, onCreated }) {
  const [step, setStep] = React.useState(1); // 1 = pick template, 2 = credentials
  const [template, setTemplate] = React.useState(null); // null when "Custom" chosen
  const [clientID, setClientID] = React.useState('');
  const [clientSecret, setClientSecret] = React.useState('');
  const [scopes, setScopes] = React.useState('');
  const [displayName, setDisplayName] = React.useState('');
  const [active, setActive] = React.useState(true);
  // Custom fields (explicit URL path)
  const [customName, setCustomName] = React.useState('');
  const [authURL, setAuthURL] = React.useState('');
  const [tokenURL, setTokenURL] = React.useState('');
  const [iconURL, setIconURL] = React.useState('');

  const [creating, setCreating] = React.useState(false);
  const [error, setError] = React.useState(null);

  const { data: tplData, loading: tplLoading } = useAPI('/vault/templates');
  const templates = tplData?.data || [];
  const isCustom = step === 2 && template === 'custom';

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const pickTemplate = (tpl) => {
    setTemplate(tpl); // tpl is the template object or 'custom'
    if (tpl && tpl !== 'custom') {
      setDisplayName(tpl.display_name || '');
      setScopes((tpl.default_scopes || []).join('\n'));
      setIconURL(tpl.icon_url || '');
    } else {
      setDisplayName('');
      setScopes('');
      setIconURL('');
    }
    setStep(2);
  };

  const handleCreate = async () => {
    setError(null);
    const scopeList = scopes.split(/\n+/).map(s => s.trim()).filter(Boolean);
    if (!clientID.trim()) { setError('client_id is required.'); return; }
    if (!clientSecret.trim()) { setError('client_secret is required.'); return; }
    let payload;
    if (isCustom) {
      if (!customName.trim()) { setError('name is required.'); return; }
      if (!authURL.trim()) { setError('auth_url is required.'); return; }
      if (!tokenURL.trim()) { setError('token_url is required.'); return; }
      payload = {
        name: customName.trim(),
        display_name: displayName.trim() || customName.trim(),
        auth_url: authURL.trim(),
        token_url: tokenURL.trim(),
        client_id: clientID.trim(),
        client_secret: clientSecret.trim(),
        scopes: scopeList,
        icon_url: iconURL.trim(),
      };
    } else {
      payload = {
        template: template.name,
        client_id: clientID.trim(),
        client_secret: clientSecret.trim(),
      };
      if (displayName.trim() && displayName.trim() !== template.display_name) {
        payload.display_name = displayName.trim();
      }
      if (scopeList.length > 0) {
        payload.scopes = scopeList;
      }
      if (iconURL.trim() && iconURL.trim() !== (template.icon_url || '')) {
        payload.icon_url = iconURL.trim();
      }
    }

    setCreating(true);
    try {
      const result = await onCreate(payload);
      // If active is false, patch immediately (create endpoint defaults to true).
      if (!active && result?.id) {
        try {
          await API.patch('/vault/providers/' + result.id, { active: false });
        } catch (e) {
          setError(`Provider created but disable failed: ${e.message}`);
          onCreated && onCreated(result);
          return;
        }
      }
      onCreated && onCreated(result);
      onClose();
    } catch (e) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{
        position:'fixed', top: 0, right: 0, bottom: 0, width: 560,
        background:'var(--surface-0)', borderLeft:'1px solid var(--hairline-bright)',
        display:'flex', flexDirection:'column', boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        <div className="row" style={{padding: '14px 16px', borderBottom:'1px solid var(--hairline)'}}>
          <h2 style={{margin:0, fontSize: 14, fontWeight: 600, flex:1}}>
            {step === 1 ? 'New provider · choose template' : (isCustom ? 'New provider · custom' : 'New provider · credentials')}
          </h2>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div style={{flex:1, overflowY:'auto', padding: 20}}>
          {step === 1 ? (
            <div>
              {tplLoading ? (
                <div className="faint" style={{ fontSize: 12 }}>Loading templates…</div>
              ) : (
                <>
                  <div className="faint" style={{ fontSize: 11.5, marginBottom: 12, lineHeight: 1.5 }}>
                    Pick a template to pre-fill the OAuth endpoints and default scopes,
                    or choose <b style={{color:'var(--fg)'}}>Custom</b> to enter everything manually.
                  </div>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 8 }}>
                    {templates.map(t => (
                      <TemplateCard key={t.name} template={t} onPick={() => pickTemplate(t)}/>
                    ))}
                    <button onClick={() => pickTemplate('custom')} style={{
                      display: 'flex', alignItems: 'center', gap: 10, padding: 12,
                      background: 'var(--surface-1)',
                      border: '1px dashed var(--hairline-strong)',
                      borderRadius: 4, cursor: 'pointer', textAlign: 'left',
                    }}>
                      <div style={{
                        width: 32, height: 32, borderRadius: 5, background: 'var(--surface-2)',
                        display:'flex', alignItems:'center', justifyContent:'center',
                        color: 'var(--fg-muted)', border:'1px solid var(--hairline)',
                      }}>
                        <Icon.Plus width={14} height={14}/>
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: 12, fontWeight: 500 }}>Custom</div>
                        <div className="faint" style={{ fontSize: 10.5, marginTop: 1 }}>
                          Enter auth_url / token_url manually
                        </div>
                      </div>
                    </button>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div style={{display:'flex', flexDirection:'column', gap: 16}}>
              {!isCustom && template && (
                <div style={{
                  display: 'flex', alignItems: 'center', gap: 10, padding: 10,
                  background: 'var(--surface-1)', border: '1px solid var(--hairline)',
                  borderRadius: 4,
                }}>
                  <div style={{
                    width: 32, height: 32, borderRadius: 5, background: 'var(--surface-2)',
                    display:'flex', alignItems:'center', justifyContent:'center',
                    border:'1px solid var(--hairline)', overflow: 'hidden',
                  }}>
                    {template.icon_url
                      ? <img src={template.icon_url} alt="" style={{ width: 26, height: 26, objectFit: 'contain' }}/>
                      : <span style={{fontSize: 14, fontWeight: 600}}>{(template.display_name || '?')[0]}</span>}
                  </div>
                  <div style={{flex:1}}>
                    <div style={{fontSize: 12.5, fontWeight: 500}}>{template.display_name}</div>
                    <div className="faint mono" style={{fontSize: 10.5}}>{template.name}</div>
                  </div>
                  <button className="btn ghost sm" onClick={() => { setTemplate(null); setStep(1); }}>
                    Change
                  </button>
                </div>
              )}

              {isCustom && (
                <>
                  <div>
                    <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                      Name <span style={{color:'var(--danger)'}}>*</span>{' '}
                      <span className="faint">(unique key, lowercase)</span>
                    </label>
                    <input
                      value={customName}
                      onChange={e => setCustomName(e.target.value)}
                      placeholder="my_custom_provider"
                      style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
                    />
                  </div>
                  <div>
                    <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                      Authorize URL <span style={{color:'var(--danger)'}}>*</span>
                    </label>
                    <input
                      value={authURL}
                      onChange={e => setAuthURL(e.target.value)}
                      placeholder="https://example.com/oauth/authorize"
                      style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
                    />
                  </div>
                  <div>
                    <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                      Token URL <span style={{color:'var(--danger)'}}>*</span>
                    </label>
                    <input
                      value={tokenURL}
                      onChange={e => setTokenURL(e.target.value)}
                      placeholder="https://example.com/oauth/token"
                      style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
                    />
                  </div>
                </>
              )}

              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                  Client ID <span style={{color:'var(--danger)'}}>*</span>
                </label>
                <input
                  value={clientID}
                  onChange={e => setClientID(e.target.value)}
                  placeholder="client_abc123"
                  style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
                />
              </div>

              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                  Client Secret <span style={{color:'var(--danger)'}}>*</span>
                </label>
                <input
                  type="password"
                  value={clientSecret}
                  onChange={e => setClientSecret(e.target.value)}
                  placeholder="••••••••••••••"
                  style={{...inputStyle, fontFamily: 'var(--font-mono)'}}
                />
                <div className="faint" style={{ fontSize: 10.5, marginTop: 4 }}>
                  Encrypted at rest. Never shown again after save.
                </div>
              </div>

              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                  Scopes <span className="faint">(one per line)</span>
                </label>
                <textarea
                  value={scopes}
                  onChange={e => setScopes(e.target.value)}
                  rows={4}
                  placeholder="openid&#10;email&#10;profile"
                  style={{...inputStyle, fontFamily: 'var(--font-mono)', fontSize: 11.5, resize: 'vertical'}}
                />
              </div>

              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                  Display name <span className="faint">(optional override)</span>
                </label>
                <input
                  value={displayName}
                  onChange={e => setDisplayName(e.target.value)}
                  placeholder={isCustom ? 'My Custom Provider' : (template?.display_name || '')}
                  style={inputStyle}
                />
              </div>

              {isCustom && (
                <div>
                  <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>
                    Icon URL <span className="faint">(optional)</span>
                  </label>
                  <input
                    value={iconURL}
                    onChange={e => setIconURL(e.target.value)}
                    placeholder="https://example.com/icon.png"
                    style={inputStyle}
                  />
                </div>
              )}

              <label className="row" style={{
                gap: 8, fontSize: 11.5, padding: '8px 10px',
                background: 'var(--surface-1)', border: '1px solid var(--hairline)',
                borderRadius: 3, cursor: 'pointer',
              }}>
                <input type="checkbox" checked={active} onChange={e => setActive(e.target.checked)}/>
                <span>Active <span className="faint">— users can start new connections immediately</span></span>
              </label>

              {error && <div style={{color:'var(--danger)', fontSize: 11}}>{error}</div>}
            </div>
          )}
        </div>

        {/* Footer — CLI hint + action buttons */}
        {step === 2 && (
          <div style={{
            borderTop: '1px solid var(--hairline)',
            padding: '8px 14px',
            background: 'var(--surface-2)',
            display: 'flex', alignItems: 'center', gap: 8,
          }}>
            <Icon.Terminal width={11} height={11} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
            <span style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', flexShrink: 0 }}>CLI</span>
            <span className="mono" style={{
              flex: 1, fontSize: 11, color: 'var(--fg-muted)',
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {isCustom
                ? `shark vault provider create --name ${customName || '<name>'} --auth-url … --token-url … --client-id … --client-secret …`
                : `shark vault provider create --template ${template?.name || '<template>'} --client-id … --client-secret …`}
            </span>
          </div>
        )}

        <div className="row" style={{padding: 12, borderTop: '1px solid var(--hairline)', gap: 8, justifyContent: 'flex-end'}}>
          {step === 1 ? (
            <button className="btn ghost" onClick={onClose}>Cancel</button>
          ) : (
            <>
              <button className="btn ghost" onClick={() => setStep(1)} disabled={creating}>Back</button>
              <button className="btn ghost" onClick={onClose} disabled={creating}>Cancel</button>
              <button className="btn primary" onClick={handleCreate} disabled={creating || !clientID.trim() || !clientSecret.trim() || (isCustom && (!customName.trim() || !authURL.trim() || !tokenURL.trim()))}>
                {creating ? 'Creating…' : 'Create provider'}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function TemplateCard({ template, onPick }) {
  const [failed, setFailed] = React.useState(false);
  const scopeCount = (template.default_scopes || []).length;
  return (
    <button onClick={onPick} style={{
      display: 'flex', alignItems: 'center', gap: 10, padding: 12,
      background: 'var(--surface-1)',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 4, cursor: 'pointer', textAlign: 'left',
      transition: 'background 60ms',
    }}
      onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
      onMouseLeave={e => (e.currentTarget.style.background = 'var(--surface-1)')}
    >
      <div style={{
        width: 32, height: 32, borderRadius: 5, background: 'var(--surface-2)',
        display:'flex', alignItems:'center', justifyContent:'center',
        border:'1px solid var(--hairline)', overflow: 'hidden', flexShrink: 0,
      }}>
        {template.icon_url && !failed
          ? <img src={template.icon_url} onError={() => setFailed(true)} alt=""
              style={{ width: 26, height: 26, objectFit: 'contain' }}/>
          : <span style={{fontSize: 14, fontWeight: 600, color: 'var(--fg)'}}>
              {(template.display_name || template.name || '?')[0].toUpperCase()}
            </span>}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 12, fontWeight: 500, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>
          {template.display_name}
        </div>
        <div className="faint mono" style={{ fontSize: 10.5, marginTop: 1, overflow:'hidden', textOverflow:'ellipsis', whiteSpace:'nowrap' }}>
          {template.name}
          {scopeCount > 0 && <span> · {scopeCount} scope{scopeCount !== 1 && 's'}</span>}
        </div>
      </div>
    </button>
  );
}

// --- Small helpers ---------------------------------------------------------

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
