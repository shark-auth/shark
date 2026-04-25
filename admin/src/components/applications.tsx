// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { usePageActions } from './useKeyboardShortcuts'
import { TeachEmptyState } from './TeachEmptyState'
import { useTabParam } from './useURLParams'
import { ProxyWizard } from './proxy_wizard'

// Applications page — OAuth/OIDC client registrations
// Table view → slide-over detail → live consent-screen preview

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };

export function Applications() {
  const [selectedId, setSelectedId] = React.useState(null);
  const [tab, setTab] = useTabParam('config');
  const [query, setQuery] = React.useState('');
  const [rotateModal, setRotateModal] = React.useState(null);
  const [createOpen, setCreateOpen] = React.useState(false);

  const { data: appsRaw, loading, refresh } = useAPI('/admin/apps');
  const apps = appsRaw?.data || [];

  // 1. Sync URL -> State on mount and when apps load
  React.useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    
    // Handle "new" app flag
    if (p.get('new') === '1') {
      setCreateOpen(true);
      p.delete('new');
      window.history.replaceState(null, '', window.location.pathname + '?' + p.toString());
    }

    // Handle search query
    const s = p.get('search');
    if (s && !query) setQuery(s);

    // Initial selection from URL
    const aid = p.get('id') || p.get('app_id');
    if (aid && !selectedId) {
       setSelectedId(aid);
    }
  }, [apps.length]); // only run once apps load initially

  const selected = apps.find(a => a.id === selectedId || a.client_id === selectedId);

  // 2. Sync State -> URL
  React.useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    if (selectedId) {
      if (p.get('id') !== selectedId) {
        p.set('id', selectedId);
        window.history.replaceState(null, '', window.location.pathname + '?' + p.toString());
      }
    } else {
      if (p.has('id')) {
        p.delete('id');
        window.history.replaceState(null, '', window.location.pathname + '?' + p.toString());
      }
    }
  }, [selectedId]);

  usePageActions({ onNew: () => setCreateOpen(true), onRefresh: refresh });

  const filtered = apps.filter(a => {
    if (query && !(a.name.toLowerCase().includes(query.toLowerCase()) || a.client_id.includes(query))) return false;
    return true;
  });

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 560px' : '1fr', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', flexDirection: 'column', overflow: 'hidden', minWidth: 0 }}>
        {/* Header */}
        <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ gap: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Applications</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                Identity mesh · {apps.length} registered
              </p>
            </div>
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
          {loading && apps.length === 0 ? (
            <div className="faint" style={{padding: 20, fontSize: 12}}>Loading applications…</div>
          ) : apps.length === 0 ? (
            <TeachEmptyState
              icon="App"
              title="No applications registered"
              description="Applications are deployment units (Gateway proxies or SDK-based apps) that use SharkAuth."
              createLabel="New Application"
              onCreate={() => setCreateOpen(true)}
              cliSnippet="shark apps create --name 'My Service'"
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
                  <th style={appThStyle}>Model</th>
                  <th style={appThStyle}>Proxy Domain</th>
                  <th style={appThStyle}>Created</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map(a => (
                  <tr key={a.id}
                    onClick={() => setSelectedId(a.id)}
                    style={{
                      cursor:'pointer',
                      background: selectedId === a.id ? 'var(--surface-2)' : 'transparent',
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
                      <span className={"chip sm " + (a.integration_mode === 'proxy' ? 'success' : 'ghost')}>
                        {a.integration_mode === 'proxy' ? 'Proxy' : 'SDK'}
                      </span>
                    </td>
                    <td style={appTdStyle}>
                      {a.proxy_public_domain ? (
                        <span className="mono" style={{fontSize: 10.5}}>{a.proxy_public_domain}</span>
                      ) : <span className="faint">—</span>}
                    </td>
                    <td style={appTdStyle}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{relativeTime(a.created_at)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {selected && (
        <AppDetail
          app={selected}
          tab={tab} setTab={setTab}
          onClose={() => setSelectedId(null)}
          onRotate={() => setRotateModal(selected)}
          onDelete={async () => {
            await API.del('/admin/apps/' + selected.id);
            setSelectedId(null);
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
          <button className="btn ghost sm" onClick={onRotate}>Rotate secret</button>
          <button className="btn danger sm" onClick={handleDelete} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </button>
          <div style={{flex:1}}/>
          <span className="chip" style={{height:18, fontSize:10}}>
            <span className="dot success"/>active
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="row" style={{ borderBottom: '1px solid var(--hairline)', padding: '0 10px', gap: 2 }}>
        {[
          ['config', 'Config'],
          ['rules', 'Proxy rules'],
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
        {tab === 'rules' && <AppProxyRules app={app}/>}
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
  
  // Internal state for the form
  const [mode, setMode] = React.useState(app.integration_mode === 'proxy' ? 'proxy' : 'sdk');
  const [sdkSubMode, setSdkSubMode] = React.useState(app.integration_mode === 'proxy' ? 'hosted' : (app.integration_mode || 'custom'));
  const [proxyPublicDomain, setProxyPublicDomain] = React.useState(app.proxy_public_domain || '');
  const [proxyProtectedUrl, setProxyProtectedUrl] = React.useState(app.proxy_protected_url || '');

  const redirectUris = app.allowed_callback_urls || [];

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
    save({ allowed_callback_urls: redirectUris.filter(u => u !== uri) });
  };

  const addRedirect = () => {
    const val = redirectInput.trim();
    if (!val) return;
    save({ allowed_callback_urls: [...redirectUris, val] });
    setRedirectInput('');
  };

  const setIntegrationMode = (newMode) => {
    setMode(newMode);
    if (newMode === 'proxy') {
      save({ integration_mode: 'proxy' });
    } else {
      save({ integration_mode: sdkSubMode });
    }
  };

  const setSdkType = (type) => {
    setSdkSubMode(type);
    save({ integration_mode: type });
  };

  const saveProxy = () => {
    save({
      proxy_public_domain: proxyPublicDomain.trim(),
      proxy_protected_url: proxyProtectedUrl.trim(),
    });
  };

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 24 }}>
      {saveError && <div style={{color:'var(--danger)', fontSize: 11, padding: '6px 8px', background:'var(--surface-1)', borderRadius: 3, border: '1px solid var(--danger)'}}>{saveError}</div>}
      
      <Section label="Deployment Model">
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, background: 'var(--surface-1)', padding: 4, borderRadius: 6, border: '1px solid var(--hairline)' }}>
          <button 
            className={"btn sm " + (mode === 'proxy' ? 'primary' : 'ghost')}
            onClick={() => setIntegrationMode('proxy')}
            style={{ border: 0, fontWeight: mode === 'proxy' ? 600 : 400 }}
          >
            Proxy Gateway
          </button>
          <button 
            className={"btn sm " + (mode === 'sdk' ? 'primary' : 'ghost')}
            onClick={() => setIntegrationMode('sdk')}
            style={{ border: 0, fontWeight: mode === 'sdk' ? 600 : 400 }}
          >
            SDK / Library
          </button>
        </div>
        <div className="faint" style={{ fontSize: 10.5, marginTop: 10, textAlign: 'center', minHeight: 14 }}>
          {saving ? (
             <span className="row" style={{ justifyContent: 'center', gap: 6 }}>
               <Icon.Refresh width={10} style={{ animation: 'spin 800ms linear infinite' }}/> Saving deployment model…
             </span>
          ) : saveOk ? (
             <span style={{ color: 'var(--success)' }}>✓ Deployment model updated</span>
          ) : (
            mode === 'proxy' 
              ? "SharkAuth sits in front of your app. Zero code changes required." 
              : "Your application code calls SharkAuth for authentication."
          )}
        </div>
      </Section>

      {mode === 'proxy' ? (
        <Section label="Proxy Settings">
          <div style={{display:'flex', flexDirection:'column', gap: 12, background:'var(--surface-1)', padding: 16, borderRadius: 6, border:'1px solid var(--hairline)'}}>
            <div className="col" style={{gap: 4}}>
              <label style={{fontSize: 10.5, fontWeight: 500, color:'var(--fg-dim)'}}>Public Domain</label>
              <input
                placeholder="api.myapp.com"
                value={proxyPublicDomain}
                onChange={e => setProxyPublicDomain(e.target.value)}
                className="mono"
                style={{fontSize:11, padding:'6px 10px', border:'1px solid var(--hairline-strong)', borderRadius:4, background:'var(--surface-2)', color:'var(--fg)', outline:'none'}}
              />
              <div className="faint" style={{ fontSize: 9.5, marginTop: 2 }}>
                Local testing: Set to <code style={{background:'var(--surface-3)', padding:'1px 4px', borderRadius:2}}>localhost:8080</code> or test via <code style={{background:'var(--surface-3)', padding:'1px 4px', borderRadius:2}}>curl -H "Host: {proxyPublicDomain || 'api.myapp.com'}" http://localhost:8080</code>
              </div>
            </div>
            <div className="col" style={{gap: 4}}>
              <label style={{fontSize: 10.5, fontWeight: 500, color:'var(--fg-dim)'}}>Internal Protected URL</label>
              <input
                placeholder="http://localhost:3001"
                value={proxyProtectedUrl}
                onChange={e => setProxyProtectedUrl(e.target.value)}
                className="mono"
                style={{fontSize:11, padding:'6px 10px', border:'1px solid var(--hairline-strong)', borderRadius:4, background:'var(--surface-2)', color:'var(--fg)', outline:'none'}}
              />
            </div>
            <div className="row" style={{marginTop: 4, justifyContent: 'space-between', alignItems: 'center'}}>
              <button className="btn primary sm" onClick={saveProxy} disabled={saving || (proxyPublicDomain === app.proxy_public_domain && proxyProtectedUrl === app.proxy_protected_url)}>
                {saving ? 'Updating…' : 'Update Proxy'}
              </button>
              {saveOk && <span style={{ color: 'var(--success)', fontSize: 11 }}>✓ Proxy settings saved</span>}
            </div>
          </div>
        </Section>
      ) : (
        <>
          <Section label="SDK Integration Type">
             <div className="row" style={{ gap: 8, marginBottom: 12 }}>
                <button className={"btn sm " + (sdkSubMode === 'hosted' ? 'solid' : 'ghost')} onClick={() => setSdkType('hosted')}>Hosted Pages</button>
                <button className={"btn sm " + (sdkSubMode === 'components' ? 'solid' : 'ghost')} onClick={() => setSdkType('components')}>Components</button>
                <button className={"btn sm " + (sdkSubMode === 'custom' ? 'solid' : 'ghost')} onClick={() => setSdkType('custom')}>Custom API</button>
             </div>
             <div className="faint" style={{ fontSize: 11, lineHeight: 1.4, minHeight: 30 }}>
                {saving && sdkSubMode !== app.integration_mode ? (
                   <span className="row" style={{ gap: 6 }}><Icon.Refresh width={10} style={{ animation: 'spin 800ms linear infinite' }}/> Updating type…</span>
                ) : (
                  <>
                    {sdkSubMode === 'hosted' && "Use SharkAuth's pre-built login and user profile pages."}
                    {sdkSubMode === 'components' && "Embed SharkAuth React/Vue components directly into your UI."}
                    {sdkSubMode === 'custom' && "Build your own UI using the SharkAuth REST API."}
                  </>
                )}
             </div>
          </Section>

          <Section label="SDK Credentials">
             <div style={{ background: 'var(--surface-1)', padding: 12, borderRadius: 5, border: '1px solid var(--hairline)' }}>
               <div className="faint" style={{ fontSize: 11, marginBottom: 8 }}>Client ID</div>
               <CopyField value={app.client_id}/>
               <div className="faint" style={{ fontSize: 11, marginTop: 12, marginBottom: 4 }}>Secret Prefix</div>
               <span className="mono" style={{ fontSize: 12 }}>{app.client_secret_prefix}********</span>
             </div>
          </Section>

          <Section label="OAuth Redirect URIs" count={redirectUris.length}>
            <div style={{ border: '1px solid var(--hairline)', borderRadius: 4, background: 'var(--surface-1)', overflow: 'hidden' }}>
              {redirectUris.length === 0 ? (
                <div style={{ padding: '12px', fontSize: 11, color: 'var(--fg-dim)', textAlign: 'center' }}>No redirect URIs configured.</div>
              ) : redirectUris.map((uri, i) => (
                <div key={i} className="row" style={{ padding: '7px 10px', gap: 8, borderBottom: i < redirectUris.length - 1 ? '1px solid var(--hairline)' : 0 }}>
                  <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{uri}</span>
                  <button className="btn ghost icon sm" onClick={() => removeRedirect(uri)} disabled={saving}><Icon.X width={10} height={10}/></button>
                </div>
              ))}
            </div>
            <div className="row" style={{gap: 6, marginTop: 8}}>
              <input
                placeholder="https://…"
                value={redirectInput}
                onChange={e => setRedirectInput(e.target.value)}
                onKeyDown={e => e.key === 'Enter' && addRedirect()}
                style={{flex:1, fontSize:11, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:4, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
              />
              <button className="btn ghost sm" onClick={addRedirect} disabled={saving || !redirectInput.trim()}><Icon.Plus width={10} height={10}/> Add</button>
            </div>
          </Section>
        </>
      )}

      <Section label="Metadata">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">App ID</span>
          <span className="mono">{app.id}</span>
          <span className="faint">Created</span>
          <span className="mono">{relativeTime(app.created_at)}</span>
        </div>
      </Section>
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

function ConsentPreview({ app }) {
  const scopes = [
    { scope: 'openid', desc: 'Sign you in and identify you' },
    { scope: 'profile', desc: 'Read your name and profile picture' },
    { scope: 'email', desc: 'Read your primary email address' },
  ];
  const redirectUris = app.allowed_callback_urls || [];
  return (
    <div style={{ padding: 16 }}>
      <div style={{
        background: 'var(--surface-0)', color: 'var(--fg)',
        borderRadius: 0, padding: 24,
        border: '1px solid var(--hairline-strong)',
        fontFamily: 'var(--font-sans)',
        boxShadow: '0 20px 60px rgba(0,0,0,0.5), 0 6px 18px rgba(0,0,0,0.25)',
      }}>
        <div style={{textAlign:'center', marginBottom: 20}}>
          <div style={{
            width: 48, height: 48, margin: '0 auto 10px',
            borderRadius: 0, background: 'var(--surface-3)',
            display:'flex', alignItems:'center', justifyContent:'center',
            color:'var(--fg)', fontWeight: 700, fontSize: 22,
          }}>{app.name[0]}</div>
          <div style={{fontSize: 15, fontWeight: 600, color:'var(--fg)'}}>{app.name} wants to sign you in</div>
          <div style={{fontSize: 12, color:'var(--fg-muted)', marginTop: 4}}>with your Shark account</div>
        </div>

        <div style={{background:'var(--surface-1)', borderRadius: 0, padding: '10px 12px', marginBottom: 12, fontSize: 12, color:'var(--fg)'}}>
          <div style={{fontWeight: 500, marginBottom: 6, fontSize: 11}}>{app.name} will be able to:</div>
          {scopes.map(s => (
            <div key={s.scope} style={{display: 'flex', gap: 8, padding: '4px 0', alignItems:'flex-start', fontSize: 11.5}}>
              <span style={{color:'var(--success)', fontWeight:600, marginTop: 1}}>✓</span>
              <div>
                <div style={{color:'var(--fg)'}}>{s.desc}</div>
                <div style={{color: 'var(--fg-muted)', fontSize: 10, fontFamily:'var(--font-mono)', marginTop: 1}}>scope: {s.scope}</div>
              </div>
            </div>
          ))}
        </div>

        <div style={{display:'flex', gap: 6, marginTop: 16}}>
          <button style={{flex:1, padding:'9px 12px', background:'var(--surface-2)', color:'var(--fg)', border:'1px solid var(--hairline-strong)', borderRadius: 0, fontSize: 12, fontWeight: 500}}>Cancel</button>
          <button style={{flex:1, padding:'9px 12px', background:'var(--fg)', color:'var(--bg)', border:0, borderRadius: 0, fontSize: 12, fontWeight: 500}}>Allow</button>
        </div>

        <div style={{textAlign:'center', fontSize: 10, color:'var(--fg-muted)', marginTop: 14, borderTop:'1px solid var(--hairline)', paddingTop: 10}}>
          Redirects to <span style={{fontFamily:'var(--font-mono)'}}>{redirectUris[0] || '(no redirect URIs)'}</span>
        </div>
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
  const { data, loading } = useAPI('/admin/audit-logs?limit=20');
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
  const [proxyPublicDomain, setProxyPublicDomain] = React.useState('');
  const [proxyProtectedUrl, setProxyProtectedUrl] = React.useState('');
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
        allowed_callback_urls: redirectUris.split('\n').map(s => s.trim()).filter(Boolean),
        allowed_origins: corsOrigins.split('\n').map(s => s.trim()).filter(Boolean),
        proxy_public_domain: proxyPublicDomain.trim(),
        proxy_protected_url: proxyProtectedUrl.trim(),
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

              <div style={{padding:'12px 0', borderTop:HAIRLINE, borderBottom:HAIRLINE, display:'flex', flexDirection:'column', gap: 12}}>
                <div style={{fontSize: 11, fontWeight: 600, textTransform:'uppercase', letterSpacing:'0.05em', color:'var(--fg-muted)'}}>Proxy Mode (Codeless Auth)</div>
                <div>
                  <label style={{display:'block', fontSize: 11, fontWeight: 500, marginBottom: 4, color:'var(--fg-dim)'}}>Public Domain</label>
                  <input
                    value={proxyPublicDomain}
                    onChange={e => setProxyPublicDomain(e.target.value)}
                    placeholder="api.myapp.com"
                    className="mono"
                    style={{width:'100%', boxSizing:'border-box', fontSize:11.5, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
                  />
                  <div className="faint" style={{ fontSize: 9.5, marginTop: 4 }}>
                    Local testing: Set to <code style={{background:'var(--surface-2)', padding:'1px 4px', borderRadius:2}}>localhost:8080</code> or test via <code style={{background:'var(--surface-2)', padding:'1px 4px', borderRadius:2}}>curl -H "Host: {proxyPublicDomain || 'api.myapp.com'}" http://localhost:8080</code>
                  </div>
                </div>
                <div>
                  <label style={{display:'block', fontSize: 11, fontWeight: 500, marginBottom: 4, color:'var(--fg-dim)'}}>Protected Internal URL</label>
                  <input
                    value={proxyProtectedUrl}
                    onChange={e => setProxyProtectedUrl(e.target.value)}
                    placeholder="http://localhost:3001"
                    className="mono"
                    style={{width:'100%', boxSizing:'border-box', fontSize:11.5, padding:'6px 9px', border:'1px solid var(--hairline-strong)', borderRadius:3, background:'var(--surface-1)', color:'var(--fg)', outline:'none'}}
                  />
                </div>
              </div>

              <div>
                <label style={{display:'block', fontSize: 11.5, fontWeight: 500, marginBottom: 4}}>OAuth Redirect URIs <span className="faint">(one per line)</span></label>
                <textarea
                  value={redirectUris}
                  onChange={e => setRedirectUris(e.target.value)}
                  placeholder={"https://app.example.com/callback"}
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

const HAIRLINE = '1px solid var(--hairline)';

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

function AppProxyRules({ app }) {
  const { data, loading, refresh } = useAPI('/admin/proxy/rules/db?app_id=' + app.id, [app.id]);
  const rules = data?.data || [];
  const [creating, setCreating] = React.useState(false);

  const handleDelete = async (id) => {
    if (!confirm('Delete this rule?')) return;
    try {
      await API.del('/admin/proxy/rules/db/' + id);
      refresh();
    } catch (e) { alert(e.message); }
  };

  return (
    <div style={{ padding: 16 }}>
      <div className="row" style={{ marginBottom: 12, gap: 10 }}>
        <div style={{ flex: 1 }}>
          <h4 style={{ ...sectionLabelStyle, margin: 0 }}>DB-backed route rules</h4>
          <p className="faint" style={{ fontSize: 11, marginTop: 4 }}>
            Rules for {app.proxy_public_domain || 'this app'} ordered by priority.
          </p>
        </div>
        <button className="btn primary sm" onClick={() => setCreating(true)}>
          <Icon.Plus width={11} height={11}/> Add rule
        </button>
      </div>

      {loading ? (
        <div className="faint" style={{ fontSize: 11 }}>Loading rules…</div>
      ) : rules.length === 0 ? (
        <div style={{ padding: 32, textAlign: 'center', border: HAIRLINE, borderRadius: 5, background: 'var(--surface-1)' }}>
          <div className="muted" style={{ fontSize: 13 }}>No custom rules for this app.</div>
        </div>
      ) : (
        <div style={{ border: HAIRLINE, borderRadius: 5, overflow: 'hidden', background: 'var(--surface-1)' }}>
          <table className="tbl" style={{ width: '100%', fontSize: 11.5 }}>
            <thead>
              <tr style={{ background: 'var(--surface-2)' }}>
                <th style={{ padding: '6px 10px', textAlign: 'left' }}>Pattern</th>
                <th style={{ padding: '6px 10px', textAlign: 'left' }}>Policy</th>
                <th style={{ padding: '6px 10px', width: 40 }}></th>
              </tr>
            </thead>
            <tbody>
              {rules.map(r => (
                <tr key={r.id} style={{ borderTop: HAIRLINE }}>
                  <td style={{ padding: '8px 10px' }}>
                    <div className="mono" style={{ fontWeight: 500 }}>{r.pattern}</div>
                    <div className="faint" style={{ fontSize: 10, marginTop: 2 }}>
                      {r.methods?.length > 0 ? r.methods.join(', ') : 'ANY'}
                    </div>
                  </td>
                  <td style={{ padding: '8px 10px' }}>
                    <span className={"chip " + (r.require ? 'solid' : 'ghost')} style={{ height: 18, fontSize: 10 }}>
                      {r.require || r.allow}
                    </span>
                  </td>
                  <td style={{ padding: '8px 10px', textAlign: 'right' }}>
                    <button className="btn ghost icon sm" onClick={() => handleDelete(r.id)}>
                      <Icon.X width={10} height={10}/>
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {creating && (
        <div style={modalBackdrop} onClick={() => setCreating(false)}>
          <div style={{ width: 500 }} onClick={e => e.stopPropagation()}>
            <ProxyWizard 
              appId={app.id} 
              onComplete={() => { setCreating(false); refresh(); }}
            />
          </div>
        </div>
      )}
    </div>
  );
}
