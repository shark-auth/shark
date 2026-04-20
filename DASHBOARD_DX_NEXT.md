# Dashboard DX — Next Up (resume here)

**Paused:** 2026-04-20T21:32Z
**Remote sha:** c5394a7 (PR #77, claude/admin-vendor-assets-fix)
**Tree:** clean (only untracked aigents.png + unrelated sharkauth.yaml)

## Score

22 of 24 tasks done. Overall DX: baseline 4.25/10 → current ~7.5/10.

| Task | Status | Commits |
|------|--------|---------|
| T01 credibility lies | done | 8406b78 |
| T02 overview mocks | done-prior | 989c319 / 7d464da |
| T03 signing rotate wire | done | 8406b78 (merged into T07) |
| T04 POST /admin/users backend | done | b922c0c + 342198f + smoke section 70 |
| T05 create-user slide-over | done | 2b4d649 + 80bef2e |
| **T06 org invitations slide-over** | **TODO** | — |
| T07 alert→toast | done | 8406b78 |
| T08 silent 404s verified | done-inspection | (prior c3609db) |
| T09 overview magical-moment hero | done | 7b34ba3 |
| T10 proxy empty-state wizard | done | 6660911 |
| T11 get-started page + redirect | done | 7e49e4f |
| T12 session debugger | done | 6380fba |
| T13 compliance demote | done | dc3cef2 + a4a47b6 (fix) + smoke section 69 |
| T14 hide phase-gates toggle | done | e5215b2 |
| T15 bootstrap token | done | eaa528f (backend) + c2dd599 (frontend) + c5394a7 (smoke 71) |
| **T16 ship-readiness checklist** | **TODO** | — |
| T17 Help menu | done | dc3cef2 |
| T18 Feedback button | done | dc3cef2 |
| T19 API keys scope picker | done-inspection | (prior at api_keys.tsx:602-614) |
| T20 webhook retry + errors | done | 9d067b1 |
| T21 RBAC permission matrix | done | 77920c3 + c2dd599 |
| T22 vault silent catches | done-inspection | (prior, all surface errors) |
| T23 CLIFooter blanket | done | 8406b78 |
| **T24 final ship review** | **TODO** | blocked on T06+T16 |

## Resume protocol

Next session, read this file first. Then:

1. `git pull --ff-only` on claude/admin-vendor-assets-fix
2. Check for missed notifications on prior subagents: review `DASHBOARD_DX_PROGRESS.md` tail + `DASHBOARD_DX_STATUS.json`
3. Dispatch the remaining tasks per the three briefs below (each ready-to-paste).

## T06 — Org invitations slide-over

**Gotcha surfaced by killed dispatch:** no admin POST create-invitation endpoint exists. Only non-admin one at `internal/api/router.go:341`. Options when you resume:
- **Option A (backend first):** ship `POST /admin/organizations/{id}/invitations` handler mirroring the session-auth version. Then frontend slide-over. ~1h CC.
- **Option B (reuse session endpoint):** frontend calls the non-admin endpoint with session cookie fallback — works if admin dashboard session is valid. Verify auth.
- **Option C (CLI-only for now):** ship read-only invitations list + resend + revoke in UI (endpoints exist). Replace "+Invite" button with copy-ready CLI. Ship T06 partial. 30 min.

Recommended: Option A. Matches acceptance bar and matches T04/T05 pattern.

Brief: see DASHBOARD_DX_SUBAGENT_BRIEFS.md "## Brief — T06".

## T16 — Ship-readiness checklist + topbar score

No gotchas. Brief ready in DASHBOARD_DX_SUBAGENT_BRIEFS.md "## Brief — T16".

Data sources all wired (/admin/stats, /admin/health, /admin/config, /admin/apps, /audit-logs).

Coordination: overview.tsx + layout.tsx both modified recently (T01, T09). Pull latest before starting. Magical-moment hero tile at T09 — place checklist compatibly.

## T24 — Final ship review

After T06 + T16 land:
1. Re-score all 8 DX dimensions
2. Run full smoke suite: `bash smoke_test.sh`
3. Run `go test ./... -count=1 -timeout 120s`
4. Run `cd admin && npx tsc -b --noEmit`
5. Update DASHBOARD_DX_REVIEW.md scorecard with post-ship numbers
6. Create PR description update with all 24 tasks rolled up
7. Tag release candidate (v0.9.0-dx or similar)

## Decisions still locked (do not revisit without new review)

Full list in DASHBOARD_DX_EXECUTION_PLAN.md "Decision Record" section.

Key ones relevant to remaining work:
- T06: full invite UI with email+role+expiry picker, shared pattern with T05
- T16: Overview card + topbar score badge (both places)
- T24: eng review recommended before ship; cut T-polish items if time tight
