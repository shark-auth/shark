// @ts-nocheck
import React from 'react'
import { Icon, Avatar, Sparkline, Donut, CopyField } from './shared'
import { useAPI } from './api'
import { MOCK } from './mock'

// Overview page — metrics, attention panel, activity, auth breakdown

export function Overview() {
  // --- Real API calls ---
  const { data: statsRaw, loading: statsLoading } = useAPI('/admin/stats');
  const { data: trendsRaw } = useAPI('/admin/stats/trends?days=14');
  const { data: healthRaw } = useAPI('/admin/health');
  const { data: activityRaw, refresh: refreshActivity } = useAPI('/audit-logs?limit=20&per_page=20');

  // 10-second polling for activity feed
  React.useEffect(() => {
    const id = setInterval(refreshActivity, 10000);
    return () => clearInterval(id);
  }, [refreshActivity]);

  // --- Map /admin/stats to metric cards ---
  // API shape: { users:{total,created_last_7d}, sessions:{active}, mfa:{enabled,total},
  //              failed_logins_24h, api_keys:{active,expiring_7d}, sso_connections:{total,enabled} }
  function mapStats(s) {
    if (!s) return null;
    return {
      users: {
        total: s.users?.total ?? 0,
        delta7d: s.users?.created_last_7d ?? 0,
      },
      sessions: {
        active: s.sessions?.active ?? 0,
      },
      mfa: {
        pct: s.mfa?.total > 0 ? (s.mfa.enabled / s.mfa.total) : 0,
        enabled: s.mfa?.enabled ?? 0,
        total: s.mfa?.total ?? 0,
      },
      failedLogins24h: {
        count: s.failed_logins_24h ?? 0,
        deltaPct: 0, // not provided by API
      },
      apiKeys: {
        count: s.api_keys?.active ?? 0,
        expiring: s.api_keys?.expiring_7d ?? 0,
      },
      // No agent-specific endpoint yet — fall back to MOCK
      agents: MOCK.stats.agents,
    };
  }

  // --- Map /admin/stats/trends to sparkline arrays ---
  // API shape: { signups:[{date,count},...], auth_methods:{password,oauth,passkey,magic_link} }
  function mapTrends(t) {
    const signupCounts = Array.isArray(t?.signups)
      ? t.signups.map(p => p.count)
      : null;
    return {
      users:    signupCounts || MOCK.trends.users,
      sessions: MOCK.trends.sessions, // not in API
      mfa:      MOCK.trends.mfa,       // not in API
      failed:   MOCK.trends.failed,    // not in API
      keys:     MOCK.trends.keys,      // not in API
      agents:   MOCK.trends.agents,    // not in API
    };
  }

  // --- Map auth_methods fractions to donut segments ---
  // auth_methods values are fractions (0–1); scale to integer counts proportionally
  // Uses a nominal total of 10000 so relative sizes are preserved
  const AUTH_COLORS = {
    password:   '#e4e4e4',
    oauth:      '#888',
    passkey:    '#555',
    magic_link: '#3a3a3a',
  };
  const AUTH_LABELS = {
    password:   'Password',
    oauth:      'OAuth',
    passkey:    'Passkey',
    magic_link: 'Magic link',
  };

  function mapAuthBreakdown(t) {
    const am = t?.auth_methods;
    if (!am) return null;
    const total = 10000;
    return Object.entries(am).map(([key, frac]) => ({
      label: AUTH_LABELS[key] || key,
      value: Math.round(frac * total),
      color: AUTH_COLORS[key] || '#aaa',
    })).filter(x => x.value > 0);
  }

  // --- Map /admin/health ---
  // Expected shape (best-effort): { version, uptime_seconds, db:{size_mb,status},
  //   migrations:{current}, jwt:{mode,algorithm,active_keys},
  //   smtp:{host,tier,sent_today,daily_limit}, oauth_providers:N, sso_connections:N }
  function mapHealth(h) {
    if (!h) return null;
    const uptimeSec = h.uptime_seconds ?? null;
    function formatUptime(s) {
      if (s == null) return '—';
      const d = Math.floor(s / 86400);
      const hh = Math.floor((s % 86400) / 3600).toString().padStart(2, '0');
      const mm = Math.floor((s % 3600) / 60).toString().padStart(2, '0');
      return `${d}d ${hh}:${mm}`;
    }
    return [
      { k: 'Version',         v: h.version ?? '—',                       sub: h.version_age ?? 'up-to-date' },
      { k: 'Uptime',          v: formatUptime(uptimeSec),                 sub: 'since last restart' },
      { k: 'Database',        v: h.db?.size_mb != null ? `${h.db.size_mb} MB` : '—',  sub: `${h.db?.driver ?? 'postgres'} · ${h.db?.status ?? 'healthy'}` },
      { k: 'Migrations',      v: h.migrations?.current ?? '—',           sub: 'cursor up-to-date' },
      { k: 'JWT mode',        v: h.jwt?.mode ?? '—',                     sub: `${h.jwt?.algorithm ?? ''} · ${h.jwt?.active_keys ?? '?'} keys active` },
      { k: 'SMTP',            v: h.smtp?.host ?? '—',                    sub: h.smtp?.tier != null ? `${h.smtp.tier} tier · ${h.smtp.sent_today ?? 0}/${h.smtp.daily_limit ?? '?'}` : '' },
      { k: 'OAuth providers', v: h.oauth_providers != null ? String(h.oauth_providers) : '—', sub: h.oauth_providers_names ?? '' },
      { k: 'SSO connections', v: h.sso_connections != null ? String(h.sso_connections) : '—', sub: h.sso_connections_names ?? '' },
    ];
  }

  // --- Map audit log entries to activity row format ---
  // API shape (audit log): [{ id, created_at, event_type|action, actor_type, actor_id,
  //                           actor_email, target_id, metadata, ip }]
  function mapActivity(raw) {
    if (!Array.isArray(raw?.items || raw)) return null;
    const items = raw?.items || raw;
    return items.map(e => {
      // actor display
      const actorType = e.actor_type || 'system';
      const actorName = e.actor_email || e.actor_id || e.actor || 'system';
      // action
      const action = e.event_type || e.action || '—';
      // meta: pull from metadata object if present
      let meta = '—';
      if (e.metadata) {
        if (typeof e.metadata === 'string') meta = e.metadata;
        else {
          const pairs = Object.entries(e.metadata)
            .slice(0, 2)
            .map(([k, v]) => `${k}: ${v}`)
            .join(' · ');
          if (pairs) meta = pairs;
        }
      }
      // target
      const name = actorName.length > 24 ? actorName.slice(0, 24) : actorName;
      // timestamp — prefer created_at ISO string, fall back to t
      const t = e.created_at ? new Date(e.created_at).getTime() : (e.t || Date.now());
      return { t, actor: actorType, name, action, meta, target: e.target_id || '—' };
    });
  }

  // Resolve data with fallbacks
  const stats    = mapStats(statsRaw);
  const trends   = mapTrends(trendsRaw);
  const authData = mapAuthBreakdown(trendsRaw) || MOCK.authBreakdown;
  const health   = mapHealth(healthRaw);
  const activity = mapActivity(activityRaw) || MOCK.activity;

  // Use real stats if available, otherwise fall through to MOCK
  const s = stats || MOCK.stats;
  const t = trends;

  const metrics = [
    { k: 'Users',            v: s.users.total.toLocaleString(),                       sub: `+${s.users.delta7d} last 7d`,                                    trend: t.users },
    { k: 'Active sessions',  v: s.sessions.active.toLocaleString(),                   sub: '—',                                                               trend: t.sessions },
    { k: 'MFA adoption',     v: Math.round(s.mfa.pct * 100) + '%',                   sub: `${s.mfa.enabled.toLocaleString()} of ${s.mfa.total.toLocaleString()}`, trend: t.mfa },
    { k: 'Failed logins 24h',v: s.failedLogins24h.count,                              sub: `${Math.round(s.failedLogins24h.deltaPct * 100)}% vs prev`,       trend: t.failed, good: true },
    { k: 'API keys active',  v: s.apiKeys.count,                                      sub: `${s.apiKeys.expiring} expiring · 7d`,                             trend: t.keys, warn: true },
    { k: 'Agents active',    v: s.agents.count,                                       sub: `${s.agents.tokensLive.toLocaleString()} tokens live`,             trend: t.agents, agent: true },
  ];

  // Skeleton helpers
  function SkeletonBox({ w, h, style }) {
    return <div style={{ width: w || '100%', height: h || 14, background: 'var(--surface-3)', borderRadius: 4, ...style }}/>;
  }

  // Activity timestamp — use live relative time if available, else MOCK helper
  function relTime(t) {
    const diff = Math.floor((Date.now() - t) / 1000);
    if (diff < 0) return 'soon';
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 16, padding: 16, height: '100%', overflow: 'auto' }}>
      {/* Main column */}
      <div className="col" style={{ minWidth: 0 }}>
        {/* Metric strip */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 8 }}>
          {statsLoading
            ? Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="card" style={{ padding: 12, minWidth: 0 }}>
                  <SkeletonBox h={10} w="60%" style={{ marginBottom: 6 }}/>
                  <SkeletonBox h={24} w="80%" style={{ marginBottom: 4 }}/>
                  <SkeletonBox h={10} w="70%"/>
                  <SkeletonBox h={22} style={{ marginTop: 10 }}/>
                </div>
              ))
            : metrics.map((m, i) => (
                <div key={i} className="card" style={{ padding: 12, minWidth: 0, cursor: 'pointer' }}>
                  <div className="row" style={{ justifyContent: 'space-between' }}>
                    <span style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', lineHeight: 1.5 }}>{m.k}</span>
                    {m.agent && <span className="chip agent" style={{ height: 14, fontSize: 10, padding: '0 4px' }}>new</span>}
                    {m.warn && <Icon.Warn width={11} height={11} style={{ color: 'var(--warn)' }}/>}
                  </div>
                  <div style={{ fontSize: 20, fontWeight: 600, fontFamily: 'var(--font-display)', letterSpacing: '-0.02em', lineHeight: 1.15, marginTop: 4, fontVariantNumeric: 'tabular-nums' }}>{m.v}</div>
                  <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 1, lineHeight: 1.5 }}>{m.sub}</div>
                  <div style={{ marginTop: 8, marginLeft: -4, marginRight: -4 }}>
                    <Sparkline data={m.trend} height={22} color={m.agent ? 'var(--agent)' : (m.good ? 'var(--success)' : 'var(--fg)')}/>
                  </div>
                </div>
              ))
          }
        </div>

        {/* Auth breakdown + activity */}
        <div style={{ display: 'grid', gridTemplateColumns: '340px 1fr', gap: 16, marginTop: 20 }}>
          <div className="card">
            <div className="card-header">
              Auth method · 30d
              <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 11 }}>
                {authData.reduce((a, x) => a + x.value, 0).toLocaleString()} sessions
              </span>
            </div>
            <div className="card-body" style={{ display: 'flex', gap: 16, alignItems: 'center' }}>
              <Donut segments={authData} size={100} thickness={16}/>
              <div className="col" style={{ flex: 1, gap: 5 }}>
                {authData.map((b, i) => {
                  const total = authData.reduce((a, x) => a + x.value, 0);
                  const pct = (b.value / total * 100).toFixed(0);
                  return (
                    <div key={i} className="row" style={{ fontSize: 13, gap: 8 }}>
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
                <span className="chip" style={{ height: 18, fontSize: 11 }}>
                  <span className="dot success pulse"/>live
                </span>
                <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 11 }}>updated just now</span>
              </div>
            </div>
            <div style={{ overflow: 'hidden' }}>
              <table className="tbl" style={{ fontSize: 13 }}>
                <tbody>
                  {!activityRaw
                    ? Array.from({ length: 9 }).map((_, i) => (
                        <tr key={i}>
                          <td style={{ width: 60 }}><SkeletonBox h={10} w={40}/></td>
                          <td style={{ width: 90 }}><SkeletonBox h={14} w={60}/></td>
                          <td><SkeletonBox h={10} w={100}/></td>
                          <td><SkeletonBox h={10} w={120}/></td>
                          <td><SkeletonBox h={10} w={160}/></td>
                        </tr>
                      ))
                    : activity.slice(0, 9).map((a, i) => (
                        <tr key={i}>
                          <td style={{ width: 60, color: 'var(--fg-muted)', fontFamily: 'var(--font-mono)', fontSize: 11, lineHeight: 1.5 }}>{relTime(a.t)}</td>
                          <td style={{ width: 90 }}>
                            <span className={"chip" + (a.actor === 'agent' ? ' agent' : a.actor === 'system' ? '' : a.actor === 'admin' ? ' warn' : '')} style={{ height: 16, fontSize: 10, padding: '0 5px' }}>{a.actor}</span>
                          </td>
                          <td style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} className="mono">
                            {a.name}
                          </td>
                          <td className="mono" style={{ color: 'var(--fg)', fontSize: 13 }}>{a.action}</td>
                          <td className="mono" style={{ color: 'var(--fg-muted)', fontSize: 11, lineHeight: 1.5, maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{a.meta}</td>
                        </tr>
                      ))
                  }
                </tbody>
              </table>
            </div>
            <div style={{ padding: '8px 12px', borderTop: '1px solid var(--hairline)', display: 'flex', justifyContent: 'space-between', alignItems: 'center', fontSize: 11 }}>
              <span className="faint">Streaming from /admin/audit-logs · 10s poll</span>
              <button className="btn ghost sm">View full log <Icon.ChevronRight width={10} height={10}/></button>
            </div>
          </div>
        </div>

        {/* Agent activity highlights — compact list (no nested cards) */}
        <div className="card" style={{ marginTop: 16 }}>
          <div className="card-header">
            <span>Agent activity · 24h</span>
            <div className="row" style={{ gap: 8 }}>
              <span className="chip agent" style={{ height: 18, fontSize: 11 }}>phase 6</span>
              <span className="faint mono" style={{ textTransform: 'none', letterSpacing: 0, fontSize: 11 }}>1,284 live tokens · 8 agents</span>
            </div>
          </div>
          <div style={{ padding: '4px 0' }}>
            {MOCK.agents.slice(0, 4).map((a, i) => (
              <div key={a.id} style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '8px 14px',
                borderBottom: i < 3 ? '1px solid var(--hairline)' : 'none',
              }}>
                <Avatar name={a.name} agent size={22}/>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
                    <span style={{ fontWeight: 600, fontSize: 13, fontFamily: 'var(--font-display)' }}>{a.name}</span>
                    {a.dpop && <span className="chip" style={{ height: 14, fontSize: 10, padding: '0 4px' }} title="DPoP bound">DPoP</span>}
                    <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{MOCK.relativeTime(a.lastUsed)}</span>
                  </div>
                  <div style={{ display: 'flex', gap: 6, marginTop: 2, alignItems: 'center' }}>
                    <span className="mono" style={{ fontSize: 11, color: 'var(--fg)', lineHeight: 1.5 }}>{a.tokensActive}</span>
                    <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>tokens</span>
                    <span className="faint" style={{ fontSize: 11 }}>·</span>
                    <span className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{a.scopes.length} scopes</span>
                  </div>
                </div>
                <div style={{ width: 80, flexShrink: 0 }}>
                  <Sparkline data={[12,18,14,22,28,24,31,29,42,38,45,52,48,a.tokensActive/10]} height={16} color="var(--agent)"/>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* System health */}
        <div className="card" style={{ marginTop: 16 }}>
          <div className="card-header">System health</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 0 }}>
            {(health || [
              { k: 'Version',         v: 'v0.8.2',      sub: '2 days old · up-to-date' },
              { k: 'Uptime',          v: '18d 04:22',   sub: 'since last restart' },
              { k: 'Database',        v: '142 MB',      sub: 'postgres · healthy' },
              { k: 'Migrations',      v: '0047',        sub: 'cursor up-to-date' },
              { k: 'JWT mode',        v: 'jwt',         sub: 'ES256 · 2 keys active' },
              { k: 'SMTP',            v: 'shark.email', sub: 'testing tier · 280/500' },
              { k: 'OAuth providers', v: '4',           sub: 'google · github · apple · discord' },
              { k: 'SSO connections', v: '2',           sub: 'stride.io · apex.dev' },
            ]).map((x, i) => (
              <div key={i} style={{
                padding: '12px 14px',
                borderRight: i % 4 !== 3 ? '1px solid var(--hairline)' : 'none',
                borderTop: i >= 4 ? '1px solid var(--hairline)' : 'none',
              }}>
                <div style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', lineHeight: 1.5 }}>{x.k}</div>
                <div style={{ fontSize: 13, fontWeight: 600, marginTop: 4, fontFamily: 'var(--font-display)', letterSpacing: '-0.01em', color: 'var(--fg)', fontVariantNumeric: 'tabular-nums' }}>{x.v}</div>
                <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 2, lineHeight: 1.5 }}>{x.sub}</div>
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
          <div style={{ padding: 24, textAlign: 'center', color: 'var(--fg-muted)' }}>
            <Icon.Check width={20} height={20} style={{ color: 'var(--success)', marginBottom: 6 }}/>
            <div style={{ fontSize: 13 }}>All clear. Nothing needs your attention.</div>
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
              <div style={{ fontSize: 13, fontWeight: 600, lineHeight: 1.4, fontFamily: 'var(--font-display)' }}>{item.title}</div>
              <div style={{ fontSize: 11, color: 'var(--fg-muted)', marginTop: 3, lineHeight: 1.5 }}>{item.sub}</div>
              <div className="row" style={{ marginTop: 8, gap: 6 }}>
                <button className="btn sm" style={{ height: 22, fontSize: 11 }}>{item.action}</button>
                <button onClick={() => setDismissed(new Set([...dismissed, item.id]))}
                  className="btn ghost sm" style={{ height: 22, fontSize: 11 }}>Dismiss</button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <div style={{ padding: '10px 12px', fontSize: 11, color: 'var(--fg-muted)', lineHeight: 1.5, display: 'flex', justifyContent: 'space-between' }}>
        <span>Auto-triaged 4 low-severity items</span>
        <button className="btn ghost sm" style={{ height: 20, padding: 0, fontSize: 11 }}>Show</button>
      </div>
    </div>
  );
}

