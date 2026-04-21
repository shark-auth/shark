"""Exception hierarchy for shark_auth."""

from __future__ import annotations


class SharkAuthError(Exception):
    """Base class for all shark_auth errors."""


class DPoPError(SharkAuthError):
    """Error constructing or signing a DPoP proof."""


class DeviceFlowError(SharkAuthError):
    """Error during RFC 8628 device authorization flow."""


class VaultError(SharkAuthError):
    """Error interacting with the Shark Token Vault."""

    def __init__(self, message: str, status_code: int | None = None) -> None:
        super().__init__(message)
        self.status_code = status_code


class TokenError(SharkAuthError):
    """Error decoding or verifying a Shark-issued agent token."""
