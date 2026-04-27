// @ts-nocheck
// delegation_canvas.tsx — shared Railway-style React Flow canvas
// Used by: delegation_chains.tsx (full graph) + agents_manage.tsx (ego graph)
// Visual contract: square chrome, monochrome, .impeccable.md v3
// F7: initials avatar on human nodes, act-as badge on agent nodes,
//     "via token_exchange · <ts>" edge tooltip (hover-only), active hop animated stroke

import React, { useState } from 'react'
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Handle,
  Position,
  getBezierPath,
  EdgeLabelRenderer,
  BaseEdge,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

// ─── keyframe animations ──────────────────────────────────────────────────────

const ANIMATION_KEYFRAMES = `
  @keyframes dash-march {
    to { stroke-dashoffset: -24; }
  }
  @keyframes node-pulse {
    0%, 100% { box-shadow: 0 0 0 0 rgba(255,255,255,0.08); }
    50%       { box-shadow: 0 0 0 5px rgba(255,255,255,0.0); }
  }
  @keyframes edge-glow {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0.65; }
  }
  @keyframes fade-in {
    from { opacity: 0; transform: scale(0.96); }
    to   { opacity: 1; transform: scale(1); }
  }
  @keyframes tooltip-in {
    from { opacity: 0; transform: translate(-50%, -50%) scale(0.92); }
    to   { opacity: 1; transform: translate(-50%, -50%) scale(1); }
  }
`

// ─── global style overrides (monochrome, square chrome) ──────────────────────

const CANVAS_OVERRIDES = `
  ${ANIMATION_KEYFRAMES}

  .react-flow__node {
    border-radius: 0 !important;
  }

  /* Handles: flush square dots */
  .react-flow__handle {
    border-radius: 2px !important;
    width: 5px !important;
    height: 5px !important;
    background: var(--fg-dim) !important;
    border: none !important;
    opacity: 0.45;
    transition: opacity 150ms;
  }
  .react-flow__node:hover .react-flow__handle {
    opacity: 1;
  }

  /* Default edge */
  .react-flow__edge-path {
    stroke: var(--fg-dim) !important;
    stroke-width: 1px !important;
  }

  /* Active-hop edge: dashed march animation — strokeWidth: 2 (bolder than hairline) */
  .react-flow__edge.active-hop .react-flow__edge-path {
    stroke: var(--fg) !important;
    stroke-width: 2px !important;
    stroke-dasharray: 6 4 !important;
    animation: dash-march 800ms linear infinite !important;
  }

  .react-flow__edge.selected .react-flow__edge-path {
    stroke: var(--fg) !important;
    stroke-width: 2px !important;
  }

  /* Controls — square, no shadow */
  .react-flow__controls {
    border-radius: 3px !important;
    border: 1px solid var(--hairline) !important;
    box-shadow: none !important;
    overflow: hidden;
  }
  .react-flow__controls-button {
    border-radius: 0 !important;
    background: var(--surface-1) !important;
    border-bottom: 1px solid var(--hairline) !important;
    color: var(--fg-dim) !important;
    fill: var(--fg-dim) !important;
    width: 22px !important;
    height: 22px !important;
    transition: background 100ms;
  }
  .react-flow__controls-button:hover {
    background: var(--surface-2) !important;
  }
  .react-flow__controls-button svg {
    width: 10px !important;
    height: 10px !important;
  }

  /* MiniMap */
  .react-flow__minimap {
    border-radius: 3px !important;
    border: 1px solid var(--hairline) !important;
    box-shadow: none !important;
  }
  .react-flow__minimap-mask { fill: var(--surface-1) !important; opacity: 0.75 !important; }
  .react-flow__minimap-node { fill: var(--fg-dim) !important; }

  .react-flow__attribution { display: none !important; }

  /* Edge label pill — always visible (static badge) */
  .edge-label-pill {
    background: var(--surface-2);
    border: 1px solid var(--hairline-strong);
    border-radius: 3px;
    padding: 1px 5px;
    font-family: ui-monospace, monospace;
    font-size: 8.5px;
    color: var(--fg-dim);
    pointer-events: none;
    white-space: nowrap;
    line-height: 1.5;
    display: inline-block;
  }
  .edge-label-pill.active {
    color: var(--fg);
    border-color: var(--fg-dim);
  }

  /* Edge hover tooltip — invisible by default, slides in on hover */
  .edge-tooltip {
    background: var(--surface-1) !important;
    border: 1px solid var(--hairline-strong) !important;
    border-radius: 2px !important;
    padding: 2px 6px !important;
    font-family: var(--font-mono) !important;
    font-size: 9px !important;
    color: var(--fg-dim) !important;
    pointer-events: none !important;
    white-space: nowrap;
    line-height: 1.5;
    opacity: 0;
    transition: opacity 120ms ease, transform 120ms ease;
    transform: translate(-50%, -50%) scale(0.94);
    box-shadow: 0 2px 8px rgba(0,0,0,0.32);
    z-index: 10;
  }
  .edge-tooltip.visible {
    opacity: 1 !important;
    transform: translate(-50%, -50%) scale(1) !important;
    animation: tooltip-in 120ms ease forwards !important;
  }
  .edge-tooltip.active {
    color: var(--fg) !important;
    border-color: var(--fg-dim) !important;
  }

  /* Faded chains */
  .react-flow__node.faded {
    opacity: 0.28 !important;
    transition: opacity 200ms ease;
  }
  .react-flow__edge.faded .react-flow__edge-path {
    opacity: 0.18 !important;
    transition: opacity 200ms ease;
  }
`

