// Applications page — OAuth/OIDC client registrations
// Table view → slide-over detail → live consent-screen preview

function Applications() {
  const [selected, setSelected] = React.useState(null);
  const [tab, setTab] = React.useState('config');
  const [filter, setFilter] = React.useState('all');
  const [query, setQuery] = React.useState('');
  const [rotateModal, setRotateModal] = React.useState(null);
  const [createOpen, setCreateOpen] = React.useState(false);

  const apps = MOCK.applications;
  const filtered = apps.filter(a => {
    if (filter === 'first' && a.type !== 'first-party') return false;
    if (filter === 'third' && a.type !== 'third-party') return false;
    if (query && !(a.name.toLowerCase().includes(query.toLowerCase()) || a.clientId.includes(query))) return false;
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
                OAuth/OIDC clients · {apps.length} registered · {apps.reduce((a,b)=>a+b.tokensIssued24h,0).toLocaleString()} tokens/24h
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
          <div className="seg" style={{height:28, display:'inline-flex', border:'1px solid var(--hairline-strong)', borderRadius:3, overflow:'hidden'}}>
            {[['all','All'],['first','First-party'],['third','Third-party']].map(([v,l]) => (
              <button key={v} onClick={() => setFilter(v)}
                style={{padding:'0 10px', height:28, fontSize:11, background: filter===v ? '#fafafa':'var(--surface-2)', color: filter===v ? '#000':'var(--fg-muted)', borderRight:'1px solid var(--hairline)'}}>
                {l}
              </button>
            ))}
          </div>
          <div style={{flex:1}}/>
          <span className="faint mono" style={{fontSize:11}}>{filtered.length} / {apps.length}</span>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={{...appThStyle, width: 28}}/>
                <th style={appThStyle}>Name</th>
                <th style={appThStyle}>Client ID</th>
                <th style={appThStyle}>Type</th>
                <th style={appThStyle}>Redirects</th>
                <th style={appThStyle}>Grants</th>
                <th style={{...appThStyle, textAlign:'right'}}>Tokens / 24h</th>
                <th style={appThStyle}>Last used</th>
                <th style={appThStyle}>Secret rotated</th>
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
                    <div className="faint" style={{fontSize: 10.5, marginTop: 1}}>{a.owner}</div>
                  </td>
                  <td style={appTdStyle}>
                    <span className="mono faint" style={{fontSize: 10.5}}>{a.clientId}</span>
                  </td>
                  <td style={appTdStyle}>
                    <span className="chip" style={{height:16, fontSize:9.5}}>
                      {a.type === 'first-party' ? '1st-party' : '3rd-party'}
                    </span>
                  </td>
                  <td style={appTdStyle}>
                    <span className="mono faint" style={{fontSize: 10.5}}>
                      {a.redirects.length} URI{a.redirects.length !== 1 && 's'}
                    </span>
                  </td>
                  <td style={appTdStyle}>
                    <div className="row" style={{gap:3, flexWrap:'wrap'}}>
                      {a.grants.slice(0,2).map(g => (
                        <span key={g} className="chip mono" style={{height:15, fontSize:9, padding:'0 4px'}}>{g.replace('authorization_','ac_').replace('client_','cl_').replace('refresh_','rt')}</span>
                      ))}
                      {a.grants.length > 2 && <span className="faint" style={{fontSize:10}}>+{a.grants.length-2}</span>}
                    </div>
                  </td>
                  <td style={{...appTdStyle, textAlign:'right'}}>
                    <span className="mono" style={{fontSize: 11.5}}>{a.tokensIssued24h.toLocaleString()}</span>
                  </td>
                  <td style={appTdStyle}>
                    <span className="mono faint" style={{fontSize: 10.5}}>{MOCK.relativeTime(a.lastUsed)}</span>
                  </td>
                  <td style={appTdStyle}>
                    <span className="row" style={{gap:5}}>
                      <span className="mono faint" style={{fontSize: 10.5}}>{MOCK.relativeTime(a.secretRotated)}</span>
                      {a.warn && <span className="dot warn" title={a.warn}/>}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* CLI parity footer */}
        <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 10.5, gap: 10 }}>
          <Icon.Debug width={11} height={11} style={{opacity:0.5}}/>
          <span className="faint">CLI parity:</span>
          <span className="mono faint">shark apps list --type={filter === 'all' ? '*' : filter}</span>
          <div style={{flex:1}}/>
          <span className="faint mono">POST /v1/oauth/clients</span>
        </div>
      </div>

      {selected && (
        <AppDetail
          app={selected}
          tab={tab} setTab={setTab}
          onClose={() => setSelected(null)}
          onRotate={() => setRotateModal(selected)}
        />
      )}

      {rotateModal && <RotateSecretModal app={rotateModal} onClose={() => setRotateModal(null)}/>}
      {createOpen && <CreateAppSlideOver onClose={() => setCreateOpen(false)}/>}
    </div>
  );
}

