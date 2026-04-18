// Overview page — metrics, attention panel, activity, auth breakdown

function Overview() {
  const s = MOCK.stats, t = MOCK.trends;
  const metrics = [
    { k: 'Users', v: s.users.total.toLocaleString(), sub: `+${s.users.delta7d} last 7d`, trend: t.users },
    { k: 'Active sessions', v: s.sessions.active.toLocaleString(), sub: '3.2k web · 712 api', trend: t.sessions },
    { k: 'MFA adoption', v: Math.round(s.mfa.pct * 100) + '%', sub: `${s.mfa.enabled.toLocaleString()} of ${s.mfa.total.toLocaleString()}`, trend: t.mfa },
    { k: 'Failed logins 24h', v: s.failedLogins24h.count, sub: `${Math.round(s.failedLogins24h.deltaPct * 100)}% vs prev`, trend: t.failed, good: true },
    { k: 'API keys active', v: s.apiKeys.count, sub: `${s.apiKeys.expiring} expiring · 7d`, trend: t.keys, warn: true },
    { k: 'Agents active', v: s.agents.count, sub: `${s.agents.tokensLive.toLocaleString()} tokens live`, trend: t.agents, agent: true },
  ];

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 16, padding: 16, height: '100%', overflow: 'auto' }}>
      {/* Main column */}
      <div className="col" style={{ gap: 16, minWidth: 0 }}>
        {/* Metric strip */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 8 }}>
          {metrics.map((m, i) => (
            <div key={i} className="card" style={{ padding: 12, minWidth: 0, cursor: 'pointer' }}>
              <div className="row" style={{ justifyContent: 'space-between' }}>
                <span style={{ fontSize: 10.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>{m.k}</span>
                {m.agent && <span className="chip agent" style={{ height: 14, fontSize: 9, padding: '0 4px' }}>new</span>}
                {m.warn && <Icon.Warn width={11} height={11} style={{ color: 'var(--warn)' }}/>}
              </div>
              <div style={{ fontSize: 22, fontWeight: 600, letterSpacing: '-0.02em', marginTop: 4, fontVariantNumeric: 'tabular-nums' }}>{m.v}</div>
              <div style={{ fontSize: 10.5, color: 'var(--fg-dim)', marginTop: 1 }}>{m.sub}</div>
              <div style={{ marginTop: 8, marginLeft: -4, marginRight: -4 }}>
                <Sparkline data={m.trend} height={22} color={m.agent ? 'var(--agent)' : (m.good ? 'var(--success)' : 'var(--fg)')}/>
              </div>
            </div>
          ))}
        </div>

        {/* Auth breakdown + activity */}
        <div style={{ display: 'grid', gridTemplateColumns: '340px 1fr', gap: 16 }}>
          <div className="card">
            <div className="card-header">
              Auth method · 30d
              <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>12,847 sessions</span>
            </div>
            <div className="card-body" style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
              <Donut segments={MOCK.authBreakdown} size={100} thickness={16}/>
              <div className="col" style={{ flex: 1, gap: 5 }}>
                {MOCK.authBreakdown.map((b, i) => {
                  const total = MOCK.authBreakdown.reduce((a, x) => a + x.value, 0);
                  const pct = (b.value / total * 100).toFixed(0);
                  return (
                    <div key={i} className="row" style={{ fontSize: 11.5, gap: 8 }}>
                      <span style={{ width: 8, height: 8, background: b.color, borderRadius: 2, flexShrink: 0 }}/>
                      <span style={{ flex: 1 }}>{b.label}</span>
                      <span className="mono muted" style={{ fontSize: 11, fontVariantNumeric: 'tabular-nums' }}>{b.value.toLocaleString()}</span>
                      <span className="mono" style={{ fontSize: 11, width: 28, textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{pct}%</span>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>

          <div className="card" style={{ minWidth: 0 }}>
            <div className="card-header">
              <span>Recent activity</span>
              <div className="row" style={{ gap: 8 }}>
                <span className="chip" style={{ height: 18, fontSize: 10 }}>
                  <span className="dot success pulse"/>live
                </span>
                <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>updated just now</span>
              </div>
            </div>
            <div style={{ overflow: 'hidden' }}>
              <table className="tbl" style={{ fontSize: 12 }}>
                <tbody>
                  {MOCK.activity.slice(0, 9).map((a, i) => (
                    <tr key={i}>
                      <td style={{ width: 60, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)', fontSize: 10.5 }}>{MOCK.relativeTime(a.t)}</td>
                      <td style={{ width: 90 }}>
                        <span className={"chip" + (a.actor === 'agent' ? ' agent' : a.actor === 'system' ? '' : a.actor === 'admin' ? ' warn' : '')} style={{ height: 16, fontSize: 9.5, padding: '0 5px' }}>{a.actor}</span>
                      </td>
                      <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} className="mono">
                        {a.name}
                      </td>
                      <td className="mono" style={{ color: 'var(--fg)', fontSize: 11.5 }}>{a.action}</td>
                      <td className="mono" style={{ color: 'var(--fg-dim)', fontSize: 11, maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{a.meta}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <div style={{ padding: '8px 12px', borderTop: '1px solid var(--hairline)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 11 }}>
              <span className="faint">Streaming from /admin/audit-logs · 10s poll</span>
              <button className="btn ghost sm">View full log <Icon.ChevronRight width={10} height={10}/></button>
            </div>
          </div>
        </div>

        {/* Agent activity highlights (standout) */}
        <div className="card">
          <div className="card-header">
            <span>Agent activity · 24h</span>
            <div className="row" style={{ gap: 8 }}>
              <span className="chip agent" style={{ height: 18, fontSize: 10 }}>phase 6</span>
              <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 10 }}>1,284 live tokens · 8 agents</span>
            </div>
          </div>
          <div className="card-body" style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 8 }}>
            {MOCK.agents.slice(0, 4).map(a => (
              <div key={a.id} style={{
                padding: 10,
                border: '1px solid var(--hairline)',
                borderRadius: 5,
                background: 'var(--surface-2)',
              }}>
                <div className="row" style={{ gap: 8, marginBottom: 6 }}>
                  <Avatar name={a.name} agent size={22}/>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 500, fontSize: 12 }}>{a.name}</div>
                    <div className="mono faint" style={{ fontSize: 10 }}>{MOCK.relativeTime(a.lastUsed)}</div>
                  </div>
                  {a.dpop && <span className="chip" style={{ height: 15, fontSize: 9, padding: '0 4px' }} title="DPoP bound">DPoP</span>}
                </div>
                <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
                  <span className="mono" style={{ fontSize: 10, color: 'var(--fg)' }}>{a.tokensActive}</span>
                  <span className="faint" style={{ fontSize: 10 }}>tokens</span>
                  <span className="faint" style={{ fontSize: 10 }}>·</span>
                  <span className="mono faint" style={{ fontSize: 10 }}>{a.scopes.length} scopes</span>
                </div>
                <div style={{ marginTop: 8, marginLeft: -4 }}>
                  <Sparkline data={[12,18,14,22,28,24,31,29,42,38,45,52,48,a.tokensActive/10]} height={16} color="var(--agent)"/>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* System health */}
        <div className="card">
          <div className="card-header">System health</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 0 }}>
            {[
              { k: 'Version', v: 'v0.8.2', sub: '2 days old · up-to-date' },
              { k: 'Uptime', v: '18d 04:22', sub: 'since last restart' },
              { k: 'Database', v: '142 MB', sub: 'postgres · healthy' },
              { k: 'Migrations', v: '0047', sub: 'cursor up-to-date' },
              { k: 'JWT mode', v: 'jwt', sub: 'ES256 · 2 keys active' },
              { k: 'SMTP', v: 'shark.email', sub: 'testing tier · 280/500' },
              { k: 'OAuth providers', v: '4', sub: 'google · github · apple · discord' },
              { k: 'SSO connections', v: '2', sub: 'stride.io · apex.dev' },
            ].map((x, i) => (
              <div key={i} style={{
                padding: '12px 14px',
                borderRight: i % 4 !== 3 ? '1px solid var(--hairline)' : 'none',
                borderTop: i >= 4 ? '1px solid var(--hairline)' : 'none',
              }}>
                <div style={{ fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)' }}>{x.k}</div>
                <div style={{ fontSize: 14, fontWeight: 500, marginTop: 4, fontFamily: 'var(--font-mono)' }}>{x.v}</div>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 2 }}>{x.sub}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Attention rail */}
      <AttentionPanel/>
    </div>
  );
}

function AttentionPanel() {
  const [dismissed, setDismissed] = React.useState(new Set());
  const items = MOCK.attention.filter(x => !dismissed.has(x.id));

  return (
    <div className="card" style={{ alignSelf: 'start', position: 'sticky', top: 0 }}>
      <div className="card-header">
        <div className="row" style={{ gap: 8 }}>
          <span>Attention</span>
          <span className="chip danger" style={{ height: 16, fontSize: 10 }}>{items.length}</span>
        </div>
        <button className="btn ghost sm" title="Refresh"><Icon.Refresh width={11} height={11}/></button>
      </div>
      <div>
        {items.length === 0 && (
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--fg-dim)' }}>
            <Icon.Check width={20} height={20} style={{ color: 'var(--success)', marginBottom: 6 }}/>
            <div style={{ fontSize: 12 }}>All clear. Nothing needs your attention.</div>
          </div>
        )}
        {items.map(item => (
          <div key={item.id} style={{
            padding: 12,
            borderBottom: '1px solid var(--hairline)',
            display: 'flex', gap: 10, alignItems: 'flex-start',
          }}>
            <div style={{
              width: 22, height: 22, borderRadius: 4, flexShrink: 0,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: item.severity === 'danger' ? 'var(--danger-bg)' : item.severity === 'warn' ? 'var(--warn-bg)' : 'var(--surface-3)',
              color: item.severity === 'danger' ? 'var(--danger)' : item.severity === 'warn' ? 'var(--warn)' : 'var(--info)',
              marginTop: 1,
            }}>
              {item.kind === 'device' && <Icon.Device width={12} height={12}/>}
              {item.kind === 'webhook' && <Icon.Webhook width={12} height={12}/>}
              {item.kind === 'key' && <Icon.Key width={12} height={12}/>}
              {item.kind === 'config' && <Icon.Info width={12} height={12}/>}
              {item.kind === 'anomaly' && <Icon.Warn width={12} height={12}/>}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 12.5, fontWeight: 500, lineHeight: 1.35 }}>{item.title}</div>
              <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginTop: 3, lineHeight: 1.4 }}>{item.sub}</div>
              <div className="row" style={{ marginTop: 8, gap: 6 }}>
                <button className="btn sm" style={{ height: 22, fontSize: 11 }}>{item.action}</button>
                <button onClick={() => setDismissed(new Set([...dismissed, item.id]))}
                  className="btn ghost sm" style={{ height: 22, fontSize: 11 }}>Dismiss</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <div style={{ padding: '10px 12px', fontSize: 11, color: 'var(--fg-dim)', display: 'flex', justifyContent: 'space-between' }}>
        <span>Auto-triaged 4 low-severity items</span>
        <button className="btn ghost sm" style={{ height: 20, padding: 0, fontSize: 11 }}>Show</button>
      </div>
    </div>
  );
}

window.Overview = Overview;
