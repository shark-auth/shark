# DEMO 01 — Zero-Config MCP Server Onboarding
## "Claude Desktop Discovers, Self-Registers, and Calls a Protected MCP Resource in 8 Seconds"

**Format:** 8–10 min live, single presenter (SE or solo founder), meetup or conference stage  
**Audience:** MCP server builders, platform engineers, DevOps leads, developer-tools founders  
**Goal:** Prove SharkAuth is the only OSS auth server that ships full MCP-compatible discovery + DCR + DPoP-bound JWTs out of the box — zero config, single binary, 8 seconds cold to authenticated.

---

## 1. Persona & Pain — Priya / Octopus

**Priya** is the founder of **Octopus**, an MCP server marketplace. She hosts 200+ MCP servers built by independent developers — think "npm for AI tools." Each server does something different: code search, internal wiki, CRM read, database query, calendar booking.

**Current reality:**
- Every one of her 200 servers uses a static API key stored in `~/.cursor/config.json` or a Claude Desktop JSON file.
- Onboarding a new tool: user emails Priya's team → team generates a secret → user pastes it manually. Average time: **2 days**.
- Security team audit (Q1 this year): **47% of active API keys have never been rotated**. Three were copy-pasted into public GitHub repos.
- Support burden: 34% of Octopus support tickets are "my API key stopped working" — manual revoke-and-reissue.
- When a developer ships a new server version, every existing client still holds the old key. There is no programmatic way to force re-auth.
- No scoping: every key grants full access. A calendar-booking tool gets the same key as the "execute arbitrary SQL" tool.

**What Priya wants:** An agent walks up to any Octopus server cold — no pre-shared secret, no human in the loop — and is authenticated with a short-lived, audience-bound, cryptographically proven token in under 10 seconds. When she revokes a developer's server, every agent that was using it is locked out immediately.

**The number that lands:** "200 servers × average 50 users × 2-day onboarding = 20,000 person-days of support overhead this year. I need that to be zero."

---

## 2. What's Wrong With the Status Quo

### Claude Desktop / Cursor / Cline today

Claude Desktop (as of April 2026) supports MCP servers configured in `claude_desktop_config.json`. Authentication is either:
- **No auth** — stdio transport, process-local, no network boundary
- **Static API key** — `"headers": {"Authorization": "Bearer sk_live_XXXX"}` hardcoded in the config file
- **OAuth via browser** — some servers (Cloudflare, Stripe) send the user through a browser redirect; this is fine for human-in-the-loop but breaks headless agents and CI pipelines entirely

Cursor's MCP support mirrors this: per-server API keys in `.cursor/mcp.json`. Continue.dev and Cline have equivalent patterns — shared secret in a config file, no rotation, no scoping, no audience binding.

**The spec says something different.** The MCP Authorization spec (draft, 2026) is explicit:

> MCP servers MUST implement OAuth 2.0 Protected Resource Metadata (RFC 9728) to indicate the locations of authorization servers.

> MCP clients MUST be able to parse WWW-Authenticate headers and respond appropriately to HTTP 401 Unauthorized responses.

> MCP servers MUST validate that access tokens were issued specifically for them as the intended audience, according to RFC 8707.

No current open-source auth server ships all three of these out of the box. Cloudflare's Workers AI Auth is proprietary and priced per-seat. WorkOS MCP and Auth0's MCP-Auth integrations require managed cloud accounts and do not support DPoP. The result: every MCP server builder either skips auth entirely or rolls their own broken version.

**SharkAuth ships all of it. In one binary. Free.**

---

