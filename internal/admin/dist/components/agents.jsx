// Agents page — table + detail slide-over (Phase 6 standout)

function Agents() {
  const [selected, setSelected] = React.useState(MOCK.agents[0]);

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* Hero strip — agents story */}
        <div style={{
          padding: '14px 16px',
          borderBottom: '1px solid var(--hairline)',
          background: 'linear-gradient(180deg, var(--surface-1), var(--surface-0))',
          display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 16,
        }}>
          <div>
            <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Registered</div>
            <div style={{ fontSize: 22, fontWeight: 600, marginTop: 2, fontVariantNumeric: 'tabular-nums' }}>23</div>
            <div className="faint" style={{ fontSize: 11 }}>+6 last 7d</div>
          </div>
          <div>
            <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Live tokens</div>
            <div style={{ fontSize: 22, fontWeight: 600, marginTop: 2, fontVariantNumeric: 'tabular-nums' }}>1,284</div>
            <div className="faint" style={{ fontSize: 11 }}>482 DPoP-bound</div>
          </div>
          <div>
            <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Issuance · 24h</div>
            <div style={{ fontSize: 22, fontWeight: 600, marginTop: 2, fontVariantNumeric: 'tabular-nums' }}>8,420</div>
            <div className="faint" style={{ fontSize: 11 }}>0.3% error rate</div>
          </div>
          <div>
            <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Consents · active</div>
            <div style={{ fontSize: 22, fontWeight: 600, marginTop: 2, fontVariantNumeric: 'tabular-nums' }}>3,817</div>
            <div className="faint" style={{ fontSize: 11 }}>across 892 users</div>
          </div>
          <div style={{ gridColumn: 5, display: 'flex', alignItems: 'flex-end', justifyContent: 'flex-end', gap: 8 }}>
            <button className="btn"><Icon.External width={11} height={11}/>Docs</button>
            <button className="btn primary"><Icon.Plus width={11} height={11}/>Register agent</button>
          </div>
        </div>

        {/* Toolbar */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 6,
            border: '1px solid var(--hairline-strong)',
            background: 'var(--surface-1)',
            borderRadius: 5, padding: '0 8px',
            height: 28, width: 240,
          }}>
            <Icon.Search width={12} height={12} style={{opacity:0.5}}/>
            <input placeholder="Filter agents, client_id…" style={{ flex: 1, fontSize: 12 }}/>
          </div>
          <button className="btn"><Icon.Filter width={11} height={11}/>Type<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn"><Icon.Filter width={11} height={11}/>Grant<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <button className="btn"><Icon.Filter width={11} height={11}/>Status<Icon.ChevronDown width={10} height={10} style={{opacity:0.5}}/></button>
          <div style={{ flex: 1 }}/>
          <span className="faint" style={{ fontSize: 11 }}>8 agents</span>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table className="tbl">
            <thead>
              <tr>
                <th style={{ paddingLeft: 16 }}>Agent</th>
                <th style={{ width: 180 }}>Client ID</th>
                <th style={{ width: 90 }}>Type</th>
                <th>Grants</th>
                <th style={{ width: 70 }}>DPoP</th>
                <th style={{ width: 90 }}>Tokens</th>
                <th style={{ width: 100 }}>Last used</th>
                <th style={{ width: 90 }}>Status</th>
                <th style={{ width: 36 }}></th>
              </tr>
            </thead>
            <tbody>
              {MOCK.agents.map(a => (
                <tr key={a.id}
                  className={selected?.id === a.id ? 'active' : ''}
                  onClick={() => setSelected(a)}
                  style={{ cursor: 'pointer' }}>
                  <td style={{ paddingLeft: 16 }}>
                    <div className="row" style={{ gap: 8 }}>
                      <Avatar name={a.name} agent/>
                      <div>
                        <div style={{ fontWeight: 500, fontSize: 12.5 }}>{a.name}</div>
                        <div className="faint" style={{ fontSize: 11 }}>{a.description}</div>
                      </div>
                    </div>
                  </td>
                  <td onClick={e => e.stopPropagation()}>
                    <CopyField value={a.clientId}/>
                  </td>
                  <td>
                    <span className={"chip" + (a.type === 'confidential' ? ' solid' : '')}>{a.type === 'confidential' ? 'conf' : 'public'}</span>
                  </td>
                  <td>
                    <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
                      {a.grants.slice(0, 2).map(g => <span key={g} className="chip mono" style={{ height: 17, fontSize: 9.5 }}>{g.replace('_', ' ')}</span>)}
                      {a.grants.length > 2 && <span className="faint mono" style={{ fontSize: 10 }}>+{a.grants.length - 2}</span>}
                    </div>
                  </td>
                  <td>{a.dpop ? <Icon.DPoP width={13} height={13} style={{color:'var(--success)'}}/> : <span className="faint">—</span>}</td>
                  <td className="mono" style={{ fontSize: 12, fontVariantNumeric: 'tabular-nums' }}>
                    {a.tokensActive > 0 ? (
                      <span style={{ color: a.tokensActive > 200 ? 'var(--agent)' : 'var(--fg)' }}>{a.tokensActive}</span>
                    ) : <span className="faint">0</span>}
                  </td>
                  <td className="mono faint" style={{ fontSize: 11 }}>{MOCK.relativeTime(a.lastUsed)}</td>
                  <td>
                    {a.status === 'active'
                      ? <span className="chip success"><span className="dot success"/>active</span>
                      : <span className="chip"><span className="dot"/>disabled</span>}
                  </td>
                  <td onClick={e => e.stopPropagation()}>
                    <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selected && <AgentSlideover agent={selected} onClose={() => setSelected(null)}/>}
    </div>
  );
}

