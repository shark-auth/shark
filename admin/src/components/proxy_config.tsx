// @ts-nocheck
import React from 'react'
import { Icon, CopyField } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'

// Phase 6 P5 — Proxy Config dashboard UI.
// Read-only circuit breaker strip, URL simulator (POST /admin/proxy/simulate),
// and rules table. Rules are declared in sharkauth.yaml; inline editor arrives in P5.1.
//
// EventSource does not support Bearer auth headers. Per the design brief, we poll
// GET /admin/proxy/status every 2s (pausing while the tab is hidden).

const SECTION_LABEL = {
  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
  color: 'var(--fg-dim)', fontWeight: 500,
};

const HAIRLINE = '1px solid var(--hairline)';

// Poll proxy status. Returns {stats, status, loading, refresh, probing}.
// `status` is the HTTP status of the last call (for detecting 404 → proxy disabled).
function useProxyStats() {
  const [stats, setStats] = React.useState(null);
  const [status, setStatus] = React.useState(0);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);
  const [probing, setProbing] = React.useState(false);

  const fetchOnce = React.useCallback(async () => {
    const key = sessionStorage.getItem('shark_admin_key');
    if (!key) return;
    try {
      const res = await fetch('/api/v1/admin/proxy/status', {
        headers: { 'Authorization': 'Bearer ' + key },
      });
      setStatus(res.status);
      if (res.status === 404) {
        setStats(null);
        setError(null);
      } else if (res.ok) {
        const j = await res.json();
        setStats(j?.data || null);
        setError(null);
      } else {
        setError('HTTP ' + res.status);
      }
    } catch (e) {
      setError(e.message || 'network error');
    } finally {
      setLoading(false);
    }
  }, []);

  const refresh = React.useCallback(async () => {
    setProbing(true);
    await fetchOnce();
    // keep the "probing" indicator up briefly so the user sees gauges flash
    setTimeout(() => setProbing(false), 600);
  }, [fetchOnce]);

  React.useEffect(() => {
    fetchOnce();
    let interval = null;
    const start = () => {
      if (interval != null) return;
      interval = setInterval(fetchOnce, 2000);
    };
    const stop = () => {
      if (interval != null) { clearInterval(interval); interval = null; }
    };
    if (document.visibilityState === 'visible') start();
    const onVis = () => {
      if (document.visibilityState === 'visible') { fetchOnce(); start(); } else stop();
    };
    document.addEventListener('visibilitychange', onVis);
    return () => { stop(); document.removeEventListener('visibilitychange', onVis); };
  }, [fetchOnce]);

  return { stats, status, loading, error, refresh, probing };
}

// ---- Header ----

function Header({ upstream, onTest, probing, disabled }) {
  return (
    <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
      <div className="row" style={{ gap: 12 }}>
        <div style={{ flex: 1 }}>
          <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>
            Proxy
          </h1>
          <div className="row" style={{ marginTop: 4, gap: 8 }}>
            <span className="faint" style={{ fontSize: 11.5 }}>
              Reverse proxy with header injection
            </span>
            {upstream && (
              <>
                <span className="faint" style={{ fontSize: 11 }}>·</span>
                <span className="chip" title={upstream} style={{ fontSize: 11 }}>
                  <Icon.Globe width={10} height={10} style={{ opacity: 0.7 }}/>
                  <span className="mono">{upstream}</span>
                </span>
              </>
            )}
          </div>
        </div>
        {!disabled && (
          <button className="btn" onClick={onTest} disabled={probing}>
            <Icon.Refresh width={11} height={11}
              style={{ animation: probing ? 'spin 800ms linear infinite' : 'none' }}/>
            {probing ? 'Testing…' : 'Test connection'}
          </button>
        )}
      </div>
    </div>
  );
}

// ---- Circuit-open banner ----

function CircuitOpenBanner({ stats }) {
  if (!stats || stats.State === 'closed') return null;
  const isOpen = stats.State === 'open';
  const color = isOpen ? 'var(--danger)' : 'var(--warn)';
  return (
    <div style={{
      margin: '12px 20px 0', padding: '10px 12px',
      border: `1px solid color-mix(in oklch, ${color} 35%, var(--hairline-strong))`,
      background: `color-mix(in oklch, ${color} 10%, var(--surface-1))`,
      borderRadius: 5,
      display: 'flex', alignItems: 'center', gap: 10,
    }}>
      <Icon.Warn width={13} height={13} style={{ color, flexShrink: 0 }}/>
      <div style={{ flex: 1, fontSize: 12.5, lineHeight: 1.5 }}>
        <strong style={{ color }}>
          {isOpen ? 'Upstream health check failing.' : 'Upstream degraded, probing.'}
        </strong>
        <span className="muted" style={{ marginLeft: 6 }}>
          Sessions serving from cache ({stats.CacheSize} entries). Agents unaffected.
        </span>
      </div>
    </div>
  );
}

