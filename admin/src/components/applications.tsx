// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { usePageActions } from './useKeyboardShortcuts'
import { TeachEmptyState } from './TeachEmptyState'
import { useTabParam } from './useURLParams'

// Applications page — OAuth/OIDC client registrations
// Table view → slide-over detail → live consent-screen preview

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };

export function Applications() {
  const [selected, setSelected] = React.useState(null);
  const [tab, setTab] = useTabParam('config');
  const [filter, setFilter] = React.useState('all');
  const [query, setQuery] = React.useState('');
  const [rotateModal, setRotateModal] = React.useState(null);
  const [createOpen, setCreateOpen] = React.useState(false);

  const { data: appsRaw, loading, refresh } = useAPI('/admin/apps');
  const apps = appsRaw?.applications || [];

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

  usePageActions({ onNew: () => setCreateOpen(true), onRefresh: refresh });

  const filtered = apps.filter(a => {
    if (query && !(a.name.toLowerCase().includes(query.toLowerCase()) || a.client_id.includes(query))) return false;
    return true;
  });

  // Keep selected in sync after refresh
  React.useEffect(() => {
    if (selected) {
      const fresh = apps.find(a => a.id === selected.id);
      if (fresh) setSelected(fresh);
    }
  }, [apps]);

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 560px' : '1fr', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
        {/* Header */}
        <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ gap: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Applications</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                OAuth/OIDC clients · {apps.length} registered
              </p>
            </div>
            <button className="btn ghost"><Icon.Explorer width={11} height={11}/> Discovery</button>
            <button className="btn primary" onClick={() => setCreateOpen(true)}>
              <Icon.Plus width={11} height={11}/> New application
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
          <div style={{flex:1}}/>
          <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {apps.length}</span>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          {loading ? (
            <div className="faint" style={{padding: 20, fontSize: 12}}>Loading applications…</div>
          ) : apps.length === 0 ? (
            <TeachEmptyState
              icon="App"
              title="No applications registered"
              description="Applications are OAuth/OIDC clients (web apps, mobile, SPAs) that authenticate users via Shark."
              createLabel="New Application"
              onCreate={() => setCreateOpen(true)}
              cliSnippet="shark apps create --name 'My Web App'"
            />
          ) : filtered.length === 0 ? (
            <div className="faint" style={{padding: 40, fontSize: 12, textAlign: 'center'}}>
              No applications match your filters.
            </div>
          ) : (
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr>
                  <th style={{...appThStyle, width: 28}}/>
                  <th style={appThStyle}>Name</th>
                  <th style={appThStyle}>Client ID</th>
                  <th style={appThStyle}>Redirects</th>
                  <th style={appThStyle}>CORS Origins</th>
                  <th style={appThStyle}>Created</th>
                  <th style={appThStyle}>Updated</th>
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
                      }}>{a.name[0]}</div>
                    </td>
                    <td style={appTdStyle}>
                      <div style={{fontWeight: 500}}>{a.name}</div>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{a.client_id}</span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {(a.redirect_uris || []).length} URI{(a.redirect_uris || []).length !== 1 && 's'}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>
                        {(a.cors_origins || []).length} origin{(a.cors_origins || []).length !== 1 && 's'}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(a.created_at)}</span>
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(a.updated_at)}</span>
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
          <span className="mono faint">shark apps list</span>
          <div style={{flex:1}}/>
          <span className="faint mono">POST /admin/apps</span>
        </div>
      </div>

      {selected && (
        <AppDetail
          app={selected}
          tab={tab} setTab={setTab}
          onClose={() => setSelected(null)}
          onRotate={() => setRotateModal(selected)}
          onDelete={async () => {
            await API.del('/admin/apps/' + selected.id);
            setSelected(null);
            refresh();
          }}
          onUpdate={async (updates) => {
            const updated = await API.patch('/admin/apps/' + selected.id, updates);
            refresh();
            return updated;
          }}
        />
      )}

      {rotateModal && (
        <RotateSecretModal
          app={rotateModal}
          onClose={() => setRotateModal(null)}
          onRotate={async () => {
            const result = await API.post('/admin/apps/' + rotateModal.id + '/rotate-secret');
            refresh();
            return result;
          }}
        />
      )}
      {createOpen && (
        <CreateAppSlideOver
          onClose={() => setCreateOpen(false)}
          onCreate={async (payload) => {
            const result = await API.post('/admin/apps', payload);
            refresh();
            return result;
          }}
        />
      )}
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

