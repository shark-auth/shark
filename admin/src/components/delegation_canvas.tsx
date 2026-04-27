// @ts-nocheck
// delegation_canvas.tsx — shared Railway-style React Flow canvas
// Used by: delegation_chains.tsx (full graph) + agents_manage.tsx (ego graph)
// Visual contract: square corners, monochrome, .impeccable.md v3
// F7: initials avatar on human nodes, act-as badge on agent nodes,
//     "via token_exchange · <ts>" edge labels, active hop bolder

import React from 'react'
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
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'

// ─── global style overrides (monochrome, square corners) ─────────────────────
const CANVAS_OVERRIDES = `
  .react-flow__node { border-radius: 0 !important; }
  .react-flow__handle { border-radius: 0 !important; width: 6px !important; height: 6px !important; background: var(--fg-dim) !important; border: none !important; }
  .react-flow__edge-path { stroke: var(--fg-dim) !important; stroke-width: 1.5px !important; }
  .react-flow__edge.active-hop .react-flow__edge-path { stroke: var(--fg) !important; stroke-width: 2.5px !important; }
  .react-flow__edge.selected .react-flow__edge-path { stroke: var(--fg) !important; stroke-width: 2px !important; }
  .react-flow__controls { border-radius: 0 !important; border: 1px solid var(--hairline) !important; box-shadow: none !important; overflow: hidden; }
  .react-flow__controls-button { border-radius: 0 !important; background: var(--surface-1) !important; border-bottom: 1px solid var(--hairline) !important; color: var(--fg) !important; fill: var(--fg) !important; }
  .react-flow__controls-button:hover { background: var(--surface-2) !important; }
  .react-flow__minimap { border-radius: 0 !important; border: 1px solid var(--hairline) !important; box-shadow: none !important; }
  .react-flow__minimap-mask { fill: var(--surface-1) !important; opacity: 0.7 !important; }
  .react-flow__minimap-node { fill: var(--fg-dim) !important; }
  .react-flow__attribution { display: none !important; }
  .react-flow__edge-label { font-family: var(--font-mono) !important; font-size: 9px !important; fill: var(--fg-dim) !important; }
  .react-flow__edgelabel-renderer .edge-label-box { background: var(--surface-1) !important; border: 1px solid var(--hairline) !important; border-radius: 0 !important; padding: 1px 4px !important; font-family: var(--font-mono) !important; font-size: 9px !important; color: var(--fg-dim) !important; pointer-events: none !important; }
`

// ─── initials helper ─────────────────────────────────────────────────────────

/**
 * getInitials — extract 1-2 uppercase letters from an email or display name.
 * Examples:
 *   "alice@corp.com"   → "AL"
 *   "research-agent"   → "RA"
 *   "tool_agent_v2"    → "TA"
 *   "Bob Smith"        → "BS"
 */
export function getInitials(label: string): string {
  if (!label) return '?';
  // Email: take local part before @
  const local = label.includes('@') ? label.split('@')[0] : label;
  // Split on space, dash, underscore, dot
  const parts = local.split(/[\s\-_.]+/).filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase().slice(0, 2);
  }
  // Single word: take first 2 chars
  return local.slice(0, 2).toUpperCase();
}

// ─── custom node types ────────────────────────────────────────────────────────

/**
 * HumanNode — square 64×64, monochrome initials avatar circle + name label below.
 * Represents a human principal (user / originating subject).
 */
function HumanNode({ data, selected }: { data: any; selected?: boolean }) {
  const initials = getInitials(data.label || '');
  const borderColor = selected ? 'var(--fg)' : 'var(--fg-dim)';
  const borderWidth = selected ? 2 : 1;

  return (
    <div style={{
      width: 64,
      height: 64,
      background: selected ? 'var(--surface-1)' : 'var(--surface-0)',
      border: `${borderWidth}px solid ${borderColor}`,
      borderRadius: 4,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '6px 4px 4px',
      cursor: 'pointer',
      position: 'relative',
      transition: 'border-color 80ms',
      gap: 3,
    }}
      onMouseEnter={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg)';
      }}
      onMouseLeave={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -4 }} />
      <Handle type="source" position={Position.Right} style={{ right: -4 }} />

      {/* Initials avatar — circular per .impeccable.md (avatars only) */}
      <div style={{
        width: 26,
        height: 26,
        borderRadius: '50%',
        background: 'var(--surface-2)',
        border: '1px solid var(--hairline-strong)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        flexShrink: 0,
      }}>
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          fontWeight: 600,
          color: 'var(--fg)',
          letterSpacing: '0.04em',
          lineHeight: 1,
        }}>{initials}</span>
      </div>

      {/* Name label */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 8.5,
        color: 'var(--fg-dim)',
        maxWidth: 56,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1,
      }} title={data.label}>{data.label}</span>
    </div>
  )
}