// ---- Circuit strip ----

function Gauge({ title, sublabel, state, emphasis }) {
  const dotClass =
    state === 'closed' ? 'dot success pulse' :
    state === 'half-open' ? 'dot warn pulse fast' :
    state === 'open' ? 'dot danger' :
    'dot';
  const label =
    state === 'closed' ? 'Healthy' :
    state === 'half-open' ? 'Half-open' :
    state === 'open' ? 'Open' : 'N/A';
  const labelColor =
    state === 'closed' ? 'var(--success)' :
    state === 'half-open' ? 'var(--warn)' :
    state === 'open' ? 'var(--danger)' :
    'var(--fg-dim)';
  return (
    <div style={{ padding: '14px 18px', display: 'flex', flexDirection: 'column', gap: 6 }}>
      <div style={{ ...SECTION_LABEL }}>{title}</div>
      <div className="row" style={{ gap: 8 }}>
        <span className={dotClass} style={{ width: 8, height: 8 }}/>
        <span style={{
          fontSize: 15,
          fontWeight: 600,
          fontFamily: 'var(--font-display)',
          color: labelColor,
          letterSpacing: '-0.01em',
        }}>{label}</span>
        {emphasis && (
          <span className="mono faint" style={{ fontSize: 11, marginLeft: 'auto' }}>{emphasis}</span>
        )}
      </div>
      <div className="mono" style={{ fontSize: 11, color: 'var(--fg-dim)' }}>{sublabel || '—'}</div>
    </div>
  );
}

function formatLatency(ns) {
  if (!ns || ns <= 0) return null;
  const us = ns / 1000;
  if (us < 1000) return us.toFixed(0) + 'µs';
  const ms = us / 1000;
  if (ms < 1000) return ms.toFixed(1) + 'ms';
  return (ms / 1000).toFixed(2) + 's';
}

function formatRelative(iso) {
  if (!iso) return null;
  const t = new Date(iso).getTime();
  if (!t) return null;
  const dt = Date.now() - t;
  if (dt < 1500) return 'just now';
  if (dt < 60_000) return Math.floor(dt / 1000) + 's ago';
  if (dt < 3_600_000) return Math.floor(dt / 60_000) + 'm ago';
  return Math.floor(dt / 3_600_000) + 'h ago';
}

function CircuitStrip({ stats }) {
  // L1 JWT local and L2 Session cache are always "closed" when the proxy is up;
  // L3 reflects the backend breaker state directly.
  const l3 = stats?.State || 'n/a';
  const lat = stats?.LastLatency ? formatLatency(stats.LastLatency) : null;
  const rel = stats?.LastCheck ? formatRelative(stats.LastCheck) : null;
  const statusCode = stats?.LastStatus ? String(stats.LastStatus) : '';
  return (
    <div style={{
      margin: '12px 20px 0',
      display: 'grid',
      gridTemplateColumns: 'repeat(3, 1fr)',
      border: HAIRLINE,
      borderRadius: 5,
      background: 'var(--surface-1)',
      overflow: 'hidden',
    }}>
      <div style={{ borderRight: HAIRLINE }}>
        <Gauge
          title="L1 · JWT local"
          state={stats ? 'closed' : 'n/a'}
          sublabel="Verified in-process"
          emphasis="<1ms"
        />
      </div>
      <div style={{ borderRight: HAIRLINE }}>
        <Gauge
          title="L2 · Session cache"
          state={stats ? 'closed' : 'n/a'}
          sublabel={stats ? (stats.CacheSize + ' hits · ' + stats.NegCacheSize + ' neg') : 'Idle'}
        />
      </div>
      <div>
        <Gauge
          title="L3 · Upstream health"
          state={l3}
          sublabel={
            rel
              ? `${rel}${statusCode ? ' · ' + statusCode : ''}${stats?.Failures ? ' · ' + stats.Failures + ' fail' : ''}`
              : 'No probes yet'
          }
          emphasis={lat}
        />
      </div>
    </div>
  );
}

