// @ts-nocheck
import React from 'react'
import { Icon, Avatar, CopyField, Kbd, Sparkline } from './shared'
import { API, useAPI } from './api'
import { MOCK } from './mock'
import { CLIFooter } from './CLIFooter'
import { useURLParam } from './useURLParams'
import { useToast } from './toast'

// Sessions — live header strip + geo view + table + detail slideover

// Local relative time helper — mirrors MOCK.relativeTime but standalone
function relTime(t) {
  if (!t) return '—';
  const NOW = Date.now();
  const diff = Math.floor((NOW - t) / 1000);
  if (diff < 0) {
    const n = Math.abs(diff);
    if (n < 60) return `in ${n}s`;
    if (n < 3600) return `in ${Math.floor(n/60)}m`;
    if (n < 86400) return `in ${Math.floor(n/3600)}h`;
    return `in ${Math.floor(n/86400)}d`;
  }
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff/60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff/3600)}h ago`;
  return `${Math.floor(diff/86400)}d ago`;
}

// Normalize a raw API session object to the shape the UI expects.
// Handles both real API field names and mock field names gracefully.
function normalizeSession(s) {
  if (!s) return s;
  // Already normalized (has .user and .ip as expected by UI)
  const email = s.user_email || s.email || s.user || '';
  const ip = s.ip_address || s.ip || '';
  const authMethod = s.auth_method || s.method || '';
  const mfaRaw = s.mfa_verified != null ? s.mfa_verified : (s.mfa_passed != null ? s.mfa_passed : s.mfa);
  // mfa: keep as string type name if present, coerce bool true → 'verified', false → null
  // Fix: explicitly check for true to avoid 'true' being returned for users without MFA
  const mfa = (mfaRaw === true || mfaRaw === 'true') ? 'verified' : null;
  // timestamps: API returns ISO strings, mock returns ms epoch numbers
  const toMs = (v) => {
    if (!v) return 0;
    if (typeof v === 'number') return v;
    return new Date(v).getTime();
  };
  return {
    ...s,
    user: email,
    name: s.name || s.user_name || email.split('@')[0] || '—',
    ip,
    method: authMethod,
    mfa,
    dev: s.user_agent || s.dev || '—',
    created: toMs(s.created_at || s.created),
    last: toMs(s.last_activity_at || s.last_seen_at || s.last || s.updated_at),
    expires: toMs(s.expires_at || s.expires),
    city: s.city || '',
    country: s.country || '',
    region: s.region || '',
    lat: s.lat || 0,
    lng: s.lng || 0,
    client: s.client || s.client_type || 'web',
    risk: s.risk || s.risk_level || 'low',
    blocked: s.blocked || s.is_blocked || false,
    suspicious: s.suspicious || s.suspicious_reason || null,
    current: s.current || s.is_current || false,
    // JTI from backend (only present for JWT-mode sessions)
    jti: s.jti || null,
  };
}

export function Sessions() {
  const [selected, setSelected] = React.useState(null);
  const [query, setQuery] = React.useState('');
  const [clientFilter, setClientFilter] = useURLParam('client', 'all');
  const [liveTail, setLiveTail] = React.useState(true);
  const [pulse, setPulse] = React.useState(0);
  const [jtiInput, setJtiInput] = React.useState('');
  const toast = useToast();

  const { data: sessionsRaw, loading, refresh } = useAPI('/admin/sessions');
  const sessions = (sessionsRaw?.data || []).map(normalizeSession);

  // Live polling: refresh every 5s when liveTail is on
  React.useEffect(() => {
    if (!liveTail) return;
    const id = setInterval(refresh, 5000);
    return () => clearInterval(id);
  }, [liveTail, refresh]);

  // tick to drive the "live" animation
  React.useEffect(() => {
    if (!liveTail) return;
    const iv = setInterval(() => setPulse(p => p + 1), 2200);
    return () => clearInterval(iv);
  }, [liveTail]);

  const handleRevoke = async (sessionId) => {
    try {
      await API.del('/admin/sessions/' + sessionId);
      toast.success('Session revoked');
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Failed to revoke session');
    }
  };

  const handleRevokeJTI = async () => {
    const jti = jtiInput.trim();
    if (!jti) return;
    try {
      await API.post('/admin/auth/revoke-jti', { jti });
      setJtiInput('');
      toast.success(`JTI ${jti.slice(0, 8)}… revoked`);
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Failed to revoke JTI');
    }
  };

  const handleRevokeAll = async () => {
    if (!window.confirm('Revoke ALL active sessions? This will log out every user immediately.')) return;
    try {
      const r = await API.del('/admin/sessions');
      toast.success(`Revoked ${r?.revoked ?? 0} sessions`);
      refresh();
    } catch (e) {
      toast.error(e?.message || 'Failed to revoke all sessions');
    }
  };

  const all = sessions;
  const filtered = all.filter(s => {
    const userStr = (s.user || '').toLowerCase();
    const ipStr = s.ip || '';
    const cityStr = (s.city || '').toLowerCase();
    if (query && !(userStr.includes(query.toLowerCase()) || ipStr.includes(query) || cityStr.includes(query.toLowerCase()))) return false;
    if (clientFilter !== 'all' && s.client !== clientFilter) return false;
    return true;
  });

  const totalActive = all.length;
  // Aggregate stats from real sessions
  const byAuthMethod = {};
  all.forEach(s => { byAuthMethod[s.method] = (byAuthMethod[s.method] || 0) + 1; });
  const mfaCount = all.filter(s => s.mfa).length;
  const mfaRate = totalActive > 0 ? Math.round((mfaCount / totalActive) * 100) : 0;
  const suspicious = all.filter(s => s.suspicious || s.blocked).length;
  const clientCounts = { web: 0, mobile: 0, api: 0, agent: 0 };
  all.forEach(s => { if (clientCounts[s.client] != null) clientCounts[s.client]++; });
  const regionCounts = {};
  all.forEach(s => { if (s.region) regionCounts[s.region] = (regionCounts[s.region] || 0) + 1; });

  if (loading && all.length === 0) {
    return <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: 'var(--fg-muted)', fontSize: 13 }}>Loading sessions…</div>;
  }

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {/* LIVE strip */}
        <LiveStrip
          totalActive={totalActive}
          suspicious={suspicious}
          clientCounts={clientCounts}
          regionCounts={regionCounts}
          mfaRate={mfaRate}
          byAuthMethod={byAuthMethod}
          pulse={pulse}
          live={liveTail}
          setLive={setLiveTail}
        />

        {/* Toolbar */}
        <div style={{
          padding: '8px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 8, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          <div style={{
            display: 'flex', alignItems: 'center', gap: 6,
            border: '1px solid var(--hairline-strong)',
            background: 'var(--surface-1)',
            borderRadius: 5, padding: '0 8px',
            height: 28, width: 300,
          }}>
            <Icon.Search width={12} height={12} style={{ opacity: 0.5 }}/>
            <input
              placeholder="Search email, IP, city…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              style={{ flex: 1, fontSize: 11, lineHeight: 1.5 }}
            />
            <Kbd keys="/"/>
          </div>

          <Segment value={clientFilter} setValue={setClientFilter}
            options={[
              { v: 'all', l: 'All' },
              { v: 'web', l: 'Web' },
              { v: 'mobile', l: 'Mobile' },
              { v: 'api', l: 'API' },
              { v: 'agent', l: 'Agent' },
            ]}/>

          {/* Revoke by JTI */}
          <div style={{
            display: 'flex', alignItems: 'center', gap: 4,
            border: '1px solid var(--hairline-strong)',
            background: 'var(--surface-1)', borderRadius: 5,
            padding: '0 6px', height: 28,
          }}>
            <Icon.Token width={11} height={11} style={{ opacity: 0.4, flexShrink: 0 }}/>
            <input
              placeholder="Revoke JTI…"
              value={jtiInput}
              onChange={e => setJtiInput(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter' && jtiInput.trim()) handleRevokeJTI(); }}
              style={{ width: 110, fontSize: 11, color: 'var(--fg)', fontFamily: 'var(--font-mono)' }}
            />
            <button className="btn ghost icon sm" disabled={!jtiInput.trim()} onClick={handleRevokeJTI} title="Revoke">
              <Icon.X width={10} height={10}/>
            </button>
          </div>

          <div style={{ flex: 1 }}/>

          <span className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
            {filtered.length.toLocaleString()} of {all.length.toLocaleString()}
          </span>

          <button className="btn sm" onClick={() => {
            const rows = filtered.map(s => [s.id, s.user, s.ip, s.city, s.client, s.current ? 'current' : '', s.blocked ? 'blocked' : ''].join(','));
            const csv = ['id,user,ip,city,client,current,blocked', ...rows].join('\n');
            const a = document.createElement('a');
            a.href = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }));
            a.download = 'sessions.csv';
            a.click();
          }}>Export</button>
          <button className="btn sm danger" onClick={handleRevokeAll}>Revoke all</button>
        </div>

        {/* Content */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <SessionsTable sessions={filtered} selected={selected} setSelected={setSelected} pulse={pulse} onRevoke={handleRevoke}/>
        </div>
      </div>

      {selected && <SessionSlideover session={selected} onClose={() => setSelected(null)} onRevoke={handleRevoke}/>}
    </div>
  );
}

/* ---------------- LIVE STRIP ---------------- */

function LiveStrip({ totalActive, suspicious, clientCounts, regionCounts, mfaRate, byAuthMethod, pulse, live, setLive }) {
  return (
    <div style={{
      borderBottom: '1px solid var(--hairline)',
      background: 'var(--surface-1)',
      padding: '12px 16px',
      display: 'grid',
      gridTemplateColumns: '240px 1fr 1fr 1fr',
      gap: 0,
    }}>
      {/* Hero: ACTIVE counter */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6, borderRight: '1px solid var(--hairline)', paddingRight: 16 }}>
        <div className="row" style={{ gap: 6 }}>
          <span
            onClick={() => setLive(!live)}
            title={live ? 'Live · click to pause' : 'Paused · click to resume'}
            style={{ cursor: 'pointer' }}>
            <span className={'dot ' + (live ? 'success pulse' : '')} style={{ display: 'inline-block' }}/>
          </span>
          <span style={{ fontSize: 11, lineHeight: 1.5, textTransform: 'uppercase', letterSpacing: '0.1em', color: live ? 'var(--success)' : 'var(--fg-muted)', fontWeight: 500 }}>
            {live ? 'Live' : 'Paused'}
          </span>
        </div>
        {/* Hero counter — display font, 20px */}
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
          <span style={{ fontFamily: 'var(--font-display)', fontSize: 20, fontWeight: 600, letterSpacing: '-0.02em', fontVariantNumeric: 'tabular-nums', lineHeight: 1 }}>
            {totalActive.toLocaleString()}
          </span>
          <span style={{ fontSize: 11, lineHeight: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)' }}>active sessions</span>
        </div>
        <div className="row" style={{ gap: 10 }}>
          <span className="row" style={{ gap: 4 }}>
            <span className="dot success"/>
            <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>healthy</span>
            <span className="mono" style={{ fontSize: 11, lineHeight: 1.5 }}>{totalActive - suspicious}</span>
          </span>
          <span className="row" style={{ gap: 4 }}>
            <span className={"dot " + (suspicious > 0 ? 'danger pulse' : 'muted')}/>
            <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>suspicious</span>
            <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: suspicious > 0 ? 'var(--danger)' : 'var(--fg)' }}>{suspicious}</span>
          </span>
        </div>
      </div>

      {/* Clients */}
      <div style={{ padding: '0 16px', borderRight: '1px solid var(--hairline)' }}>
        <div style={{ fontSize: 11, lineHeight: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 8 }}>
          By client
        </div>
        <StackedBar
          entries={[
            { k: 'web',    v: clientCounts.web,    color: 'var(--fg)' },
            { k: 'mobile', v: clientCounts.mobile, color: 'var(--fg-muted)' },
            { k: 'api',    v: clientCounts.api,    color: '#4a4a4a' },
            { k: 'agent',  v: clientCounts.agent,  color: 'var(--agent)' },
          ]}
        />
      </div>

      {/* Auth methods */}
      <div style={{ padding: '0 16px', borderRight: '1px solid var(--hairline)' }}>
        <div style={{ fontSize: 11, lineHeight: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 8 }}>
          By auth method
        </div>
        <div className="col" style={{ gap: 3 }}>
          {Object.entries(byAuthMethod || {})
            .sort((a, b) => b[1] - a[1])
            .slice(0, 4)
            .map(([method, count]) => {
              const pct = totalActive > 0 ? count / totalActive : 0;
              return (
                <div key={method} className="row" style={{ gap: 8 }}>
                  <span className="mono" style={{ width: 72, fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{method || '—'}</span>
                  <div style={{ flex: 1, height: 4, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
                    <div style={{ width: `${pct*100}%`, height: '100%', background: 'var(--fg)' }}/>
                  </div>
                  <span className="mono" style={{ width: 18, fontSize: 11, lineHeight: 1.5, textAlign: 'right' }}>{count}</span>
                </div>
              );
            })}
          {totalActive > 0 && (
            <div style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)', marginTop: 4 }}>MFA coverage: {mfaRate}%</div>
          )}
        </div>
      </div>

      {/* Regions */}
      <div style={{ paddingLeft: 16 }}>
        <div style={{ fontSize: 11, lineHeight: 1.5, textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-muted)', marginBottom: 8 }}>
          By region
        </div>
        <div className="col" style={{ gap: 3 }}>
          {Object.entries(regionCounts)
            .sort((a, b) => b[1] - a[1])
            .slice(0, 4)
            .map(([r, n]) => {
              const pct = n / Math.max(...Object.values(regionCounts));
              return (
                <div key={r} className="row" style={{ gap: 8 }}>
                  <span className="mono" style={{ width: 78, fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{r}</span>
                  <div style={{ flex: 1, height: 4, background: 'var(--surface-3)', borderRadius: 2, overflow: 'hidden' }}>
                    <div style={{ width: `${pct*100}%`, height: '100%', background: r === 'tor' ? 'var(--danger)' : 'var(--fg)' }}/>
                  </div>
                  <span className="mono" style={{ width: 18, fontSize: 11, lineHeight: 1.5, textAlign: 'right' }}>{n}</span>
                </div>
              );
            })}
        </div>
      </div>
    </div>
  );
}

function StackedBar({ entries }) {
  const total = entries.reduce((a, e) => a + e.v, 0);
  return (
    <>
      <div style={{ display: 'flex', height: 8, borderRadius: 2, overflow: 'hidden', marginBottom: 8 }}>
        {entries.map((e, i) => (
          <div key={i} title={`${e.k}: ${e.v}`} style={{ flex: e.v, background: e.color, opacity: e.v === 0 ? 0.2 : 1 }}/>
        ))}
      </div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '2px 10px' }}>
        {entries.map((e, i) => (
          <div key={i} className="row" style={{ gap: 6 }}>
            <span style={{ width: 8, height: 8, background: e.color, borderRadius: 1, opacity: e.v === 0 ? 0.2 : 1, flexShrink: 0 }}/>
            <span style={{ color: 'var(--fg-muted)', fontSize: 11, lineHeight: 1.5 }}>{e.k}</span>
            <span className="mono" style={{ marginLeft: 'auto', fontSize: 11, lineHeight: 1.5, fontVariantNumeric: 'tabular-nums' }}>{e.v}</span>
            <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{total > 0 ? Math.round((e.v/total)*100) : 0}%</span>
          </div>
        ))}
      </div>
    </>
  );
}

function Segment({ value, setValue, options }) {
  return (
    <div className="seg" style={{ display: 'inline-flex', border: '1px solid var(--hairline-strong)', borderRadius: 5, overflow: 'hidden', height: 28 }}>
      {options.map((o, i) => (
        <button key={o.v} onClick={() => setValue(o.v)}
          style={{
            padding: '0 10px', height: 28, fontSize: 11, lineHeight: '28px',
            background: value === o.v ? 'var(--surface-3)' : 'var(--surface-1)',
            color: value === o.v ? 'var(--fg)' : 'var(--fg-muted)',
            borderRight: i < options.length - 1 ? '1px solid var(--hairline)' : 'none',
          }}>
          {o.l}
        </button>
      ))}
    </div>
  );
}

/* ---------------- TABLE VIEW ---------------- */

export function SessionsTable({ sessions, selected, setSelected, pulse, onRevoke }) {
  return (
    <table className="tbl">
      <thead>
        <tr>
          <th style={{ width: 24, paddingLeft: 16, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}></th>
          <th style={{ fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>User</th>
          <th style={{ width: 220, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>Device</th>
          <th style={{ width: 80, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>Client</th>
          <th style={{ width: 150, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>IP · Location</th>
          <th style={{ width: 80, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>MFA</th>
          <th style={{ width: 90, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>Started</th>
          <th style={{ width: 110, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>Last seen</th>
          <th style={{ width: 80, fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.07em', color: 'var(--fg-muted)', position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}>JTI</th>
          <th style={{ width: 60, position: 'sticky', top: 0, background: 'var(--surface-0)', zIndex: 1 }}></th>
        </tr>
      </thead>
      <tbody>
        {sessions.map(s => {
          const recent = Date.now() - s.last < 60 * 1000; // < 1m
          const veryRecent = Date.now() - s.last < 15 * 1000;
          // Status dot color: danger=blocked, warn=suspicious, success=active
          const dotClass = s.blocked ? 'danger' : s.suspicious ? 'warn' : recent ? 'success pulse' : '';
          return (
            <tr key={s.id}
                className={selected?.id === s.id ? 'active' : ''}
                onClick={() => setSelected(s)}
                style={{ cursor: 'pointer' }}>
              <td style={{ paddingLeft: 16 }}>
                <span
                  className={"dot " + dotClass}
                  style={{ display: 'inline-block' }}/>
              </td>
              <td>
                <div className="row" style={{ gap: 8 }}>
                  <Avatar name={s.name} email={s.user} agent={s.client === 'agent' || s.client === 'api'}/>
                  <div style={{ minWidth: 0 }}>
                    <div style={{ fontWeight: 500, fontSize: 13 }}>{s.name}</div>
                    <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{s.user}</div>
                  </div>
                  {s.current && <span className="chip success" style={{ height: 18 }}>you</span>}
                </div>
              </td>
              <td>
                <div style={{ fontSize: 13 }}>{s.dev.split('·')[0].trim()}</div>
                <div className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>{s.dev.split('·').slice(1).join('·').trim() || '—'}</div>
              </td>
              <td>
                <span className={'chip' + (s.client === 'agent' ? ' agent' : '')} style={{ height: 18 }}>{s.client}</span>
              </td>
              <td>
                <div className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: s.region === 'tor' ? 'var(--danger)' : 'var(--fg)' }}>{s.ip}</div>
                <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{s.city}{s.country && s.country !== '—' && ` · ${s.country}`}</div>
              </td>
              <td>
                {s.mfa ? (
                  <span className="row" style={{ gap: 4, color: 'var(--success)' }}>
                    <Icon.Shield width={11} height={11}/>
                    <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{s.mfa}</span>
                  </span>
                ) : <span className="faint mono" style={{ fontSize: 11, lineHeight: 1.5 }}>none</span>}
              </td>
              <td className="mono faint" style={{ fontSize: 11, lineHeight: 1.5 }}>{relTime(s.created)}</td>
              <td className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: veryRecent ? 'var(--success)' : recent ? 'var(--fg)' : 'var(--fg-muted)' }}>
                {veryRecent ? <span className="row" style={{gap:4}}><span className="dot success pulse"/>now</span> : relTime(s.last)}
              </td>
              <td className="mono faint" style={{ fontSize: 10, lineHeight: 1.5, maxWidth: 70, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {s.jti ? s.jti.slice(0, 8) : '—'}
              </td>
              {/* Inline revoke — ghost danger sm, stops row click */}
              <td onClick={e => e.stopPropagation()} style={{ paddingRight: 8 }}>
                <button
                  className="btn ghost sm"
                  onClick={() => onRevoke && onRevoke(s.id)}
                  style={{ fontSize: 11, color: 'var(--danger)', opacity: 0.7 }}
                  title="Revoke session"
                >
                  Revoke
                </button>
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

/* ---------------- DETAIL SLIDEOVER ---------------- */

function SessionSlideover({ session, onClose, onRevoke }) {
  const [tab, setTab] = React.useState('overview');
  const [revoking, setRevoking] = React.useState(false);
  const [moreOpen, setMoreOpen] = React.useState(false);

  const handleRevoke = async () => {
    setRevoking(true);
    try {
      await onRevoke(session.id);
      onClose();
    } finally {
      setRevoking(false);
    }
  };
  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'claims', label: 'Token claims' },
    { id: 'events', label: 'Events' },
  ];

  return (
    <div style={{
      width: 540, borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex', flexDirection: 'column',
      flexShrink: 0,
      animation: 'slideIn 140ms ease-out',
    }}>
      {/* Header */}
      <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
        <div className="row" style={{ justifyContent: 'space-between', marginBottom: 12 }}>
          <button className="btn ghost sm" onClick={onClose}><Icon.X width={12} height={12}/>Close</button>
          <div className="row" style={{ gap: 4 }}>
            {/* Revoke: ghost danger sm — present but not screaming */}
            <button
              className="btn ghost sm"
              onClick={handleRevoke}
              disabled={revoking}
              style={{ color: 'var(--danger)' }}
            >
              {revoking ? 'Revoking…' : 'Revoke'}
            </button>
            <div style={{ position: 'relative' }}>
              <button
                className="btn ghost icon sm"
                aria-label="More options"
                onClick={() => setMoreOpen(o => !o)}
              ><Icon.More width={12} height={12}/></button>
              {moreOpen && (
                <div style={{
                  position: 'absolute', top: '100%', right: 0, marginTop: 4,
                  background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)',
                  zIndex: 50, minWidth: 140,
                }} onClick={() => setMoreOpen(false)}>
                  <button className="btn ghost sm" style={{ width: '100%', textAlign: 'left', justifyContent: 'flex-start', borderRadius: 0, color: 'var(--danger)' }}
                    onClick={handleRevoke} disabled={revoking}>
                    Revoke session
                  </button>
                  <button className="btn ghost sm" style={{ width: '100%', textAlign: 'left', justifyContent: 'flex-start', borderRadius: 0 }}
                    onClick={() => navigator.clipboard?.writeText(session.id)}>
                    Copy session ID
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
        {/* User identity */}
        <div className="row" style={{ gap: 12 }}>
          <Avatar name={session.name} email={session.user} size={38} agent={session.client === 'agent' || session.client === 'api'}/>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ fontFamily: 'var(--font-display)', fontSize: 16, fontWeight: 500, letterSpacing: '-0.01em' }}>{session.name}</div>
            <div className="row" style={{ gap: 6, marginTop: 2 }}>
              <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{session.user}</span>
              <CopyField value={session.id}/>
            </div>
          </div>
        </div>
        {/* Status chips */}
        <div className="row" style={{ gap: 6, marginTop: 12, flexWrap: 'wrap' }}>
          <span className={'chip' + (session.client === 'agent' ? ' agent' : '')}>{session.client}</span>
          {session.blocked && <span className="chip danger"><Icon.X width={9} height={9}/>blocked</span>}
          {session.suspicious && <span className="chip warn"><Icon.Warn width={9} height={9}/>suspicious</span>}
          {session.mfa && <span className="chip"><Icon.Shield width={10} height={10}/>mfa · {session.mfa}</span>}
          <span className="chip">{session.method}</span>
          {session.current && <span className="chip success">you</span>}
        </div>
        {/* Suspicious banner */}
        {session.suspicious && (
          <div style={{
            marginTop: 12, padding: '8px 10px',
            background: 'color-mix(in oklch, var(--danger) 10%, var(--surface-1))',
            border: '1px solid color-mix(in oklch, var(--danger) 40%, var(--hairline))',
            borderRadius: 4,
            fontSize: 11, lineHeight: 1.5, color: 'var(--danger)',
          }}>
            <Icon.Warn width={11} height={11} style={{ verticalAlign: 'middle', marginRight: 6 }}/>
            {session.suspicious === 'impossible-travel' ? 'Impossible travel — last seen in SF 8m ago; now Berlin.' : 'Unusual activity detected.'}
          </div>
        )}
      </div>

      {/* Tabs */}
      <div style={{
        display: 'flex', gap: 2, padding: '0 16px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-0)',
      }}>
        {tabs.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            padding: '10px 10px', fontSize: 11, lineHeight: 1.5,
            color: tab === t.id ? 'var(--fg)' : 'var(--fg-muted)',
            fontWeight: tab === t.id ? 500 : 400,
            borderBottom: tab === t.id ? '1.5px solid var(--fg)' : '1.5px solid transparent',
            marginBottom: -1,
          }}>{t.label}</button>
        ))}
      </div>

      {/* Content */}
      <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
        {tab === 'overview' && <SessionOverviewTab s={session}/>}
        {tab === 'claims' && <SessionClaimsTab s={session}/>}
        {tab === 'events' && <SessionEventsTab s={session}/>}
      </div>

      <CLIFooter command={`shark session show ${session.id}`}/>
    </div>
  );
}

/* Shared label style for section headings inside slideover */
const sectionHeadingStyle = {
  fontSize: 11,
  lineHeight: 1.5,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.08em',
  color: 'var(--fg-muted)',
  fontWeight: 500,
  marginBottom: 4,
};

function SessionField({ label, children, mono }) {
  return (
    <div style={{ padding: '6px 0', borderBottom: '1px solid var(--hairline)', display: 'grid', gridTemplateColumns: '110px 1fr', gap: 12, alignItems: 'start' }}>
      <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{label}</span>
      <div className={mono ? 'mono' : ''} style={{ fontSize: 13, lineHeight: 1.5 }}>{children}</div>
    </div>
  );
}

function FieldGroup({ title, children }) {
  return (
    <div style={{ marginBottom: 0 }}>
      <div style={{ ...sectionHeadingStyle, paddingTop: 4 }}>{title}</div>
      {children}
    </div>
  );
}

function SessionOverviewTab({ s }) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>

      {/* ── User info ── */}
      <FieldGroup title="User">
        <SessionField label="Name">{s.name}</SessionField>
        <SessionField label="Email">{s.user}</SessionField>
        <SessionField label="Auth method">{s.method}{s.mfa && <span style={{ color: 'var(--fg-muted)', marginLeft: 6, fontSize: 11 }}>+ {s.mfa}</span>}</SessionField>
        <SessionField label="Client"><span className={'chip' + (s.client === 'agent' ? ' agent' : '')}>{s.client}</span></SessionField>
      </FieldGroup>

      {/* ── Connection info ── */}
      <FieldGroup title="Connection">
        <SessionField label="IP address" mono>{s.ip}</SessionField>
        <SessionField label="Location">
          {s.city}{s.country && s.country !== '—' && `, ${s.country}`}
          {s.region && <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)', marginLeft: 6 }}>{s.region}</span>}
        </SessionField>
        <SessionField label="Device">{s.dev.split('·')[0].trim()}</SessionField>
        <SessionField label="User agent">
          <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)', wordBreak: 'break-all' }}>{s.dev}</span>
        </SessionField>
      </FieldGroup>

      {/* ── Timing ── */}
      <FieldGroup title="Timing">
        <SessionField label="Started" mono>{relTime(s.created)}</SessionField>
        <SessionField label="Last seen" mono>{relTime(s.last)}</SessionField>
        <SessionField label="Expires" mono>{relTime(s.expires)}</SessionField>
      </FieldGroup>

      {/* ── Security ── */}
      <FieldGroup title="Security">
        <SessionField label="Status">
          <div className="row" style={{ gap: 8 }}>
            <span className={"dot " + (s.blocked ? 'danger' : s.suspicious ? 'warn' : 'success')}/>
            <span style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>
              {s.blocked ? 'Account blocked' : s.suspicious ? 'Flagged for review' : 'Active and healthy'}
            </span>
          </div>
        </SessionField>
      </FieldGroup>

      {/* ── Token references ── */}
      {s.jti && (
        <div style={{ padding: 10, border: '1px solid var(--hairline)', borderRadius: 4, background: 'var(--surface-1)' }}>
          <div style={{ ...sectionHeadingStyle, marginBottom: 8 }}>Token refs</div>
          <div className="col" style={{ gap: 6 }}>
            <div className="row" style={{ fontSize: 13 }}>
              <span style={{ width: 90, fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>Access JTI</span>
              <CopyField value={s.jti}/>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}

function SessionClaimsTab({ s }) {
  // If there's no JTI, this is a pure cookie session — no JWT was issued.
  if (!s.jti) {
    return (
      <div style={{
        padding: 16, border: '1px dashed var(--hairline-strong)', borderRadius: 5,
        background: 'var(--surface-1)', textAlign: 'center',
      }}>
        <div style={{ fontSize: 12, marginBottom: 6 }}>Cookie session — no JWT claims</div>
        <div className="faint" style={{ fontSize: 11, lineHeight: 1.5 }}>
          This session was created without a bearer token. Token claims are only available for JWT-mode sessions.
        </div>
      </div>
    );
  }
  const claims = {
    iss: window.location.origin,
    sub: s.user_id || ('usr_' + s.id.slice(-8)),
    sid: s.id,
    jti: s.jti,
    iat: Math.floor(s.created / 1000),
    exp: Math.floor(Math.abs(s.expires) / 1000),
    amr: [s.method, s.mfa].filter(Boolean),
    acr: s.mfa ? 'urn:shark:mfa' : 'urn:shark:single',
    scope: s.client === 'agent' ? 'openid profile workspace:read workspace:write' : 'openid profile email',
    ...(s.client === 'agent' || s.client === 'api' ? { cnf: { 'jkt': '(DPoP key thumbprint)' } } : {}),
  };
  return (
    <pre className="mono" style={{
      fontSize: 11, lineHeight: 1.6,
      background: 'var(--surface-1)',
      border: '1px solid var(--hairline-strong)',
      padding: 12, borderRadius: 5,
      color: 'var(--fg-muted)',
      margin: 0, overflow: 'auto',
    }}>{JSON.stringify(claims, null, 2)}</pre>
  );
}

function SessionEventsTab({ s }) {
  // Fetch real audit logs scoped to this user. A session_id filter would be
  // ideal, but audit logs are keyed by user_id — this is accurate (scoped to
  // the right user) though not session-isolated.
  const { data, loading } = useAPI(
    s.user_id ? `/audit-logs?actor_id=${s.user_id}&limit=30` : null,
    [s.user_id]
  );
  const events = data?.data || [];

  if (loading) {
    return <div className="faint" style={{ fontSize: 11, padding: 8 }}>Loading events…</div>;
  }
  if (events.length === 0) {
    return <div className="faint" style={{ fontSize: 11, padding: 8 }}>No audit events found for this user.</div>;
  }
  return (
    <div className="col" style={{ gap: 0 }}>
      {events.map((e, i) => (
        <div key={e.id || i} className="row" style={{ padding: '7px 0', borderBottom: '1px solid var(--hairline)', gap: 10 }}>
          <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)', width: 70 }}>
            {relTime(new Date(e.created_at).getTime())}
          </span>
          <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, width: 160 }}>{e.action}</span>
          <span className="mono" style={{ fontSize: 11, lineHeight: 1.5, color: 'var(--fg-muted)' }}>{e.status || ''}</span>
        </div>
      ))}
    </div>
  );
}
