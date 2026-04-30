// @ts-nocheck
import React from 'react'
import { Icon, Sparkline, Donut } from './shared'
import { useAPI } from './api'

// useAgentMetrics — derives total, active, created-this-week, and a 7-day sparkline
// from /api/v1/agents (client-side aggregation; no new backend endpoints needed).
//
// "Active" definition: agent has active=true AND issued ≥1 token in the last 7 days.
// We approximate the "issued ≥1 token" part via audit-logs token.exchange events.
// If audit-logs are unavailable, we fall back to active=true only.
function useAgentMetrics() {
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(false);

  const load = React.useCallback(() => {
    const key = localStorage.getItem('shark_admin_key');
    if (!key) { setLoading(false); return; }
    const headers = { Authorization: 'Bearer ' + key };

    const since7d = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();

    Promise.all([
      // All agents (cap 500 to cover most deployments)
      fetch('/api/v1/agents?limit=500', { headers }).then(r => r.ok ? r.json() : { data: [], total: 0 }),
      // Audit: token.exchange events last 7d — used to determine which agents are "active"
      fetch(`/api/v1/admin/audit-logs?action=token.exchange&from=${encodeURIComponent(since7d)}&limit=500`, { headers })
        .then(r => r.ok ? r.json() : { data: [] })
        .catch(() => ({ data: [] })),
    ]).then(([agentsResp, auditResp]) => {
      const agents = Array.isArray(agentsResp?.data) ? agentsResp.data : [];
      const total = agentsResp?.total ?? agents.length;

      // Build set of agent client_ids that issued a token in last 7d
      const auditEvents = Array.isArray(auditResp?.data) ? auditResp.data : [];
      const activeClientIds = new Set(
        auditEvents
          .map(e => e.client_id || e.actor_id || e.metadata?.client_id)
          .filter(Boolean)
      );

      // Active = not deactivated (active===true) AND issued ≥1 token last 7d
      // If no audit events at all, fall back to active flag only
      const hasAuditData = auditEvents.length > 0 || activeClientIds.size > 0;
      const activeCount = agents.filter(ag =>
        ag.active && (hasAuditData ? (activeClientIds.has(ag.client_id) || activeClientIds.has(ag.id)) : true)
      ).length;

      // Agents created this week (client-side)
      const cutoff = Date.now() - 7 * 24 * 60 * 60 * 1000;
      const createdThisWeek = agents.filter(ag => new Date(ag.created_at).getTime() >= cutoff).length;

      // Build 7-day sparkline: count agents created per day (day 0 = 7d ago, day 6 = today)
      const dayCounts = Array(7).fill(0);
      for (const ag of agents) {
        const ageMs = Date.now() - new Date(ag.created_at).getTime();
        const dayIdx = Math.floor(ageMs / (24 * 60 * 60 * 1000));
        if (dayIdx >= 0 && dayIdx < 7) {
          // dayIdx 0 = today, map to sparkline index 6-dayIdx
          dayCounts[6 - dayIdx]++;
        }
      }

      setData({ total, activeCount, createdThisWeek, sparkline: dayCounts });
      setLoading(false);
      setError(false);
    }).catch(() => {
      setError(true);
      setLoading(false);
    });
  }, []);

  React.useEffect(() => {
    load();
    const id = setInterval(load, 60000);
    return () => clearInterval(id);
  }, [load]);

  return { data, loading, error, refresh: load };
}

// Poll /admin/proxy/status just for the 404 (disabled) signal.
function useProxyConfigured() {
  const [configured, setConfigured] = React.useState(null); // null = unknown
  React.useEffect(() => {
    const key = localStorage.getItem('shark_admin_key');
    if (!key) return;
    let cancelled = false;
    fetch('/api/v1/admin/proxy/status', { headers: { Authorization: 'Bearer ' + key } })
      .then(res => {
        if (cancelled) return;
        if (res.status === 404) setConfigured(false);
        else if (res.ok) setConfigured(true);
        else setConfigured(null);
      })
      .catch(() => { if (!cancelled) setConfigured(null); });
    return () => { cancelled = true; };
  }, []);
  return configured;
}

