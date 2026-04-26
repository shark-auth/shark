# Investigation Report — Proxy Bugs + CLI/CRUD Parity
Date: 2026-04-26

---

## Part 1 — Proxy Bug Hunt

### P0 Findings

**P0-1 — Hop-by-hop headers not stripped from upstream response / client request**
- File: `internal/proxy/proxy.go` — `director()` function (line 363–392)
- Root cause: The custom `director` never removes HTTP/1.1 hop-by-hop headers (`Connection`, `Proxy-Connection`, `Proxy-Authorization`, `TE`, `Trailers`, `Transfer-Encoding`, `Upgrade`, `Keep-Alive`) from the outbound request before forwarding. `httputil.ReverseProxy` strips these only when using its *default* director; a custom director that does not call `httputil.NewSingleHostReverseProxy` loses that protection entirely. Attackers or buggy clients can slip `Connection: Upgrade` or `Proxy-Authorization` headers through to the upstream service.
- Evidence: No string match for `Connection`, `Upgrade`, `hop`, or `Transfer-Encoding` anywhere in `internal/proxy/proxy.go`, `headers.go`, or `rules.go`. The `StripIdentityHeaders` function (headers.go:56) strips only `X-User-*`, `X-Agent-*`, `X-Shark-*` — it has no knowledge of hop-by-hop headers. The director (proxy.go:363) only rewrites Scheme/Host/Path.
- Repro: Send `curl -H "Connection: keep-alive, Upgrade" -H "Upgrade: websocket"` through the proxy; the upstream receives both headers raw.
- Fix: In `director()`, call `httputil.RemoveHopByHopHeaders(req.Header)` (available from `net/http/httputil` internals) or manually delete the standard hop-by-hop set before forwarding.

**P0-2 — DPoP JTI cache not wired by default in Listener/NewListener — enforcement silently disabled**
- File: `internal/proxy/listener.go` — `NewListener()` (line 67–124)
- Root cause: `NewListener` builds a `ReverseProxy` via `New(cfg, engine, logger)` but never calls `handler.SetDPoPCache(...)`. The proxy field `dpopCache` stays nil. Per proxy.go:187, `if id.AuthMethod == identity.AuthMethodDPoP && p.dpopCache != nil` — when `dpopCache` is nil, DPoP enforcement is **completely skipped** for all multi-listener deployments. Any `AuthMethodDPoP` identity passes through without proof validation.
- Evidence: `dpop-jti-global-vs-perlistener` grep confirms `SetDPoPCache` is declared (proxy.go:143) but no call to it appears in `listener.go`. `ListenerParams` has no `DPoPCache` field.
- Repro: Start a listener-based proxy, present a DPoP-bound bearer token — the DPoP proof field is ignored entirely.
- Fix: Add `DPoPCache *oauth.DPoPJTICache` to `ListenerParams`; call `handler.SetDPoPCache(p.DPoPCache)` in `NewListener` after line 99.

---

### P1 Findings

**P1-1 — HalfOpen state allows multiple concurrent live probes (no in-flight guard)**
- File: `internal/proxy/circuit.go` — `onSuccessLocked()` (line 412) and `probe()` (line 346)
- Root cause: The health monitor fires on a ticker; `probe()` takes the write lock only after the HTTP call returns (line 369). If `HealthInterval` is short (e.g. 1 s) and the upstream is slow, two ticks can be in-flight simultaneously. Both complete, both see state==Open, both call `onSuccessLocked()` which transitions Open→HalfOpen on the first and HalfOpen→Closed on the second in rapid succession. The intended "one probe to confirm recovery" semantics are violated; a single slow-success can jump straight to Closed.
- Evidence: `probe()` at line 347–381: the HTTP call (line 359) runs outside the lock; `b.mu.Lock()` is acquired only at line 369. `BreakerResolver.Resolve` also calls `br.callLive` in HalfOpen state (circuit.go:586–591) from request goroutines in parallel with the probe goroutine — there is no in-flight counter or sync.Once guarding a single probe.
- Repro: `go test -race -count 100` with a mock server that sleeps 2 s per response and HealthInterval=1 s; observe `b.state == Closed` after a single success.
- Fix: Add an `atomic.Bool probing` field; set it in `onSuccessLocked` when transitioning Open→HalfOpen and clear it on the next probe, preventing concurrent probes.

