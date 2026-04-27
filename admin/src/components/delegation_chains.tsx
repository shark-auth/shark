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

/** Parse a raw JSON metadata string (or object) into a plain object. */
function parseMeta(ev: any): Record<string, any> {
  if (!ev) return {};
  if (typeof ev.metadata === 'string') {
    try { return JSON.parse(ev.metadata) || {}; } catch { return {}; }
  }
  return ev.metadata || {};
}

/** Parse a raw audit-log entry into a normalised shape. */
function normalizeEntry(e: any) {
  const meta = parseMeta(e);
  // Three-way resolution: flat array → use as-is; nested RFC 8693 object → flatten; else empty.
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
    const sorted = [...evts].sort((a, b) =>
      new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );
    const newest = sorted[0];

    const segMap = new Map<string, { sub: string; jkt?: string; label?: string; isUser: boolean }>();
    for (const ev of evts) {
      ev.actChain.forEach((seg, i) => {
        const k = seg.sub || seg.label || String(i);
        if (!segMap.has(k)) segMap.set(k, { ...seg, isUser: i === 0 });
      });
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

  return chains.sort((a, b) =>
    new Date(b.latestAt).getTime() - new Date(a.latestAt).getTime()
  );
}

// ─── breadcrumb renderer (with collapse >3 hops) ─────────────────────────────

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
    fontSize: 10.5,
    border: '1px solid var(--hairline-strong)',
    borderRadius: 3,
    background: 'var(--surface-2)',
    color: 'var(--fg)',
    fontFamily: 'var(--font-mono)',
    whiteSpace: 'nowrap' as const,
    cursor: 'default',
  };

  const arrow = (
    <span style={{ color: 'var(--fg-dim)', fontSize: 10, fontFamily: 'var(--font-mono)', flexShrink: 0 }}>→</span>
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
          fontSize: 10,
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

  // Build react-flow nodes/edges for the linear chain canvas (drawer view)
  const chainRFNodes = React.useMemo(() => {
    return chain.segments.map((seg, i) => ({
      id: seg.sub || seg.label || String(i),
      type: seg.isUser ? 'humanNode' : 'agentNode',
      position: { x: i * 180, y: 60 },
      data: {
        label: seg.label || seg.sub,
        jkt: seg.jkt,
        isUser: seg.isUser,
        actAsCount: chain.segments.length > 1 ? chain.segments.length : undefined,
      },
    }));
  }, [chain.segments]);

  const chainRFEdges = React.useMemo(() => {
    const rawEdges = chain.segments.slice(1).map((seg, i) => {
      const prev = chain.segments[i];
      const fromId = prev.sub || prev.label || String(i);
      const toId = seg.sub || seg.label || String(i + 1);
      const ev = chain.events[i];
      const isActive = i === chain.segments.length - 2; // last hop = most recent
      return {
        id: `${fromId}->${toId}`,
        from: fromId,
        to: toId,
        timestamp: ev?.created_at,
        isActivHop: isActive,
        eventId: ev?.id,
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
    }}>
      {/* Header */}
      <div style={{
        padding: '10px 14px',
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

      <div style={{ flex: 1, overflowY: 'auto', padding: '12px 14px' }}>
        {/* Timestamp */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>Latest event</p>
          <span className="mono" style={{ fontSize: 11 }}>
            {fmtTime(chain.latestAt)}
          </span>
          <CopyField value={chain.latestAt} truncate={0}/>
        </div>

        {/* Chain canvas — React Flow linear hop layout */}
        <div style={{ marginBottom: 14 }}>
          <p style={sectionLabel}>Chain tree</p>
          <div style={{
            border: '1px solid var(--hairline)',
            borderRadius: 0,
            overflow: 'hidden',
            height: Math.max(200, chain.segments.length * 100 + 40),
          }}>
            <DelegationCanvasWithProvider
              rfNodes={chainRFNodes}
              rfEdges={chainRFEdges}
              height={Math.max(200, chain.segments.length * 100 + 40)}
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

        {/* cnf.jkt verification per hop */}
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
                    fontSize: 10.5,
                    color: check.ok ? 'var(--fg-dim)' : 'var(--danger)',
                    border: `1px solid ${check.ok ? 'var(--hairline-strong)' : 'var(--danger)'}`,
                    borderRadius: 2,
                    padding: '0px 4px',
                    flexShrink: 0,
                  }}>
                    {check.ok ? '[ok]' : '[mismatch]'}
                  </span>
                  <span className="mono" style={{ flex: 1, fontSize: 10.5 }}>{seg.label || seg.sub}</span>
                  <span style={{ color: 'var(--fg-dim)', fontSize: 9.5 }}>{check.reason}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* JWT claims per token (collapsible) */}
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
                  }}
                >
                  <Icon.ChevronDown width={10} height={10} style={{
                    transform: open ? 'rotate(0deg)' : 'rotate(-90deg)',
                    transition: 'transform 120ms',
                    opacity: 0.5,
                    flexShrink: 0,
                  }}/>
                  <span className="mono" style={{ flex: 1, fontSize: 10.5 }}>{seg.label || seg.sub}</span>
                  {seg.jkt && (
                    <span style={{ color: 'var(--fg-dim)', fontSize: 9.5, fontFamily: 'var(--font-mono)' }}>
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

      <div style={{ padding: '8px 14px', borderTop: '1px solid var(--hairline)' }}>
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
            color: value === v ? 'var(--fg)' : 'var(--fg-muted)',
            borderRight: '1px solid var(--hairline)',
            cursor: 'pointer',
            fontWeight: value === v ? 500 : 400,
          }}
        >{label}</button>
      ))}
    </div>
  );
}

