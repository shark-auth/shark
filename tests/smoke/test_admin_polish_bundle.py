"""
Smoke tests for the admin polish bundle (Asks 3, 4, 5).

Ask 3: layout.tsx no longer contains the version chip span
Ask 4: get_started.tsx no longer contains "shark serve"
Ask 5: login.tsx contains the firstboot key banner OR the API handler exists
"""
import pathlib
import re

ROOT = pathlib.Path(__file__).parent.parent.parent
ADMIN_SRC = ROOT / "admin" / "src" / "components"
API_SRC = ROOT / "internal" / "api"


# ── Ask 3 ─────────────────────────────────────────────────────────────────────
def test_ask3_no_version_chip_in_layout():
    """layout.tsx must not contain the version badge span that overlapped the logo."""
    layout = (ADMIN_SRC / "layout.tsx").read_text(encoding="utf-8")
    # The old chip rendered version.startsWith('v') — that exact pattern should be gone
    assert "version.startsWith('v')" not in layout, (
        "Version chip still present in layout.tsx — Ask 3 not applied"
    )


# ── Ask 4 ─────────────────────────────────────────────────────────────────────
def test_ask4_no_shark_serve_in_get_started():
    """get_started.tsx must not mention 'shark serve' anywhere."""
    src = (ADMIN_SRC / "get_started.tsx").read_text(encoding="utf-8")
    assert "shark serve" not in src, (
        "'shark serve' still present in get_started.tsx — Ask 4 not applied"
    )


# ── Ask 5 ─────────────────────────────────────────────────────────────────────
def test_ask5_firstboot_banner_in_login():
    """login.tsx must contain the firstboot key banner UI."""
    login = (ADMIN_SRC / "login.tsx").read_text(encoding="utf-8")
    assert "firstbootKey" in login, (
        "First-boot key banner state not found in login.tsx — Ask 5 not applied"
    )
    assert "First-boot setup key" in login, (
        "First-boot setup key label not found in login.tsx — Ask 5 not applied"
    )


def test_ask5_firstboot_key_handler_exists():
    """admin_bootstrap_handlers.go must contain the handleFirstbootKey function."""
    handler = (API_SRC / "admin_bootstrap_handlers.go").read_text(encoding="utf-8")
    assert "handleFirstbootKey" in handler, (
        "handleFirstbootKey not found in admin_bootstrap_handlers.go — Ask 5 not applied"
    )
    assert "firstbootKeyPath" in handler, (
        "firstbootKeyPath helper not found in admin_bootstrap_handlers.go — Ask 5 not applied"
    )


# ── Ask 1 ─────────────────────────────────────────────────────────────────────
def test_ask1_email_config_handler_exists():
    """email_config_handlers.go must implement GET+PATCH /admin/email-config."""
    handler = (API_SRC / "email_config_handlers.go").read_text(encoding="utf-8")
    assert "handleGetEmailConfig" in handler, "handleGetEmailConfig missing"
    assert "handlePatchEmailConfig" in handler, "handlePatchEmailConfig missing"
    assert "verify_redirect_url" in handler, "verify_redirect_url field missing"
    assert "reset_redirect_url" in handler, "reset_redirect_url field missing"
    assert "magic_link_redirect_url" in handler, "magic_link_redirect_url field missing"


def test_ask1_email_config_routes_registered():
    """router.go must register the email-config GET and PATCH routes."""
    router = (API_SRC / "router.go").read_text(encoding="utf-8")
    assert "email-config" in router, "email-config routes not registered in router.go"


# ── Ask 2 ─────────────────────────────────────────────────────────────────────
def test_ask2_dcr_explainer_in_overview():
    """overview.tsx must contain the agent self-register / DCR explainer card."""
    overview = (ADMIN_SRC / "overview.tsx").read_text(encoding="utf-8")
    assert "Self-Registration" in overview or "self-register" in overview.lower(), (
        "Agent self-register explainer card not found in overview.tsx — Ask 2 not applied"
    )
    assert "RFC 7591" in overview, (
        "RFC 7591 DCR reference not found in overview.tsx — Ask 2 not applied"
    )
