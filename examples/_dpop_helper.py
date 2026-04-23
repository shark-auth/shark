#!/usr/bin/env python3
"""DPoP proof helper for smoke tests and manual testing (RFC 9449).

CLI usage
---------
  python _dpop_helper.py <method> <url> [options]

Options
  --nonce=<str>           DPoP nonce advertised by the server
  --jti=<str>             Override jti (default: random)
  --iat-offset=<seconds>  Offset added to current Unix time for iat
                          (e.g. --iat-offset=-3600 for 1 hr in the past)
  --access-token=<str>    Bind proof to an access token via ath claim
  --key-file=<path>       PEM key file; created on first run if absent
  --print-jkt             Print the JWK thumbprint of the key to stdout
                          instead of the proof JWT (used by smoke tests)

Outputs a single DPoP proof JWT (or JKT) to stdout.
Key is persisted to key-file so subsequent calls reuse the same keypair,
enabling replay and key-mismatch test scenarios.
"""
from __future__ import annotations

import argparse
import os
import sys
import time

# Allow running from the repo root or the examples/ directory.
_repo_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if _repo_root not in sys.path:
    sys.path.insert(0, _repo_root)

try:
    from sdk.python.shark_auth.dpop import DPoPProver
except ModuleNotFoundError:
    # Fallback: assume the sdk package is installed in the active environment.
    from shark_auth.dpop import DPoPProver  # type: ignore[no-redef]


def _load_or_create_prover(key_file: str | None) -> DPoPProver:
    """Return a DPoPProver, loading from key_file or creating a fresh key."""
    if key_file and os.path.exists(key_file):
        pem = open(key_file, "rb").read()
        return DPoPProver.from_pem(pem)

    prover = DPoPProver.generate()

    if key_file:
        pem = prover.private_key_pem()
        # Write atomically via temp file to avoid partial writes.
        tmp = key_file + ".tmp"
        with open(tmp, "wb") as fh:
            fh.write(pem)
        os.replace(tmp, key_file)
        os.chmod(key_file, 0o600)

    return prover


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(
        prog="_dpop_helper",
        description="Emit a DPoP proof JWT to stdout.",
    )
    p.add_argument("method", help="HTTP method (e.g. POST, GET)")
    p.add_argument("url", help="Target URL")
    p.add_argument("--nonce", default=None, help="DPoP nonce")
    p.add_argument("--jti", default=None, help="Override jti")
    p.add_argument(
        "--iat-offset",
        type=int,
        default=0,
        metavar="SECONDS",
        help="Offset added to current time for iat claim",
    )
    p.add_argument("--access-token", default=None, help="Access token for ath claim")
    p.add_argument(
        "--key-file",
        default=None,
        metavar="PATH",
        help="PEM key file (created if absent)",
    )
    p.add_argument(
        "--print-jkt",
        action="store_true",
        help="Print JWK thumbprint instead of proof JWT",
    )
    return p.parse_args()


def main() -> None:
    args = _parse_args()
    prover = _load_or_create_prover(args.key_file)

    if args.print_jkt:
        print(prover.jkt)
        return

    iat = int(time.time()) + args.iat_offset if args.iat_offset != 0 else None

    proof = prover.make_proof(
        htm=args.method,
        htu=args.url,
        nonce=args.nonce,
        access_token=args.access_token,
        iat=iat,
        jti=args.jti,
    )
    print(proof)


if __name__ == "__main__":
    main()