// ─── initials helper ─────────────────────────────────────────────────────────

/**
 * getInitials — extract 1-2 uppercase letters from an email or display name.
 *   "alice@corp.com"  → "AL"
 *   "research-agent"  → "RA"
 *   "tool_agent_v2"   → "TA"
 *   "Bob Smith"       → "BS"
 */
export function getInitials(label: string): string {
  if (!label) return '?';
  const local = label.includes('@') ? label.split('@')[0] : label;
  const parts = local.split(/[\s\-_.]+/).filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase().slice(0, 2);
  }
  return local.slice(0, 2).toUpperCase();
}

// ─── custom edge type: AnimatedBezierEdge ─────────────────────────────────────
// Edges are clean hairlines by default.
// Hover anywhere on the edge path area to reveal the hover tooltip
// showing "via token_exchange · HH:MM" at the midpoint.
// Active hop uses marching dashes + strokeWidth: 2 (bolder than hairline).

function AnimatedBezierEdge({
  id, sourceX, sourceY, targetX, targetY,
  sourcePosition, targetPosition,
  data, style, markerEnd, selected,
}) {
  const [hovered, setHovered] = useState(false);

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX, sourceY, sourcePosition,
    targetX, targetY, targetPosition,
    curvature: 0.18,
  });

  const isActive = data?.isActive;
  // Full label contains "via token_exchange · HH:MM" — used for hover tooltip
  const label = data?.label;
  // Static pill shows "acts-as · HH:MM" (user-friendly, no RFC jargon)
  const pillLabel = React.useMemo(() => {
    if (!label) return 'acts-as';
    // Extract time portion after last " · "
    const parts = label.split(' · ');
    const ts = parts.length > 1 ? parts[parts.length - 1] : '';
    return ts ? `acts-as · ${ts}` : 'acts-as';
  }, [label]);
  // Hover tooltip shows full technical detail
  const tooltipLabel = label
    ? label.replace('via token_exchange', `RFC 8693 token_exchange`)
    : '';

  return (
    <>
      {/* Invisible wide hit area for hover detection */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={14}
        style={{ cursor: 'crosshair' }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      />
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={style}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            pointerEvents: 'none',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 2,
          }}
          className="nodrag nopan"
        >
          {/* Static always-visible pill */}
          <div className={`edge-label-pill${isActive ? ' active' : ''}`}>
            {pillLabel}
          </div>
          {/* Hover tooltip with full RFC detail */}
          {tooltipLabel && (
            <div className={`edge-tooltip${hovered ? ' visible' : ''}${isActive ? ' active' : ''}`}
              style={{ position: 'static', transform: 'none', marginTop: 2 }}
            >
              {tooltipLabel}
            </div>
          )}
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

// ─── custom node types ────────────────────────────────────────────────────────

/**
 * HumanNode — 80×80, circular initials avatar + name below.
 * Generous breathing room, whisper shadow, 8px radius (Apple-polish exception).
 */
function HumanNode({ data, selected }: { data: any; selected?: boolean }) {
  const initials = getInitials(data.label || '');

  return (
    <div style={{
      width: 80,
      height: 80,
      background: selected ? 'var(--surface-2)' : 'var(--surface-1)',
      border: `1px solid ${selected ? 'var(--fg)' : 'var(--hairline-strong)'}`,
      borderRadius: 8,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '10px 8px 8px',
      cursor: 'pointer',
      position: 'relative',
      transition: 'border-color 100ms, box-shadow 100ms, background 100ms',
      gap: 6,
      boxShadow: selected
        ? '0 0 0 1px var(--fg), 0 4px 16px rgba(0,0,0,0.45)'
        : '0 2px 8px rgba(0,0,0,0.28)',
      animation: selected ? 'node-pulse 2s ease-in-out infinite' : 'fade-in 160ms ease-out',
    }}
      onMouseEnter={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 12px rgba(0,0,0,0.38)';
        }
      }}
      onMouseLeave={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--hairline-strong)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.28)';
        }
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -3 }} />
      <Handle type="source" position={Position.Right} style={{ right: -3 }} />

      {/* Circular initials avatar */}
      <div style={{
        width: 32,
        height: 32,
        borderRadius: '50%',
        background: 'var(--surface-0)',
        border: '1px solid var(--hairline-strong)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexShrink: 0,
      }}>
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          fontWeight: 600,
          color: 'var(--fg)',
          letterSpacing: '0.04em',
          lineHeight: 1,
        }}>{initials}</span>
      </div>

      {/* Name */}
      <span style={{
        fontFamily: 'var(--font-body, var(--font-mono))',
        fontSize: 9.5,
        color: selected ? 'var(--fg)' : 'var(--fg-dim)',
        maxWidth: 70,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1.2,
        fontWeight: 500,
        transition: 'color 100ms',
      }} title={data.label}>{data.label}</span>
    </div>
  )
}