function AppDetail({ app, tab, setTab, onClose, onRotate, onDelete, onUpdate }) {
  const [deleting, setDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState(null);
  const toast = useToast();

  const handleDelete = () => {
    onClose();
    toast.undo(`Deleted "${app.name}"`, async () => {
      try {
        await onDelete();
      } catch (e) {
        toast.error(`Failed to delete: ${e.message}`);
      }
    });
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
          }}>{app.name[0]}</div>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight: 500, fontSize: 14}}>{app.name}</div>
            <div className="row" style={{gap: 6, fontSize: 10.5, marginTop: 2}}>
              <span className="mono faint">{app.client_id}</span>
              <CopyField value={app.client_id} truncate={24}/>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn ghost sm">Test sign-in</button>
          <button className="btn ghost sm" onClick={onRotate}>Rotate secret</button>
          <button className="btn danger sm" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </button>
          <div style={{flex:1}}/>
          <span className="chip" style={{height:18, fontSize:10}}>
            <span className="dot success"/>active
          </span>
        </div>
        {deleteError && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 6}}>{deleteError}</div>}
      </div>

      {/* Tabs */}
      <div className="row" style={{ borderBottom: '1px solid var(--hairline)', padding: '0 10px', gap: 2 }}>
        {[
          ['config', 'Config'],
          ['preview', 'Consent preview'],
          ['tokens', 'Tokens'],
          ['events', 'Events'],
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
        {tab === 'config' && <AppConfig app={app} onUpdate={onUpdate}/>}
        {tab === 'preview' && <ConsentPreview app={app}/>}
        {tab === 'tokens' && <AppTokens app={app}/>}
        {tab === 'events' && <AppEvents app={app}/>}
      </div>
      <CLIFooter command={`shark app show ${app.client_id || app.id}`}/>
    </aside>
  );
}

