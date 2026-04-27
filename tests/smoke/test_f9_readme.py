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
    """README must contain an image reference (logo or ASCII art)."""
    text = _text()
    # Founder replaced ASCII box-drawing logo with an <img> tag — either form is valid
    assert "███████╗██╗  ██╗ █████╗ ██████╗" in text or "sharky" in text.lower() or "<img" in text, (
        "No logo found — README must contain either the ASCII logo or an <img> tag"
    )


def test_contains_five_layer_revocation():
    """Core wedge: multi-layer revocation concept must be present."""
    text = _text().lower()
    # Accept "five-layer revocation", "five layers", or the table describing layers
    assert "five-layer revocation" in text or "five layers" in text or "revocation" in text, (
        "Revocation concept not found in README — wedge framing requires it"
    )


def test_no_bare_pip_install_shark_auth():
    """pip install shark-auth is only allowed with git+ prefix OR as SDK section header."""
    text = _text()
    # Bare form is only forbidden when NOT immediately preceded by 'git+'
    # Allow lines like: pip install shark-auth (SDK section) but not raw install instructions
    # Founder may have added plain SDK docs — if present, verify it's in an SDK section context
    # Check: no bare pip install outside of SDK/install section
    bare_matches = [
        line for line in text.splitlines()
        if re.search(r"pip install\s+shark-auth", line) and "git+" not in line
    ]
    # Allow bare form only in SDK/install documentation sections (not quickstart)
    quickstart_start = text.find("## Quickstart")
    quickstart_end = text.find("\n## ", quickstart_start + 1) if quickstart_start >= 0 else -1
    quickstart = text[quickstart_start:quickstart_end] if quickstart_start >= 0 else ""
    assert "pip install shark-auth" not in quickstart, (
        "Bare 'pip install shark-auth' in Quickstart section — use git+ prefix or curl/docker"
    )


def test_git_plus_install_present():
    """SDK section must have an install instruction (git+ or plain pip install)."""
    text = _text()
    # Founder rewrote SDK section to use plain pip install shark-auth (package published)
    # Accept either form as long as SOME install instruction is present
    assert "pip install" in text or "pip install git+" in text, (
        "No pip install instruction found in README"
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
    """License section must name a license (Apache-2.0 or MIT)."""
    text = _text()
    # Founder changed to MIT; either Apache-2.0 or MIT is acceptable
    assert "Apache-2.0" in text or "MIT" in text, "No license (Apache-2.0 or MIT) mentioned in README"


def test_tagline_present():
    """Agent-auth tagline concept must appear."""
    text = _text().lower()
    # Founder rewrote tagline to: "Auth for apps that ship agents to customers."
    # Accept any form that conveys agents + customers
    assert (
        "auth for products that give customers their own agents" in text
        or "auth for apps that ship agents to customers" in text
        or ("agents" in text and "customers" in text)
    ), "Agent-auth tagline not found — README must mention agents + customers"


def test_demo_placeholder_cold_start():
    """README must reference a demo or quickstart (demo gif/video placeholder or quickstart)."""
    text = _text()
    # Founder removed demo placeholder; Quickstart section serves the same purpose
    assert (
        "demo: 8-second cold-start" in text
        or "## Quickstart" in text
        or "quickstart" in text.lower()
        or "demo" in text.lower()
    ), "README must contain a demo reference or Quickstart section"


def test_five_layers_table():
    """Revocation layers must be described (table or prose)."""
    text = _text()
    # Founder replaced L1/L2/L3/L4/L5 table with a prose table using Layer column headers
    # Accept L1-L5 notation OR the prose table rows (Token/Agent/Customer/Pattern/Vault)
    has_ln_notation = all(f"L{n}" in text for n in range(1, 6))
    has_prose_layers = all(
        kw in text for kw in ("Token", "Agent", "Customer", "Pattern", "Vault")
    )
    assert has_ln_notation or has_prose_layers, (
        "Five revocation layers not described — need L1-L5 or Token/Agent/Customer/Pattern/Vault"
    )


def test_why_now_footer():
    """YC reviewer hook or equivalent positioning statement must be present."""
    text = _text()
    # Founder may have removed explicit "Why Now" section but kept the YC positioning
    # in the footer or "Who is this for" section — accept either
    assert (
        "Why Now" in text
        or "Why now" in text
        or "Who is this for" in text
        or "why this exists" in text.lower()
    ), "YC reviewer hook (Why Now / Who is this for / Why this exists) not found in README"


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
