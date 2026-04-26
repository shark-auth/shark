"""
Wave 1.7 Edit 4 — Get-Started impeccable rebuild smoke tests.

Three tests:
  1. test_get_started_renders_six_sections  — all six section-id markers present in built JS bundle
  2. test_code_block_python_snippet_present — the 10-liner DPoP snippet is in the bundle
  3. test_no_legacy_oauth_step_phrases      — old generic-OAuth onboarding phrasing is gone

These tests operate on the compiled admin bundle (internal/admin/dist/) — no live server needed.
D2 rule: NEVER run via ctx plugin. Orchestrator runs directly: pytest tests/smoke/ -v
"""
import os
import glob
import pytest

DIST_DIR = os.path.join(
    os.path.dirname(__file__), '..', '..', 'internal', 'admin', 'dist', 'assets'
)

def _load_bundle_text() -> str:
    """Load all JS assets from the compiled admin bundle into one string."""
    dist_assets = os.path.abspath(DIST_DIR)
    js_files = glob.glob(os.path.join(dist_assets, '*.js'))
    if not js_files:
        pytest.fail(
            f"No JS bundle files found in {dist_assets}. "
            "Run `cd admin && npx vite build` first."
        )
    text = ''
    for path in js_files:
        with open(path, 'r', encoding='utf-8', errors='replace') as f:
            text += f.read()
    return text


# ── Section marker test ───────────────────────────────────────────────────────

REQUIRED_SECTION_IDS = [
    'why-sharkauth',
    'sixty-second-setup',
    'your-first-agent',
    'dpop-token-ten-lines',
    'delegation-chain-demo',
    'next-steps',
]

def test_get_started_renders_six_sections():
    """All six section marker IDs must appear in the compiled admin bundle."""
    bundle = _load_bundle_text()
    missing = [sid for sid in REQUIRED_SECTION_IDS if sid not in bundle]
    assert not missing, (
        f"Missing section markers in bundle: {missing}\n"
        "Ensure get_started.tsx SECTIONS array contains all six IDs and the build is current."
    )


# ── DPoP Python 10-liner test ─────────────────────────────────────────────────

# Key phrases from the required 10-liner snippet — all must appear
PYTHON_SNIPPET_MARKERS = [
    'DPoPProver.generate',
    'get_token_with_dpop',
    'shark_auth.Client',
    'access_token',
]

def test_code_block_python_snippet_present():
    """The DPoP Python 10-liner snippet must be present in the compiled bundle."""
    bundle = _load_bundle_text()
    missing = [phrase for phrase in PYTHON_SNIPPET_MARKERS if phrase not in bundle]
    assert not missing, (
        f"DPoP Python snippet markers missing from bundle: {missing}\n"
        "Ensure the DPOP_PYTHON_SNIPPET constant in get_started.tsx contains the full 10-liner."
    )


# ── Legacy OAuth phrase removal test ─────────────────────────────────────────

# Phrases that identified the old generic-OAuth onboarding content.
# These must NOT appear in the new bundle — the rebuild supersedes them.
LEGACY_PHRASES = [
    'Mount SharkProvider',
    'Wire a Sign-in button',
    'Mount the callback route',
    'Configure proxy upstream',
    'Add a protect rule',
    'Brand the hosted login page',
    'Test the proxy URL',
    'npm install @sharkauth/react',
    'SharkCallback',
    'proxy.upstream',
    'proxy.brand',
    'sdk.provider',
    'sdk.signin',
    'sdk.callback',
    'sdk.session',
]

def test_no_legacy_oauth_step_phrases():
    """Old generic-OAuth onboarding step phrases must not appear in the rebuilt bundle."""
    bundle = _load_bundle_text()
    found = [phrase for phrase in LEGACY_PHRASES if phrase in bundle]
    assert not found, (
        f"Legacy OAuth step phrases still present in bundle: {found}\n"
        "The Wave 1.7 Edit 4 rebuild must fully replace the old 12-step OAuth-generic flow. "
        "Check get_started.tsx for leftover PROXY_TASKS / SDK_TASKS arrays."
    )
