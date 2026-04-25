// @ts-nocheck
import React from 'react'
import { Icon } from './shared'
import { API, useAPI } from './api'
import { CLIFooter } from './CLIFooter'
import { useToast } from './toast'
import { ProxyWizard } from './proxy_wizard'

// Proxy Gateway — sibling page to users.tsx.
// Monochrome, square (radius cap 5px / inputs 4px), hairline borders,
// 13px base, dense 7-10px row padding. Drawers are right-side fixed
// hairline-bordered panels — never modals. YAML UI removed entirely.

// ─── Require grammar validator ────────────────────────────────────────────────

const REQUIRE_RE = /^(anonymous|authenticated|agent|role:.+|global_role:.+|permission:.+:.+|scope:.+|tier:.+)$/;
const ALLOW_VALUES = ['', 'anonymous'];

function validateRule(f) {
  const errs: any = {};
  if (!f.name?.trim()) errs.name = 'Required';
  if (!f.pattern?.trim()) errs.pattern = 'Required';
  if (f.pattern && !f.pattern.startsWith('/')) errs.pattern = 'Must start with /';
  const hasRequire = f.require?.trim();
  const hasAllow = f.allow?.trim();
  if (!hasRequire && !hasAllow) errs.require = 'Set require or allow';
  if (hasRequire && hasAllow) errs.require = 'Set only one of require / allow';
  if (hasRequire && !REQUIRE_RE.test(f.require.trim())) errs.require = 'Unknown require value';
  if (hasAllow && !ALLOW_VALUES.includes(f.allow.trim())) errs.allow = 'Only "anonymous" accepted';
  return errs;
}

// ─── Lifecycle hook ───────────────────────────────────────────────────────────

