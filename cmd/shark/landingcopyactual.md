v0.9.0 · Now Shipping · Open Source · Self-hosted
Auth for the Agent Era.
Your agents already work. With Shark, they do it securely.
Real delegation. Real DPoP. Real audit. One ~40 MB Go binary with embedded SQLite.
Zero dependencies. Runs anywhere — even on a Raspberry Pi.

Get the Binary
curl -fsSL get.sharkauth.dev | sh
Read the Docs
Join Cloud Waitlist
OAuth 2.1
OpenID Connect
RFC 8693 Token Exchange
RFC 9449 DPoP
MIT
Scroll
shark://admin / delegation_canvas
grant_id
g_8FaQ•••
act_chain depth 2
Delegation Canvas
jane@acme.dev → research-orchestrator → web-fetcher
Reset
Revoke B
Cascade revoke
may_act → research-orchestrator
scope: read:gmail read:calendar · exp 1h
token_exchange (RFC 8693)
scope: web:fetch · exp 5m · DPoP-bound
Jane Park
user · acme.dev
kid: rsa-2048-aD3
VERIFIED
research-orchestrator
agent_a · openai-gpt5
kid: ec-p256-7Hx
ACTIVE
web-fetcher
agent_b · anthropic-haiku
kid: ec-p256-2Mn
ACTIVE
act_chain: [user:jane@acme.dev → agent_a:research-orchestrator → agent_b:web-fetcher]
16:42:11.084 POST /token grant_type=urn:ietf:params:oauth:grant-type:token-exchange · subject=jane@acme.dev · actor=research-orchestrator
200 OK
Honest comparison
Built for the world that
actually exists in 2026.
Identity vendors were built for users clicking buttons. SharkAuth was built for agents calling agents calling APIs — with the same RFC-grade rigor.

Feature
SharkAuth LogoSharkAuth
Auth0 Clerk Keycloak
Agent as first-class identity
RFC 8693 Token Exchange Partial Partial
Full act / actor chain (depth ≥ 4)
may_act_grants & granular policy
RFC 9449 DPoP key binding Partial
Cascade revocation
Audit indexed by grant_id Partial Partial Partial
Single ~40 MB binary
Self-hostable & open-source
Runs on a Raspberry Pi
Comparison reflects publicly documented features as of April 2026.
60-second quickstart
From zero to token exchange in three commands.
No Postgres. No Helm chart. No identity vendor SDK with 18 transitive dependencies. One binary, one SQLite file, zero excuses.
~/acme · zsh
$
curl -fsSL get.sharkauth.dev | sh
·
✓ downloaded shark-0.9.0-linux-amd64 (39.8 MB)
$
shark init --issuer https://auth.acme.dev
·
✓ wrote shark.toml ✓ generated EC P-256 signing key
$
shark serve
·
listening on :4444 · admin on :4445 · sqlite at ./shark.db
·
discovery → /.well-known/oauth-authorization-server
$
agent.ts
@sharkauth/sdk · v0.9.0
import { Shark } from "@sharkauth/sdk";

const shark = new Shark({ issuer: "https://auth.acme.dev" });

// Exchange the user's id_token for an agent token bound to a DPoP key
const agentToken = await shark.tokenExchange({
subject_token: userIdToken,
actor_token: agentJwk, // RFC 9449 DPoP-bound
scope: "read:gmail read:calendar",
may_act: { client_id: "research-orchestrator" },
});
STEP 01
Drop the binary
Single Go binary, ~40 MB. macOS, Linux, ARM. No runtime, no daemon.
STEP 02
Configure once
shark.toml. Issuers, clients, may_act_grants. Or POST to the admin API.
STEP 03
Mint agent tokens
OAuth 2.1, OIDC, Token Exchange, DPoP. Audit by grant_id. Done.
What makes it different
Six primitives that nobody else actually ships.
may_act_grants
Authorization that knows who you are and who you brought.
Express delegation policy as data. "research-orchestrator may act on behalf of jane@acme.dev for read:gmail when actor presents DPoP key kid=ec-p256-7Hx." Granted, scoped, expirable, revocable.

◆ spec: act / actor / may_act
RFC 9449 DPoP
Tokens bound to keys, not bearers.
Every access token is cryptographically pinned to the agent's key. Steal the token, you get nothing. Steal the key on a TPM-protected device — also nothing.

◆ replay-resistant by default
Full act chain
Provable provenance up to depth N.
When agent A delegates to agent B who calls API C, the act chain is preserved end-to-end and surfaced in introspection. No more "the agent did it" with nothing else to go on.

◆ chain depth observed: up to 7
Cascade revocation
Pull one thread, the whole graph unravels.
Revoke a grant, a session, an actor — and every downstream token derived from it goes with it. Indexed by grant_id, propagated through the introspection endpoint in single-digit ms.

◆ p99 propagation < 12 ms
~40 MB binary
Embedded SQLite. Zero deps. Anywhere.
Static Go binary with WAL-mode SQLite inside. Run it next to your app, in a Fly.io machine, in a corporate VM, on an air-gapped Pi. Backup is cp shark.db.

