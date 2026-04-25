// @ts-nocheck
import React from 'react'
import { Icon, CopyField, Sparkline } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useURLParam } from './useURLParams'
import { usePageActions } from './useKeyboardShortcuts'
import { TeachEmptyState } from './TeachEmptyState'

// API Keys page — M2M credential management
// Table + detail w/ rotation timeline + create/rotate modals + scope matrix

const kThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const kTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const segStyle = { height: 28, display: 'inline-flex', border: '1px solid var(--hairline-strong)', borderRadius: 3, overflow: 'hidden' };
const segBtn = { padding: '0 10px', height: 28, fontSize: 11, borderRight: '1px solid var(--hairline)' };
const labelStyle = { display: 'block', fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 5 };
const inputStyle = { width: '100%', padding: '7px 10px', background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 3, color: 'var(--fg)', fontSize: 12, outline: 'none' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };

export function ApiKeys() {
  const [selected, setSelected] = React.useState(null);
  const [filter, setFilter] = useURLParam('status', 'all'); // all/active/revoked/expiring
  const [env, setEnv] = React.useState('all');
  const [query, setQuery] = React.useState('');
  const [createOpen, setCreateOpen] = React.useState(false);
  const [rotateModal, setRotateModal] = React.useState(null);
  const [revealKey, setRevealKey] = React.useState(null);

  const { data: keysRaw, loading, refresh } = useAPI('/api-keys');
  const keys = keysRaw?.api_keys || [];

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

  const isExpiringSoon = (k) => {
    if (!k.expires_at || k.revoked_at) return false;
    const msLeft = new Date(k.expires_at) - Date.now();
    return msLeft > 0 && msLeft < 7 * 86400e3;
  };

  const keyStatus = (k) => {
    if (k.revoked_at) return 'revoked';
    return 'active';
  };

  const filtered = keys.filter(k => {
    const status = keyStatus(k);
    if (filter === 'active' && status !== 'active') return false;
    if (filter === 'revoked' && status !== 'revoked') return false;
    if (filter === 'expiring' && !isExpiringSoon(k)) return false;
    if (query && !k.name.toLowerCase().includes(query.toLowerCase()) && !k.key_prefix?.includes(query)) return false;
    return true;
  });

  const stats = {
    active: keys.filter(k => keyStatus(k) === 'active').length,
    revoked: keys.filter(k => keyStatus(k) === 'revoked').length,
    expiring: keys.filter(k => isExpiringSoon(k)).length,
    reqs24h: 0,
  };

  const handleCreate = async (form) => {
    const result = await API.post('/api-keys', {
      name: form.name,
      scopes: form.scopes,
      expires_at: form.expiresAt || null,
    });
    setRevealKey(result.key || result.api_key);
    refresh();
  };

  const handleRotate = async (keyId) => {
    const result = await API.post(`/api-keys/${keyId}/rotate`);
    setRevealKey(result.key || result.api_key);
    refresh();
  };

  const handleRevoke = async (keyId) => {
    await API.del(`/api-keys/${keyId}`);
    if (selected?.id === keyId) setSelected(null);
    refresh();
  };

  const handleUpdate = async (keyId, updates) => {
    await API.patch(`/api-keys/${keyId}`, updates);
    refresh();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Reveal banner — non-dismissable until user acknowledges */}
      {revealKey && (
        <div style={{
          background: '#000', color: '#fff', padding: '12px 20px',
          display: 'flex', alignItems: 'center', gap: 12, flexShrink: 0,
          borderBottom: '2px solid var(--success)',
        }}>
          <Icon.Key width={14} height={14} style={{ color: 'var(--success)', flexShrink: 0 }}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontSize: 11, color: 'var(--success)', fontWeight: 600, marginBottom: 3, textTransform: 'uppercase', letterSpacing: '0.06em' }}>
              New key — copy now. This is the only time it will be shown.
            </div>
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak: 'break-all' }}>{revealKey}</span>
          </div>
          <button className="btn ghost sm" style={{ color: 'var(--fg)', borderColor: 'var(--hairline-strong)', flexShrink: 0 }}
            onClick={() => navigator.clipboard?.writeText(revealKey)}>
            <Icon.Copy width={10} height={10}/> Copy
          </button>
          <button className="btn ghost sm" style={{ color: 'var(--fg)', borderColor: 'var(--hairline-strong)', flexShrink: 0 }}
            onClick={() => setRevealKey(null)}>
            I've saved it
          </button>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 480px' : '1fr', flex: 1, overflow: 'hidden' }}>
        <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
          {/* Header */}
          <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
            <div className="row" style={{ gap: 12 }}>
              <div style={{ flex: 1 }}>
                <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>API Keys</h1>
                <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                  Server-to-server credentials · {stats.reqs24h.toLocaleString()} requests in last 24h
                </p>
              </div>
              <button className="btn ghost"><Icon.Explorer width={11} height={11}/> Try in API Explorer</button>
              <button className="btn primary" onClick={() => setCreateOpen(true)}>
                <Icon.Plus width={11} height={11}/> New key
              </button>
            </div>

            {/* Stat strip */}
            <div className="row" style={{ gap: 18, marginTop: 14 }}>
              <Stat label="Active" value={stats.active}/>
              <Stat label="Expiring ≤7d" value={stats.expiring} warn/>
              <Stat label="Revoked" value={stats.revoked} dim/>
              <div style={{ flex: 1 }}/>
              <div className="row" style={{ gap: 4 }}>
                <span className="faint" style={{fontSize: 10, letterSpacing: '0.08em', textTransform:'uppercase'}}>Last 24h</span>
                <span className="mono" style={{fontSize: 13}}>{stats.reqs24h.toLocaleString()}</span>
                <span className="faint" style={{fontSize: 11}}>req</span>
              </div>
            </div>
          </div>

          {/* Toolbar */}
          <div className="row" style={{ padding: '10px 20px', gap: 8, borderBottom: '1px solid var(--hairline)' }}>
            <div className="row" style={{
              flex: 1, gap: 6, padding: '0 8px', height: 28, maxWidth: 300,
              background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 4,
            }}>
              <Icon.Search width={12} height={12} style={{opacity:0.6}}/>
              <input placeholder="Filter by name or prefix…"
                value={query} onChange={e => setQuery(e.target.value)}
                style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
            </div>

            <div className="seg" style={segStyle}>
              {[['all','All'],['active','Active'],['expiring','Expiring'],['revoked','Revoked']].map(([v,l]) => (
                <button key={v} onClick={() => setFilter(v)}
                  style={{...segBtn, background: filter===v ? 'var(--surface-3)':'var(--surface-2)', color: filter===v ? 'var(--fg)':'var(--fg-muted)'}}>
                  {l}
                </button>
              ))}
            </div>

            <div style={{flex:1}}/>
            <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {keys.length}</span>
          </div>

          {/* Table */}
          <div style={{ flex: 1, overflow: 'auto' }}>
            {loading ? (
              <div className="faint" style={{ padding: 40, textAlign: 'center', fontSize: 12 }}>Loading…</div>
            ) : keys.length === 0 ? (
              <TeachEmptyState
                icon="Key"
                title="No API keys"
                description="API keys are M2M credentials for service-to-service authentication. Each key is scoped, rotatable, and can have an expiry."
                createLabel="New API Key"
                onCreate={() => setCreateOpen(true)}
                cliSnippet="shark keys create --name 'ci-deploy' --scope 'admin'"
              />
            ) : (
              <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
                <thead>
                  <tr>
                    <th style={kThStyle}>Name</th>
                    <th style={kThStyle}>Prefix</th>
                    <th style={kThStyle}>Scopes</th>
                    <th style={kThStyle}>Last used</th>
                    <th style={kThStyle}>Expires</th>
                    <th style={kThStyle}>Created</th>
                    <th style={{...kThStyle, width: 40}}/>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map(k => {
                    const status = keyStatus(k);
                    const expiring = isExpiringSoon(k);
                    return (
                      <tr key={k.id} onClick={() => setSelected(k)}
                        style={{
                          cursor: 'pointer',
                          background: selected?.id === k.id ? 'var(--surface-2)' : 'transparent',
                          opacity: status === 'revoked' ? 0.55 : 1,
                        }}>
                        <td style={kTdStyle}>
                          <div style={{ fontWeight: 500 }}>{k.name}</div>
                          {expiring && <div className="row" style={{gap:4, marginTop: 2}}>
                            <span className="dot warn"/>
                            <span className="faint" style={{fontSize: 10.5}}>expiring soon</span>
                          </div>}
                        </td>
                        <td style={kTdStyle}>
                          <span className="mono" style={{fontSize: 11}}>{k.key_prefix || '—'}</span>
                        </td>
                        <td style={kTdStyle}>
                          <div className="row" style={{gap:3, flexWrap:'wrap', maxWidth: 180}}>
                            {(k.scopes || []).slice(0,2).map(s => (
                              <span key={s} className="chip mono" style={{height:15, fontSize:9, padding:'0 4px'}}>{s}</span>
                            ))}
                            {(k.scopes || []).length > 2 && <span className="faint" style={{fontSize:10}}>+{k.scopes.length-2}</span>}
                          </div>
                        </td>
                        <td style={kTdStyle}>
                          <span className="mono faint" style={{fontSize: 10.5}}>
                            {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : '—'}
                          </span>
                        </td>
                        <td style={kTdStyle}>
                          {status === 'revoked' ? (
                            <span className="chip danger" style={{height:15, fontSize:9}}>revoked</span>
                          ) : expiring ? (
                            <span className="row" style={{gap:5}}>
                              <span className="dot warn"/>
                              <span className="mono" style={{fontSize: 10.5, color:'var(--warn)'}}>
                                {k.expires_at ? new Date(k.expires_at).toLocaleDateString() : '—'}
                              </span>
                            </span>
                          ) : (
                            <span className="mono faint" style={{fontSize: 10.5}}>
                              {k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'never'}
                            </span>
                          )}
                        </td>
                        <td style={kTdStyle}>
                          <span className="faint mono" style={{fontSize: 10.5}}>
                            {k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}
                          </span>
                        </td>
                        <td style={kTdStyle}>
                          <button className="btn ghost icon sm" onClick={(e) => { e.stopPropagation(); setRotateModal(k); }} title="Rotate">
                            <Icon.Signing width={11} height={11}/>
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                  {filtered.length === 0 && !loading && (
                    <tr>
                      <td colSpan={7} style={{ padding: 40, textAlign: 'center' }}>
                        <span className="faint" style={{ fontSize: 12 }}>No keys match the current filter.</span>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            )}
          </div>

          {/* Footer */}
          <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
            <Icon.Debug width={11} height={11} style={{opacity:0.5}}/>
            <span className="faint">CLI parity:</span>
            <span className="mono faint">shark keys list</span>
            <div style={{flex:1}}/>
            <span className="faint mono">POST /v1/api-keys</span>
          </div>
        </div>

        {selected && (
          <KeyDetail
            k={selected}
            onClose={() => setSelected(null)}
            onRotate={() => setRotateModal(selected)}
            onRevoke={handleRevoke}
            onUpdate={handleUpdate}
          />
        )}
        {createOpen && (
          <CreateKeyModal
            onClose={() => setCreateOpen(false)}
            onCreate={handleCreate}
          />
        )}
        {rotateModal && (
          <RotateKeyModal
            k={rotateModal}
            onClose={() => setRotateModal(null)}
            onRotate={handleRotate}
          />
        )}
      </div>
    </div>
  );
}

function Stat({ label, value, warn, dim }) {
  return (
    <div>
      <div className="faint" style={{fontSize: 10, letterSpacing:'0.08em', textTransform:'uppercase', marginBottom: 2}}>{label}</div>
      <div className="row" style={{gap: 5}}>
        <span style={{fontSize: 16, fontWeight: 500, color: warn ? 'var(--warn)' : dim ? 'var(--fg-dim)' : 'var(--fg)', fontVariantNumeric:'tabular-nums'}}>{value}</span>
        {warn && <span className="dot warn pulse"/>}
      </div>
    </div>
  );
}

function ReqSparkline({ count }) {
  // Fake deterministic spark based on count
  if (count === 0) return <span className="faint mono" style={{fontSize: 11}}>0</span>;
  const seed = count % 7;
  const vals = Array.from({length: 14}, (_, i) => {
    const phase = (i + seed) * 0.7;
    return 0.4 + 0.3 * Math.sin(phase) + 0.3 * Math.cos(phase * 0.6);
  });
  const max = Math.max(...vals);
  const pts = vals.map((v, i) => `${(i/13) * 60},${18 - (v/max)*16}`).join(' ');
  return (
    <div className="row" style={{justifyContent:'flex-end', gap: 6}}>
      <svg width="60" height="18" style={{display:'block'}}>
        <polyline points={pts} fill="none" stroke="currentColor" strokeWidth="1" opacity="0.55"/>
      </svg>
      <span className="mono" style={{fontSize: 11, minWidth: 52, textAlign:'right'}}>{count.toLocaleString()}</span>
    </div>
  );
}

function KeyDetail({ k, onClose, onRotate, onRevoke, onUpdate }) {
  const allScopes = ['users:read', 'users:write', 'orgs:read', 'orgs:write', 'audit:export', 'webhooks:manage', 'agents:manage', 'billing:write', 'billing:read', 'metrics:read'];
  const status = k.revoked_at ? 'revoked' : 'active';

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)', background: 'var(--surface-0)',
      display:'flex', flexDirection:'column', overflow:'hidden',
    }}>
      <div style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 10 }}>
          <div style={{
            width: 28, height: 28, background: 'var(--surface-3)', borderRadius: 0,
            display:'flex', alignItems:'center', justifyContent:'center',
          }}>
            <Icon.Key width={14} height={14} style={{color:'var(--fg)'}}/>
          </div>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight: 500, fontSize: 14}}>{k.name}</div>
            <div className="row" style={{gap: 6, fontSize: 10.5, marginTop: 2}}>
              <span className="mono faint">{k.key_prefix || '—'}</span>
              {k.key_prefix && <CopyField value={k.key_prefix} truncate={20}/>}
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn ghost sm" onClick={onRotate} disabled={status === 'revoked'}>
            <Icon.Signing width={11} height={11}/> Rotate
          </button>
          <button className="btn ghost sm" style={{color:'var(--danger)'}} disabled={status === 'revoked'}
            onClick={() => onRevoke(k.id)}>
            Revoke
          </button>
          <div style={{flex:1}}/>
          {status === 'active' ? (
            <span className="chip success" style={{height:18, fontSize:10}}><span className="dot success"/>active</span>
          ) : <span className="chip danger" style={{height:18, fontSize:10}}>revoked</span>}
        </div>
      </div>

      <div style={{flex:1, overflowY:'auto', padding: 16, display:'flex', flexDirection:'column', gap: 20}}>
        {/* Metadata */}
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">Created</span>
          <span className="mono">{k.created_at ? new Date(k.created_at).toLocaleDateString() : '—'}</span>
          <span className="faint">Expires</span>
          <span>
            <span className="mono">{k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'never'}</span>
            {k.expires_at && !k.revoked_at && new Date(k.expires_at) - Date.now() < 7 * 86400e3 && new Date(k.expires_at) > Date.now() && (
              <span className="chip warn" style={{height:15, fontSize:9, marginLeft:6}}>expiring soon</span>
            )}
          </span>
          <span className="faint">Last used</span>
          <span className="mono">
            {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : '—'}
          </span>
          {k.rate_limit != null && <>
            <span className="faint">Rate limit</span>
            <span className="mono">{k.rate_limit.toLocaleString()} / min</span>
          </>}
          {k.revoked_at && <>
            <span className="faint">Revoked</span>
            <span className="mono">{new Date(k.revoked_at).toLocaleDateString()}</span>
          </>}
        </div>

        {/* Scope matrix */}
        <div>
          <h4 style={sectionLabelStyle}>Scopes</h4>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 6 }}>
            {allScopes.map(s => {
              const granted = (k.scopes || []).includes(s);
              const danger = s.endsWith(':write') || s.includes('manage');
              return (
                <div key={s} style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  padding: '6px 10px', borderRadius: 3,
                  background: granted ? 'var(--surface-2)' : 'var(--surface-1)',
                  border: '1px solid ' + (granted ? 'var(--hairline-bright)' : 'var(--hairline)'),
                  opacity: granted ? 1 : 0.45,
                }}>
                  <span style={{
                    width: 10, height: 10, borderRadius: 2,
                    background: granted ? (danger ? 'var(--warn)' : 'var(--success)') : 'transparent',
                    border: '1px solid ' + (granted ? 'transparent' : 'var(--hairline-strong)'),
                  }}/>
                  <span className="mono" style={{fontSize: 10.5}}>{s}</span>
                </div>
              );
            })}
          </div>
          <div className="faint" style={{fontSize: 10.5, marginTop: 8}}>
            <span className="row" style={{gap: 10, display:'inline-flex'}}>
              <span className="row" style={{gap: 4}}><span style={{width:8,height:8,background:'var(--success)'}}/>read</span>
              <span className="row" style={{gap: 4}}><span style={{width:8,height:8,background:'var(--warn)'}}/>write/manage</span>
            </span>
          </div>
        </div>

        {/* Usage audit */}
        <KeyUsageAudit keyId={k.id}/>
      </div>
      {/* curl snippet */}
      <div style={{ padding: '10px 14px', borderTop: '1px solid var(--hairline)', background: 'var(--surface-1)' }}>
        <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 6 }}>curl snippet</div>
        <CurlSnippet keyPrefix={k.key_prefix || 'sk_live_...'} scopes={k.scopes || []}/>
      </div>
      <CLIFooter command={`shark api-key show ${k.id}`}/>
    </aside>
  );
}