## 3. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│  MCP CLIENT (Claude Desktop / Cursor / CLI agent)                       │
│                                                                         │
│  • Holds: nothing (cold start, no pre-shared secret)                    │
│  • Generates: ECDSA P-256 keypair in memory (DPoP key holder)           │
│  • Speaks: HTTP/1.1 + MCP JSON-RPC over SSE or stdio                    │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
           Step 1: GET /mcp/server-42  (no token)
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  OCTOPUS MCP SERVER  (Python, ~50 lines, uses shark-auth middleware)     │
│                                                                         │
│  • Returns: 401 Unauthorized                                            │
│  • Header:  WWW-Authenticate: Bearer                                    │
│               resource_metadata="http://octopus.dev/.well-known/        │
│                 oauth-protected-resource"          ← RFC 9728           │
│  • Hosts:   GET /.well-known/oauth-protected-resource                   │
│               → { "resource": "mcp://octopus.dev/server-42",            │
│                   "authorization_servers": ["http://shark:8080"] }      │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
        Steps 2–3: fetch resource metadata → discover AS URL
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  SHARK AS  (single Go binary, :8080)                                    │
│                                                                         │
│  Step 3: GET /.well-known/oauth-authorization-server         RFC 8414   │
│    ← { "issuer": "http://shark:8080",                                   │
│         "registration_endpoint": "http://shark:8080/oauth/register",    │
│         "token_endpoint":        "http://shark:8080/oauth/token",       │
│         "dpop_signing_alg_values_supported": ["ES256"],                 │
│         "client_id_metadata_document_supported": true }                 │
│                                                                         │
│  Step 4: POST /oauth/register                                RFC 7591   │
│    → { "client_id_metadata_document_uri":                               │
│          "https://octopus.dev/.well-known/client-metadata.json" }       │
│    ← { "client_id": "shark_dcr_abc123",                                 │
│         "registration_access_token": "rat_xyz..." }                     │
│    (Shark fetches + validates CIMD from the HTTPS URL — CIMD)           │
│                                                                         │
│  Step 5: Client generates P-256 keypair → builds DPoP proof JWT         │
│    DPoP header: { "alg":"ES256", "typ":"dpop+jwt",                      │
│                   "jwk": { public key JWK } }                           │
│    DPoP payload:{ "htm":"POST", "htu":"http://shark:8080/oauth/token",  │
│                   "iat":<now>, "jti":"<uuid>" }                         │
│                                                                         │
│  Step 6: POST /oauth/token                          RFC 9449 + RFC 8707 │
│    Headers: DPoP: <proof JWT>                                           │
│    Body:    grant_type=client_credentials                                │
│             &client_id=shark_dcr_abc123                                  │
│             &resource=mcp://octopus.dev/server-42                       │
│    ← ES256 JWT access token:                                            │
│        { "iss":"http://shark:8080",                                     │
│          "aud":"mcp://octopus.dev/server-42",     ← audience-bound      │
│          "cnf":{"jkt":"<P256-thumbprint>"},       ← DPoP-bound          │
│          "exp":<now+3600>, "scope":"mcp:read" }                         │
│                                                                         │
│  internal/oauth/metadata.go   — serves RFC 8414 well-known             │
│  internal/oauth/dcr.go        — RFC 7591 register + CIMD fetch         │
│  internal/oauth/dpop.go       — RFC 9449 proof validation + JTI cache  │
│  internal/oauth/audience.go   — RFC 8707 resource→aud binding           │
│  internal/oauth/handlers.go   — issues JWT with cnf.jkt                 │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
     Step 7: GET /mcp/server-42
       Authorization: DPoP <access_token>
       DPoP: <new proof with ath=hash(access_token)>
                           │
                           ▼
             MCP server verifies locally (3 lines, SDK):
             decode_agent_token(token, jwks_url, issuer, audience)
             → checks cnf.jkt == DPoP proof thumbprint
             → 200 OK  {"result": "Hello from server-42"}
