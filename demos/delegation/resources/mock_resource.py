"""Stub resource server.

Decodes inbound Shark JWTs, renders the act chain, enforces audience binding,
and returns a stub response. Run multiple instances on different ports for
each audience (kb-api, gmail-vault, smtp-relay, salesforce-api, gcal-api).

Usage:
  python resources/mock_resource.py --port 9101 --audience kb-api --required-scope kb:read
"""
from __future__ import annotations

import argparse
import os
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

try:
    from dotenv import load_dotenv  # type: ignore
    load_dotenv(ROOT / ".env")
except Exception:
    pass

from lib import decode, render_chain  # noqa: E402


def make_handler(audience: str, required_scope: str):
    auth_url = os.environ["SHARK_AUTH_URL"]

    class H(BaseHTTPRequestHandler):
        def log_message(self, *_a):
            return

        def do_GET(self):  # noqa: N802
            authz = self.headers.get("Authorization", "")
            if not authz.startswith("Bearer "):
                return self._reject(401, "missing bearer token")
            token = authz[len("Bearer "):]
            try:
                claims = decode(token, auth_url, audience)
            except Exception as exc:
                return self._reject(401, f"token validation failed: {exc}")

            scopes = (claims.scope or "").split()
            if required_scope not in scopes:
                return self._reject(
                    403,
                    f"insufficient_scope (need {required_scope}, got {scopes})",
                )

            print(f"\n[{audience}] {self.path}")
            print(f"  sub:       {claims.sub}")
            print(f"  scope:     {claims.scope}")
            print(f"  cnf.jkt:   {claims.jkt}")
            print(f"  act chain:")
            for line in render_chain(claims.act).splitlines():
                print(f"    {line}")

            body = b'{"ok":true,"resource":"' + audience.encode() + b'"}\n'
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def _reject(self, status: int, reason: str):
            print(f"[{audience}] REJECT {status}: {reason}")
            body = f'{{"error":"{reason}"}}\n'.encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header(
                "WWW-Authenticate",
                f'Bearer error="invalid_token", error_description="{reason}"',
            )
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return H


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("--port", type=int, required=True)
    p.add_argument("--audience", required=True)
    p.add_argument("--required-scope", required=True)
    args = p.parse_args()

    handler = make_handler(args.audience, args.required_scope)
    srv = HTTPServer(("127.0.0.1", args.port), handler)
    print(f"mock-resource listening on http://127.0.0.1:{args.port}  aud={args.audience}  scope={args.required_scope}")
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        return 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