**P1-2 — `validateDPoPProof` swallows the underlying error detail, returns generic string**
- File: `internal/proxy/proxy.go:476`
- Root cause: `oauth.ValidateDPoPProof` returns a specific error (e.g. "dpop: jti already seen", "dpop: iat too old", "dpop: alg mismatch") but the wrapper discards it: `return fmt.Errorf("dpop proof invalid")`. The `X-Shark-Deny-Reason` header surfaces only the generic string, making replay attacks and clock-skew issues invisible in operator logs and client debugging.
- Evidence: proxy.go:475–477 — the specific `err` from `ValidateDPoPProof` is captured but immediately discarded; only the generic string is returned.
- Fix: `return fmt.Errorf("dpop proof invalid: %w", err)` — surfaces the root cause while still being safe to put on the response header (DPoP error strings from `internal/oauth/dpop.go` do not contain user-supplied data).

**P1-3 — `fullURL` trusts client-supplied `X-Forwarded-Proto` without validation — open redirect risk**
- File: `internal/proxy/proxy.go:350–356` (`fullURL`)
- Root cause: `fullURL` uses `r.Header.Get("X-Forwarded-Proto")` verbatim to construct the `return_to` URL injected into the login redirect. Any client can send `X-Forwarded-Proto: javascript` (or any other string) which becomes `javascript://host/path` — a URL that browsers will attempt to navigate to. The login page then redirects to that URL after authentication.
- Evidence: proxy.go:352 — no validation or allowlist on the header value. `StripIdentityHeaders` does not strip `X-Forwarded-Proto` (it only strips `X-User-*`, `X-Agent-*`, `X-Shark-*`).
- Repro: Send `X-Forwarded-Proto: javascript` → proxy redirects browser to `javascript://host/path?return_to=javascript://...` — open redirect.
- Fix: Allowlist: `if proto == "http" || proto == "https" { scheme = proto }` — reject anything else.

**P1-4 — `http.Server` in Listener has only `ReadHeaderTimeout`, no `ReadTimeout` or `WriteTimeout`**
- File: `internal/proxy/listener.go:107–111`
- Root cause: The `http.Server` constructed in `NewListener` sets only `ReadHeaderTimeout: 10 * time.Second`. `ReadTimeout` and `WriteTimeout` are zero (unlimited). A slow client that sends headers quickly but trickles the request body (Slowloris body attack) can hold a goroutine indefinitely. Similarly, a slow upstream response holds a write goroutine forever even when the proxy timeout context fires — the context cancels the director's outbound request but `http.Server.WriteTimeout` is not set, so the goroutine writing the error response to the client can also block.
- Evidence: listener.go:107–111 shows only `ReadHeaderTimeout`.
- Fix: Add `ReadTimeout: p.Timeout + 5*time.Second` and `WriteTimeout: p.Timeout + 10*time.Second` to the `http.Server` literal.

**P1-5 — `probe()` body not drained before `Close()` — connection pool poisoning**
- File: `internal/proxy/circuit.go:364`
- Root cause: `_ = resp.Body.Close()` closes the response body without draining it first. For HTTP/1.1 upstream health endpoints that return a body (e.g. JSON health payload), the underlying TCP connection cannot be returned to the client's transport pool and is instead closed, wasting connections and increasing latency on each probe interval.
- Evidence: circuit.go:364 — `Close()` only, no `io.Copy(io.Discard, resp.Body)` before close.
- Fix: `_, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close()` at circuit.go:364.

---

### P2 Findings