/**
 * AgentNode — 80×80, square-ish, mono name, act-as count badge, jkt hint.
 */
function AgentNode({ data, selected }: { data: any; selected?: boolean }) {
  return (
    <div style={{
      width: 80,
      height: 80,
      background: selected ? 'var(--surface-2)' : 'var(--surface-1)',
      border: `1px solid ${selected ? 'var(--fg)' : 'var(--hairline-strong)'}`,
      borderRadius: 6,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '8px 6px',
      cursor: 'pointer',
      position: 'relative',
      transition: 'border-color 100ms, box-shadow 100ms, background 100ms',
      gap: 4,
      boxShadow: selected
        ? '0 0 0 1px var(--fg), 0 4px 16px rgba(0,0,0,0.45)'
        : '0 2px 8px rgba(0,0,0,0.28)',
      animation: 'fade-in 160ms ease-out',
    }}
      onMouseEnter={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 12px rgba(0,0,0,0.38)';
        }
      }}
      onMouseLeave={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--hairline-strong)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.28)';
        }
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -3 }} />
      <Handle type="source" position={Position.Right} style={{ right: -3 }} />

      {/* "agent" type label */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 7,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.1em',
        lineHeight: 1,
        opacity: 0.6,
      }}>agent</span>

      {/* Agent name */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 10,
        color: selected ? 'var(--fg)' : 'var(--fg)',
        fontWeight: 500,
        maxWidth: 70,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1.25,
        transition: 'color 100ms',
      }} title={data.label}>{data.label}</span>

      {/* jkt hint */}
      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 7.5,
          color: 'var(--fg-dim)',
          lineHeight: 1,
          opacity: 0.55,
        }}>jkt:{data.jkt.slice(0, 6)}</span>
      )}

      {/* act-as count badge — bottom-right */}
      {data.actAsCount != null && data.actAsCount > 1 && (
        <div style={{
          position: 'absolute',
          bottom: 4,
          right: 4,
          minWidth: 14,
          height: 14,
          borderRadius: 3,
          background: 'var(--surface-0)',
          border: '1px solid var(--hairline-strong)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '0 3px',
        }}>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 7.5,
            color: 'var(--fg-dim)',
            fontWeight: 600,
            lineHeight: 1,
          }}>{data.actAsCount}</span>
        </div>
      )}
    </div>
  )
}

/**
 * CenterAgentNode — ego-graph focal node, slightly larger, full-brightness border.
 */
