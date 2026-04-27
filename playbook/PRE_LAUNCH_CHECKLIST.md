# Pre-Launch Checklist — 2026-04-26

Final state after F1–F10 launch-readiness sprint + security review.
Branch: `main` @ `7a81df7` (pushed).

---

## Smoke Baseline

```
382 pass / 11 fail / 5 err / 13 xfail / 66 skipped
```

Zero F1–F10 regressions. Every failure pre-existing, cut, or deferred.

### 11 Fails — Categorized

| Category | Count | Disposition |
|---|---|---|
| Pre-existing W1 work (delegation chains UI, audit quality) | 4 | Tracked in `13-delegation-chains-investigation.md` / `14-audit-log-quality-audit.md` |
| Cut features (Branding, Impersonate, Device flow, hosted pages) | 4 | Founder-locked exclusions — see below |
| Deferred (F1 setup-token persistence, F4 conftest wiring) | 2 | W+1 |
| F10 install.sh regex too broad on `docs.sharkauth.dev` | 1 | Cosmetic, deferred |

### 5 Errors — Fixture Issues

`w17_identity_settings`, `w2_sdk_method_8/9/10` — pre-existing pytest fixture problems, not blocking launch.

---

## Pre-Launch Sequence

Run in order, fresh shell, repo root.

```bash
# 1. Frontend bundle
pnpm --filter admin build

# 2. Go binary (embeds dist/)
go build -o shark.exe ./cmd/shark

# 3. Sync smoke fixture
cp -f shark.exe tests/smoke/shark.exe

# 4. Restart server (kill old, run fresh)
./shark.exe serve
```

Watch stdout for:
- First-boot admin key URL (only on virgin DB)
- Re-run banner if admin already configured
- Port 8080 bind confirmation

---

## Manual Verification Walkthrough

Founder must click through these 11 items before launch announce.

| # | Surface | Check |
|---|---|---|
| 1 | `/api/docs` | Scalar UI renders, no white screen, 118 endpoints visible |
| 2 | `/agents` user filter | Drill-down shows delegations (not "no delegations") for users with chains |
| 3 | `/applications` | Name + CORS editable inline, Proxy Rules tab GONE |
| 4 | `/orgs` | Invite flow works end-to-end |
| 5 | `/vault` | Provider edit (auth_url / token_url) accepts https only, rejects internal IPs |
| 6 | `/users` → roles tab | "Assign role" picker dropdown POSTs successfully |
| 7 | `/delegation-chains` | Canvas hover-tooltip edges, NodeDrawer opens on click |
| 8 | `/dev-email` (no `--dev` flag) | Tab visible to admin, polls every 1.5s silently |
| 9 | `shark serve` re-run | Banner reads "admin configured" or "setup pending" correctly |
| 10 | `shark doctor` | All 9 checks present, exit 0 healthy / exit 1 with admin_key missing |
| 11 | `shark demo` | End-to-end agent demo runs without manual prompts |

---

## Security Review — Resolved

| Vuln | Severity | File | Fix |
|---|---|---|---|
| SSRF on PATCH provider URLs | HIGH | `internal/api/vault_handlers.go` | `isHTTPSURL()` gate on `auth_url` + `token_url` |
| Filesystem path leaked in firstboot key response | MEDIUM | `internal/api/admin_bootstrap_handlers.go` | Removed `path` field from JSON |
| Audit-event gate bypassable on DB error | MEDIUM | `internal/api/admin_bootstrap_handlers.go` | Replaced w/ `ListAPIKeys()` fail-closed |

Pushed in commit `7a81df7`. LOW open-redirect (admin-set redirect URLs) filtered out — confidence < 8.

---

## Founder-Locked Exclusions (Do Not Re-enable)

- **Branding tab** → ComingSoon (not battle-tested)
- **Impersonate** → unwired (not battle-tested)
- **Device flow** → 501 + removed from `/.well-known/oauth-authorization-server` (not battle-tested)
- **Hosted pages** → not this version (devs supply their own redirect URLs)
- **PyPI publish** → not yet
- **F1 setup-token DB persistence** → W+1

---

## W+1 Backlog (Post-Launch)

- F1 setup-token DB persistence
- F4 re-run banner conftest integration
- F10 install.sh `sharkauth.dev` regex tighten
- Email-verify smoke
- Password-reset smoke
- Account-self-delete smoke
- Logout smoke
- Failed-login-lockout smoke
- 5 fixture ERROR tests (`w17_identity_settings`, `w2_sdk_method_8/9/10`)
- Proxy bug backlog (2 P0 + 4 P1 + 6 P2 — see `INVESTIGATE_REPORT.md`)

---

## Quick Rollback

If launch goes sideways:

```bash
git reset --hard 051f6e5   # last green pre-sprint commit
git push --force-with-lease origin main
```

(Pre-sprint commit `051f6e5` = "docs(f10): OpenAPI 3.1 spec + Scalar UI wiring".)

---

## Sign-Off

- [ ] Founder walks 11-item manual checklist
- [ ] `shark doctor` exits 0 on prod-like instance
- [ ] Demo command produces clean screencast
- [ ] README YC blurb final
- [ ] Launch tweet drafted
- [ ] Rollback path tested in staging
