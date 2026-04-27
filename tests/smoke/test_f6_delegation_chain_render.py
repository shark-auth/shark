"""
F6 — flattenActClaim helper smoke tests.

Static checks:
  - delegation_chains.tsx contains the flattenActClaim function
  - agents_manage.tsx contains inline flatten logic with flattenActClaim reference
  - Both files handle array input (use as-is) and nested RFC 8693 object (flatten)

Live checks (skipif shark not reachable):
  - GET /api/v1/audit-logs?action=oauth.token.exchanged returns valid JSON
  - At least one event with act_chain that either:
      a) is already a flat array of length >= 2, or
      b) is a nested RFC 8693 object that flattens to length >= 2
"""

import json
import os
import re
import socket

import pytest
import requests

BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")
REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
ADMIN_DIR = os.path.join(REPO_ROOT, "admin", "src", "components")


def _shark_reachable() -> bool:
    """Return True if shark is already listening on BASE_URL host:port."""
    try:
        import urllib.parse
        parsed = urllib.parse.urlparse(BASE_URL)
        host = parsed.hostname or "localhost"
        port = parsed.port or 8080
        with socket.create_connection((host, port), timeout=1):
            return True
    except OSError:
        return False


# ─── static checks ───────────────────────────────────────────────────────────

def test_delegation_chains_contains_flatten_helper():
    """delegation_chains.tsx must define flattenActClaim."""
    path = os.path.join(ADMIN_DIR, "delegation_chains.tsx")
    assert os.path.exists(path), f"File not found: {path}"
    content = open(path, encoding="utf-8").read()
    assert "function flattenActClaim" in content, (
        "flattenActClaim helper not found in delegation_chains.tsx"
    )


def test_delegation_chains_three_way_check():
    """normalizeEntry must use three-way check: array, nested object, empty."""
    path = os.path.join(ADMIN_DIR, "delegation_chains.tsx")
    content = open(path, encoding="utf-8").read()
    # Should have resolveChain or equivalent logic that handles both array and nested object
    assert "resolveChain" in content or "flattenActClaim" in content, (
        "delegation_chains.tsx must use resolveChain / flattenActClaim in normalizeEntry"
    )
    # Three-way: array branch
    assert "Array.isArray" in content, "Must have Array.isArray branch"
    # Nested object branch — checks for .sub property
    assert ".sub" in content, "Must have nested object branch checking .sub"


def test_agents_manage_contains_flatten_logic():
    """agents_manage.tsx must inline flatten logic for act_chain."""
    path = os.path.join(ADMIN_DIR, "agents_manage.tsx")
    assert os.path.exists(path), f"File not found: {path}"
    content = open(path, encoding="utf-8").read()
    assert "flattenActClaim" in content, (
        "flattenActClaim inline logic not found in agents_manage.tsx"
    )
    assert "resolveChain" in content, (
        "resolveChain helper not found in agents_manage.tsx"
    )


def test_agents_manage_array_branch():
    """agents_manage.tsx flatten logic must handle flat array input."""
    path = os.path.join(ADMIN_DIR, "agents_manage.tsx")
    content = open(path, encoding="utf-8").read()
    assert "Array.isArray" in content, (
        "agents_manage.tsx must preserve flat array input (Array.isArray branch)"
    )


# ─── unit-level flatten logic verification ───────────────────────────────────

def flatten_act_claim(nested):
    """Python mirror of the TypeScript flattenActClaim helper."""
    hops = []
    cur = nested
    while cur and isinstance(cur, dict) and "sub" in cur:
        hops.append({"sub": cur["sub"], "jkt": cur.get("jkt")})
        cur = cur.get("act")
    return hops


def resolve_chain(raw):
    if isinstance(raw, list) and len(raw) > 0:
        return raw
    if isinstance(raw, dict) and "sub" in raw:
        return flatten_act_claim(raw)
    return []


def test_flatten_rfc8693_nested_object():
    """flattenActClaim must unroll {"sub":"a2","act":{"sub":"a1"}} to 2-hop array."""
    nested = {"sub": "a2", "act": {"sub": "a1"}}
    result = flatten_act_claim(nested)
    assert len(result) == 2, f"Expected 2 hops, got {len(result)}: {result}"
    assert result[0]["sub"] == "a2"
    assert result[1]["sub"] == "a1"


def test_flatten_3hop_nested():
    """flattenActClaim must unroll a 3-hop nested chain."""
    nested = {"sub": "a3", "act": {"sub": "a2", "act": {"sub": "a1"}}}
    result = flatten_act_claim(nested)
    assert len(result) == 3
    assert [h["sub"] for h in result] == ["a3", "a2", "a1"]


def test_resolve_chain_flat_array_passthrough():
    """resolve_chain must return a flat array as-is."""
    raw = [{"sub": "a1"}, {"sub": "a2"}]
    result = resolve_chain(raw)
    assert result == raw


def test_resolve_chain_empty():
    """resolve_chain must return [] for None / missing."""
    assert resolve_chain(None) == []
    assert resolve_chain([]) == []
    assert resolve_chain({}) == []


# ─── live checks ─────────────────────────────────────────────────────────────

@pytest.mark.skipif(not _shark_reachable(), reason="shark not reachable")
def test_audit_log_token_exchange_endpoint():
    """GET /api/v1/audit-logs?action=oauth.token.exchanged must return 200."""
    admin_key = os.environ.get("SHARK_ADMIN_KEY", "")
    headers = {"Authorization": f"Bearer {admin_key}"} if admin_key else {}
    resp = requests.get(
        f"{BASE_URL}/api/v1/audit-logs",
        params={"action": "oauth.token.exchanged", "limit": "10"},
        headers=headers,
        timeout=5,
    )
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}: {resp.text[:200]}"
    data = resp.json()
    items = data.get("items") or data.get("audit_logs") or data.get("data") or (data if isinstance(data, list) else [])
    # Not asserting chain length here — data may not exist in CI baseline
    # Presence of endpoint + valid JSON is the live gate
    assert isinstance(items, list)


@pytest.mark.skipif(not _shark_reachable(), reason="shark not reachable")
def test_audit_log_has_chain_with_flatten():
    """If token-exchange events exist, at least one must have a flattenable act_chain."""
    admin_key = os.environ.get("SHARK_ADMIN_KEY", "")
    headers = {"Authorization": f"Bearer {admin_key}"} if admin_key else {}
    resp = requests.get(
        f"{BASE_URL}/api/v1/audit-logs",
        params={"action": "oauth.token.exchanged", "limit": "50"},
        headers=headers,
        timeout=5,
    )
    assert resp.status_code == 200
    data = resp.json()
    items = data.get("items") or data.get("audit_logs") or data.get("data") or (data if isinstance(data, list) else [])

    if not items:
        pytest.skip("No oauth.token.exchanged events found — run delegation demo first")

    chains_found = []
    for ev in items:
        meta = {}
        if isinstance(ev.get("metadata"), str):
            try:
                meta = json.loads(ev["metadata"])
            except Exception:
                pass
        elif isinstance(ev.get("metadata"), dict):
            meta = ev["metadata"]

        raw = ev.get("act_chain") or meta.get("act_chain")
        chain = resolve_chain(raw)
        if len(chain) >= 2:
            chains_found.append(chain)

    assert len(chains_found) >= 1, (
        "Expected at least one event with act_chain length >= 2 after flatten. "
        "Run: python tools/agent_demo_tester.py to populate delegation data."
    )
