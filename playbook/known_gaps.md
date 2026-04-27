# Known Gaps & Battle-Test Requirements

_Documented on 2026-04-27 based on Sonnet Subagent Audit_

## 1. SDK Coverage & Integrity

- **Claim vs Reality**: `documentation/sdk/index.md` claims 75% coverage. Actual coverage is ~56% for Python and ~55% for TypeScript.
- **Broken Wrap (TS)**: `VaultClient.fetchToken` calls non-existent route `/admin/vault/connections/{id}/token`. Should be `/api/v1/vault/{provider}/token`.
- **SDK Gaps**: Passkeys (7), SSO (9), Flow Builder (7), Vault provider/template CRUD (11), admin sessions/stats (~20).

## 2. The "Moat" Bugs (P0 Launch Blockers)

- **Max Hops Enforcement**: `may_act_grants.max_hops` is stored but NOT enforced in `internal/oauth/exchange.go:272`. Agents can delegate to arbitrary depths.
- **Scope Widening**: `may_act_grants.scopes` is NOT enforced. Agents can exchange for scopes wider than the grant allows if the subject token carries them.
- **DPoP Replay**: JTI cache is in-memory only. Replay window re-opens on server restart.

## 3. Developer Experience (DX) & Quickstart

- **Broken Quickstart**: README suggests `pip install shark-auth`, but the package isn't on PyPI and the workflow is missing.
- **Device Flow**: Routes are mounted (`/oauth/device*`) but `HandleToken` returns 501. Either un-mount or finish implementation.
- **React Syntax**: Potential stray paren in `SharkProvider.tsx:86`.

## 4. Dogfooding Requirements

