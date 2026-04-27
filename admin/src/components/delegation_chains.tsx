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
import { DelegationCanvasWithProvider, toReactFlowNodes, toReactFlowEdges } from './delegation_canvas'

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
  padding: '6px 14px',
  borderBottom: '1px solid var(--hairline)',
  verticalAlign: 'middle',
  fontSize: 11,
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

/**
 * flattenActClaim — convert an RFC 8693 nested act-chain object into a flat
 * array of hops.  Backend buildActClaim emits:
 *   { "sub": "a2", "act": { "sub": "a1" } }
 * This walker unrolls the chain so delegation_chains and agents_manage can
 * render hops correctly regardless of whether the server returns an array or
 * a nested object.
 */
function flattenActClaim(nested: any): Array<{sub: string; jkt?: string; label?: string}> {
  const hops: Array<{sub: string; jkt?: string}> = [];
  let cur = nested;
  while (cur && typeof cur === 'object' && cur.sub) {
    hops.push({ sub: cur.sub, jkt: cur.jkt });
    cur = cur.act;
  }
  return hops;
}

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

function parseMeta(ev: any): Record<string, any> {
  if (!ev) return {};
  if (typeof ev.metadata === 'string') {
    try { return JSON.parse(ev.metadata) || {}; } catch { return {}; }
  }
  return ev.metadata || {};
}

function normalizeEntry(e: any) {
  const meta = parseMeta(e);
  const resolveChain = (raw: any): Array<{sub: string; jkt?: string; label?: string}> => {
    if (Array.isArray(raw) && raw.length > 0) return raw;
    if (raw && typeof raw === 'object' && raw.sub) return flattenActClaim(raw);
    return [];
  };

  const actChain: Array<{ sub: string; jkt?: string; label?: string }> =
    resolveChain(e.act_chain).length > 0
      ? resolveChain(e.act_chain)
      : resolveChain(meta.act_chain).length > 0
        ? resolveChain(meta.act_chain)
        : e.oauth?.act
          ? [{ sub: e.oauth.act.sub || '', label: e.oauth.act.email || e.oauth.act.sub || 'user' }]
          : [];

  const rootSub = actChain.length > 0 ? (actChain[0].label || actChain[0].sub) : (e.actor_email || e.actor_id || 'unknown');

  // Read structured scope fields emitted by backend (v0.2+).
  // Fall back to the legacy flat "scope" string for older events.
  const grantedScope: string[] =
    Array.isArray(meta.granted_scope) ? meta.granted_scope :
    Array.isArray(meta.scope) ? meta.scope :
    typeof meta.scope === 'string' ? meta.scope.split(' ').filter(Boolean) :
    Array.isArray(meta.scopes) ? meta.scopes :
    [];

  const subjectScope: string[] =
    Array.isArray(meta.subject_scope) ? meta.subject_scope : grantedScope;

  const droppedScope: string[] =
    Array.isArray(meta.dropped_scope) ? meta.dropped_scope :
    subjectScope.filter((s: string) => !grantedScope.includes(s));

  const requestedScope: string[] =
    Array.isArray(meta.requested_scope) ? meta.requested_scope : grantedScope;

  return {
    id: e.id || '',
    created_at: e.created_at || '',
    action: e.action || e.event_type || '',
    actor: e.actor_email || e.actor_id || '',
    actor_type: e.actor_type || 'system',
    target: [e.target_type, e.target_id].filter(Boolean).join('_') || '',
    actChain,
    rootSub,
    grantedScope,
    subjectScope,
    droppedScope,
    requestedScope,
    _raw: e,
  };
}

type NormEntry = ReturnType<typeof normalizeEntry>;

interface Chain {
  rootSub: string;
  latestAt: string;
  events: NormEntry[];
  segments: Array<{
    sub: string;
    jkt?: string;
    label?: string;
    isUser: boolean;
    grantedScope: string[];
    subjectScope: string[];
    droppedScope: string[];
  }>;
  lastAction: string;
  lastTarget: string;
}