// Magical-moment hero tile shown on cold-start.
function MagicalMomentTile({ onGo, onSkip }) {
  return (
    <div className="card" style={{
      padding: '28px 32px',
      background: 'linear-gradient(135deg, var(--surface-2), var(--surface-1))',
      border: '1px solid var(--hairline-strong)',
    }}>
      <div className="row" style={{ gap: 8, marginBottom: 10 }}>
        <span className="chip agent" style={{ height: 18, fontSize: 10 }}>Magical moment</span>
        <span className="faint mono" style={{ fontSize: 11 }}>~60s · no code changes</span>
      </div>
      <h2 style={{
        margin: 0, fontSize: 26, fontWeight: 600,
        fontFamily: 'var(--font-display)', letterSpacing: '-0.02em', lineHeight: 1.15,
      }}>
        Add auth to any app in 60 seconds
      </h2>
      <p style={{
        margin: '8px 0 18px',
        fontSize: 13.5, color: 'var(--fg-muted)', lineHeight: 1.55, maxWidth: 620,
      }}>
        SharkAuth can protect any upstream with zero code changes. Paste a URL,
        pick routes, done.
      </p>
      <div className="row" style={{ gap: 12 }}>
        <button className="btn primary" onClick={onGo}>
          Configure proxy <Icon.ChevronRight width={11} height={11}/>
        </button>
        <button className="btn ghost sm" onClick={onSkip} style={{ fontSize: 12 }}>
          Skip, show me the dashboard
        </button>
      </div>
    </div>
  );
}

// Real-time activity stream via SSE.
function useRealtimeActivity() {
  const [events, setEvents] = React.useState([]);
  const [status, setStatus] = React.useState('connecting');
  const [retryCount, setRetryCount] = React.useState(0);

  React.useEffect(() => {
    const key = localStorage.getItem('shark_admin_key');
    if (!key) return;

    let timer = null;
    const url = `/api/v1/admin/logs/stream?token=${key}`;
    const es = new EventSource(url);

    es.onopen = () => {
      setStatus('live');
      setRetryCount(0);
      if (timer) clearTimeout(timer);
    };

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        if (data.event === 'connected') return;
        setEvents(prev => [data, ...prev].slice(0, 50));
      } catch (err) {
        console.error('SSE parse error:', err);
      }
    };

    es.onerror = (e) => {
      console.error('SSE error:', e);
      setStatus('reconnecting');
      es.close();
      // Backoff retry
      timer = setTimeout(() => {
        setRetryCount(prev => prev + 1);
      }, Math.min(1000 * Math.pow(2, retryCount), 30000));
    };

    return () => {
      es.close();
      if (timer) clearTimeout(timer);
    };
  }, [retryCount]);

  return { events, status };
}

// Overview page — metrics, attention panel, activity, auth breakdown

