"""Shared test fixtures."""

from __future__ import annotations

import pytest

from shark_auth import tokens as tokens_mod


@pytest.fixture(autouse=True)
def _clear_jwks_cache():
    """Ensure every test starts with a clean JWKS cache."""
    tokens_mod.clear_jwks_cache()
    yield
    tokens_mod.clear_jwks_cache()