function CurlSnippet({ keyPrefix, scopes }) {
  const [copied, setCopied] = React.useState(false);
  const endpoint = scopes.includes('users:read') ? '/api/v1/users'
    : scopes.includes('orgs:read') ? '/api/v1/organizations'
    : scopes.includes('audit:export') ? '/api/v1/audit-logs'
    : '/api/v1/users';
  const cmd = `curl -H "Authorization: Bearer ${keyPrefix}" ${window.location.origin}${endpoint}`;

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 6,
      padding: '6px 8px', borderRadius: 4,
      background: 'var(--surface-3)', fontFamily: 'var(--font-mono)',
      fontSize: 10.5, color: 'var(--fg-muted)', lineHeight: 1.4,
    }}>
      <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{cmd}</span>
      <button className="btn ghost icon sm" style={{ color: 'var(--fg)', borderColor: 'var(--hairline-strong)' }}
        onClick={() => { navigator.clipboard?.writeText(cmd); setCopied(true); setTimeout(() => setCopied(false), 900); }}>
        {copied ? <Icon.Check width={10} height={10} style={{color:'var(--success)'}}/> : <Icon.Copy width={10} height={10}/>}
      </button>
    </div>
  );
}

function KeyUsageAudit({ keyId }) {
  const { data, loading } = useAPI('/audit-logs?limit=10&actor_type=api_key&actor_id=' + keyId);
  const events = data?.items || data || [];

  return (
    <div>
      <h4 style={sectionLabelStyle}>Recent usage</h4>
      {loading ? (
        <div className="faint" style={{ fontSize: 11 }}>Loading…</div>
      ) : events.length === 0 ? (
        <div className="faint" style={{ fontSize: 11 }}>No audit events for this key.</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {events.slice(0, 8).map((e, i) => (
            <div key={e.id || i} className="row" style={{
              padding: '5px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11,
            }}>
              <span className="mono faint" style={{ width: 55, flexShrink: 0 }}>
                {(() => {
                  const d = Date.now() - new Date(e.created_at || e.t).getTime();
                  if (d < 60e3) return Math.floor(d/1e3) + 's';
                  if (d < 3600e3) return Math.floor(d/60e3) + 'm';
                  if (d < 86400e3) return Math.floor(d/3600e3) + 'h';
                  return Math.floor(d/86400e3) + 'd';
                })()}
              </span>
              <span className="mono" style={{ flex: 1 }}>{e.event_type || e.action || '—'}</span>
              <span className="faint" style={{ fontSize: 10 }}>{e.ip || ''}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateKeyModal({ onClose, onCreate }) {
  const [name, setName] = React.useState('');
  const [selectedScopes, setSelectedScopes] = React.useState(new Set(['users:read']));
  const [expires, setExpires] = React.useState('90d');
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState(null);

  const allScopes = [
    { id: 'users:read', danger: false, desc: 'Read user records' },
    { id: 'users:write', danger: true, desc: 'Create and modify users' },
    { id: 'orgs:read', danger: false, desc: 'Read org records' },
    { id: 'orgs:write', danger: true, desc: 'Create and modify orgs' },
    { id: 'audit:export', danger: false, desc: 'Export audit events' },
    { id: 'webhooks:manage', danger: true, desc: 'Manage webhook endpoints' },
    { id: 'agents:manage', danger: true, desc: 'Register and rotate agents' },
    { id: 'metrics:read', danger: false, desc: 'Read usage metrics' },
  ];

  const toggle = (id) => {
    const next = new Set(selectedScopes);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedScopes(next);
  };

  const expiresAt = () => {
    if (expires === 'never') return null;
    const d = new Date();
    if (expires === '30d') d.setDate(d.getDate() + 30);
    else if (expires === '90d') d.setDate(d.getDate() + 90);
    else if (expires === '1y') d.setFullYear(d.getFullYear() + 1);
    return d.toISOString();
  };

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);
    try {
      await onCreate({
        name,
        scopes: Array.from(selectedScopes),
        expiresAt: expiresAt(),
      });
      onClose();
    } catch (e) {
      setError(e?.message || 'Failed to create key');
      setSubmitting(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 540, maxHeight: '80vh', overflow:'auto'}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>New API key</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6}}>Create a server-to-server credential with scoped access.</p>

        <div style={{marginTop: 16}}>
          <label style={labelStyle}>Name</label>
          <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Production billing sync" style={inputStyle}/>
          <div className="faint" style={{fontSize: 10.5, marginTop: 4}}>Descriptive — shown in audit logs and dashboards.</div>
        </div>

        <div style={{marginTop: 14}}>
          <label style={labelStyle}>Expires in</label>
          <div className="seg" style={{...segStyle, width: '100%'}}>
            {[['30d','30d'],['90d','90d'],['1y','1y'],['never','never']].map(([v,l]) => (
              <button key={v} onClick={() => setExpires(v)}
                style={{...segBtn, flex:1, padding: '6px 10px', background: expires===v ? 'var(--surface-3)':'var(--surface-2)', color: expires===v ? 'var(--fg)':'var(--fg-muted)'}}>
                {l}
              </button>
            ))}
          </div>
          {expires === 'never' && <div className="faint" style={{fontSize: 10.5, marginTop: 4, color:'var(--warn)'}}>⚠ Non-expiring keys require explicit rotation.</div>}
        </div>

        <div style={{marginTop: 14}}>
          <label style={labelStyle}>Scopes · {selectedScopes.size} selected</label>
          <div style={{border:'1px solid var(--hairline)', borderRadius: 3, maxHeight: 200, overflow:'auto'}}>
            {allScopes.map(s => (
              <label key={s.id} className="row" style={{padding: '6px 10px', borderBottom: '1px solid var(--hairline)', gap: 10, cursor:'pointer'}}>
                <input type="checkbox" checked={selectedScopes.has(s.id)} onChange={() => toggle(s.id)}/>
                <span className="mono" style={{fontSize: 11, flex: 1}}>{s.id}</span>
                {s.danger && <span className="chip warn" style={{height:15, fontSize:9}}>write</span>}
                <span className="faint" style={{fontSize: 10.5, width: 160, textAlign:'right'}}>{s.desc}</span>
              </label>
            ))}
          </div>
        </div>

        {error && (
          <div style={{marginTop: 12, padding: '8px 10px', background: 'var(--surface-1)', border: '1px solid var(--danger)', borderRadius: 3, color: 'var(--danger)', fontSize: 11.5}}>
            {error}
          </div>
        )}

        <div className="row" style={{marginTop: 20, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={submitting}>Cancel</button>
          <button className="btn primary" onClick={handleSubmit} disabled={!name || submitting}>
            {submitting ? 'Creating…' : 'Create key'}
          </button>
        </div>
      </div>
    </div>
  );
}

function RotateKeyModal({ k, onClose, onRotate }) {
  const [grace, setGrace] = React.useState('24h');
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState(null);

  const handleRotate = async () => {
    setSubmitting(true);
    setError(null);
    try {
      await onRotate(k.id);
      onClose();
    } catch (e) {
      setError(e?.message || 'Failed to rotate key');
      setSubmitting(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Rotate {k.name}</h2>
        <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
          Issue a new key. The old key ({k.key_prefix || k.id}) will continue working during the grace period, then stop.
        </p>
        <div style={{marginTop: 14}}>
          <label style={labelStyle}>Grace period</label>
          <div className="seg" style={{...segStyle, width: '100%'}}>
            {[['1h','1h'],['24h','24h'],['7d','7d'],['0','Immediate']].map(([v,l]) => (
              <button key={v} onClick={() => setGrace(v)}
                style={{...segBtn, flex:1, padding: '6px 10px', background: grace===v ? 'var(--surface-3)':'var(--surface-2)', color: grace===v ? 'var(--fg)':'var(--fg-muted)'}}>
                {l}
              </button>
            ))}
          </div>
          {grace === '0' && <div className="faint" style={{fontSize: 10.5, marginTop: 4, color:'var(--danger)'}}>⚠ All in-flight requests with old key will fail.</div>}
        </div>

        {error && (
          <div style={{marginTop: 12, padding: '8px 10px', background: 'var(--surface-1)', border: '1px solid var(--danger)', borderRadius: 3, color: 'var(--danger)', fontSize: 11.5}}>
            {error}
          </div>
        )}

        <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
          <button className="btn ghost" onClick={onClose} disabled={submitting}>Cancel</button>
          <button className="btn primary" onClick={handleRotate} disabled={submitting}>
            {submitting ? 'Rotating…' : 'Rotate'}
          </button>
        </div>
      </div>
    </div>
  );
}