function AgentSlideover({ agent, onClose }) {
  const [tab, setTab] = React.useState('tokens');
  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'tokens', label: 'Tokens', chip: agent.tokensActive },
    { id: 'consents', label: 'Consents' },
    { id: 'usage', label: 'Usage' },
    { id: 'delegation', label: 'Delegation' },
  ];

  // Fake token list for selected agent
  const tokenSample = Array.from({ length: 7 }).map((_, i) => ({
    jti: `${Math.random().toString(36).slice(2, 6)}…${Math.random().toString(36).slice(2, 6)}`,
    user: ['amelia@nimbus.sh','priya.nair@stride.io','tomas@orbit.so','kenji@hexcel.co','zara@apex.dev','—','dev@hexcel.co'][i],
    scope: agent.scopes.slice(0, Math.min(3, agent.scopes.length)).join(' '),
    dpop: agent.dpop && i !== 3,
    expires: `${5 + i * 9}m`,
    family: `fam_${Math.random().toString(36).slice(2, 6)}`,
    created: `${1 + i * 3}m ago`,
  }));

  return (
    <div style={{
      width: 620, borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column', flexShrink: 0,
    }}>
      {/* Header */}
      <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
          <button className="btn ghost sm" onClick={onClose}><Icon.X width={12} height={12}/>Close</button>
          <div className="row" style={{ gap: 4 }}>
            <button className="btn sm">Rotate secret</button>
            <button className="btn sm danger">Revoke all tokens</button>
            <button className="btn ghost icon sm"><Icon.More width={12} height={12}/></button>
          </div>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <Avatar name={agent.name} agent size={44}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div className="row" style={{ gap: 6 }}>
              <span style={{ fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em' }}>{agent.name}</span>
              {agent.status === 'active'
                ? <span className="chip success" style={{ height: 18 }}><span className="dot success"/>active</span>
                : <span className="chip" style={{ height: 18 }}>disabled</span>}
            </div>
            <div style={{ fontSize: 12, color: 'var(--fg-muted)', marginTop: 2 }}>{agent.description}</div>
            <div className="row" style={{ gap: 6, marginTop: 8 }}>
              <CopyField value={agent.clientId}/>
              <span className="chip">{agent.type}</span>
              {agent.dpop && <span className="chip success"><Icon.DPoP width={10} height={10}/>DPoP bound</span>}
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 2,
        padding: '0 16px',
        borderBottom: '1px solid var(--hairline)',
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '10px 10px',
            fontSize: 12,
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
          }}>
            {t.label}
            {t.chip !== undefined && <span className="chip agent" style={{ marginLeft: 5, height: 15, fontSize: 9, padding: '0 4px' }}>{t.chip}</span>}
          </button>
        ))}
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {tab === 'overview' && <AgentOverview agent={agent}/>}
        {tab === 'tokens' && <AgentTokens agent={agent} tokens={tokenSample}/>}
        {tab === 'consents' && <AgentConsents agent={agent}/>}
        {tab === 'usage' && <AgentUsage agent={agent}/>}
        {tab === 'delegation' && <AgentDelegation agent={agent}/>}
      </div>

      {/* CLI footer */}
      <div style={{
        borderTop: '1px solid var(--hairline)',
        padding: '10px 16px',
        background: 'var(--surface-1)',
        fontSize: 11,
      }}>
        <div className="row" style={{ gap: 8 }}>
          <Icon.Terminal width={12} height={12} style={{ opacity: 0.5 }}/>
          <code className="mono" style={{ flex: 1, color: 'var(--fg-muted)' }}>shark agent show {agent.id}</code>
          <button className="btn ghost icon sm"><Icon.Copy width={11} height={11}/></button>
        </div>
      </div>
    </div>
  );
}

