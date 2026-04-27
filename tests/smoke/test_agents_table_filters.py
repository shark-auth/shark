"""
Static-analysis smoke tests for agents_manage.tsx filter/search/user-column
and applications.tsx proxy-rules removal.

These are pure file-content checks — no server required.
"""
import re
from pathlib import Path

REPO = Path(__file__).parent.parent.parent
AGENTS_TSX = REPO / "admin" / "src" / "components" / "agents_manage.tsx"
APPS_TSX   = REPO / "admin" / "src" / "components" / "applications.tsx"


def _src(path: Path) -> str:
    return path.read_text(encoding="utf-8")


# ── agents_manage.tsx ─────────────────────────────────────────────────────────

def test_agents_filter_chip_pattern_present():
    """Filter chips must use the canonical pattern: filter === v ? ..."""
    src = _src(AGENTS_TSX)
    assert "filter === v" in src, (
        "Expected filter chip pattern `filter === v` not found in agents_manage.tsx"
    )


def test_agents_search_input_present():
    """Search input for agent name / client_id must exist."""
    src = _src(AGENTS_TSX)
    assert "Filter by name or client_id" in src or "filter by name" in src.lower(), (
        "Search input placeholder not found in agents_manage.tsx"
    )


def test_agents_user_column_header_present():
    """Table must have a 'Created by' or 'User' column header."""
    src = _src(AGENTS_TSX)
    assert "Created by" in src or ">User<" in src, (
        "User/Created-by column header not found in agents_manage.tsx"
    )


def test_agents_user_picker_dropdown_present():
    """Creator filter dropdown (All creators / user picker) must exist."""
    src = _src(AGENTS_TSX)
    assert "All creators" in src or "creatorFilter" in src, (
        "User picker / creatorFilter not found in agents_manage.tsx"
    )


def test_agents_client_side_user_join_present():
    """userMap lookup for client-side join must exist."""
    src = _src(AGENTS_TSX)
    assert "userMap" in src, (
        "userMap (client-side user join) not found in agents_manage.tsx"
    )


def test_agents_identity_section_in_drawer():
    """Drawer must display linked user Identity section."""
    src = _src(AGENTS_TSX)
    assert "Bound to user" in src or "Identity" in src, (
        "Identity / 'Bound to user' section not found in AgentConfig drawer"
    )


def test_agents_view_user_link_navigates_to_users():
    """Drawer linked-user button must navigate to users page."""
    src = _src(AGENTS_TSX)
    assert "setPage" in src and "users" in src, (
        "setPage('users') navigation not found in agents_manage.tsx"
    )


def test_agents_f6_flatten_act_chain_preserved():
    """
    F6 delegation chain test relies on the flattenActClaim / resolveChain logic
    at line ~1543. Ensure it is still present.
    """
    src = _src(AGENTS_TSX)
    assert "flattenActClaim" in src, (
        "flattenActClaim function removed — breaks F6 delegation chain test"
    )
    assert "resolveChain" in src, (
        "resolveChain function removed — breaks F6 delegation chain test"
    )


# ── applications.tsx ──────────────────────────────────────────────────────────

def test_applications_proxy_rules_tab_removed():
    """'Proxy rules' tab must not appear in the tab list."""
    src = _src(APPS_TSX)
    # The tab entry was: ['rules', 'Proxy rules']
    assert "'rules'" not in src or "Proxy rules" not in src, (
        "Proxy rules tab still present in applications.tsx tab list"
    )


def test_applications_no_proxy_rules_string():
    """
    The string 'proxy_rules' or the AppProxyRules component must not be
    rendered (function may remain as dead code comment, but must not be called).
    """
    src = _src(APPS_TSX)
    # tab === 'rules' render must be gone
    assert "tab === 'rules'" not in src, (
        "tab === 'rules' render still present in applications.tsx"
    )
    # AppProxyRules must not be called/rendered
    assert "<AppProxyRules" not in src, (
        "<AppProxyRules ... /> render still present in applications.tsx"
    )


def test_applications_config_tab_intact():
    """Config tab must still be present after proxy rules removal."""
    src = _src(APPS_TSX)
    assert "'config'" in src, "Config tab removed from applications.tsx — unintended regression"
    assert "AppConfig" in src, "AppConfig component removed — unintended regression"


def test_applications_create_flow_intact():
    """CreateAppSlideOver must still be present."""
    src = _src(APPS_TSX)
    assert "CreateAppSlideOver" in src, (
        "CreateAppSlideOver removed from applications.tsx — unintended regression"
    )


def test_applications_rotate_secret_intact():
    """RotateSecretModal must still be present."""
    src = _src(APPS_TSX)
    assert "RotateSecretModal" in src, (
        "RotateSecretModal removed from applications.tsx — unintended regression"
    )