/**
 * AgentNode — square 64×64, mono name + optional act-as count badge bottom-right.
 * Badge shown when data.actAsCount > 1 (chain length > 1).
 */
function AgentNode({ data, selected }: { data: any; selected?: boolean }) {
  const borderColor = selected ? 'var(--fg)' : 'var(--fg-dim)';
  const borderWidth = selected ? 2 : 1;

  return (
    <div style={{
      width: 64,
      height: 64,
      background: selected ? 'var(--surface-1)' : 'var(--surface-0)',
      border: `${borderWidth}px solid ${borderColor}`,
      borderRadius: 4,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '6px 4px 4px',
      cursor: 'pointer',
      position: 'relative',
      transition: 'border-color 80ms',
      gap: 3,
    }}
      onMouseEnter={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg)';
      }}
      onMouseLeave={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -4 }} />
      <Handle type="source" position={Position.Right} style={{ right: -4 }} />

      {/* Type label */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 7.5,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.07em',
        lineHeight: 1,
      }}>agent</span>

      {/* Agent name */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9.5,
        color: 'var(--fg)',
        fontWeight: 500,
        maxWidth: 56,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1.2,
      }} title={data.label}>{data.label}</span>

      {/* jkt hint */}
      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 7.5,
          color: 'var(--fg-dim)',
          lineHeight: 1,
        }}>jkt:{data.jkt.slice(0, 6)}</span>
      )}

      {/* act-as count badge — bottom-right, shown when chain length > 1 */}
      {data.actAsCount != null && data.actAsCount > 1 && (
        <div style={{
          position: 'absolute',
          bottom: 3,
          right: 3,
          minWidth: 14,
          height: 14,
          borderRadius: 3,
          background: 'var(--surface-2)',
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

function CenterAgentNode({ data, selected }: { data: any; selected?: boolean }) {
  // Same as AgentNode but with thicker border + larger — used for ego-graph center node
  return (
    <div style={{
      width: 72,
      height: 72,
      background: 'var(--surface-1)',
      border: `2px solid var(--fg)`,
      borderRadius: 4,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '6px 4px 4px',
      cursor: 'default',
      position: 'relative',
      gap: 3,
    }}>
      <Handle type="target" position={Position.Top} style={{ top: -4 }} />
      <Handle type="source" position={Position.Bottom} style={{ bottom: -4 }} />
      <Handle type="target" position={Position.Left} style={{ left: -4 }} />
      <Handle type="source" position={Position.Right} style={{ right: -4 }} />

      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 7.5,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.07em',
        lineHeight: 1,
      }}>agent</span>

      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9.5,
        color: 'var(--fg)',
        fontWeight: 600,
        maxWidth: 64,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
        lineHeight: 1.2,
      }} title={data.label}>{data.label}</span>

      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 7.5,
          color: 'var(--fg-dim)',
          lineHeight: 1,
        }}>jkt:{data.jkt.slice(0, 6)}</span>
      )}
    </div>
  )
}

// UserNode kept as alias for HumanNode (used by agents_manage ego graph)
const UserNode = HumanNode;

const nodeTypes = {
  agentNode: AgentNode,
  userNode: UserNode,
  humanNode: HumanNode,
  centerAgentNode: CenterAgentNode,
}

// ─── edge label renderer ──────────────────────────────────────────────────────

function EdgeLabel({ label }: { label: string }) {
  return (
    <div className="edge-label-box">{label}</div>
  )
}

// ─── public types ─────────────────────────────────────────────────────────────

export interface DCanvasNode {
  id: string
  label: string
  isUser: boolean
  isCenter?: boolean  // for ego-graph center
  layer: number
  slotInLayer: number
  jkt?: string
  meta?: any
  actAsCount?: number  // badge: how many act-as hops this node participates in
}

export interface DCanvasEdge {
  id: string
  from: string
  to: string
  timestamp?: string
  action?: string
  eventId?: string
  label?: string
  isActivHop?: boolean  // most-recent hop — rendered bolder
}

// ─── layout helper ────────────────────────────────────────────────────────────

// Left-to-right layered layout: x = layer * LAYER_GAP, y = slot * SLOT_GAP
const LAYER_GAP = 180
const SLOT_GAP = 100