**P2-1 — Path matching does not normalize percent-encoded segments — double-encoded bypass**
- File: `internal/proxy/rules.go` — `compilePath` / `match` (lines 589–697)
- Root cause: `match()` compares raw `r.URL.Path` segments against literal rule segments using `==`. Go's `net/http` server decodes `%2F` in the path before routing *only for simple paths*; `r.URL.RawPath` preserves the original if encoding is non-standard. A client sending `/api/%61dmin` (where `%61` = `a`) will be compared as the literal string `%61dmin` against rule segment `admin` — it will fail to match an `admin` rule and may bypass a deny rule intended to block `/api/admin`. Conversely, double-encoding `%2561` (decoded to `%61`, not `a`) could bypass an allow rule.
- Evidence: `match()` at rules.go:660 calls `strings.Split(trimmed, "/")` on `strings.TrimPrefix(urlPath, "/")` — `urlPath` is `r.URL.Path` which Go already decoded once, but `r.URL.RawPath` may differ and is never consulted.
- Repro: Rule `/api/admin` → `require:authenticated`. Send `GET /api/%61dmin` — no rule matches, falls through to "no rule matched" → default deny. Depending on application, this may expose or hide resources.
- Fix: In `Evaluate`, use `r.URL.EscapedPath()` and then `url.PathUnescape()` to normalize before matching, or compare against both `r.URL.Path` and `r.URL.RawPath`.

**P2-2 — `failureCounts` (`fails`) unbounded growth in Open state**
- File: `internal/proxy/circuit.go` — `onFailureLocked()` (lines 387–405)
- Root cause: In `Open` state, `onFailureLocked` increments `b.fails` on every probe failure (line 388: `b.fails++`) without a cap. Over a long outage at 10-second probe interval, `b.fails` grows to arbitrarily large integers. This is purely a cosmetic/stats issue (no divide-by-zero risk since `FailureThreshold` is only consulted in Closed state), but `Stats()` surfaces `Failures` on the wire and in logs, potentially confusing operators with enormous numbers after hours of downtime.
- Evidence: onFailureLocked line 387–406: `b.fails++` at line 388, then `switch b.state` — the `case Open:` branch (implicit fall-through to nothing) still increments but never resets or caps.
- Fix: Cap `b.fails` in the Open case: `if b.fails > 1000 { b.fails = 1000 }` or don't increment in Open state.

