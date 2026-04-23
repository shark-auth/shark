# SharkAuth Proxy V1.5 — The Agent-Built Gateway

**Status:** Plan Phase (Pre-Implementation)
**Target:** 2026-04-25
**Core Thesis:** Turn the reverse proxy from a static "upstream pipe" into a dynamic, identity-aware ingress that can be configured by autonomous agents via the Admin API.

## 1. Objectives
- **Dynamic Routing:** Add/Update proxy rules without restarting the `shark` binary.
- **Identity Gating:** Inject X-Shark-User-ID and X-Shark-Scope headers automatically after verifying JWT/Session.
- **CEL Support:** Implement Common Expression Language (CEL) rules for path-level access control.
- **Transparent Mode:** Support "pass-through" for legacy endpoints while protecting new ones.

## 2. Technical Specs
- **Data Plane:** Refactor `internal/proxy/proxy.go` to use a thread-safe routing table.
- **Control Plane:** New Admin API endpoints:
  - `GET /admin/proxy/rules`
  - `POST /admin/proxy/rules`
  - `DELETE /admin/proxy/rules/{id}`
- **Security:** Proxy MUST verify DPoP proofs if present in the incoming request.

## 3. Implementation Steps
1. **[LANE A]** Implement `internal/proxy/engine.go` (CEL rule evaluator).
2. **[LANE B]** Wire Proxy rules to SQLite storage.
3. **[LANE C]** Add "Proxy" management tab to Admin Dashboard.
4. **[LANE D]** Add `shark proxy rules add` CLI command.

## 4. Testing Plan
- Smoke test: Send request to `/protected` -> get 401.
- Smoke test: Send valid JWT to `/protected` -> proxy to upstream with headers.
- Stress test: 10k RPS through proxy with dynamic rule updates.

---

### 6. Architectural & Testing Requirements (Eng Review)

#### 6.1. Lock-Free Data Plane
The Proxy routing table MUST use `atomic.Pointer[RoutingTable]` for configuration updates. This ensures sub-millisecond response times by eliminating `RWMutex` contention during high-volume agentic traffic.

#### 6.2. Header Spoofing Protection
The Proxy MUST explicitly strip any incoming `X-Shark-*` headers from the client before injecting authenticated identity headers. A regression test `tests/smoke/test_proxy_spoofing.py` must verify this.

#### 6.3. CEL Integration
Access rules will be defined using **Common Expression Language (CEL)**. 
- **Security:** CEL is non-Turing complete and sandboxed (no loops, no system access).
- **Performance:** Rules MUST be pre-compiled during the hot-reload phase to minimize request-time latency.

#### 6.4. Testing Matrix (Gaps to Close)
- **E2E Onboarding:** Verify the flow from `shark serve --dev` -> Magic Link -> `/get-started` -> Proxy Wizard.
- **Walkthrough Persistence:** Verify `localStorage` correctly hides the walkthrough after completion or skip.
- **Agent Delegation (DPoP):** Smoke test for RFC 8693 token exchange via the Proxy.

#### 6.5. CLI Interactivity (DX Polish)
`shark serve` MUST detect first-time boot (0 users) and provide an interactive prompt: `Open Dashboard? [Y/n]`. This reduces TTHW by eliminating manual URL copying.

#### 6.6. Component Scaffolding (Phased)
- **Phase 1 (v0.1):** Dashboard-based visual branding with copy-paste code snippets.
- **Phase 2 (v0.2):** CLI command `shark components add <component>` to scaffold editable Tailwind/CSS components directly into the local repo.

### 7. Implementation Steps (Worktree Strategy)
Lane A: Proxy Core (atomic pointer, CEL integration, header stripping)
Lane B: Onboarding E2E (frontend routing fixes, smoke tests, interactive CLI prompt)
Lane C: Agentic Trace (DPoP verification, delegation headers)

Launch A + B in parallel. Then C.

---
[gstack-context]
This plan was initialized after the DX review found the proxy was the highest-friction part of onboarding. We are prioritizing "No-Bullshit" integration for Cursor/Claude-driven builds.
[/gstack-context]