export function toReactFlowNodes(nodes: DCanvasNode[]) {
  return nodes.map(n => ({
    id: n.id,
    type: n.isCenter ? 'centerAgentNode' : n.isUser ? 'humanNode' : 'agentNode',
    position: {
      x: n.layer * LAYER_GAP,
      y: n.slotInLayer * SLOT_GAP,
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
  }))
}

export function toReactFlowEdges(edges: DCanvasEdge[]) {
  return edges.map((e, idx) => {
    // Build "via token_exchange · HH:MM" label
    const ts = e.timestamp
      ? new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      : ''
    const baseLabel = ts ? `via token_exchange · ${ts}` : 'via token_exchange'
    const edgeLabel = e.label || baseLabel

    // Active hop (most recent = last edge in sequence) gets bolder stroke
    const isActive = e.isActivHop === true
    const strokeColor = isActive ? 'var(--fg)' : 'var(--fg-dim)'
    const strokeWidth = isActive ? 2.5 : 1.5

    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      className: isActive ? 'active-hop' : '',
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: isActive ? 12 : 10,
        height: isActive ? 12 : 10,
        color: strokeColor,
      },
      style: { stroke: strokeColor, strokeWidth },
      label: edgeLabel || undefined,
      labelStyle: {
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        fill: isActive ? 'var(--fg)' : 'var(--fg-dim)',
        fontWeight: isActive ? 600 : 400,
      },
      labelBgStyle: {
        fill: 'var(--surface-1)',
        fillOpacity: 1,
      },
      data: { eventId: e.eventId },
    }
  })
}

// ─── ego-graph layout (inbound above, center middle, outbound below) ──────────

export function toEgoLayout(
  centerNode: DCanvasNode,
  inbound: DCanvasNode[],
  outbound: DCanvasNode[],
  inboundEdges: DCanvasEdge[],
  outboundEdges: DCanvasEdge[],
) {
  const NODE_W = 72
  const ROW_GAP = 120
  const COL_GAP = 160

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
    const edgeLabel = e.label || (ts ? `via token_exchange · ${ts}` : 'via token_exchange')
    const strokeColor = isActive ? 'var(--fg)' : 'var(--fg-dim)'
    const strokeWidth = isActive ? 2.5 : 1.5
    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      className: isActive ? 'active-hop' : '',
      markerEnd: { type: MarkerType.ArrowClosed, width: isActive ? 12 : 10, height: isActive ? 12 : 10, color: strokeColor },
      style: { stroke: strokeColor, strokeWidth },
      label: edgeLabel,
      labelStyle: { fontFamily: 'var(--font-mono)', fontSize: 9, fill: isActive ? 'var(--fg)' : 'var(--fg-dim)', fontWeight: isActive ? 600 : 400 },
      labelBgStyle: { fill: 'var(--surface-1)', fillOpacity: 1 },
      data: { eventId: e.eventId },
    }
  }

  const rfEdges = [
    ...inboundEdges.map((e, i) => makeEdge(e, i === inboundEdges.length - 1)),
    ...outboundEdges.map((e, i) => makeEdge(e, i === outboundEdges.length - 1)),
  ]

  return { rfNodes, rfEdges }
}

// ─── main canvas component ────────────────────────────────────────────────────

interface DelegationCanvasProps {
  /** pre-built react-flow nodes (use toReactFlowNodes or toEgoLayout) */
  rfNodes: any[]
  /** pre-built react-flow edges (use toReactFlowEdges or toEgoLayout) */
  rfEdges: any[]
  onNodeClick?: (nodeId: string, nodeData: any) => void
  onEdgeClick?: (edgeData: any) => void
  height?: number
  fitView?: boolean
}

export function DelegationCanvas({
  rfNodes,
  rfEdges,
  onNodeClick,
  onEdgeClick,
  height = 520,
  fitView = true,
}: DelegationCanvasProps) {
  const [nodes, , onNodesChange] = useNodesState(rfNodes)
  const [edges, , onEdgesChange] = useEdgesState(rfEdges)

  return (
    <div style={{ width: '100%', height, position: 'relative', background: 'var(--surface-0)' }}>
      <style>{CANVAS_OVERRIDES}</style>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => onNodeClick?.(node.id, node.data)}
        onEdgeClick={(_, edge) => onEdgeClick?.(edge.data)}
        fitView={fitView}
        fitViewOptions={{ padding: 0.25 }}
        minZoom={0.2}
        maxZoom={3}
        defaultEdgeOptions={{
          style: { stroke: 'var(--fg-dim)', strokeWidth: 1.5 },
          markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--fg-dim)' },
        }}
        proOptions={{ hideAttribution: true }}
        style={{ background: 'var(--surface-0)' }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={18}
          size={1.5}
          color="var(--hairline)"
        />
        <Controls
          style={{ bottom: 16, left: 16, top: 'auto', right: 'auto' }}
          showInteractive={false}
        />
        <MiniMap
          style={{ top: 12, right: 12, bottom: 'auto', left: 'auto', width: 120, height: 70 }}
          maskColor="rgba(0,0,0,0.06)"
          nodeColor="var(--fg-dim)"
        />
      </ReactFlow>
    </div>
  )
}

// ─── wrapped with provider (use this at call sites) ──────────────────────────

export function DelegationCanvasWithProvider(props: DelegationCanvasProps) {
  return (
    <ReactFlowProvider>
      <DelegationCanvas {...props} />
    </ReactFlowProvider>
  )
}