export function Overview({ setPage } = {}) {
  const { data: statsRaw, loading: statsLoading } = useAPI('/admin/stats');
  const proxyConfigured = useProxyConfigured();
  const [heroHidden, setHeroHidden] = React.useState(
    () => localStorage.getItem('shark_hide_hero') === '1'
  );

  // Tutorial tab: visible after first-boot banner dismissed, until walkthrough is seen.
  const [walkthroughSeen, setWalkthroughSeen] = React.useState(
    () => localStorage.getItem('shark_walkthrough_seen') === '1'
  );
  const firstBootDone = localStorage.getItem('shark_first_boot_done') === '1';
  const showTutorialTab = firstBootDone && !walkthroughSeen;

  const dismissTutorialTab = () => {
    try { localStorage.setItem('shark_walkthrough_seen', '1'); } catch {}
    setWalkthroughSeen(true);
  };

  const openGetStarted = () => {
    if (typeof setPage === 'function') {
      setPage('get-started');
    } else {
      window.history.pushState(null, '', '/admin/get-started');
      window.dispatchEvent(new PopStateEvent('popstate'));
    }
  };
  const usersZero = (statsRaw?.users?.total ?? null) === 0;
  const showHero = !heroHidden && usersZero && proxyConfigured === false;
  
  const goConfigureProxy = () => {
    if (typeof setPage === 'function') {
      setPage('proxy', { new: '1' });
    } else {
      window.history.pushState(null, '', '/admin/proxy?new=1');
      window.dispatchEvent(new PopStateEvent('popstate'));
    }
  };
  
  const dismissHero = () => {
    try { localStorage.setItem('shark_hide_hero', '1'); } catch {}
    setHeroHidden(true);
  };

  const { data: trendsRaw } = useAPI('/admin/stats/trends?days=14');
  const { data: healthRaw, loading: healthLoading, error: healthError, refresh: refreshHealth } = useAPI('/admin/health');
  const { data: agentMetrics, loading: agentMetricsLoading, error: agentMetricsError } = useAgentMetrics();
  const { events: liveActivity, status: streamStatus } = useRealtimeActivity();

  function mapStats(s) {
    if (!s) return null;
    return {
      users: { total: s.users?.total ?? 0, delta7d: s.users?.created_last_7d ?? 0 },
      sessions: { active: s.sessions?.active ?? 0 },
      mfa: { pct: s.mfa?.total > 0 ? (s.mfa.enabled / s.mfa.total) : 0, enabled: s.mfa?.enabled ?? 0, total: s.mfa?.total ?? 0 },
      failedLogins24h: { count: s.failed_logins_24h ?? 0, deltaPct: 0 },
      apiKeys: { count: s.api_keys?.active ?? 0, expiring: s.api_keys?.expiring_7d ?? 0 },
    };
  }

  function mapTrends(t) {
    return {
      users: Array.isArray(t?.signups_by_day) ? t.signups_by_day.map(p => p.count) : null,
      sessions: Array.isArray(t?.sessions_by_day) ? t.sessions_by_day.map(p => p.count) : null,
      mfa: Array.isArray(t?.mfa_by_day) ? t.mfa_by_day.map(p => p.count) : null,
      failed: Array.isArray(t?.failed_by_day) ? t.failed_by_day.map(p => p.count) : null,
      keys: Array.isArray(t?.api_keys_by_day) ? t.api_keys_by_day.map(p => p.count) : null,
    };
  }

  const AUTH_COLORS = { password: '#e4e4e4', oauth: '#888', passkey: '#555', magic_link: '#3a3a3a' };
  const AUTH_LABELS = { password: 'Password', oauth: 'OAuth', passkey: 'Passkey', magic_link: 'Magic link' };

  function mapAuthBreakdown(t) {
    const am = t?.auth_methods;
    if (!Array.isArray(am) || am.length === 0) return null;
    const total = am.reduce((acc, x) => acc + (x.count || 0), 0);
    if (total === 0) return null;
    return am.filter(x => x.count > 0).map(x => ({
      label: AUTH_LABELS[x.auth_method] || x.auth_method,
      value: x.count,
      color: AUTH_COLORS[x.auth_method] || '#aaa',
    }));
  }

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
      { k: 'Version', v: h.version ?? '—', sub: h.version_age ?? 'up-to-date' },
      { k: 'Uptime', v: formatUptime(uptimeSec), sub: 'since last restart' },
      { k: 'Database', v: h.db?.size_mb != null ? `${h.db.size_mb} MB` : '—', sub: `${h.db?.driver ?? 'sqlite'} · ${h.db?.status ?? 'healthy'}` },
      { k: 'Migrations', v: h.migrations?.current ?? '—', sub: 'cursor up-to-date' },
      { k: 'JWT mode', v: h.jwt?.mode ?? '—', sub: `${h.jwt?.algorithm ?? ''} · ${h.jwt?.active_keys ?? '?'} keys active` },
      { k: 'SMTP', v: h.smtp?.host ?? '—', sub: h.smtp?.tier != null ? `${h.smtp.tier} tier` : '—' },
      { k: 'OAuth', v: h.oauth_providers?.length || 0, sub: (h.oauth_providers || []).join(', ') },
      { k: 'SSO', v: h.sso_connections || 0, sub: 'connections' },
    ];
  }

  function mapActivity(items) {
    if (!Array.isArray(items)) return [];
    return items.map(e => {
      const actorType = e.actor_type || 'system';
      const actorName = e.actor_email || e.actor_id || e.actor || 'system';
      const action = e.event_type || e.action || '—';
      let meta = '—';
      if (e.metadata) {
        if (typeof e.metadata === 'string') meta = e.metadata;
        else {
          const pairs = Object.entries(e.metadata).slice(0, 2).map(([k, v]) => `${k}: ${v}`).join(' · ');
          if (pairs) meta = pairs;
        }
      }
      const t = e.created_at ? new Date(e.created_at).getTime() : Date.now();
      return { t, actor: actorType, name: actorName.length > 24 ? actorName.slice(0, 24) : actorName, action, meta };
    });
  }

  const stats = mapStats(statsRaw) || { users: { total: 0, delta7d: 0 }, sessions: { active: 0 }, mfa: { pct: 0, enabled: 0, total: 0 }, failedLogins24h: { count: 0, deltaPct: 0 }, apiKeys: { count: 0, expiring: 0 } };
  const trends = mapTrends(trendsRaw);
  const authData = mapAuthBreakdown(trendsRaw);
  const health = mapHealth(healthRaw);
  const activity = mapActivity(liveActivity);

  // Agent metric helpers — resolve loading/error states cleanly
  const agentTotal = agentMetricsLoading ? null : agentMetricsError ? 'err' : (agentMetrics?.total ?? 0);
  const agentActive = agentMetricsLoading ? null : agentMetricsError ? 'err' : (agentMetrics?.activeCount ?? 0);
  const agentThisWeek = agentMetricsLoading ? null : agentMetricsError ? 'err' : (agentMetrics?.createdThisWeek ?? 0);
  const agentSparkline = agentMetrics?.sparkline ?? null;

  const ACTIVE_AGENT_TOOLTIP = 'Active = issued ≥1 token in the last 7 days, not deactivated';

  const metrics = [
    { k: 'Users', v: stats.users.total.toLocaleString(), sub: `+${stats.users.delta7d} last 7d`, trend: trends.users, path: 'users' },
    { k: 'Active sessions', v: stats.sessions.active.toLocaleString(), sub: '—', trend: trends.sessions, path: 'sessions' },
    { k: 'MFA adoption', v: Math.round(stats.mfa.pct * 100) + '%', sub: `${stats.mfa.enabled.toLocaleString()} enabled`, trend: trends.mfa, path: 'users', extra: { mfa: 'true' } },
    { k: 'Failed logins 24h', v: stats.failedLogins24h.count, sub: '—', trend: trends.failed, good: true, path: 'audit', extra: { status: 'failure', action: 'user.login' } },
    { k: 'API keys active', v: stats.apiKeys.count, sub: `${stats.apiKeys.expiring} expiring`, trend: trends.keys, warn: true, path: 'keys' },
    { k: 'Total agents', v: agentTotal, sub: `+${agentThisWeek === null ? '…' : agentThisWeek === 'err' ? '?' : agentThisWeek} this week`, trend: agentSparkline, agent: true, agentLoading: agentMetricsLoading, agentError: agentMetricsError, path: 'agents' },
    { k: 'Active agents', v: agentActive, sub: null, trend: agentSparkline, agent: true, agentLoading: agentMetricsLoading, agentError: agentMetricsError, activeTooltip: ACTIVE_AGENT_TOOLTIP, path: 'agents' },
  ];

  function SkeletonBox({ w, h, style }) {
    return <div style={{ width: w || '100%', height: h || 14, background: 'var(--surface-3)', borderRadius: 4, ...style }}/>;
  }

  function relTime(t) {
    const diff = Math.floor((Date.now() - t) / 1000);
    if (diff < 0) return 'now';
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    return `${Math.floor(diff / 3600)}h ago`;
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {showTutorialTab && (
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '8px 16px',
          background: 'var(--surface-2)',
          borderBottom: '1px solid var(--hairline)',
          flexShrink: 0,
          fontSize: 13,
        }}>
          <span style={{ flex: 1, color: 'var(--fg-muted)' }}>Finish setup tour →</span>
          <button
            onClick={openGetStarted}
            style={{
              fontSize: 12,
              padding: '3px 10px',
              background: 'none',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4,
              cursor: 'pointer',
              color: 'var(--fg)',
              fontFamily: 'inherit',
            }}
          >Open</button>
          <button
            onClick={dismissTutorialTab}
            style={{
              fontSize: 12,
              padding: '3px 10px',
              background: 'none',
              border: '1px solid var(--hairline)',
              borderRadius: 4,
              cursor: 'pointer',
              color: 'var(--fg-muted)',
              fontFamily: 'inherit',
            }}
          >nvm</button>
        </div>
      )}
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16, padding: 16, flex: 1, overflow: 'auto' }}>
      <div className="col" style={{ minWidth: 0, gap: 16 }}>
        {showHero ? <MagicalMomentTile onGo={goConfigureProxy} onSkip={dismissHero}/> : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(7, 1fr)', gap: 8 }}>
            {statsLoading ? Array.from({ length: 7 }).map((_, i) => (
              <div key={i} className="card" style={{ padding: 12 }}><SkeletonBox h={10} w="60%"/><SkeletonBox h={24} w="80%" style={{ margin: '4px 0' }}/><SkeletonBox h={10} w="70%"/></div>
            )) : metrics.map((m, i) => {
              // Resolve display value for agent cards
              let displayVal = m.v;
              if (m.agentLoading) displayVal = <span style={{ color: 'var(--fg-muted)', fontSize: 16 }}>—</span>;
              else if (m.agentError) displayVal = (
                <span title="couldn't load" style={{ color: 'var(--warn)', fontSize: 13, cursor: 'default' }}>
                  <Icon.Warn width={13} height={13} style={{ verticalAlign: 'middle', marginRight: 4 }}/>
                  <span style={{ fontSize: 12 }}>couldn't load</span>
                </span>
              );
              else if (m.v !== null && m.v !== undefined && m.v !== 'err') displayVal = typeof m.v === 'number' ? m.v.toLocaleString() : m.v;

              return (
                <div key={i} className="card hoverable" style={{ padding: 12, cursor: 'pointer' }} onClick={() => setPage(m.path, m.extra)}>
                  <div className="row" style={{ justifyContent: 'space-between' }}>
                    <span style={{ fontSize: 11, textTransform: 'uppercase', color: 'var(--fg-muted)' }}>
                      {m.k}
                      {m.activeTooltip && (
                        <span
                          title={m.activeTooltip}
                          style={{ marginLeft: 4, cursor: 'help', opacity: 0.55, fontSize: 10, fontFamily: 'var(--font-mono)' }}
                        >(?)</span>
                      )}
                    </span>
                    {m.agent && !m.activeTooltip && <span className="chip agent sm">live</span>}
                    {m.warn && <Icon.Warn width={11} height={11} style={{ color: 'var(--warn)' }}/>}
                  </div>
                  <div style={{ fontSize: 20, fontWeight: 600, marginTop: 4 }}>{displayVal}</div>
                  {m.sub !== null && <div style={{ fontSize: 11, color: 'var(--fg-muted)' }}>{m.sub}</div>}
                  {m.activeTooltip && (
                    <div style={{ fontSize: 10, color: 'var(--fg-dim)', marginTop: 1, lineHeight: 1.3 }}>
                      issued ≥1 token · 7d · not deactivated
                    </div>
                  )}
                  {m.trend && !m.agentLoading && !m.agentError && (
                    <Sparkline data={m.trend} height={22} color={m.agent ? 'var(--agent)' : (m.good ? 'var(--success)' : 'var(--fg)')}/>
                  )}
                </div>
              );
            })}
          </div>
        )}

        <div style={{ display: 'grid', gridTemplateColumns: '340px 1fr', gap: 16 }}>
          <div className="card">
            <div className="card-header">Auth method · 30d</div>
            <div className="card-body" style={{ display: 'flex', alignItems: 'center', gap: 24, padding: '20px 24px' }}>
              {authData ? (
                <>
                  <Donut segments={authData} size={100} thickness={16}/>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8, flex: 1 }}>
                    {authData.map((s, i) => (
                      <div key={i} className="row" style={{ gap: 8, fontSize: 12 }}>
                        <div style={{ width: 8, height: 8, borderRadius: '50%', background: s.color }}/>
                        <span style={{ flex: 1, color: 'var(--fg-muted)' }}>{s.label}</span>
                        <span className="mono" style={{ fontWeight: 600 }}>{s.value}</span>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="faint" style={{ flex: 1, textAlign: 'center', padding: 24 }}>No sessions yet.</div>
              )}
            </div>
          </div>

          <div className="card" style={{ minWidth: 0, display: 'flex', flexDirection: 'column' }}>
            <div className="card-header" style={{ flexShrink: 0 }}>
              <span>Recent activity</span>
              <span className={"chip " + (streamStatus === 'live' ? 'success' : streamStatus === 'reconnecting' ? 'warn' : 'faint')}>
                <span className={"dot " + (streamStatus === 'live' ? 'success pulse' : streamStatus === 'reconnecting' ? 'warn pulse' : 'faint')}/> {streamStatus}
              </span>
            </div>
            <div style={{ flex: 1, overflowY: 'auto' }}>
              <table className="tbl" style={{ fontSize: 13, width: '100%', tableLayout: 'fixed' }}>
                <tbody>
                  {activity.length === 0 ? <tr><td colSpan={5} style={{ padding: 40, textAlign: 'center' }}><div className="faint">Waiting for events…</div></td></tr> : activity.map((a, i) => (
                    <tr key={i}>
                      <td style={{ width: 60, color: 'var(--fg-muted)', fontSize: 11 }}>{relTime(a.t)}</td>
                      <td style={{ width: 80 }}><span className="chip sm">{a.actor}</span></td>
                      <td className="mono" style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{a.name}</td>
                      <td className="mono">{a.action}</td>
                      <td className="mono faint" style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{a.meta}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'minmax(0, 2fr) minmax(0, 1fr) minmax(0, 1fr)', gap: 16 }}>
          <div className="card" style={{ display: 'flex', flexDirection: 'column' }}>
            <div className="card-header" style={{ flexShrink: 0 }}>System health</div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', flex: 1 }}>
              {(health || []).map((x, i) => (
                <div key={i} style={{ padding: 12, borderRight: i % 4 !== 3 ? '1px solid var(--hairline)' : 'none', borderTop: i >= 4 ? '1px solid var(--hairline)' : 'none' }}>
                  <div style={{ fontSize: 11, textTransform: 'uppercase', color: 'var(--fg-muted)' }}>{x.k}</div>
                  <div style={{ fontSize: 13, fontWeight: 600, marginTop: 4 }}>{x.v}</div>
                  <div style={{ fontSize: 11, color: 'var(--fg-muted)' }}>{x.sub}</div>
                </div>
              ))}
            </div>
          </div>
          <AttentionPanel healthRaw={healthRaw} stats={stats} onRefresh={refreshHealth} setPage={setPage}/>
          <UpdateProberPanel healthRaw={healthRaw} />
        </div>
      </div>
    </div>
    </div>
  );
}

