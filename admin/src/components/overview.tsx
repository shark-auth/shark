// @ts-nocheck
import React from 'react'
import { Icon, Sparkline, Donut } from './shared'
import { useAPI } from './api'

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
  const { data: agentsRaw } = useAPI('/agents?limit=1');
  const { events: liveActivity, status: streamStatus } = useRealtimeActivity();

  function mapStats(s, agentsTotal) {
    if (!s) return null;
    return {
      users: { total: s.users?.total ?? 0, delta7d: s.users?.created_last_7d ?? 0 },
      sessions: { active: s.sessions?.active ?? 0 },
      mfa: { pct: s.mfa?.total > 0 ? (s.mfa.enabled / s.mfa.total) : 0, enabled: s.mfa?.enabled ?? 0, total: s.mfa?.total ?? 0 },
      failedLogins24h: { count: s.failed_logins_24h ?? 0, deltaPct: 0 },
      apiKeys: { count: s.api_keys?.active ?? 0, expiring: s.api_keys?.expiring_7d ?? 0 },
      agents: { count: agentsTotal ?? 0 },
    };
  }

  function mapTrends(t) {
    const signupCounts = Array.isArray(t?.signups_by_day) ? t.signups_by_day.map(p => p.count) : null;
    return { users: signupCounts, sessions: null, mfa: null, failed: null, keys: null, agents: null };
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

  const agentsTotal = agentsRaw?.total ?? null;
  const stats = mapStats(statsRaw, agentsTotal) || { users: { total: 0, delta7d: 0 }, sessions: { active: 0 }, mfa: { pct: 0, enabled: 0, total: 0 }, failedLogins24h: { count: 0, deltaPct: 0 }, apiKeys: { count: 0, expiring: 0 }, agents: { count: 0 } };
  const trends = mapTrends(trendsRaw);
  const authData = mapAuthBreakdown(trendsRaw);
  const health = mapHealth(healthRaw);
  const activity = mapActivity(liveActivity);

  const metrics = [
    { k: 'Users', v: stats.users.total.toLocaleString(), sub: `+${stats.users.delta7d} last 7d`, trend: trends.users },
    { k: 'Active sessions', v: stats.sessions.active.toLocaleString(), sub: '—', trend: trends.sessions },
    { k: 'MFA adoption', v: Math.round(stats.mfa.pct * 100) + '%', sub: `${stats.mfa.enabled.toLocaleString()} enabled`, trend: trends.mfa },
    { k: 'Failed logins 24h', v: stats.failedLogins24h.count, sub: '—', trend: trends.failed, good: true },
    { k: 'API keys active', v: stats.apiKeys.count, sub: `${stats.apiKeys.expiring} expiring`, trend: trends.keys, warn: true },
    { k: 'Agents active', v: stats.agents.count, sub: '—', trend: trends.agents, agent: true },
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
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 340px', gap: 16, padding: 16, flex: 1, overflow: 'auto' }}>
      <div className="col" style={{ minWidth: 0 }}>
        {showHero ? <MagicalMomentTile onGo={goConfigureProxy} onSkip={dismissHero}/> : (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 8 }}>
            {statsLoading ? Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="card" style={{ padding: 12 }}><SkeletonBox h={10} w="60%"/><SkeletonBox h={24} w="80%" style={{ margin: '4px 0' }}/><SkeletonBox h={10} w="70%"/></div>
            )) : metrics.map((m, i) => (
              <div key={i} className="card" style={{ padding: 12 }}>
                <div className="row" style={{ justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 11, textTransform: 'uppercase', color: 'var(--fg-muted)' }}>{m.k}</span>
                  {m.agent && <span className="chip agent sm">new</span>}
                  {m.warn && <Icon.Warn width={11} height={11} style={{ color: 'var(--warn)' }}/>}
                </div>
                <div style={{ fontSize: 20, fontWeight: 600, marginTop: 4 }}>{m.v}</div>
                <div style={{ fontSize: 11, color: 'var(--fg-muted)' }}>{m.sub}</div>
                {m.trend && <Sparkline data={m.trend} height={22} color={m.agent ? 'var(--agent)' : (m.good ? 'var(--success)' : 'var(--fg)')}/>}
              </div>
            ))}
          </div>
        )}

        <div style={{ display: 'grid', gridTemplateColumns: '340px 1fr', gap: 16, marginTop: 20 }}>
          <div className="card">
            <div className="card-header">Auth method · 30d</div>
            <div className="card-body">
              {authData ? <Donut segments={authData} size={100} thickness={16}/> : <div className="faint" style={{ padding: 24, textAlign: 'center' }}>No sessions yet.</div>}
            </div>
          </div>

          <div className="card" style={{ minWidth: 0 }}>
            <div className="card-header">
              <span>Recent activity</span>
              <span className={"chip " + (streamStatus === 'live' ? 'success' : streamStatus === 'reconnecting' ? 'warn' : 'faint')}>
                <span className={"dot " + (streamStatus === 'live' ? 'success pulse' : streamStatus === 'reconnecting' ? 'warn pulse' : 'faint')}/> {streamStatus}
              </span>
            </div>
            <div style={{ overflow: 'hidden' }}>
              <table className="tbl" style={{ fontSize: 13 }}>
                <tbody>
                  {activity.length === 0 ? <tr><td colSpan={5} style={{ padding: 40, textAlign: 'center' }}><div className="faint">Waiting for events…</div></td></tr> : activity.map((a, i) => (
                    <tr key={i}>
                      <td style={{ width: 60, color: 'var(--fg-muted)', fontSize: 11 }}>{relTime(a.t)}</td>
                      <td style={{ width: 80 }}><span className="chip sm">{a.actor}</span></td>
                      <td className="mono" style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis' }}>{a.name}</td>
                      <td className="mono">{a.action}</td>
                      <td className="mono faint" style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>{a.meta}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div className="card" style={{ marginTop: 16 }}>
          <div className="card-header">System health</div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)' }}>
            {(health || []).map((x, i) => (
              <div key={i} style={{ padding: 12, borderRight: i % 4 !== 3 ? '1px solid var(--hairline)' : 'none', borderTop: i >= 4 ? '1px solid var(--hairline)' : 'none' }}>
                <div style={{ fontSize: 11, textTransform: 'uppercase', color: 'var(--fg-muted)' }}>{x.k}</div>
                <div style={{ fontSize: 13, fontWeight: 600, marginTop: 4 }}>{x.v}</div>
                <div style={{ fontSize: 11, color: 'var(--fg-muted)' }}>{x.sub}</div>
              </div>
            ))}
          </div>
        </div>
      </div>
      <AttentionPanel healthRaw={healthRaw} stats={stats} onRefresh={refreshHealth}/>
    </div>
    </div>
  );
}

function AttentionPanel({ healthRaw, stats, onRefresh }) {
  return (
    <div className="card" style={{ alignSelf: 'start', position: 'sticky', top: 0 }}>
      <div className="card-header">Attention <button className="btn ghost sm" onClick={onRefresh}><Icon.Refresh width={11}/></button></div>
      <div style={{ padding: 12, fontSize: 13, color: 'var(--fg-muted)' }}>All systems healthy.</div>
    </div>
  );
}