// ---- Simulator ----

const METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

function Simulator({ method, setMethod, path, setPath, identity, setIdentity, onRun, running, result, rulesEmpty }) {
  const [idOpen, setIdOpen] = React.useState(false);
  const setField = (k, v) => setIdentity({ ...identity, [k]: v });

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ ...SECTION_LABEL, marginBottom: 10 }}>URL Simulator</div>
      <div className="row" style={{ gap: 8, alignItems: 'stretch' }}>
        <select
          value={method}
          onChange={e => setMethod(e.target.value)}
          className="mono"
          style={{
            height: 32, padding: '0 8px',
            background: 'var(--surface-2)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4,
            color: 'var(--fg)',
            fontSize: 12,
            fontFamily: 'var(--font-mono)',
            minWidth: 90,
          }}
        >
          {METHODS.map(m => <option key={m} value={m}>{m}</option>)}
        </select>
        <input
          placeholder="/path/to/resource"
          value={path}
          onChange={e => setPath(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter' && path.trim()) onRun(); }}
          className="mono"
          style={{
            flex: 1, height: 32, padding: '0 10px',
            background: 'var(--surface-2)',
            border: '1px solid var(--hairline-strong)',
            borderRadius: 4,
            color: 'var(--fg)',
            fontSize: 12.5,
            fontFamily: 'var(--font-mono)',
            outline: 'none',
          }}
        />
        <button className="btn primary" onClick={onRun} disabled={running || !path.trim()}>
          <Icon.Bolt width={11} height={11}/>
          {running ? 'Simulating…' : 'Simulate'}
        </button>
      </div>

      <button
        className="btn ghost sm"
        onClick={() => setIdOpen(o => !o)}
        style={{ marginTop: 10, paddingLeft: 0 }}
      >
        {idOpen ? <Icon.ChevronDown width={10} height={10}/> : <Icon.ChevronRight width={10} height={10}/>}
        Identity override
        {!idOpen && (identity.user_id || identity.roles || identity.scopes || identity.auth_method) && (
          <span className="faint" style={{ fontSize: 10.5, marginLeft: 6 }}>· custom</span>
        )}
      </button>

      {idOpen && (
        <div style={{
          marginTop: 8,
          padding: 12,
          border: HAIRLINE,
          borderRadius: 4,
          background: 'var(--surface-1)',
          display: 'grid',
          gridTemplateColumns: '1fr 1fr',
          columnGap: 10,
          rowGap: 8,
        }}>
          <IdentityInput label="user_id" value={identity.user_id} onChange={v => setField('user_id', v)} placeholder="usr_abc123"/>
          <IdentityInput label="auth_method" value={identity.auth_method} onChange={v => setField('auth_method', v)} placeholder="session | jwt | apikey"/>
          <IdentityInput label="roles" value={identity.roles} onChange={v => setField('roles', v)} placeholder="admin,billing (comma)"/>
          <IdentityInput label="scopes" value={identity.scopes} onChange={v => setField('scopes', v)} placeholder="read:users,write:orgs (comma)"/>
        </div>
      )}

      {result && <ResultCard result={result} path={path} rulesEmpty={rulesEmpty}/>}
    </div>
  );
}

function IdentityInput({ label, value, onChange, placeholder }) {
  return (
    <label className="col" style={{ gap: 4 }}>
      <span className="mono" style={{ fontSize: 10, color: 'var(--fg-dim)' }}>{label}</span>
      <input
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="mono"
        style={{
          height: 26, padding: '0 8px',
          background: 'var(--surface-2)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 3,
          color: 'var(--fg)',
          fontSize: 11.5,
          fontFamily: 'var(--font-mono)',
          outline: 'none',
        }}
      />
    </label>
  );
}

// ---- Result card ----

