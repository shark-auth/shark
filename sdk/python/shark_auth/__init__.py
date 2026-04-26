"""shark-auth: Python SDK for SharkAuth agent-auth primitives.

Public API
----------
- :class:`Client`          — unified admin client (proxy rules, lifecycle, branding, paywall, users, agents)
- :class:`DPoPProver`      — RFC 9449 DPoP proof JWT emission
- :class:`DeviceFlow`      — RFC 8628 device authorization grant
- :class:`VaultClient`     — Shark Token Vault client
- :class:`OAuthClient`     — RFC 7009 revoke + RFC 7662 introspect
- :class:`MagicLinkClient` — send magic-link sign-in emails
- :class:`DPoPHTTPClient`  — DPoP-authenticated HTTP helpers (get/post/delete)
- :func:`decode_agent_token` — verify Shark-issued agent access tokens
"""

from .agents import AgentsClient
from .branding import BrandingClient
from .client import Client
from .device_flow import DeviceFlow, DeviceInit, TokenResponse
from .dpop import DPoPProver
from .errors import (
    DeviceFlowError,
    DPoPError,
    OAuthError,
    SharkAuthError,
    TokenError,
    VaultError,
)
from .http_client import DPoPHTTPClient
from .magic_link import MagicLinkClient
from .oauth import OAuthClient, Token
from .paywall import PaywallClient
from .proxy_lifecycle import ProxyLifecycleClient, ProxyStatus
from .proxy_rules import (
    CreateProxyRuleInput,
    ImportResult,
    ProxyRule,
    ProxyRulesClient,
    SharkAPIError,
    UpdateProxyRuleInput,
)
from .session import AgentSession
from .tokens import AgentTokenClaims, clear_jwks_cache, decode_agent_token, exchange_token
from .users import UsersClient
from .vault import VaultClient, VaultToken

__version__ = "0.1.0"

__all__ = [
    # v1.5 admin client + sub-clients
    "Client",
    "ProxyRulesClient",
    "ProxyLifecycleClient",
    "BrandingClient",
    "PaywallClient",
    "UsersClient",
    "AgentsClient",
    "OAuthClient",
    "MagicLinkClient",
    # v1.5 types
    "ProxyRule",
    "CreateProxyRuleInput",
    "UpdateProxyRuleInput",
    "ImportResult",
    "ProxyStatus",
    "SharkAPIError",
    # pre-existing
    "DPoPProver",
    "AgentSession",
    "DeviceFlow",
    "DeviceInit",
    "TokenResponse",
    "VaultClient",
    "VaultToken",
    "AgentTokenClaims",
    "decode_agent_token",
    "exchange_token",
    "clear_jwks_cache",
    "SharkAuthError",
    "DPoPError",
    "DeviceFlowError",
    "OAuthError",
    "VaultError",
    "TokenError",
    # W2 Method 3 — DPoP HTTP helpers
    "DPoPHTTPClient",
    # W2 DPoP token request
    "Token",
    "__version__",
]
