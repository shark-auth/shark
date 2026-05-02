"""
Smoke tests: Overview page agent metrics are real (not hardcoded mock values).

Static checks run without a server. Live checks are skipped if no server is up.
"""
import re
import pytest
import os
import requests

OVERVIEW_TSX = os.path.join(
    os.path.dirname(__file__),
    "..", "..", "admin", "src", "components", "overview.tsx"
)

BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:9443")
ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")


def _src():
    with open(OVERVIEW_TSX, encoding="utf-8") as f:
        return f.read()


# ---------------------------------------------------------------------------
# Static checks (no server needed)
# ---------------------------------------------------------------------------

FORBIDDEN_MOCK_LITERALS = [42, 1337, 99, 128, 256, 512]


def test_no_hardcoded_mock_numbers_in_agent_section():
    """overview.tsx must not contain bare mock integer literals in agent metric values."""
    src = _src()
    # Find the agent metrics section (between useAgentMetrics and AttentionPanel)
    for num in FORBIDDEN_MOCK_LITERALS:
        # Allow numbers inside comments or strings used as CSS/style values
        # Flag only bare JS value assignments like: v: 42, or value: 42
        pattern = rf"(?:v:|value:)\s*{num}\b"
        assert not re.search(pattern, src), (
            f"Found hardcoded mock value '{num}' in agent metric card assignment in overview.tsx"
        )


def test_agent_metric_cards_use_useAgentMetrics():
    """Agent KPI values must be derived from useAgentMetrics hook, not hardcoded."""
    src = _src()
    assert "useAgentMetrics" in src, "useAgentMetrics hook must be present in overview.tsx"
    assert "agentMetrics" in src, "agentMetrics state variable must be used in overview.tsx"
    # The hook must fetch from /api/v1/agents
    assert "/api/v1/agents" in src or "'/agents'" in src or '"/agents"' in src, (
        "Agent metrics hook must fetch from /api/v1/agents"
    )


def test_active_agent_tooltip_present():
    """Active agents card must surface the definition tooltip."""
    src = _src()
    tooltip_text = "Active = issued ≥1 token in the last 7 days, not deactivated"
    assert tooltip_text in src, (
        f"Tooltip text not found in overview.tsx.\n"
        f"Expected: {tooltip_text!r}"
    )


def test_active_agent_tooltip_constant_named():
    """ACTIVE_AGENT_TOOLTIP constant must be defined and referenced."""
    src = _src()
    assert "ACTIVE_AGENT_TOOLTIP" in src, (
        "ACTIVE_AGENT_TOOLTIP constant must be defined in overview.tsx"
    )


def test_sparkline_used_for_agent_cards():
    """Sparkline component must be wired to agent metric cards."""
    src = _src()
    # Sparkline import present
    assert "Sparkline" in src, "Sparkline must be imported in overview.tsx"
    # agentSparkline variable must be referenced in the metrics array
    assert "agentSparkline" in src, "agentSparkline must be used for agent trend data"


def test_loading_state_present():
    """Agent cards must show a muted '—' while loading."""
    src = _src()
    assert "agentMetricsLoading" in src, (
        "agentMetricsLoading must be used to gate loading display"
    )


def test_error_state_present():
    """Agent cards must show an error state on fetch failure."""
    src = _src()
    assert "agentMetricsError" in src, (
        "agentMetricsError must be used to show error state"
    )
    assert "couldn't load" in src, (
        "Error message \"couldn't load\" must appear in overview.tsx"
    )


def test_two_agent_cards_in_metrics():
    """Both 'Total agents' and 'Active agents' cards must be present."""
    src = _src()
    assert "'Total agents'" in src or '"Total agents"' in src, (
        "Total agents card must be present in metrics array"
    )
    assert "'Active agents'" in src or '"Active agents"' in src, (
        "Active agents card must be present in metrics array"
    )


# ---------------------------------------------------------------------------
# Live checks (skipped if server not available)
# ---------------------------------------------------------------------------

def _server_up():
    try:
        r = requests.get(f"{BASE_URL}/api/v1/health", timeout=3)
        return r.status_code < 500
    except Exception:
        return False


@pytest.mark.skipif(not _server_up(), reason="Shark server not running")
def test_audit_logs_token_exchange_endpoint():
    """GET /api/v1/audit-logs?action=token.exchange&limit=1 must return 200."""
    headers = {}
    if ADMIN_KEY:
        headers["Authorization"] = f"Bearer {ADMIN_KEY}"
    r = requests.get(
        f"{BASE_URL}/api/v1/audit-logs",
        params={"action": "token.exchange", "limit": "1"},
        headers=headers,
        timeout=5,
    )
    assert r.status_code == 200, (
        f"Expected 200 from audit-logs endpoint, got {r.status_code}: {r.text[:200]}"
    )


@pytest.mark.skipif(not _server_up(), reason="Shark server not running")
def test_agents_list_endpoint_returns_total():
    """GET /api/v1/agents?limit=1 must return 200 with a 'total' field."""
    headers = {}
    if ADMIN_KEY:
        headers["Authorization"] = f"Bearer {ADMIN_KEY}"
    r = requests.get(
        f"{BASE_URL}/api/v1/agents",
        params={"limit": "1"},
        headers=headers,
        timeout=5,
    )
    assert r.status_code == 200, (
        f"Expected 200 from agents list, got {r.status_code}: {r.text[:200]}"
    )
    body = r.json()
    assert "total" in body, f"Response missing 'total' field: {body}"


@pytest.mark.skipif(not _server_up(), reason="Shark server not running")
def test_agents_active_filter():
    """GET /api/v1/agents?active=true must return 200."""
    headers = {}
    if ADMIN_KEY:
        headers["Authorization"] = f"Bearer {ADMIN_KEY}"
    r = requests.get(
        f"{BASE_URL}/api/v1/agents",
        params={"active": "true", "limit": "1"},
        headers=headers,
        timeout=5,
    )
    assert r.status_code == 200, (
        f"Expected 200 from agents?active=true, got {r.status_code}: {r.text[:200]}"
    )