function ResultCard({ result, path }) {
  if (result.error) {
    return (
      <div style={{
        marginTop: 12, padding: 12,
        border: '1px solid color-mix(in oklch, var(--danger) 35%, var(--hairline-strong))',
        background: 'color-mix(in oklch, var(--danger) 8%, var(--surface-2))',
        borderRadius: 5, fontSize: 12.5,
      }}>
        <div style={{ color: 'var(--danger)', fontWeight: 600, marginBottom: 4 }}>Simulator error</div>
        <div className="muted">{result.error}</div>
      </div>
    );
  }

  const allow = result.decision === 'allow';
  const noMatch = !result.matched_rule;
  const headers = result.injected_headers || {};
  const headerEntries = Object.entries(headers);
  const evalUs = typeof result.eval_us === 'number' ? result.eval_us : null;

  // No-match = deny with the specific copy from the brief.
  if (noMatch) {
    return (
      <div style={{
        marginTop: 12, padding: 14,
        border: '1px solid color-mix(in oklch, var(--warn) 40%, var(--hairline-strong))',
        background: 'color-mix(in oklch, var(--warn) 6%, var(--surface-2))',
        borderRadius: 5,
      }}>
        <div className="row" style={{ gap: 8, marginBottom: 6 }}>
          <span className="chip danger" style={{ height: 22 }}>403 · deny</span>
          <span style={{ fontSize: 13, color: 'var(--warn)', fontWeight: 600 }}>
            No rule matched. Would 403.
          </span>
        </div>
        <div className="muted" style={{ fontSize: 12 }}>
          <span className="mono">{path || '(empty path)'}</span> doesn't match any configured rule.
        </div>
        {evalUs != null && (
          <div className="faint mono" style={{ fontSize: 10.5, marginTop: 8 }}>
            eval {evalUs}µs
          </div>
        )}
      </div>
    );
  }

  return (
    <div style={{
      marginTop: 12, padding: 14,
      border: HAIRLINE,
      background: 'var(--surface-2)',
      borderRadius: 5,
    }}>
      <div className="row" style={{ gap: 8, marginBottom: 10, flexWrap: 'wrap' }}>
        <span className={'chip ' + (allow ? 'success' : 'danger')} style={{ height: 22 }}>
          {allow ? '200 · allow' : '403 · deny'}
        </span>
        {result.matched_rule && (
          <span className="chip" title="matched rule">
            <span className="mono">{result.matched_rule.path || result.matched_rule}</span>
            {result.matched_rule.require && (
              <span className="faint">· require {String(result.matched_rule.require)}</span>
            )}
          </span>
        )}
        <div style={{ flex: 1 }}/>
        {evalUs != null && (
          <span className="faint mono" style={{ fontSize: 10.5 }}>
            eval {evalUs}µs
          </span>
        )}
      </div>

      {result.reason && (
        <div style={{ fontSize: 12.5, marginBottom: 10, color: 'var(--fg-muted)' }}>
          {result.reason}
        </div>
      )}

      {headerEntries.length > 0 && (
        <>
          <div style={{ ...SECTION_LABEL, marginBottom: 6 }}>Injected headers</div>
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'auto 1fr',
            columnGap: 14, rowGap: 4,
            alignItems: 'center',
          }}>
            {headerEntries.map(([k, v]) => (
              <React.Fragment key={k}>
                <span className="mono" style={{ fontSize: 11, color: 'var(--fg-dim)' }}>{k}</span>
                <span
                  onClick={() => { navigator.clipboard?.writeText(String(v)); }}
                  className="mono"
                  title="Click to copy"
                  style={{
                    fontSize: 11.5,
                    color: 'var(--fg)',
                    cursor: 'pointer',
                    wordBreak: 'break-all',
                  }}
                >{String(v)}</span>
              </React.Fragment>
            ))}
          </div>
        </>
      )}

      {headerEntries.length === 0 && allow && (
        <div className="faint" style={{ fontSize: 11 }}>No headers injected.</div>
      )}
    </div>
  );
}

// ---- Rules table ----

function methodChips(methods) {
  if (!methods || methods.length === 0) {
    return <span className="chip ghost" style={{ height: 18, fontSize: 10 }}>ANY</span>;
  }
  return (
    <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
      {methods.map(m => (
        <span key={m} className="chip" style={{ height: 18, fontSize: 10 }}>{m}</span>
      ))}
    </div>
  );
}

function requireChip(require) {
  if (!require || require === 'anonymous') {
    return <span className="chip ghost" style={{ height: 18, fontSize: 10 }}>anonymous</span>;
  }
  const label = typeof require === 'string' ? require : JSON.stringify(require);
  const cls =
    label === 'user' ? 'chip' :
    label === 'agent' ? 'chip agent' :
    label === 'admin' ? 'chip success' :
    'chip solid';
  return <span className={cls} style={{ height: 18, fontSize: 10 }}>{label}</span>;
}

function scopesChip(scopes) {
  if (!scopes || scopes.length === 0) return <span className="faint" style={{ fontSize: 11 }}>—</span>;
  return (
    <div className="row" style={{ gap: 3, flexWrap: 'wrap' }}>
      {scopes.map(s => (
        <span key={s} className="chip" style={{ height: 18, fontSize: 10 }}>
          <span className="mono">{s}</span>
        </span>
      ))}
    </div>
  );
}

