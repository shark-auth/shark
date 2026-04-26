"""
Smoke test: delegation-chains page bundle-shape verification.

This test does NOT start a browser or require a running server.
It verifies that the Vite bundle produced by `cd admin && npx vite build`
contains the expected markers for the delegation-chains route.

Run after building admin:
    cd admin && npx vite build
    pytest tests/smoke/test_delegation_chains_page.py -v
"""

import glob
import os
import re
import sys

DIST_DIR = os.path.join(os.path.dirname(__file__), "..", "..", "internal", "admin", "dist")


def _read_js_bundle() -> str:
    """Return the concatenated content of all JS chunks in the dist/assets folder."""
    assets_dir = os.path.join(DIST_DIR, "assets")
    js_files = glob.glob(os.path.join(assets_dir, "*.js"))
    if not js_files:
        raise FileNotFoundError(
            f"No JS files found in {assets_dir}. "
            "Run `cd admin && npx vite build` first."
        )
    parts = []
    for path in js_files:
        with open(path, "r", encoding="utf-8", errors="replace") as f:
            parts.append(f.read())
    return "\n".join(parts)


def test_dist_exists():
    """The dist directory must exist (admin has been built)."""
    assert os.path.isdir(DIST_DIR), (
        f"Admin dist directory not found at {DIST_DIR}. "
        "Run `cd admin && npx vite build` first."
    )


def test_index_html_exists():
    """dist/index.html must be present."""
    index = os.path.join(DIST_DIR, "index.html")
    assert os.path.isfile(index), f"index.html missing from {DIST_DIR}"


def test_delegation_chains_route_in_bundle():
    """The string 'delegation-chains' must appear in the JS bundle (router map)."""
    bundle = _read_js_bundle()
    assert "delegation-chains" in bundle, (
        "'delegation-chains' not found in JS bundle. "
        "Check that App.tsx was updated and admin was rebuilt."
    )


def test_delegation_chains_title_in_bundle():
    """The page title 'Delegation Chains' must appear in the JS bundle."""
    bundle = _read_js_bundle()
    assert "Delegation Chains" in bundle, (
        "'Delegation Chains' title not found in JS bundle. "
        "Check delegation_chains.tsx and layout.tsx were updated and admin was rebuilt."
    )


def test_delegation_chains_component_exported():
    """The DelegationChains symbol must be present in the bundle."""
    bundle = _read_js_bundle()
    # Vite may mangle 'DelegationChains' — check for the route string instead.
    # We also verify the action filter string used in the data fetch.
    assert "oauth.token.exchanged" in bundle, (
        "'oauth.token.exchanged' action filter not found in bundle. "
        "The DelegationChains component may not have been included."
    )


def test_delegation_chains_empty_state_text():
    """Empty-state CLI hint must be present in the bundle."""
    bundle = _read_js_bundle()
    assert "delegation-with-trace" in bundle, (
        "Empty state CLI hint 'delegation-with-trace' not found in bundle."
    )


def test_delegation_chains_backend_todo_comment():
    """Wave 1.6 backend TODO comment should be present in the source file."""
    src = os.path.join(
        os.path.dirname(__file__), "..", "..", "admin", "src", "components", "delegation_chains.tsx"
    )
    src = os.path.normpath(src)
    assert os.path.isfile(src), f"delegation_chains.tsx not found at {src}"
    with open(src, "r", encoding="utf-8") as f:
        content = f.read()
    assert "Wave 1.6 backend" in content, (
        "Expected 'Wave 1.6 backend' TODO comment in delegation_chains.tsx"
    )


def test_nav_entry_in_layout():
    """layout.tsx must contain the delegation-chains nav entry."""
    layout = os.path.join(
        os.path.dirname(__file__), "..", "..", "admin", "src", "components", "layout.tsx"
    )
    layout = os.path.normpath(layout)
    assert os.path.isfile(layout), f"layout.tsx not found at {layout}"
    with open(layout, "r", encoding="utf-8") as f:
        content = f.read()
    assert "delegation-chains" in content, (
        "'delegation-chains' not found in layout.tsx. Nav entry missing."
    )
    assert "Delegation Chains" in content, (
        "'Delegation Chains' label not found in layout.tsx."
    )


def test_app_tsx_imports_component():
    """App.tsx must import DelegationChains."""
    app = os.path.join(
        os.path.dirname(__file__), "..", "..", "admin", "src", "components", "App.tsx"
    )
    app = os.path.normpath(app)
    assert os.path.isfile(app), f"App.tsx not found at {app}"
    with open(app, "r", encoding="utf-8") as f:
        content = f.read()
    assert "DelegationChains" in content, (
        "'DelegationChains' not found in App.tsx. Import or route map missing."
    )
    assert "'delegation-chains'" in content or '"delegation-chains"' in content, (
        "Route key 'delegation-chains' not found in App.tsx component map."
    )
