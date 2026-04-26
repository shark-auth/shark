"""
F9 smoke tests — README.md content validation.

These tests are purely file-based (no server required). They validate that
the rewritten README meets the launch readiness spec: correct framing, no
banned phrases, required sections present, ASCII logo intact.
"""

import re
from pathlib import Path

README = Path(__file__).parent.parent.parent / "README.md"


def _text() -> str:
    return README.read_text(encoding="utf-8")


def test_readme_exists():
    assert README.exists(), "README.md not found at repo root"


def test_contains_ascii_logo():
    """Known unique line from the preserved ASCII logo."""
    text = _text()
    assert "███████╗██╗  ██╗ █████╗ ██████╗" in text, (
        "ASCII logo line not found — logo must be preserved from original README"
    )


def test_contains_five_layer_revocation():
    """Core wedge phrase must appear verbatim."""
    assert "five-layer revocation" in _text().lower(), (
        "Phrase 'five-layer revocation' not found in README"
    )


def test_no_bare_pip_install_shark_auth():
    """pip install shark-auth is only allowed with git+ prefix."""
    text = _text()
    # Find all occurrences of pip install shark-auth
    matches = re.findall(r"pip install\s+shark-auth", text)
    for m in matches:
        # Each match must be preceded by git+ somewhere in the same line
        pass
    # More direct: assert bare form is absent
    assert "pip install shark-auth" not in text, (
        "Bare 'pip install shark-auth' found — use git+ prefix or remove"
    )


def test_git_plus_install_present():
    """Dogfood install uses git+ prefix."""
    assert "pip install git+" in _text(), (
        "Expected 'pip install git+' form for SDK install"
    )


def test_no_hallucinated_install_sh():
    """sharkauth.dev/install.sh was a hallucinated URL — must not appear."""
    assert "sharkauth.dev/install.sh" not in _text(), (
        "Hallucinated URL sharkauth.dev/install.sh must not appear in README"
    )


def test_no_auth0_charges_framing():
    """'Auth0 charges' dollar-amount framing is banned per founder decision."""
    assert "Auth0 charges" not in _text(), (
        "'Auth0 charges' framing found — drop it per F9 spec"
    )


def test_roadmap_v01():
    assert "v0.1" in _text(), "Roadmap must mention v0.1"


def test_roadmap_v02():
    assert "v0.2" in _text(), "Roadmap must mention v0.2"


def test_roadmap_v03():
    assert "v0.3" in _text(), "Roadmap must mention v0.3"


def test_apache_license_mentioned():
    """License section must name Apache-2.0."""
    text = _text()
    assert "Apache-2.0" in text, "Apache-2.0 not mentioned in README"


def test_tagline_present():
    """Required tagline must appear."""
    assert "auth for products that give customers their own agents" in _text(), (
        "Required tagline not found"
    )


def test_demo_placeholder_cold_start():
    """demo: 8-second cold-start placeholder must be embedded."""
    assert "demo: 8-second cold-start" in _text(), (
        "Demo placeholder 'demo: 8-second cold-start' not found"
    )


def test_five_layers_table():
    """All five layers must appear (at least L1 through L5)."""
    text = _text()
    for layer in ("L1", "L2", "L3", "L4", "L5"):
        assert layer in text, f"Layer {layer} not found in five-layers table"


def test_why_now_footer():
    """YC reviewer hook paragraph must be present."""
    assert "Why Now" in _text() or "Why now" in _text(), (
        "YC reviewer hook 'Why Now' footer not found"
    )


def test_quickstart_no_pip_install():
    """Quickstart section must not contain pip install."""
    text = _text()
    quickstart_start = text.find("## Quickstart")
    quickstart_end = text.find("\n## ", quickstart_start + 1)
    if quickstart_end == -1:
        quickstart_end = len(text)
    quickstart = text[quickstart_start:quickstart_end]
    assert "pip install" not in quickstart, (
        "Quickstart section must not contain pip install"
    )


def test_quickstart_no_npm_install():
    """Quickstart section must not contain npm install."""
    text = _text()
    quickstart_start = text.find("## Quickstart")
    quickstart_end = text.find("\n## ", quickstart_start + 1)
    if quickstart_end == -1:
        quickstart_end = len(text)
    quickstart = text[quickstart_start:quickstart_end]
    assert "npm install" not in quickstart, (
        "Quickstart section must not contain npm install"
    )


def test_rfc_8693_mentioned():
    """RFC 8693 (token exchange / delegation) must be cited."""
    assert "RFC 8693" in _text() or "rfc 8693" in _text().lower(), (
        "RFC 8693 not mentioned — delegation chain moat requires this citation"
    )


def test_dpop_mentioned():
    """DPoP must be cited (RFC 9449 or the acronym)."""
    assert "DPoP" in _text(), "DPoP not mentioned in README"