// useAgentSecurityMetrics — derives 4 agent-security KPIs from existing endpoints.
// No new backend: uses /api/v1/admin/audit-logs (token.exchange events) and
// /api/v1/agents + /api/v1/agents/{id}/tokens (DPoP binding, delegation depth).
function useAgentSecurityMetrics() {
  const [metrics, setMetrics] = React.useState(null);
  const [loading, setLoading] = React.useState(true);

  const compute = React.useCallback(() => {
    const key = localStorage.getItem('shark_admin_key');
    if (!key) { setLoading(false); return; }
    const headers = { Authorization: 'Bearer ' + key };

    const since24h = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();

    Promise.all([
      // token-exchange grants last 24h
      fetch(`/api/v1/admin/audit-logs?action=token.exchange&from=${encodeURIComponent(since24h)}&limit=200`, { headers })
        .then(r => r.ok ? r.json() : { data: [] }),
      // all agents (for delegation + DPoP stats)
      fetch('/api/v1/agents?limit=200', { headers })
        .then(r => r.ok ? r.json() : { data: [] }),
    ]).then(([auditResp, agentsResp]) => {
      const auditEvents = Array.isArray(auditResp?.data) ? auditResp.data : [];
      const agents = Array.isArray(agentsResp?.data) ? agentsResp.data : [];

      // 1. Active token-exchange grants last 24h
      const tokenExchangeGrants = auditEvents.length;

      // 2. Delegation chains: count events with act_chain length > 1; max depth across all
      let chainCount = 0;
      let maxDepth = 0;
      for (const ev of auditEvents) {
        const chain = ev.act_chain;
        if (Array.isArray(chain) && chain.length > 1) {
          chainCount++;
          if (chain.length > maxDepth) maxDepth = chain.length;
        } else if (ev.oauth?.act) {
          // legacy single-hop delegation
          chainCount++;
          if (2 > maxDepth) maxDepth = 2;
        }
      }

      // 3 & 4. DPoP binding % + expired DPoP keys — derive from agent token list
      // Fetch tokens for each agent (cap at 10 agents to stay cheap)
      const agentSlice = agents.slice(0, 10);
      const tokenFetches = agentSlice.map(ag =>
        fetch(`/api/v1/agents/${ag.id}/tokens`, { headers })
          .then(r => r.ok ? r.json() : { data: [] })
          .catch(() => ({ data: [] }))
      );

      Promise.all(tokenFetches).then(tokenResps => {
        let totalTokens = 0;
        let dpopBound = 0;
        let expiredDpop = 0;
        const now = Date.now();

        for (const tr of tokenResps) {
          const toks = Array.isArray(tr?.data) ? tr.data : [];
          for (const tok of toks) {
            totalTokens++;
            const hasDpop = !!(tok.dpop_jkt || tok.cnf?.jkt);
            if (hasDpop) dpopBound++;
            // expired DPoP key: token has a DPoP binding but the token itself is expired
            if (hasDpop && tok.expires_at) {
              const exp = new Date(tok.expires_at).getTime();
              if (exp < now) expiredDpop++;
            }
          }
        }

        const dpopPct = totalTokens > 0 ? Math.round((dpopBound / totalTokens) * 100) : 0;

        setMetrics({ tokenExchangeGrants, chainCount, maxDepth, dpopPct, expiredDpop });
        setLoading(false);
      });
    }).catch(() => setLoading(false));
  }, []);

  React.useEffect(() => {
    compute();
    // refresh every 60s to stay in sync with SSE-driven live data
    const id = setInterval(compute, 60000);
    return () => clearInterval(id);
  }, [compute]);

  return { metrics, loading, refresh: compute };
}

