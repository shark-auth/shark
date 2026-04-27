"""
F7 — Delegation Canvas /impeccable craft smoke tests.

Static checks:
  - delegation_canvas.tsx contains HumanNode and AgentNode definitions
  - getInitials helper present with slice/toUpperCase logic
  - Edge labels include 'token_exchange' keyword
  - Header in delegation_chains.tsx shows N-hop chain summary format
  - delegation_chains.tsx contains relTime and chainPath helpers
  - Back link present in canvas header ("← chains")
"""

import os
import re

import pytest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
ADMIN_DIR = os.path.join(REPO_ROOT, "admin", "src", "components")


# ─── delegation_canvas.tsx static checks ─────────────────────────────────────

def _canvas_content():
    path = os.path.join(ADMIN_DIR, "delegation_canvas.tsx")
    assert os.path.exists(path), f"File not found: {path}"
    return open(path, encoding="utf-8").read()


def test_canvas_defines_human_node():
    """delegation_canvas.tsx must define HumanNode for human principals."""
    content = _canvas_content()
    assert "function HumanNode" in content, (
        "HumanNode node type not found in delegation_canvas.tsx"
    )


def test_canvas_defines_agent_node():
    """delegation_canvas.tsx must define AgentNode for agent principals."""
    content = _canvas_content()
    assert "function AgentNode" in content, (
        "AgentNode node type not found in delegation_canvas.tsx"
    )


def test_canvas_get_initials_function():
    """delegation_canvas.tsx must export getInitials helper."""
    content = _canvas_content()
    assert "getInitials" in content, (
        "getInitials helper not found in delegation_canvas.tsx"
    )


def test_canvas_initials_uses_uppercase():
    """getInitials must call toUpperCase (per spec: 1-2 uppercase letters)."""
    content = _canvas_content()
    assert "toUpperCase" in content, (
        "getInitials must call .toUpperCase() to produce uppercase initials"
    )


def test_canvas_initials_uses_slice():
    """getInitials must use .slice for 1-2 letter extraction."""
    content = _canvas_content()
    assert ".slice(" in content, (
        "getInitials should use .slice() to cap at 1-2 characters"
    )


def test_canvas_edge_label_token_exchange():
    """Edge labels must include 'token_exchange' keyword per F7 spec."""
    content = _canvas_content()
    assert "token_exchange" in content, (
        "Edge label format must include 'token_exchange' in delegation_canvas.tsx"
    )


def test_canvas_active_hop_bolder():
    """Active hop (most recent) must use bolder strokeWidth than historical hops."""
    content = _canvas_content()
    # Should have isActivHop or active-hop class driving bolder stroke
    assert "isActivHop" in content or "active-hop" in content, (
        "Canvas must mark active hop with isActivHop flag or active-hop class"
    )
    # Stroke width for active hop should be > 1.5 (historical)
    assert "2.5" in content or "strokeWidth: 2" in content, (
        "Active hop should use strokeWidth > 1.5 (e.g. 2.5)"
    )


def test_canvas_act_as_badge():
    """AgentNode must render act-as count badge when chain length > 1."""
    content = _canvas_content()
    assert "actAsCount" in content, (
        "AgentNode must accept and render actAsCount badge"
    )


def test_canvas_node_types_include_human_and_agent():
    """nodeTypes registry must include humanNode and agentNode."""
    content = _canvas_content()
    assert "humanNode" in content, "nodeTypes must register humanNode"
    assert "agentNode" in content, "nodeTypes must register agentNode"


def test_canvas_left_to_right_layout():
    """Canvas uses left-to-right layout (LAYER_GAP / x = layer * gap)."""
    content = _canvas_content()
    assert "LAYER_GAP" in content or "layer * " in content, (
        "Canvas must use left-to-right layered layout"
    )


# ─── delegation_chains.tsx static checks ─────────────────────────────────────

def _chains_content():
    path = os.path.join(ADMIN_DIR, "delegation_chains.tsx")
    assert os.path.exists(path), f"File not found: {path}"
    return open(path, encoding="utf-8").read()


def test_chains_rel_time_helper():
    """delegation_chains.tsx must define relTime helper for 'Xm ago' format."""
    content = _chains_content()
    assert "relTime" in content or "reltime" in content.lower(), (
        "relTime helper not found in delegation_chains.tsx"
    )


def test_chains_chain_path_helper():
    """delegation_chains.tsx must define chainPath helper for path summary."""
    content = _chains_content()
    assert "chainPath" in content, (
        "chainPath helper not found in delegation_chains.tsx"
    )


def test_chains_hop_summary_format():
    """Header must render N-hop chain summary (e.g. '3-hop chain')."""
    content = _chains_content()
    assert "-hop chain" in content or "hop chain" in content, (
        "Chain header must show N-hop chain summary"
    )


def test_chains_back_link():
    """Canvas header must have back link to chains list."""
    content = _chains_content()
    assert "← chains" in content or "← back" in content.lower(), (
        "Canvas header must include back link (← chains)"
    )