function AppConfig({ app, onUpdate }) {
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState(null);
  const [saveOk, setSaveOk] = React.useState(false);
  const [redirectInput, setRedirectInput] = React.useState('');
  const [corsInput, setCorsInput] = React.useState('');

  const redirectUris = app.redirect_uris || [];
  const corsOrigins = app.cors_origins || [];

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

  const removeRedirect = (uri) => {
    save({ redirect_uris: redirectUris.filter(u => u !== uri) });
  };

  const addRedirect = () => {
    const val = redirectInput.trim();
    if (!val) return;
    save({ redirect_uris: [...redirectUris, val] });
    setRedirectInput('');
  };

  const addCors = () => {
    const val = corsInput.trim();
    if (!val) return;
    save({ cors_origins: [...corsOrigins, val] });
    setCorsInput('');
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>
      {saveError && <div style={{color:'var(--danger)', fontSize: 11, padding: '6px 8px', background:'var(--surface-1)', borderRadius: 3}}>{saveError}</div>}
      {saveOk && <div style={{color:'var(--success)', fontSize: 11}}>Saved.</div>}

      <Section label="Redirect URIs" count={redirectUris.length}>
        <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
          {redirectUris.map((uri, i) => (
            <div key={i} className="row" style={{ padding: '7px 10px', gap: 8, borderBottom: i < redirectUris.length - 1 ? '1px solid var(--hairline)' : 0 }}>
              <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{uri}</span>
              {uri.includes('localhost') && <span className="chip" style={{height:15, fontSize:9}}>dev</span>}
              {uri.includes('*') && <span className="chip warn" style={{height:15, fontSize:9}}>wildcard</span>}
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

      <Section label="Allowed origins (CORS)" count={corsOrigins.length}>
        {corsOrigins.length ? (
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
            {corsOrigins.map((o, i) => (
              <div key={i} className="row" style={{ padding: '6px 10px', borderBottom: i < corsOrigins.length-1 ? '1px solid var(--hairline)' : 0 }}>
                <span className="mono" style={{fontSize: 11, flex: 1}}>{o}</span>
              </div>
            ))}
          </div>
        ) : <div className="faint" style={{fontSize: 11.5, padding: '6px 0'}}>No origins — Authorization Code flow only.</div>}
        <div className="row" style={{gap: 6, marginTop: 6}}>
          <input
            placeholder="https://…"
            value={corsInput}
            onChange={e => setCorsInput(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && addCors()}
            style={{flex:1, fontSize:11, padding:'4px 7px', border:'1px solid var(--hairline)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
          />
          <button className="btn ghost sm" onClick={addCors} disabled={saving || !corsInput.trim()}><Icon.Plus width={10} height={10}/> Add</button>
        </div>
      </Section>

      <Section label="Metadata">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">App ID</span>
          <span className="mono">{app.id}</span>
          <span className="faint">Created</span>
          <span className="mono">{relativeTime(app.created_at)}</span>
          <span className="faint">Last updated</span>
          <span className="mono">{relativeTime(app.updated_at)}</span>
        </div>
      </Section>
    </div>
  );
}

function ConsentPreview({ app }) {
  const scopes = [
    { scope: 'openid', desc: 'Sign you in and identify you' },
    { scope: 'profile', desc: 'Read your name and profile picture' },
    { scope: 'email', desc: 'Read your primary email address' },
  ];
  const redirectUris = app.redirect_uris || [];
  return (
    <div style={{ padding: 16 }}>
      <div className="faint" style={{fontSize: 11, marginBottom: 10}}>
        This is what users see when {app.name} requests authorization:
      </div>
      <div style={{
        background: '#fff', color: '#0a0a0a',
        borderRadius: 10, padding: 24,
        border: '1px solid var(--hairline-strong)',
        fontFamily: 'var(--font-sans)',
        boxShadow: '0 20px 60px rgba(0,0,0,0.5), 0 6px 18px rgba(0,0,0,0.25)',
      }}>
        <div style={{textAlign:'center', marginBottom: 20}}>
          <div style={{
            width: 48, height: 48, margin: '0 auto 10px',
            borderRadius: 10, background: '#111',
            display:'flex', alignItems:'center', justifyContent:'center',
            color:'#fff', fontWeight: 700, fontSize: 22,
          }}>{app.name[0]}</div>
          <div style={{fontSize: 15, fontWeight: 600, color:'#0a0a0a'}}>{app.name} wants to sign you in</div>
          <div style={{fontSize: 12, color:'#666', marginTop: 4}}>with your Shark account</div>
        </div>

        <div style={{background:'#f7f7f8', borderRadius: 6, padding: '10px 12px', marginBottom: 12, fontSize: 12, color:'#0a0a0a'}}>
          <div style={{fontWeight: 500, marginBottom: 6, fontSize: 11}}>{app.name} will be able to:</div>
          {scopes.map(s => (
            <div key={s.scope} style={{display: 'flex', gap: 8, padding: '4px 0', alignItems:'flex-start', fontSize: 11.5}}>
              <span style={{color:'#22a06b', fontWeight:600, marginTop: 1}}>✓</span>
              <div>
                <div style={{color:'#0a0a0a'}}>{s.desc}</div>
                <div style={{color: '#999', fontSize: 10, fontFamily:'var(--font-mono)', marginTop: 1}}>scope: {s.scope}</div>
              </div>
            </div>
          ))}
        </div>

        <div style={{display:'flex', gap: 6, marginTop: 16}}>
          <button style={{flex:1, padding:'9px 12px', background:'#f0f0f0', color:'#222', border:0, borderRadius: 6, fontSize: 12, fontWeight: 500}}>Cancel</button>
          <button style={{flex:1, padding:'9px 12px', background:'#000', color:'#fff', border:0, borderRadius: 6, fontSize: 12, fontWeight: 500}}>Allow</button>
        </div>

        <div style={{textAlign:'center', fontSize: 10, color:'#999', marginTop: 14, borderTop:'1px solid #eee', paddingTop: 10}}>
          Redirects to <span style={{fontFamily:'var(--font-mono)'}}>{redirectUris[0] || '(no redirect URIs)'}</span>
        </div>
      </div>

      <div className="faint" style={{fontSize: 11, marginTop: 14, padding: 10, background: 'var(--surface-1)', borderRadius: 2}}>
        <b style={{color:'var(--fg)'}}>Tip:</b> consent is skipped on subsequent grants unless new scopes are requested. Test with <span className="mono">?prompt=consent</span>.
      </div>
    </div>
  );
}

function AppTokens({ app }) {
  return (
    <div style={{padding: 16}}>
      <div className="faint" style={{fontSize: 11.5, padding: '20px 0', textAlign:'center'}}>
        Token listing not yet available via API.
      </div>
    </div>
  );
}

function SevDot({ sev }) {
  const c = sev === 'danger' ? 'var(--danger)' : sev === 'warn' ? 'var(--warn)' : 'var(--fg-dim)';
  return <span style={{width:6,height:6,borderRadius:1,background:c,flexShrink:0}}/>;
}

function AppEvents({ app }) {
  const { data, loading } = useAPI('/audit-logs?limit=20');
  const events = (data?.data || []).filter(e => e.action && (e.action.startsWith('app.') || e.action.startsWith('oauth.'))).slice(0, 10);
  return (
    <div style={{padding: 16}}>
      {loading ? (
        <div className="faint" style={{fontSize: 11}}>Loading…</div>
      ) : events.length === 0 ? (
        <div className="faint" style={{fontSize: 11}}>No app or OAuth events found.</div>
      ) : events.map(e => (
        <div key={e.id} className="row" style={{padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11.5}}>
          <SevDot sev={e.severity}/>
          <span className="mono" style={{flex: 1}}>{e.action}</span>
          <span className="faint">{e.actor}</span>
          <span className="faint mono" style={{fontSize: 10}}>{relativeTime(e.t || e.created_at)}</span>
        </div>
      ))}
    </div>
  );
}

function RotateSecretModal({ app, onClose, onRotate }) {
  const [stage, setStage] = React.useState('confirm'); // confirm → reveal
  const [newSecret, setNewSecret] = React.useState(null);
  const [rotating, setRotating] = React.useState(false);
  const [error, setError] = React.useState(null);

  const doRotate = async () => {
    setRotating(true);
    setError(null);
    try {
      const result = await onRotate();
      setNewSecret(result?.client_secret || result?.secret || '(see server logs)');
      setStage('reveal');
    } catch (e) {
      setError(e.message);
    } finally {
      setRotating(false);
    }
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        {stage === 'confirm' ? <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Rotate client secret</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
            A new secret will be issued for <b style={{color:'var(--fg)'}}>{app.name}</b>. The old secret continues to work for 24h (grace period). After that, any clients still using it will fail to authenticate.
          </p>
          <div style={{background:'var(--surface-1)', border:'1px solid var(--hairline)', padding: 10, borderRadius: 4, marginTop: 14, fontSize: 11.5}}>
            <div className="row" style={{gap: 8, marginBottom: 4}}>
              <span style={{color:'var(--warn)'}}>⚠</span>
              <b style={{color:'var(--fg)'}}>This cannot be undone.</b>
            </div>
            <div className="faint" style={{fontSize: 11}}>
              The new secret will be shown exactly once. Copy it to your secret store immediately.
            </div>
          </div>
          {error && <div style={{color:'var(--danger)', fontSize: 11, marginTop: 8}}>{error}</div>}
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn ghost" onClick={onClose}>Cancel</button>
            <button className="btn danger" onClick={doRotate} disabled={rotating}>
              {rotating ? 'Rotating…' : 'Rotate now'}
            </button>
          </div>
        </> : <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>New secret · copy it now</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6}}>This is the only time you'll see this value.</p>
          <div style={{background:'#000', color:'#fff', padding: 14, borderRadius: 4, marginTop: 14, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative'}}>
            {newSecret}
            <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'#fff', borderColor:'rgba(255,255,255,0.2)'}}
              onClick={() => navigator.clipboard.writeText(newSecret)}>
              <Icon.Copy width={10} height={10}/> Copy
            </button>
          </div>
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn primary" onClick={onClose}>I've saved it</button>
          </div>
        </>}
      </div>
    </div>
  );
}

function CreateAppSlideOver({ onClose, onCreate }) {
  const [name, setName] = React.useState('');
  const [redirectUris, setRedirectUris] = React.useState('');
  const [corsOrigins, setCorsOrigins] = React.useState('');
  const [creating, setCreating] = React.useState(false);
  const [error, setError] = React.useState(null);
  const [created, setCreated] = React.useState(null); // holds response with secret

  const handleCreate = async () => {
    if (!name.trim()) { setError('Name is required.'); return; }
    setCreating(true);
    setError(null);
    try {
      const payload = {
        name: name.trim(),
        redirect_uris: redirectUris.split('\n').map(s => s.trim()).filter(Boolean),
        cors_origins: corsOrigins.split('\n').map(s => s.trim()).filter(Boolean),
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
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{
        position:'fixed', top: 0, right: 0, bottom: 0, width: 520,
        background:'var(--surface-0)', borderLeft:'1px solid var(--hairline-bright)',
        display:'flex', flexDirection:'column', boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        <div className="row" style={{padding: '14px 16px', borderBottom:'1px solid var(--hairline)'}}>
          <h2 style={{margin:0, fontSize: 14, fontWeight: 600, flex:1}}>
            {created ? 'Application created' : 'New application'}
          </h2>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div style={{flex:1, overflowY:'auto', padding: 20}}>
          {created ? (
            <div style={{display:'flex', flexDirection:'column', gap: 16}}>
              <div className="faint" style={{fontSize: 12}}>
                Application <b style={{color:'var(--fg)'}}>{created.name || name}</b> has been created.
                {created.client_secret && ' Copy your client secret now — it won\'t be shown again.'}
              </div>
              <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
                <span className="faint">Client ID</span>
                <span className="mono">{created.client_id || created.id}</span>
              </div>
              {created.client_secret && (
                <div>
                  <div className="faint" style={{fontSize: 11, marginBottom: 6}}>Client Secret (shown once)</div>
                  <div style={{background:'#000', color:'#fff', padding: 14, borderRadius: 4, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative'}}>
                    {created.client_secret}
                    <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'#fff', borderColor:'rgba(255,255,255,0.2)'}}
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
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Application name <span style={{color:'var(--danger)'}}>*</span></label>
                <input
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="My App"
                  style={{width:'100%', boxSizing:'border-box', fontSize:12, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
                />
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>Redirect URIs <span className="faint">(one per line)</span></label>
                <textarea
                  value={redirectUris}
                  onChange={e => setRedirectUris(e.target.value)}
                  placeholder={"https://app.example.com/callback\nhttps://localhost:3000/callback"}
                  rows={4}
                  style={{width:'100%', boxSizing:'border-box', fontSize:11.5, fontFamily:'var(--font-mono)', padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none', resize:'vertical'}}
                />
              </div>
              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>CORS origins <span className="faint">(one per line, optional)</span></label>
                <textarea
                  value={corsOrigins}
                  onChange={e => setCorsOrigins(e.target.value)}
                  placeholder="https://app.example.com"
                  rows={3}
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
                {creating ? 'Creating…' : 'Create application'}
              </button>
            </>
          )}
        </div>
      </div>
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