function AppDetail({ app, tab, setTab, onClose, onRotate }) {
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
              <span className="mono faint">{app.clientId}</span>
              <CopyField value={app.clientId} truncate={24}/>
            </div>
          </div>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>

        <div className="row" style={{ gap: 6, marginTop: 10 }}>
          <button className="btn ghost sm">Test sign-in</button>
          <button className="btn ghost sm" onClick={onRotate}>Rotate secret</button>
          <button className="btn ghost sm">Disable</button>
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
        {tab === 'config' && <AppConfig app={app}/>}
        {tab === 'preview' && <ConsentPreview app={app}/>}
        {tab === 'tokens' && <AppTokens app={app}/>}
        {tab === 'events' && <AppEvents app={app}/>}
      </div>
    </aside>
  );
}

function AppConfig({ app }) {
  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 20 }}>
      <Section label="Redirect URIs" count={app.redirects.length}>
        <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
          {app.redirects.map((uri, i) => (
            <div key={i} className="row" style={{ padding: '7px 10px', gap: 8, borderBottom: i < app.redirects.length - 1 ? '1px solid var(--hairline)' : 0 }}>
              <span className="mono" style={{ fontSize: 11, flex: 1, wordBreak: 'break-all' }}>{uri}</span>
              {uri.includes('localhost') && <span className="chip" style={{height:15, fontSize:9}}>dev</span>}
              {uri.includes('*') && <span className="chip warn" style={{height:15, fontSize:9}}>wildcard</span>}
              <button className="btn ghost icon sm"><Icon.X width={10} height={10}/></button>
            </div>
          ))}
        </div>
        <button className="btn ghost sm" style={{marginTop: 6}}><Icon.Plus width={10} height={10}/> Add URI</button>
      </Section>

      <Section label="Post-logout redirects">
        {app.logoutUris.length ? (
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
            {app.logoutUris.map((u, i) => (
              <div key={i} style={{ padding: '7px 10px', borderBottom: i < app.logoutUris.length-1 ? '1px solid var(--hairline)' : 0 }}>
                <span className="mono" style={{fontSize: 11}}>{u}</span>
              </div>
            ))}
          </div>
        ) : <div className="faint" style={{fontSize: 11.5, padding: '6px 0'}}>No logout URIs configured.</div>}
      </Section>

      <Section label="Allowed origins (CORS)" count={app.origins.length}>
        {app.origins.length ? (
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
            {app.origins.map((o, i) => (
              <div key={i} className="row" style={{ padding: '6px 10px', borderBottom: i < app.origins.length-1 ? '1px solid var(--hairline)' : 0 }}>
                <span className="mono" style={{fontSize: 11, flex: 1}}>{o}</span>
              </div>
            ))}
          </div>
        ) : <div className="faint" style={{fontSize: 11.5, padding: '6px 0'}}>No origins — Authorization Code flow only.</div>}
      </Section>

      <Section label="Grants & security">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">Grants</span>
          <div className="row" style={{gap: 4, flexWrap:'wrap'}}>
            {app.grants.map(g => <span key={g} className="chip mono" style={{height:17, fontSize:10}}>{g}</span>)}
          </div>
          <span className="faint">PKCE</span>
          <span><span className={"chip " + (app.pkce === 'required' ? 'success' : '')} style={{height:17,fontSize:10}}>{app.pkce}</span></span>
          <span className="faint">Token endpoint auth</span>
          <span className="mono">{app.type === 'first-party' ? 'client_secret_post' : 'client_secret_basic'}</span>
          <span className="faint">Secret rotated</span>
          <span>
            <span className="mono">{MOCK.relativeTime(app.secretRotated)}</span>
            {app.warn && <span className="chip warn" style={{height:16,fontSize:9.5,marginLeft:6}}>{app.warn}</span>}
          </span>
        </div>
      </Section>

      <Section label="Signing & tokens">
        <div style={{display:'grid', gridTemplateColumns:'auto 1fr', gap:'8px 14px', fontSize: 11.5}}>
          <span className="faint">ID token signing</span>
          <span className="mono">RS256</span>
          <span className="faint">Access token TTL</span>
          <span className="mono">1h</span>
          <span className="faint">Refresh token TTL</span>
          <span className="mono">30d · rotate on use</span>
          <span className="faint">Current signing kid</span>
          <span className="mono">2026-04-b</span>
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
          Redirects to <span style={{fontFamily:'var(--font-mono)'}}>{app.redirects[0]}</span>
        </div>
      </div>

      <div className="faint" style={{fontSize: 11, marginTop: 14, padding: 10, background: 'var(--surface-1)', borderLeft: '2px solid var(--accent, var(--fg-dim))', borderRadius: 2}}>
        <b style={{color:'var(--fg)'}}>Tip:</b> consent is skipped on subsequent grants unless new scopes are requested. Test with <span className="mono">?prompt=consent</span>.
      </div>
    </div>
  );
}

