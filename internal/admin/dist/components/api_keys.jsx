// API Keys page — M2M credential management
// Table + detail w/ rotation timeline + create/rotate modals + scope matrix

function ApiKeys() {
  const [selected, setSelected] = React.useState(null);
  const [filter, setFilter] = React.useState('all'); // all/active/revoked/expiring
  const [env, setEnv] = React.useState('all');
  const [query, setQuery] = React.useState('');
  const [createOpen, setCreateOpen] = React.useState(false);
  const [rotateModal, setRotateModal] = React.useState(null);

  const keys = MOCK.apiKeys;
  const filtered = keys.filter(k => {
    if (filter === 'active' && k.status !== 'active') return false;
    if (filter === 'revoked' && k.status !== 'revoked') return false;
    if (filter === 'expiring' && !k.expiringSoon) return false;
    if (env !== 'all' && k.env !== env) return false;
    if (query && !(k.name.toLowerCase().includes(query.toLowerCase()) || k.fingerprint.includes(query))) return false;
    return true;
  });

  const stats = {
    active: keys.filter(k => k.status === 'active').length,
    revoked: keys.filter(k => k.status === 'revoked').length,
    expiring: keys.filter(k => k.expiringSoon).length,
    reqs24h: keys.reduce((a, k) => a + k.requests24h, 0),
  };

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 480px' : '1fr', height: '100%', overflow: 'hidden' }}>
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
            <input placeholder="Filter by name or fingerprint…"
              value={query} onChange={e => setQuery(e.target.value)}
              style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12 }}/>
          </div>

          <div className="seg" style={segStyle}>
            {[['all','All'],['active','Active'],['expiring','Expiring'],['revoked','Revoked']].map(([v,l]) => (
              <button key={v} onClick={() => setFilter(v)}
                style={{...segBtn, background: filter===v ? '#fafafa':'var(--surface-2)', color: filter===v ? '#000':'var(--fg-muted)'}}>
                {l}
              </button>
            ))}
          </div>

          <div className="seg" style={segStyle}>
            {[['all','Any env'],['prod','prod'],['test','test']].map(([v,l]) => (
              <button key={v} onClick={() => setEnv(v)}
                className={v !== 'all' ? 'mono' : ''}
                style={{...segBtn, background: env===v ? '#fafafa':'var(--surface-2)', color: env===v ? '#000':'var(--fg-muted)'}}>
                {l}
              </button>
            ))}
          </div>

          <div style={{flex:1}}/>
          <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {keys.length}</span>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={kThStyle}>Name</th>
                <th style={kThStyle}>Fingerprint</th>
                <th style={kThStyle}>Env</th>
                <th style={kThStyle}>Scopes</th>
                <th style={{...kThStyle, textAlign: 'right'}}>Req / 24h</th>
                <th style={kThStyle}>Last used</th>
                <th style={kThStyle}>Expires</th>
                <th style={kThStyle}>Created by</th>
                <th style={{...kThStyle, width: 40}}/>
              </tr>
            </thead>
            <tbody>
              {filtered.map(k => (
                <tr key={k.id} onClick={() => setSelected(k)}
                  style={{
                    cursor: 'pointer',
                    background: selected?.id === k.id ? 'var(--surface-2)' : 'transparent',
                    opacity: k.status === 'revoked' ? 0.55 : 1,
                  }}>
                  <td style={kTdStyle}>
                    <div style={{ fontWeight: 500 }}>{k.name}</div>
                    {k.warn && <div className="row" style={{gap:4, marginTop: 2}}>
                      <span className="dot warn"/>
                      <span className="faint" style={{fontSize: 10.5}}>{k.warn}</span>
                    </div>}
                  </td>
                  <td style={kTdStyle}>
                    <span className="mono" style={{fontSize: 11}}>{k.fingerprint}</span>
                  </td>
                  <td style={kTdStyle}>
                    <span className={"chip " + (k.env === 'prod' ? 'danger' : '')} style={{height:15, fontSize:9, fontFamily:'var(--font-mono)'}}>
                      {k.env}
                    </span>
                  </td>
                  <td style={kTdStyle}>
                    <div className="row" style={{gap:3, flexWrap:'wrap', maxWidth: 180}}>
                      {k.scopes.slice(0,2).map(s => (
                        <span key={s} className="chip mono" style={{height:15, fontSize:9, padding:'0 4px'}}>{s}</span>
                      ))}
                      {k.scopes.length > 2 && <span className="faint" style={{fontSize:10}}>+{k.scopes.length-2}</span>}
                    </div>
                  </td>
                  <td style={{...kTdStyle, textAlign: 'right'}}>
                    <ReqSparkline count={k.requests24h}/>
                  </td>
                  <td style={kTdStyle}>
                    <span className="mono faint" style={{fontSize: 10.5}}>{MOCK.relativeTime(k.lastUsed)}</span>
                  </td>
                  <td style={kTdStyle}>
                    {k.status === 'revoked' ? (
                      <span className="chip danger" style={{height:15, fontSize:9}}>revoked</span>
                    ) : k.expiringSoon ? (
                      <span className="row" style={{gap:5}}>
                        <span className="dot warn"/>
                        <span className="mono" style={{fontSize: 10.5, color:'var(--warn)'}}>{MOCK.relativeTime(k.expires)}</span>
                      </span>
                    ) : (
                      <span className="mono faint" style={{fontSize: 10.5}}>{MOCK.relativeTime(k.expires)}</span>
                    )}
                  </td>
                  <td style={kTdStyle}>
                    <span className="faint" style={{fontSize: 11}}>{k.creator.split('@')[0]}</span>
                  </td>
                  <td style={kTdStyle}>
                    <button className="btn ghost icon sm" onClick={(e) => { e.stopPropagation(); setRotateModal(k); }} title="Rotate">
                      <Icon.Signing width={11} height={11}/>
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Footer */}
        <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
          <Icon.Debug width={11} height={11} style={{opacity:0.5}}/>
          <span className="faint">CLI parity:</span>
          <span className="mono faint">shark keys list --env={env}</span>
          <div style={{flex:1}}/>
          <span className="faint mono">POST /v1/api-keys</span>
        </div>
      </div>

      {selected && <KeyDetail k={selected} onClose={() => setSelected(null)} onRotate={() => setRotateModal(selected)}/>}
      {createOpen && <CreateKeyModal onClose={() => setCreateOpen(false)}/>}
      {rotateModal && <RotateKeyModal k={rotateModal} onClose={() => setRotateModal(null)}/>}
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

function KeyDetail({ k, onClose, onRotate }) {
  // Fake rotation timeline: most recent, then older versions
  const rotations = [
    { id: 'v3', created: k.created, retired: null, fingerprint: k.fingerprint, current: true },
    { id: 'v2', created: k.created + 1000, retired: k.created, fingerprint: 'sk_live_…old2x' },
    { id: 'v1', created: k.created - 80*86400e3, retired: k.created, fingerprint: 'sk_live_…f1rst' },
  ];

  const allScopes = ['users:read', 'users:write', 'orgs:read', 'orgs:write', 'audit:export', 'webhooks:manage', 'agents:manage', 'billing:write', 'billing:read', 'metrics:read'];

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)', background: 'var(--surface-0)',
      display:'flex', flexDirection:'column', overflow:'hidden',
    }}>
      <div style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ gap: 10 }}>
          <div style={{
            width: 28, height: 28, background: '#000', borderRadius: 4,
            display:'flex', alignItems:'center', justifyContent:'center',
          }}>
            <Icon.Key width={14} height={14} style={{color:'#fff'}}/>
          </div>
          <div style={{flex:1, minWidth:0}}>
            <div style={{fontWeight: 500, fontSize: 14}}>{k.name}</div>
            <div className="row" style={{gap: 6, fontSize: 10.5, marginTop: 2}}>
              <span className="mono faint">{k.fingerprint}</span>
              <CopyField value={k.fingerprint} truncate={20}/>
              <span className="chip" style={{height:15, fontSize:9, fontFamily:'var(--font-mono)'}}>{k.env}</span>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn ghost sm" onClick={onRotate}><Icon.Signing width={11} height={11}/> Rotate</button>
          <button className="btn ghost sm">Edit scopes</button>
          <button className="btn ghost sm" style={{color:'var(--danger)'}}>Revoke</button>
          <div style={{flex:1}}/>
          {k.status === 'active' ? (
            <span className="chip success" style={{height:18, fontSize:10}}><span className="dot success"/>active</span>
          ) : <span className="chip danger" style={{height:18, fontSize:10}}>revoked</span>}
        </div>
      </div>

      <div style={{flex:1, overflowY:'auto', padding: 16, display:'flex', flexDirection:'column', gap: 20}}>
        {/* Metadata */}
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">Created by</span>
          <span>{k.creator} <span className="faint">· {MOCK.relativeTime(k.created)}</span></span>
          <span className="faint">Expires</span>
          <span>
            <span className="mono">{new Date(Date.now() - k.expires*0 - (Date.now() - (Date.now() + (-k.expires))) ).toLocaleDateString()}</span>
            <span className="faint"> · {MOCK.relativeTime(k.expires)}</span>
            {k.expiringSoon && <span className="chip warn" style={{height:15, fontSize:9, marginLeft:6}}>expiring soon</span>}
          </span>
          <span className="faint">Last used</span>
          <span>
            <span className="mono">{MOCK.relativeTime(k.lastUsed)}</span>
            <span className="faint" style={{marginLeft: 8, fontSize: 10.5}}>from 3.142.88.22 · AWS us-east-1</span>
          </span>
          <span className="faint">Requests 24h</span>
          <span className="mono">{k.requests24h.toLocaleString()}</span>
        </div>

        {/* Rotation timeline */}
        <div>
          <h4 style={sectionLabelStyle}>Rotation history</h4>
          <div style={{ position: 'relative' }}>
            <div style={{ position: 'absolute', left: 7, top: 10, bottom: 10, width: 1, background: 'var(--hairline-strong)' }}/>
            {rotations.map((r, i) => (
              <div key={r.id} className="row" style={{ gap: 10, padding: '6px 0', alignItems: 'flex-start' }}>
                <div style={{
                  width: 14, height: 14, borderRadius: 14,
                  background: r.current ? '#fff' : 'var(--surface-3)',
                  border: '2px solid ' + (r.current ? '#fff' : 'var(--hairline-strong)'),
                  zIndex: 1, marginTop: 2,
                  boxShadow: r.current ? '0 0 0 3px rgba(255,255,255,0.1)' : 'none',
                }}/>
                <div style={{flex: 1, fontSize: 11.5}}>
                  <div className="row" style={{gap: 6}}>
                    <span className="mono">{r.fingerprint}</span>
                    {r.current && <span className="chip success" style={{height:14, fontSize:9}}>current</span>}
                    {!r.current && <span className="chip" style={{height:14, fontSize:9}}>retired</span>}
                  </div>
                  <div className="faint mono" style={{fontSize: 10, marginTop: 2}}>
                    created {MOCK.relativeTime(r.created)}{r.retired && ` · retired ${MOCK.relativeTime(r.retired)}`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Scope matrix */}
        <div>
          <h4 style={sectionLabelStyle}>Scopes</h4>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 6 }}>
            {allScopes.map(s => {
              const granted = k.scopes.includes(s);
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

        {/* Recent events */}
        <div>
          <h4 style={sectionLabelStyle}>Recent requests</h4>
          <div style={{border:'1px solid var(--hairline)', borderRadius:3, background:'var(--surface-1)', fontSize: 10.5, fontFamily:'var(--font-mono)'}}>
            {[
              ['GET', '/v1/users', '200', '12ms', MOCK.relativeTime(Date.now() - 30e3)],
              ['POST', '/v1/users', '201', '89ms', MOCK.relativeTime(Date.now() - 180e3)],
              ['GET', '/v1/orgs/org_nimbus/members', '200', '34ms', MOCK.relativeTime(Date.now() - 420e3)],
              ['GET', '/v1/audit?since=24h', '200', '220ms', MOCK.relativeTime(Date.now() - 1200e3)],
              ['POST', '/v1/webhooks/…/retry', '202', '18ms', MOCK.relativeTime(Date.now() - 1800e3)],
            ].map((row, i) => (
              <div key={i} className="row" style={{padding: '5px 10px', gap: 10, borderBottom: i < 4 ? '1px solid var(--hairline)' : 0}}>
                <span style={{width: 36, color: row[0] === 'GET' ? 'var(--fg-dim)' : 'var(--fg)', fontWeight: 500}}>{row[0]}</span>
                <span style={{flex: 1}}>{row[1]}</span>
                <span style={{color: row[2].startsWith('2') ? 'var(--success)' : 'var(--warn)', width: 32}}>{row[2]}</span>
                <span className="faint" style={{width: 48}}>{row[3]}</span>
                <span className="faint">{row[4]}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </aside>
  );
}

function CreateKeyModal({ onClose }) {
  const [stage, setStage] = React.useState('config');
  const [name, setName] = React.useState('');
  const [env, setEnv] = React.useState('test');
  const [selectedScopes, setSelectedScopes] = React.useState(new Set(['users:read']));
  const [expires, setExpires] = React.useState('90d');

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

  const newKey = 'sk_live_' + Math.random().toString(36).slice(2, 14) + Math.random().toString(36).slice(2, 14);

  const toggle = (id) => {
    const next = new Set(selectedScopes);
    if (next.has(id)) next.delete(id); else next.add(id);
    setSelectedScopes(next);
  };

  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 540, maxHeight: '80vh', overflow:'auto'}} onClick={e => e.stopPropagation()}>
        {stage === 'config' ? <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>New API key</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6}}>Create a server-to-server credential with scoped access.</p>

          <div style={{marginTop: 16}}>
            <label style={labelStyle}>Name</label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. Production billing sync" style={inputStyle}/>
            <div className="faint" style={{fontSize: 10.5, marginTop: 4}}>Descriptive — shown in audit logs and dashboards.</div>
          </div>

          <div style={{marginTop: 14}}>
            <label style={labelStyle}>Environment</label>
            <div className="seg" style={{...segStyle, width: '100%'}}>
              {[['test','test'],['prod','prod']].map(([v,l]) => (
                <button key={v} onClick={() => setEnv(v)} className="mono"
                  style={{...segBtn, flex:1, padding: '6px 10px', background: env===v ? '#fafafa':'var(--surface-2)', color: env===v ? '#000':'var(--fg-muted)'}}>
                  {l}
                </button>
              ))}
            </div>
          </div>

          <div style={{marginTop: 14}}>
            <label style={labelStyle}>Expires in</label>
            <div className="seg" style={{...segStyle, width: '100%'}}>
              {[['30d','30d'],['90d','90d'],['1y','1y'],['never','never']].map(([v,l]) => (
                <button key={v} onClick={() => setExpires(v)}
                  style={{...segBtn, flex:1, padding: '6px 10px', background: expires===v ? '#fafafa':'var(--surface-2)', color: expires===v ? '#000':'var(--fg-muted)'}}>
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

          <div className="row" style={{marginTop: 20, justifyContent:'flex-end', gap: 8}}>
            <button className="btn ghost" onClick={onClose}>Cancel</button>
            <button className="btn primary" onClick={() => setStage('reveal')} disabled={!name}>Create key</button>
          </div>
        </> : <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Your new API key</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6}}>Copy this value now. After closing, only the fingerprint is retrievable.</p>
          <div style={{background:'#000', color:'#fff', padding: 14, borderRadius: 4, marginTop: 14, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative'}}>
            {newKey}
            <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'#fff', borderColor:'rgba(255,255,255,0.2)'}}>
              <Icon.Copy width={10} height={10}/> Copy
            </button>
          </div>
          <div style={{marginTop: 14, padding: 10, background:'var(--surface-1)', border:'1px solid var(--hairline)', borderRadius: 3, fontSize: 11}}>
            <div className="row" style={{gap: 6, marginBottom: 6}}>
              <Icon.Debug width={11} height={11}/>
              <b>Use it:</b>
            </div>
            <pre style={{margin: 0, fontFamily:'var(--font-mono)', fontSize: 10.5, color:'var(--fg-muted)'}}>{`curl https://api.shark.sh/v1/users \\
  -H "Authorization: Bearer ${newKey.slice(0, 18)}…"`}</pre>
          </div>
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn primary" onClick={onClose}>I've saved it</button>
          </div>
        </>}
      </div>
    </div>
  );
}

function RotateKeyModal({ k, onClose }) {
  const [stage, setStage] = React.useState('confirm');
  const [grace, setGrace] = React.useState('24h');
  const newKey = 'sk_live_' + Math.random().toString(36).slice(2, 14) + Math.random().toString(36).slice(2, 14);
  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{...modalCard, width: 480}} onClick={e => e.stopPropagation()}>
        {stage === 'confirm' ? <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>Rotate {k.name}</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6, lineHeight: 1.5}}>
            Issue a new key. The old key ({k.fingerprint}) will continue working during the grace period, then stop.
          </p>
          <div style={{marginTop: 14}}>
            <label style={labelStyle}>Grace period</label>
            <div className="seg" style={{...segStyle, width: '100%'}}>
              {[['1h','1h'],['24h','24h'],['7d','7d'],['0','Immediate']].map(([v,l]) => (
                <button key={v} onClick={() => setGrace(v)}
                  style={{...segBtn, flex:1, padding: '6px 10px', background: grace===v ? '#fafafa':'var(--surface-2)', color: grace===v ? '#000':'var(--fg-muted)'}}>
                  {l}
                </button>
              ))}
            </div>
            {grace === '0' && <div className="faint" style={{fontSize: 10.5, marginTop: 4, color:'var(--danger)'}}>⚠ All in-flight requests with old key will fail.</div>}
          </div>
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn ghost" onClick={onClose}>Cancel</button>
            <button className="btn primary" onClick={() => setStage('reveal')}>Rotate</button>
          </div>
        </> : <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>New key issued</h2>
          <div style={{background:'#000', color:'#fff', padding: 14, borderRadius: 4, marginTop: 14, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative'}}>
            {newKey}
            <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'#fff', borderColor:'rgba(255,255,255,0.2)'}}>
              <Icon.Copy width={10} height={10}/> Copy
            </button>
          </div>
          <div className="faint mono" style={{fontSize: 10.5, marginTop: 10}}>
            old key {k.fingerprint} expires in {grace === '0' ? 'now' : grace}
          </div>
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end'}}>
            <button className="btn primary" onClick={onClose}>Done</button>
          </div>
        </>}
      </div>
    </div>
  );
}

const kThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const kTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const segStyle = { height: 28, display: 'inline-flex', border: '1px solid var(--hairline-strong)', borderRadius: 3, overflow: 'hidden' };
const segBtn = { padding: '0 10px', height: 28, fontSize: 11, borderRight: '1px solid var(--hairline)' };
const labelStyle = { display: 'block', fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--fg-dim)', fontWeight: 500, marginBottom: 5 };
const inputStyle = { width: '100%', padding: '7px 10px', background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 3, color: 'var(--fg)', fontSize: 12, outline: 'none' };

Object.assign(window, { ApiKeys });