function buildChains(entries: NormEntry[]): Chain[] {
  const map = new Map<string, NormEntry[]>();
  for (const e of entries) {
    const key = e.rootSub;
    if (!map.has(key)) map.set(key, []);
    map.get(key)!.push(e);
  }

  const chains: Chain[] = [];
  map.forEach((evts, rootSub) => {
    const sorted = [...evts].sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    const newest = sorted[0];

    // Build a map of segment key → segment metadata with scope info from the
    // most-recent event that mentions each hop.
    const segMap = new Map<string, {
      sub: string; jkt?: string; label?: string; isUser: boolean;
      grantedScope: string[]; subjectScope: string[]; droppedScope: string[];
    }>();

    for (const ev of sorted) {
      ev.actChain.forEach((seg, i) => {
        const k = seg.sub || seg.label || String(i);
        if (!segMap.has(k)) {
          segMap.set(k, {
            ...seg,
            isUser: i === 0,
            grantedScope: ev.grantedScope,
            subjectScope: ev.subjectScope,
            droppedScope: ev.droppedScope,
          });
        }
      });
      if (ev.actor && !segMap.has(ev.actor)) {
        segMap.set(ev.actor, {
          sub: ev.actor, label: ev.actor, isUser: false,
          grantedScope: ev.grantedScope,
          subjectScope: ev.subjectScope,
          droppedScope: ev.droppedScope,
        });
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
  const [expanded, setExpanded] = React.useState(false);

  const chipBase: React.CSSProperties = {
    display: 'inline-flex',
    alignItems: 'center',
    gap: 3,
    padding: '1px 6px',
    height: 18,
    fontSize: 10,
    border: '1px solid var(--hairline-strong)',
    borderRadius: 3,
    background: 'var(--surface-2)',
    color: 'var(--fg)',
    fontFamily: 'var(--font-mono)',
    whiteSpace: 'nowrap' as const,
    cursor: 'default',
    letterSpacing: '0.01em',
  };

  const arrow = (
    <span style={{ color: 'var(--fg-dim)', fontSize: 9, fontFamily: 'var(--font-mono)', flexShrink: 0, opacity: 0.5 }}>→</span>
  );

  const renderSegment = (seg: Chain['segments'][0], i: number) => {
    const label = seg.label || seg.sub;
    const jkt = jktShort(seg.jkt);
    const isClickable = !seg.isUser && !!seg.sub;
    const display = jkt ? `${label} (${jkt}…)` : label;

    if (isClickable) {
      return (
        <button
          key={i}
          onClick={(e) => { e.stopPropagation(); onAgentClick(seg.sub); }}
          style={{ ...chipBase, background: 'var(--surface-1)', cursor: 'pointer' }}
          title={seg.sub}
        >
          {display}
        </button>
      );
    }
    return (
      <span key={i} style={chipBase} title={seg.sub}>
        {display}
      </span>
    );
  };

  const COLLAPSE_THRESHOLD = 3;
  const shouldCollapse = segments.length > COLLAPSE_THRESHOLD && !expanded;

  let rendered: React.ReactNode[];
  if (shouldCollapse) {
    const hidden = segments.length - 2;
    rendered = [
      renderSegment(segments[0], 0),
      arrow,
      <button
        key="collapse"
        onClick={e => { e.stopPropagation(); setExpanded(true); }}
        style={{
          ...chipBase,
          background: 'transparent',
          border: '1px dashed var(--hairline-strong)',
          cursor: 'pointer',
          color: 'var(--fg-dim)',
          fontSize: 9.5,
          opacity: 0.65,
        }}
      >
        … ({hidden} more) …
      </button>,
      arrow,
      renderSegment(segments[segments.length - 1], segments.length - 1),
    ];
  } else {
    rendered = [];
    segments.forEach((seg, i) => {
      if (i > 0) rendered.push(<React.Fragment key={`a${i}`}>{arrow}</React.Fragment>);
      rendered.push(renderSegment(seg, i));
    });
  }

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 4 }}>
      {rendered}
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

  const chainRFNodes = React.useMemo(() => {
    const total = chain.segments.length;
    return chain.segments.map((seg, i) => ({
      id: seg.sub || seg.label || String(i),
      type: seg.isUser ? 'humanNode' : 'agentNode',
      position: { x: i * 200, y: 60 },
      data: {
        label: seg.label || seg.sub,
        jkt: seg.jkt,
        isUser: seg.isUser,
        chainPos: i + 1,
        chainTotal: total > 1 ? total : undefined,
      },
    }));
  }, [chain.segments]);

  const chainRFEdges = React.useMemo(() => {
    const now = Date.now();
    const rawEdges = chain.segments.slice(1).map((seg, i) => {
      const prev = chain.segments[i];
      const fromId = prev.sub || prev.label || String(i);
      const toId = seg.sub || seg.label || String(i + 1);
      const ev = chain.events[i];
      const isActive = !!ev?.created_at && (now - new Date(ev.created_at).getTime()) < 60_000;
      return {
        id: `${fromId}->${toId}`,
        from: fromId,
        to: toId,
        timestamp: ev?.created_at,
        isActivHop: isActive,
        eventId: ev?.id,
        scopeFrom: seg.subjectScope.length > 0 ? seg.subjectScope : prev.grantedScope,
        scopeTo: seg.grantedScope,
      };
    });
    return toReactFlowEdges(rawEdges);
  }, [chain.segments, chain.events]);

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
      boxShadow: '-4px 0 24px rgba(0,0,0,0.3)',
    }}>
      {/* Header */}
      <div style={{
        padding: '10px 14px',
        borderBottom: '1px solid var(--hairline)',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexShrink: 0,
      }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--fg-dim)',
            textTransform: 'uppercase' as const,
            letterSpacing: '0.08em',
            display: 'block',
          }}>Chain detail</span>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9.5,
            color: 'var(--fg-dim)',
            opacity: 0.55,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            display: 'block',
            marginTop: 1,
          }} title={chain.rootSub}>
            {chain.rootSub}
          </span>
        </div>
        <button className="btn ghost icon sm" onClick={onClose}>
          <Icon.X width={11} height={11}/>
        </button>
      </div>

      <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
        {/* Latest event */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>Latest event</p>
          <span className="mono" style={{ fontSize: 11 }}>
            {fmtTime(chain.latestAt)}
          </span>
          <CopyField value={chain.latestAt} truncate={0}/>
        </div>

        {/* Chain canvas */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>Chain tree</p>
          <div style={{
            border: '1px solid var(--hairline)',
            borderRadius: 3,
            overflow: 'hidden',
            height: Math.max(220, chain.segments.length * 120 + 60),
          }}>
            <DelegationCanvasWithProvider
              rfNodes={chainRFNodes}
              rfEdges={chainRFEdges}
              height={Math.max(220, chain.segments.length * 120 + 60)}
              onNodeClick={(nodeId, nodeData) => {
                if (!nodeData.isUser && nodeId) onAgentClick(nodeId);
              }}
              onEdgeClick={(edgeData) => {
                if (edgeData?.eventId) onAuditClick(edgeData.eventId);
              }}
              fitView
            />
          </div>
        </div>

        {/* cnf.jkt verification */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>Cnf.jkt verification</p>
          <div style={{ border: '1px solid var(--hairline)', borderRadius: 3, background: 'var(--surface-1)' }}>
            {chain.segments.map((seg, i) => {
              const check = jktChecks[i];
              return (
                <div key={i} style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '5px 10px',
                  borderBottom: i < chain.segments.length - 1 ? '1px solid var(--hairline)' : 'none',
                  fontSize: 11,
                }}>
                  <span style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10,
                    color: check.ok ? 'var(--fg-dim)' : 'var(--danger)',
                    border: `1px solid ${check.ok ? 'var(--hairline-strong)' : 'var(--danger)'}`,
                    borderRadius: 2,
                    padding: '0px 4px',
                    flexShrink: 0,
                    opacity: check.ok ? 0.7 : 1,
                  }}>
                    {check.ok ? '[ok]' : '[mismatch]'}
                  </span>
                  <span className="mono" style={{ flex: 1, fontSize: 10.5 }}>{seg.label || seg.sub}</span>
                  <span style={{ color: 'var(--fg-dim)', fontSize: 9.5, opacity: 0.6 }}>{check.reason}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* JWT claims per hop */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>JWT claims per hop</p>
          {chain.segments.map((seg, i) => {
            const claims = {
              sub: seg.sub,
              ...(seg.jkt ? { 'cnf.jkt': seg.jkt } : {}),
              label: seg.label,
            };
            const open = expandedTokens.has(i);
            return (
              <div key={i} style={{ marginBottom: 3, border: '1px solid var(--hairline)', borderRadius: 3 }}>
                <button
                  onClick={() => toggleToken(i)}
                  style={{
                    width: '100%',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    padding: '6px 10px',
                    background: open ? 'var(--surface-2)' : 'var(--surface-1)',
                    fontSize: 11,
                    cursor: 'pointer',
                    borderRadius: open ? '3px 3px 0 0' : 3,
                    textAlign: 'left' as const,
                    transition: 'background 100ms',
                  }}
                >
                  <Icon.ChevronDown width={10} height={10} style={{
                    transform: open ? 'rotate(0deg)' : 'rotate(-90deg)',
                    transition: 'transform 120ms',
                    opacity: 0.4,
                    flexShrink: 0,
                  }}/>
                  <span className="mono" style={{ flex: 1, fontSize: 10.5 }}>{seg.label || seg.sub}</span>
                  {seg.jkt && (
                    <span style={{ color: 'var(--fg-dim)', fontSize: 9.5, fontFamily: 'var(--font-mono)', opacity: 0.6 }}>
                      jkt:{seg.jkt.slice(0, 8)}…
                    </span>
                  )}
                </button>
                {open && (
                  <pre style={{
                    margin: 0,
                    padding: '7px 10px',
                    background: 'var(--surface-0)',
                    fontFamily: 'var(--font-mono)',
                    fontSize: 10.5,
                    lineHeight: 1.6,
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

        {/* Audit events */}
        <div style={{ marginBottom: 14 }}>
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

      <div style={{ padding: '8px 14px', borderTop: '1px solid var(--hairline)', flexShrink: 0 }}>
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
      height: 28,
    }}>
      {opts.map(([v, label]) => (
        <button
          key={String(v)}
          onClick={() => onChange(v)}
          style={{
            padding: '0 9px',
            height: 28,
            fontSize: 11,
            fontFamily: 'var(--font-mono)',
            background: value === v ? 'var(--surface-3)' : 'var(--surface-2)',
            color: value === v ? 'var(--fg)' : 'var(--fg-dim)',
            borderRight: '1px solid var(--hairline)',
            cursor: 'pointer',
            fontWeight: value === v ? 500 : 400,
            transition: 'background 80ms, color 80ms',
          }}
        >{label}</button>
      ))}
    </div>
  );
}

// ─── helpers ─────────────────────────────────────────────────────────────────

/** Format relative time: "2m ago", "3h ago", "5d ago" */
function relTime(iso: string): string {
  try {
    const diff = Date.now() - new Date(iso).getTime();
    const s = Math.floor(diff / 1000);
    if (s < 60) return `${s}s ago`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ago`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h}h ago`;
    return `${Math.floor(h / 24)}d ago`;
  } catch {
    return '';
  }
}

/** Build "alice@corp → research-agent → tool-agent" path string */
function chainPath(segments: Chain['segments']): string {
  const labels = segments.map(s => s.label || s.sub);
  if (labels.length <= 4) return labels.join(' → ');
  return labels[0] + ' → … → ' + labels[labels.length - 1];
}

// ─── chain selector panel (left side of canvas split) ────────────────────────

function ChainSelectorItem({
  chain,
  isSelected,
  onClick,
}: {
  chain: Chain;
  isSelected: boolean;
  onClick: () => void;
}) {
  const hopCount = chain.segments.length;
  const started = relTime(chain.latestAt);
  const path = chainPath(chain.segments);

  return (
    <div
      onClick={onClick}
      style={{
        padding: '9px 14px',
        borderBottom: '1px solid var(--hairline)',
        background: isSelected ? 'var(--surface-2)' : 'transparent',
        cursor: 'pointer',
        transition: 'background 80ms',
        position: 'relative',
      }}
      onMouseEnter={e => {
        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'var(--surface-1)';
      }}
      onMouseLeave={e => {
        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'transparent';
      }}
    >
      {/* Subtle selection indicator — a thin right-side hairline inset via outline, NOT border-left stripe */}
      {isSelected && (
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          bottom: 0,
          width: 2,
          background: 'var(--fg)',
          opacity: 0.8,
        }}/>
      )}

      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        {/* Hop count chip */}
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          color: isSelected ? 'var(--fg)' : 'var(--fg-dim)',
          border: '1px solid var(--hairline-strong)',
          borderRadius: 3,
          padding: '0 4px',
          height: 14,
          lineHeight: '14px',
          display: 'inline-flex',
          alignItems: 'center',
          flexShrink: 0,
          transition: 'color 80ms',
        }}>
          {hopCount}-hop
        </span>

        {/* Root subject */}
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10.5,
          color: isSelected ? 'var(--fg)' : 'var(--fg-dim)',
          fontWeight: isSelected ? 500 : 400,
          flex: 1,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          transition: 'color 80ms',
        }} title={chain.rootSub}>
          {chain.rootSub}
        </span>

        {/* Relative time */}
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9.5,
          color: 'var(--fg-dim)',
          flexShrink: 0,
          opacity: 0.5,
        }}>
          {started}
        </span>
      </div>

      {/* Path summary */}
      <div style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9.5,
        color: 'var(--fg-dim)',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        opacity: isSelected ? 0.65 : 0.38,
        paddingLeft: hopCount >= 10 ? 0 : 0,
        transition: 'opacity 80ms',
      }}>
        {path}
      </div>
    </div>
  );
}