function CenterAgentNode({ data, selected }: { data: any; selected?: boolean }) {
  return (
    <div style={{
      width: 88,
      height: 88,
      background: 'var(--surface-2)',
      border: '1.5px solid var(--fg)',
      borderRadius: 8,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '8px 6px',
      cursor: 'default',
      position: 'relative',
      gap: 4,
      boxShadow: '0 4px 20px rgba(0,0,0,0.5)',
      animation: 'node-pulse 2.5s ease-in-out infinite',
    }}>
      <Handle type="target" position={Position.Top} style={{ top: -3 }} />
      <Handle type="source" position={Position.Bottom} style={{ bottom: -3 }} />
      <Handle type="target" position={Position.Left} style={{ left: -3 }} />
      <Handle type="source" position={Position.Right} style={{ right: -3 }} />

      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 7,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.1em',
        lineHeight: 1,
        opacity: 0.7,
      }}>agent</span>

      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 10.5,
        color: 'var(--fg)',
        fontWeight: 600,
        maxWidth: 76,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1.25,
      }} title={data.label}>{data.label}</span>

      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 7.5,
          color: 'var(--fg-dim)',
          lineHeight: 1,
          opacity: 0.55,
        }}>jkt:{data.jkt.slice(0, 6)}</span>
      )}
    </div>
  )
}

// UserNode kept as alias for HumanNode (used by agents_manage ego graph)
const UserNode = HumanNode;

/**
 * LaneLabelNode — gutter label for swim-lane, no handles, not selectable.
 * Displays "alice@corp · 3-hop · 14:32" in fg-dim 9.5px monospace.
 */
function LaneLabelNode({ data }: { data: any }) {
  return (
    <div style={{
      pointerEvents: 'none',
      userSelect: 'none',
      fontFamily: 'ui-monospace, monospace',
      fontSize: 9.5,
      color: 'var(--fg-dim)',
      opacity: 0.55,
      whiteSpace: 'nowrap',
      lineHeight: 1.4,
      borderRight: '1px solid var(--hairline)',
      paddingRight: 8,
      textAlign: 'right',
      minWidth: 140,
    }}>
      {data.label}
    </div>
  )
}

const nodeTypes = {
  agentNode: AgentNode,
  userNode: UserNode,
  humanNode: HumanNode,
  centerAgentNode: CenterAgentNode,
  laneLabel: LaneLabelNode,
}

const edgeTypes = {
  animatedBezier: AnimatedBezierEdge,
}

// ─── public types ─────────────────────────────────────────────────────────────

export interface DCanvasNode {
  id: string
  label: string
  isUser: boolean
  isCenter?: boolean
  layer: number
  slotInLayer: number
  lane?: number          // swim-lane index (0-based); each chain gets its own horizontal band
  laneLabel?: string     // gutter label, e.g. "alice@corp · 3-hop · 14:32"
  jkt?: string
  meta?: any
  actAsCount?: number
}

export interface DCanvasEdge {
  id: string
  from: string
  to: string
  timestamp?: string
  action?: string
  eventId?: string
  label?: string
  isActivHop?: boolean
}

// ─── layout helper ────────────────────────────────────────────────────────────

// Generous spacing: nodes have more breathing room
const LAYER_GAP = 200
const SLOT_GAP = 120
// Swim-lane vertical offset per chain (240px bands)
const LANE_GAP = 240

export function toReactFlowNodes(nodes: DCanvasNode[]) {
  const rfNodes: any[] = []

  // Track which lanes already have a gutter label rendered
  const labeledLanes = new Set<number>()

  for (const n of nodes) {
    const laneOffset = (n.lane ?? 0) * LANE_GAP
    rfNodes.push({
      id: n.id,
      type: n.isCenter ? 'centerAgentNode' : n.isUser ? 'humanNode' : 'agentNode',
      position: {
        x: n.layer * LAYER_GAP,
        y: n.slotInLayer * SLOT_GAP + laneOffset,
      },
      data: {
        label: n.label,
        jkt: n.jkt,
        isUser: n.isUser,
        isCenter: n.isCenter,
        meta: n.meta,
        actAsCount: n.actAsCount,
      },
      selected: false,
    })

    // Gutter label node: one per lane, pinned to x=-170 at mid-lane y
    if (n.lane != null && n.laneLabel && !labeledLanes.has(n.lane)) {
      labeledLanes.add(n.lane)
      rfNodes.push({
        id: `__lane_label_${n.lane}`,
        type: 'laneLabel',
        position: { x: -170, y: laneOffset + LANE_GAP / 2 - 10 },
        data: { label: n.laneLabel },
        selectable: false,
        draggable: false,
      })
    }
  }

  return rfNodes
}

