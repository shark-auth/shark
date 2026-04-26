// @ts-nocheck
// TODO Wave 1.6 backend: add /api/v1/audit-logs/chains aggregator endpoint.
// Current implementation queries GET /api/v1/audit-logs?action=oauth.token.exchanged&limit=50
// and GET /api/v1/audit-logs?action=vault.token.retrieved&limit=50, then groups by act_chain
// root subject client-side. A dedicated /audit-logs/chains endpoint would return pre-aggregated
// chain objects with all hops, reducing round-trips and enabling server-side pagination.

import React from 'react'
import { Icon, CopyField } from './shared'
import { useAPI } from './api'
import { usePageActions } from './useKeyboardShortcuts'

// ─── styles reused from audit.tsx / users.tsx gold standard ───────────────────
const thStyle: React.CSSProperties = {
  textAlign: 'left',
  padding: '7px 14px',
  fontSize: 10,
  fontWeight: 500,
  color: 'var(--fg-dim)',
  borderBottom: '1px solid var(--hairline)',
  background: 'var(--surface-1)',
  position: 'sticky',
  top: 0,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.05em',
};
const tdStyle: React.CSSProperties = {
  padding: '7px 14px',
  borderBottom: '1px solid var(--hairline)',
  verticalAlign: 'top',
  fontSize: 12,
};
const sectionLabel: React.CSSProperties = {
  fontSize: 10,
  textTransform: 'uppercase' as const,
  letterSpacing: '0.08em',
  color: 'var(--fg-dim)',
  fontWeight: 500,
  margin: '0 0 8px',
};

// ─── helpers ──────────────────────────────────────────────────────────────────

function fmtTime(iso: string) {
  try {
    return new Date(iso).toLocaleString(undefined, {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
  } catch {
    return iso;
  }
}

function jktShort(jkt?: string) {
  if (!jkt) return null;
  return jkt.slice(0, 4);
}

/** Parse a raw audit-log entry into a normalised shape. */
function normalizeEntry(e: any) {
  const actChain: Array<{ sub: string; jkt?: string; label?: string }> =
    Array.isArray(e.act_chain) && e.act_chain.length > 0
      ? e.act_chain
      : e.oauth?.act
        ? [{ sub: e.oauth.act.sub || '', label: e.oauth.act.email || e.oauth.act.sub || 'user' }]
        : [];

  const rootSub = actChain.length > 0 ? (actChain[0].label || actChain[0].sub) : (e.actor_email || e.actor_id || 'unknown');

  return {
    id: e.id || '',
    created_at: e.created_at || '',
    action: e.action || e.event_type || '',
    actor: e.actor_email || e.actor_id || '',
    actor_type: e.actor_type || 'system',
    target: [e.target_type, e.target_id].filter(Boolean).join('_') || '',
    actChain,
    rootSub,
    _raw: e,
  };
}

type NormEntry = ReturnType<typeof normalizeEntry>;

interface Chain {
  rootSub: string;
  latestAt: string;
  events: NormEntry[];
  segments: Array<{ sub: string; jkt?: string; label?: string; isUser: boolean }>;
  lastAction: string;
  lastTarget: string;
}

/** Group entries into chains by root subject. */
function buildChains(entries: NormEntry[]): Chain[] {
  const map = new Map<string, NormEntry[]>();
  for (const e of entries) {
    const key = e.rootSub;
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(e);
  }

  const chains: Chain[] = [];
  map.forEach((evts, rootSub) => {
    // Sort events newest-first for display
    const sorted = [...evts].sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    const newest = sorted[0];

    // Build unique ordered segment list from all events' act_chains
    const segMap = new Map<string, { sub: string; jkt?: string; label?: string; isUser: boolean }>();
    for (const ev of evts) {
      ev.actChain.forEach((seg, i) => {
        const k = seg.sub || seg.label || String(i);
        if (!segMap.has(k)) segMap.set(k, { ...seg, isUser: i === 0 });
      });
      // Terminal actor (the agent that fired the event)
      if (ev.actor && !segMap.has(ev.actor)) {
        segMap.set(ev.actor, { sub: ev.actor, label: ev.actor, isUser: false });
      }
    }

    chains.push({
      rootSub,
      latestAt: newest.created_at,
      events: sorted,
      segments: Array.from(segMap.values()),
      lastAction: newest.action,
      lastTarget: newest.target,
    });
  });

  // Sort chains newest-first
  return chains.sort((a, b) =>
    new Date(b.latestAt).getTime() - new Date(a.latestAt).getTime()
  );
}

// ─── breadcrumb renderer ──────────────────────────────────────────────────────

function ChainBreadcrumb({
  segments,
  onAgentClick,
}: {
  segments: Chain['segments'];
  onAgentClick: (sub: string) => void;
}) {
  const chipBase: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 4,
    padding: '2px 7px',
    height: 20,
    fontSize: 11,
    border: '1px solid var(--hairline-strong)',
    borderRadius: 3,
    background: 'var(--surface-2)',
    color: 'var(--fg)',
    fontFamily: 'var(--font-mono)',
    whiteSpace: 'nowrap' as const,
  };

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 5 }}>
      {segments.map((seg, i) => {
        const label = seg.label || seg.sub;
        const jkt = jktShort(seg.jkt);
        const isClickable = !seg.isUser && !!seg.sub;
        return (
          <React.Fragment key={i}>
            {i > 0 && (
              <span style={{ color: 'var(--fg-dim)', fontSize: 11, fontFamily: 'var(--font-mono)' }}>→</span>
            )}
            {isClickable ? (
              <button
                onClick={() => onAgentClick(seg.sub)}
                style={{
                  ...chipBase,
                  cursor: 'pointer',
                  background: 'var(--surface-1)',
                }}
                title={seg.sub}
              >
                {label}
                {jkt && (
                  <span style={{ color: 'var(--fg-dim)', fontSize: 9.5 }}>
                    {' '}jkt:{jkt}
                  </span>
                )}
              </button>
            ) : (
              <span style={chipBase} title={seg.sub}>
                {label}
                {jkt && (
                  <span style={{ color: 'var(--fg-dim)', fontSize: 9.5 }}>
                    {' '}jkt:{jkt}
                  </span>
                )}
              </span>
            )}
          </React.Fragment>
        );
      })}
    </div>
  );
}