```

---

## 4. Shark Feature Map Table

| Demo Step | What Happens | RFC | Shark File | Key Symbol |
|-----------|-------------|-----|-----------|------------|
| 1 | MCP server returns 401 + `WWW-Authenticate: Bearer resource_metadata=...` | RFC 9728 §5.1 | `demos/mcp_cold_start/octopus-mcp-server/server.py` (stub, see Gap section) | `resource_metadata` header |
| 2 | Client fetches `/.well-known/oauth-protected-resource` | RFC 9728 §3 | MCP server hosts this JSON doc | `authorization_servers` array |
| 3 | Client fetches `/.well-known/oauth-authorization-server` | RFC 8414 | `internal/oauth/metadata.go:36 MetadataHandler()` | `registration_endpoint`, `dpop_signing_alg_values_supported` |
| 4 | Client POSTs to `/oauth/register` with CIMD URL | RFC 7591 + CIMD draft | `internal/oauth/dcr.go` | `client_id_metadata_document_uri`, CIMD fetch+validate |
| 5 | Client generates P-256 keypair, builds DPoP proof JWT | RFC 9449 §4 | `sdk/python/shark_auth/dpop.py:33 DPoPProver` | `jkt`, `jti`, `htm`, `htu` |
| 6 | Client POSTs `/oauth/token` with DPoP header + `resource=` | RFC 9449 + RFC 8707 | `internal/oauth/handlers.go:47 HandleToken()`, `internal/oauth/dpop.go:94 ValidateDPoPProof()`, `internal/oauth/audience.go:31 ValidateAudience()` | `cnf.jkt`, `aud` bound to resource |
| 7 | MCP server verifies token locally, no introspection round-trip | RFC 7519 + RFC 9449 | `sdk/python/shark_auth/tokens.py decode_agent_token()` | 3-line local verify |
| 8 | Revoke DCR client → agent gets 401 → DCR restarts | RFC 7592 | `GET/PUT/DELETE /oauth/register/{client_id}` | registration lifecycle |
| BONUS | Step-up: different server returns `403 + insufficient_scope` | OAuth 2.1 §5.3 | `internal/oauth/handlers.go` | `WWW-Authenticate: Bearer error="insufficient_scope"` |
| BONUS | Admin dashboard shows agent + audit log | — | `admin/` | Agents tab, audit trail |

---

## 5. Live Demo Script

**Setup (before stage):** Two terminal panes visible on screen. Shark is NOT running yet. No env vars set.

### T+0:00 — The Pain (30 seconds, no keyboard)

> "Priya has 200 MCP servers. Every one of them requires users to get an API key from a form, paste it into a JSON config file, and hope they never rotate it. Her security team flagged 47% of keys as never rotated. I'm going to show you what happens when you give her SharkAuth."

### T+0:30 — Cold Start Shark (15 seconds)

**Terminal 1:**
```bash
./bin/shark serve --dev
```

Expected output (live, timestamps):
```
2026-04-24T10:00:00Z INF SharkAuth starting  addr=:8080 dev_mode=true
2026-04-24T10:00:00Z INF admin dashboard     url=http://localhost:8080/admin
2026-04-24T10:00:00Z INF health check        url=http://localhost:8080/healthz
2026-04-24T10:00:00Z INF default app created client_id=shark_app_abc123
```

> "Single binary, zero config, dev mode. No database to provision. No cloud account. Already listening."

### T+0:45 — Confirm AS Metadata Exists (10 seconds)

**Terminal 2:**
```bash
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq '{issuer,registration_endpoint,dpop_signing_alg_values_supported}'
```

Output:
```json
{
  "issuer": "http://localhost:8080",
  "registration_endpoint": "http://localhost:8080/oauth/register",
  "dpop_signing_alg_values_supported": ["ES256"]
}
```

> "Step 3 of the MCP spec is already done. Any MCP client that knows the AS URL can discover everything it needs."

### T+0:55 — Start the MCP Server (10 seconds)

**Terminal 2:**
```bash
cd demos/mcp_cold_start/octopus-mcp-server
python server.py --shark http://localhost:8080 --resource mcp://octopus.dev/server-42
```

Output:
```
[octopus-mcp] server-42 listening on :9000
[octopus-mcp] Protected Resource Metadata at http://localhost:9000/.well-known/oauth-protected-resource
[octopus-mcp] AS: http://localhost:8080
```

### T+1:05 — The Cold Start (THE MONEY SHOT — 8 seconds)

**Terminal 2** (new pane or same):
```bash
python demos/mcp_cold_start/cli-client/run.py \
  --mcp http://localhost:9000 \
  --resource mcp://octopus.dev/server-42
```

**Watch the timestamps:**
```
[T+0.00s] → GET http://localhost:9000/mcp/call   (no token)
[T+0.01s] ← 401 Unauthorized
           WWW-Authenticate: Bearer resource_metadata="http://localhost:9000/.well-known/oauth-protected-resource"

[T+0.05s] → GET http://localhost:9000/.well-known/oauth-protected-resource
[T+0.06s] ← {"resource":"mcp://octopus.dev/server-42",
              "authorization_servers":["http://localhost:8080"]}

