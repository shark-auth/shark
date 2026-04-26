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


class OAuthError(SharkAuthError):
    """Error returned by the OAuth token endpoint (4xx/5xx)."""

    def __init__(
        self,
        error: str,
        error_description: str | None = None,
        status_code: int | None = None,
    ) -> None:
        msg = error
        if error_description:
            msg = f"{error}: {error_description}"
        super().__init__(msg)
        self.error = error
        self.error_description = error_description
        self.status_code = status_code
