# Subagent Dispatch Briefs — ready-to-paste

Copy any brief below into an Agent dispatch to run independently. Each assumes working dir `/home/raul/Desktop/projects/shark`, branch `claude/admin-vendor-assets-fix`, plan source `DASHBOARD_DX_EXECUTION_PLAN.md`, progress protocol per PROGRESS.md + STATUS.json conventions.

Shared boilerplate (prepend to each):

```
Working directory: /home/raul/Desktop/projects/shark
Branch: claude/admin-vendor-assets-fix
Read DASHBOARD_DX_EXECUTION_PLAN.md for context + T<N> spec before starting.
Respect the Execution Protocol section at the top of that file.

Commit convention: feat(dx): T<N> — <subject>  OR  fix(dx): T<N> — <subject>
Include Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com> footer.

Progress log: before committing, append a claim row. After commit, append a done row
with the commit sha. Update STATUS.json task entry. Do not touch other tasks' rows.

Typecheck gate: run `cd admin && npx tsc -b --noEmit` before each frontend commit.
Build gate: run `go build ./...` before each backend commit.

Push to origin claude/admin-vendor-assets-fix after all commits land.
```

---

## Brief — T05 (create-user slide-over)

**Depends on:** T04 landed (POST /admin/users backend).

**File scope:**
- admin/src/components/users.tsx (replace alert at line 65, add CreateUserSlideover component)
- No backend edits. No other frontend files.

**Acceptance:**
1. Remove `alert('Create user from the dashboard…')` at users.tsx:65
2. `usePageActions.onNew` opens `<CreateUserSlideover/>` instead
3. Slide-over fields: email (required, email validation), password (optional, masked, min 12 chars if provided, reveal eye icon), name (optional), roles (multiselect from `/roles`), orgs (multiselect from `/admin/organizations`), email_verified (checkbox, default false)
4. Submit → `API.post('/admin/users', {...})`. On 201, toast success, refresh list, auto-select new user's row
5. On 409 email_exists → highlight email field, toast error
6. On 400 invalid_request → inline error under first invalid field
7. `?new=1` query param auto-opens slide-over on mount (already wired via URLSearchParams)
8. ESC closes, clicking backdrop closes with confirm if dirty

**Subagent prompt:**
> Ship T05 create-user slide-over per DASHBOARD_DX_EXECUTION_PLAN.md T05 section. Backend POST /admin/users is live (T04 landed). Edit only admin/src/components/users.tsx. Replace the alert at line 65. Build CreateUserSlideover per acceptance above. Toast wiring uses existing useToast() import. Typecheck clean before commit. One atomic commit. Push. Log progress.

---

## Brief — T06 (org invitations slide-over)

**Depends on:** T05 landed (shared slide-over pattern).

**File scope:**
- admin/src/components/organizations.tsx (invitations tab — replace "use CLI" text)
- Consider extracting shared `<InviteSlideover/>` component if reuse warrants (optional)

**Acceptance:**
1. Invitations tab currently shows "use CLI" fallback. Replace with real UI.
2. "+ Invite" button opens slide-over with: email (required), role (select from org roles), expiry (segmented: 24h / 7d / 30d).
3. Submit → `API.post('/admin/organizations/{id}/invitations', {email, role_id, expiry_days})` (verify exact shape in admin_organization_handlers.go)
4. Pending invitations list shows real data from `GET /admin/organizations/{id}/invitations` (already wired per smoke 57b).
5. Each pending row has: email · role · expires in · resend button · revoke button.
6. Resend calls `POST /admin/organizations/{id}/invitations/{invitationId}/resend`, toast success.
7. Revoke calls `DELETE /admin/organizations/{id}/invitations/{invitationId}`, toast.undo with 5s undo window.

**Subagent prompt:**
> Ship T06 org invitations per DASHBOARD_DX_EXECUTION_PLAN.md T06. Edit only organizations.tsx. Backend routes verified present (router.go:586-588). Typecheck clean. One atomic commit. Push. Log progress.

---

## Brief — T15 (bootstrap token login)

**Depends on:** nothing (parallel-safe with everything except T11 if it touches login.tsx).