// ─── canvas graph view ────────────────────────────────────────────────────────

type ViewMode = 'list' | 'canvas';

interface GraphEdge {
  from: string;
  to: string;
  timestamp: string;
  action: string;
  eventId: string;
}

interface GraphNode {
  id: string;
  label: string;
  isUser: boolean;
  layer: number;
  slotInLayer: number;
  lane: number;
  laneLabel: string;
}

function buildGraph(chains: Chain[]): { nodes: GraphNode[]; edges: GraphEdge[] } {
  // Emit per-chain swim lanes — nodes scoped to their own lane so distinct
  // user→agent chains don't fuse into a tangled shared graph.
  const nodes: GraphNode[] = [];
  const edgeSet = new Map<string, GraphEdge>();

  for (let laneIdx = 0; laneIdx < chains.length; laneIdx++) {
    const chain = chains[laneIdx];
    const segs = chain.segments;

    // Build lane label: "alice@corp · N-hop · HH:MM"
    const root = chain.rootSub.length > 18 ? chain.rootSub.slice(0, 16) + '…' : chain.rootSub;
    const hopCount = segs.length;
    const ts = chain.latestAt
      ? new Date(chain.latestAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : '';
    const laneLabel = ts
      ? `${root} · ${hopCount}-hop · ${ts}`
      : `${root} · ${hopCount}-hop`;

    for (let i = 0; i < segs.length; i++) {
      const seg = segs[i];
      // Scope id to lane to prevent cross-chain node fusion
      const id = `lane${laneIdx}__${seg.sub || seg.label || String(i)}`;
      nodes.push({
        id,
        label: seg.label || seg.sub,
        isUser: seg.isUser,
        layer: i,
        slotInLayer: 0,
        lane: laneIdx,
        laneLabel,
      });

      if (i > 0) {
        const prev = segs[i - 1];
        const fromId = `lane${laneIdx}__${prev.sub || prev.label || String(i - 1)}`;
        const edgeKey = `${fromId}→${id}`;
        if (!edgeSet.has(edgeKey)) {
          const ev = chain.events[i - 1] || chain.events[0];
          edgeSet.set(edgeKey, {
            from: fromId, to: id,
            timestamp: ev?.created_at || '',
            action: ev?.action || '',
            eventId: ev?.id || '',
          });
        }
      }
    }
  }

  return { nodes, edges: Array.from(edgeSet.values()) };
}

function ChainCanvas({
  chains,
  setPage,
  onAuditClick,
}: {
  chains: Chain[];
  setPage?: (p: string, extra?: any) => void;
  onAuditClick: (id: string) => void;
}) {
  const [selectedChain, setSelectedChain] = React.useState<Chain | null>(null);

  const { nodes: graphNodes, edges: graphEdges } = React.useMemo(() => buildGraph(chains), [chains]);

  const rfNodes = React.useMemo(() => {
    const base = toReactFlowNodes(graphNodes);
    return base;
  }, [graphNodes]);

  const rfEdges = React.useMemo(() => {
    const now = Date.now();
    const edges = graphEdges.map((e) => ({
      ...e,
      id: e.from + '->' + e.to,
      isActivHop: !!e.timestamp && (now - new Date(e.timestamp).getTime()) < 60_000,
    }));
    return toReactFlowEdges(edges);
  }, [graphEdges]);

  // Per-chain nodes/edges for focused view
  const chainRFNodes = React.useMemo(() => {
    if (!selectedChain) return null;
    const total = selectedChain.segments.length;
    return selectedChain.segments.map((seg, i) => ({
      id: seg.sub || seg.label || String(i),
      type: seg.isUser ? 'humanNode' : 'agentNode',
      position: { x: i * 200, y: 80 },
      data: {
        label: seg.label || seg.sub,
        jkt: seg.jkt,
        isUser: seg.isUser,
        chainPos: i + 1,
        chainTotal: total > 1 ? total : undefined,
      },
    }));
  }, [selectedChain]);

  const chainRFEdges = React.useMemo(() => {
    if (!selectedChain) return null;
    const now = Date.now();
    return selectedChain.segments.slice(1).map((seg, i) => {
      const prev = selectedChain.segments[i];
      const fromId = prev.sub || prev.label || String(i);
      const toId = seg.sub || seg.label || String(i + 1);
      const ev = selectedChain.events[i];
      const isActive = !!ev?.created_at && (now - new Date(ev.created_at).getTime()) < 60_000;
      return {
        id: `${fromId}->${toId}`,
        from: fromId,
        to: toId,
        timestamp: ev?.created_at,
        isActivHop: isActive,
        eventId: ev?.id,
      };
    });
  }, [selectedChain]);

  const chainRFEdgesBuilt = React.useMemo(
    () => chainRFEdges ? toReactFlowEdges(chainRFEdges) : null,
    [chainRFEdges]
  );

  if (graphNodes.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', padding: 40 }}>
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 10,
        }}>
          <svg width="36" height="24" viewBox="0 0 36 24" fill="none" style={{ opacity: 0.18 }}>
            <circle cx="4" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
            <circle cx="18" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
            <circle cx="32" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
            <line x1="7" y1="12" x2="15" y2="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="2 2"/>
            <line x1="21" y1="12" x2="29" y2="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="2 2"/>
          </svg>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--fg-dim)',
            textAlign: 'center' as const,
            lineHeight: 1.6,
            opacity: 0.6,
          }}>
            No delegation chains in the selected window.<br/>
            Run shark demo delegation-with-trace.
          </span>
        </div>
      </div>
    );
  }

  // ── Selected chain detail view (split: left selector + right canvas) ──────
  if (selectedChain && chainRFNodes && chainRFEdgesBuilt) {
    const hopCount = selectedChain.segments.length;
    const started = relTime(selectedChain.latestAt);
    const summary = chainPath(selectedChain.segments);

    // Assign a stable color per hop-count bucket (1-hop grey, 2-hop teal, 3+-hop amber)
    const dotColor = (c: Chain) => {
      const h = c.segments.length;
      if (h <= 1) return 'var(--fg-dim)';
      if (h === 2) return 'var(--accent, #5eead4)';
      if (h === 3) return '#f59e0b';
      return 'var(--danger, #ef4444)';
    };

    return (
      <div style={{ display: 'flex', height: '100%' }}>
        {/* Icon-rail sidebar — 56px, reclaims 164px vs old 220px sidebar */}
        <div style={{
          width: 56,
          borderRight: '1px solid var(--hairline)',
          flexShrink: 0,
          background: 'var(--surface-0)',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          paddingTop: 6,
          gap: 2,
          overflowY: 'auto',
        }}>
          {/* Back-to-all button */}
          <button
            onClick={() => setSelectedChain(null)}
            title="All chains"
            style={{
              width: 32,
              height: 22,
              background: 'transparent',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 3,
              cursor: 'pointer',
              color: 'var(--fg-dim)',
              fontFamily: 'var(--font-mono)',
              fontSize: 9,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              marginBottom: 6,
              transition: 'color 80ms, border-color 80ms',
            }}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.color = 'var(--fg)'; (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.color = 'var(--fg-dim)'; (e.currentTarget as HTMLElement).style.borderColor = 'var(--hairline-strong)'; }}
          >←</button>

          {/* One colored dot per chain — hover = path tooltip, click = select */}
          {chains.map((chain, idx) => {
            const isSelected = chain.rootSub === selectedChain.rootSub;
            const color = dotColor(chain);
            const path = chainPath(chain.segments);
            return (
              <button
                key={chain.rootSub}
                onClick={() => setSelectedChain(chain)}
                title={`${chain.segments.length}-hop · ${path}`}
                style={{
                  width: 28,
                  height: 28,
                  borderRadius: '50%',
                  background: isSelected ? color : 'transparent',
                  border: `2px solid ${color}`,
                  cursor: 'pointer',
                  flexShrink: 0,
                  opacity: isSelected ? 1 : 0.45,
                  transition: 'opacity 80ms, background 80ms',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
                onMouseEnter={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.opacity = '0.85'; }}
                onMouseLeave={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.opacity = '0.45'; }}
              >
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 7.5, color: isSelected ? 'var(--surface-0)' : color, fontWeight: 600, lineHeight: 1, userSelect: 'none' }}>
                  {chain.segments.length}
                </span>
              </button>
            );
          })}
        </div>

        {/* Canvas + header */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* Compact chain header — single-line, no path duplication */}
          <div style={{
            padding: '7px 16px',
            borderBottom: '1px solid var(--hairline)',
            background: 'var(--surface-1)',
            display: 'flex',
            alignItems: 'center',
            gap: 10,
            flexShrink: 0,
          }}>
            <span style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--fg)',
              fontWeight: 500,
              flexShrink: 0,
            }}>
              {hopCount}-hop
            </span>
            {started && (
              <>
                <span style={{ color: 'var(--hairline-strong)', fontSize: 10, flexShrink: 0, opacity: 0.4 }}>·</span>
                <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', flexShrink: 0, opacity: 0.65 }}>
                  {started}
                </span>
              </>
            )}
            <span style={{ color: 'var(--hairline-strong)', fontSize: 10, flexShrink: 0, opacity: 0.4 }}>·</span>
            <span style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--fg-dim)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              flex: 1,
              minWidth: 0,
              opacity: 0.55,
            }} title={summary}>{summary}</span>

            {/* Event count */}
            <span style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 9.5,
              color: 'var(--fg-dim)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 3,
              padding: '0 5px',
              height: 16,
              lineHeight: '16px',
              display: 'inline-flex',
              alignItems: 'center',
              flexShrink: 0,
              opacity: 0.6,
            }}>
              {selectedChain.events.length} evt
            </span>
          </div>

          <DelegationCanvasWithProvider
            rfNodes={chainRFNodes}
            rfEdges={chainRFEdgesBuilt}
            height={520}
            onNodeClick={(nodeId, nodeData) => {
              if (!setPage) return;
              if (nodeData.isUser) setPage('users', { userId: nodeId });
              else setPage('agents', { q: nodeId });
            }}
            onEdgeClick={(edgeData) => {
              if (edgeData?.eventId) onAuditClick(edgeData.eventId);
            }}
            fitView
          />
        </div>
      </div>
    );
  }

  // ── Aggregate graph ──────────────────────────────────────────────────────
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Aggregate header */}
      <div style={{
        padding: '7px 16px',
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-1)',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        flexShrink: 0,
      }}>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.7 }}>
          {chains.length} chain{chains.length !== 1 ? 's' : ''} · click a chain row to drill in
        </span>
        {chains.length > 0 && (
          <>
            <span style={{ color: 'var(--fg-dim)', fontSize: 10, opacity: 0.3 }}>·</span>
            <button
              onClick={() => setSelectedChain(chains[0])}
              style={{
                background: 'transparent',
                border: '1px solid var(--hairline-strong)',
                borderRadius: 3,
                cursor: 'pointer',
                color: 'var(--fg-dim)',
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                padding: '1px 7px',
                height: 20,
                transition: 'color 80ms, border-color 80ms',
              }}
              onMouseEnter={e => {
                (e.currentTarget as HTMLElement).style.color = 'var(--fg)';
                (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
              }}
              onMouseLeave={e => {
                (e.currentTarget as HTMLElement).style.color = 'var(--fg-dim)';
                (e.currentTarget as HTMLElement).style.borderColor = 'var(--hairline-strong)';
              }}
            >
              view latest chain
            </button>
          </>
        )}
      </div>

      <DelegationCanvasWithProvider
        rfNodes={rfNodes}
        rfEdges={rfEdges}
        height={554}
        onNodeClick={(nodeId, nodeData) => {
          if (!setPage) return;
          if (nodeData.isUser) setPage('users', { userId: nodeId });
          else setPage('agents', { q: nodeId });
        }}
        onEdgeClick={(edgeData) => {
          if (edgeData?.eventId) onAuditClick(edgeData.eventId);
        }}
        fitView
      />
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
  const rowRefs = React.useRef<(HTMLDivElement | null)[]>([]);
  const [focusedIdx, setFocusedIdx] = React.useState<number>(-1);
  const [viewMode, setViewMode] = React.useState<ViewMode>(() => {
    try { return (localStorage.getItem('shark_chains_view') as ViewMode) || 'list'; } catch { return 'list'; }
  });

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

  React.useEffect(() => {
    try { localStorage.setItem('shark_chains_view', viewMode); } catch {}
  }, [viewMode]);

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

  React.useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (selected) {
        if (e.key === 'Escape') { setSelected(null); setFocusedIdx(-1); return; }
      }
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setFocusedIdx(prev => {
          const next = Math.min(prev + 1, pageChains.length - 1);
          if (pageChains[next]) { setSelected(pageChains[next]); }
          return next;
        });
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setFocusedIdx(prev => {
          const next = Math.max(prev - 1, 0);
          if (pageChains[next]) { setSelected(pageChains[next]); }
          return next;
        });
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [pageChains, selected]);

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

        {/* Page header */}
        <div style={{ padding: '12px 20px 0', borderBottom: '1px solid var(--hairline)' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, marginBottom: 10 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 16, margin: 0, fontWeight: 600 }}>Delegation Chains</h1>
              <p style={{ margin: '2px 0 0', fontSize: 11, color: 'var(--fg-dim)', lineHeight: 1.5 }}>
                Token-exchange delegation graph · grouped by originating subject · sources:{' '}
                <span className="mono">oauth.token.exchanged</span>,{' '}
                <span className="mono">vault.token.retrieved</span>
              </p>
            </div>
            <button
              className="btn ghost"
              onClick={refresh}
              disabled={loading}
              style={{ flexShrink: 0, fontSize: 11 }}
            >
              <Icon.Refresh width={11} height={11}/>
              {loading ? 'Loading…' : 'Refresh'}
            </button>
          </div>

          {/* Filters */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, paddingBottom: 8, flexWrap: 'wrap' }}>
            <Seg
              value={timeRange}
              onChange={setTimeRange}
              opts={[['1h','1h'], ['24h','24h'], ['7d','7d'], ['all','all']]}
            />
            <Seg
              value={viewMode}
              onChange={(v: ViewMode) => setViewMode(v)}
              opts={[['list','List'], ['canvas','Canvas']]}
            />

            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 5,
              padding: '0 8px',
              height: 28,
              background: 'var(--surface-1)',
              border: '1px solid var(--hairline-strong)',
              borderRadius: 3,
              minWidth: 180,
            }}>
              <Icon.Search width={10} height={10} style={{ opacity: 0.4, flexShrink: 0 }}/>
              <input
                placeholder="Filter by actor…"
                value={actorFilter}
                onChange={e => setActorFilter(e.target.value)}
                title="Tip: press / to focus (global shortcut)"
                style={{
                  flex: 1,
                  background: 'transparent',
                  border: 0,
                  outline: 'none',
                  color: 'var(--fg)',
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                }}
              />
            </div>

            <div style={{ flex: 1 }}/>
            <span style={{ fontSize: 10, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)', opacity: 0.55 }}>
              {filteredChains.length} / {allChains.length}
            </span>
          </div>
        </div>

        {/* Content area */}
        <div style={{ flex: 1, overflowY: 'auto' }}>

          {/* Loading state */}
          {loading && (
            <div style={{
              padding: '24px 20px',
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              justifyContent: 'center',
            }}>
              {[0,1,2].map(i => (
                <div key={i} style={{
                  width: 3,
                  height: 3,
                  borderRadius: '50%',
                  background: 'var(--fg-dim)',
                  opacity: 0.35,
                  animation: `edge-glow 1.2s ease-in-out ${i * 200}ms infinite`,
                }}/>
              ))}
              <style>{`@keyframes edge-glow { 0%,100%{opacity:0.35} 50%{opacity:0.9} }`}</style>
            </div>
          )}

          {/* Canvas mode */}
          {!loading && viewMode === 'canvas' && (
            <ChainCanvas
              chains={filteredChains}
              setPage={setPage}
              onAuditClick={navigateToAudit}
            />
          )}

          {/* Empty state — list mode */}
          {!loading && viewMode === 'list' && filteredChains.length === 0 && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 200, padding: 40 }}>
              <div style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                gap: 10,
              }}>
                <svg width="36" height="24" viewBox="0 0 36 24" fill="none" style={{ opacity: 0.18 }}>
                  <circle cx="4" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
                  <circle cx="18" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
                  <circle cx="32" cy="12" r="3" stroke="currentColor" strokeWidth="1.5"/>
                  <line x1="7" y1="12" x2="15" y2="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="2 2"/>
                  <line x1="21" y1="12" x2="29" y2="12" stroke="currentColor" strokeWidth="1.5" strokeDasharray="2 2"/>
                </svg>
                <span style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 11,
                  color: 'var(--fg-dim)',
                  opacity: 0.6,
                  textAlign: 'center' as const,
                  lineHeight: 1.6,
                }}>
                  No delegation chains in the selected window.<br/>
                  Run shark demo delegation-with-trace.
                </span>
              </div>
            </div>
          )}

          {/* Table */}
          {!loading && viewMode === 'list' && pageChains.length > 0 && (
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={thStyle}>Root subject</th>
                  <th style={thStyle}>Chain</th>
                  <th style={thStyle}>Last action</th>
                  <th style={thStyle}>Latest</th>
                  <th style={thStyle}>Events</th>
                </tr>
              </thead>
              <tbody>
                {pageChains.map((chain, idx) => {
                  const isSelected = selected?.rootSub === chain.rootSub;
                  return (
                    <tr
                      key={chain.rootSub + idx}
                      ref={el => { rowRefs.current[idx] = el as any; }}
                      onClick={() => { setSelected(chain); setFocusedIdx(idx); }}
                      style={{
                        cursor: 'pointer',
                        background: isSelected
                          ? 'var(--surface-1)'
                          : (focusedIdx === idx ? 'var(--surface-1)' : 'transparent'),
                        outline: focusedIdx === idx ? '1px solid var(--hairline-strong)' : 'none',
                        outlineOffset: -1,
                        transition: 'background 80ms',
                      }}
                      onMouseEnter={e => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'var(--surface-1)';
                      }}
                      onMouseLeave={e => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'transparent';
                      }}
                    >
                      <td style={{ ...tdStyle, maxWidth: 140 }}>
                        <span className="mono" style={{ fontSize: 10.5, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {chain.rootSub.length > 20 ? chain.rootSub.slice(0, 18) + '…' : chain.rootSub}
                        </span>
                      </td>
                      <td style={{ ...tdStyle, minWidth: 200 }}>
                        <ChainBreadcrumb
                          segments={chain.segments}
                          onAgentClick={sub => navigateToAgent(sub)}
                        />
                      </td>
                      <td style={{ ...tdStyle, maxWidth: 160 }}>
                        <span className="mono" style={{ fontSize: 10, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--fg-dim)' }}>
                          {chain.lastAction}
                        </span>
                      </td>
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>
                        <span className="mono faint" style={{ fontSize: 10 }}>
                          {fmtTime(chain.latestAt)}
                        </span>
                      </td>
                      <td style={tdStyle}>
                        <span style={{
                          display: 'inline-block',
                          fontFamily: 'var(--font-mono)',
                          fontSize: 10,
                          padding: '0 5px',
                          height: 16,
                          lineHeight: '16px',
                          border: '1px solid var(--hairline-strong)',
                          borderRadius: 3,
                          color: 'var(--fg-dim)',
                        }}>
                          {chain.events.length}
                        </span>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          )}

          {/* Pagination */}
          {totalPages > 1 && (
            <div style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 20px',
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
              <span style={{ fontSize: 10.5, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>
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
          padding: '6px 20px',
          borderTop: '1px solid var(--hairline)',
          fontSize: 10,
          display: 'flex',
          gap: 10,
          color: 'var(--fg-dim)',
          fontFamily: 'var(--font-mono)',
          opacity: 0.65,
        }}>
          <span>{filteredChains.length} chains</span>
          <span>·</span>
          <span>{allChains.reduce((s, c) => s + c.events.length, 0)} events</span>
          <div style={{ flex: 1 }}/>
          <span style={{ fontSize: 9, opacity: 0.7 }}>
            shark audit tail --filter "oauth.token.exchanged"
          </span>
        </div>
      </div>

      {/* Detail drawer */}
      {selected && (
        <ChainDrawer
          chain={selected}
          onClose={() => { setSelected(null); setFocusedIdx(-1); }}
          onAuditClick={navigateToAudit}
          onAgentClick={navigateToAgent}
        />
      )}
    </div>
  );
}