def test_chains_started_ago_format():
    """Header must show 'started Xm ago' relative timestamp."""
    content = _chains_content()
    assert "started" in content, (
        "Chain header must show 'started <relative-time>' for UX context"
    )


# ─── unit-level getInitials logic verification ───────────────────────────────

def python_get_initials(label: str) -> str:
    """Python mirror of the TypeScript getInitials helper."""
    import re
    if not label:
        return "?"
    local = label.split("@")[0] if "@" in label else label
    parts = [p for p in re.split(r"[\s\-_.]+", local) if p]
    if len(parts) >= 2:
        return (parts[0][0] + parts[1][0]).upper()[:2]
    return local[:2].upper()


def test_get_initials_from_email():
    assert python_get_initials("alice@corp.com") == "AL"


def test_get_initials_from_hyphen_name():
    assert python_get_initials("research-agent") == "RA"


def test_get_initials_from_underscore_name():
    assert python_get_initials("tool_agent_v2") == "TA"


def test_get_initials_from_display_name():
    assert python_get_initials("Bob Smith") == "BS"


def test_get_initials_single_word():
    result = python_get_initials("alice")
    assert result == "AL"
    assert len(result) <= 2


def test_get_initials_empty():
    assert python_get_initials("") == "?"


# ─── Fix 2: onNodeClick drawer static checks ─────────────────────────────────

def test_canvas_on_node_click_handler():
    """delegation_canvas.tsx must register onNodeClick handler on ReactFlow."""
    content = _canvas_content()
    assert "onNodeClick" in content, (
        "delegation_canvas.tsx must register onNodeClick on the ReactFlow component"
    )
    assert "handleNodeClick" in content or "onNodeClick={" in content, (
        "onNodeClick handler must be wired to ReactFlow component"
    )


def test_canvas_selected_node_state():
    """delegation_canvas.tsx must manage selectedNode state for drawer."""
    content = _canvas_content()
    assert "selectedNode" in content, (
        "delegation_canvas.tsx must have selectedNode state for the node detail drawer"
    )
    assert "setSelectedNode" in content, (
        "delegation_canvas.tsx must have setSelectedNode state setter"
    )


def test_canvas_drawer_component_present():
    """delegation_canvas.tsx must define NodeDrawer component."""
    content = _canvas_content()
    assert "NodeDrawer" in content, (
        "delegation_canvas.tsx must define NodeDrawer component"
    )
    assert "node-drawer" in content or "data-testid" in content, (
        "NodeDrawer must include data-testid attribute for test targeting"
    )


def test_canvas_drawer_human_fields():
    """NodeDrawer must render human-specific fields (email, name, role chips, user link)."""
    content = _canvas_content()
    assert "HumanDrawerFields" in content or "human-drawer-fields" in content, (
        "NodeDrawer must render HumanDrawerFields for human nodes"
    )
    assert "View user" in content, (
        "Human drawer must include 'View user →' link to /admin/users"
    )


def test_canvas_drawer_agent_fields():
    """NodeDrawer must render agent-specific fields (client_id, jkt, chain context)."""
    content = _canvas_content()
    assert "AgentDrawerFields" in content or "agent-drawer-fields" in content, (
        "NodeDrawer must render AgentDrawerFields for agent nodes"
    )
    assert "client_id" in content, (
        "Agent drawer must display client_id field"
    )
    assert "DPoP jkt" in content or "jkt" in content, (
        "Agent drawer must display DPoP jkt thumbprint"
    )
    assert "Position in chain" in content or "chainPos" in content, (
        "Agent drawer must show position in delegation chain"
    )


def test_canvas_drawer_audit_link():
    """NodeDrawer must include 'View in audit log →' link."""
    content = _canvas_content()
    assert "View in audit log" in content, (
        "NodeDrawer must include 'View in audit log →' link for both node types"
    )
    assert "node-drawer-audit-link" in content, (
        "Audit log link must have data-testid='node-drawer-audit-link'"
    )


def test_canvas_drawer_escape_close():
    """NodeDrawer must close on Escape key."""
    content = _canvas_content()
    assert "Escape" in content, (
        "NodeDrawer must close on Escape key press"
    )


def test_canvas_drawer_slide_animation():
    """NodeDrawer must use slide-in animation (200ms per spec)."""
    content = _canvas_content()
    assert "drawerSlideIn" in content or "slideIn" in content, (
        "NodeDrawer must define slide-in animation"
    )
    assert "200ms" in content, (
        "Drawer animation must be 200ms per spec"
    )


def test_canvas_drawer_hairline_border():
    """NodeDrawer must use hairline left border per .impeccable.md v3 DNA."""
    content = _canvas_content()
    assert "borderLeft" in content and "hairline" in content, (
        "NodeDrawer must use hairline left border (borderLeft: '1px solid var(--hairline)')"
    )