[T+0.08s] → GET http://localhost:8080/.well-known/oauth-authorization-server
[T+0.09s] ← {"issuer":"http://localhost:8080",
              "registration_endpoint":"http://localhost:8080/oauth/register",
              "dpop_signing_alg_values_supported":["ES256"],
              "client_id_metadata_document_supported":true}

[T+0.12s] → POST http://localhost:8080/oauth/register
            {"client_id_metadata_document_uri":"https://octopus.dev/.well-known/client-metadata.json",
             "grant_types":["client_credentials"],
             "scope":"mcp:read"}
[T+0.35s] ← {"client_id":"shark_dcr_f8a2c1",
              "registration_access_token":"rat_9xKpL...",
              "registration_client_uri":"http://localhost:8080/oauth/register/shark_dcr_f8a2c1"}
            [Shark fetched + validated CIMD from octopus.dev in 230ms]

[T+0.37s] Generating P-256 keypair...
[T+0.37s] DPoP thumbprint: abc123def456...
[T+0.38s] → POST http://localhost:8080/oauth/token
            DPoP: eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3AranNvbiIsImp3ayI6e...
            grant_type=client_credentials&client_id=shark_dcr_f8a2c1
            &resource=mcp%3A%2F%2Foctopus.dev%2Fserver-42
[T+0.45s] ← ES256 JWT issued
            cnf.jkt: abc123def456...   ← DPoP key thumbprint bound
            aud:     mcp://octopus.dev/server-42   ← audience bound
            exp:     T+3600s

[T+0.46s] Verifying token locally (no introspection round-trip)...
            decode_agent_token(token, "http://localhost:8080/.well-known/jwks.json",
                               "http://localhost:8080",
                               "mcp://octopus.dev/server-42")
[T+0.48s] ✓ Token valid. iss=http://localhost:8080 aud=mcp://octopus.dev/server-42

[T+0.50s] → GET http://localhost:9000/mcp/call
            Authorization: DPoP eyJhbGci...
            DPoP: eyJhbGciOiJFUzI1NiIsInR5cCI6ImRwb3AranNvbiIsImp3ayI6e... (with ath)
[T+0.51s] ← 200 OK
            {"result":"Hello from server-42","server":"octopus-mcp","tool":"echo"}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  COLD START → AUTHENTICATED CALL: 0.51s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

> "Half a second. Zero pre-shared secrets. The agent found the AS, registered itself, generated a keypair, got a DPoP-bound token, and called the protected resource. Nobody emailed Priya."

### T+1:35 — Audience Rejection (30 seconds)

> "But here's the security story. That token is bound to server-42. What happens if it tries to hit server-99?"

```bash
# Try to use server-42's token against server-99
curl -s http://localhost:9001/mcp/call \
  -H "Authorization: DPoP $TOKEN" \
  -H "DPoP: $PROOF"
```

Output:
```json
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer error="invalid_token",
                  error_description="token audience mcp://octopus.dev/server-42 does not match resource mcp://octopus.dev/server-99"
```

> "Token stolen from server-42? It's worthless on server-99. Audience binding via RFC 8707. The token says who it's for. The server checks."

### T+2:05 — Revoke the DCR Client (45 seconds)

> "Now Priya finds out one of her developers shipped a server with a vulnerability. She needs to revoke all agents for that server immediately."

```bash
# Revoke the DCR client via registration management (RFC 7592)
curl -s -X DELETE http://localhost:8080/oauth/register/shark_dcr_f8a2c1 \
  -H "Authorization: Bearer rat_9xKpL..."
```

Output:
```
HTTP/1.1 204 No Content
```

```bash
# Next call from the agent
curl -s http://localhost:9000/mcp/call \
  -H "Authorization: DPoP $TOKEN" \
  -H "DPoP: $PROOF"
```

Output:
```
HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer resource_metadata="http://localhost:9000/.well-known/oauth-protected-resource"
```

> "The agent gets a 401. It re-discovers, re-registers, re-authenticates. The new registration_access_token is different. The old one is dead. Priya gets full audit log of every step in the admin dashboard."

### T+2:50 — Open Admin (30 seconds)

Open browser: `http://localhost:8080/admin` → Agents tab

