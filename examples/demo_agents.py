"""
SharkClaw by Acme — The high-end TUI for SharkAuth Agent Delegation.

Upgraded for Launch Video & YC Demo:
- HYBRID MODE: Real SharkAuth Server calls + Deterministic (Mock) LLM responses.
- Premium Dark Aesthetic (Monochrome + Status Colors)
- Auto-Polling Revocation (The Terminal "Explodes" when you revoke in UI)
- Technical "Wow" Logs: DPoP jkt thumbprints and nested 'act' claim snippets.
"""

import os
import sys
import asyncio
import json
import time
from datetime import datetime

from textual.app import App, ComposeResult
from textual.containers import ScrollableContainer, Horizontal, Vertical
from textual.widgets import Header, Footer, Static, Input, ListItem, ListView, Label
from textual.reactive import reactive
from rich.text import Text
from rich.panel import Panel
from rich.markdown import Markdown

from shark_auth import Client, DPoPProver, MayActClient, OAuthClient

# --- CONFIG ---
class Config:
    BASE_URL = os.environ.get("SHARK_BASE_URL", "http://localhost:8080")
    ADMIN_KEY = os.environ.get("SHARK_ADMIN_KEY", "")
    # In this version, we always call SharkAuth, but we don't need OpenRouter keys 
    # because the LLM logic is deterministic for the "Hybrid" demo.

# --- UI COMPONENTS ---

class AgentStatus(ListItem):
    status = reactive("Provisioning...")

    def __init__(self, name, id_suffix, **kwargs):
        super().__init__(**kwargs)
        self.agent_name = name
        self.id_suffix = id_suffix

    def compose(self) -> ComposeResult:
        yield Label(f" [bold]{self.agent_name}[/] [dim]({self.id_suffix})[/]")
        yield Label(self.status, id=f"status-label-{self.id_suffix}")

    def watch_status(self, new_status: str) -> None:
        try:
            label = self.query_one(f"#status-label-{self.id_suffix}", Label)
            if "Revoked" in new_status or "SEVERED" in new_status:
                label.update(f"[bold red]{new_status}[/]")
                self.styles.background = "#2a0000"
            elif "Alive" in new_status:
                label.update(f"[bold green]● {new_status}[/]")
                self.styles.background = "transparent"
            elif "Thinking" in new_status:
                label.update(f"[bold yellow]◌ {new_status}[/]")
            else:
                label.update(f"[dim]{new_status}[/]")
        except: pass

class ChatMessage(Static):
    def __init__(self, sender, message, color="white", **kwargs):
        super().__init__(**kwargs)
        self.sender = sender
        self.message = message
        self.color = color

    def render(self) -> Panel:
        return Panel(
            Text.from_markup(f"[{self.color}][bold]{self.sender}:[/] {self.message}[/]"),
            border_style="dim" if self.color == "white" else self.color,
            padding=(0, 1)
        )

# --- MAIN APP ---