export function toReactFlowEdges(edges: DCanvasEdge[]) {
  return edges.map((e) => {
    const ts = e.timestamp
      ? new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : ''
    // label text is stored in data for hover tooltip; contains "token_exchange" keyword
    const baseLabel = ts ? `via token_exchange · ${ts}` : 'via token_exchange'
    const edgeLabel = e.label || baseLabel

    const isActive = e.isActivHop === true
    const strokeColor = isActive ? 'var(--fg)' : 'var(--fg-dim)'
    // Active hop: strokeWidth: 2 (bolder than hairline 1px)
    const strokeWidth = isActive ? 2 : 1

    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'animatedBezier',
      className: isActive ? 'active-hop' : '',
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: isActive ? 10 : 8,
        height: isActive ? 10 : 8,
        color: strokeColor,
      },
      style: {
        stroke: strokeColor,
        // strokeWidth: 2 for active-hop (bolder), 1 for historical hairline
        strokeWidth,
        ...(isActive ? {
          strokeDasharray: '6 4',
          animation: 'dash-march 800ms linear infinite',
        } : {}),
      },
      data: {
        label: edgeLabel,
        eventId: e.eventId,
        isActive,
      },
    }
  })
}

// ─── ego-graph layout ─────────────────────────────────────────────────────────

export function toEgoLayout(
  centerNode: DCanvasNode,
  inbound: DCanvasNode[],
  outbound: DCanvasNode[],
  inboundEdges: DCanvasEdge[],
  outboundEdges: DCanvasEdge[],
) {
  const NODE_W = 88
  const ROW_GAP = 140
  const COL_GAP = 180

  const layoutRow = (items: DCanvasNode[], y: number) =>
    items.map((n, i) => ({
      id: n.id,
      type: n.isUser ? 'humanNode' : 'agentNode',
      position: {
        x: (i - (items.length - 1) / 2) * COL_GAP,
        y,
      },
      data: { label: n.label, jkt: n.jkt, isUser: n.isUser, actAsCount: n.actAsCount },
    }))

  const rfNodes = [
    ...layoutRow(inbound, 0),
    {
      id: centerNode.id,
      type: 'centerAgentNode',
      position: { x: -(NODE_W / 2), y: ROW_GAP },
      data: { label: centerNode.label, jkt: centerNode.jkt },
    },
    ...layoutRow(outbound, ROW_GAP * 2),
  ]

  const makeEdge = (e: DCanvasEdge, isActive: boolean) => {
    const ts = e.timestamp
      ? new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : ''
    // label stored for hover tooltip; "token_exchange" keyword always present
    const edgeLabel = e.label || (ts ? `via token_exchange · ${ts}` : 'via token_exchange')
    const strokeColor = isActive ? 'var(--fg)' : 'var(--fg-dim)'
    // Active hop: strokeWidth: 2 (bolder than hairline 1px)
    const strokeWidth = isActive ? 2 : 1
    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'animatedBezier',
      className: isActive ? 'active-hop' : '',
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: isActive ? 10 : 8,
        height: isActive ? 10 : 8,
        color: strokeColor,
      },
      style: {
        stroke: strokeColor,
        strokeWidth,
        ...(isActive ? { strokeDasharray: '6 4', animation: 'dash-march 800ms linear infinite' } : {}),
      },
      data: { label: edgeLabel, eventId: e.eventId, isActive },
    }
  }

  const rfEdges = [
    ...inboundEdges.map((e, i) => makeEdge(e, i === inboundEdges.length - 1)),
    ...outboundEdges.map((e, i) => makeEdge(e, i === outboundEdges.length - 1)),
  ]

  return { rfNodes, rfEdges }
}

// ─── empty / loading / error states ──────────────────────────────────────────

function EmptyCanvas({ message, hint }: { message: string; hint?: string }) {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100%',
      padding: 48,
    }}>
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 10,
        maxWidth: 280,
      }}>
        {/* Sparse graph icon */}
        <svg width="36" height="24" viewBox="0 0 36 24" fill="none" style={{ opacity: 0.2 }}>
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
          textAlign: 'center',
          lineHeight: 1.6,
        }}>{message}</span>
        {hint && (
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 9.5,
            color: 'var(--fg-dim)',
            opacity: 0.5,
            textAlign: 'center',
            lineHeight: 1.5,
          }}>{hint}</span>
        )}
      </div>
    </div>
  );
}