**File scope:**
- Backend: new internal/api/admin_bootstrap_handlers.go, router.go wire, cmd/shark/*.go startup print
- Frontend: admin/src/components/login.tsx (consume `?bootstrap=` param), maybe App.tsx (redirect)

**Acceptance:**
1. Startup detection: on `shark serve`, query audit log for any `admin.*` event. If zero, mint a cryptographic random 32-byte token, store hashed in-memory (or `bootstrap_tokens` table) with 10-min expiry.
2. Print to stdout (if TTY): `Open http://<host>:<port>/admin/?bootstrap=<tok>  (expires in 10 minutes)`
3. New route `POST /api/v1/admin/bootstrap/consume` body `{token}` — validates unused + unexpired, issues a one-time short-lived admin session (or mints a proper admin-key scoped to current user), marks token used.
4. Frontend login.tsx: if `URLSearchParams.has('bootstrap')`, auto-POST consume, on success call `onLogin(receivedKey)` and strip query param + redirect to `/admin/get-started` (if T11 landed) else `/admin/overview`.
5. Fallback: always-visible `? Where is my key?` link below password input, shows inline hint with `shark admin-key show` CLI command.
6. Smoke tests:
   - Token printed on fresh DB
   - POST consume with valid token → 200 + session
   - POST consume with used token → 409 / 400
   - POST consume with expired token → 401
   - No print on DB with prior admin actions

**Subagent prompt:**
> Ship T15 bootstrap token per DASHBOARD_DX_EXECUTION_PLAN.md T15. Backend + frontend scope. cmd/shark server startup + new admin_bootstrap_handlers.go + router wire + login.tsx frontend consume. Add smoke section. Two or three atomic commits (backend / frontend / smoke). Push. Log progress.

---

## Brief — T16 (ship-readiness checklist)

**Depends on:** T01, T02 (done). T09 hero tile considered — if the hero is present, the checklist can live BELOW the hero or in a collapsed state. Resolve on read.

**File scope:**
- admin/src/components/overview.tsx (add `<ShipReadinessCard/>` component below metric strip)
- admin/src/components/layout.tsx (topbar score badge, click opens modal)

**Acceptance:**
1. Card title: "Ready to ship: N/8"
2. 8 items with check state derived from existing endpoints:
   - SMTP configured (health.smtp.host != null && host != 'mock')
   - First app created (`/admin/apps` count > 0)
   - First user (`/admin/users` count > 0)
   - Signing key healthy (health.jwt.active_keys >= 1)
   - Webhook test fired (any delivery with attempt=1 success in last 30d — requires T20 endpoint OR fallback check: any webhook subscription exists)
   - Redirect whitelist set (any app has redirect_uri_allowlist entries)
   - Branding set (`adminConfig.branding?.logo_url` or similar — accept null as "skipped" with warn chip)
   - Audit reviewed (>0 audit rows in last 24h)
3. Each row clickable → navigates to relevant page (users, apps, signing, webhooks, etc.)
4. Topbar badge: score % (N/8 × 100). Green ≥75%, warn 50-74%, danger <50%.
5. Click badge → modal with same checklist.

**Subagent prompt:**
> Ship T16 ship-readiness checklist per DASHBOARD_DX_EXECUTION_PLAN.md T16. Overview adds card component below metrics (but above magical-moment hero if T09 active). Topbar adds score badge. Derive all 8 signals from existing endpoints. No new backend. Two commits (overview card + topbar badge). Push. Log progress.

---

## Brief — T21 (RBAC permission matrix)

**Depends on:** nothing (rbac.tsx is isolated).

**File scope:**
- admin/src/components/rbac.tsx — add `<PermissionMatrix/>` tab (adjacent to existing Roles list tab or as new 3rd tab)
- Backend: verify `/permissions` list endpoint + `POST /roles/{id}/permissions` + `DELETE /roles/{id}/permissions/{permission_id}` exist (or equivalent). If missing, ship them.

**Acceptance:**
1. New "Matrix" tab shows grid: rows = permissions (alphabetical), columns = roles.
2. Cell = toggle checkbox. Checked = role has permission. Click = optimistic toggle + API call.
3. Top of grid: permission search input (filters rows).
4. Right of each row: "Used by N users" count (from existing reverse lookup shipped in commit 6ffaa5c).
5. Bulk select: click column header = select all permissions for that role (dangerous, confirm).
6. Optimistic UI with rollback on error toast.

**Subagent prompt:**
> Ship T21 RBAC matrix per DASHBOARD_DX_EXECUTION_PLAN.md T21. Add Matrix tab to rbac.tsx with role × permission grid. Optimistic UI. Backend: verify permission CRUD routes, ship if missing. Typecheck + build gates. Push. Log progress.

---

## Dispatch sequencing (post-current-burst)

After Wave-β (A, B, C) lands:

1. **T05** — unblocked by T04. Frontend slide-over. Parallel-safe. ~1 agent.
2. **T06** — blocked by T05. Extract shared InviteSlideover if patterns diverge. Parallel-safe with T15/T16/T21. ~1 agent.
3. **T15 + T16 + T21** — three parallel agents, independent file scopes. Safe parallel dispatch.
4. **T24** — final ship review (rescore, smoke, eng review). Solo, after all others land.

Expected final state: 24/24 tasks done by Day 7 (2026-04-27).
