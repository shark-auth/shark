"""
F10 — Scalar UI + OpenAPI spec smoke tests.

Verifies:
  - GET /api/docs returns 200 + Scalar HTML
  - GET /api/openapi.yaml returns 200 + valid YAML
  - Parsed spec has openapi: 3.x
  - Parsed spec has at least 25 paths
  - Moat endpoints present: /oauth/register, /oauth/token,
    /api/v1/agents/{id}/policies, /api/v1/agents/{id}/rotate-dpop-key
  - Excluded (cut) endpoints absent: /proxy/, bulk-revoke-pattern
  - Stale ref sweep: zero hits in docs/ for shark init, install.sh, bare pip install shark-auth
"""

import os
import re

import pytest
import requests
import yaml

BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")
REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
DOCS_DIR = os.path.join(REPO_ROOT, "docs")


# ─── Scalar UI ────────────────────────────────────────────────────────────────

def test_api_docs_returns_200():
    """GET /api/docs must respond 200."""
    resp = requests.get(f"{BASE_URL}/api/docs", timeout=10)
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}"


def test_api_docs_contains_scalar():
    """GET /api/docs body must reference Scalar (the API reference renderer)."""
    resp = requests.get(f"{BASE_URL}/api/docs", timeout=10)
    assert resp.status_code == 200
    body = resp.text.lower()
    assert "scalar" in body, "Expected 'scalar' keyword in /api/docs HTML body"


# ─── OpenAPI YAML ─────────────────────────────────────────────────────────────

def test_api_openapi_yaml_returns_200():
    """GET /api/openapi.yaml must respond 200."""
    resp = requests.get(f"{BASE_URL}/api/openapi.yaml", timeout=10)
    assert resp.status_code == 200, f"Expected 200, got {resp.status_code}"


def test_api_openapi_yaml_parses_as_valid_yaml():
    """GET /api/openapi.yaml body must parse as valid YAML."""
    resp = requests.get(f"{BASE_URL}/api/openapi.yaml", timeout=10)
    assert resp.status_code == 200
    parsed = yaml.safe_load(resp.text)
    assert isinstance(parsed, dict), "Parsed YAML must be a mapping"


@pytest.fixture(scope="module")
def openapi_spec():
    """Fetch and parse the OpenAPI spec once for all spec-content tests."""
    resp = requests.get(f"{BASE_URL}/api/openapi.yaml", timeout=10)
    assert resp.status_code == 200
    return yaml.safe_load(resp.text)


def test_spec_openapi_version(openapi_spec):
    """Spec must declare openapi: 3.x.x (3.0.x or 3.1.x)."""
    version = openapi_spec.get("openapi", "")
    assert str(version).startswith("3."), (
        f"Expected openapi version 3.x, got: {version!r}"
    )


def test_spec_has_at_least_25_paths(openapi_spec):
    """Spec must document at least 25 paths."""
    paths = openapi_spec.get("paths", {})
    count = len(paths)
    assert count >= 25, f"Expected at least 25 paths, found {count}"


# ─── Moat endpoints ──────────────────────────────────────────────────────────

MOAT_ENDPOINTS = [
    "/oauth/register",
    "/oauth/token",
    "/api/v1/agents/{id}/policies",
    "/api/v1/agents/{id}/rotate-dpop-key",
]


@pytest.mark.parametrize("path", MOAT_ENDPOINTS)
def test_moat_endpoint_present(openapi_spec, path):
    """Each moat endpoint must appear in the spec paths."""
    paths = openapi_spec.get("paths", {})
    assert path in paths, (
        f"Moat endpoint {path!r} missing from spec. "
        f"Found paths: {sorted(paths.keys())}"
    )


def test_oauth_token_has_token_exchange_documented(openapi_spec):
    """POST /oauth/token description must mention token-exchange grant."""
    token_path = openapi_spec.get("paths", {}).get("/oauth/token", {})
    post_op = token_path.get("post", {})
    description = post_op.get("description", "") + post_op.get("summary", "")
    assert "token-exchange" in description.lower() or "urn:ietf:params:oauth:grant-type:token-exchange" in description, (
        "POST /oauth/token must document RFC 8693 token-exchange grant"
    )


# ─── Cut surfaces must be absent ─────────────────────────────────────────────

def test_proxy_endpoints_absent(openapi_spec):
    """Spec must NOT contain any /proxy/ paths (cut for v0.2)."""
    paths = openapi_spec.get("paths", {})
    proxy_paths = [p for p in paths if "/proxy" in p]
    assert not proxy_paths, (
        f"Spec contains /proxy/ paths (cut for v0.2): {proxy_paths}"
    )


def test_bulk_revoke_pattern_absent(openapi_spec):
    """Spec must NOT contain /api/v1/policies/bulk-revoke-pattern (cut for v0.2)."""
    paths = openapi_spec.get("paths", {})
    bulk = [p for p in paths if "bulk-revoke-pattern" in p]
    assert not bulk, (
        f"Spec contains bulk-revoke-pattern path (cut for v0.2): {bulk}"
    )


# ─── Stale ref sweep ─────────────────────────────────────────────────────────

def _collect_md_files(root: str):
    """Walk root and yield .md file paths (skip docs/internal/ and proxy_v1_5/)."""
    skip_dirs = {"internal", "proxy_v1_5"}
    for dirpath, dirnames, filenames in os.walk(root):
        # Prune internal/proxy dirs — they are historical docs, not user-facing
        dirnames[:] = [d for d in dirnames if d not in skip_dirs]
        for fname in filenames:
            if fname.endswith(".md"):
                yield os.path.join(dirpath, fname)


def test_no_shark_init_in_public_docs():
    """Public docs must not reference `shark init` (YAML config is dead since W17)."""
    hits = []
    for fpath in _collect_md_files(DOCS_DIR):
        with open(fpath, encoding="utf-8") as f:
            for lineno, line in enumerate(f, 1):
                if "shark init" in line:
                    hits.append(f"{fpath}:{lineno}: {line.rstrip()}")
    assert not hits, "Found `shark init` references in public docs:\n" + "\n".join(hits)


def test_no_install_sh_in_public_docs():
    """Public docs must not reference sharkauth.dev/install.sh (hallucinated URL)."""
    hits = []
    for fpath in _collect_md_files(DOCS_DIR):
        with open(fpath, encoding="utf-8") as f:
            for lineno, line in enumerate(f, 1):
                if "sharkauth.dev" in line or "install.sh" in line:
                    hits.append(f"{fpath}:{lineno}: {line.rstrip()}")
    assert not hits, (
        "Found sharkauth.dev/install.sh references in public docs:\n"
        + "\n".join(hits)
    )


def test_no_bare_pip_install_shark_auth_in_public_docs():
    """Public docs must not have bare `pip install shark-auth` (without git+)."""
    # Pattern: pip install shark (without git+)
    pattern = re.compile(r"pip install shark(?!.*git\+)", re.IGNORECASE)
    hits = []
    for fpath in _collect_md_files(DOCS_DIR):
        with open(fpath, encoding="utf-8") as f:
            for lineno, line in enumerate(f, 1):
                if pattern.search(line) and "git+" not in line:
                    hits.append(f"{fpath}:{lineno}: {line.rstrip()}")
    assert not hits, (
        "Found bare `pip install shark-auth` (without git+) in public docs:\n"
        + "\n".join(hits)
    )