function AppTokens({ app }) {
  const rows = [
    { jti: '8f9a2b', user: 'amelia@nimbus.sh', issued: Date.now() - 120e3, exp: Date.now() + 40*60e3, scope: 'openid profile email' },
    { jti: '3c1d4e', user: 'priya.nair@stride.io', issued: Date.now() - 8*60e3, exp: Date.now() + 52*60e3, scope: 'openid profile' },
    { jti: '7b2a9f', user: 'tomas@orbit.so', issued: Date.now() - 22*60e3, exp: Date.now() + 38*60e3, scope: 'openid profile email' },
    { jti: '9e4f3a', user: 'kenji@hexcel.co', issued: Date.now() - 44*60e3, exp: Date.now() + 16*60e3, scope: 'openid' },
  ];
  return (
    <div style={{padding: 16}}>
      <div className="faint" style={{fontSize: 11, marginBottom: 10}}>
        {app.tokensIssued24h.toLocaleString()} tokens issued in last 24h. Active: <span className="mono">{rows.length * 52}</span>.
      </div>
      <table style={{width:'100%', borderCollapse:'collapse', fontSize: 11}}>
        <thead><tr>
          <th style={appThStyle}>JTI</th><th style={appThStyle}>User</th><th style={appThStyle}>Issued</th><th style={appThStyle}>Expires</th>
        </tr></thead>
        <tbody>
          {rows.map(r => (
            <tr key={r.jti}>
              <td style={appTdStyle}><span className="mono">{r.jti}</span></td>
              <td style={appTdStyle}>{r.user}</td>
              <td style={appTdStyle}><span className="mono faint">{MOCK.relativeTime(r.issued)}</span></td>
              <td style={appTdStyle}><span className="mono faint">{MOCK.relativeTime(r.exp)}</span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AppEvents({ app }) {
  const events = MOCK.audit.filter(e => e.action.startsWith('app.') || e.action.startsWith('oauth.')).slice(0, 10);
  return (
    <div style={{padding: 16}}>
      {events.map(e => (
        <div key={e.id} className="row" style={{padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 8, fontSize: 11.5}}>
          <SevDot sev={e.severity}/>
          <span className="mono" style={{flex: 1}}>{e.action}</span>
          <span className="faint">{e.actor}</span>
          <span className="faint mono" style={{fontSize: 10}}>{MOCK.relativeTime(e.t)}</span>
        </div>
      ))}
    </div>
  );
}

function RotateSecretModal({ app, onClose }) {
  const [stage, setStage] = React.useState('confirm'); // confirm → reveal
  const newSecret = 'sk_live_8f92aB_' + Math.random().toString(36).slice(2, 14) + 'Wq9rTn';
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
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn ghost" onClick={onClose}>Cancel</button>
            <button className="btn danger" onClick={() => setStage('reveal')}>Rotate now</button>
          </div>
        </> : <>
          <h2 style={{margin:0, fontSize: 15, fontWeight: 600}}>New secret · copy it now</h2>
          <p className="faint" style={{fontSize: 12, marginTop: 6}}>This is the only time you'll see this value.</p>
          <div style={{background:'#000', color:'#fff', padding: 14, borderRadius: 4, marginTop: 14, fontFamily: 'var(--font-mono)', fontSize: 12, wordBreak:'break-all', position:'relative'}}>
            {newSecret}
            <button className="btn ghost sm" style={{position:'absolute', top: 6, right: 6, color:'#fff', borderColor:'rgba(255,255,255,0.2)'}}>
              <Icon.Copy width={10} height={10}/> Copy
            </button>
          </div>
          <div className="faint mono" style={{fontSize: 11, marginTop: 10}}>
            kid: sec_2026_Q1 · valid from {new Date().toISOString().slice(0,16)}Z · old secret expires in 24h
          </div>
          <div className="row" style={{marginTop: 18, justifyContent:'flex-end', gap: 8}}>
            <button className="btn primary" onClick={onClose}>I've saved it</button>
          </div>
        </>}
      </div>
    </div>
  );
}

function CreateAppSlideOver({ onClose }) {
  const [step, setStep] = React.useState(1);
  return (
    <div style={modalBackdrop} onClick={onClose}>
      <div style={{
        position:'fixed', top: 0, right: 0, bottom: 0, width: 520,
        background:'var(--surface-0)', borderLeft:'1px solid var(--hairline-bright)',
        display:'flex', flexDirection:'column', boxShadow: 'var(--shadow-lg)',
      }} onClick={e => e.stopPropagation()}>
        <div className="row" style={{padding: '14px 16px', borderBottom:'1px solid var(--hairline)'}}>
          <h2 style={{margin:0, fontSize: 14, fontWeight: 600, flex:1}}>New application · step {step} of 3</h2>
          <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
        </div>
        <div style={{flex:1, overflowY:'auto', padding: 20}}>
          {step === 1 && <div>
            <h3 style={{fontSize: 12.5, margin: '0 0 14px'}}>Pick application type</h3>
            {[
              ['Web app', 'Server-side. Uses client_secret. Authorization Code + PKCE.'],
              ['Single-page app', 'Browser-only. Public client. PKCE required.'],
              ['Mobile app', 'Native iOS/Android. Custom schemes for redirect.'],
              ['Service (M2M)', 'No end user. Client credentials grant.'],
            ].map(([name, desc]) => (
              <button key={name} style={{display:'block', width:'100%', textAlign:'left', padding: 12, border:'1px solid var(--hairline)', borderRadius: 4, marginBottom: 8, background: 'var(--surface-1)'}}>
                <div style={{fontWeight: 500, fontSize: 12.5}}>{name}</div>
                <div className="faint" style={{fontSize: 11, marginTop: 2}}>{desc}</div>
              </button>
            ))}
          </div>}
          {step > 1 && <div className="faint">Config fields…</div>}
        </div>
        <div className="row" style={{padding: 12, borderTop: '1px solid var(--hairline)', gap: 8, justifyContent: 'flex-end'}}>
          <button className="btn ghost" onClick={onClose}>Cancel</button>
          <button className="btn ghost" onClick={() => setStep(Math.max(1, step - 1))} disabled={step === 1}>Back</button>
          <button className="btn primary" onClick={() => step < 3 ? setStep(step + 1) : onClose()}>
            {step < 3 ? 'Next' : 'Create'}
          </button>
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

const appThStyle = { textAlign: 'left', padding: '8px 14px', fontSize: 10, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const appTdStyle = { padding: '9px 14px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'middle' };
const modalBackdrop = { position:'fixed', inset: 0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex: 50 };
const modalCard = { background:'var(--surface-1)', border:'1px solid var(--hairline-bright)', borderRadius: 6, padding: 18 };

Object.assign(window, { Applications });