function LoadingCanvas() {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      height: '100%',
    }}>
      <div style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
      }}>
        {/* Three-dot pulse */}
        {[0, 1, 2].map(i => (
          <div key={i} style={{
            width: 4,
            height: 4,
            borderRadius: '50%',
            background: 'var(--fg-dim)',
            opacity: 0.35,
            animation: `edge-glow 1.2s ease-in-out ${i * 200}ms infinite`,
          }}/>
        ))}
      </div>
    </div>
  );
}

// ─── node detail drawer ───────────────────────────────────────────────────────
// Slides in from right on node click. 400px, hairline left border, monochrome.
// Human node: email, name, ID, signup date, role chips, owned agents, sessions.
// Agent node: name, client_id, DPoP jkt, status, created_at, scopes, chain ctx.
// Both: "View in audit log →" link.

const DRAWER_SLIDE_IN = `
  @keyframes drawerSlideIn {
    from { transform: translateX(100%); opacity: 0; }
    to   { transform: translateX(0);    opacity: 1; }
  }
`

function NodeDrawer({ node, rfNodes, rfEdges, onClose }: {
  node: { id: string; data: any } | null
  rfNodes: any[]
  rfEdges: any[]
  onClose: () => void
}) {
  // Close on Escape
  React.useEffect(() => {
    if (!node) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [node, onClose])

  if (!node) return null

  const d = node.data || {}
  const isHuman = d.isUser === true || d.type === 'human'

  // Derive chain context for agent nodes
  // Position = 1-based index among all edges where this node appears as source or target
  const inEdges = rfEdges.filter(e => e.target === node.id)
  const outEdges = rfEdges.filter(e => e.source === node.id)
  const allNodes = rfNodes || []
  const prevHop = inEdges.length > 0 ? allNodes.find(n => n.id === inEdges[0].source) : null
  const nextHop = outEdges.length > 0 ? allNodes.find(n => n.id === outEdges[0].target) : null
  const chainPos = inEdges.length + 1  // 1 = first actor, 2 = second, etc.
  const edgeLabel = inEdges[0]?.data?.label || outEdges[0]?.data?.label || ''
  const tokenType = edgeLabel.includes('token_exchange') ? 'token_exchange' : (edgeLabel || '—')
  const hopTs = edgeLabel.replace('via token_exchange · ', '').replace('via token_exchange', '').trim() || '—'

  const auditLink = `/admin/audit?actor_id=${encodeURIComponent(node.id)}`

  return (
    <>
      <style>{DRAWER_SLIDE_IN}</style>
      {/* Backdrop — click outside to close */}
      <div
        onClick={onClose}
        style={{
          position: 'absolute', inset: 0, zIndex: 20,
          background: 'rgba(0,0,0,0.18)',
          cursor: 'default',
        }}
        data-testid="node-drawer-backdrop"
      />
      {/* Drawer panel */}
      <div
        onClick={e => e.stopPropagation()}
        data-testid="node-drawer"
        style={{
          position: 'absolute', top: 0, right: 0, bottom: 0,
          width: 400, maxWidth: '92%',
          zIndex: 21,
          background: 'var(--surface-0)',
          borderLeft: '1px solid var(--hairline)',
          display: 'flex', flexDirection: 'column',
          animation: 'drawerSlideIn 200ms ease-out',
          overflowY: 'auto',
        }}
      >
        {/* Header */}
        <div style={{
          padding: '12px 14px 10px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          flexShrink: 0,
        }}>
          <span style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 10,
            textTransform: 'uppercase',
            letterSpacing: '0.08em',
            color: 'var(--fg-dim)',
          }}>
            {isHuman ? 'human · ' : 'agent · '}{node.id.slice(0, 12)}
          </span>
          <button
            onClick={onClose}
            data-testid="node-drawer-close"
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'var(--fg-dim)', fontSize: 16, lineHeight: 1,
              padding: '2px 4px', borderRadius: 2,
              fontFamily: 'var(--font-mono)',
            }}
            title="Close (Esc)"
          >✕</button>
        </div>

        {/* Body */}
        <div style={{ flex: 1, padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 16 }}>

          {isHuman ? (
            // ── Human fields ──────────────────────────────────────────────────
            <HumanDrawerFields node={node} />
          ) : (
            // ── Agent fields ──────────────────────────────────────────────────
            <AgentDrawerFields
              node={node}
              chainPos={chainPos}
              prevHop={prevHop}
              nextHop={nextHop}
              tokenType={tokenType}
              hopTs={hopTs}
            />
          )}

          {/* View in audit log — both types */}
          <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
            <a
              href={auditLink}
              data-testid="node-drawer-audit-link"
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--fg-dim)',
                textDecoration: 'none',
                display: 'inline-flex', alignItems: 'center', gap: 4,
              }}
              onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
              onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-dim)')}
            >
              View in audit log →
            </a>
          </div>
        </div>
      </div>
    </>
  )
}