**P2-3 — `Stop()` can panic if called twice concurrently**
- File: `internal/proxy/circuit.go:213–233`
- Root cause: `Stop()` releases the lock before the `select` block (line 225 `b.mu.Unlock()`), then checks `b.stopCh` with a `select`. Two concurrent goroutines can both pass the `if !b.started` check (it's read-locked above), then both attempt `close(b.stopCh)` — the second close panics with "close of closed channel".
- Evidence: circuit.go:226–232 — the `select { case <-b.stopCh: default: close(b.stopCh) }` pattern is not safe if two goroutines reach it concurrently, because the lock was already released at line 225.
- Fix: Use a `sync.Once` to close `stopCh`, or hold the lock across the entire Stop body.

**P2-4 — `lruCache` uses `sync.Mutex` (exclusive lock) for `get()` — unnecessary contention**
- File: `internal/proxy/lru.go:60–84`
- Root cause: `get()` takes a full write lock (`c.mu.Lock()`) even though most paths are read-only (the `MoveToFront` call is the only mutation). Under high request concurrency (10k RPS spec), every cache lookup serializes all goroutines. A `sync.RWMutex` would allow concurrent reads in the common non-expired-entry path without locking out writers.
- Evidence: lru.go:61 `c.mu.Lock()` — `lruCache.mu` is typed as `sync.Mutex` (line 24), not `sync.RWMutex`.
- Fix: Separate the read path (check + return) using `c.mu.RLock()` when not moving to front; upgrade to write lock only on expiry eviction or MoveToFront. Alternatively accept the current design as intentional for simplicity.
- Severity: P2 (performance, not correctness).

**P2-5 — `Manager.Reload()` sets state=StateReloading then calls `stopLocked()` which resets state=StateStopped on any Stop failure, losing the Reloading→Running audit trail**
- File: `internal/proxy/lifecycle.go:180–203`
- Root cause: `Reload()` sets `m.state = StateReloading` at line 184, then calls `stopLocked()`. If `stopLocked` fails, it transitions to `StateStopped` (line 189). But `startLocked()` at line 109 checks `if m.state == StateRunning { return error }` — after a failed stop, state is Stopped, so start proceeds. The net result is that an observer polling `/status` during a failed reload sees Reloading→Stopped→Running without any indication that the stop half failed. The error is still returned to the caller, but `lastError` is overwritten by `startLocked`'s success.
- Fix: Preserve the stop error in a local and surface it alongside start success in the Status or return a wrapped error.

**P2-6 — `StripIdentityHeaders` does not strip `X-Auth-Method` (only `X-Shark-*` prefix)**
- File: `internal/proxy/headers.go:33,56`
- Root cause: `strippedPrefixes` is `["X-User-", "X-Agent-", "X-Shark-"]`. The constant `HeaderAuthMethod = "X-Auth-Method"` (headers.go:26) does NOT match any prefix — it starts with `X-Auth-`, not `X-User-`, `X-Agent-`, or `X-Shark-`. A client can inject `X-Auth-Method: jwt` and it will survive `StripIdentityHeaders`, then be overwritten by `InjectIdentity`. However if `StripIncoming` is false (the intentionally unsafe path), the client-supplied value leaks to upstream.
- Evidence: headers.go:33 — no `X-Auth-` in strippedPrefixes. InjectIdentity at line 100 does `setOrDelete(h, HeaderAuthMethod, ...)` which overwrites, but only after the rule engine has potentially already acted on it.
- Fix: Add `"X-Auth-"` to `strippedPrefixes`.

---

## Part 2 — CLI/CRUD Parity Matrix

| Resource | API CRUD | CLI | Dashboard | Gaps |
|---|---|---|---|---|
| user | C✓ R✓ U✓ D✓ L✓ (`admin_user_handlers.go`, `user_handlers.go`) | create, show, list, update, delete, tier (`user_create.go`…`user_tier.go`) | `users.tsx` ✓ | No CLI `user password-reset` or `user verify-email`; API has those flows |
| application | C✓ R✓ U✓ D✓ L✓ (`application_handlers.go`) | create, show, list, update, delete, rotate (`app_create.go`…`app_rotate.go`) | `applications.tsx` ✓ | No CLI for hosted-page config; no CLI `app import/export` |
| agent | C✓ R✓ U✓ D✓ L✓ (`agent_handlers.go`) | register, show, update, delete, revoke-tokens, rotate-secret (`agent_*.go`) | `agents_manage.tsx` ✓ | No CLI `agent list`; only `register`+`show`+`update`+`delete` — missing list subcommand |
| api_key | C✓ R✓ D✓ L✓ (`apikey_handlers.go`) | create, list, revoke, rotate (`api_key_*.go`) | `api_keys.tsx` ✓ | No CLI `api-key show <id>`; no API update (by design — keys are rotated not patched) |
| webhook | C✓ R✓ U✓ D✓ L✓ (`webhook_handlers.go`) | **none** — `webhook_emit.go` is server-side only; no CLI webhook subcommand | `webhooks.tsx` ✓ | **CLI entirely absent**: no `shark webhook list/create/delete/test`. P1 gap. |
| rbac (role, permission) | C✓ R✓ U✓ D✓ L✓ for global roles+perms (`rbac_handlers.go`); C✓ R✓ D✓ for org roles (`org_rbac_handlers.go`) | **none** — no `rbac.go` subcommands with CRUD verbs found | `rbac.tsx`, `rbac_matrix.tsx` ✓ | **CLI entirely absent**: no `shark role list/create`, `shark permission list/assign`. P1 gap. |
| organization | C✓ R✓ U✓ D✓ L✓ (`organization_handlers.go`, `admin_organization_handlers.go`) | `org show` only (`org_show.go`) — no create/delete/update/list | `organizations.tsx` ✓ | CLI missing create, list, update, delete. Only `show` exists. |
| consent | R✓ D✓ L✓ (`consent_handlers.go`; list+revoke+admin list) | `consents` parent command exists (`consent.go`) but **no subcommands** wired | `consents_manage.tsx` ✓ | CLI parent exists but is empty — no `list`, `revoke`, `show` subcommands. |
| session | R✓ D✓ L✓ (`session_handlers.go`; list/show/revoke/admin-list/admin-delete/purge) | list, show, revoke (`session_list.go`, `session_show.go`, `session_revoke.go`) | `sessions.tsx` ✓ | No CLI `session purge-expired`; no CLI admin-revoke-all |
| audit | R✓ L✓ (`audit_handlers.go`) | `audit export` only (`audit_export.go`) | `audit.tsx` ✓ | No CLI `audit list` with filters; export only |
| sso connection | C✓ R✓ U✓ D✓ L✓ (`sso_handlers.go`) | create, list, show, update, delete (`sso_*.go`) | `sso.tsx` ✓ | Full parity |
| oauth client (DCR) | C✓ R✓ U✓ D✓ (`admin_oauth_handlers.go`) | **none** — no `shark oauth-client` or `shark dcr` subcommand | `authentication.tsx` (partial) | **CLI entirely absent** for DCR management. |
| vault entry | C✓ R✓ U✓ D✓ L✓ for providers; R✓ D✓ for connections; token GET (`vault_handlers.go`) | `vault provider` parent + `vault_provider_show.go` only | `vault_manage.tsx` ✓ | CLI missing `vault provider list`, `vault provider create/update/delete`, `vault connection list/delete`, `vault token` |
| branding | C✓ R✓ U✓ D✓ (`branding_handlers.go`) | `branding set <slug>`, `branding get <slug>` (`branding.go`) | `branding/` directory + `branding.tsx` ✓ | No CLI `branding delete` or `branding list`; partial parity |
| system_config | R✓ U✓ (`admin_system_handlers.go` config endpoint; `system_handlers.go` mode/reset) | `mode`, `reset`, `admin-config-dump` (`mode.go`, `reset.go`, `admin_config_dump.go`) | `system_settings.tsx` ✓ | No CLI for individual config key get/set; `admin-config-dump` is read-only |
| secret | no dedicated secret CRUD API — secrets managed via vault provider connect flow | **none** | no dedicated surface (covered under vault) | No explicit secret store API or CLI — managed implicitly via vault OAuth flow |

---

## Part 3 — Top 5 Recommended Fixes (Priority Order)

1. **[P0-1] Strip hop-by-hop headers in `director()`** (`internal/proxy/proxy.go` ~line 392): Add explicit removal of `Connection`, `Proxy-Connection`, `Proxy-Authorization`, `TE`, `Trailers`, `Transfer-Encoding`, `Upgrade`, `Keep-Alive` from the outbound request. This is the highest-severity header-smuggling vector — no test currently exercises it.

2. **[P0-2] Wire DPoP JTI cache into `NewListener` / `ListenerParams`** (`internal/proxy/listener.go` ~line 67): Add `DPoPCache *oauth.DPoPJTICache` to `ListenerParams` and call `handler.SetDPoPCache(p.DPoPCache)`. All multi-listener deployments currently have DPoP enforcement silently disabled, defeating replay protection entirely.

3. **[P1-3] Allowlist `X-Forwarded-Proto` values in `fullURL()`** (`internal/proxy/proxy.go:352`): Restrict accepted values to `"http"` and `"https"` before constructing the `return_to` URL. The current code enables an open-redirect via a crafted `X-Forwarded-Proto` header.

4. **[P1-1] Add in-flight guard for HalfOpen probes** (`internal/proxy/circuit.go`): Add an `atomic.Bool` that is set when the breaker enters HalfOpen and prevents the background ticker from firing a second concurrent probe. Without this, fast ticker + slow upstream can race two probes and jump from Open directly to Closed, skipping the required confirmation step.

5. **[P1-4] Add `ReadTimeout`/`WriteTimeout` to the Listener's `http.Server`** (`internal/proxy/listener.go:107`): Set `ReadTimeout` and `WriteTimeout` based on `p.Timeout` to prevent Slowloris-body attacks and goroutine leaks from unbounded slow writes.