// ─── chain summary header helpers ────────────────────────────────────────────

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

/** Build "alice@corp → research-agent → tool-agent" path string (max 3 shown) */
function chainPath(segments: Chain['segments']): string {
  const labels = segments.map(s => s.label || s.sub);
  if (labels.length <= 4) return labels.join(' → ');
  return labels[0] + ' → … → ' + labels[labels.length - 1];
}

// ─── canvas graph view ────────────────────────────────────────────────────────

type ViewMode = 'list' | 'canvas';

interface GraphNode {
  id: string;
  label: string;
  isUser: boolean;
  layer: number;
  slotInLayer: number;
}
interface GraphEdge {
  from: string;
  to: string;
  timestamp: string;
  action: string;
  eventId: string;
}

function buildGraph(chains: Chain[]): { nodes: GraphNode[]; edges: GraphEdge[] } {
  // Collect unique nodes and edges from all chains
  const nodeMap = new Map<string, { label: string; isUser: boolean }>();
  const edgeSet = new Map<string, GraphEdge>();

  for (const chain of chains) {
    const segs = chain.segments;
    for (let i = 0; i < segs.length; i++) {
      const seg = segs[i];
      const id = seg.sub || seg.label || String(i);
      if (!nodeMap.has(id)) {
        nodeMap.set(id, { label: seg.label || seg.sub, isUser: seg.isUser });
      }
      // Edge from previous segment
      if (i > 0) {
        const prev = segs[i - 1];
        const fromId = prev.sub || prev.label || String(i - 1);
        const edgeKey = `${fromId}→${id}`;
        if (!edgeSet.has(edgeKey)) {
          const ev = chain.events[0];
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

  // Topological layering: count inbound edges per node
  const inbound = new Map<string, number>();
  for (const [, edge] of edgeSet) {
    inbound.set(edge.to, (inbound.get(edge.to) || 0) + 1);
  }

  // BFS layer assignment
  const layerMap = new Map<string, number>();
  const queue: string[] = [];
  for (const [id] of nodeMap) {
    if (!inbound.has(id) || inbound.get(id) === 0) {
      layerMap.set(id, 0);
      queue.push(id);
    }
  }
  // Process queue
  let qi = 0;
  while (qi < queue.length) {
    const cur = queue[qi++];
    const curLayer = layerMap.get(cur) || 0;
    for (const [, edge] of edgeSet) {
      if (edge.from === cur && !layerMap.has(edge.to)) {
        layerMap.set(edge.to, Math.min(curLayer + 1, 4)); // cap at layer 4
        queue.push(edge.to);
      }
    }
  }
  // Assign remaining nodes to layer 0
  for (const [id] of nodeMap) {
    if (!layerMap.has(id)) layerMap.set(id, 0);
  }

  // Assign slot within each layer
  const layerSlots = new Map<number, number>();
  const nodes: GraphNode[] = [];
  // Sort for determinism
  const sortedIds = Array.from(nodeMap.keys()).sort((a, b) => {
    const la = layerMap.get(a) || 0;
    const lb = layerMap.get(b) || 0;
    return la !== lb ? la - lb : a.localeCompare(b);
  });
  for (const id of sortedIds) {
    const layer = layerMap.get(id) || 0;
    const slot = layerSlots.get(layer) || 0;
    layerSlots.set(layer, slot + 1);
    const info = nodeMap.get(id)!;
    nodes.push({ id, label: info.label, isUser: info.isUser, layer, slotInLayer: slot });
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
  // Selected chain for detail view (null = show aggregate graph)
  const [selectedChain, setSelectedChain] = React.useState<Chain | null>(null);

  const { nodes: graphNodes, edges: graphEdges } = React.useMemo(() => buildGraph(chains), [chains]);

  const rfNodes = React.useMemo(() => toReactFlowNodes(graphNodes), [graphNodes]);

  // Mark the last edge in the sequence as the active hop (bolder)
  const rfEdges = React.useMemo(() => {
    const edges = graphEdges.map((e, i) => ({ ...e, id: e.from + '->' + e.to, isActivHop: i === graphEdges.length - 1 }));
    return toReactFlowEdges(edges);
  }, [graphEdges]);

  // Per-chain view: when a chain is selected, show its linear hop canvas with header
  const chainRFNodes = React.useMemo(() => {
    if (!selectedChain) return null;
    return selectedChain.segments.map((seg, i) => ({
      id: seg.sub || seg.label || String(i),
      type: seg.isUser ? 'humanNode' : 'agentNode',
      position: { x: i * 180, y: 60 },
      data: {
        label: seg.label || seg.sub,
        jkt: seg.jkt,
        isUser: seg.isUser,
        actAsCount: selectedChain.segments.length > 1 ? selectedChain.segments.length : undefined,
      },
    }));
  }, [selectedChain]);

  const chainRFEdges = React.useMemo(() => {
    if (!selectedChain) return null;
    return selectedChain.segments.slice(1).map((seg, i) => {
      const prev = selectedChain.segments[i];
      const fromId = prev.sub || prev.label || String(i);
      const toId = seg.sub || seg.label || String(i + 1);
      const ev = selectedChain.events[i];
      const isActive = i === selectedChain.segments.length - 2; // last hop
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
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--fg-dim)',
          lineHeight: 1.7,
          textAlign: 'center' as const,
          border: '1px solid var(--hairline)',
          borderRadius: 3,
          padding: '20px 32px',
          background: 'var(--surface-1)',
        }}>
          No delegation chains in the selected window.<br/>
          Run shark demo delegation-with-trace.
        </div>
      </div>
    );
  }

  // ── Per-chain detail view ─────────────────────────────────────────────────
  if (selectedChain && chainRFNodes && chainRFEdgesBuilt) {
    const hopCount = selectedChain.segments.length;
    const summary = chainPath(selectedChain.segments);
    const started = relTime(selectedChain.latestAt);

    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Chain summary header */}
        <div style={{
          padding: '8px 16px',
          borderBottom: '1px solid var(--hairline)',
          background: 'var(--surface-1)',
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          flexShrink: 0,
        }}>
          <button
            onClick={() => setSelectedChain(null)}
            style={{
              background: 'transparent',
              border: 0,
              cursor: 'pointer',
              color: 'var(--fg-dim)',
              fontFamily: 'var(--font-mono)',
              fontSize: 10.5,
              padding: 0,
              display: 'flex',
              alignItems: 'center',
              gap: 4,
              flexShrink: 0,
            }}
          >
            ← chains
          </button>
          <span style={{ color: 'var(--hairline-strong)', fontSize: 10, flexShrink: 0 }}>|</span>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10.5,
            color: 'var(--fg)',
            fontWeight: 500,
            flexShrink: 0,
          }}>
            {hopCount}-hop chain
          </span>
          {started && (
            <>
              <span style={{ color: 'var(--fg-dim)', fontSize: 10, flexShrink: 0 }}>·</span>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', flexShrink: 0 }}>
                started {started}
              </span>
            </>
          )}
          <span style={{ color: 'var(--fg-dim)', fontSize: 10, flexShrink: 0 }}>·</span>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            color: 'var(--fg-dim)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            flex: 1,
            minWidth: 0,
          }} title={summary}>{summary}</span>
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
    );
  }

  // ── Aggregate graph with clickable nodes → drill into chain ───────────────
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
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)' }}>
          {chains.length} chain{chains.length !== 1 ? 's' : ''} · click a chain row to drill in
        </span>
        {chains.length > 0 && (
          <>
            <span style={{ color: 'var(--fg-dim)', fontSize: 10 }}>·</span>
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
  // View-mode toggle — persisted to localStorage
  const [viewMode, setViewMode] = React.useState<ViewMode>(() => {
    try { return (localStorage.getItem('shark_chains_view') as ViewMode) || 'list'; } catch { return 'list'; }
  });

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

  // Persist view-mode to localStorage on change
  React.useEffect(() => {
    try { localStorage.setItem('shark_chains_view', viewMode); } catch {}
  }, [viewMode]);

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

  // Keyboard navigation (arrow up/down through rows)
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

        {/* Header */}
        <div style={{ padding: '12px 20px 0', borderBottom: '1px solid var(--hairline)' }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, marginBottom: 10 }}>
            <div style={{ flex: 1 }}>
              <h1 style={{ fontSize: 16, margin: 0, fontWeight: 600 }}>Delegation Chains</h1>
              <p style={{ margin: '2px 0 0', fontSize: 11, color: 'var(--fg-dim)' }}>
                Token-exchange delegation graph · grouped by originating subject · sources:{' '}
                <span className="mono">oauth.token.exchanged</span>,{' '}
                <span className="mono">vault.token.retrieved</span>
              </p>
            </div>
            <button className="btn ghost" onClick={refresh} disabled={loading} style={{ flexShrink: 0, fontSize: 11 }}>
              <Icon.Refresh width={11} height={11}/>
              {loading ? 'Loading…' : 'Refresh'}
            </button>
          </div>

          {/* Filters bar — monospace inputs, 28px height, tight */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, paddingBottom: 8, flexWrap: 'wrap' }}>
            <Seg
              value={timeRange}
              onChange={setTimeRange}
              opts={[['1h','1h'], ['24h','24h'], ['7d','7d'], ['all','all']]}
            />
            {/* View-mode toggle: List / Canvas */}
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
              <Icon.Search width={10} height={10} style={{ opacity: 0.45, flexShrink: 0 }}/>
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
            <span style={{ fontSize: 10.5, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>
              {filteredChains.length} / {allChains.length}
            </span>
          </div>
        </div>

        {/* Chain table / canvas */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {loading && (
            <div style={{ padding: '10px 20px', fontSize: 11, color: 'var(--fg-dim)', fontFamily: 'var(--font-mono)' }}>Loading…</div>
          )}

          {/* Canvas mode */}
          {!loading && viewMode === 'canvas' && (
            <ChainCanvas
              chains={filteredChains}
              setPage={setPage}
              onAuditClick={navigateToAudit}
            />
          )}

          {!loading && viewMode === 'list' && filteredChains.length === 0 && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', padding: 40 }}>
              <div style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--fg-dim)',
                lineHeight: 1.7,
                textAlign: 'center' as const,
                border: '1px solid var(--hairline)',
                borderRadius: 3,
                padding: '20px 32px',
                background: 'var(--surface-1)',
              }}>
                No delegation chains in the selected window.<br/>
                Run shark demo delegation-with-trace.
              </div>
            </div>
          )}

          {/* Table header row */}
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
                        background: isSelected ? 'var(--surface-1)' : (focusedIdx === idx ? 'var(--surface-1)' : 'transparent'),
                        outline: focusedIdx === idx ? '1px solid var(--hairline-strong)' : 'none',
                        outlineOffset: -1,
                      }}
                      onMouseEnter={e => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'var(--surface-1)';
                      }}
                      onMouseLeave={e => {
                        if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'transparent';
                      }}
                    >
                      {/* Root subject */}
                      <td style={{ ...tdStyle, maxWidth: 140 }}>
                        <span className="mono" style={{ fontSize: 10.5, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {chain.rootSub.length > 20 ? chain.rootSub.slice(0, 18) + '…' : chain.rootSub}
                        </span>
                      </td>

                      {/* Breadcrumb */}
                      <td style={{ ...tdStyle, minWidth: 200 }}>
                        <ChainBreadcrumb
                          segments={chain.segments}
                          onAgentClick={sub => navigateToAgent(sub)}
                        />
                      </td>

                      {/* Last action */}
                      <td style={{ ...tdStyle, maxWidth: 160 }}>
                        <span className="mono" style={{ fontSize: 10, display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--fg-dim)' }}>
                          {chain.lastAction}
                        </span>
                      </td>

                      {/* Timestamp */}
                      <td style={{ ...tdStyle, whiteSpace: 'nowrap' }}>
                        <span className="mono faint" style={{ fontSize: 10 }}>
                          {fmtTime(chain.latestAt)}
                        </span>
                      </td>

                      {/* Event count chip */}
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
          fontSize: 10.5,
          display: 'flex',
          gap: 10,
          color: 'var(--fg-dim)',
          fontFamily: 'var(--font-mono)',
        }}>
          <span>{filteredChains.length} chains</span>
          <span>·</span>
          <span>{allChains.reduce((s, c) => s + c.events.length, 0)} events</span>
          <div style={{ flex: 1 }}/>
          <span style={{ fontSize: 9.5, opacity: 0.6 }}>
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