// ── Human drawer fields ───────────────────────────────────────────────────────

function HumanDrawerFields({ node }: { node: { id: string; data: any } }) {
  const d = node.data || {}
  return (
    <div data-testid="human-drawer-fields" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <DrawerField label="Name" value={d.label || d.name || '—'} />
      <DrawerField label="Email" value={d.email || d.label || '—'} />
      <DrawerField label="ID" value={node.id} mono />
      <DrawerField label="Signed up" value={d.created_at ? new Date(d.created_at).toLocaleDateString() : '—'} />

      {/* Role chips */}
      {d.roles && d.roles.length > 0 && (
        <div>
          <div style={drawerLabelStyle}>Roles</div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
            {d.roles.map((r: any) => (
              <span key={r.id || r} style={chipStyle}>{r.name || r}</span>
            ))}
          </div>
        </div>
      )}

      {/* Owned agents */}
      {d.ownedAgents && d.ownedAgents.length > 0 && (
        <div data-testid="human-owned-agents">
          <div style={drawerLabelStyle}>Owned agents</div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 3, marginTop: 4 }}>
            {d.ownedAgents.map((a: any) => (
              <span key={a.id || a} style={{
                fontFamily: 'var(--font-mono)', fontSize: 10,
                color: 'var(--fg-dim)', cursor: 'pointer',
                padding: '2px 0',
              }}
                onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-dim)')}
              >
                {a.name || a.client_id || a}
              </span>
            ))}
          </div>
        </div>
      )}

      <DrawerField label="Recent sessions" value={d.sessionCount != null ? String(d.sessionCount) : '—'} />

      {/* View user link */}
      <a
        href={`/admin/users?id=${encodeURIComponent(node.id)}`}
        data-testid="human-drawer-user-link"
        style={{
          fontFamily: 'var(--font-mono)', fontSize: 11,
          color: 'var(--fg-dim)', textDecoration: 'none',
        }}
        onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
        onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-dim)')}
      >
        View user →
      </a>
    </div>
  )
}

// ── Agent drawer fields ───────────────────────────────────────────────────────

function AgentDrawerFields({ node, chainPos, prevHop, nextHop, tokenType, hopTs }: {
  node: { id: string; data: any }
  chainPos: number
  prevHop: any
  nextHop: any
  tokenType: string
  hopTs: string
}) {
  const d = node.data || {}
  const status = d.status || d.active === false ? 'inactive' : 'active'
  const statusColor = status === 'active' ? 'var(--success, #22c55e)' : 'var(--danger, #ef4444)'

  return (
    <div data-testid="agent-drawer-fields" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <DrawerField label="Name" value={d.label || d.name || '—'} />
      <DrawerField label="client_id" value={d.client_id || d.clientId || node.id} mono />
      {d.jkt && <DrawerField label="DPoP jkt" value={d.jkt} mono />}

      {/* Status — color-coded (only place color is used per .impeccable.md) */}
      <div>
        <div style={drawerLabelStyle}>Status</div>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: statusColor }}>
          {status}
        </span>
      </div>

      {d.created_at && <DrawerField label="Created" value={new Date(d.created_at).toLocaleDateString()} />}
      {d.scopes && <DrawerField label="Scopes" value={Array.isArray(d.scopes) ? d.scopes.join(' ') : d.scopes} mono />}

      {/* Delegation context for this chain */}
      <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
        <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>Delegation context</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          <DrawerField label="Position in chain" value={`${chainPos === 1 ? '1st' : chainPos === 2 ? '2nd' : chainPos === 3 ? '3rd' : `${chainPos}th`} actor`} />
          <DrawerField label="Delegated by" value={prevHop?.data?.label || prevHop?.id || '—'} />
          <DrawerField label="Delegated to" value={nextHop?.data?.label || nextHop?.id || '—'} />
          <DrawerField label="Token type" value={tokenType} mono />
          <DrawerField label="Hop timestamp" value={hopTs} />
        </div>
      </div>
    </div>
  )
}