function useLifecycle() {
  const [status, setStatus] = React.useState<any>(null);
  const [loading, setLoading] = React.useState(true);
  const [missing, setMissing] = React.useState(false);

  const fetch_ = React.useCallback(async () => {
    try {
      const r = await API.get('/admin/proxy/lifecycle');
      setStatus(r?.data || null);
      setMissing(false);
    } catch (e: any) {
      if (e.message?.includes('404') || e.message === 'HTTP 404') {
        setMissing(true);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetch_();
    const iv = setInterval(fetch_, 5000);
    return () => clearInterval(iv);
  }, [fetch_]);

  const act = async (action) => {
    const r = await API.post('/admin/proxy/' + action);
    setStatus(r?.data || null);
  };

  return { status, loading, missing, refresh: fetch_, act };
}

// ─── Live proxy status (SSE w/ poll fallback) ────────────────────────────────
// Backs the Breaker / Failures / Cache tiles + per-listener upstream health.
// /admin/proxy/status returns { state, cache_size, neg_cache_size, failures,
// last_check, last_latency_ms, last_status, health_url, upstream, breaker_state }.
// 404 when proxy disabled — caller already handles via useLifecycle.

function useProxyStatus(enabled: boolean) {
  const [data, setData] = React.useState<any>(null);

  React.useEffect(() => {
    if (!enabled) { setData(null); return; }
    let cancelled = false;
    let es: EventSource | null = null;
    let pollTimer: any = null;
    let backoffTimer: any = null;
    let retry = 0;

    const pollOnce = async () => {
      try {
        const r = await API.get('/admin/proxy/status');
        if (!cancelled) setData(r?.data || null);
      } catch { /* 404 = disabled, ignore */ }
    };

    const startPoll = () => {
      pollOnce();
      pollTimer = setInterval(pollOnce, 5000);
    };

    const startStream = () => {
      const key = localStorage.getItem('shark_admin_key');
      if (!key) { startPoll(); return; }
      try {
        es = new EventSource(`/api/v1/admin/proxy/status/stream?token=${key}`);
        es.onopen = () => { retry = 0; };
        es.onmessage = (ev) => {
          try {
            const parsed = JSON.parse(ev.data);
            if (!cancelled) setData(parsed?.data || parsed);
          } catch { /* keep last */ }
        };
        es.onerror = () => {
          es?.close();
          es = null;
          // Exponential backoff, max 30s. Mirrors overview activity stream.
          backoffTimer = setTimeout(() => {
            retry += 1;
            startStream();
          }, Math.min(1000 * Math.pow(2, retry), 30000));
        };
      } catch {
        startPoll();
      }
    };

    startStream();
    // Always seed with one poll so tiles populate before first SSE message.
    pollOnce();

    return () => {
      cancelled = true;
      es?.close();
      if (pollTimer) clearInterval(pollTimer);
      if (backoffTimer) clearTimeout(backoffTimer);
    };
  }, [enabled]);

  return data;
}

// ─── Compiled engine rules view (read-only) ──────────────────────────────────
// /admin/proxy/rules returns the *compiled* rules — what the engine actually
// loaded after parsing the DB rules + any boot-time YAML. Useful for verifying
// a DB write made it through and for spotting compile-time drops.

function useCompiledRules(enabled: boolean) {
  const [rules, setRules] = React.useState<any[] | null>(null);

  const refresh = React.useCallback(async () => {
    if (!enabled) { setRules(null); return; }
    try {
      const r = await API.get('/admin/proxy/rules');
      setRules(r?.data || []);
    } catch {
      setRules([]);
    }
  }, [enabled]);

  React.useEffect(() => { refresh(); }, [refresh]);

  return { rules, refresh };
}

// ─── Format helpers ───────────────────────────────────────────────────────────

function fmtUptime(startedAt) {
  if (!startedAt) return '—';
  const ms = Date.now() - new Date(startedAt).getTime();
  if (ms < 0 || isNaN(ms)) return '—';
  const s = Math.floor(ms / 1000);
  if (s < 60) return s + 's';
  const m = Math.floor(s / 60);
  if (m < 60) return m + 'm';
  const h = Math.floor(m / 60);
  if (h < 24) return h + 'h ' + (m % 60) + 'm';
  const d = Math.floor(h / 24);
  return d + 'd ' + (h % 24) + 'h';
}

const dotColor = (state) => {
  if (state === 'running') return 'var(--success)';
  if (state === 'reloading') return 'var(--warn)';
  return 'var(--fg-dim)';
};

// ─── Power toggle (monochrome, square, hairline) ──────────────────────────────

function PowerToggle({ status, act, toast }) {
  const state = status?.state_str || 'unknown';
  const [busy, setBusy] = React.useState(false);
  const isRunning = state === 'running' || state === 'reloading';
  const label = busy ? '…' : isRunning ? 'Stop proxy' : 'Start proxy';

  const onClick = async () => {
    if (busy) return;
    if (isRunning && !window.confirm('Stop the proxy? In-flight requests will drain.')) return;
    setBusy(true);
    try {
      await act(isRunning ? 'stop' : 'start');
      toast.success(isRunning ? 'Proxy stopped' : 'Proxy started');
    } catch (e: any) {
      toast.error(e.message || 'Action failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <button
      onClick={onClick}
      disabled={busy}
      title={isRunning ? 'Stop proxy' : 'Start proxy'}
      style={{
        display: 'inline-flex', alignItems: 'center', gap: 8,
        height: 28, padding: '0 12px',
        background: 'var(--surface-1)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 4,
        color: 'var(--fg)',
        fontSize: 13, lineHeight: 1, cursor: busy ? 'default' : 'pointer',
        opacity: busy ? 0.6 : 1,
      }}
    >
      <span style={{
        width: 7, height: 7, borderRadius: '50%',
        background: dotColor(state),
        flexShrink: 0,
      }}/>
      {label}
    </button>
  );
}

// ─── Status strip (monochrome stat tiles) ─────────────────────────────────────

function StatTile({ label, value, mono = false, dotState = null, last = false }) {
  return (
    <div style={{
      flex: 1, minWidth: 0,
      padding: '8px 12px',
      borderRight: last ? 'none' : '1px solid var(--hairline)',
      display: 'flex', flexDirection: 'column', gap: 3,
    }}>
      <div style={{
        fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
        color: 'var(--fg-muted)', lineHeight: 1.3, fontWeight: 500,
      }}>{label}</div>
      <div className="row" style={{ gap: 6, alignItems: 'center', minHeight: 16 }}>
        {dotState && (
          <span style={{
            width: 6, height: 6, borderRadius: '50%',
            background: dotColor(dotState), flexShrink: 0,
          }}/>
        )}
        <span
          className={mono ? 'mono' : ''}
          style={{
            fontSize: 13, color: 'var(--fg)',
            lineHeight: 1.3,
            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
          }}
        >
          {value ?? '—'}
        </span>
      </div>
    </div>
  );
}

function StatusStrip({ lifecycle, status, rulesCount, lastSimMicros }) {
  const state = lifecycle?.state_str || 'unknown';
  const breaker = status?.breaker_state || status?.breaker || lifecycle?.breaker_state || '—';
  const failures = status?.failures ?? status?.failure_count ?? lifecycle?.failure_count ?? 0;
  const cacheSize = status?.cache_size ?? status?.cache_entries ?? lifecycle?.cache_size ?? '—';
  return (
    <div style={{
      display: 'flex',
      border: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      borderRadius: 4,
      overflow: 'hidden',
    }}>
      <StatTile label="State" value={state} dotState={state}/>
      <StatTile label="Listeners" value={lifecycle?.listeners ?? 0} mono/>
      <StatTile label="Rules" value={rulesCount} mono/>
      <StatTile label="Uptime" value={fmtUptime(lifecycle?.started_at)} mono/>
      <StatTile label="Breaker" value={breaker} mono/>
      <StatTile label="Failures" value={failures} mono/>
      <StatTile label="Cache" value={cacheSize} mono/>
      <StatTile label="Last sim" value={lastSimMicros != null ? lastSimMicros + 'µs' : '—'} mono last/>
    </div>
  );
}

// ─── Topology row ─────────────────────────────────────────────────────────────

function TopologyRow({ app, health, onReload, busy }) {
  // Health surfaces only when this row's protected_url matches the live
  // upstream — backend's /admin/proxy/status reports a single upstream today.
  const showHealth = health && health.upstream && app.proxy_protected_url
    && (health.upstream === app.proxy_protected_url
        || app.proxy_protected_url.startsWith(health.upstream)
        || health.upstream.startsWith(app.proxy_protected_url));
  const statusCode = health?.last_status;
  const latency = health?.last_latency_ms;
  const healthy = statusCode != null && statusCode >= 200 && statusCode < 400;
  const healthDot = statusCode == null
    ? 'var(--fg-dim)'
    : healthy ? 'var(--success)' : 'var(--danger)';
  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 10,
      padding: '8px 12px',
      borderBottom: '1px solid var(--hairline)',
    }}>
      <div className="mono" style={{
        flex: 1, minWidth: 0, fontSize: 12,
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        color: 'var(--fg)',
      }}>{app.proxy_public_domain}</div>
      <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>→</span>
      <div className="mono faint" style={{
        flex: 1, minWidth: 0, fontSize: 11,
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
      }}>{app.proxy_protected_url}</div>
      {showHealth && (
        <span title={`Last upstream check: ${statusCode} in ${latency}ms`} style={{
          display: 'inline-flex', alignItems: 'center', gap: 5,
          fontSize: 11, color: 'var(--fg-muted)',
          fontFamily: 'var(--font-mono)',
        }}>
          <span style={{
            width: 6, height: 6, borderRadius: '50%',
            background: healthDot,
            display: 'inline-block',
          }}/>
          {statusCode ?? '—'}{latency != null && ` · ${latency}ms`}
        </span>
      )}
      <button
        onClick={onReload}
        disabled={busy}
        title="Reload listeners"
        style={{
          height: 22, padding: '0 8px',
          background: 'transparent',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 4,
          color: 'var(--fg-muted)',
          fontSize: 11, cursor: busy ? 'default' : 'pointer',
          opacity: busy ? 0.5 : 1,
          display: 'inline-flex', alignItems: 'center', gap: 4,
        }}
      >
        <Icon.Refresh width={10} height={10}/>
        Reload
      </button>
    </div>
  );
}

// ─── Inline square slide-toggle ───────────────────────────────────────────────

function InlineToggle({ on, busy, onChange, title }) {
  return (
    <button
      onClick={(e) => { e.stopPropagation(); if (!busy) onChange(); }}
      disabled={busy}
      title={title}
      aria-pressed={on}
      style={{
        width: 26, height: 14,
        background: on ? 'var(--fg)' : 'var(--surface-2)',
        border: '1px solid var(--hairline-strong)',
        borderRadius: 3,
        position: 'relative',
        cursor: busy ? 'default' : 'pointer',
        opacity: busy ? 0.5 : 1,
        padding: 0,
        transition: 'background 100ms ease',
      }}
    >
      <span style={{
        position: 'absolute',
        top: 1, left: on ? 13 : 1,
        width: 10, height: 10,
        background: on ? 'var(--bg)' : 'var(--fg-dim)',
        borderRadius: 2,
        transition: 'left 100ms ease',
      }}/>
    </button>
  );
}

// ─── Main component ──────────────────────────────────────────────────────────

export function Proxy() {
  const toast = useToast();
  const lifecycle = useLifecycle();
  const { data: appsRaw } = useAPI('/admin/apps');
  const apps = appsRaw?.data || [];
  const { data: rulesRaw, refresh: refreshRules } = useAPI('/admin/proxy/rules/db');
  const allRules = rulesRaw?.data || [];

  const [appFilter, setAppFilter] = React.useState('all');
  const [rulesView, setRulesView] = React.useState<'db' | 'compiled'>('db');
  const [editing, setEditing] = React.useState<any>(null); // null | rule | { __new: true }
  const [simulateOpen, setSimulateOpen] = React.useState(false);
  const [reloadBusy, setReloadBusy] = React.useState(false);
  const [lastSimMicros, setLastSimMicros] = React.useState<number | null>(null);

  // Live status (breaker, cache, upstream health) — only meaningful when proxy
  // is actually wired. Disable the hook (and SSE) when lifecycle is missing.
  const proxyStatus = useProxyStatus(!lifecycle.missing);
  const compiled = useCompiledRules(!lifecycle.missing && rulesView === 'compiled');

  const rules = appFilter === 'all' ? allRules : allRules.filter(r => r.app_id === appFilter);
  const meshApps = apps.filter(a => a.proxy_public_domain);

  if (lifecycle.missing) {
    return (
      <div style={{ padding: 40, textAlign: 'center' }}>
        <div className="faint" style={{ fontSize: 13, marginBottom: 16 }}>
          Proxy Gateway disabled. Enable proxy in sharkauth.yaml.
        </div>
        <div style={{ maxWidth: 500, margin: '24px auto' }}>
          <ProxyWizard onComplete={() => lifecycle.refresh()}/>
        </div>
      </div>
    );
  }

  const doReload = async () => {
    if (reloadBusy) return;
    setReloadBusy(true);
    try {
      await lifecycle.act('reload');
      // Reload is the only path that refreshes engine rules from DB — pull
      // the new compiled view so operator can verify the swap landed.
      compiled.refresh();
      toast.success('Reloaded');
    } catch (e: any) {
      toast.error(e.message || 'Reload failed');
    } finally {
      setReloadBusy(false);
    }
  };

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, overflow: 'hidden' }}>

        {/* Header — title, rule count chip, New rule */}
        <div style={{
          padding: '10px 16px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', gap: 10, alignItems: 'center',
          background: 'var(--surface-0)',
        }}>
          <h1 style={{
            fontSize: 16, margin: 0, fontWeight: 600,
            fontFamily: 'var(--font-display)', letterSpacing: '-0.01em', lineHeight: 1.2,
          }}>Proxy Gateway</h1>
          <span className="chip" style={{ height: 18 }}>{allRules.length} rule{allRules.length !== 1 ? 's' : ''}</span>
          <PowerToggle status={lifecycle.status} act={lifecycle.act} toast={toast}/>
          {lifecycle.status?.last_error && (
            <span title={lifecycle.status.last_error} style={{
              display: 'inline-flex', alignItems: 'center', gap: 4,
              fontSize: 11, color: 'var(--danger)',
              maxWidth: 280,
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              <Icon.Warn width={11} height={11}/>
              {lifecycle.status.last_error}
            </span>
          )}
          <div style={{ flex: 1 }}/>
          <button
            onClick={() => setSimulateOpen(true)}
            style={{
              height: 28, padding: '0 10px',
              background: 'transparent',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 4,
              color: 'var(--fg)',
              fontSize: 13, cursor: 'pointer',
              display: 'inline-flex', alignItems: 'center', gap: 6,
            }}
          >Simulate</button>
          <button
            className="btn primary sm"
            onClick={() => setEditing({ __new: true })}
          >
            <Icon.Plus width={11} height={11}/>New rule
          </button>
        </div>

        {/* Body — scrollable */}
        <div style={{ flex: 1, overflow: 'auto' }}>
          <div style={{ padding: '12px 16px', display: 'flex', flexDirection: 'column', gap: 16 }}>

            {/* Status strip */}
            <StatusStrip
              lifecycle={lifecycle.status}
              status={proxyStatus}
              rulesCount={allRules.length}
              lastSimMicros={lastSimMicros}
            />

            {/* Topology */}
            {meshApps.length > 0 && (
              <div>
                <div style={{
                  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                  color: 'var(--fg-muted)', fontWeight: 500, marginBottom: 6,
                }}>Topology</div>
                <div style={{
                  border: '1px solid var(--hairline)',
                  background: 'var(--surface-0)',
                  borderRadius: 4,
                  overflow: 'hidden',
                }}>
                  {meshApps.map((app, i) => (
                    <div key={app.id} style={{
                      borderBottom: i === meshApps.length - 1 ? 'none' : undefined,
                    }}>
                      <TopologyRow app={app} health={proxyStatus} onReload={doReload} busy={reloadBusy}/>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Rules table */}
            <div>
              <div className="row" style={{
                marginBottom: 6, justifyContent: 'space-between', alignItems: 'center',
              }}>
                <div style={{
                  fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                  color: 'var(--fg-muted)', fontWeight: 500,
                }}>Rules</div>
                <div className="row" style={{ gap: 6, alignItems: 'center' }}>
                  {/* DB / Compiled view toggle — Compiled = what engine actually loaded */}
                  <div style={{
                    display: 'inline-flex',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 4,
                    overflow: 'hidden',
                    height: 24,
                  }}>
                    {(['db', 'compiled'] as const).map(v => (
                      <button
                        key={v}
                        onClick={() => setRulesView(v)}
                        title={v === 'db' ? 'Editable DB rules' : 'Compiled engine view (read-only)'}
                        style={{
                          padding: '0 10px', height: 22,
                          background: rulesView === v ? 'var(--surface-2)' : 'transparent',
                          color: rulesView === v ? 'var(--fg)' : 'var(--fg-muted)',
                          border: 0,
                          borderRight: v === 'db' ? '1px solid var(--hairline-strong)' : 0,
                          fontSize: 11, fontWeight: rulesView === v ? 500 : 400,
                          cursor: 'pointer',
                          fontFamily: 'inherit',
                        }}
                      >{v === 'db' ? 'DB' : 'Compiled'}</button>
                    ))}
                  </div>
                  <select
                    value={appFilter}
                    onChange={e => setAppFilter(e.target.value)}
                    disabled={rulesView === 'compiled'}
                    style={{
                      appearance: 'none', WebkitAppearance: 'none',
                      height: 24, padding: '0 22px 0 8px',
                      background: 'var(--surface-1)',
                      border: '1px solid var(--hairline-strong)',
                      borderRadius: 4,
                      color: rulesView === 'compiled' ? 'var(--fg-dim)' : 'var(--fg)',
                      fontSize: 12, cursor: rulesView === 'compiled' ? 'not-allowed' : 'pointer',
                      opacity: rulesView === 'compiled' ? 0.5 : 1,
                      colorScheme: 'dark',
                    }}
                  >
                    <option value="all">All applications</option>
                    {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
                  </select>
                </div>
              </div>

              <div style={{
                border: '1px solid var(--hairline)',
                background: 'var(--surface-0)',
                borderRadius: 4,
                overflow: 'hidden',
              }}>
                <table className="tbl" style={{ width: '100%', fontSize: 13 }}>
                  <thead>
                    <tr style={{ background: 'var(--surface-1)' }}>
                      <Th width={44}>Prio</Th>
                      <Th>Name</Th>
                      <Th>Pattern</Th>
                      <Th width={140}>Require</Th>
                      <Th width={120}>App</Th>
                      <Th width={50}>M2M</Th>
                      <Th width={56}>On</Th>
                    </tr>
                  </thead>
                  <tbody>
                    {rulesView === 'db' ? (
                      <>
                        {rules.length === 0 && (
                          <tr>
                            <td colSpan={7} style={{ padding: 28, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 13 }}>
                              No rules yet — click <strong>New rule</strong> to create one.
                            </td>
                          </tr>
                        )}
                        {rules.map(r => (
                          <RuleRow
                            key={r.id}
                            rule={r}
                            apps={apps}
                            active={editing && editing.id === r.id}
                            onClick={() => setEditing(r)}
                            onToggled={refreshRules}
                          />
                        ))}
                      </>
                    ) : (
                      <>
                        {compiled.rules == null && (
                          <tr><td colSpan={7} style={{ padding: 20, color: 'var(--fg-muted)', fontSize: 12 }}>Loading compiled rules…</td></tr>
                        )}
                        {compiled.rules?.length === 0 && (
                          <tr><td colSpan={7} style={{ padding: 28, textAlign: 'center', color: 'var(--fg-muted)', fontSize: 13 }}>
                            Engine has zero rules loaded. Press <strong>Reload</strong> on a topology row, or check the proxy log.
                          </td></tr>
                        )}
                        {compiled.rules?.map((r: any, i: number) => (
                          <CompiledRuleRow key={i} rule={r}/>
                        ))}
                      </>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>

        <CLIFooter command="shark proxy status"/>
      </div>

      {/* Right-side editor drawer */}
      {editing && (
        <RuleDrawer
          rule={editing.__new ? null : editing}
          apps={apps}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); refreshRules(); compiled.refresh(); }}
          onDeleted={() => { setEditing(null); refreshRules(); compiled.refresh(); }}
        />
      )}

      {/* Simulate drawer */}
      {simulateOpen && (
        <SimulateDrawer
          onClose={() => setSimulateOpen(false)}
          onResult={(micros) => setLastSimMicros(micros)}
        />
      )}
    </div>
  );
}

// ─── Table column header ──────────────────────────────────────────────────────

function Th({ children, width }: { children?: any; width?: number }) {
  return (
    <th style={{
      width,
      padding: '7px 12px',
      textAlign: 'left',
      fontSize: 11,
      textTransform: 'uppercase',
      letterSpacing: '0.06em',
      color: 'var(--fg-muted)',
      fontWeight: 500,
      position: 'sticky', top: 0, zIndex: 1,
      background: 'var(--surface-1)',
      borderBottom: '1px solid var(--hairline)',
    }}>{children}</th>
  );
}

// ─── Rule row ─────────────────────────────────────────────────────────────────

function RuleRow({ rule, apps, active, onClick, onToggled }) {
  const toast = useToast();
  const [busy, setBusy] = React.useState(false);
  const appName = apps.find(a => a.id === rule.app_id)?.name;
  const dim = !rule.enabled;

  const toggle = async () => {
    setBusy(true);
    try {
      await API.patch('/admin/proxy/rules/db/' + rule.id, { enabled: !rule.enabled });
      onToggled();
    } catch (e: any) {
      toast.error(e.message || 'Toggle failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <tr
      onClick={onClick}
      className={active ? 'active' : ''}
      style={{
        cursor: 'pointer',
        opacity: dim ? 0.55 : 1,
        borderTop: '1px solid var(--hairline)',
      }}
    >
      <td className="mono" style={{ padding: '7px 12px', color: 'var(--fg-muted)', fontSize: 12 }}>{rule.priority ?? 0}</td>
      <td style={{ padding: '7px 12px', fontWeight: 500, maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{rule.name}</td>
      <td style={{ padding: '7px 12px' }}>
        <div className="mono" style={{ fontSize: 12 }}>{rule.pattern}</div>
        {rule.methods?.length > 0 && (
          <div className="faint mono" style={{ fontSize: 10, lineHeight: 1.4 }}>
            {rule.methods.join(' ')}
          </div>
        )}
      </td>
      <td style={{ padding: '7px 12px' }}>
        <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
          {(rule.require || rule.allow) && (
            <span className="chip" style={{ height: 17, fontSize: 10 }}>
              {rule.require || rule.allow}
            </span>
          )}
          {rule.tier_match && <span className="chip" style={{ height: 17, fontSize: 10 }}>tier:{rule.tier_match}</span>}
        </div>
      </td>
      <td className="faint" style={{ padding: '7px 12px', fontSize: 11 }}>
        {appName || <span style={{ opacity: 0.5 }}>global</span>}
      </td>
      <td style={{ padding: '7px 12px' }}>
        {rule.m2m && <span className="chip" style={{ height: 17, fontSize: 10 }}>m2m</span>}
      </td>
      <td style={{ padding: '7px 12px' }} onClick={e => e.stopPropagation()}>
        <InlineToggle on={!!rule.enabled} busy={busy} onChange={toggle} title={rule.enabled ? 'Disable' : 'Enable'}/>
      </td>
    </tr>
  );
}

// ─── Compiled engine rule row (read-only) ─────────────────────────────────────
// Mirrors RuleRow shape so view-toggle doesn't reflow the table. Compiled rules
// expose only what the engine retained: path, methods, require, scopes. No id,
// no app_id, no enable toggle — by definition, anything in this list is live.

function CompiledRuleRow({ rule }) {
  return (
    <tr style={{ borderTop: '1px solid var(--hairline)', cursor: 'default' }}>
      <td className="mono" style={{ padding: '7px 12px', color: 'var(--fg-dim)', fontSize: 12 }}>—</td>
      <td className="faint" style={{ padding: '7px 12px', fontStyle: 'italic', maxWidth: 180, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        compiled
      </td>
      <td style={{ padding: '7px 12px' }}>
        <div className="mono" style={{ fontSize: 12 }}>{rule.path || rule.pattern}</div>
        {rule.methods?.length > 0 && (
          <div className="faint mono" style={{ fontSize: 10, lineHeight: 1.4 }}>
            {rule.methods.join(' ')}
          </div>
        )}
      </td>
      <td style={{ padding: '7px 12px' }}>
        <div className="row" style={{ gap: 4, flexWrap: 'wrap' }}>
          {rule.require && <span className="chip" style={{ height: 17, fontSize: 10 }}>{rule.require}</span>}
          {(rule.scopes || []).map((s: string) => (
            <span key={s} className="chip" style={{ height: 17, fontSize: 10 }}>scope:{s}</span>
          ))}
        </div>
      </td>
      <td className="faint" style={{ padding: '7px 12px', fontSize: 11, opacity: 0.5 }}>—</td>
      <td style={{ padding: '7px 12px' }}/>
      <td style={{ padding: '7px 12px', color: 'var(--success)', fontSize: 11 }}>live</td>
    </tr>
  );
}

// ─── Right-side rule editor drawer ────────────────────────────────────────────

const BLANK = {
  name: '', pattern: '', methods: '', require: 'authenticated', allow: '',
  scopes: '', enabled: true, priority: 0, tier_match: '', m2m: false, app_id: '',
  protected_url: '', public_domain: '',
};

function RuleDrawer({ rule, apps, onClose, onSaved, onDeleted }) {
  const toast = useToast();
  const isNew = !rule;
  const [f, setF] = React.useState<any>(() => rule ? {
    ...rule,
    methods: (rule.methods || []).join(', '),
    scopes: (rule.scopes || []).join(', '),
    public_domain: rule.public_domain || '',
    protected_url: rule.protected_url || '',
  } : { ...BLANK });
  const [errs, setErrs] = React.useState<any>({});
  const [saving, setSaving] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [apiErr, setApiErr] = React.useState('');

  const set = (k, v) => setF(p => ({ ...p, [k]: v }));

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const submit = async () => {
    const e = validateRule(f);
    if (Object.keys(e).length > 0) { setErrs(e); return; }
    setSaving(true); setApiErr('');
    try {
      const body: any = {
        name: f.name, pattern: f.pattern, require: f.require, allow: f.allow,
        enabled: !!f.enabled, m2m: !!f.m2m, app_id: f.app_id || undefined,
        tier_match: f.tier_match || undefined,
        priority: Number(f.priority) || 0,
        methods: f.methods ? f.methods.split(',').map(s => s.trim()).filter(Boolean) : [],
        scopes: f.scopes ? f.scopes.split(',').map(s => s.trim()).filter(Boolean) : [],
      };
      let res;
      if (isNew) res = await API.post('/admin/proxy/rules/db', body);
      else res = await API.patch('/admin/proxy/rules/db/' + rule.id, body);
      if (res?.engine_refresh_error) {
        toast.warn('Saved — engine refresh: ' + res.engine_refresh_error);
      } else {
        toast.success(isNew ? 'Rule created' : 'Rule updated');
      }
      onSaved();
    } catch (e: any) {
      setApiErr(e.message || 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const del = async () => {
    if (!rule) return;
    if (!window.confirm(`Delete rule "${rule.name}"?`)) return;
    setDeleting(true);
    try {
      await API.del('/admin/proxy/rules/db/' + rule.id);
      toast.success('Rule deleted');
      onDeleted();
    } catch (e: any) {
      toast.error(e.message || 'Delete failed');
      setDeleting(false);
    }
  };

  const busy = saving || deleting;

  return (
    <>
      <div onClick={onClose} style={{
        position: 'fixed', inset: 0, zIndex: 40,
        background: 'rgba(0,0,0,0.35)',
      }}/>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          position: 'fixed', top: 0, right: 0, bottom: 0,
          width: 420, maxWidth: '100vw',
          zIndex: 41,
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          display: 'flex', flexDirection: 'column',
          animation: 'slideIn 140ms ease-out',
        }}
      >
        {/* Header */}
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 10 }}>
            <button className="btn ghost sm" onClick={onClose} disabled={busy}>
              <Icon.X width={12} height={12}/>Close
            </button>
            <span className="faint mono" style={{ fontSize: 11 }}>
              {isNew ? 'POST /admin/proxy/rules/db' : 'PATCH /admin/proxy/rules/db/' + rule.id.slice(0, 8)}
            </span>
          </div>
          <div style={{
            fontSize: 18, fontWeight: 500, letterSpacing: '-0.01em',
            fontFamily: 'var(--font-display)', lineHeight: 1.2,
          }}>
            {isNew ? 'New rule' : 'Edit rule'}
          </div>
          {!isNew && (
            <div className="faint mono" style={{ fontSize: 11, marginTop: 4 }}>{rule.id}</div>
          )}
        </div>

        {/* Body */}
        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>

            <FieldGroup label="App" hint="Which application this rule applies to. Empty = global.">
              <select
                value={f.app_id}
                onChange={e => set('app_id', e.target.value)}
                style={{
                  appearance: 'none', WebkitAppearance: 'none',
                  width: '100%', height: 28, padding: '0 8px',
                  background: 'var(--surface-1)',
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4,
                  fontSize: 13, color: 'var(--fg)',
                  colorScheme: 'dark',
                }}
              >
                <option value="">Global (any app)</option>
                {apps.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
              </select>
            </FieldGroup>

            <FieldGroup label="Name" err={errs.name}>
              <Input value={f.name} placeholder="block-unauth-writes"
                err={!!errs.name}
                onChange={(v) => { set('name', v); setErrs(p => ({ ...p, name: undefined })); }}/>
            </FieldGroup>

            <FieldGroup label="Public domain" hint="Where the proxy listens (read-only when set on the app).">
              <Input value={f.public_domain} placeholder="api.example.com"
                onChange={(v) => set('public_domain', v)}/>
            </FieldGroup>

            <FieldGroup label="Protected URL" hint="Upstream the proxy forwards to.">
              <Input value={f.protected_url} placeholder="http://localhost:8080"
                onChange={(v) => set('protected_url', v)}/>
            </FieldGroup>

            <FieldGroup label="Path pattern" err={errs.pattern}>
              <Input value={f.pattern} placeholder="/api/*"
                err={!!errs.pattern} mono
                onChange={(v) => { set('pattern', v); setErrs(p => ({ ...p, pattern: undefined })); }}/>
            </FieldGroup>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <FieldGroup label="Methods" hint="comma-sep · empty = any">
                <Input value={f.methods} placeholder="GET, POST"
                  onChange={(v) => set('methods', v)}/>
              </FieldGroup>
              <FieldGroup label="Priority">
                <Input type="number" value={String(f.priority)} placeholder="0"
                  onChange={(v) => set('priority', v)}/>
              </FieldGroup>
            </div>

            <FieldGroup label="Require" err={errs.require}
              hint="anonymous · authenticated · agent · role:X · global_role:X · tier:X · scope:X · permission:X:Y">
              <Input value={f.require} placeholder="authenticated"
                err={!!errs.require}
                onChange={(v) => { set('require', v); set('allow', ''); setErrs(p => ({ ...p, require: undefined })); }}/>
            </FieldGroup>

            <FieldGroup label="Allow override" err={errs.allow}>
              <div className="row" style={{ gap: 6 }}>
                <button
                  type="button"
                  onClick={() => { set('allow', f.allow === 'anonymous' ? '' : 'anonymous'); set('require', ''); }}
                  style={{
                    height: 26, padding: '0 10px',
                    background: f.allow === 'anonymous' ? 'var(--fg)' : 'transparent',
                    color: f.allow === 'anonymous' ? 'var(--bg)' : 'var(--fg)',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 4, fontSize: 12, cursor: 'pointer',
                  }}
                >anonymous</button>
                {f.allow && (
                  <button
                    type="button"
                    onClick={() => set('allow', '')}
                    style={{
                      height: 26, padding: '0 10px',
                      background: 'transparent', color: 'var(--fg-muted)',
                      border: '1px solid var(--hairline-strong)',
                      borderRadius: 4, fontSize: 12, cursor: 'pointer',
                    }}
                  >Clear</button>
                )}
              </div>
            </FieldGroup>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
              <FieldGroup label="Tier match">
                <Input value={f.tier_match} placeholder="free · pro"
                  onChange={(v) => set('tier_match', v)}/>
              </FieldGroup>
              <FieldGroup label="Scopes" hint="comma-sep">
                <Input value={f.scopes} placeholder="read:data"
                  onChange={(v) => set('scopes', v)}/>
              </FieldGroup>
            </div>

            <div className="row" style={{ gap: 18, marginTop: 4 }}>
              <Checkbox
                checked={!!f.enabled}
                onChange={(v) => set('enabled', v)}
                label="Enabled"
              />
              <Checkbox
                checked={!!f.m2m}
                onChange={(v) => set('m2m', v)}
                label="M2M only"
              />
            </div>

            {apiErr && (
              <div style={{
                padding: 8,
                border: '1px solid var(--danger)',
                borderRadius: 4,
                fontSize: 12, color: 'var(--danger)', lineHeight: 1.5,
              }}>{apiErr}</div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div style={{
          padding: 12, borderTop: '1px solid var(--hairline)',
          display: 'flex', gap: 8,
          background: 'var(--surface-0)',
        }}>
          {!isNew && (
            <button className="btn sm danger" onClick={del} disabled={busy}>
              {deleting ? 'Deleting…' : 'Delete'}
            </button>
          )}
          <div style={{ flex: 1 }}/>
          <button className="btn ghost sm" onClick={onClose} disabled={busy}>Cancel</button>
          <button className="btn primary sm" onClick={submit} disabled={busy}>
            {saving ? 'Saving…' : (isNew ? 'Create rule' : 'Save')}
          </button>
        </div>
      </div>
    </>
  );
}

// ─── Simulate drawer ──────────────────────────────────────────────────────────

function SimulateDrawer({ onClose, onResult }) {
  const [method, setMethod] = React.useState('GET');
  const [path, setPath] = React.useState('/api/users');
  const [identityJson, setIdentityJson] = React.useState('{\n  "user_id": "",\n  "roles": []\n}');
  const [running, setRunning] = React.useState(false);
  const [result, setResult] = React.useState<any>(null);
  const [err, setErr] = React.useState('');

  React.useEffect(() => {
    const onKey = (e) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  const run = async () => {
    setRunning(true); setErr(''); setResult(null);
    let identity: any = {};
    try { identity = identityJson.trim() ? JSON.parse(identityJson) : {}; }
    catch (e: any) { setErr('Identity JSON: ' + e.message); setRunning(false); return; }
    const t0 = performance.now();
    try {
      const r = await API.post('/admin/proxy/simulate', { method, path, identity });
      const micros = Math.max(1, Math.round((performance.now() - t0) * 1000));
      setResult(r?.data || r);
      onResult(micros);
    } catch (e: any) {
      setErr(e.message || 'Simulate failed');
    } finally {
      setRunning(false);
    }
  };

  const decision = result?.decision;
  const decisionColor = decision === 'allow' ? 'var(--success)' : decision === 'deny' ? 'var(--danger)' : 'var(--fg-dim)';

  return (
    <>
      <div onClick={onClose} style={{
        position: 'fixed', inset: 0, zIndex: 40,
        background: 'rgba(0,0,0,0.35)',
      }}/>
      <div
        onClick={e => e.stopPropagation()}
        style={{
          position: 'fixed', top: 0, right: 0, bottom: 0,
          width: 440, maxWidth: '100vw',
          zIndex: 41,
          borderLeft: '1px solid var(--hairline)',
          background: 'var(--surface-0)',
          display: 'flex', flexDirection: 'column',
          animation: 'slideIn 140ms ease-out',
        }}
      >
        <div style={{ padding: 16, borderBottom: '1px solid var(--hairline)' }}>
          <div className="row" style={{ justifyContent: 'space-between', marginBottom: 10 }}>
            <button className="btn ghost sm" onClick={onClose}>
              <Icon.X width={12} height={12}/>Close
            </button>
            <span className="faint mono" style={{ fontSize: 11 }}>POST /admin/proxy/simulate</span>
          </div>
          <div style={{
            fontSize: 18, fontWeight: 500, letterSpacing: '-0.01em',
            fontFamily: 'var(--font-display)', lineHeight: 1.2,
          }}>Simulate request</div>
          <div className="faint" style={{ fontSize: 12, marginTop: 4 }}>
            Dry-run a request through the rule engine.
          </div>
        </div>

        <div style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            <div style={{ display: 'grid', gridTemplateColumns: '90px 1fr', gap: 8 }}>
              <FieldGroup label="Method">
                <select
                  value={method}
                  onChange={e => setMethod(e.target.value)}
                  style={{
                    appearance: 'none', WebkitAppearance: 'none',
                    width: '100%', height: 28, padding: '0 8px',
                    background: 'var(--surface-1)',
                    border: '1px solid var(--hairline-strong)',
                    borderRadius: 4,
                    fontSize: 13, color: 'var(--fg)',
                    colorScheme: 'dark',
                  }}
                >
                  {['GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS'].map(m => <option key={m} value={m}>{m}</option>)}
                </select>
              </FieldGroup>
              <FieldGroup label="Path">
                <Input value={path} mono placeholder="/api/users"
                  onChange={(v) => setPath(v)}/>
              </FieldGroup>
            </div>

            <FieldGroup label="Identity (JSON)" hint="user_id, user_email, roles, agent_id, agent_name, auth_method, scopes, tier">
              <textarea
                value={identityJson}
                onChange={e => setIdentityJson(e.target.value)}
                spellCheck={false}
                style={{
                  width: '100%', minHeight: 110,
                  background: 'var(--surface-1)',
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4, padding: 8,
                  fontFamily: 'var(--font-mono)', fontSize: 12, lineHeight: 1.5,
                  color: 'var(--fg)', resize: 'vertical',
                }}
              />
            </FieldGroup>

            <button
              onClick={run}
              disabled={running}
              className="btn primary sm"
              style={{ alignSelf: 'flex-start' }}
            >
              {running ? 'Running…' : 'Run simulation'}
            </button>

            {err && (
              <div style={{
                padding: 8, border: '1px solid var(--danger)',
                borderRadius: 4, fontSize: 12, color: 'var(--danger)', lineHeight: 1.5,
              }}>{err}</div>
            )}

            {result && (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginTop: 4 }}>
                <div style={{
                  padding: 10,
                  border: '1px solid var(--hairline-strong)',
                  borderRadius: 4,
                  background: 'var(--surface-1)',
                }}>
                  <div className="row" style={{ gap: 8, alignItems: 'center', marginBottom: 4 }}>
                    <span style={{
                      width: 7, height: 7, borderRadius: '50%',
                      background: decisionColor,
                    }}/>
                    <span className="mono" style={{ fontSize: 13, fontWeight: 600 }}>
                      {decision || '—'}
                    </span>
                  </div>
                  {result.reason && (
                    <div className="faint" style={{ fontSize: 12, lineHeight: 1.5 }}>{result.reason}</div>
                  )}
                </div>

                <div>
                  <div style={{
                    fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                    color: 'var(--fg-muted)', fontWeight: 500, marginBottom: 4,
                  }}>Matched rule</div>
                  {result.matched_rule ? (
                    <div style={{
                      padding: 10, border: '1px solid var(--hairline-strong)',
                      borderRadius: 4, background: 'var(--surface-1)',
                    }}>
                      <div style={{ fontSize: 13, fontWeight: 500 }}>{result.matched_rule.name}</div>
                      <div className="mono faint" style={{ fontSize: 11, marginTop: 2 }}>
                        {result.matched_rule.pattern}
                      </div>
                    </div>
                  ) : (
                    <span className="faint" style={{ fontSize: 12 }}>No rule matched (default policy applied).</span>
                  )}
                </div>

                {result.injected_headers && Object.keys(result.injected_headers).length > 0 && (
                  <div>
                    <div style={{
                      fontSize: 10, textTransform: 'uppercase', letterSpacing: '0.08em',
                      color: 'var(--fg-muted)', fontWeight: 500, marginBottom: 4,
                    }}>Injected headers</div>
                    <div style={{
                      border: '1px solid var(--hairline-strong)',
                      borderRadius: 4, overflow: 'hidden',
                    }}>
                      {Object.entries(result.injected_headers).map(([k, v]) => (
                        <div key={k} className="row" style={{
                          gap: 8, padding: '6px 10px',
                          borderBottom: '1px solid var(--hairline)',
                          fontSize: 12,
                        }}>
                          <span className="mono" style={{ color: 'var(--fg-muted)', minWidth: 140 }}>{k}</span>
                          <span className="mono" style={{ color: 'var(--fg)', wordBreak: 'break-all' }}>{String(v)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}

// ─── Form primitives ──────────────────────────────────────────────────────────

function FieldGroup({ label, children, hint, err }: { label: string; children: any; hint?: string; err?: string }) {
  return (
    <div>
      <div style={{
        fontSize: 11, textTransform: 'uppercase', letterSpacing: '0.08em',
        color: 'var(--fg-muted)', marginBottom: 4, lineHeight: 1.5, fontWeight: 500,
      }}>{label}</div>
      {children}
      {err && <div style={{ fontSize: 11, color: 'var(--danger)', marginTop: 3, lineHeight: 1.5 }}>{err}</div>}
      {hint && !err && <div className="faint" style={{ fontSize: 11, lineHeight: 1.5, marginTop: 3 }}>{hint}</div>}
    </div>
  );
}

function Input({ value, onChange, placeholder, type = 'text', err = false, mono = false }) {
  return (
    <input
      type={type}
      value={value ?? ''}
      onChange={e => onChange(e.target.value)}
      placeholder={placeholder}
      className={mono ? 'mono' : ''}
      style={{
        width: '100%', height: 28, padding: '0 8px',
        background: 'var(--surface-1)',
        border: '1px solid ' + (err ? 'var(--danger)' : 'var(--hairline-strong)'),
        borderRadius: 4,
        fontSize: 13, color: 'var(--fg)',
        fontFamily: mono ? 'var(--font-mono)' : undefined,
      }}
    />
  );
}

function Checkbox({ checked, onChange, label }) {
  return (
    <label className="row" style={{ gap: 8, cursor: 'pointer', fontSize: 13 }}>
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
      />
      <span style={{ color: 'var(--fg)' }}>{label}</span>
    </label>
  );
}
