"""F8 smoke tests — agent-platform-quickstart.md content validation.

These are static doc-lint tests: no server required.
"""

import re
from pathlib import Path

REPO_ROOT = Path(__file__).parent.parent.parent
DOC_PATH = REPO_ROOT / "docs" / "agent-platform-quickstart.md"
SDK_INIT = REPO_ROOT / "sdk" / "python" / "shark_auth" / "__init__.py"


def _doc() -> str:
    return DOC_PATH.read_text(encoding="utf-8")


def _sdk_exports() -> set:
    """Return the set of names in __all__ from the SDK __init__.py."""
    text = SDK_INIT.read_text(encoding="utf-8")
    # Extract quoted strings from __all__ = [...] block
    all_block_match = re.search(r"__all__\s*=\s*\[(.*?)\]", text, re.DOTALL)
    if not all_block_match:
        return set()
    block = all_block_match.group(1)
    return set(re.findall(r'"([^"]+)"', block))


class TestQuickstartDocExists:
    def test_file_exists(self):
        assert DOC_PATH.exists(), f"Missing: {DOC_PATH}"

    def test_file_nonempty(self):
        assert len(_doc()) > 500, "Doc appears too short (< 500 chars)"


class TestQuickstartDocStructure:
    def test_has_5_minute_integration_section(self):
        doc = _doc()
        assert re.search(
            r"5.minute integration",
            doc,
            re.IGNORECASE,
        ), "Missing '5-minute integration' section"

    def test_has_who_this_is_for(self):
        doc = _doc()
        assert "Who this is for" in doc, "Missing 'Who this is for' section"

    def test_has_mental_model(self):
        doc = _doc()
        assert re.search(r"mental model|3.minute", doc, re.IGNORECASE), (
            "Missing mental model section"
        )

    def test_has_five_layer_revocation(self):
        doc = _doc()
        assert re.search(r"five.layer revocation|5.layer", doc, re.IGNORECASE), (
            "Missing five-layer revocation section"
        )

    def test_has_comparison_table(self):
        doc = _doc()
        assert "Auth0" in doc and "Clerk" in doc and "Stytch" in doc, (
            "Missing comparison table entries"
        )

    def test_has_where_to_next(self):
        doc = _doc()
        assert re.search(r"where to next", doc, re.IGNORECASE), (
            "Missing 'Where to next' section"
        )

    def test_dashboard_links_present(self):
        doc = _doc()
        assert "/admin/agents" in doc, "Missing /admin/agents link"
        assert "/admin/delegation-chains" in doc, "Missing /admin/delegation-chains link"
        assert "/admin/vault" in doc, "Missing /admin/vault link"

    def test_cli_demo_command_present(self):
        doc = _doc()
        assert "shark demo delegation-with-trace" in doc, (
            "Missing shark demo delegation-with-trace reference"
        )


class TestQuickstartDocAudience:
    def test_does_not_lead_with_mcp_server_developers(self):
        """MCP server developers must NOT be the primary audience."""
        doc = _doc()
        lines = doc.splitlines()
        # Check first 20 non-empty lines don't open with MCP framing
        first_content = "\n".join(
            line for line in lines[:20] if line.strip()
        )
        assert not re.match(
            r"^#\s*MCP", first_content, re.IGNORECASE
        ), "Doc title must not start with 'MCP'"
        # Primary audience paragraph should not begin with "MCP server developers"
        assert not first_content.strip().startswith("MCP server developers"), (
            "Primary audience framing must not lead with 'MCP server developers'"
        )

    def test_mentions_customer_agents_framing(self):
        doc = _doc()
        assert re.search(
            r"customer.*(agent|own agent)|agent.*(per.customer|per customer)",
            doc,
            re.IGNORECASE,
        ), "Doc should describe the per-customer agent model"


class TestQuickstartDocSDKMethods:
    def test_sdk_exports_loadable(self):
        exports = _sdk_exports()
        assert len(exports) > 0, "Could not parse SDK __all__"

    def test_agents_client_referenced(self):
        doc = _doc()
        assert "AgentsClient" in doc, "AgentsClient not referenced in doc"

    def test_agents_client_in_sdk(self):
        exports = _sdk_exports()
        assert "AgentsClient" in exports, "AgentsClient not in SDK __all__"

    def test_users_client_referenced(self):
        doc = _doc()
        assert "UsersClient" in doc, "UsersClient not referenced in doc"

    def test_users_client_in_sdk(self):
        exports = _sdk_exports()
        assert "UsersClient" in exports, "UsersClient not in SDK __all__"

    def test_dpop_prover_referenced(self):
        doc = _doc()
        assert "DPoPProver" in doc, "DPoPProver not referenced in doc"

    def test_dpop_prover_in_sdk(self):
        exports = _sdk_exports()
        assert "DPoPProver" in exports, "DPoPProver not in SDK __all__"

    def test_exchange_token_referenced(self):
        doc = _doc()
        assert "exchange_token" in doc, "exchange_token not referenced in doc"

    def test_exchange_token_in_sdk(self):
        exports = _sdk_exports()
        assert "exchange_token" in exports, "exchange_token not in SDK __all__"

    def test_register_agent_referenced(self):
        doc = _doc()
        assert "register_agent" in doc, "register_agent not referenced in doc"

    def test_revoke_agents_referenced(self):
        doc = _doc()
        assert "revoke_agents" in doc, "revoke_agents not referenced in doc"

    def test_cascade_revoke_result_in_sdk(self):
        exports = _sdk_exports()
        assert "CascadeRevokeResult" in exports, "CascadeRevokeResult not in SDK __all__"


class TestQuickstartDocInstallSnippet:
    def test_no_bare_pip_install_shark_auth(self):
        """pip install shark-auth without git+ prefix must not appear."""
        doc = _doc()
        # Find all pip install lines
        pip_lines = [
            line.strip()
            for line in doc.splitlines()
            if "pip install" in line and "shark" in line.lower()
        ]
        for line in pip_lines:
            assert "git+" in line, (
                f"pip install line missing git+ prefix: {line!r}\n"
                "Use: pip install git+https://github.com/..."
            )