- **Admin Self-Auth**: Replace bootstrap-key login with real OAuth 2.1 PKCE flow against its own server.
- **Release Agent**: Create a Python `release-agent` script that uses the delegation chain to post release notes to GitHub.
  ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀
  > A, images has those showing. also before going on that:  
  >  ❯ use sonnet subagents to validate, documentation/sdk/index claims 75%  
  >  endpoint
       coverage, is it true? second task: should I launch now? what is missing to
       battle test? run a full investigation with agents and generate a checklist
  with  
   instructions for qa and real battle-testing. I think a great option for  
   battle-testing could be dogfooding shark. you think of it.
  ● Two parallel sonnet subagents.
  ● 2 agents finished (ctrl+o to expand)  
   ├ SDK coverage claim audit · 0 tool uses  
   │ ⎿ Done  
   └ Launch readiness QA + dogfood plan · 0 tool uses  
   ⎿ Done
  ● Two reports back. Honest summary.
       SDK coverage — 75% claim is overstated

       Real numbers against 130 meaningful SDK-callable endpoints (excluded healthz
  /  
   SPA / static):  
   - Python: 73 / 130 = 56%  
   - TypeScript: 71 / 130 = 55%
       Closest favorable read (excluding ~40 admin-panel-only endpoints like
       email-templates, dev-inbox, system mode): 73/90 ≈ 81%.

       The documentation/sdk/index.md "75%" is wrong as written. Two options: fix
  doc  
   to honest number + denominator definition, OR ship enough additional SDK  
   methods  
   to actually hit 75%.
       Plus 1 broken wrap (silent 404): TS VaultClient.fetchToken calls
       /admin/vault/connections/{id}/token — that route doesn't exist. Real route:
       /api/v1/vault/{provider}/token. One-line fix.

       Big SDK gaps: Passkeys (7), SSO (9), Flow Builder (7), Vault
  provider/template  
   CRUD + user connect (11), admin sessions/stats/email-templates (~20).
       Launch readiness — TWO MOAT-CLASS BUGS

       SHIP IN 2 DAYS, not today. Here's why:

       1. may_act_grants.max_hops stored but NEVER enforced. Operator sets
  max_hops=1  
   in admin UI; agent can still delegate to depth 3+.  
   internal/oauth/exchange.go:272 logs it to audit, doesn't gate. This is the  
   moat.  
   Will be the first HN comment.  
   2. may_act_grants.scopes stored but NEVER enforced. Grant says ["mcp:read"];
  agent can exchange for mcp:write if subject token carries it. Same file.  
   Same  
   issue.  
   3. Device flow schizophrenic. Routes mounted at /oauth/device\*, but  
   HandleToken  
   returns 501 for device_code grant. Already hidden from sidebar this session,
  but  
   routes still advertised. Either un-mount or implement.  
   4. Python SDK quickstart in README doesn't work — pip install shark-auth  
   requires PyPI publish. Workflow missing. Cold-clone HN visitor will hit  
   error.  
   5. DPoP JTI cache in-memory only — across restart, 60s replay window  
   reopens.  
   Single-instance OK; security-aware reviewer will flag.
       What I propose right now

       Fire moat-fix subagent: enforce max_hops + grant.scopes intersection in
       exchange.go. Estimated ~50 lines. Plus tests proving 2-hop chain rejected
  when  
   max_hops=1, scope widening rejected.
       Plus separately: fix TS vault path (1 line), fix documentation/sdk/index.md
       honest number, fix SharkProvider.tsx:86 stray paren if it exists.

       Dogfood plan (subagent recommendation)

       Do before 4/29:
       1. Admin UI logs into SharkAuth via SharkAuth itself. Replace bootstrap-key
       login with real OAuth 2.1 PKCE flow against own server. ~4-6 hrs. The demo
  video  
   shows admin logging into SharkAuth via SharkAuth — every YC partner gets  
   the  
   point. Blog title writes itself.  
   2. Release agent. Real registered agent (release-agent Python script) that  
   uses  
   delegation chain (maintainer → release-agent → GitHub API) to post release  
   notes  
   via DPoP-bound delegated token. Audit log shows full chain. ~4-6 hrs total.
  Live demo flow + screenshot for YC app traction section.
       Defer: Vault-gated contributor dashboard on website (additive polish, not
       launch-blocking).

       Battle-test checklist

       Subagent shipped 40-item P0/P1 checklist covering API/server smoke (5),
  OAuth  
   flows (7), delegation moat (8), DPoP (4), Admin UI (6), SDK quickstart (4),  
   restart resilience (2), fresh install (3), CLI (10). Each item: single  
   command +  
   expected result + fail location. Want me to write it to  
   .planning/launch-checklist.md so you can check items off as you battle-test?
       ---
       Decision points for you:

       A) Fire moat-fix subagent now (max_hops + scope enforcement) — answer "go"
       B) Fix SDK doc claim — honest number vs ship more endpoints to actually hit
  75%?  
   Pick.  
   C) Dogfood priority — Admin UI OAuth, Release agent, both, or punt?  
   D) Persist battle-test checklist to disk?
       Hearing your sequencing.

  ✻ Cooked for 5m 55s
  ❯ please create a big plan with the subplans, for subagents to work on all  
   stuff  
   you just mentioned, no gaps, EVERYTHING must be donepypi release ill take  
   care  
   of it. I encourage you also to before doing that update readme with a  
   specialized subagent for stuff that will really make shark viral you can use
  images/ remember you should include new pytest cases for new logic or fixes.
  run the full pytest after all subagents finish.
  ● PLAN — Launch readiness, no gaps
       WAVE 1 (parallel, low-risk polish)
         ├─ W1_README     viral rewrite, screenshots/images, moat-forward
         ├─ W1_DOCS       SDK index honest coverage % + gap inventory
         ├─ W1_REACT      SharkProvider.tsx:86 stray paren audit + fix
         ├─ W1_DEVICE     device flow routes — un-mount cleanly for v0.1
         └─ W1_CHECKLIST  persist launch-checklist.md to disk

       WAVE 2 (parallel, after W1)
         ├─ W2_MOAT       enforce may_act max_hops + scope in exchange.go
         │                 + Go unit tests + pytest smoke for both gates
         ├─ W2_PASSKEYS   Python + TS clients (7 endpoints) + smoke tests
         ├─ W2_VAULT      fix TS path bug + provider CRUD + user connect
         │                 (11 endpoints) + smoke tests
         └─ W2_FLOW       Flow builder CRUD (7 endpoints) + smoke tests

       WAVE 3 (parallel, after W2)
         ├─ W3_ADMIN_OAUTH     replace bootstrap-key login with real
         │                     OAuth 2.1 PKCE against own server
         └─ W3_RELEASE_AGENT   demo-release-agent/ Python script + grant
                                + audit-chain screenshot

       WAVE 4 (final, sequential)
         ├─ E1   admin build + Go binary embed
         ├─ E2   FULL pytest run (tests/smoke + sdk smoke + new tests)
         └─ E3   ship report — coverage % delta, test pass count, binary path

       Dispatching Wave 1.

  ● 5 agents finished (ctrl+o to expand)  
   ├ README viral rewrite · 0 tool uses  
   │ ⎿ Done  
   ├ SDK docs honest coverage · 0 tool uses  
   │ ⎿ Done  
   ├ React provider stray paren fix · 0 tool uses  
   │ ⎿ Done  
   ├ Device flow un-mount · 0 tool uses  
   │ ⎿ Done  
   └ Persist launch checklist · 0 tool uses  
   ⎿ Done please put this into playbook/ as known gaps.
