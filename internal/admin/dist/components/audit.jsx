// Audit log page — live-tail event stream w/ filters, event detail, CSV export

function Audit() {
  const [events, setEvents] = React.useState(() => MOCK.audit);
  const [liveTail, setLiveTail] = React.useState(false);
  const [selected, setSelected] = React.useState(null);
  const [query, setQuery] = React.useState('');
  const [filters, setFilters] = React.useState({
    actorType: null,   // user/admin/agent/system
    severity: null,    // info/warn/danger
    actionPrefix: null,
    timeRange: '24h',  // 1h/24h/7d/all
    delegated: false,
  });
  const [savedView, setSavedView] = React.useState(null);
  const tailRef = React.useRef(null);

  // Live tail: prepend a synthetic event every ~3s
  React.useEffect(() => {
    if (!liveTail) return;
    const id = setInterval(() => {
      const pool = MOCK.audit;
      const base = pool[Math.floor(Math.random() * pool.length)];
      const ev = {
        ...base,
        id: 'evt_live_' + Math.random().toString(36).slice(2, 10),
        t: Date.now(),
        _live: true,
      };
      setEvents(prev => [ev, ...prev].slice(0, 500));
    }, 2400 + Math.random() * 1400);
    return () => clearInterval(id);
  }, [liveTail]);

  const filtered = React.useMemo(() => {
    const now = Date.now();
    const cutoff = filters.timeRange === '1h' ? now - 3600e3
                 : filters.timeRange === '24h' ? now - 24 * 3600e3
                 : filters.timeRange === '7d' ? now - 7 * 24 * 3600e3 : 0;
    return events.filter(e => {
      if (cutoff && e.t < cutoff) return false;
      if (filters.actorType && e.actor_type !== filters.actorType) return false;
      if (filters.severity && e.severity !== filters.severity) return false;
      if (filters.actionPrefix && !e.action.startsWith(filters.actionPrefix)) return false;
      if (filters.delegated && e.actor_type !== 'agent') return false;
      if (query) {
        const q = query.toLowerCase();
        if (!e.action.includes(q) && !String(e.actor).toLowerCase().includes(q) &&
            !String(e.target).toLowerCase().includes(q) && !String(e.ip || '').includes(q) &&
            !e.id.includes(q)) return false;
      }
      return true;
    });
  }, [events, filters, query]);

  const stats = React.useMemo(() => {
    const byActor = {};
    const bySev = { info: 0, warn: 0, danger: 0 };
    filtered.forEach(e => {
      byActor[e.actor_type] = (byActor[e.actor_type] || 0) + 1;
      bySev[e.severity]++;
    });
    return { byActor, bySev, total: filtered.length };
  }, [filtered]);

  // Activity bars — 60 buckets
  const buckets = React.useMemo(() => {
    const now = Date.now();
    const step = (filters.timeRange === '1h' ? 3600e3 : filters.timeRange === '7d' ? 7*24*3600e3 : 24*3600e3) / 60;
    const start = now - step * 60;
    const out = new Array(60).fill(0).map(() => ({ info: 0, warn: 0, danger: 0 }));
    filtered.forEach(e => {
      const idx = Math.floor((e.t - start) / step);
      if (idx >= 0 && idx < 60) out[idx][e.severity]++;
    });
    return out;
  }, [filtered, filters.timeRange]);

  const maxBucket = Math.max(1, ...buckets.map(b => b.info + b.warn + b.danger));

  const savedViews = [
    { id: 'fails', label: 'Failed logins', apply: () => setFilters(f => ({...f, actionPrefix: 'user.login.failed', timeRange: '24h'})) },
    { id: 'secrets', label: 'Secret rotations', apply: () => setFilters(f => ({...f, actionPrefix: 'agent.secret', timeRange: '7d'})) },
    { id: 'tokens', label: 'Token issuance', apply: () => setFilters(f => ({...f, actionPrefix: 'oauth.token', actorType: 'agent'})) },
    { id: 'admin', label: 'Admin actions', apply: () => setFilters(f => ({...f, actorType: 'admin'})) },
    { id: 'danger', label: 'High severity', apply: () => setFilters(f => ({...f, severity: 'danger'})) },
    { id: 'webhooks', label: 'Webhook deliveries', apply: () => setFilters(f => ({...f, actionPrefix: 'webhook.'})) },
  ];

  return (
    <div style={{ display: 'grid', gridTemplateColumns: selected ? '1fr 420px' : '1fr', height: '100%', overflow: 'hidden' }}>
      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0, overflow: 'hidden' }}>
        {/* Header */}
        <div style={{ padding: '14px 20px 0', borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ marginBottom: 12, gap: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Audit Log</h1>
              <p className="faint" style={{ margin: '2px 0 0', fontSize: 11.5 }}>
                Immutable event stream · <span className="mono">retention 2y</span> · <span className="mono">shark-audit-v2</span>
              </p>
            </div>
            <button className={"btn " + (liveTail ? 'primary' : 'ghost')} onClick={() => setLiveTail(v => !v)}>
              <span className={"dot " + (liveTail ? 'warn pulse' : '')} style={{background: liveTail ? undefined : 'var(--fg-faint)'}}/>
              {liveTail ? 'Streaming' : 'Live tail'}
            </button>
            <button className="btn ghost">
              <Icon.Copy width={11} height={11}/> Export CSV
            </button>
            <button className="btn ghost">
              <Icon.Webhook width={11} height={11}/> Send to sink
            </button>
          </div>

          {/* Activity bars */}
          <div style={{ display: 'flex', alignItems: 'flex-end', gap: 2, height: 40, marginBottom: 10 }}>
            {buckets.map((b, i) => {
              const total = b.info + b.warn + b.danger;
              const h = (total / maxBucket) * 40;
              return (
                <div key={i} style={{ flex: 1, height: 40, display: 'flex', flexDirection: 'column', justifyContent: 'flex-end' }}>
                  {b.danger > 0 && <div style={{ background: 'var(--danger)', height: (b.danger / maxBucket) * 40, opacity: 0.9 }}/>}
                  {b.warn > 0 && <div style={{ background: 'var(--warn)', height: (b.warn / maxBucket) * 40, opacity: 0.85 }}/>}
                  {b.info > 0 && <div style={{ background: 'var(--fg-dim)', height: (b.info / maxBucket) * 40, opacity: 0.55 }}/>}
                  {total === 0 && <div style={{ background: 'var(--hairline)', height: 1 }}/>}
                </div>
              );
            })}
          </div>
          <div className="row faint" style={{ fontSize: 10, marginBottom: 10, justifyContent: 'space-between' }}>
            <span className="mono">{filters.timeRange === '1h' ? '60 min ago' : filters.timeRange === '7d' ? '7 days ago' : '24h ago'}</span>
            <span>
              <span className="row" style={{ display: 'inline-flex', gap: 10 }}>
                <span className="row" style={{gap:4}}><span style={{width:8,height:8,background:'var(--fg-dim)',opacity:0.55}}/>info {stats.bySev.info}</span>
                <span className="row" style={{gap:4}}><span style={{width:8,height:8,background:'var(--warn)'}}/>warn {stats.bySev.warn}</span>
                <span className="row" style={{gap:4}}><span style={{width:8,height:8,background:'var(--danger)'}}/>danger {stats.bySev.danger}</span>
              </span>
            </span>
            <span className="mono">now</span>
          </div>

          {/* Filter chips */}
          <div className="row" style={{ gap: 6, flexWrap: 'wrap', paddingBottom: 10 }}>
            <span className="faint" style={{fontSize:11}}>Views:</span>
            {savedViews.map(v => (
              <button key={v.id} className="chip" onClick={v.apply} style={{cursor:'pointer'}}>{v.label}</button>
            ))}
            <div style={{flex:1}}/>
            <button className="chip" onClick={() => setFilters({ actorType: null, severity: null, actionPrefix: null, timeRange: '24h', delegated: false })}>
              <Icon.X width={9} height={9}/> Clear
            </button>
          </div>
        </div>

        {/* Toolbar */}
        <div className="row" style={{ padding: '10px 20px', gap: 8, borderBottom: '1px solid var(--hairline)', flexWrap: 'wrap' }}>
          <div className="row" style={{
            flex: 1, minWidth: 260, gap: 6, padding: '0 8px', height: 28,
            background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 4,
          }}>
            <Icon.Search width={12} height={12} style={{opacity:0.6}}/>
            <input
              placeholder="Filter by action, actor, target, IP, or evt_…"
              value={query} onChange={e => setQuery(e.target.value)}
              style={{ flex: 1, background: 'transparent', border: 0, outline: 'none', color: 'var(--fg)', fontSize: 12, fontFamily: 'var(--font-mono)' }}
            />
            <Kbd keys="/"/>
          </div>

          <Seg value={filters.timeRange} onChange={v => setFilters(f => ({...f, timeRange: v}))}
            opts={[['1h','1h'], ['24h','24h'], ['7d','7d'], ['all','all']]}/>

          <Seg value={filters.actorType} onChange={v => setFilters(f => ({...f, actorType: v}))}
            opts={[[null,'All'], ['user','👤 user'], ['admin','admin'], ['agent','agent'], ['system','system']]}
            mono/>

          <Seg value={filters.severity} onChange={v => setFilters(f => ({...f, severity: v}))}
            opts={[[null,'Any'], ['info','info'], ['warn','warn'], ['danger','danger']]}
            mono/>

          <button className={"chip " + (filters.delegated ? 'warn' : '')}
            onClick={() => setFilters(f => ({...f, delegated: !f.delegated}))}
            style={{cursor:'pointer', height:22}}>
            <Icon.Agent width={10} height={10}/>
            delegation chain only
          </button>
        </div>

        {/* Table */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <table className="data-table" style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr>
                <th style={thStyle}>When</th>
                <th style={thStyle}>Action</th>
                <th style={thStyle}>Actor</th>
                <th style={thStyle}>Target</th>
                <th style={thStyle}>IP</th>
                <th style={{...thStyle, width: 110, textAlign: 'right'}}>Event ID</th>
              </tr>
            </thead>
            <tbody>
              {filtered.slice(0, 200).map(e => (
                <tr key={e.id}
                  onClick={() => setSelected(e)}
                  className={selected?.id === e.id ? 'row-selected' : ''}
                  style={{
                    cursor: 'pointer',
                    background: selected?.id === e.id ? 'var(--surface-2)' : e._live ? 'rgba(255,170,0,0.04)' : 'transparent',
                    animation: e._live ? 'fadeIn 240ms ease-out' : undefined,
                  }}>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{fontSize: 10.5}}>
                      {new Date(e.t).toLocaleTimeString(undefined, {hour: '2-digit', minute:'2-digit', second:'2-digit'})}
                    </span>
                    <div className="faint" style={{fontSize: 10}}>{MOCK.relativeTime(e.t)}</div>
                  </td>
                  <td style={tdStyle}>
                    <span className="row" style={{gap:6}}>
                      <SevDot sev={e.severity}/>
                      <span className="mono" style={{fontSize: 11.5}}>{e.action}</span>
                    </span>
                  </td>
                  <td style={tdStyle}>
                    <span className="row" style={{gap:6}}>
                      <span className="chip" style={{height:16, fontSize:9.5, textTransform:'uppercase', letterSpacing:'0.05em'}}>{e.actor_type}</span>
                      <span className={e.actor_type === 'agent' ? 'mono' : ''} style={{fontSize: 11.5}}>{e.actor}</span>
                    </span>
                  </td>
                  <td style={tdStyle}>
                    <span className="mono faint" style={{fontSize: 11}}>{e.target}</span>
                  </td>
                  <td style={tdStyle}>
                    {e.ip && <span className="mono faint" style={{fontSize: 11}}>{e.ip}</span>}
                  </td>
                  <td style={{...tdStyle, textAlign: 'right'}}>
                    <span className="mono faint" style={{fontSize: 10}}>{e.id.slice(0, 14)}…</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {filtered.length === 0 && (
            <div style={{padding: '60px 20px', textAlign: 'center'}} className="faint">No events match these filters.</div>
          )}
        </div>

        {/* Footer */}
        <div className="row" style={{ padding: '8px 20px', borderTop: '1px solid var(--hairline)', fontSize: 11, gap: 14 }}>
          <span className="faint mono">{filtered.length.toLocaleString()} events</span>
          <span className="faint">·</span>
          <span className="mono faint">
            {stats.byActor.user || 0} user · {stats.byActor.admin || 0} admin · {stats.byActor.agent || 0} agent · {stats.byActor.system || 0} system
          </span>
          <div style={{flex:1}}/>
          <span className="faint mono" style={{fontSize:10.5}}>shark audit tail --follow  --filter "{filters.actorType || '*'}"</span>
        </div>
      </div>

      {selected && <EventDetail event={selected} onClose={() => setSelected(null)}/>}
    </div>
  );
}

function SevDot({ sev }) {
  const c = sev === 'danger' ? 'var(--danger)' : sev === 'warn' ? 'var(--warn)' : 'var(--fg-dim)';
  return <span style={{width:6,height:6,borderRadius:1,background:c,flexShrink:0}}/>;
}

function Seg({ value, onChange, opts, mono }) {
  return (
    <div className="seg" style={{height: 22, display:'inline-flex', border:'1px solid var(--hairline-strong)', borderRadius:3, overflow:'hidden'}}>
      {opts.map(([v, label]) => (
        <button key={String(v)} onClick={() => onChange(v)}
          className={mono ? 'mono' : ''}
          style={{
            padding: '0 8px', height: 22,
            fontSize: 10.5,
            background: value === v ? '#fafafa' : 'var(--surface-2)',
            color: value === v ? '#000' : 'var(--fg-muted)',
            borderRight: '1px solid var(--hairline)',
          }}>{label}</button>
      ))}
    </div>
  );
}

function EventDetail({ event, onClose }) {
  // Synthesize a representative payload
  const payload = {
    id: event.id,
    type: event.action,
    timestamp: new Date(event.t).toISOString(),
    severity: event.severity,
    actor: {
      type: event.actor_type,
      id: event.actor,
      ip: event.ip,
      user_agent: event.actor_type === 'agent' ? 'shark-sdk/2.3 (node)' : 'Mozilla/5.0 … Chrome/124',
    },
    target: { id: event.target, type: event.target.split('_')[0] || 'unknown' },
    resource: 'https://api.nimbus.sh',
    request_id: 'req_' + Math.random().toString(36).slice(2, 14),
    session_id: event.actor_type === 'user' ? 'sess_' + Math.random().toString(36).slice(2, 10).toUpperCase() : null,
    jti: event.target.startsWith('jti_') ? event.target : null,
    ...(event.action.startsWith('oauth.token') ? {
      oauth: {
        client_id: 'cli_01HNGZK8RXT4A2VPQW',
        scope: 'openid profile email',
        grant_type: 'authorization_code',
        dpop: true,
        act: event.actor_type === 'agent' ? { sub: 'usr_7fK2a8', email: 'amelia@nimbus.sh' } : null,
      },
    } : {}),
    ...(event.action === 'agent.secret.rotated' ? {
      diff: { old_kid: 'sec_2025_Q4', new_kid: 'sec_2026_Q1', reason: 'scheduled' },
    } : {}),
  };

  return (
    <aside style={{
      borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column',
      overflow: 'hidden',
    }}>
      <div className="row" style={{ padding: '14px 16px', borderBottom: '1px solid var(--hairline)', gap: 8 }}>
        <SevDot sev={event.severity}/>
        <h3 className="mono" style={{ fontSize: 13, margin: 0, fontWeight: 500, flex: 1 }}>{event.action}</h3>
        <button className="btn ghost icon sm" onClick={onClose}><Icon.X width={11} height={11}/></button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '14px 16px' }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'auto 1fr', gap: '8px 14px', fontSize: 11.5, marginBottom: 18 }}>
          <span className="faint">Event ID</span>
          <span className="mono">{event.id}</span>
          <span className="faint">Timestamp</span>
          <span className="mono">{new Date(event.t).toISOString()}</span>
          <span className="faint">Actor</span>
          <span>
            <span className="chip" style={{height:16, fontSize:9.5, marginRight:6, textTransform:'uppercase'}}>{event.actor_type}</span>
            <span className="mono">{event.actor}</span>
          </span>
          <span className="faint">Target</span>
          <span className="mono">{event.target}</span>
          {event.ip && <>
            <span className="faint">IP</span>
            <span className="mono">{event.ip}</span>
          </>}
          <span className="faint">Severity</span>
          <span>
            <span className={"chip " + (event.severity === 'danger' ? 'danger' : event.severity === 'warn' ? 'warn' : '')} style={{height:16, fontSize:9.5}}>{event.severity}</span>
          </span>
        </div>

        {/* Delegation chain for agent events */}
        {event.actor_type === 'agent' && (
          <div style={{ marginBottom: 18 }}>
            <h4 style={sectionLabelStyle}>Delegation chain</h4>
            <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, padding: 10, background: 'var(--surface-1)' }}>
              <div className="row" style={{ gap: 8, fontSize: 11 }}>
                <Avatar name="Amelia Chen" email="amelia@nimbus.sh" size={18}/>
                <span>amelia@nimbus.sh</span>
                <span className="faint mono">→</span>
                <span className="chip" style={{height:16, fontSize:9.5}}>act</span>
                <span className="mono">{event.actor}</span>
                <span className="faint mono">→</span>
                <span className="faint">resource</span>
                <span className="mono">api.nimbus.sh</span>
              </div>
              <div className="faint" style={{ fontSize: 10.5, marginTop: 6 }}>
                User delegated via authorization_code · DPoP-bound · exp in 18m
              </div>
            </div>
          </div>
        )}

        <h4 style={sectionLabelStyle}>Payload</h4>
        <pre style={{
          margin: 0, padding: 12,
          background: 'var(--surface-1)', border: '1px solid var(--hairline)', borderRadius: 3,
          fontSize: 10.5, lineHeight: 1.55, fontFamily: 'var(--font-mono)',
          overflowX: 'auto',
          color: 'var(--fg)',
        }}>{JSON.stringify(payload, null, 2)}</pre>

        <h4 style={{...sectionLabelStyle, marginTop: 18}}>Related events</h4>
        <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
          {MOCK.audit.filter(e => e.actor === event.actor && e.id !== event.id).slice(0, 5).map(e => (
            <div key={e.id} className="row" style={{ padding: '6px 10px', borderBottom: '1px solid var(--hairline)', fontSize: 11, gap: 8 }}>
              <SevDot sev={e.severity}/>
              <span className="mono" style={{ flex: 1 }}>{e.action}</span>
              <span className="faint mono" style={{fontSize:10}}>{MOCK.relativeTime(e.t)}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="row" style={{ padding: 12, borderTop: '1px solid var(--hairline)', gap: 6 }}>
        <button className="btn ghost" style={{flex:1}}><Icon.Copy width={11} height={11}/> Copy JSON</button>
        <button className="btn ghost" style={{flex:1}}><Icon.Webhook width={11} height={11}/> Replay</button>
      </div>
    </aside>
  );
}

const thStyle = { textAlign: 'left', padding: '7px 16px', fontSize: 10.5, fontWeight: 500, color: 'var(--fg-dim)', borderBottom: '1px solid var(--hairline)', background: 'var(--surface-0)', position: 'sticky', top: 0, textTransform: 'uppercase', letterSpacing: '0.05em' };
const tdStyle = { padding: '7px 16px', borderBottom: '1px solid var(--hairline)', verticalAlign: 'top' };
const sectionLabelStyle = { fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)', fontWeight: 500, margin: '0 0 8px' };

Object.assign(window, { Audit });