Show:
- `shark_dcr_f8a2c1` — REVOKED — timestamp
- Audit log: REGISTER / TOKEN_ISSUED / TOKEN_USED / REVOKED — every step with timestamp, IP, client_id

> "Full audit trail. No lambda. No cloud. Runs on a $5 VPS."

### T+3:20 — Wrap (40 seconds, no keyboard)

> "What just happened: a completely cold MCP client — no pre-shared secret, no human in the loop — discovered the auth server, self-registered using a Client ID Metadata Document hosted at Octopus.dev, generated a DPoP keypair, got an audience-bound ES256 JWT, and called a protected MCP tool. In half a second. And when the developer's server was compromised, Priya revoked the client and every agent was locked out — immediately, not after key rotation emails."

---

## 6. Implementation Plan

### Files to Create

#### `demos/mcp_cold_start/octopus-mcp-server/server.py` (~55 lines)

A minimal Python HTTP server (stdlib `http.server`, no framework) that:
- `GET /mcp/call` — checks `Authorization: DPoP` header; if missing → `401 + WWW-Authenticate: Bearer resource_metadata=<own URL>/.well-known/oauth-protected-resource`
- If token present → calls `decode_agent_token(token, jwks_url, issuer, expected_audience)` from `shark_auth` SDK; on success returns `{"result":"Hello from server-42"}`; on failure returns `401`
- `GET /.well-known/oauth-protected-resource` — returns RFC 9728 JSON doc: `{"resource":"mcp://octopus.dev/server-42","authorization_servers":["<shark_url>"]}`
- CLI args: `--shark <url>` (AS base), `--resource <mcp://...>` (own resource identifier), `--port` (default 9000)
- Also starts a second instance on `:9001` for `server-99` (audience rejection demo) — same code, different `--resource`

#### `demos/mcp_cold_start/cli-client/run.py` (~80 lines)

A Python script that orchestrates the full 7-step flow with **millisecond-precision timestamps** on every line:
1. Unauthenticated request → parse 401 + `WWW-Authenticate`
2. Fetch `oauth-protected-resource` JSON
3. Fetch `oauth-authorization-server` JSON
4. POST `/oauth/register` with CIMD URI
5. Generate P-256 keypair via `DPoPProver.generate()` from `shark_auth` SDK
6. POST `/oauth/token` with DPoP header + `resource=` param
7. Call protected MCP resource with `Authorization: DPoP` + new DPoP proof (with `ath`)
8. Print final summary: elapsed time, client_id, token `exp`, `aud`, `cnf.jkt`

#### `demos/mcp_cold_start/seed.sh` (~20 lines)

Shell script that:
1. Checks `./bin/shark` exists (builds if not)
2. Starts `shark serve --dev` in background, captures admin API key
3. Starts both MCP server instances (server-42 on :9000, server-99 on :9001)
4. Prints "Ready — run: python demos/mcp_cold_start/cli-client/run.py ..."
5. `trap` to kill all background processes on Ctrl-C

#### `demos/mcp_cold_start/fixtures/client-metadata.json` (CIMD fixture)

A static JSON file served at a known HTTPS URL (or via `--cimd-url file://...` for local dev) that represents the Octopus client's metadata document per the CIMD draft spec:
```json
{
  "client_name": "Octopus MCP Agent",
  "grant_types": ["client_credentials"],
  "token_endpoint_auth_method": "none",
  "scope": "mcp:read mcp:write",
  "contacts": ["security@octopus.dev"],
  "client_uri": "https://octopus.dev"
}
```

For the demo, host this on a public URL (GitHub raw, Cloudflare Pages) so Shark's CIMD fetch is real — not mocked. The demo README explains how to substitute your own URL.

---

## 7. Honest Gaps

### RFC 9728 — Protected Resource Metadata: Not Yet on the AS Side

The AGENT_AUTH.md checklist is clear:

```
[ ] Protected Resource Metadata (RFC 9728) — only if Shark also hosts MCP servers
```

**What this means for the demo:** RFC 9728's `/.well-known/oauth-protected-resource` endpoint lives on the **MCP server** (the resource server), not on the AS. Shark is the AS. So Shark does not need to implement RFC 9728 to make this demo work — the Octopus MCP server (`server.py`) implements the RFC 9728 endpoint and the `WWW-Authenticate` header. This is architecturally correct: RFC 9728 is a resource server responsibility.

