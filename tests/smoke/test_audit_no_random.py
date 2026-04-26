"""
Smoke test: audit drawer no longer fabricates data via Math.random().

Assertions:
1. audit.tsx source has zero Math.random() calls (source-level check).
2. The built bundle contains 'act_chain' and the em-dash character '—' (real
   placeholders are wired up).
3. 'Math.random' and 'act_chain' do NOT appear within 200 chars of each other
   in the bundle (proxy: act_chain rendering no longer uses Math.random fallback).
"""

import pathlib
import re
import glob as _glob

REPO = pathlib.Path(__file__).parent.parent.parent
AUDIT_TSX = REPO / "admin" / "src" / "components" / "audit.tsx"
DIST_ASSETS = REPO / "internal" / "admin" / "dist" / "assets"


def _latest_main_js() -> pathlib.Path:
    candidates = sorted(
        DIST_ASSETS.glob("main-*.js"),
        key=lambda p: p.stat().st_mtime,
        reverse=True,
    )
    assert candidates, f"No main-*.js found under {DIST_ASSETS}"
    return candidates[0]


def test_audit_tsx_no_math_random():
    """Source file must contain zero Math.random() calls."""
    src = AUDIT_TSX.read_text(encoding="utf-8")
    count = src.count("Math.random")
    assert count == 0, (
        f"audit.tsx still contains {count} Math.random reference(s). "
        "All fabricated values must be removed."
    )


def test_bundle_has_act_chain_and_emdash():
    """Bundle must contain both 'act_chain' (real field) and em-dash '—' (placeholder)."""
    bundle = _latest_main_js().read_text(encoding="utf-8", errors="replace")
    assert "act_chain" in bundle, "Bundle missing 'act_chain' — feature may have been stripped."
    assert "—" in bundle or "—" in bundle, (
        "Bundle missing em-dash placeholder '—'. "
        "Graceful missing-field rendering may be broken."
    )


def test_bundle_act_chain_not_near_math_random():
    """
    'act_chain' and 'Math.random' must NOT appear within 200 characters of each
    other in the bundle. This guards against any re-introduction of random
    fallbacks in the delegation-chain rendering path.
    """
    bundle = _latest_main_js().read_text(encoding="utf-8", errors="replace")

    # Find all positions of act_chain
    for m in re.finditer(r"act_chain", bundle):
        pos = m.start()
        window = bundle[max(0, pos - 200): pos + 200]
        assert "Math.random" not in window, (
            f"Found 'Math.random' within 200 chars of 'act_chain' at bundle "
            f"offset {pos}. Fabricated data may still be rendered in the "
            f"delegation chain section."
        )
