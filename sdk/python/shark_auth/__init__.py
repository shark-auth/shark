"""shark-auth: Python SDK for SharkAuth agent-auth primitives.

Public API
----------
- :class:`DPoPProver` — RFC 9449 DPoP proof JWT emission
- :class:`DeviceFlow` — RFC 8628 device authorization grant
- :class:`VaultClient` — Shark Token Vault client
- :func:`decode_agent_token` — verify Shark-issued agent access tokens
"""

from .device_flow import DeviceFlow, DeviceInit, TokenResponse
from .dpop import DPoPProver
from .errors import (
    DeviceFlowError,
    DPoPError,
    SharkAuthError,
    TokenError,
    VaultError,
)
from .session import AgentSession
from .tokens import AgentTokenClaims, clear_jwks_cache, decode_agent_token, exchange_token
from .vault import VaultClient, VaultToken

__version__ = "0.1.0"

__all__ = [
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
    "VaultError",
    "TokenError",
    "__version__",
]
