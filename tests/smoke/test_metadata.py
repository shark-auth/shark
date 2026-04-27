import pytest
import requests

import os
BASE_URL = os.environ.get("BASE", "http://localhost:8080")

def test_jwks_parity():
    """Section 4: JWKS endpoint validation."""
    resp = requests.get(f"{BASE_URL}/.well-known/jwks.json")
    assert resp.status_code == 200
    
    data = resp.json()
    keys = data.get("keys", [])
    assert len(keys) >= 1, "No keys in JWKS"
    
    # RS256 key expected for session JWTs
    rs256_keys = [k for k in keys if k.get("alg") == "RS256"]
    assert len(rs256_keys) >= 1, "No RS256 keys in JWKS"
    assert rs256_keys[0].get("kty") == "RSA"
    assert rs256_keys[0].get("use") == "sig"
    
    # Cache control check
    cache_control = resp.headers.get("Cache-Control", "").lower()
    assert "max-age=" in cache_control

def test_as_metadata_rfc8414():
    """Section 26 & 28: Authorization Server Metadata."""
    endpoints = [
        "/.well-known/oauth-authorization-server",
        "/.well-known/openid-configuration" # Often alias
    ]
    
    for ep in endpoints:
        resp = requests.get(f"{BASE_URL}{ep}")
        if resp.status_code != 200 and ep == "/.well-known/openid-configuration":
            continue # OIDC might not be enabled in this mode
            
        assert resp.status_code == 200
        meta = resp.json()
        
        assert "issuer" in meta
        assert "authorization_endpoint" in meta
        assert "token_endpoint" in meta
        assert "registration_endpoint" in meta
        
        # PKCE
        assert "S256" in meta.get("code_challenge_methods_supported", [])
        
        # Grants
        grants = meta.get("grant_types_supported", [])
        assert "client_credentials" in grants
        # device_code was intentionally removed (commit f6462a6) — assert it is absent
        assert "urn:ietf:params:oauth:grant-type:device_code" not in grants, (
            "device_authorization_endpoint grant was intentionally disabled — must not appear in metadata"
        )
        assert "urn:ietf:params:oauth:grant-type:token-exchange" in grants
        assert "authorization_code" in grants
        assert "refresh_token" in grants
        
        # DPoP
        assert "ES256" in meta.get("dpop_signing_alg_values_supported", [])