// ── Shared drawer primitives ──────────────────────────────────────────────────

const drawerLabelStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9,
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
  color: 'var(--fg-dim)',
  marginBottom: 2,
  opacity: 0.7,
}

const chipStyle: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 9.5,
  padding: '2px 6px',
  border: '1px solid var(--hairline)',
  borderRadius: 2,
  color: 'var(--fg-dim)',
  background: 'var(--surface-1)',
}

function DrawerField({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <div style={drawerLabelStyle}>{label}</div>
      <div style={{
        fontFamily: mono ? 'var(--font-mono)' : 'var(--font-body, var(--font-mono))',
        fontSize: mono ? 10 : 11,
        color: 'var(--fg)',
        wordBreak: 'break-all',
        lineHeight: 1.45,
      }}>{value || '—'}</div>
    </div>
  )
}

// ─── main canvas component ────────────────────────────────────────────────────

interface DelegationCanvasProps {
  rfNodes: any[]
  rfEdges: any[]
  onNodeClick?: (nodeId: string, nodeData: any) => void
  onEdgeClick?: (edgeData: any) => void
  height?: number
  fitView?: boolean
  loading?: boolean
  emptyMessage?: string
  emptyHint?: string
}

export function DelegationCanvas({
  rfNodes,
  rfEdges,
  onNodeClick,
  onEdgeClick,
  height = 520,
  fitView = true,
  loading = false,
  emptyMessage = 'No delegation chains.',
  emptyHint,
}: DelegationCanvasProps) {
  const [nodes, , onNodesChange] = useNodesState(rfNodes)
  const [edges, , onEdgesChange] = useEdgesState(rfEdges)
  // selectedNode drives the right-side drawer
  const [selectedNode, setSelectedNode] = useState<{ id: string; data: any } | null>(null)

  const handleNodeClick = (_: any, node: any) => {
    setSelectedNode({ id: node.id, data: node.data })
    onNodeClick?.(node.id, node.data)
  }

  if (loading) {
    return (
      <div style={{ width: '100%', height, position: 'relative', background: 'var(--surface-0)' }}>
        <style>{CANVAS_OVERRIDES}</style>
        <LoadingCanvas />
      </div>
    );
  }

  if (!rfNodes || rfNodes.length === 0) {
    return (
      <div style={{ width: '100%', height, position: 'relative', background: 'var(--surface-0)' }}>
        <style>{CANVAS_OVERRIDES}</style>
        <EmptyCanvas message={emptyMessage} hint={emptyHint} />
      </div>
    );
  }

  return (
    <div style={{ width: '100%', height, position: 'relative', background: 'var(--surface-0)' }}>
      <style>{CANVAS_OVERRIDES}</style>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        onNodeClick={handleNodeClick}
        onEdgeClick={(_, edge) => onEdgeClick?.(edge.data)}
        fitView={fitView}
        fitViewOptions={{ padding: 0.32 }}
        minZoom={0.15}
        maxZoom={3}
        defaultEdgeOptions={{
          type: 'animatedBezier',
          style: { stroke: 'var(--fg-dim)', strokeWidth: 1 },
          markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--fg-dim)', width: 8, height: 8 },
        }}
        proOptions={{ hideAttribution: true }}
        style={{ background: 'var(--surface-0)' }}
      >
        {/* Very faint dot grid — 20px spacing, 1px dots */}
        <Background
          variant={BackgroundVariant.Dots}
          gap={20}
          size={1}
          color="var(--hairline)"
          style={{ opacity: 0.5 }}
        />
        <Controls
          style={{ bottom: 14, left: 14, top: 'auto', right: 'auto' }}
          showInteractive={false}
        />
        <MiniMap
          style={{ top: 10, right: 10, bottom: 'auto', left: 'auto', width: 110, height: 64 }}
          maskColor="rgba(0,0,0,0.08)"
          nodeColor="var(--fg-dim)"
        />
      </ReactFlow>
      {/* Node detail drawer — slides in from right */}
      <NodeDrawer
        node={selectedNode}
        rfNodes={rfNodes}
        rfEdges={rfEdges}
        onClose={() => setSelectedNode(null)}
      />
    </div>
  )
}

// ─── wrapped with provider ────────────────────────────────────────────────────

export function DelegationCanvasWithProvider(props: DelegationCanvasProps) {
  return (
    <ReactFlowProvider>
      <DelegationCanvas {...props} />
    </ReactFlowProvider>
  )
}