**What we must be honest about:** The MCP spec says "MCP servers MUST implement RFC 9728." The MCP server in this demo does implement it — but it's a ~15-line stub we write for the demo. Real MCP server builders need SDK middleware that makes this trivial to add. The `shark-auth` Python SDK should ship a `ResourceServerMiddleware` class (future work, not blocking this demo).

**The one-paragraph plan:** For demo validity, `server.py` implements the two RFC 9728 requirements in ~15 lines of stdlib Python: (1) `GET /.well-known/oauth-protected-resource` returns the correct JSON document, and (2) unauthenticated requests get `401 + WWW-Authenticate: Bearer resource_metadata=<url>`. We call this out explicitly during the demo: "This is the part Priya's developers have to add to their servers — 15 lines. In the next sprint we'll ship a Python/TS middleware decorator that makes it a one-liner." That's the right honest framing: the AS is complete, the resource server stub is simple, the SDK middleware is roadmap.

---

## 8. The Wow Moment

The wow moment is the **timestamped terminal output scrolling in real time** — every HTTP exchange visible, every RFC cited inline, the final elapsed time printed in bold.

What the audience sees and feels:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [T+0.00s]  → cold request (no token)
  [T+0.01s]  ← 401  resource_metadata discovered          RFC 9728
  [T+0.06s]  ← AS URL extracted from resource metadata    RFC 9728
  [T+0.09s]  ← AS metadata fetched (DCR + DPoP advertised) RFC 8414
  [T+0.35s]  ← client_id issued (CIMD validated)          RFC 7591
  [T+0.37s]     P-256 keypair generated                   RFC 9449
  [T+0.45s]  ← ES256 JWT  aud=mcp://octopus.dev/server-42  RFC 8707
                           cnf.jkt=abc123...               RFC 9449
  [T+0.51s]  ← 200 OK  {"result":"Hello from server-42"}

  ✓ COLD → AUTHENTICATED: 0.51 seconds
  ✓ No pre-shared secrets
  ✓ No human in the loop
  ✓ Token audience-bound, DPoP-bound, non-reusable
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

The RFC labels in the right column are intentional — they signal to the technical audience that every step is spec-correct, not ad-hoc. The presenter says: "Half a second. And every one of those lines is a published RFC, not a proprietary SDK you have to pay for."

The second wow: the audience rejection test. Same token, different server. `401 invalid_token: audience mismatch`. The presenter says: "Stolen token is worthless. That's RFC 8707. It's in the binary."

---

## 9. Sellable Angle

**1-line pitch:**  
*"SharkAuth is the only self-hosted auth server that gives your MCP server spec-compliant OAuth — cold-start discovery, self-registration, DPoP-bound tokens — in a single binary, free forever."*

**3 customer types:**

| Customer | Pain | Shark Solves |
|----------|------|-------------|
| **MCP marketplace operators** (Priya / Octopus) | 200 servers × static API keys = security audit nightmare + support overhead | DCR + CIMD = agents self-register, no human in the loop; RFC 7592 revocation = instant kill switch |
| **Enterprise AI platform teams** | Compliance requires token scoping, audit logs, audience binding; cloud auth vendors charge per-MAU at scale | Self-hosted, audience-bound JWTs, full audit trail, DPoP prevents token theft; no per-seat pricing |
| **MCP server developers** | Every server has to build its own auth middleware; no reference implementation exists | `shark-auth` SDK: `decode_agent_token()` in 3 lines; `ResourceServerMiddleware` on roadmap; follows the spec so Claude Desktop / Cursor / Cline just work |

---

## 10. UAT Checklist

Pre-demo verification. Run `demos/mcp_cold_start/seed.sh` then verify each item.