// ─── chain-detail drawer ──────────────────────────────────────────────────────

function ChainDrawer({
  chain,
  onClose,
  onAuditClick,
  onAgentClick,
}: {
  chain: Chain;
  onClose: () => void;
  onAuditClick: (id: string) => void;
  onAgentClick: (sub: string) => void;
}) {
  const [expandedTokens, setExpandedTokens] = React.useState<Set<number>>(new Set());
  const toggleToken = (i: number) =>
    setExpandedTokens(prev => {
      const next = new Set(prev);
      next.has(i) ? next.delete(i) : next.add(i);
      return next;
    });

  // ASCII-art tree
  const treeLines: string[] = chain.segments.map((seg, i) => {
    const indent = '  '.repeat(i);
    const prefix = i === 0 ? '' : i === chain.segments.length - 1 ? indent + '└─ ' : indent + '├─ ';
    const label = seg.label || seg.sub;
    const jkt = jktShort(seg.jkt);
    return `${prefix}${label}${jkt ? ` (jkt:${jkt}…)` : ''}`;
  });

  // jkt continuity check: each hop's jkt should match the previous sub's jkt
  // (simplified check: flag any hop where jkt is present but differs from prior)
  const jktChecks: Array<{ ok: boolean; reason: string }> = chain.segments.map((seg, i) => {
    if (i === 0) return { ok: true, reason: 'origin' };
    const prev = chain.segments[i - 1];
    if (!seg.jkt) return { ok: true, reason: 'no jkt claim' };
    if (!prev.jkt) return { ok: true, reason: 'prior hop has no jkt to compare' };
    const ok = seg.jkt === prev.jkt;
    return { ok, reason: ok ? 'jkt matches prior hop' : `jkt drift: expected ${prev.jkt?.slice(0,8)} got ${seg.jkt?.slice(0,8)}` };
  });

  return (
    <aside style={{
      position: 'fixed',
      top: 0, right: 0, bottom: 0,
      width: 440,
      borderLeft: '1px solid var(--hairline)',
      background: 'var(--surface-0)',
      display: 'flex',
      flexDirection: 'column',
      zIndex: 200,
      overflow: 'hidden',
    }}>
      {/* Header */}
      <div style={{
        padding: '12px 16px',
        borderBottom: '1px solid var(--hairline)',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
      }}>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', flex: 1, textTransform: 'uppercase' as const, letterSpacing: '0.08em' }}>
          Chain detail
        </span>
        <button className="btn ghost icon sm" onClick={onClose}>
          <Icon.X width={11} height={11}/>
        </button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '14px 16px' }}>
        {/* Timestamp */}
        <div style={{ marginBottom: 16 }}>
          <p style={sectionLabel}>Latest event</p>
          <span className="mono" style={{ fontSize: 11.5 }}>{fmtTime(chain.latestAt)}</span>
        </div>

        {/* ASCII tree */}
        <div style={{ marginBottom: 16 }}>
          <p style={sectionLabel}>Chain tree</p>
          <pre style={{
            margin: 0,
            padding: '10px 12px',
            background: 'var(--surface-1)',
            border: '1px solid var(--hairline)',
            borderRadius: 3,
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            lineHeight: 1.6,
            color: 'var(--fg)',
            overflowX: 'auto',
          }}>{treeLines.join('\n')}</pre>
        </div>

        {/* jkt verification per hop */}
        <div style={{ marginBottom: 16 }}>
          <p style={sectionLabel}>Cnf.jkt verification</p>
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
            {chain.segments.map((seg, i) => {
              const check = jktChecks[i];
              return (
                <div key={i} style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '6px 10px',
                  borderBottom: i < chain.segments.length - 1 ? '1px solid var(--hairline)' : 'none',
                  fontSize: 11,
                }}>
                  <span style={{
                    color: check.ok ? 'var(--success, #22c55e)' : 'var(--danger)',
                    fontWeight: 600,
                    fontSize: 12,
                    width: 14,
                    flexShrink: 0,
                  }}>
                    {check.ok ? '✓' : '✗'}
                  </span>
                  <span className="mono" style={{ flex: 1 }}>{seg.label || seg.sub}</span>
                  <span style={{ color: 'var(--fg-dim)', fontSize: 10 }}>{check.reason}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* JWT claims per token (collapsible) */}
        <div style={{ marginBottom: 16 }}>
          <p style={sectionLabel}>JWT claims per hop</p>
          {chain.segments.map((seg, i) => {
            const claims = {
              sub: seg.sub,
              ...(seg.jkt ? { 'cnf.jkt': seg.jkt } : {}),
              label: seg.label,
            };
            const open = expandedTokens.has(i);
            return (
              <div key={i} style={{ marginBottom: 4, border: '1px solid var(--hairline)', borderRadius: 3 }}>
                <button
                  onClick={() => toggleToken(i)}
                  style={{
                    width: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    padding: '7px 10px',
                    background: open ? 'var(--surface-2)' : 'var(--surface-1)',
                    fontSize: 11,
                    cursor: 'pointer',
                    borderRadius: 3,
                    textAlign: 'left' as const,
                  }}
                >
                  <Icon.ChevronDown width={10} height={10} style={{
                    transform: open ? 'rotate(0deg)' : 'rotate(-90deg)',
                    transition: 'transform 120ms',
                    opacity: 0.5,
                  }}/>
                  <span className="mono" style={{ flex: 1 }}>{seg.label || seg.sub}</span>
                  {seg.jkt && (
                    <span style={{ color: 'var(--fg-dim)', fontSize: 10, fontFamily: 'var(--font-mono)' }}>
                      jkt:{seg.jkt.slice(0, 8)}…
                    </span>
                  )}
                </button>
                {open && (
                  <pre style={{
                    margin: 0,
                    padding: '8px 10px',
                    background: 'var(--surface-0)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10.5,
                    lineHeight: 1.55,
                    color: 'var(--fg)',
                    borderTop: '1px solid var(--hairline)',
                    borderRadius: '0 0 3px 3px',
                    overflowX: 'auto',
                  }}>{JSON.stringify(claims, null, 2)}</pre>
                )}
              </div>
            );
          })}
        </div>

        {/* All audit events */}
        <div style={{ marginBottom: 16 }}>
          <p style={sectionLabel}>Audit events ({chain.events.length})</p>
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)', overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 11 }}>
              <thead>
                <tr>
                  <th style={thStyle}>Time</th>
                  <th style={thStyle}>Action</th>
                  <th style={thStyle}>Event ID</th>
                </tr>
              </thead>
              <tbody>
                {chain.events.map(ev => (
                  <tr key={ev.id}
                    style={{ cursor: 'pointer' }}
                    onClick={() => onAuditClick(ev.id)}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface-2)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td style={tdStyle}>
                      <span className="mono faint" style={{ fontSize: 10 }}>
                        {fmtTime(ev.created_at)}
                      </span>
                    </td>
                    <td style={tdStyle}>
                      <span className="mono" style={{ fontSize: 10.5 }}>{ev.action}</span>
                    </td>
                    <td style={tdStyle}>
                      <CopyField value={ev.id} truncate={14}/>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div style={{ padding: '10px 16px', borderTop: '1px solid var(--hairline)' }}>
        <button className="btn ghost" style={{ width: '100%', fontSize: 11 }}
          onClick={() => {
            const json = JSON.stringify(chain.events.map(e => e._raw), null, 2);
            const blob = new Blob([json], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `chain-${chain.rootSub.slice(0, 16)}.json`;
            a.click();
            URL.revokeObjectURL(url);
          }}
        >
          <Icon.Copy width={11} height={11}/> Export chain JSON
        </button>
      </div>
    </aside>
  );
}

// ─── filter controls ──────────────────────────────────────────────────────────

type TimeRange = '1h' | '24h' | '7d' | 'all';

function Seg({ value, onChange, opts }: {
  value: any;
  onChange: (v: any) => void;
  opts: Array<[any, string]>;
}) {
  return (
    <div style={{
      display: 'inline-flex',
      border: '1px solid var(--hairline-strong)',
      borderRadius: 3,
      overflow: 'hidden',
      height: 24,
    }}>
      {opts.map(([v, label]) => (
        <button
          key={String(v)}
          onClick={() => onChange(v)}
          style={{
            padding: '0 9px',
            height: 24,
            fontSize: 11,
            background: value === v ? 'var(--surface-3)' : 'var(--surface-2)',
            color: value === v ? 'var(--fg)' : 'var(--fg-muted)',
            borderRight: '1px solid var(--hairline)',
            cursor: 'pointer',
          }}
        >{label}</button>
      ))}
    </div>
  );
}

// ─── main page ────────────────────────────────────────────────────────────────

const PAGE_SIZE = 20;

export function DelegationChains({ setPage }: { setPage?: (p: string, extra?: any) => void }) {
  const [timeRange, setTimeRange] = React.useState<TimeRange>('24h');
  const [actorFilter, setActorFilter] = React.useState('');
  const [statusFilter, setStatusFilter] = React.useState('all');
  const [selected, setSelected] = React.useState<Chain | null>(null);
  const [page, setPageNum] = React.useState(0);

  // Build query params for token-exchange events
  const exchangeParams = React.useMemo(() => {
    const p = new URLSearchParams();
    p.set('action', 'oauth.token.exchanged');
    p.set('limit', '50');
    if (timeRange !== 'all') {
      const hours: Record<string, number> = { '1h': 1, '24h': 24, '7d': 168 };
      const h = hours[timeRange];
      if (h) p.set('since', new Date(Date.now() - h * 3600000).toISOString());
    }
    return '/audit-logs?' + p.toString();
  }, [timeRange]);

  const retrieveParams = React.useMemo(() => {
    const p = new URLSearchParams();
    p.set('action', 'vault.token.retrieved');
    p.set('limit', '50');
    if (timeRange !== 'all') {
      const hours: Record<string, number> = { '1h': 1, '24h': 24, '7d': 168 };
      const h = hours[timeRange];
      if (h) p.set('since', new Date(Date.now() - h * 3600000).toISOString());
    }
    return '/audit-logs?' + p.toString();
  }, [timeRange]);

  const { data: exchangeData, loading: loadingExchange, refresh: refreshExchange } = useAPI(exchangeParams);
  const { data: retrieveData, loading: loadingRetrieve, refresh: refreshRetrieve } = useAPI(retrieveParams);

  const refresh = () => { refreshExchange(); refreshRetrieve(); };
  usePageActions({ onRefresh: refresh });

  // Reset page on filter change
  React.useEffect(() => { setPageNum(0); }, [timeRange, actorFilter, statusFilter]);

  const allChains = React.useMemo(() => {
    const rawExchange = exchangeData?.items || exchangeData?.audit_logs || exchangeData?.data || (Array.isArray(exchangeData) ? exchangeData : []);
    const rawRetrieve = retrieveData?.items || retrieveData?.audit_logs || retrieveData?.data || (Array.isArray(retrieveData) ? retrieveData : []);
    const all = [...rawExchange, ...rawRetrieve].map(normalizeEntry);
    return buildChains(all);
  }, [exchangeData, retrieveData]);

  const filteredChains = React.useMemo(() => {
    return allChains.filter(c => {
      if (actorFilter && !c.rootSub.toLowerCase().includes(actorFilter.toLowerCase()) &&
          !c.segments.some(s => (s.label || s.sub).toLowerCase().includes(actorFilter.toLowerCase()))) {
        return false;
      }
      return true;
    });
  }, [allChains, actorFilter]);

  const loading = loadingExchange || loadingRetrieve;
  const totalPages = Math.ceil(filteredChains.length / PAGE_SIZE);
  const pageChains = filteredChains.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const navigateToAudit = (id: string) => {
    if (setPage) setPage('audit', { q: id });
    else window.location.hash = '/audit?q=' + encodeURIComponent(id);
  };

  const navigateToAgent = (sub: string) => {
    if (setPage) setPage('agents', { q: sub });
    else window.location.hash = '/agents?q=' + encodeURIComponent(sub);
  };

  return (
    <div style={{ display: 'flex', height: '100%', overflow: 'hidden' }}>
      {/* Main panel */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, overflow: 'hidden' }}>

        {/* Header */}
        <div style={{ padding: '14px 20px 0', borderBottom: '1px solid var(--hairline)' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, marginBottom: 12 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 18, margin: 0, fontWeight: 600 }}>Delegation Chains</h1>
              <p style={{ margin: '2px 0 0', fontSize: 11.5, color: 'var(--fg-dim)' }}>
                Token-exchange delegation graph · grouped by originating subject · sources:{' '}
                <span className="mono">oauth.token.exchanged</span>,{' '}
                <span className="mono">vault.token.retrieved</span>
              </p>
            </div>
            <button className="btn ghost" onClick={refresh} disabled={loading} style={{ flexShrink: 0 }}>
              <Icon.Refresh width={11} height={11}/>
              {loading ? 'Loading…' : 'Refresh'}
            </button>
          </div>

          {/* Filters */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, paddingBottom: 10, flexWrap: 'wrap' }}>
            <Seg
              value={timeRange}
              onChange={setTimeRange}
              opts={[['1h','1h'], ['24h','24h'], ['7d','7d'], ['all','all']]}
            />

            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 6,
              padding: '0 8px',
              height: 24,
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 3,
              minWidth: 160,
            }}>
              <Icon.Search width={11} height={11} style={{ opacity: 0.5 }}/>
              <input
                placeholder="Filter by actor…"
                value={actorFilter}
                onChange={e => setActorFilter(e.target.value)}
                style={{
                  flex: 1,
                  background: 'transparent',
                  border: 0,
                  outline: 'none',
                  color: 'var(--fg)',
                  fontSize: 11.5,
                  fontFamily: 'var(--font-mono)',
                }}
              />
            </div>

            <div style={{ flex: 1 }}/>
            <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>
              {filteredChains.length} chains
            </span>
          </div>
        </div>

        {/* Chain list */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {loading && (
            <div style={{ padding: '12px 20px', fontSize: 11, color: 'var(--fg-dim)' }}>Loading…</div>
          )}

          {!loading && filteredChains.length === 0 && (
            <div style={{ padding: '60px 20px', textAlign: 'center' }}>
              <div style={{
                display: 'inline-block',
                padding: '32px 40px',
                border: '1px solid var(--hairline)',
                borderRadius: 4,
                background: 'var(--surface-1)',
                maxWidth: 500,
              }}>
                <p style={{ fontWeight: 600, margin: '0 0 8px', fontSize: 14 }}>No delegation chains in the selected window</p>
                <p style={{ margin: '0 0 16px', fontSize: 12, color: 'var(--fg-dim)', lineHeight: 1.6 }}>
                  Chains appear when agents exchange tokens on behalf of users. Generate sample data with:
                </p>
                <pre style={{
                  margin: '0 0 12px',
                  padding: '8px 12px',
                  background: 'var(--surface-0)',
                  border: '1px solid var(--hairline)',
                  borderRadius: 3,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  textAlign: 'left' as const,
                  color: 'var(--fg)',
                }}>{'shark demo delegation-with-trace'}</pre>
                <p style={{ margin: 0, fontSize: 11, color: 'var(--fg-dim)' }}>
                  or use the agent demo tester at{' '}
                  <span className="mono">tools/agent_demo_tester.py</span>
                </p>
              </div>
            </div>
          )}

          {pageChains.map((chain, idx) => (
            <div
              key={chain.rootSub + idx}
              onClick={() => setSelected(chain)}
              style={{
                padding: '12px 20px',
                borderBottom: '1px solid var(--hairline)',
                cursor: 'pointer',
                background: selected?.rootSub === chain.rootSub ? 'var(--surface-1)' : 'transparent',
                transition: 'background 60ms',
              }}
              onMouseEnter={e => {
                if (selected?.rootSub !== chain.rootSub)
                  (e.currentTarget as HTMLElement).style.background = 'var(--surface-1)';
              }}
              onMouseLeave={e => {
                if (selected?.rootSub !== chain.rootSub)
                  (e.currentTarget as HTMLElement).style.background = 'transparent';
              }}
            >
              {/* Row header */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <span style={{
                  fontSize: 10,
                  fontWeight: 600,
                  letterSpacing: '0.1em',
                  textTransform: 'uppercase' as const,
                  color: 'var(--fg-dim)',
                  fontFamily: 'var(--font-mono)',
                }}>CHAIN</span>
                <span className="faint mono" style={{ fontSize: 10.5 }}>
                  {fmtTime(chain.latestAt)}
                </span>
                <div style={{ flex: 1 }}/>
                <span style={{ fontSize: 10, color: 'var(--fg-dim)' }}>
                  {chain.events.length} event{chain.events.length !== 1 ? 's' : ''}
                </span>
              </div>

              {/* Breadcrumb */}
              <div style={{ marginBottom: 8 }}>
                <ChainBreadcrumb
                  segments={chain.segments}
                  onAgentClick={sub => {
                    navigateToAgent(sub);
                  }}
                />
              </div>

              {/* Last action */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6, flexWrap: 'wrap' }}>
                <span style={{ fontSize: 11, color: 'var(--fg-dim)' }}>Last action:</span>
                <span className="mono" style={{ fontSize: 11 }}>{chain.lastAction}</span>
                {chain.lastTarget && (
                  <>
                    <span style={{ color: 'var(--fg-dim)', fontSize: 11 }}>·</span>
                    <span className="mono faint" style={{ fontSize: 10.5 }}>target={chain.lastTarget}</span>
                  </>
                )}
              </div>

              {/* Audit ID chips */}
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {chain.events.slice(0, 5).map(ev => (
                  <button
                    key={ev.id}
                    onClick={e => { e.stopPropagation(); navigateToAudit(ev.id); }}
                    style={{
                      fontSize: 10,
                      fontFamily: 'var(--font-mono)',
                      padding: '1px 6px',
                      height: 18,
                      border: '1px solid var(--hairline)',
                      borderRadius: 3,
                      background: 'var(--surface-2)',
                      color: 'var(--fg-dim)',
                      cursor: 'pointer',
                    }}
                    title="Jump to audit event"
                  >
                    {ev.id.slice(0, 16)}…
                  </button>
                ))}
                {chain.events.length > 5 && (
                  <span style={{ fontSize: 10, color: 'var(--fg-dim)', lineHeight: '18px' }}>
                    +{chain.events.length - 5} more
                  </span>
                )}
              </div>
            </div>
          ))}

          {/* Pagination */}
          {totalPages > 1 && (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '12px 20px',
              borderTop: '1px solid var(--hairline)',
            }}>
              <button
                className="btn ghost"
                disabled={page === 0}
                onClick={() => setPageNum(p => Math.max(0, p - 1))}
                style={{ fontSize: 11 }}
              >
                ← Prev
              </button>
              <span style={{ fontSize: 11, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>
                {page + 1} / {totalPages}
              </span>
              <button
                className="btn ghost"
                disabled={page >= totalPages - 1}
                onClick={() => setPageNum(p => Math.min(totalPages - 1, p + 1))}
                style={{ fontSize: 11 }}
              >
                Next →
              </button>
            </div>
          )}
        </div>

        {/* Footer */}
        <div style={{
          padding: '8px 20px',
          borderTop: '1px solid var(--hairline)',
          fontSize: 11,
          display: 'flex',
          gap: 10,
          color: 'var(--fg-dim)',
        }}>
          <span className="mono">{filteredChains.length} chains</span>
          <span>·</span>
          <span className="mono">{allChains.reduce((s, c) => s + c.events.length, 0)} total events</span>
          <div style={{ flex: 1 }}/>
          <span className="mono faint" style={{ fontSize: 10 }}>
            shark audit tail --filter "oauth.token.exchanged"
          </span>
        </div>
      </div>

      {/* Detail drawer */}
      {selected && (
        <ChainDrawer
          chain={selected}
          onClose={() => setSelected(null)}
          onAuditClick={navigateToAudit}
          onAgentClick={navigateToAgent}
        />
      )}
    </div>
  );
}
