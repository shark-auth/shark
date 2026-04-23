#!/usr/bin/env python3
"""Hello Agent — end-to-end demo for SharkAuth + shark-auth Python SDK.

Flow
----
1. Mint an access token via the OAuth 2.1 client_credentials grant.
2. Generate a DPoP proof JWT bound to a hypothetical resource call.
3. Decode + verify the access token locally with ``decode_agent_token``
   against ``/.well-known/jwks.json`` — the canonical SharkAuth promise:
   "OSS MCP-native agent auth — decode tokens in 3 lines."
4. Print structured step output + exit 0 on full success.

This demo deliberately skips the Token Vault call (which requires a
user-delegated token) — see docs/hello-agent.md for that follow-up step.
"""
from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from urllib.parse import urljoin

import requests

from shark_auth import DPoPProver, decode_agent_token


def _iso(ts: int) -> str:
    return datetime.fromtimestamp(ts, tz=timezone.utc).isoformat()


def step(n: int, total: int, msg: str) -> None:
    print(f"[{n}/{total}] {msg}", flush=True)


def mint_token(auth_url: str, client_id: str, client_secret: str, scope: str) -> dict:
    url = urljoin(auth_url + "/", "oauth/token")
    resp = requests.post(
        url,
        data={"grant_type": "client_credentials", "scope": scope},
        auth=(client_id, client_secret),
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        timeout=10,
    )
    resp.raise_for_status()
    return resp.json()


def main() -> int:
    p = argparse.ArgumentParser(description="SharkAuth hello-agent demo")
    p.add_argument("--auth", required=True, help="SharkAuth base URL, e.g. http://localhost:8080")
    p.add_argument("--client-id", required=True, help="Agent client_id (from DCR)")
    p.add_argument("--client-secret", required=True, help="Agent client_secret (from DCR)")
    p.add_argument("--scope", default="openid", help="Scope to request (default: openid)")
    p.add_argument(
        "--resource",
        default="https://api.example/data",
        help="Hypothetical resource URL the DPoP proof binds to",
    )
    args = p.parse_args()

    total = 4
    try:
        step(1, total, f"Minting access token via client_credentials @ {args.auth}/oauth/token ...")
        tok = mint_token(args.auth, args.client_id, args.client_secret, args.scope)
        access_token = tok["access_token"]
        print(
            f"       token_type={tok.get('token_type')} "
            f"expires_in={tok.get('expires_in')}s scope={tok.get('scope')}"
        )

        step(2, total, "Generating DPoP proof (ECDSA P-256) bound to resource call ...")
        prover = DPoPProver.generate()
        proof = prover.make_proof(
            htm="GET",
            htu=args.resource,
            access_token=access_token,
        )
        print(f"       jkt={prover.jkt}  proof_len={len(proof)}")

        step(
            3,
            total,
            "Decoding token locally via decode_agent_token + /.well-known/jwks.json ...",
        )
        jwks_url = urljoin(args.auth + "/", ".well-known/jwks.json")
        # Server uses the configured base URL as both issuer and (when no
        # explicit aud is requested) audience. For client_credentials the
        # subject is the client_id itself.
        claims = decode_agent_token(
            access_token,
            jwks_url,
            expected_issuer=args.auth,
            expected_audience=args.client_id,
        )
        print(
            f"       sub={claims.sub} "
            f"iss={claims.iss} "
            f"exp={_iso(claims.exp)} "
            f"scope={claims.scope}"
        )

        step(4, total, "All checks passed. Token minted, DPoP proof built, JWT decoded.")
        return 0
    except requests.HTTPError as exc:
        print(f"HTTP error: {exc} — body={exc.response.text[:400]}", file=sys.stderr)
        return 3
    except KeyError as exc:
        print(f"Missing field in response: {exc}", file=sys.stderr)
        return 4
    except Exception as exc:  # noqa: BLE001
        print(f"Unexpected error: {exc}", file=sys.stderr)
        return 5


if __name__ == "__main__":
    sys.exit(main())