function RulesTable({ rules, onTest }) {
  return (
    <div style={{ padding: '0 20px 20px' }}>
      <div className="row" style={{ marginBottom: 8 }}>
        <div style={SECTION_LABEL}>Rules</div>
        <div style={{ flex: 1 }}/>
        <span className="faint" style={{ fontSize: 11 }}>
          {rules.length} {rules.length === 1 ? 'rule' : 'rules'} · evaluated top-to-bottom
        </span>
      </div>

      {/* Inline banner */}
      <div style={{
        padding: '8px 12px',
        border: HAIRLINE,
        background: 'var(--surface-1)',
        borderRadius: 4,
        display: 'flex', alignItems: 'center', gap: 8,
        marginBottom: 10,
      }}>
        <Icon.Info width={12} height={12} style={{ color: 'var(--fg-dim)', flexShrink: 0 }}/>
        <span style={{ fontSize: 12, color: 'var(--fg-muted)' }}>
          Rules are defined in <span className="mono">sharkauth.yaml</span>. Reload config to apply changes.
        </span>
        <div style={{ flex: 1 }}/>
        <span className="faint mono" style={{ fontSize: 10.5 }}>
          Inline editor · P5.1
        </span>
      </div>

      {rules.length === 0 ? (
        <RulesEmpty/>
      ) : (
        <div style={{ border: HAIRLINE, borderRadius: 5, overflow: 'hidden', background: 'var(--surface-1)' }}>
          <table className="tbl" style={{ width: '100%' }}>
            <thead>
              <tr>
                <th style={{ width: 48 }}>#</th>
                <th>Path</th>
                <th style={{ width: 160 }}>Methods</th>
                <th style={{ width: 120 }}>Require</th>
                <th>Scopes</th>
                <th style={{ width: 80 }}></th>
              </tr>
            </thead>
            <tbody>
              {rules.map((r, i) => (
                <tr key={i} style={{ height: 40 }}>
                  <td className="mono faint" style={{ fontSize: 11 }}>{i + 1}</td>
                  <td>
                    <span className="mono" style={{ fontSize: 11.5, color: 'var(--fg)' }}>
                      {r.path || '(catch-all)'}
                    </span>
                  </td>
                  <td>{methodChips(r.methods)}</td>
                  <td>{requireChip(r.require)}</td>
                  <td>{scopesChip(r.scopes)}</td>
                  <td style={{ textAlign: 'right' }}>
                    <button
                      className="btn ghost sm"
                      onClick={() => onTest(r)}
                      title="Test this rule in the simulator"
                    >
                      <Icon.Bolt width={10} height={10}/>
                      Test
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function RulesEmpty() {
  return (
    <div style={{
      padding: '32px 20px', textAlign: 'center',
      border: HAIRLINE, borderRadius: 5, background: 'var(--surface-1)',
    }}>
      <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--fg)', marginBottom: 6 }}>
        No rules configured.
      </div>
      <div className="muted" style={{ fontSize: 12.5, maxWidth: 460, margin: '0 auto', lineHeight: 1.6 }}>
        Every request returns 403 (default deny). Add rules to{' '}
        <span className="mono">sharkauth.yaml</span> or ship an open route with{' '}
        <span className="mono">allow: anonymous</span>.
      </div>
    </div>
  );
}

// ---- Proxy-disabled empty state ----

function ProxyDisabledEmpty() {
  const yaml = `proxy:
  enabled: true
  upstream: http://localhost:3000`;
  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      <div style={{ padding: '14px 20px', borderBottom: HAIRLINE }}>
        <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600, fontFamily: 'var(--font-display)' }}>
          Proxy
        </h1>
        <div className="faint" style={{ marginTop: 4, fontSize: 11.5 }}>
          Reverse proxy with header injection
        </div>
      </div>
      <div style={{
        flex: 1,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        padding: 40,
      }}>
        <div style={{ maxWidth: 520, width: '100%' }}>
          <div className="row" style={{ gap: 8, marginBottom: 10 }}>
            <Icon.Proxy width={14} height={14} style={{ color: 'var(--fg-dim)' }}/>
            <span style={SECTION_LABEL}>Not configured</span>
          </div>
          <div style={{ fontSize: 15, fontWeight: 600, color: 'var(--fg)', marginBottom: 6 }}>
            Proxy not configured.
          </div>
          <div className="muted" style={{ fontSize: 12.5, lineHeight: 1.6, marginBottom: 14 }}>
            Enable in <span className="mono">sharkauth.yaml</span> to route requests through Shark.
          </div>
          <div style={{
            position: 'relative',
            border: HAIRLINE,
            borderRadius: 5,
            background: 'var(--surface-1)',
            padding: 12,
            paddingRight: 46,
          }}>
            <pre className="mono" style={{
              margin: 0,
              fontSize: 11.5,
              color: 'var(--fg)',
              lineHeight: 1.65,
              whiteSpace: 'pre-wrap',
            }}>{yaml}</pre>
            <div style={{ position: 'absolute', top: 8, right: 8 }}>
              <CopyField value={yaml} truncate={0}/>
            </div>
          </div>
          <div style={{ marginTop: 14 }}>
            <CLIFooter command="shark serve --proxy-upstream http://localhost:3000"/>
          </div>
        </div>
      </div>
    </div>
  );
}

// ---- Main ----

export function Proxy() {
  const toast = useToast();
  const { stats, status, loading, refresh, probing } = useProxyStats();
  const { data: rulesData, loading: rulesLoading } = useAPI(status === 404 ? null : '/admin/proxy/rules');

  const [method, setMethod] = React.useState('GET');
  const [path, setPath] = React.useState('');
  const [identity, setIdentity] = React.useState({ user_id: '', roles: '', scopes: '', auth_method: '' });
  const [result, setResult] = React.useState(null);
  const [running, setRunning] = React.useState(false);

  if (status === 404) return <ProxyDisabledEmpty/>;

  if (loading && !stats) {
    return (
      <div className="faint" style={{ padding: 24, fontSize: 12 }}>Loading proxy status…</div>
    );
  }

  const rules = rulesData?.data || [];
  const upstream = stats?.HealthURL || '';

  const onRun = async () => {
    if (!path.trim()) return;
    setRunning(true);
    setResult(null);
    try {
      const body = {
        method,
        path: path.trim(),
        identity: {
          user_id: identity.user_id || '',
          auth_method: identity.auth_method || '',
          roles: identity.roles ? identity.roles.split(',').map(s => s.trim()).filter(Boolean) : [],
          scopes: identity.scopes ? identity.scopes.split(',').map(s => s.trim()).filter(Boolean) : [],
        },
      };
      const res = await API.post('/admin/proxy/simulate', body);
      // Backend returns the raw simulation payload, not wrapped in {data}
      setResult(res?.data || res);
    } catch (e) {
      setResult({ error: e.message || 'Simulation failed' });
      toast?.error?.('Simulation failed: ' + (e.message || 'unknown'));
    } finally {
      setRunning(false);
    }
  };

  const onTestRule = (rule) => {
    setPath(rule.path || '/');
    if (rule.methods && rule.methods.length > 0) setMethod(rule.methods[0]);
    // scroll up to bring the simulator into view
    setTimeout(() => {
      document.querySelector('[data-proxy-sim]')?.scrollIntoView({ block: 'start', behavior: 'smooth' });
    }, 30);
  };

  const onTest = async () => {
    await refresh();
    toast?.info?.('Probed upstream');
  };

  return (
    <div className="col" style={{ height: '100%', overflow: 'auto' }}>
      <Header upstream={upstream} onTest={onTest} probing={probing}/>
      <CircuitOpenBanner stats={stats}/>
      <CircuitStrip stats={stats}/>
      <div data-proxy-sim>
        <Simulator
          method={method} setMethod={setMethod}
          path={path} setPath={setPath}
          identity={identity} setIdentity={setIdentity}
          onRun={onRun} running={running}
          result={result}
          rulesEmpty={rules.length === 0}
        />
      </div>
      <RulesTable rules={rules} onTest={onTestRule}/>
      {rulesLoading && rules.length === 0 && (
        <div className="faint" style={{ padding: '0 20px 12px', fontSize: 11.5 }}>Loading rules…</div>
      )}
      <div style={{ marginTop: 'auto' }}>
        <CLIFooter command={`shark serve --proxy-upstream ${upstream || 'http://your-api'}`}/>
      </div>
      {/* Inline keyframes for refresh icon spin — keeps this page self-contained */}
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } } .dot.pulse.fast { animation-duration: 1s; }`}</style>
    </div>
  );
}