function AgentSecurityCard({ setPage }) {
  const { metrics, loading } = useAgentSecurityMetrics();

  const navigate = () => {
    if (typeof setPage === 'function') {
      setPage('agents');
    } else {
      window.history.pushState(null, '', '/admin/agents');
      window.dispatchEvent(new PopStateEvent('popstate'));
    }
  };

  const rows = loading || !metrics ? null : [
    {
      label: 'Active token-exchange grants',
      value: `${metrics.tokenExchangeGrants}`,
      sub: 'last 24h',
      warn: false,
    },
    {
      label: 'Delegation chains',
      value: `${metrics.chainCount}`,
      sub: `max depth: ${metrics.maxDepth || '—'}`,
      warn: false,
    },
    {
      label: 'DPoP binding',
      value: `${metrics.dpopPct}%`,
      sub: 'of agent tokens',
      warn: false,
    },
    {
      label: 'Expired DPoP keys',
      value: `${metrics.expiredDpop}`,
      sub: metrics.expiredDpop > 0 ? 'rotate recommended' : 'none',
      warn: metrics.expiredDpop > 0,
    },
  ];

  return (
    <div
      onClick={navigate}
      style={{
        borderBottom: '1px solid var(--hairline)',
        padding: '12px 14px',
        cursor: 'pointer',
        transition: 'background 0.1s',
      }}
      onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-2)'}
      onMouseLeave={e => e.currentTarget.style.background = ''}
    >
      <div style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        marginBottom: 10,
      }}>
        <span style={{
          fontSize: 11,
          textTransform: 'uppercase',
          letterSpacing: '0.05em',
          color: 'var(--fg-muted)',
          fontWeight: 600,
        }}>Agent Security</span>
        <span className="chip agent sm">live</span>
      </div>

      {loading ? (
        <div style={{ fontSize: 12, color: 'var(--fg-muted)' }}>Loading…</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 7 }}>
          {rows.map((row, i) => (
            <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', gap: 8 }}>
              <span style={{ fontSize: 12, color: 'var(--fg-muted)', flexShrink: 0 }}>{row.label}</span>
              <span style={{ display: 'flex', alignItems: 'baseline', gap: 4 }}>
                <span style={{
                  fontSize: 13,
                  fontWeight: 600,
                  fontFamily: 'var(--font-mono)',
                  color: row.warn ? 'var(--warn, #d97706)' : 'var(--fg)',
                }}>{row.value}</span>
                <span style={{ fontSize: 11, color: 'var(--fg-muted)' }}>{row.sub}</span>
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function AttentionPanel({ healthRaw, stats, onRefresh, setPage }) {
  const alerts = [];
  if (stats.failedLogins24h.count > 10) {
    alerts.push({ text: `High failed logins: ${stats.failedLogins24h.count} (24h)`, type: 'warn', path: 'audit', extra: { status: 'failure', action: 'user.login' } });
  }
  if (stats.apiKeys.expiring > 0) {
    alerts.push({ text: `${stats.apiKeys.expiring} API keys expiring soon`, type: 'info', path: 'keys' });
  }
  if (healthRaw?.db?.status !== 'healthy' && healthRaw?.db?.status) {
    alerts.push({ text: `Database status: ${healthRaw.db.status}`, type: 'danger' });
  }

  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="card-header" style={{ flexShrink: 0 }}>
        Attention 
        <button className="btn ghost sm" onClick={onRefresh}><Icon.Refresh width={11}/></button>
      </div>
      <div style={{ flex: 1, overflowY: 'auto' }}>
        <div className="col" style={{ gap: 1 }}>
          {alerts.length === 0 ? (
            <div style={{ padding: 12, fontSize: 13, color: 'var(--fg-muted)' }}>All systems healthy.</div>
          ) : alerts.map((a, i) => (
            <div key={i} 
                 onClick={() => a.path && setPage(a.path, a.extra)}
                 style={{ 
                   padding: '10px 14px', 
                   fontSize: 13, 
                   borderBottom: '1px solid var(--hairline)',
                   background: a.type === 'danger' ? 'var(--danger-bg)' : a.type === 'warn' ? 'var(--warn-bg)' : 'transparent',
                   color: a.type === 'danger' ? 'var(--danger)' : a.type === 'warn' ? 'var(--warn)' : 'var(--fg)',
                   cursor: a.path ? 'pointer' : 'default',
                 }}>
              {a.text}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function UpdateProberPanel({ healthRaw }) {
  const [status, setStatus] = React.useState('checking');
  const [latestVersion, setLatestVersion] = React.useState(null);
  const activeVersion = healthRaw?.version || 'unknown';

  React.useEffect(() => {
    if (!healthRaw) return;
    
    // We only check if we actually have an active version to compare against, 
    // and it's not a generic dev string, though we can still check registry anyway.
    let cancelled = false;
    
    fetch('https://api.github.com/repos/sharkauth/shark/releases/latest')
      .then(res => {
        if (!res.ok) throw new Error('Registry unavailable');
        return res.json();
      })
      .then(data => {
        if (cancelled) return;
        const latest = data.tag_name ? data.tag_name.replace(/^v/, '') : null;
        if (!latest) throw new Error('No latest release');
        
        setLatestVersion(latest);
        
        const active = activeVersion.replace(/^v/, '');
        
        if (active === 'unknown' || active === '0.0.0-dev') {
          // Local/dev builds
          setStatus('unknown');
          return;
        }

        // Basic semver compare (Major.Minor.Patch)
        const [aMaj, aMin, aPat] = active.split('.').map(n => parseInt(n, 10) || 0);
        const [lMaj, lMin, lPat] = latest.split('.').map(n => parseInt(n, 10) || 0);

        if (lMaj > aMaj) {
          setStatus('outdated'); // Major update
        } else if (lMaj === aMaj && lMin > aMin) {
          setStatus('update-available'); // Minor update
        } else if (lMaj === aMaj && lMin === aMin && lPat > aPat) {
          setStatus('update-available'); // Patch update
        } else {
          setStatus('up-to-date');
        }
      })
      .catch(err => {
        if (!cancelled) setStatus('error');
      });

    return () => { cancelled = true; };
  }, [healthRaw, activeVersion]);

  // Derive visual properties based on status
  let dotColor = 'var(--fg-muted)';
  let chipClass = 'faint';
  let chipText = 'Checking';
  let statusText = 'Querying release registry...';

  if (status === 'up-to-date') {
    dotColor = 'var(--success)';
    chipClass = 'success';
    chipText = 'Up to date';
    statusText = `Binary is current (v${activeVersion})`;
  } else if (status === 'update-available') {
    dotColor = 'var(--warn)';
    chipClass = 'warn';
    chipText = 'Update';
    statusText = `v${latestVersion} is available`;
  } else if (status === 'outdated') {
    dotColor = 'var(--danger)';
    chipClass = 'danger';
    chipText = 'Outdated';
    statusText = `Critical update v${latestVersion} available`;
  } else if (status === 'error') {
    dotColor = 'var(--warn)';
    chipClass = 'warn';
    chipText = 'Offline';
    statusText = 'Could not reach registry';
  } else if (status === 'unknown') {
    dotColor = 'var(--fg-dim)';
    chipClass = 'faint';
    chipText = 'Dev build';
    statusText = `Unreleased binary (v${activeVersion})`;
  }

  return (
    <div className="card" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="card-header" style={{ flexShrink: 0 }}>
        <span>Update prober</span>
        <span className={`chip ${chipClass}`}>
           {chipText}
        </span>
      </div>
      <div style={{ flex: 1, padding: '12px 16px', display: 'flex', flexDirection: 'column', justifyContent: 'center', gap: 10 }}>
        <div className="col" style={{ gap: 2 }}>
          <div style={{ fontSize: 10, textTransform: 'uppercase', color: 'var(--fg-dim)', letterSpacing: '0.05em' }}>Current binary</div>
          <div className="row" style={{ alignItems: 'baseline', gap: 6 }}>
            <span className="mono" style={{ fontSize: 13, fontWeight: 600 }}>
              {activeVersion !== 'unknown' ? `v${activeVersion}` : '—'}
            </span>
            {healthRaw?.commit && healthRaw.commit !== 'none' && (
              <span className="mono faint" style={{ fontSize: 10 }}>({healthRaw.commit})</span>
            )}
          </div>
        </div>

        <div style={{ fontSize: 11, color: 'var(--fg-dim)', display: 'flex', alignItems: 'center', gap: 6 }}>
          <div style={{ flexShrink: 0, width: 6, height: 6, borderRadius: '50%', background: dotColor }} />
          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {statusText}
          </span>
        </div>

        {healthRaw?.build_date && healthRaw.build_date !== 'unknown' && (
          <div style={{ fontSize: 10, color: 'var(--fg-faint)', borderTop: '1px solid var(--hairline)', paddingTop: 8, marginTop: 2 }}>
            Build date: {new Date(healthRaw.build_date).toLocaleDateString()}
          </div>
        )}
      </div>
    </div>
  );
}