- [ ] **1. Cold start** — `curl -s http://localhost:9000/mcp/call` returns `401` with `WWW-Authenticate: Bearer resource_metadata=...`
- [ ] **2. Resource metadata** — `curl -s http://localhost:9000/.well-known/oauth-protected-resource | jq .authorization_servers` returns `["http://localhost:8080"]`
- [ ] **3. AS metadata** — `curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq .dpop_signing_alg_values_supported` returns `["ES256"]`
- [ ] **4. AS metadata has registration_endpoint** — same response includes `"registration_endpoint":"http://localhost:8080/oauth/register"`
- [ ] **5. DCR succeeds** — POST to `/oauth/register` with CIMD URI returns `client_id` + `registration_access_token` within 500ms (CIMD fetch included)
- [ ] **6. DCR CIMD fetch** — Shark logs confirm it fetched and validated the `client_id_metadata_document_uri`
- [ ] **7. DPoP token issued** — POST to `/oauth/token` with `DPoP:` header and `resource=mcp://octopus.dev/server-42` returns JWT containing `cnf.jkt` and `aud=mcp://octopus.dev/server-42`
- [ ] **8. Token audience matches** — `decode_agent_token(token, jwks_url, issuer, "mcp://octopus.dev/server-42")` returns without error
- [ ] **9. Authenticated call succeeds** — `GET /mcp/call` with `Authorization: DPoP <token>` + `DPoP: <proof-with-ath>` returns `200 {"result":"Hello from server-42"}`
- [ ] **10. Audience rejection** — Same token presented to server-99 (`:9001`) returns `401` with `error=invalid_token` mentioning audience mismatch
- [ ] **11. DPoP replay rejection** — Reusing the same `DPoP:` proof (same `jti`) on a second request to Shark `/oauth/token` returns `400 use_dpop_nonce` or equivalent replay error
- [ ] **12. DCR revocation** — `DELETE /oauth/register/shark_dcr_xxx` with `registration_access_token` returns `204`; subsequent token use returns `401`
- [ ] **13. Post-revoke re-register** — After revocation, the cold-start flow re-runs successfully and issues a new `client_id`
- [ ] **14. Step-up 403** — Requesting a scope-restricted resource without the required scope returns `403 + WWW-Authenticate: Bearer error="insufficient_scope", scope="mcp:admin"`
- [ ] **15. Admin dashboard shows agent** — `http://localhost:8080/admin` Agents tab lists `shark_dcr_xxx` with status ACTIVE → REVOKED lifecycle visible
- [ ] **16. Audit log completeness** — Audit log shows: REGISTER, TOKEN_ISSUED, TOKEN_USED, REVOKED — all with timestamps, client_id, IP

---

## Appendix: Key File References

| File | Purpose | Key Lines |
|------|---------|-----------|
| `internal/oauth/metadata.go` | RFC 8414 AS metadata handler | `L36 MetadataHandler()`, `L38–44` endpoint URLs including `registration_endpoint` + `dpop_signing_alg_values_supported` |
| `internal/oauth/dcr.go` | RFC 7591 DCR + CIMD fetch/validate | DCR POST handler; CIMD fetch from `client_id_metadata_document_uri` |
| `internal/oauth/dpop.go` | RFC 9449 DPoP proof validation | `L54 NewDPoPJTICache()`, `L60 MarkSeen()` (replay), `L94 ValidateDPoPProof()` returns `jkt` |
| `internal/oauth/audience.go` | RFC 8707 resource → aud binding | `L12 contextWithResource()`, `L31 ValidateAudience()` |
| `internal/oauth/handlers.go` | `/oauth/token` issues JWT w/ cnf.jkt | `L47 HandleToken()`, `L64–79` DPoP intercept + jkt extraction, `L84–88` resource param, `L124–145` cnf.jkt injection into JWT |
| `internal/oauth/introspect.go` | RFC 7662 introspection (fallback only) | Not called in demo — local verify wins |
| `sdk/python/shark_auth/dpop.py` | DPoPProver class | `L33 class DPoPProver`, `L72–73 generate()`, `L92–94 jkt property` |
| `sdk/python/shark_auth/tokens.py` | `decode_agent_token()` | 3-line local verify, `cnf.jkt` extraction |
| `cmd/shark/cmd/agent_register.go` | `shark agent register` CLI | `L20 agentRegisterCmd`, `L79–83` flags: `--app`, `--name`, `--description` |
| `docs/hello-agent.md` | 15-min walkthrough (existing) | Full DCR → DPoP flow in curl |
| `examples/hello_agent.py` | Python demo script (existing) | mint_token + DPoP proof + decode_agent_token |