◆ cold start: 38 ms
Audit by grant_id
Every byte that left the building, attributable.
Audit log is structured JSON, addressable by grant_id, subject, actor, or grant. Stream to your SIEM via the events websocket. Compliance team will not believe their eyes.

◆ append-only · hash-chained
The flow
Watch a delegated request cross the wire.
POST /oauth/token · RFC 8693
1
Authenticated user
jane@acme.dev
OIDC code+PKCE · iss https://auth.acme.dev
2
Agent A presents DPoP
research-orchestrator
kid ec-p256-7Hx · htu /token · htm POST
3
Token Exchange
subject + actor
grant_type token-exchange · scope read:gmail
4
Audit, scoped, bound
access_token (DPoP-bound)
grant_id g_8FaQ••• · exp 5m · act_chain depth 2
← 200 OK · application/json
DPoP-bound · 5m
{
"access_token": "eyJhbGciOi…",
"token_type": "DPoP",
"expires_in": 300,
"scope": "read:gmail read:calendar",
"grant_id": "g_8FaQ4tR7L",
"act": {
"sub": "research-orchestrator",
"act": { "sub": "jane@acme.dev" }
},
"cnf": {
"jkt": "Bf9c…ec-p256-7Hx"
}
}
Admin UI
The console is embedded. The surface is small. The depth is real.
Every screen is a thin shell over the same admin API your CI/CD already calls. Nothing is "console-only."

shark://admin / delegation_canvas
grant_id
g_8FaQ•••
act_chain depth 2
Delegation Canvas
jane@acme.dev → research-orchestrator → web-fetcher
Reset
Revoke B
Cascade revoke
may_act → research-orchestrator
scope: read:gmail read:calendar · exp 1h
token_exchange (RFC 8693)
scope: web:fetch · exp 5m · DPoP-bound
Jane Park
user · acme.dev
kid: rsa-2048-aD3
VERIFIED
research-orchestrator
agent_a · openai-gpt5
kid: ec-p256-7Hx
ACTIVE
web-fetcher
agent_b · anthropic-haiku
kid: ec-p256-2Mn
ACTIVE
act_chain: [user:jane@acme.dev → agent_a:research-orchestrator → agent_b:web-fetcher]
16:42:11.084 POST /token grant_type=urn:ietf:params:oauth:grant-type:token-exchange · subject=jane@acme.dev · actor=research-orchestrator
200 OK
shark://admin / applications
5 clients
research-orchestrator
kid ec-p256-7Hx
agent
read:gmail · read:calendar · web:fetch
4 grants
web-fetcher
kid ec-p256-2Mn
agent
web:fetch
1 grants
acme-web
kid rsa-2048-bX1
spa
openid profile email
0 grants
acme-mobile
kid rsa-2048-cY9
native
openid profile email
0 grants
invoice-bot
kid ec-p256-9Lm
agent
read:invoices · write:invoices
2 grants
shark://admin / audit · grant_id g_8FaQ4tR7L
live
16:42:11.084
token.issue
jane@acme.dev → research-orchestrator
OK
16:42:11.219
token.exchange
research-orchestrator → web-fetcher
OK
16:42:11.681
introspect
web-fetcher active=true
OK
16:43:05.102
grant.revoke
g_8FaQ4tR7L (cascade=true)
OK
16:43:05.144
token.deny
web-fetcher reason=revoked
401
16:43:05.601
introspect
web-fetcher active=false
OK
Devlogs · /vlogs
Building in the open.
Bi-weekly write-ups on the future of agentic auth.
View all logs
APR 24, 2026
ARCHITECTURE
Zero-Code Auth Proxy: The Performance Gap
Benchmarking the new Go-based identity proxy against existing OIDC gateways. We achieved <2ms overhead on the hot path.

Read log
APR 12, 2026
PHILOSOPHY
Designing for the Agent Era
Why standard OAuth 2.1 isn’t enough for autonomous agents. Exploring DPoP and token exchange at scale.

Read log
MAR 28, 2026
RELEASE
Shark v0.9.0: Now Shipping
Our biggest release yet. Embedded SQLite, RFC 9449 compliance, and the new cascade revocation engine.

Read log
Use cases
The shape of agent infrastructure people are actually shipping.
01
Personal AI assistant with real keys to your kingdom.
Your assistant reads your inbox because you said it could — bound to one DPoP key, scoped to one label, expiring at 5pm.

02
Multi-agent orchestrator with provable provenance.
Orchestrator fans out to a fetcher, a summarizer, a writer. Each link in the chain is signed, scoped, and revocable independently.

03
Embedded auth for OSS SaaS.
Drop the binary next to your app. Get OAuth 2.1, OIDC, SCIM, and an admin UI without a vendor on the critical path.

04
Internal platforms that need real audit.
Every agent action attributable to a human, indexed by grant_id, hash-chained, replayable into your SIEM.

05
Edge & air-gapped deployments.
A 40 MB binary on a Pi, talking to a SQLite file. No outbound calls. No phone-home. No "free until we change our mind."

06
Compliance teams who have seen too much.
MIT, hash-chained audit, deterministic config in shark.toml. Auditors get receipts. You get to sleep.