function AgentOverview({ agent }) {
  return (
    <div style={{ padding: 16 }}>
      <div style={{ marginBottom: 10, fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Grants</div>
      <div className="row" style={{ gap: 4, flexWrap: 'wrap', marginBottom: 14 }}>
        {agent.grants.map(g => <span key={g} className="chip mono">{g}</span>)}
      </div>

      <div style={{ marginBottom: 10, fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Scopes</div>
      <div className="row" style={{ gap: 4, flexWrap: 'wrap', marginBottom: 14 }}>
        {agent.scopes.map(s => <span key={s} className="chip mono">{s}</span>)}
      </div>

      <div style={{ marginBottom: 10, fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Endpoints</div>
      <div style={{ border: '1px solid var(--hairline-strong)', borderRadius: 5, background: 'var(--surface-1)', overflow: 'hidden', marginBottom: 14 }}>
        {[
          ['authorize', 'https://auth.shark.sh/oauth/authorize'],
          ['token', 'https://auth.shark.sh/oauth/token'],
          ['revoke', 'https://auth.shark.sh/oauth/revoke'],
          ['introspect', 'https://auth.shark.sh/oauth/introspect'],
          ['jwks', 'https://auth.shark.sh/.well-known/jwks.json'],
        ].map(([k, v], i, arr) => (
          <div key={k} className="row" style={{ padding: '7px 10px', borderBottom: i < arr.length - 1 ? '1px solid var(--hairline)' : 'none' }}>
            <span className="mono" style={{ fontSize: 10.5, color: 'var(--fg-dim)', width: 80 }}>{k}</span>
            <span className="mono" style={{ fontSize: 11, flex: 1 }}>{v}</span>
            <button className="btn ghost icon sm"><Icon.Copy width={11} height={11}/></button>
          </div>
        ))}
      </div>

      <div style={{ marginBottom: 10, fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>Request a token (curl)</div>
      <pre className="mono" style={{
        margin: 0, padding: 12,
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 5,
        fontSize: 11, lineHeight: 1.5,
        overflow: 'auto', color: 'var(--fg-muted)',
      }}>{`curl -X POST https://auth.shark.sh/oauth/token \\
  -H "Content-Type: application/x-www-form-urlencoded" \\
  -H "DPoP: <proof>" \\
  -d "grant_type=${agent.grants[0]}" \\
  -d "client_id=${agent.clientId}" \\
  -d "scope=${agent.scopes.slice(0,3).join(' ')}"`}</pre>
    </div>
  );
}

function AgentTokens({ agent, tokens }) {
  return (
    <>
      <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--hairline)', display: 'flex', gap: 8, alignItems: 'center' }}>
        <span style={{ fontSize: 12 }}>{agent.tokensActive} active tokens</span>
        <span className="faint" style={{ fontSize: 11 }}>· {tokens.filter(t => t.dpop).length}/{tokens.length} DPoP-bound</span>
        <div style={{ flex: 1 }}/>
        <button className="btn sm"><Icon.Refresh width={11} height={11}/></button>
        <button className="btn sm">Bulk revoke</button>
      </div>
      <table className="tbl">
        <thead>
          <tr>
            <th style={{ paddingLeft: 16 }}>JTI</th>
            <th>Subject</th>
            <th>Scope</th>
            <th style={{ width: 50 }}>DPoP</th>
            <th style={{ width: 70 }}>Expires</th>
            <th style={{ width: 36 }}></th>
          </tr>
        </thead>
        <tbody>
          {tokens.map((t, i) => (
            <tr key={i}>
              <td style={{ paddingLeft: 16 }} className="mono" onClick={e => e.stopPropagation()}>
                <CopyField value={t.jti} truncate={0}/>
              </td>
              <td style={{ fontSize: 11.5 }}>{t.user}</td>
              <td className="mono faint" style={{ fontSize: 10.5, maxWidth: 160, overflow:'hidden', textOverflow:'ellipsis', whiteSpace: 'nowrap' }}>{t.scope}</td>
              <td>{t.dpop ? <Icon.Check width={11} height={11} style={{color:'var(--success)'}}/> : <span className="faint">—</span>}</td>
              <td className="mono" style={{ fontSize: 11, color: parseInt(t.expires) < 15 ? 'var(--warn)' : 'var(--fg-muted)' }}>{t.expires}</td>
              <td><button className="btn ghost icon sm"><Icon.More width={11} height={11}/></button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}

function AgentConsents({ agent }) {
  const users = [
    { email: 'amelia@nimbus.sh', granted: '12d ago', tokens: 3 },
    { email: 'priya.nair@stride.io', granted: '8d ago', tokens: 2 },
    { email: 'tomas@orbit.so', granted: '5d ago', tokens: 1 },
    { email: 'kenji@hexcel.co', granted: '4d ago', tokens: 2 },
    { email: 'zara@apex.dev', granted: '2d ago', tokens: 1 },
  ];
  return (
    <div style={{ padding: 16 }}>
      <div className="faint" style={{ fontSize: 11.5, marginBottom: 10 }}>
        {users.length * 35} users have granted <strong style={{color:'var(--fg)'}}>{agent.name}</strong> access. Revoking a consent invalidates all tokens.
      </div>
      <table className="tbl" style={{ border: '1px solid var(--hairline)', borderRadius: 5 }}>
        <thead><tr><th style={{ paddingLeft: 12 }}>User</th><th>Scopes</th><th style={{ width: 60 }}>Tokens</th><th style={{ width: 80 }}>Granted</th><th style={{ width: 36 }}></th></tr></thead>
        <tbody>
          {users.map((u, i) => (
            <tr key={i}>
              <td style={{ paddingLeft: 12 }}>
                <div className="row" style={{ gap: 8 }}>
                  <Avatar name={u.email} email={u.email} size={20}/>
                  <span style={{ fontSize: 11.5 }}>{u.email}</span>
                </div>
              </td>
              <td><span className="mono faint" style={{ fontSize: 10.5 }}>{agent.scopes.length} scopes</span></td>
              <td className="mono" style={{ fontSize: 11 }}>{u.tokens}</td>
              <td className="mono faint" style={{ fontSize: 10.5 }}>{u.granted}</td>
              <td><button className="btn ghost sm danger" style={{fontSize:10.5}}>Revoke</button></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function AgentUsage({ agent }) {
  const hrs = [120, 135, 142, 155, 168, 182, 210, 245, 310, 380, 420, 465, 490, 520, 550, 580, 610, 595, 570, 540, 510, 470, 420, 380];
  return (
    <div style={{ padding: 16 }}>
      <div style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 8 }}>Token issuance · last 24h</div>
      <div style={{
        height: 140, padding: 12, border: '1px solid var(--hairline)', borderRadius: 5,
        background: 'var(--surface-1)', display: 'flex', alignItems: 'flex-end', gap: 3,
      }}>
        {hrs.map((v, i) => (
          <div key={i} style={{
            flex: 1, background: 'var(--agent)', opacity: 0.4 + (v / 600) * 0.6,
            height: `${(v / 700) * 100}%`, borderRadius: '2px 2px 0 0', minHeight: 2,
          }}/>
        ))}
      </div>
      <div className="row" style={{ justifyContent: 'space-between', marginTop: 4, fontSize: 10, color: 'var(--fg-faint)' }}>
        <span className="mono">24h ago</span><span className="mono">12h</span><span className="mono">now</span>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginTop: 16 }}>
        <div>
          <div style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 8 }}>Top scopes</div>
          <div className="col" style={{ gap: 4 }}>
            {agent.scopes.slice(0, 5).map((s, i) => (
              <div key={s} className="row" style={{ fontSize: 11 }}>
                <span className="mono" style={{ flex: 1 }}>{s}</span>
                <span className="mono faint">{Math.round(agent.tokensActive * (0.9 - i * 0.15))}</span>
                <div style={{ width: 50, height: 3, background: 'var(--surface-3)', borderRadius: 1, overflow: 'hidden' }}>
                  <div style={{ width: `${100 - i * 18}%`, height: '100%', background: 'var(--agent)' }}/>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div>
          <div style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', marginBottom: 8 }}>Errors · 7d</div>
          <div className="col" style={{ gap: 4, fontSize: 11 }}>
            <div className="row"><span className="mono" style={{flex:1}}>invalid_grant</span><span className="mono faint">12</span></div>
            <div className="row"><span className="mono" style={{flex:1}}>invalid_scope</span><span className="mono faint">4</span></div>
            <div className="row"><span className="mono" style={{flex:1}}>dpop.replay</span><span className="mono faint">1</span></div>
            <div className="row"><span className="mono" style={{flex:1}}>rate_limited</span><span className="mono faint">0</span></div>
          </div>
          <div style={{ marginTop: 10, padding: 8, background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 4, fontSize: 11 }}>
            <div className="row" style={{ gap: 6 }}>
              <span className="dot success"/>
              <span>Error rate</span>
              <span className="mono" style={{ marginLeft: 'auto' }}>0.3%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function AgentDelegation({ agent }) {
  return (
    <div style={{ padding: 16 }}>
      <div className="faint" style={{ fontSize: 11.5, marginBottom: 14 }}>
        RFC 8693 token exchange — agents this one delegates to or acts on behalf of.
      </div>
      <div style={{ padding: 20, border: '1px solid var(--hairline)', borderRadius: 5, background: 'var(--surface-1)', textAlign: 'center' }}>
        <svg width="100%" height="140" viewBox="0 0 400 140">
          <defs>
            <marker id="arr" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto">
              <path d="M0,0 L10,5 L0,10 z" fill="#666"/>
            </marker>
          </defs>
          <g fontFamily="var(--font-mono)" fontSize="11" fill="#fafafa">
            <rect x="20" y="55" width="110" height="34" rx="4" fill="#111" stroke="#2a2a2a"/>
            <text x="75" y="76" textAnchor="middle">amelia@nimbus</text>

            <line x1="130" y1="72" x2="175" y2="72" stroke="#666" markerEnd="url(#arr)"/>

            <rect x="175" y="45" width="100" height="54" rx="4" fill="#1c1c1c" stroke="oklch(0.9 0.03 80)"/>
            <text x="225" y="68" textAnchor="middle">{agent.name}</text>
            <text x="225" y="85" textAnchor="middle" fontSize="9" fill="#999">act: on behalf of</text>

            <line x1="275" y1="72" x2="320" y2="72" stroke="#666" markerEnd="url(#arr)"/>

            <rect x="320" y="55" width="70" height="34" rx="4" fill="#111" stroke="#2a2a2a"/>
            <text x="355" y="76" textAnchor="middle" fontSize="10">svc-worker</text>
          </g>
        </svg>
      </div>
      <div style={{ marginTop: 14, fontSize: 11.5, color: 'var(--fg-muted)' }}>
        <span className="mono">act</span> chain: user → agent → downstream service. 2,140 exchanges / 24h.
      </div>
    </div>
  );
}

window.Agents = Agents;
