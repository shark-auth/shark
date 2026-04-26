// @ts-nocheck
// delegation_canvas.tsx — shared Railway-style React Flow canvas
// Used by: delegation_chains.tsx (full graph) + agents_manage.tsx (ego graph)
// Visual contract: square corners, monochrome, .impeccable.md v3

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

// ─── custom node types ────────────────────────────────────────────────────────

function AgentNode({ data, selected }: { data: any; selected?: boolean }) {
  return (
    <div style={{
      width: 160,
      height: 60,
      background: selected ? 'var(--surface-1)' : 'var(--surface-0)',
      border: `${selected ? 2 : 1}px solid ${selected ? 'var(--fg)' : 'var(--fg-dim)'}`,
      borderRadius: 0,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '4px 8px',
      cursor: 'pointer',
      transition: 'border-color 80ms, border-width 80ms',
    }}
      onMouseEnter={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg)';
      }}
      onMouseLeave={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -4 }}/>
      <Handle type="source" position={Position.Right} style={{ right: -4 }}/>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.06em',
        marginBottom: 2,
      }}>agent</span>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--fg)',
        fontWeight: 500,
        maxWidth: 144,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
      }} title={data.label}>{data.label}</span>
      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          color: 'var(--fg-dim)',
          marginTop: 1,
        }}>jkt:{data.jkt.slice(0, 8)}</span>
      )}
    </div>
  )
}

function UserNode({ data, selected }: { data: any; selected?: boolean }) {
  return (
    <div style={{
      width: 160,
      height: 60,
      background: selected ? 'var(--surface-1)' : 'var(--surface-0)',
      border: `${selected ? 2 : 1}px solid ${selected ? 'var(--fg)' : 'var(--fg-dim)'}`,
      borderRadius: 0,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '4px 8px',
      cursor: 'pointer',
      transition: 'border-color 80ms',
    }}
      onMouseEnter={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg)';
      }}
      onMouseLeave={e => {
        if (!selected) (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -4 }}/>
      <Handle type="source" position={Position.Right} style={{ right: -4 }}/>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.06em',
        marginBottom: 2,
      }}>user</span>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--fg)',
        fontWeight: 500,
        maxWidth: 144,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
      }} title={data.label}>{data.label}</span>
    </div>
  )
}

function CenterAgentNode({ data, selected }: { data: any; selected?: boolean }) {
  // Same as AgentNode but with a thicker border — used for the ego-graph center node
  return (
    <div style={{
      width: 160,
      height: 64,
      background: 'var(--surface-1)',
      border: `2px solid var(--fg)`,
      borderRadius: 0,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '4px 8px',
      cursor: 'default',
    }}>
      <Handle type="target" position={Position.Top} style={{ top: -4 }}/>
      <Handle type="source" position={Position.Bottom} style={{ bottom: -4 }}/>
      <Handle type="target" position={Position.Left} style={{ left: -4 }}/>
      <Handle type="source" position={Position.Right} style={{ right: -4 }}/>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        color: 'var(--fg-dim)',
        textTransform: 'uppercase',
        letterSpacing: '0.06em',
        marginBottom: 2,
      }}>agent (current)</span>
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 11,
        color: 'var(--fg)',
        fontWeight: 600,
        maxWidth: 144,
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        whiteSpace: 'nowrap',
        textAlign: 'center',
      }} title={data.label}>{data.label}</span>
      {data.jkt && (
        <span style={{
          fontFamily: 'var(--font-mono)',
          fontSize: 9,
          color: 'var(--fg-dim)',
          marginTop: 1,
        }}>jkt:{data.jkt.slice(0, 8)}</span>
      )}
    </div>
  )
}

const nodeTypes = {
  agentNode: AgentNode,
  userNode: UserNode,
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
}

export interface DCanvasEdge {
  id: string
  from: string
  to: string
  timestamp?: string
  action?: string
  eventId?: string
  label?: string
}

// ─── layout helper ────────────────────────────────────────────────────────────

// Left-to-right layered layout: x = layer * LAYER_GAP, y = slot * SLOT_GAP
const LAYER_GAP = 240
const SLOT_GAP = 100

export function toReactFlowNodes(nodes: DCanvasNode[]) {
  return nodes.map(n => ({
    id: n.id,
    type: n.isCenter ? 'centerAgentNode' : n.isUser ? 'userNode' : 'agentNode',
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
    },
    selected: false,
  }))
}

export function toReactFlowEdges(edges: DCanvasEdge[]) {
  return edges.map(e => {
    const ts = e.timestamp ? new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : ''
    const edgeLabel = e.label || [ts, e.action].filter(Boolean).join(' ')
    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 10,
        height: 10,
        color: 'var(--fg-dim)',
      },
      style: { stroke: 'var(--fg-dim)', strokeWidth: 1.5 },
      label: edgeLabel || undefined,
      labelStyle: {
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        fill: 'var(--fg-dim)',
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
  const NODE_W = 160
  const ROW_GAP = 120
  const COL_GAP = 180

  const layoutRow = (items: DCanvasNode[], y: number) =>
    items.map((n, i) => ({
      id: n.id,
      type: n.isUser ? 'userNode' : 'agentNode',
      position: {
        x: (i - (items.length - 1) / 2) * COL_GAP,
        y,
      },
      data: { label: n.label, jkt: n.jkt, isUser: n.isUser },
    }))

  const rfNodes = [
    ...layoutRow(inbound, 0),
    {
      id: centerNode.id,
      type: 'centerAgentNode',
      position: { x: -NODE_W / 2, y: ROW_GAP },
      data: { label: centerNode.label, jkt: centerNode.jkt },
    },
    ...layoutRow(outbound, ROW_GAP * 2),
  ]

  const rfEdges = [
    ...inboundEdges.map(e => ({
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      markerEnd: { type: MarkerType.ArrowClosed, width: 10, height: 10, color: 'var(--fg-dim)' },
      style: { stroke: 'var(--fg-dim)', strokeWidth: 1.5 },
      label: e.label,
      labelStyle: { fontFamily: 'var(--font-mono)', fontSize: 9, fill: 'var(--fg-dim)' },
      labelBgStyle: { fill: 'var(--surface-1)', fillOpacity: 1 },
      data: { eventId: e.eventId },
    })),
    ...outboundEdges.map(e => ({
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'default',
      markerEnd: { type: MarkerType.ArrowClosed, width: 10, height: 10, color: 'var(--fg-dim)' },
      style: { stroke: 'var(--fg-dim)', strokeWidth: 1.5 },
      label: e.label,
      labelStyle: { fontFamily: 'var(--font-mono)', fontSize: 9, fill: 'var(--fg-dim)' },
      labelBgStyle: { fill: 'var(--surface-1)', fillOpacity: 1 },
      data: { eventId: e.eventId },
    })),
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
        fitViewOptions={{ padding: 0.2 }}
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
