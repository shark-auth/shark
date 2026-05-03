// @ts-nocheck
// delegation_canvas.tsx — shared Railway-style React Flow canvas
// Used by: delegation_chains.tsx (full graph) + agents_manage.tsx (ego graph)
// Visual contract: square chrome, monochrome, .impeccable.md v3
// F7: initials avatar on human nodes, act-as badge on agent nodes,
//     "via token_exchange · <ts>" edge tooltip (hover-only), active hop animated stroke

import React, { useState } from 'react'
import dagre from 'dagre'
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  useReactFlow,
  MarkerType,
  Handle,
  Position,
  getBezierPath,
  EdgeLabelRenderer,
  BaseEdge,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { API } from './api'
import { useToast } from './toast'

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

  /* Active-hop edge: dashed march animation — strokeWidth: 1.5 (bolder than hairline) */
  .react-flow__edge.active-hop .react-flow__edge-path {
    stroke: var(--fg) !important;
    stroke-width: 1.5px !important;
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
 * stripLanePrefix — buildGraph in delegation_chains scopes node ids to a lane
 * (lane0__sub, lane1__sub) so distinct chains stay disjoint on the aggregate
 * canvas. The lane is a render-only concern; never show it to operators or
 * pass it to APIs.
 */
export function stripLanePrefix(id: string): string {
  if (!id) return id
  return id.replace(/^lane\d+__/, '')
}

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

  const revoked = data?.revoked === true;
  const isActive = !revoked && data?.isActive;
  // Full label contains "via token_exchange · HH:MM" — used for hover tooltip
  const label = data?.label;
  const scopeFrom: string[] | undefined = data?.scopeFrom;
  const scopeTo: string[] | undefined = data?.scopeTo;

  // Static pill shows "acts-as · HH:MM" (user-friendly, no RFC jargon)
  const pillLabel = React.useMemo(() => {
    if (revoked) return 'revoked';
    if (!label) return 'acts-as';
    // Extract time portion after last " · "
    const parts = label.split(' · ');
    const ts = parts.length > 1 ? parts[parts.length - 1] : '';
    return ts ? `acts-as · ${ts}` : 'acts-as';
  }, [label, revoked]);
  // Hover tooltip shows full RFC detail (jargon-free on pill, technical here)
  const tooltipLabel = React.useMemo(() => {
    if (!label) return '';
    return label.replace(/^via token_exchange/, 'RFC 8693 token_exchange');
  }, [label]);

  // Scope delta chip — uses granted_scope/subject_scope from audit metadata.
  const scopeChip = React.useMemo(() => {
    if (scopeFrom && scopeFrom.length > 0 && scopeTo) {
      const diff = scopeFrom.length - scopeTo.length;
      if (diff > 0) return { text: `${scopeFrom.length}→${scopeTo.length} scopes`, color: 'var(--danger, #ef4444)', dropped: scopeFrom.filter(s => !scopeTo.includes(s)) };
      return { text: `=${scopeTo.length} scopes`, color: 'var(--fg-dim)', dropped: [] };
    }
    if (scopeTo && scopeTo.length > 0) {
      return { text: `=${scopeTo.length} scopes`, color: 'var(--fg-dim)', dropped: [] };
    }
    return null;
  }, [scopeFrom, scopeTo]);

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
          <div
            className={`edge-label-pill${isActive ? ' active' : ''}`}
            style={revoked ? {
              color: 'var(--danger, #ef4444)',
              borderColor: 'var(--danger, #ef4444)',
              background: 'color-mix(in oklch, var(--danger, #ef4444) 10%, var(--surface-0))',
            } : undefined}
          >
            {pillLabel}
          </div>
          {/* Scope delta chip — visible when scope data is available */}
          {scopeChip && (
            <div style={{
              fontFamily: 'ui-monospace, monospace',
              fontSize: 7.5,
              color: scopeChip.color,
              opacity: 0.8,
              lineHeight: 1,
            }} title={scopeChip.dropped.length > 0 ? `Dropped: ${scopeChip.dropped.join(', ')}` : undefined}>
              {scopeChip.text}
            </div>
          )}
          {/* Hover tooltip with full RFC detail + dropped scopes */}
          {tooltipLabel && (
            <div className={`edge-tooltip${hovered ? ' visible' : ''}${isActive ? ' active' : ''}`}
              style={{ position: 'static', transform: 'none', marginTop: 2 }}
            >
              {tooltipLabel}
              {scopeChip && scopeChip.dropped.length > 0 && (
                <div style={{ marginTop: 2, color: 'var(--danger, #ef4444)', fontSize: 8 }}>
                  dropped: {scopeChip.dropped.map(s => <span key={s} style={{ textDecoration: 'line-through', marginRight: 3 }}>{s}</span>)}
                </div>
              )}
            </div>
          )}
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

// ─── custom node types ────────────────────────────────────────────────────────

/**
 * HumanNode — circular (borderRadius 50%) with accent border, initials chip 32px.
 * Distinguished from AgentNode (square) at a glance.
 */
function HumanNode({ data, selected }: { data: any; selected?: boolean }) {
  const initials = getInitials(data.label || '');

  return (
    <div style={{
      width: 80,
      height: 80,
      background: selected ? 'var(--surface-2)' : 'var(--surface-1)',
      border: `2px solid ${selected ? 'var(--fg)' : 'var(--accent, #5eead4)'}`,
      borderRadius: '50%',
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
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 12px rgba(0,0,0,0.38)';
        }
      }}
      onMouseLeave={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--accent, #5eead4)';
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
 * AgentNode — 80×80, square, 2% teal tint bg, </> glyph top-left, bold AGENT label 9px.
 * Visually distinct from HumanNode (circular accent border).
 */
function AgentNode({ data, selected }: { data: any; selected?: boolean }) {
  const revoked = data.revoked === true;
  return (
    <div style={{
      width: 80,
      height: 80,
      background: selected ? 'var(--surface-2)' : 'rgba(94, 234, 212, 0.02)',
      border: `1px solid ${selected ? 'var(--fg)' : revoked ? 'var(--danger, #ef4444)' : 'var(--hairline-strong)'}`,
      borderRadius: 6,
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '8px 6px',
      cursor: 'pointer',
      position: 'relative',
      transition: 'border-color 100ms, box-shadow 100ms, background 100ms, filter 200ms ease',
      gap: 4,
      boxShadow: selected
        ? '0 0 0 1px var(--fg), 0 4px 16px rgba(0,0,0,0.45)'
        : '0 2px 8px rgba(0,0,0,0.28)',
      animation: 'fade-in 160ms ease-out',
      // Revoked: desaturate + dim. Border goes red. REVOKED stripe overlays.
      filter: revoked ? 'grayscale(1) brightness(0.7)' : 'none',
      opacity: revoked ? 0.78 : 1,
    }}
      onMouseEnter={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = 'var(--fg-dim)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 12px rgba(0,0,0,0.38)';
        }
      }}
      onMouseLeave={e => {
        if (!selected) {
          (e.currentTarget as HTMLElement).style.borderColor = revoked ? 'var(--danger, #ef4444)' : 'var(--hairline-strong)';
          (e.currentTarget as HTMLElement).style.boxShadow = '0 2px 8px rgba(0,0,0,0.28)';
        }
      }}
    >
      <Handle type="target" position={Position.Left} style={{ left: -3 }} />
      <Handle type="source" position={Position.Right} style={{ right: -3 }} />

      {/* REVOKED diagonal stripe — emotionally legible kill mark */}
      {revoked && (
        <div style={{
          position: 'absolute',
          top: 0, left: 0, right: 0, bottom: 0,
          pointerEvents: 'none',
          overflow: 'hidden',
          borderRadius: 6,
        }}>
          <div style={{
            position: 'absolute',
            top: '50%', left: '-15%',
            width: '130%',
            transform: 'translateY(-50%) rotate(-18deg)',
            background: 'rgba(239, 68, 68, 0.85)',
            color: '#fff',
            fontFamily: 'var(--font-mono)',
            fontSize: 9,
            fontWeight: 700,
            letterSpacing: '0.18em',
            textAlign: 'center',
            padding: '2px 0',
            textTransform: 'uppercase',
            boxShadow: '0 1px 3px rgba(0,0,0,0.4)',
          }}>revoked</div>
        </div>
      )}

      {/* </> glyph — top-left corner indicator */}
      <span style={{
        position: 'absolute',
        top: 4,
        left: 5,
        fontFamily: 'ui-monospace, monospace',
        fontSize: 8,
        color: 'var(--fg-dim)',
        opacity: 0.45,
        lineHeight: 1,
        userSelect: 'none',
      }}>{'</>'}</span>

      {/* "AGENT" type label — 9px bold */}
      <span style={{
        fontFamily: 'var(--font-mono)',
        fontSize: 9,
        fontWeight: 700,
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

      {/* DPoP-bound shield icon — replaces jkt text */}
      {data.jkt && (
        <div title="DPoP-bound" style={{ lineHeight: 1, display: 'flex', alignItems: 'center' }}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--accent, #5eead4)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.75 }}>
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            <polyline points="9 12 11 14 15 10"/>
          </svg>
        </div>
      )}

      {/* Chain position chip — bottom-right: "2/4" style */}
      {data.chainPos != null && data.chainTotal != null && data.chainTotal > 1 && (
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
          }}>{data.chainPos}/{data.chainTotal}</span>
        </div>
      )}
    </div>
  )
}

/**
 * CenterAgentNode — ego-graph focal node, slightly larger, full-brightness border.
 */
function CenterAgentNode({ data, selected }: { data: any; selected?: boolean }) {
  const revoked = data.revoked === true;
  return (
    <div style={{
      width: 88,
      height: 88,
      background: 'var(--surface-2)',
      border: `1.5px solid ${revoked ? 'var(--danger, #ef4444)' : 'var(--fg)'}`,
      borderRadius: 8,
      filter: revoked ? 'grayscale(1) brightness(0.7)' : 'none',
      opacity: revoked ? 0.85 : 1,
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

      {revoked && (
        <div style={{
          position: 'absolute', inset: 0, pointerEvents: 'none',
          overflow: 'hidden', borderRadius: 8,
        }}>
          <div style={{
            position: 'absolute', top: '50%', left: '-15%', width: '130%',
            transform: 'translateY(-50%) rotate(-18deg)',
            background: 'rgba(239, 68, 68, 0.85)', color: '#fff',
            fontFamily: 'var(--font-mono)', fontSize: 10, fontWeight: 700,
            letterSpacing: '0.2em', textAlign: 'center', padding: '3px 0',
            textTransform: 'uppercase', boxShadow: '0 1px 3px rgba(0,0,0,0.4)',
          }}>revoked</div>
        </div>
      )}

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
        <div title="DPoP-bound" style={{ lineHeight: 1, display: 'flex', alignItems: 'center' }}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="var(--accent, #5eead4)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ opacity: 0.75 }}>
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            <polyline points="9 12 11 14 15 10"/>
          </svg>
        </div>
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
  actAsCount?: number    // legacy — ignored; use chainPos/chainTotal
  chainPos?: number      // 1-based position in chain (1 = first actor)
  chainTotal?: number    // total nodes in this chain
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
  revoked?: boolean
  revoked_at?: string
  grantId?: string
  scopeFrom?: string[]   // scope at source hop (subject_scope from audit metadata)
  scopeTo?: string[]     // scope at target hop (granted_scope from audit metadata)
}

// ─── layout helper ────────────────────────────────────────────────────────────

// Generous spacing: nodes have more breathing room
const LAYER_GAP = 200
const SLOT_GAP = 120
// Swim-lane vertical offset per chain (240px bands)
const LANE_GAP = 240

export function toReactFlowNodes(nodes: DCanvasNode[]) {
  const rfNodes: any[] = []
  const NODE_W = 80
  const NODE_H = 80

  // Group nodes by lane (each chain = one horizontal band)
  const laneGroups = new Map<number, DCanvasNode[]>()
  for (const n of nodes) {
    const lane = n.lane ?? 0
    if (!laneGroups.has(lane)) laneGroups.set(lane, [])
    laneGroups.get(lane)!.push(n)
  }

  // Sort lanes to ensure consistent vertical stacking
  const sortedLanes = Array.from(laneGroups.keys()).sort((a, b) => a - b)

  // Track cumulative y offset (dagre outputs absolute coords; we shift per lane)
  let cumulativeY = 0
  const labeledLanes = new Set<number>()

  // We need edges to run dagre — they're not passed here, so we reconstruct
  // topology from layer/slot (layer = graph depth, adjacent layers = edges).
  // Build a simple chain: nodes sorted by layer within each lane.

  for (const lane of sortedLanes) {
    const laneNodes = laneGroups.get(lane)!

    const g = new dagre.graphlib.Graph()
    g.setGraph({ rankdir: 'LR', nodesep: 60, ranksep: 160 })
    g.setDefaultEdgeLabel(() => ({}))

    // Register all nodes
    laneNodes.forEach(n => g.setNode(n.id, { width: NODE_W, height: NODE_H }))

    // Add edges based on layer adjacency (layer i → layer i+1 within lane)
    const byLayer = new Map<number, DCanvasNode[]>()
    for (const n of laneNodes) {
      if (!byLayer.has(n.layer)) byLayer.set(n.layer, [])
      byLayer.get(n.layer)!.push(n)
    }
    const layers = Array.from(byLayer.keys()).sort((a, b) => a - b)
    for (let li = 0; li < layers.length - 1; li++) {
      const fromLayer = byLayer.get(layers[li])!
      const toLayer = byLayer.get(layers[li + 1])!
      // Connect each node in this layer to the next layer's node
      // (simple chain — enough for dagre to rank correctly)
      const pairs = Math.max(fromLayer.length, toLayer.length)
      for (let pi = 0; pi < pairs; pi++) {
        const src = fromLayer[Math.min(pi, fromLayer.length - 1)]
        const tgt = toLayer[Math.min(pi, toLayer.length - 1)]
        g.setEdge(src.id, tgt.id)
      }
    }

    dagre.layout(g)

    // Find bounding box height for this lane to compute next lane offset
    let laneMinY = Infinity
    let laneMaxY = -Infinity
    laneNodes.forEach(n => {
      const pos = g.node(n.id)
      if (pos) {
        laneMinY = Math.min(laneMinY, pos.y - NODE_H / 2)
        laneMaxY = Math.max(laneMaxY, pos.y + NODE_H / 2)
      }
    })
    if (laneMinY === Infinity) { laneMinY = 0; laneMaxY = NODE_H }

    laneNodes.forEach(n => {
      const pos = g.node(n.id)
      if (!pos) return
      rfNodes.push({
        id: n.id,
        type: n.isCenter ? 'centerAgentNode' : n.isUser ? 'humanNode' : 'agentNode',
        position: {
          x: pos.x - NODE_W / 2,
          y: cumulativeY + (pos.y - laneMinY),
        },
        data: {
          label: n.label,
          jkt: n.jkt,
          isUser: n.isUser,
          isCenter: n.isCenter,
          meta: n.meta,
          chainPos: n.chainPos,
          chainTotal: n.chainTotal,
        },
        selected: false,
      })
    })

    // Gutter label: pinned left at mid-lane y
    if (lane != null && laneNodes[0]?.laneLabel && !labeledLanes.has(lane)) {
      labeledLanes.add(lane)
      const laneHeight = laneMaxY - laneMinY
      rfNodes.push({
        id: `__lane_label_${lane}`,
        type: 'laneLabel',
        position: { x: -170, y: cumulativeY + laneHeight / 2 - 10 },
        data: { label: laneNodes[0].laneLabel },
        selectable: false,
        draggable: false,
      })
    }

    cumulativeY += (laneMaxY - laneMinY) + LANE_GAP
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

    const revoked = e.revoked === true
    const isActive = !revoked && e.isActivHop === true
    const strokeColor = revoked ? 'var(--danger, #ef4444)' : isActive ? 'var(--fg)' : 'var(--fg-dim)'
    // Active hop: strokeWidth 1.5 (bolder than hairline 1px)
    const strokeWidth = isActive ? 1.5 : 1

    const scopeFrom = e.scopeFrom
    const scopeTo = e.scopeTo

    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'animatedBezier',
      className: revoked ? 'revoked-hop' : isActive ? 'active-hop' : '',
      // P0-6: arrowheads 14/18 for legibility at fitView zoom
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: isActive ? 18 : 14,
        height: isActive ? 18 : 14,
        color: strokeColor,
      },
      style: {
        stroke: strokeColor,
        // strokeWidth: 1.5 for active-hop (bolder), 1 for historical hairline
        strokeWidth,
        opacity: revoked ? 0.58 : 1,
        ...(revoked ? { strokeDasharray: '2 5' } : {}),
        ...(isActive ? {
          strokeDasharray: '6 4',
          animation: 'dash-march 800ms linear infinite',
        } : {}),
      },
      data: {
        label: edgeLabel,
        eventId: e.eventId,
        grantId: e.grantId,
        isActive,
        revoked,
        revoked_at: e.revoked_at,
        scopeFrom,
        scopeTo,
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
      data: { label: n.label, jkt: n.jkt, isUser: n.isUser },
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
    const revoked = e.revoked === true
    const strokeColor = revoked ? 'var(--danger, #ef4444)' : isActive ? 'var(--fg)' : 'var(--fg-dim)'
    // Active hop: strokeWidth 1.5 (bolder than hairline 1px)
    const strokeWidth = isActive ? 1.5 : 1
    return {
      id: e.id,
      source: e.from,
      target: e.to,
      type: 'animatedBezier',
      className: revoked ? 'revoked-hop' : isActive ? 'active-hop' : '',
      // P0-6: arrowheads 14/18 for legibility at fitView zoom
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: isActive ? 18 : 14,
        height: isActive ? 18 : 14,
        color: strokeColor,
      },
      style: {
        stroke: strokeColor,
        strokeWidth,
        opacity: revoked ? 0.58 : 1,
        ...(revoked ? { strokeDasharray: '2 5' } : {}),
        ...(isActive ? { strokeDasharray: '6 4', animation: 'dash-march 800ms linear infinite' } : {}),
      },
      data: { label: edgeLabel, eventId: e.eventId, grantId: e.grantId, isActive: !revoked && isActive, revoked, revoked_at: e.revoked_at, scopeFrom: e.scopeFrom, scopeTo: e.scopeTo },
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

function NodeDrawer({ node, rfNodes, rfEdges, onClose, onNavigate, onCanvasRefresh, markRevoked }: {
  node: { id: string; data: any } | null
  rfNodes: any[]
  rfEdges: any[]
  onClose: () => void
  onNavigate?: (target: 'agent' | 'user', id: string) => void
  onCanvasRefresh?: () => void
  markRevoked?: (ids: string[]) => void
}) {
  // Close on Escape
  React.useEffect(() => {
    if (!node) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [node, onClose])

  if (!node) return null

  const liveNode = rfNodes?.find(n => n.id === node.id) || node
  const d = liveNode.data || {}
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
  // Scope: derive from inbound edge data
  const scopeFrom: string[] | undefined = inEdges[0]?.data?.scopeFrom
  const scopeTo: string[] | undefined = inEdges[0]?.data?.scopeTo

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
            {isHuman ? 'human · ' : 'agent · '}{stripLanePrefix(node.id).slice(0, 12)}
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
            <HumanDrawerFields node={liveNode} onNavigate={onNavigate} />
          ) : (
            // ── Agent fields ──────────────────────────────────────────────────
            <AgentDrawerFields
              node={liveNode}
              chainPos={chainPos}
              prevHop={prevHop}
              nextHop={nextHop}
              tokenType={tokenType}
              hopTs={hopTs}
              scopeFrom={scopeFrom}
              scopeTo={scopeTo}
              rfEdges={rfEdges}
              onNavigate={onNavigate}
              onClose={onClose}
              onCanvasRefresh={onCanvasRefresh}
              markRevoked={markRevoked}
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

// ── Edge drawer ───────────────────────────────────────────────────────────────
// Slides in for delegation edges. Shows from/to, hop timestamp, token type,
// scope delta (granted_scope vs subject_scope), and audit-event link.
// Backend doesn't yet expose grant-level fields (max_hops, expires_at, revoked,
// grant_id) on token-exchange events; surfaced as "—" until /api/v1/may-act
// joins are added (see follow-up note in coverage matrix).
// Audit/event loaders should follow next_cursor when paginating backend results.

function EdgeDrawer({ edge, rfNodes, onClose, onAuditClick, onCanvasRefresh }: {
  edge: { id: string; source: string; target: string; data: any } | null
  rfNodes: any[]
  onClose: () => void
  onAuditClick?: (eventId: string, grantId?: string) => void
  onCanvasRefresh?: () => void
}) {
  React.useEffect(() => {
    if (!edge) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [edge, onClose])

  // Fetch matching may_act grants for this edge. Pick most recent un-revoked,
  // else most recent overall. Falls back to "no grant correlated" when empty.
  const [grant, setGrant] = React.useState<any>(null)
  const [grantLoading, setGrantLoading] = React.useState(false)
  const [revokeBusy, setRevokeBusy] = React.useState(false)
  const toast = useToast()

  React.useEffect(() => {
    if (!edge) { setGrant(null); return }
    let cancelled = false
    setGrantLoading(true)
    const source = stripLanePrefix(edge.source)
    const target = stripLanePrefix(edge.target)
    const qs = `from_id=${encodeURIComponent(source)}&to_id=${encodeURIComponent(target)}&include_revoked=true`
    API.get(`/admin/may-act?${qs}`)
      .then((d: any) => {
        if (cancelled) return
        const list: any[] = d?.grants || []
        const live = list.find(g => !g.revoked_at)
        setGrant(live || list[0] || null)
      })
      .catch(() => { if (!cancelled) setGrant(null) })
      .finally(() => { if (!cancelled) setGrantLoading(false) })
    return () => { cancelled = true }
  }, [edge?.source, edge?.target])

  if (!edge) return null

  const d = edge.data || {}
  const fromNode = rfNodes.find(n => n.id === edge.source)
  const toNode = rfNodes.find(n => n.id === edge.target)
  const fromLabel = fromNode?.data?.label || edge.source
  const toLabel = toNode?.data?.label || edge.target
  const tsRaw = (d.label || '').match(/· (.+)$/)?.[1] || ''
  const isRevoked = d.revoked === true || !!grant?.revoked_at
  const isActive = !isRevoked && !!d.isActive
  const tokenType = (d.label || '').includes('token_exchange') ? 'token_exchange' : '—'

  const scopeFrom: string[] | undefined = d.scopeFrom
  const scopeTo: string[] | undefined = d.scopeTo
  const dropped = scopeFrom && scopeTo ? scopeFrom.filter(s => !scopeTo.includes(s)) : []

  const auditHref = d.eventId ? `/admin/audit?q=${encodeURIComponent(d.eventId)}` : null

  return (
    <>
      <style>{DRAWER_SLIDE_IN}</style>
      <div
        onClick={onClose}
        style={{ position: 'absolute', inset: 0, zIndex: 20, background: 'rgba(0,0,0,0.18)' }}
        data-testid="edge-drawer-backdrop"
      />
      <div
        onClick={e => e.stopPropagation()}
        data-testid="edge-drawer"
        style={{
          position: 'absolute', top: 0, right: 0, bottom: 0,
          width: 400, maxWidth: '92%', zIndex: 21,
          background: 'var(--surface-0)',
          borderLeft: '1px solid var(--hairline)',
          display: 'flex', flexDirection: 'column',
          animation: 'drawerSlideIn 200ms ease-out',
          overflowY: 'auto',
        }}
      >
        <div style={{
          padding: '12px 14px 10px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          flexShrink: 0,
        }}>
          <span style={{
            fontFamily: 'var(--font-mono)', fontSize: 10,
            textTransform: 'uppercase', letterSpacing: '0.08em', color: 'var(--fg-dim)',
          }}>
            delegation · {isRevoked ? 'revoked' : isActive ? 'active hop' : 'historical'}
          </span>
          <button
            onClick={onClose}
            data-testid="edge-drawer-close"
            style={{
              background: 'none', border: 'none', cursor: 'pointer',
              color: 'var(--fg-dim)', fontSize: 16, lineHeight: 1,
              padding: '2px 4px', fontFamily: 'var(--font-mono)',
            }}
            title="Close (Esc)"
          >✕</button>
        </div>

        <div style={{ flex: 1, padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <DrawerField label="From" value={fromLabel} />
          <DrawerField label="From id" value={stripLanePrefix(edge.source)} mono />
          <DrawerField label="To" value={toLabel} />
          <DrawerField label="To id" value={stripLanePrefix(edge.target)} mono />
          <DrawerField label="Token type" value={tokenType} mono />
          <DrawerField label="Hop timestamp" value={tsRaw || '—'} />

          {/* Scope delta */}
          <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
            <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>Scope</div>
            {scopeFrom || scopeTo ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {scopeFrom && <DrawerField label="Subject scope" value={scopeFrom.join(', ')} mono />}
                {scopeTo && <DrawerField label="Granted scope" value={scopeTo.join(', ')} mono />}
                {dropped.length > 0 && (
                  <div>
                    <div style={drawerLabelStyle}>Dropped</div>
                    <div style={{
                      fontFamily: 'var(--font-mono)', fontSize: 10,
                      color: 'var(--danger, #ef4444)', wordBreak: 'break-all', lineHeight: 1.45,
                    }}>{dropped.join(', ')}</div>
                  </div>
                )}
              </div>
            ) : (
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.55 }}>
                No scope data for this hop.
              </span>
            )}
          </div>

          {/* Grant fields — real data from /admin/may-act */}
          <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
            <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>Delegation grant</div>
            {grantLoading ? (
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.55 }}>
                Loading grant…
              </span>
            ) : grant ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <DrawerField label="grant_id" value={grant.id} mono />
                <DrawerField label="max_hops" value={String(grant.max_hops ?? 1)} />
                <DrawerField
                  label="expires_at"
                  value={grant.expires_at ? new Date(grant.expires_at).toLocaleString() : 'never'}
                />
                <DrawerField
                  label="revoked"
                  value={grant.revoked_at ? new Date(grant.revoked_at).toLocaleString() : '—'}
                />
                {Array.isArray(grant.scopes) && grant.scopes.length > 0 && (
                  <DrawerField label="scopes" value={grant.scopes.join(', ')} mono />
                )}
                <button
                  data-testid="revoke-grant-btn"
                  disabled={!!grant.revoked_at || revokeBusy}
                  onClick={async () => {
                    if (!grant?.id) return
                    if (!confirm(`Revoke grant ${grant.id}?\n\nFuture token-exchange via this grant will be denied. This cannot be undone.`)) return
                    setRevokeBusy(true)
                    try {
                      await API.del('/admin/may-act/' + encodeURIComponent(grant.id))
                      toast?.success?.(`Grant revoked.`)
                      onClose()
                      onCanvasRefresh?.()
                    } catch (err: any) {
                      console.error('revoke grant failed', err)
                      toast?.error?.(`Revoke failed: ${err?.message || err}`)
                    } finally {
                      setRevokeBusy(false)
                    }
                  }}
                  style={{
                    marginTop: 6, alignSelf: 'flex-start',
                    fontFamily: 'var(--font-mono)', fontSize: 10,
                    padding: '4px 10px',
                    background: grant.revoked_at ? 'var(--surface-1)' : 'var(--surface-0)',
                    color: grant.revoked_at ? 'var(--fg-dim)' : 'var(--fg)',
                    border: '1px solid var(--hairline)',
                    cursor: grant.revoked_at ? 'not-allowed' : 'pointer',
                  }}
                >
                  {grant.revoked_at ? 'Already revoked' : (revokeBusy ? 'Revoking…' : 'Revoke this grant')}
                </button>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                <DrawerField label="max_hops" value="—" />
                <DrawerField label="expires_at" value="—" />
                <DrawerField label="revoked" value="—" />
                <div style={{ marginTop: 6, fontFamily: 'var(--font-mono)', fontSize: 9, color: 'var(--fg-dim)', opacity: 0.55, lineHeight: 1.5 }}>
                  No may_act grant found for this edge.
                </div>
              </div>
            )}
          </div>

          {/* Audit link — pass grant_id when known so the audit page filters too */}
          {auditHref && (
            <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
              <a
                href={auditHref}
                data-testid="edge-drawer-audit-link"
                onClick={e => {
                  if (onAuditClick && d.eventId) {
                    e.preventDefault()
                    onAuditClick(d.eventId, grant?.id)
                  }
                }}
                style={{
                  fontFamily: 'var(--font-mono)', fontSize: 11,
                  color: 'var(--fg-dim)', textDecoration: 'none',
                  display: 'inline-flex', alignItems: 'center', gap: 4,
                }}
                onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
                onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-dim)')}
              >
                View audit →
              </a>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

// ── Human drawer fields ───────────────────────────────────────────────────────

function HumanDrawerFields({ node, onNavigate }: { node: { id: string; data: any }; onNavigate?: (target: 'agent' | 'user', id: string) => void }) {
  const d = node.data || {}
  // Lead with Name → ID → Email (only when truly email-shaped). Don't fake an
  // email by falling back to label — label is often an agent_id for service-mode.
  const realEmail = typeof d.email === 'string' && d.email.includes('@') ? d.email : null
  return (
    <div data-testid="human-drawer-fields" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <DrawerField label="Name" value={d.label || d.name || '—'} />
      <DrawerField label="ID" value={stripLanePrefix(node.id)} mono />
      {realEmail && <DrawerField label="Email" value={realEmail} />}
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

      {/* View user link — uses in-app router when wired, hash fallback otherwise */}
      <a
        href={`/admin/users?id=${encodeURIComponent(node.id)}`}
        data-testid="human-drawer-user-link"
        onClick={e => {
          if (onNavigate) { e.preventDefault(); onNavigate('user', node.id); }
        }}
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

/** BFS from nodeId through rfEdges — returns all reachable descendant node ids */
function getDescendants(nodeId: string, rfEdges: any[]): string[] {
  const visited = new Set<string>()
  const queue = [nodeId]
  while (queue.length) {
    const cur = queue.shift()!
    for (const e of rfEdges) {
      if (e.source === cur && !visited.has(e.target)) {
        visited.add(e.target)
        queue.push(e.target)
      }
    }
  }
  visited.delete(nodeId)
  return Array.from(visited)
}

function AgentDrawerFields({ node, chainPos, prevHop, nextHop, tokenType, hopTs, scopeFrom, scopeTo, rfEdges, onNavigate, onClose, onCanvasRefresh, markRevoked }: {
  node: { id: string; data: any }
  chainPos: number
  prevHop: any
  nextHop: any
  tokenType: string
  hopTs: string
  scopeFrom?: string[]
  scopeTo?: string[]
  rfEdges?: any[]
  onNavigate?: (target: 'agent' | 'user', id: string) => void
  onClose?: () => void
  onCanvasRefresh?: () => void
  markRevoked?: (ids: string[]) => void
}) {
  const d = node.data || {}
  const toast = useToast()
  // d.revoked is set by canvas when agentStatus.active === false OR fresh kill.
  const isRevoked = d.revoked === true || d.active === false || !!d.revoked_at
  const status = isRevoked ? 'inactive' : 'active'
  const statusColor = status === 'active' ? 'var(--success, #22c55e)' : 'var(--danger, #ef4444)'
  const revokedAt = d.revoked_at

  // Scope delta from inbound edge data (subject_scope / granted_scope in audit metadata).
  const inherited = scopeFrom || (d.scopes ? (Array.isArray(d.scopes) ? d.scopes : d.scopes.split(' ').filter(Boolean)) : null)
  const granted = scopeTo || null
  const dropped = inherited && granted ? inherited.filter((s: string) => !granted.includes(s)) : []

  // BFS descendants for blast radius
  const descendants = React.useMemo(
    () => rfEdges ? getDescendants(node.id, rfEdges) : [],
    [node.id, rfEdges]
  )

  const revokeAgent = async (id: string) => {
    const label = d.label || id
    // Strip lane prefix before calling the API — the prefix is render-only.
    const apiId = stripLanePrefix(id)
    if (!confirm(`Revoke agent ${label}?\n\nAll outstanding tokens will be revoked. This cannot be undone.`)) return
    try {
      await API.del(`/agents/${encodeURIComponent(apiId)}`)
    } catch (err: any) {
      console.error('revoke failed', err)
      toast?.error?.(`Revoke failed: ${err?.message || err}`)
      return
    }
    // Optimistic: grey the node immediately. Map both lane-prefixed id and
    // canonical id + client_id so the canvas picks it up regardless of key.
    const cid = d.client_id || d.clientId
    markRevoked?.([id, apiId, cid].filter(Boolean) as string[])
    toast?.success?.(`Agent ${label} revoked. Tokens revoked.`)
    if (typeof onCanvasRefresh === 'function') onCanvasRefresh()
    if (typeof onClose === 'function') onClose()
  }

  const revokeBranch = async () => {
    const targets = [node.id, ...descendants]
    if (!confirm(`Revoke ${targets.length} agent(s) in this branch?\n\nAll outstanding tokens will be revoked. This cannot be undone.`)) return
    const results = await Promise.allSettled(
      // Strip lane prefix per id — prefix is render-only.
      targets.map(id => API.del(`/agents/${encodeURIComponent(stripLanePrefix(id))}`))
    )
    const ok = results.filter(r => r.status === 'fulfilled').length
    const failed = results.length - ok
    // Mark non-failed targets as revoked (optimistic). Include both forms so
    // the canvas matches regardless of which key the node uses.
    const succeeded = targets.filter((_, i) => results[i].status === 'fulfilled')
    const succeededCanonical = succeeded.map(stripLanePrefix)
    markRevoked?.([...succeeded, ...succeededCanonical])
    if (failed > 0) {
      console.error('branch revoke partial failure', results)
      toast?.error?.(`Revoked ${ok} of ${targets.length} agents. ${failed} failed.`)
    } else {
      toast?.success?.(`Branch revoked. ${ok} agent${ok !== 1 ? 's' : ''} killed.`)
    }
    if (typeof onCanvasRefresh === 'function') onCanvasRefresh()
    if (typeof onClose === 'function') onClose()
  }

  return (
    <div data-testid="agent-drawer-fields" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <DrawerField label="Name" value={d.label || d.name || '—'} />
      <DrawerField label="ID" value={stripLanePrefix(node.id)} mono />
      {(d.client_id || d.clientId) && (d.client_id || d.clientId) !== stripLanePrefix(node.id) && (
        <DrawerField label="client_id" value={d.client_id || d.clientId} mono />
      )}
      {d.jkt && <DrawerField label="DPoP jkt" value={d.jkt} mono />}

      {/* Status — color-coded (only place color is used per .impeccable.md) */}
      <div>
        <div style={drawerLabelStyle}>Status</div>
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: statusColor }}>
          {status}
        </span>
      </div>

      {revokedAt && (
        <DrawerField label="Revoked at" value={new Date(revokedAt).toLocaleString()} />
      )}

      {d.created_at && <DrawerField label="Created" value={new Date(d.created_at).toLocaleDateString()} />}

      {/* Scope section */}
      <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
        <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>Scope</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          {inherited ? (
            <>
              <DrawerField label="Inherited" value={inherited.join(', ')} mono />
              {granted && <DrawerField label="Granted" value={granted.join(', ')} mono />}
              {dropped.length > 0 && (
                <div>
                  <div style={drawerLabelStyle}>Dropped</div>
                  <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--danger, #ef4444)', wordBreak: 'break-all', lineHeight: 1.45 }}>
                    {dropped.join(', ')}
                  </div>
                </div>
              )}
            </>
          ) : (
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.55 }}>
              No scope data for this hop.
            </span>
          )}
        </div>
      </div>

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

      {/* Revocation section */}
      <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
        <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>Revocation</div>
        <div style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', marginBottom: 8, lineHeight: 1.6 }}>
          Blast radius:<br/>
          <span style={{ color: 'var(--fg)' }}>{descendants.length} downstream agent{descendants.length !== 1 ? 's' : ''}</span>
        </div>
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <button
            data-testid="revoke-agent-btn"
            disabled={isRevoked}
            onClick={() => revokeAgent(node.id)}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              padding: '4px 10px',
              background: isRevoked ? 'var(--surface-1)' : 'transparent',
              border: `1px solid ${isRevoked ? 'var(--hairline)' : 'var(--danger, #ef4444)'}`,
              borderRadius: 3,
              color: isRevoked ? 'var(--fg-dim)' : 'var(--danger, #ef4444)',
              cursor: isRevoked ? 'not-allowed' : 'pointer',
              transition: 'background 100ms',
            }}
            onMouseEnter={e => { if (!isRevoked) e.currentTarget.style.background = 'rgba(239,68,68,0.08)' }}
            onMouseLeave={e => { if (!isRevoked) e.currentTarget.style.background = 'transparent' }}
          >
            {isRevoked ? 'Already revoked' : 'Revoke this agent'}
          </button>
          {descendants.length > 0 && !isRevoked && (
            <button
              data-testid="revoke-branch-btn"
              onClick={revokeBranch}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 10,
                padding: '4px 10px',
                background: 'transparent',
                border: '1px solid var(--danger, #ef4444)',
                borderRadius: 3,
                color: 'var(--danger, #ef4444)',
                cursor: 'pointer',
                transition: 'background 100ms',
              }}
              onMouseEnter={e => (e.currentTarget.style.background = 'rgba(239,68,68,0.08)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              Revoke this branch ({descendants.length + 1})
            </button>
          )}
        </div>
      </div>

      {/* Grants from/to this agent — surfaced from /admin/may-act */}
      <AgentGrantsTables agentId={node.id} />

      {/* View agent page link */}
      <a
        href={`/admin/agents?q=${encodeURIComponent(node.id)}`}
        data-testid="agent-drawer-agent-link"
        onClick={e => {
          if (onNavigate) { e.preventDefault(); onNavigate('agent', node.id); }
        }}
        style={{
          fontFamily: 'var(--font-mono)', fontSize: 11,
          color: 'var(--fg-dim)', textDecoration: 'none',
        }}
        onMouseEnter={e => (e.currentTarget.style.color = 'var(--fg)')}
        onMouseLeave={e => (e.currentTarget.style.color = 'var(--fg-dim)')}
      >
        View agent →
      </a>
    </div>
  )
}

// AgentGrantsTables fetches both directions of grants for an agent and renders
// two compact tables. Empty-state per direction. Skipped entirely when fetch
// fails (best-effort UI surface — never breaks the drawer).
function AgentGrantsTables({ agentId }: { agentId: string }) {
  const [from, setFrom] = React.useState<any[] | null>(null)
  const [to, setTo] = React.useState<any[] | null>(null)
  React.useEffect(() => {
    let cancelled = false
    const canonicalId = stripLanePrefix(agentId)
    Promise.all([
      API.get(`/admin/may-act?from_id=${encodeURIComponent(canonicalId)}&include_revoked=true`).catch(() => null),
      API.get(`/admin/may-act?to_id=${encodeURIComponent(canonicalId)}&include_revoked=true`).catch(() => null),
    ]).then(([a, b]) => {
      if (cancelled) return
      setFrom(a?.grants || [])
      setTo(b?.grants || [])
    })
    return () => { cancelled = true }
  }, [agentId])

  const renderTable = (title: string, rows: any[] | null, dirLabel: 'to_id' | 'from_id') => (
    <div style={{ borderTop: '1px solid var(--hairline)', paddingTop: 12 }}>
      <div style={{ ...drawerLabelStyle, marginBottom: 8 }}>{title}</div>
      {rows == null ? (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.55 }}>
          Loading…
        </span>
      ) : rows.length === 0 ? (
        <span style={{ fontFamily: 'var(--font-mono)', fontSize: 10, color: 'var(--fg-dim)', opacity: 0.55 }}>
          no grants
        </span>
      ) : (
        <table style={{ width: '100%', borderCollapse: 'collapse', fontFamily: 'var(--font-mono)', fontSize: 10 }}>
          <thead>
            <tr style={{ color: 'var(--fg-dim)', textAlign: 'left' }}>
              <th style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>{dirLabel}</th>
              <th style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>scopes</th>
              <th style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>hops</th>
              <th style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>exp</th>
              <th style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>revoked</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(g => (
              <tr key={g.id}>
                <td style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)', wordBreak: 'break-all' }}>
                  {dirLabel === 'to_id' ? g.to_id : g.from_id}
                </td>
                <td style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>
                  {(g.scopes || []).slice(0, 3).join(',') || '—'}
                  {(g.scopes || []).length > 3 ? '…' : ''}
                </td>
                <td style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>{g.max_hops}</td>
                <td style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)' }}>
                  {g.expires_at ? new Date(g.expires_at).toLocaleDateString() : '—'}
                </td>
                <td style={{ padding: '4px 6px', borderBottom: '1px solid var(--hairline)', color: g.revoked_at ? 'var(--danger, #ef4444)' : 'var(--fg-dim)' }}>
                  {g.revoked_at ? 'yes' : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )

  return (
    <div data-testid="agent-grants-tables">
      {renderTable('Grants FROM this agent', from, 'to_id')}
      {renderTable('Grants TO this agent', to, 'from_id')}
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
  // onNodeClick / onEdgeClick stay for telemetry/debug, but the drawer is the
  // primary interaction. Don't navigate from these callbacks — the drawer's
  // "View user/agent →" links call onNavigate instead.
  onNodeClick?: (nodeId: string, nodeData: any) => void
  onEdgeClick?: (edgeData: any) => void
  // onNavigate fires when the user clicks "View agent/user →" inside the drawer.
  // Pass it the page's setPage to wire up real routing.
  onNavigate?: (target: 'agent' | 'user', id: string) => void
  // onAuditClick fires when the edge drawer's "View audit event →" is clicked.
  // grantId (when known) lets the parent route to /admin/audit?grant_id=<id>.
  onAuditClick?: (eventId: string, grantId?: string) => void
  // onCanvasRefresh fires after a destructive action (e.g. revoke grant) so
  // the parent page can refetch graph data. Optional.
  onCanvasRefresh?: () => void
  // agentStatus: id-or-client_id → {active, updated_at}. Lets the canvas
  // overlay REVOKED on historical nodes whose backing agent is now inactive.
  agentStatus?: Map<string, { active: boolean; updated_at?: string; deactivated_at?: string }>
  // grantStatus: "from->to" → may_act grant status. Lets the canvas render
  // externally revoked delegation grants without relying on the edge drawer.
  grantStatus?: Map<string, { revoked_at?: string; id?: string }>
  height?: number
  fitView?: boolean
  loading?: boolean
  emptyMessage?: string
  emptyHint?: string
  // change me to force a re-fit after data swap (e.g. selectedChain.rootSub).
  // RF's `fitView` PROP only runs at mount; this drives an imperative fitView.
  chainKey?: string
}

export function DelegationCanvas({
  rfNodes,
  rfEdges,
  onNodeClick,
  onEdgeClick,
  onNavigate,
  onAuditClick,
  onCanvasRefresh,
  agentStatus,
  grantStatus,
  height = 520,
  fitView = true,
  loading = false,
  emptyMessage = 'No delegation chains.',
  emptyHint,
  chainKey,
}: DelegationCanvasProps) {
  // Optimistic kill list — feels instant, before refetch lands
  const [freshlyRevoked, setFreshlyRevoked] = React.useState<Set<string>>(new Set())
  const markRevoked = React.useCallback((ids: string[]) => {
    setFreshlyRevoked(prev => {
      const next = new Set(prev)
      for (const id of ids) next.add(id)
      return next
    })
  }, [])

  // Decorate nodes with revoked state from agentStatus + freshlyRevoked.
  // Match by node.id and by data.client_id (audit-derived nodes use client_id).
  const decoratedRfNodes = React.useMemo(() => {
    if (!rfNodes) return rfNodes
    return rfNodes.map(n => {
      const cid = n.data?.client_id || n.data?.clientId
      const lookupKeys = [n.id, stripLanePrefix(n.id), cid].filter(Boolean) as string[]
      let statusEntry: any = null
      if (agentStatus) {
        for (const k of lookupKeys) {
          const e = agentStatus.get(k)
          if (e) { statusEntry = e; break }
        }
      }
      const fresh = lookupKeys.some(k => freshlyRevoked.has(k))
      const revokedFromStatus = statusEntry && statusEntry.active === false
      const revoked = fresh || revokedFromStatus || n.data?.revoked === true
      const revokedAt = revoked
        ? (statusEntry?.deactivated_at || statusEntry?.updated_at || n.data?.revoked_at)
        : undefined
      if (!revoked && !statusEntry) return n
      return {
        ...n,
        data: {
          ...n.data,
          revoked,
          revoked_at: revokedAt,
          active: statusEntry?.active,
        },
      }
    })
  }, [rfNodes, agentStatus, freshlyRevoked])

  const [nodes, setNodes, onNodesChange] = useNodesState(decoratedRfNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(rfEdges)
  // useNodesState/useEdgesState only seed from initial props. Sync when parent
  // hands us a new set (e.g. circular chain-button swap) — without this the
  // canvas keeps rendering the previous chain's graph.
  React.useEffect(() => { setNodes(decoratedRfNodes) }, [decoratedRfNodes, setNodes])
  const decoratedRfEdges = React.useMemo(() => {
    if (!rfEdges || !grantStatus) return rfEdges
    return rfEdges.map(e => {
      const source = stripLanePrefix(e.source)
      const target = stripLanePrefix(e.target)
      const status = grantStatus.get(`${source}->${target}`)
      if (!status?.revoked_at) return e
      return {
        ...e,
        className: 'revoked-hop',
        markerEnd: {
          ...e.markerEnd,
          color: 'var(--danger, #ef4444)',
        },
        style: {
          ...(e.style || {}),
          stroke: 'var(--danger, #ef4444)',
          opacity: 0.58,
          strokeDasharray: '2 5',
          animation: undefined,
        },
        data: {
          ...(e.data || {}),
          grantId: status.id || e.data?.grantId,
          revoked: true,
          revoked_at: status.revoked_at,
          isActive: false,
        },
      }
    })
  }, [rfEdges, grantStatus])

  React.useEffect(() => { setEdges(decoratedRfEdges) }, [decoratedRfEdges, setEdges])
  const rfApi = useReactFlow()
  // Re-fit viewport when chainKey flips. RF preserves pan/zoom across renders,
  // so swapping rfNodes alone leaves the camera on the old graph.
  // Timeout: let RF apply new nodes/edges before measuring bounds.
  React.useEffect(() => {
    if (!chainKey) return
    const t = setTimeout(() => {
      try { rfApi.fitView({ padding: 0.32, duration: 300 }) } catch {}
    }, 80)
    return () => clearTimeout(t)
  }, [chainKey, rfApi])
  // selectedNode/Edge drive the right-side drawer (one open at a time)
  const [selectedNode, setSelectedNode] = useState<{ id: string; data: any } | null>(null)
  const [selectedEdge, setSelectedEdge] = useState<{ id: string; source: string; target: string; data: any } | null>(null)

  const handleNodeClick = (_: any, node: any) => {
    setSelectedEdge(null)
    setSelectedNode({ id: node.id, data: node.data })
    onNodeClick?.(node.id, node.data)
  }

  const handleEdgeClick = (_: any, edge: any) => {
    setSelectedNode(null)
    setSelectedEdge({ id: edge.id, source: edge.source, target: edge.target, data: edge.data })
    onEdgeClick?.(edge.data)
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
        onEdgeClick={handleEdgeClick}
        fitView={fitView}
        fitViewOptions={{ padding: 0.32 }}
        minZoom={0.15}
        maxZoom={3}
        defaultEdgeOptions={{
          type: 'animatedBezier',
          style: { stroke: 'var(--fg-dim)', strokeWidth: 1 },
          // P0-6: arrowheads 14 default for legibility at fitView zoom
          markerEnd: { type: MarkerType.ArrowClosed, color: 'var(--fg-dim)', width: 14, height: 14 },
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
        rfNodes={decoratedRfNodes}
        rfEdges={rfEdges}
        onClose={() => setSelectedNode(null)}
        onNavigate={onNavigate}
        onCanvasRefresh={onCanvasRefresh}
        markRevoked={markRevoked}
      />
      {/* Edge detail drawer — same right slot, mutually exclusive with node drawer */}
      <EdgeDrawer
        edge={selectedEdge}
        rfNodes={decoratedRfNodes}
        onClose={() => setSelectedEdge(null)}
        onAuditClick={onAuditClick}
        onCanvasRefresh={onCanvasRefresh}
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
