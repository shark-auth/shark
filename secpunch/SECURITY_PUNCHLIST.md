# Security Punchlist for v0.1 Launch

This folder contains a pytest suite that targets the launch-blocking issues found in the admin bootstrap, OAuth delegation, runtime config, and admin frontend flows.

Run it from the repository root:

```powershell
python -m pytest secpunch
```

The suite has static source checks and optional integration checks. Integration checks start `./shark.exe` on Windows or `./shark` elsewhere unless `SHARK_SECPUNCH_BASE_URL` points at an already running server.

Useful environment variables:

- `SHARK_SECPUNCH_BINARY`: path to the shark binary to start.
- `SHARK_SECPUNCH_BASE_URL`: use an already running server instead of starting one.
- `SHARK_SECPUNCH_ADMIN_KEY`: admin key for an already running server.
- `SHARK_SECPUNCH_SKIP_INTEGRATION=1`: run static tests only.

## P0: Public firstboot admin key disclosure

Unauthenticated `GET /api/v1/admin/firstboot/key` can return a live `sk_live_*` admin key. The recent consume-on-read change limits repeated disclosure, but the first browser or remote caller to hit the endpoint can still steal the key. The frontend login page also fetches this endpoint without authentication.

Proposed fix:

- Remove public key return entirely, or require a local-only setup nonce printed to the operator and validated by the endpoint.
- Do not expose `sk_live_*` in frontend JSON.
- Treat the file on disk as server-side secret material only.
- Add an explicit first-run setup flow that cannot be won by a remote unauthenticated request.

Covered by:

- `test_firstboot_key_endpoint_never_publicly_returns_secret`
- `test_login_frontend_does_not_fetch_firstboot_key`

## P0: Bootstrap can be reissued from audit-log heuristics

`MintBootstrapToken` decides whether setup is complete by scanning recent audit rows for admin activity. A durable admin key can exist without such an audit row, causing restarts to mint/print bootstrap material again.

Proposed fix:

- Store setup completion in durable state, or derive it from durable admin key/user state.
- Never use recent audit logs as the source of truth for whether bootstrap is complete.
- Validate `SHARKAUTH_BOOTSTRAP_ADMIN_KEY` format on startup.

Covered by:

- `test_bootstrap_completion_not_based_on_recent_audit_rows`
- `test_invalid_env_bootstrap_key_is_rejected`

## P0: Runtime config is persisted but not loaded

Admin PATCH writes runtime config to the database, but startup does not load it back. Settings appear saved in the current process and silently reset after restart.

Proposed fix:

- Call `config.LoadRuntime` during server startup after DB initialization and before routes/middleware are built.
- Make the active runtime config reflect persisted DB state on every boot.

Covered by:

- `test_runtime_config_survives_restart`

## P0: Dashboard exposes ignored server settings

The frontend sends settings such as `server.base_url`, `server.port`, and `server.cors_origins`, but the backend PATCH handler only accepts a subset such as `server.cors_relaxed`. This makes critical production settings look editable while being ignored.

Proposed fix:

- Either persist/apply the fields the UI exposes, or reject unknown/unsupported fields with a 4xx response and field-specific errors.
- For restart-required settings, persist and mark them pending.

Covered by:

- `test_server_config_fields_are_not_silently_ignored`

## P0: Revoked subject tokens can still be exchanged

`/oauth/revoke` marks `oauth_tokens.revoked_at`, but token exchange only checks `revoked_jtis`. A revoked access token can still be used as a `subject_token` for token exchange.

Proposed fix:

- During token exchange, look up the subject token/JTI and reject if the persisted token row has `revoked_at`.
- Keep `revoked_jtis` as an acceleration/cache path, not the only revocation source.

Covered by:

- `test_revoked_subject_token_cannot_be_exchanged`

## P0: Token exchange does not require a live may_act grant

Token exchange currently allows delegation when the subject JWT lacks `may_act`, and the database grant lookup is best-effort. This can mint unauthorized delegated tokens.

Proposed fix:

- Require a live persisted grant, or a trusted `may_act` claim issued by Shark and verified against policy.
- Treat missing policy as deny.

Covered by:

- `test_token_exchange_requires_live_may_act_grant`

## P1: DPoP-bound tokens accepted as bearer for vault reads

Vault token retrieval accepts `Authorization: Bearer` and checks DB expiry/scope, but does not require or validate a DPoP proof against the token JKT.

Proposed fix:

- If an access token has `cnf.jkt`, require a valid `DPoP` proof for the method and URL.
- Compare the proof key thumbprint to `cnf.jkt`.

Covered by:

- `test_vault_reads_validate_dpop_for_bound_tokens`

## P1: Session JWTs can survive session deletion

When JWT-backed sessions are enabled, deleting a session removes DB/cache state but does not necessarily revoke the JWT JTI or force middleware to verify that the session row still exists.

Proposed fix:

- Revoke the session JWT JTI on session deletion, or make middleware require the session row to still be active.
- Tests should cover deleted session JWT reuse against an authenticated endpoint.

Covered by:

- `test_session_jwt_auth_checks_live_session_or_revocation`

## P1: DPoP HTU path comparison is case-insensitive

DPoP HTU comparison uses case-insensitive URL equality. Scheme and host can be case-insensitive, but path and query must be exact.

Proposed fix:

- Normalize scheme/host as needed, but compare path and query byte-for-byte.

Covered by:

- `test_dpop_htu_path_is_case_sensitive`

## P1: Admin key stored in localStorage

The admin frontend stores the bearer admin key in `localStorage`. Any same-origin XSS becomes full admin compromise.

Proposed fix:

- Prefer short-lived server sessions in `HttpOnly`, `Secure`, `SameSite` cookies.
- If bearer keys must exist, keep them in memory only and require re-entry after reload.

Covered by:

- `test_admin_key_not_persisted_in_local_storage`

## P2: Dev email links can render unsafe schemes

The dev email viewer extracts any `href` containing magic/token/reset and renders it as a clickable link. `javascript:` or other unsafe schemes can be shown as links.

Proposed fix:

- Parse URLs and allow only `http:` and `https:`.
- Render unsafe values as inert text.

Covered by:

- `test_dev_email_links_reject_unsafe_schemes`

## P2: Delegation canvas may pass lane-prefixed IDs to grant APIs

The frontend graph can use synthetic lane-prefixed node IDs, but drawer grant queries need canonical agent IDs. Passing raw graph IDs makes grants appear empty or wrong.

Proposed fix:

- Canonicalize graph node IDs before every backend call.
- Keep display IDs separate from API IDs.

Covered by:

- `test_delegation_grant_drawer_uses_canonical_agent_ids`

## P2: Delegation chains ignore audit pagination

The chain loader reads a fixed first page of audit rows and does not follow cursors. Larger demos or real deployments can show partial/duplicated stale chains.

Proposed fix:

- Follow `next_cursor` until exhausted or a bounded safety cap is reached.
- De-duplicate events by stable event ID.

Covered by:

- `test_delegation_chain_loader_follows_audit_pagination`

## P2: Docker Go version mismatch

`go.mod` requires Go 1.26.2 while the Dockerfile references Go 1.25. This can break release builds.

Proposed fix:

- Keep Dockerfile builder image aligned with `go.mod`.
- Consider a single version source for CI/release.

Covered by:

- `test_dockerfile_go_version_matches_go_mod`