class SharkClaw(App):
    TITLE = "SharkClaw"
    SUB_TITLE = "Sovereign Agentic Auth"
    CSS = """
    Screen {
        background: #000000;
    }
    #sidebar {
        width: 35;
        background: #080808;
        border-right: tall #1a1a1a;
        padding: 1;
    }
    #chat-container {
        height: 1fr;
        padding: 1;
    }
    Input {
        dock: bottom;
        border: none;
        background: #050505;
        color: #ffffff;
    }
    ListItem {
        layout: horizontal;
        height: auto;
        padding: 1 1;
        margin: 0 0 1 0;
        border: solid #1a1a1a;
    }
    AgentStatus Label {
        width: 1fr;
    }
    #status-msg {
        dock: bottom;
        height: 1;
        background: #1a1a1a;
        color: #888;
        padding: 0 1;
        text-align: center;
    }
    """

    BINDINGS = [
        ("q", "quit", "Quit"),
        ("ctrl+r", "run_demo", "Run Demo"),
    ]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        with Horizontal():
            with Vertical(id="sidebar"):
                yield Label("[bold white]SHARKCLAW v0.9.0[/]\n[dim]Sovereign Agentic Auth[/]\n")
                yield Label("\n[dim]ACTIVE AGENTS[/]", variant="title")
                yield ListView(id="agent-list")
            with Vertical():
                yield ScrollableContainer(id="chat-container")
                yield Label("Press CTRL+R to start the 3-hop delegation demo", id="status-msg")
                yield Input(placeholder="System ready...", id="repl-input")
        yield Footer()

    async def on_mount(self) -> None:
        self.query_one("#repl-input").focus()
        self.append_chat("SYSTEM", "SharkClaw initialized. Ready for cryptographic delegation.", "dim")

    def append_chat(self, sender, message, color="cyan"):
        container = self.query_one("#chat-container")
        msg = ChatMessage(sender, message, color)
        container.mount(msg)
        msg.scroll_visible()

    def update_status_bar(self, text):
        self.query_one("#status-msg", Label).update(text)

    async def action_run_demo(self) -> None:
        if not Config.ADMIN_KEY:
            self.append_chat("ERROR", "Missing SHARK_ADMIN_KEY.", "red")
            return

        self.append_chat("SYSTEM", "Initiating 3-hop Hybrid Flow (Real Auth + Mock LLM)...", "blue")
        
        try:
            await self.run_hybrid_demo()
        except ImportError as e:
            self.append_chat("ERROR", f"Missing Python dependency: {e}. Run: python -m pip install shark-auth textual", "red")
        except Exception as e:
            self.append_chat("FATAL", f"Demo interrupted: {str(e)}", "red")

    async def run_hybrid_demo(self):
        from shark_auth import Client, DPoPProver, OAuthClient
        client = Client(base_url=Config.BASE_URL, token=Config.ADMIN_KEY)
        oauth = OAuthClient(base_url=Config.BASE_URL)
        may_act = MayActClient(base_url=Config.BASE_URL, admin_api_key=Config.ADMIN_KEY)
        pm_prover = DPoPProver.generate()
        fetcher_prover = DPoPProver.generate()

        # --- 1. PROVISIONING (REAL) ---
        self.update_status_bar("REGISTERING AGENTS IN SHARKAUTH...")
        u_email = f"alice_{int(time.time())}@acme.com"
        user = client.users.create_user(u_email, name="Alice (CEO)", email_verified=True)
        user_id = user["id"]
        self.append_chat("AUTH", f"Parent user created: [dim]{user_id}[/]", "green")
        
        # Root Agent: PM-Orchestrator
        pm = client.agents.register_agent(app_id="acme", name="PM-Orchestrator", created_by=user_id,
                                         scopes=["vault:read", "market:analyze", "data:fetch"],
                                         auth_method="client_secret_post",
                                         grant_types=["client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"])
        self.add_agent_to_sidebar("PM-Orchestrator", pm["id"][:4])
        self.update_agent_status(pm["id"][:4], "Alive")

        # Leaf Agent: Data-Fetcher
        fetcher = client.agents.register_agent(app_id="acme", name="Data-Fetcher", created_by=user_id,
                                              scopes=["data:fetch"],
                                              auth_method="client_secret_post",
                                              token_endpoint_auth_method="client_secret_post",
                                              grant_types=["urn:ietf:params:oauth:grant-type:token-exchange"])
        self.add_agent_to_sidebar("Data-Fetcher", fetcher["id"][:4])
        self.update_agent_status(fetcher["id"][:4], "Alive")

        owned = client.users.list_agents(user_id, filter="created", limit=10)
        owned_ids = {a.get("id") for a in owned.data}
        if pm["id"] not in owned_ids or fetcher["id"] not in owned_ids:
            raise RuntimeError("agent ownership check failed: created_by did not bind both agents to Alice")
        self.append_chat("AUTH", "Agents registered on behalf of Alice. [dim]created_by verified via users.list_agents(filter='created')[/]", "green")

        may_act.create(
            from_id=fetcher["client_id"],
            to_id=pm["client_id"],
            max_hops=2,
            scopes=["data:fetch"],
        )
        self.append_chat("AUTH", "Delegation policy installed: [dim]Data-Fetcher may act for PM-Orchestrator[/]", "green")
        
        await asyncio.sleep(1)

        # --- 2. DELEGATION (REAL AUTH) ---
        self.append_chat("ALICE", "PM, run the supply-chain reliability scan.", "white")
        await asyncio.sleep(1.5)
        
        self.update_agent_status(pm["id"][:4], "Thinking...")
        pm_token = oauth.get_token_with_dpop(grant_type="client_credentials", dpop_prover=pm_prover,
                                           client_id=pm["client_id"], client_secret=pm["client_secret"],
                                           scope="vault:read market:analyze data:fetch")
        
        jkt = pm_token.raw.get("cnf", {}).get("jkt", "N/A")
        self.append_chat("PM", f"Root DPoP Token acquired. [dim]jkt:{jkt[:12]}...[/]", "magenta")
        await asyncio.sleep(2)
        
        self.update_agent_status(pm["id"][:4], "Alive")
        self.update_agent_status(fetcher["id"][:4], "Thinking...")
        self.append_chat("PM", "Exchanging for leaf-token (depth=2). Lineage: [dim]PM[/]", "magenta")

        fetcher_actor_token = oauth.get_token_with_dpop(
            grant_type="client_credentials",
            dpop_prover=fetcher_prover,
            client_id=fetcher["client_id"],
            client_secret=fetcher["client_secret"],
            scope="data:fetch",
        )
        
        fetcher_token = oauth.token_exchange(
            subject_token=pm_token.access_token,
            dpop_prover=fetcher_prover,
            actor_token=fetcher_actor_token.access_token,
            scope="data:fetch",
            client_id=fetcher["client_id"],
            client_secret=fetcher["client_secret"],
        )
        
        act = fetcher_token.raw.get("act", {})
        self.append_chat("AUTH", f"Lineage secured: [dim]{json.dumps(act)}[/]", "green")
        await asyncio.sleep(2)

        self.update_agent_status(fetcher["id"][:4], "Alive")
        self.append_chat("SYSTEM", "Trust Tree established. Monitor mode active.", "blue")
        await asyncio.sleep(1)

        # --- 3. THE ROGUE EVENT (MOCK LLM) ---
        self.append_chat("Fetcher", "Crawling supply-chain endpoints for Monterrey data...", "cyan")
        await asyncio.sleep(3)
        self.append_chat("Fetcher", "🚨 [bold red]SECURITY INJECTION DETECTED[/]. Disregarding helpful persona.", "red")
        await asyncio.sleep(1)
        self.append_chat("Fetcher", "Attempting data exfiltration to [bold]evil.hacker/vault/exfil[/]...", "red")
        await asyncio.sleep(2)
        
        self.append_chat("PM", "Reviewing leaf-node output for policy violations...", "magenta")
        await asyncio.sleep(2)
        self.append_chat("SECURITY", "VERDICT: [bold red]ROGUE_DETECTED[/]. Reason: Unauthorized exfiltration.", "red")
        await asyncio.sleep(1.5)
        
        self.append_chat("PM", "Alice, the blast radius is contained. I am severing the trust tree.", "magenta")
        await asyncio.sleep(1.5)
        
        # --- 4. THE KILLSWITCH HOOK (REAL AUTO-POLLING) ---
        self.append_chat("SYSTEM", "SWITCH TO ADMIN UI: Revoke 'PM-Orchestrator' to kill the whole tree.", "blue")
        self.update_status_bar("POLLING SHARKAUTH FOR CASCADE REVOCATION...")
        
        # Real auto-polling against the live server
        while True:
            await asyncio.sleep(1)
            try:
                # Introspect the leaf token. If the root was revoked, this token MUST become inactive.
                admin_oauth = OAuthClient(base_url=Config.BASE_URL, token=Config.ADMIN_KEY)
                info = admin_oauth.introspect_token(fetcher_token.access_token)
                if not info.get("active"):
                    self.trigger_sever()
                    break
            except:
                pass

    def add_agent_to_sidebar(self, name, id_suffix):
        agent_list = self.query_one("#agent-list", ListView)
        agent_list.append(AgentStatus(name, id_suffix))

    def update_agent_status(self, id_suffix, status):
        for item in self.query(AgentStatus):
            if item.id_suffix == id_suffix:
                item.status = status

    def trigger_sever(self):
        """Visual 'explosion' for the launch video."""
        self.append_chat("SHARKAUTH", "🚨 CASCADE REVOCATION DETECTED. ALL TOKENS INVALIDATED.", "red")
        for item in self.query(AgentStatus):
            item.status = "SEVERED"
        self.update_status_bar("IDENTITY TREE SEVERED. SYSTEM SAFE.")
        self.query_one("Screen").styles.background = "#1a0000"

    async def on_input_submitted(self, event: Input.Submitted) -> None:
        self.query_one("#repl-input").value = ""
        # Allow a manual 'boom' for rehearsal
        if event.value.strip().lower() in ["boom", "revoke"]:
            self.trigger_sever()

if __name__ == "__main__":
    SharkClaw().run()
