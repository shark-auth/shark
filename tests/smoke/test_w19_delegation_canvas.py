"""
W19 smoke test: Per-agent Delegations canvas tab.

Assumes admin dist has already been built by the orchestrator.
Checks bundle contents for required SVG primitives and absence of chart libs.
"""
import glob
import os
import pytest

DIST_ASSETS = os.path.join(
    os.path.dirname(__file__), "..", "..", "internal", "admin", "dist", "assets"
)
SRC_AGENTS = os.path.join(
    os.path.dirname(__file__), "..", "..", "admin", "src", "components", "agents_manage.tsx"
)


def find_main_js():
    pattern = os.path.join(DIST_ASSETS, "main-*.js")
    files = glob.glob(pattern)
    assert files, f"No main-*.js found in {DIST_ASSETS}. Run 'cd admin && npx vite build' first."
    # pick largest (most likely the real bundle, not a chunk)
    return max(files, key=os.path.getsize)


@pytest.fixture(scope="module")
def bundle():
    path = find_main_js()
    with open(path, encoding="utf-8", errors="replace") as f:
        return f.read()


def test_delegations_tab_label_present(bundle):
    """Bundle must include the 'Delegations' tab label string."""
    assert "Delegations" in bundle, (
        "'Delegations' tab label not found in bundle — tab may not have been added."
    )


def test_svg_arrow_marker_present(bundle):
    """Bundle must include the SVG arrow marker (id or markerEnd reference)."""
    has_marker_id = 'id="arrow"' in bundle or "id='arrow'" in bundle or 'id:\\"arrow\\"' in bundle
    has_marker_end = 'url(#arrow)' in bundle or "url(#arrow)" in bundle
    # In minified output the strings may appear as template literals or plain strings
    has_marker_any = (
        "arrow" in bundle and ("marker" in bundle or "markerEnd" in bundle or "markerend" in bundle)
    )
    assert has_marker_id or has_marker_end or has_marker_any, (
        "SVG arrow marker not found in bundle — DelegationsTab SVG primitives may be missing."
    )


def test_no_d3_in_bundle(bundle):
    """Bundle must NOT include d3 (chart library)."""
    # d3 typically exports 'select', 'selectAll', etc. from 'd3-selection'
    assert '"d3"' not in bundle and "'d3'" not in bundle and "from 'd3'" not in bundle, (
        "d3 detected in bundle — violates .impeccable.md no-chart-libs rule."
    )


def test_no_recharts_in_bundle(bundle):
    """Bundle must NOT include recharts."""
    assert "recharts" not in bundle.lower(), (
        "recharts detected in bundle — violates .impeccable.md no-chart-libs rule."
    )


def test_no_viz_lib_in_bundle(bundle):
    """Bundle must NOT include mermaid or viz-js."""
    lower = bundle.lower()
    assert "mermaid" not in lower, "mermaid detected in bundle."
    assert "viz-js" not in lower and "graphviz" not in lower, "viz-js/graphviz detected in bundle."


def test_agents_manage_no_chart_imports():
    """agents_manage.tsx source must not import from d3/recharts/mermaid."""
    with open(SRC_AGENTS, encoding="utf-8") as f:
        src = f.read()
    forbidden = ["from 'd3'", 'from "d3"', "from 'recharts'", 'from "recharts"',
                 "from 'mermaid'", 'from "mermaid"']
    for imp in forbidden:
        assert imp not in src, (
            f"Forbidden import '{imp}' found in agents_manage.tsx."
        )


def test_agents_manage_has_delegations_tab():
    """agents_manage.tsx source must declare the 'delegations' tab."""
    with open(SRC_AGENTS, encoding="utf-8") as f:
        src = f.read()
    assert "'delegations'" in src or '"delegations"' in src, (
        "Tab id 'delegations' not found in agents_manage.tsx."
    )
    assert "DelegationsTab" in src, (
        "DelegationsTab component not found in agents_manage.tsx."
    )
