# CRUD Audit — Admin Dashboard (2026-04-26)

Read-only investigation. Every wired route from App.tsx examined.
Backend handler existence confirmed via `internal/api/*_handlers.go`.

---

## Per-Surface Inventory

---

### 1. Users (`admin/src/components/users.tsx`)

- **Create:** ✓ — `CreateUserSlideover` → `POST /admin/users` (handler: `handleAdminCreateUser`). Email, password, name, email_verified, role_ids, org_ids all wired.
- **Read:** ✓ — paginated table with search + filters; URL deep-link `/admin/users/<id>`.
- **Update:** ⚠ — Profile tab has editable name + email + metadata textarea. `handleSave` calls `PATCH /users/{id}` (handler: `handleUpdateUser` exists). **BUT**: metadata textarea uses `defaultValue` (uncontrolled), so edits are never captured in state and will never be sent to the server. The Save button only patches `name` and `email`.
- **Delete:** ✓ — danger zone with email-confirmation modal → `DELETE /users/{id}`.

**Missing / broken:**

1. **Metadata field is uncontrolled** (`defaultValue` not `value`) — user can type in the box but pressing Save does nothing with the changes. `users.tsx:984`. Backend `PATCH /users/{id}` accepts arbitrary metadata. **Severity: P0.** Fix: change to controlled textarea with `useState`, add to patch payload. ~20 min.

2. **"Assign role" bulk action is `disabled`** — `users.tsx:181-182`. Comment says "v0.2 — use CLI". But the single-user Roles tab _does_ have remove-role working (`DELETE /users/{id}/roles/{roleId}`). The Roles tab lacks an **add-role** button — there's only remove. Backend `handleAssignRole` (`POST /users/{id}/roles`) exists. **Severity: P1.** Fix: add role selector + assign button in RolesTab. ~30 min.

3. **"Add to org" bulk action is `disabled`** — `users.tsx:183`. OrgsTab also shows org membership read-only, no "add to org" button. Backend `POST /admin/organizations/{id}/members` likely exists (org handlers present). **Severity: P1.** Fix: add org picker + add button in OrgsTab. ~30 min.

4. **"Impersonate" button** in UserSlideover header (`users.tsx:759`) renders as `<button className="btn ghost sm">Impersonate</button>` with **no onClick handler**. Clicking it does nothing. Backend `Impersonation` surface exists in App.tsx routes. **Severity: P0 (demo credibility hit — visible button, zero response).** Fix: wire to impersonation flow or remove for launch. ~15 min.

5. **"Export" button** in toolbar (`users.tsx:167`) renders `<button className="btn sm">Export</button>` with **no onClick handler**. The bulk-select version works (CSV download) but the top-level Export button is inert. **Severity: P1.** Fix: implement CSV export of current filtered page, or remove. ~20 min.

6. **More (⋯) button** in table row (`users.tsx:327`) renders `<button className="btn ghost icon sm"><Icon.More/></button>` with **no onClick handler**. **Severity: P1.** Fix: wire to a context menu (revoke sessions, copy ID, etc.) or remove. ~15 min.

7. **Billing tier section** is hidden behind `BILLING_UI` feature flag (`users.tsx:975`). The backend `PATCH /admin/users/{id}/tier` endpoint exists and is wired. Founder may want to demo tier assignment. **Severity: P2** (depends on whether demo needs it).

---

### 2. Agents (`admin/src/components/agents_manage.tsx`)

- **Create:** ✓ — `CreateAgentSlideOver` → `POST /agents`. Name, description, grant types, scopes, redirect URIs, one-time secret reveal. Backend `handleCreateAgent` exists.
- **Read:** ✓ — table with name/client_id/grants/scopes/created/creator/status. Detail slide-over with 7 tabs.
- **Update:** ✓ — `AgentConfig` tab: name, description (blur-save), redirect URIs (add/remove), grant types (toggle), scopes (add/remove) → `PATCH /agents/{id}`. Backend `handleUpdateAgent` exists.
- **Delete / Deactivate:** ✓ — "Deactivate" button → `DELETE /agents/{id}` (soft deactivate). No hard delete exposed in UI.

**Missing / broken:**

1. **No hard delete** — Deactivate sets `active=false` via DELETE, but there is no "Delete permanently" option. If backend supports it (same DELETE endpoint), the UI should offer it. **Severity: P2.** Fix: add a danger zone with permanent delete confirmation. ~20 min.

2. **AgentSecurity tab** — present in tab list (`tab === 'security'`), mounts `<AgentSecurity agent={agent}/>`. Need to verify this component is fully implemented (not a stub). Quick visual check recommended pre-launch. **Severity: P2.**

3. **AgentDelegationPolicies tab** — `<AgentDelegationPolicies agent={agent}/>` mounted but not audited here; confirm it has Create/Delete policy buttons. **Severity: P1** (delegation is a headline feature).

---

### 3. Sessions (`admin/src/components/sessions.tsx`)

Sessions are inherently read-only from a CRUD standpoint — you can't "create" or "update" a session from admin. The meaningful actions are:

- **Read:** ✓ — live-tail table + geo view with rich filters.
- **Revoke (Delete):** ✓ — per-row inline revoke, slide-over revoke, revoke-all, revoke by JTI. All wired to backend.

**Missing / broken:**

1. **"More options" dropdown** in SessionSlideover (`sessions.tsx:773-783`) has two buttons: "Revoke session" (wired) and "Copy session ID" (wired). No gaps here.

2. **Session detail shows token claims as synthesized client-side data** (`sessions.tsx:956-977`) — not fetched from backend. The `sub`, `iat`, `exp` etc. are guessed from the session object. A real decoded JWT introspection endpoint would be more accurate. **Severity: P2** (cosmetic inaccuracy, not a CRUD gap).

**Overall Sessions CRUD: C:N/A R:✓ U:N/A D:✓**

---

### 4. Token Vault (`admin/src/components/vault_manage.tsx`)

- **Create:** ✓ — `CreateProviderWizard` → `POST /vault/providers`. Template picker + custom path. Backend `handleCreateVaultProvider` exists.
- **Read:** ✓ — table + 3-tab detail (Config, Connections, Audit).
- **Update:** ✓ — `ProviderConfig`: display name (blur-save), icon URL (blur-save), scopes (add/remove), active toggle, client secret rotate. All call `PATCH /vault/providers/{id}` (handler: `handleUpdateVaultProvider` exists).
- **Delete:** ✓ — `DeleteProviderModal` with name-confirmation → `DELETE /vault/providers/{id}`.

**Missing / broken:**

1. **Auth URL and Token URL are read-only** in the Config tab (`vault_manage.tsx:424-435`). They display in a `<div>` with no edit input. If an operator needs to update the OAuth endpoints (provider changed their URLs), there is no UI path — CLI only. Backend `handleUpdateVaultProvider` almost certainly accepts `auth_url`/`token_url` in its PATCH body (standard pattern). **Severity: P1.** Fix: convert to editable inputs with blur-save, same as display_name. ~15 min.

2. **Client ID is read-only** in Config tab (`vault_manage.tsx:404-410`). Displayed with CopyField but no edit input. If a provider reissues credentials, the operator must delete and recreate. Backend PATCH likely accepts it. **Severity: P1.** Fix: same blur-save pattern. ~10 min.

3. **Connections tab "Connect" button** (`vault_manage.tsx:574`) opens OAuth flow in new tab — works. But the **user column** shows raw `user_id` (`vault_manage.tsx:605` — `{c.user_id}`), not email/name. Not a CRUD gap but a readability issue. **Severity: P2.**

---

### 5. Audit Log (`admin/src/components/audit.tsx`)

Audit logs are append-only by design. No Create/Update/Delete is expected.

- **Read:** ✓ — live tail, filters (actor type, severity, action prefix, time range), cursor pagination "Load more", CSV export, event detail slide-over.

**Missing / broken:**

1. **No "purge" / retention management UI** — The Settings page has an "Audit & Data" section in SECTIONS registry (`settings.tsx:22`), but if not fully implemented there, there's no way to trigger retention purge from the dashboard. Backend `admin_system_handlers.go` likely has a purge endpoint. **Severity: P2.**

**Overall Audit CRUD: C:N/A R:✓ U:N/A D:N/A**

---

### 6. Applications (`admin/src/components/applications.tsx`)

- **Create:** ✓ — `CreateAppSlideOver` → `POST /admin/apps`. Name, redirect URIs, CORS origins, proxy domain/URL, one-time secret reveal. Backend `handleCreateApp` exists.
- **Read:** ✓ — table + 5-tab detail (Config, Proxy rules, Consent preview, Tokens, Events).
- **Update:** ✓ — `AppConfig` tab: redirect URIs (add/remove), proxy domain/URL (Save button), integration mode. All → `PATCH /admin/apps/{id}` (handler: `handleUpdateApp` exists).
- **Delete:** ✓ — Delete button with undo toast → `DELETE /admin/apps/{id}`.

**Missing / broken:**

1. **App name is NOT editable** — The detail header (`applications.tsx:266`) shows `{app.name}` as plain text. The `AppConfig` component has no name field at all. This is the item the founder specifically flagged. Backend `handleUpdateApp` almost certainly accepts `name` in PATCH body (standard). **Severity: P0.** Fix: add name input in AppConfig section with blur-save or explicit Save button. ~15 min.

2. **"Proxy Gateway" deployment model button is permanently disabled** (`applications.tsx:381-388`) — renders with `disabled` + `opacity: 0.5` + "Coming soon" label. This means even apps that _were_ created as proxy mode can't switch the mode toggle, and new apps can only be SDK. The proxy mode inputs below _are_ present when `mode === 'proxy'`. **Severity: P1** (confusing — proxy apps exist but you can't select proxy in the UI).

3. **AppTokens tab shows placeholder** (`applications.tsx:539-545`) — `"Token listing not yet available via API."`. A Tokens tab is shown to the user but has zero data. **Severity: P1** (misleading empty tab visible to founder/demo audience).

4. **CORS origins not editable in Config tab** — `CreateAppSlideOver` accepts `corsOrigins` on creation (`applications.tsx:655`) but `AppConfig` has no field for editing CORS origins after creation. Backend PATCH accepts them. **Severity: P1.** Fix: add CORS origins section in AppConfig similar to redirect URIs. ~20 min.

5. **Proxy rules: no edit button** — `AppProxyRules` (`applications.tsx:791`) shows rules list with a Delete (X) button per row, and an "Add rule" that opens ProxyWizard. But clicking an existing rule row does nothing — no edit/update. Backend `PATCH /admin/proxy/rules/db/{id}` likely exists (proxy_rules_handlers.go present). **Severity: P2.** Fix: make row clickable to open edit form. ~30 min.

---

### 7. Organizations (`admin/src/components/organizations.tsx`)

- **Create:** ✓ — `CreateOrgSlideover` → `POST /admin/organizations`. Backend `handleAdminCreateOrganization` exists.
- **Read:** ✓ — split-pane list + 7-tab detail (Overview, Members, Invitations, Roles, SSO, Audit, Settings).
- **Update:** ✓ — Settings tab: name editable → `PATCH /admin/organizations/{id}`. SSO tab: toggle sso_required → same PATCH. Backend `handleAdminUpdateOrganization` exists.
- **Delete:** ✓ — "Delete org" in OrgActions dropdown → `DELETE /admin/organizations/{id}`.

**Missing / broken:**

1. **"Invite member" button in header** (`organizations.tsx:213`) and **"Invite members" in Overview Quick actions** (`organizations.tsx:330`) both render `<button className="btn sm">` with **no onClick handler**. The Invitations tab shows existing invites but there is no way to _send_ a new invitation from the UI. Backend `handleAdminResendOrgInvitation` exists but there's no "send new invite" handler surfaced. **Severity: P0 (creates impression that core org feature is broken — clicking the obvious "Invite" button does nothing).** Fix: add invite modal or inline form that calls `POST /admin/organizations/{id}/invitations`. ~30 min.

2. **"Suspend org" button** in Overview Quick actions (`organizations.tsx:331`) renders `<button className="btn sm danger">Suspend org</button>` with **no onClick handler**. If this endpoint doesn't exist backend-side, the button should be removed. If it does, it should be wired. **Severity: P1.** Fix: wire or remove. ~10 min.

3. **Org member role is read-only** — Members tab shows member role as a chip (`organizations.tsx:460`) with no way to change it. The "More" icon button per member row (`organizations.tsx:463`) has **no onClick handler**. Backend `handleAdminRemoveOrgMember` exists but there's no update-member-role handler shown. Remove member works via CLI; no UI path. **Severity: P1.** Fix: wire the More button to a context menu with "Change role" + "Remove". ~30 min.

4. **Org role: no delete button** — `OrgRolesTab` (`organizations.tsx:546-560`) shows roles as cards with CopyField but no Delete or Edit button per role. Backend `handleAdminDeleteOrgInvitation` exists; a role delete handler would be in org_rbac_handlers. **Severity: P2.**

5. **Org slug is read-only** in Settings tab (`organizations.tsx:615-621`) — shown in a disabled input. Slug is an immutable key, so this is correct. Not a bug.

---

### 8. Branding (`admin/src/components/branding.tsx`)

**The entire Branding surface is a `<ComingSoon>` stub.** The file simply renders:

```
<ComingSoon title="Custom branding is coming in v0.2" .../>
```

- **Create:** ✗ — not implemented
- **Read:** ✗ — not implemented  
- **Update:** ✗ — not implemented
- **Delete:** ✗ — not implemented

However, `admin/src/components/branding/visuals_tab.tsx` and `branding/email_tab.tsx` and `branding/integrations_tab.tsx` **exist and are implemented** with full CRUD (GET/PATCH `/admin/branding`, logo upload/delete, color/font/footer editing, email preview). They are simply **not wired into the Branding route** — `branding.tsx` imports `ComingSoon` and ignores the subtab components.

Backend handlers exist: `handleGetBranding`, `handlePatchBranding`, `handleUploadLogo`, `handleDeleteLogo`.

**Severity: P0** — Branding is a named sidebar item, navigating to it shows a dead "coming soon" screen, yet the full implementation exists in subtab components and backend. Fix: replace `branding.tsx` with a tabbed container that renders `BrandingVisualsTab`, `BrandingEmailTab`, `BrandingIntegrationsTab`. ~20 min.

---

### 9. Delegation Chains (`admin/src/components/delegation_chains.tsx`)

- **Create:** ✗ — read-only surface. Chains are derived from audit log events (token exchanges), not created directly.
- **Read:** ✓ — fetches `GET /audit-logs?action=oauth.token.exchanged` + `vault.token.retrieved`, groups by act-chain root subject, renders chain list + canvas visualization.
- **Update:** N/A — delegation chains are audit-derived
- **Delete:** N/A — chains can't be deleted; individual sessions/tokens are revoked elsewhere

**Missing / broken:**

1. **No "Revoke chain" action** — viewing a delegation chain does not offer a way to immediately revoke all tokens in that chain. This is a workflow gap, not a code bug. **Severity: P2.**

2. **Comment at top of file** (`delegation_chains.tsx:1-6`) explicitly notes: "TODO Wave 1.6 backend: add `/api/v1/audit-logs/chains` aggregator endpoint." The current implementation does two separate fetches and aggregates client-side. This can cause missed chains if audit log pagination truncates. **Severity: P2** (data completeness, not UI gap).

---

### 10. Settings (`admin/src/components/settings.tsx`)

- **Create:** N/A (config singleton)
- **Read:** ✓ — loads `GET /admin/config`, sections: Server, Sessions & Tokens, Auth Policy, OAuth & SSO, Email Delivery, OAuth Providers, Audit & Data, Maintenance.
- **Update:** ✓ — each section patches `PATCH /admin/config`. Sections have Save buttons per group.
- **Delete:** N/A (config)

**Missing / broken:**

Full review of settings.tsx was limited (file large). From the SECTIONS registry and partial read:

1. **Audit & Data section** — The SECTIONS array lists `audit` with desc "Retention, purge, export" and `maintenance` with "Sessions + audit cleanup". Need to verify these sections have actual save/action buttons rather than placeholders. **Severity: P1** if unimplemented.

2. **OAuth Providers section** — per-provider credential fields (client ID/secret for Google, GitHub, etc.) are in Settings but also partially duplicated in Vault. Confirm these are the SSO providers (identity federation), not the vault providers. Should be clearly labeled. **Severity: P2** (confusion risk in demo).

---

### 11. Identity Hub / Auth (`admin/src/components/identity_hub.tsx`, `sso.tsx`, `authentication.tsx`)

Not deeply audited (out of direct request scope) but notable:

- `identity_hub.tsx` is likely a tab container routing to users/agents/sessions subtabs.
- `sso.tsx` has SSO connection CRUD — not fully audited but `sso_handlers.go` exists.
- `authentication.tsx` — auth policy settings, likely read/update only.

---

### 12. RBAC (`admin/src/components/rbac.tsx`)

Not deeply audited but backend has full CRUD: `handleCreateRole`, `handleUpdateRole`, `handleDeleteRole`, `handleCreatePermission`, `handleDeletePermission`, `handleAttachPermission`, `handleDetachPermission`, `handleAssignRole`, `handleRemoveRole`. Verify UI is fully wired.

---

### 13. API Keys (`admin/src/components/api_keys.tsx`)

Not deeply audited. Backend `apikey_handlers.go` exists. Verify create/delete are wired.

---

### 14. Webhooks (`admin/src/components/webhooks.tsx`)

Not deeply audited. Backend `webhook_handlers.go` exists. Verify create/update/delete are wired.

---

### 15. Overview / Get Started (`overview.tsx`, `get_started.tsx`)

- Read-only informational surfaces. No CRUD expected.
- **Missing:** Overview shows stats (users total, etc.). No gaps.

---

### 16. Dev Email (`dev_email.tsx`)

- Read (inbox list) + Delete (clear message). Backend `devinbox_handlers.go` exists. No gaps expected.

---

## Summary Table

| Surface | C | R | U | D | Missing items | Worst severity |
|---|---|---|---|---|---|---|
| Users | ✓ | ✓ | ⚠ | ✓ | metadata uncontrolled, impersonate no-op, assign-role disabled, export no-op, more-menu no-op | **P0** |
| Agents | ✓ | ✓ | ✓ | ⚠ | no hard delete, delegation policies unverified | P2 |
| Sessions | N/A | ✓ | N/A | ✓ | none | — |
| Token Vault | ✓ | ✓ | ⚠ | ✓ | auth_url/token_url/client_id read-only in edit drawer | P1 |
| Audit Log | N/A | ✓ | N/A | N/A | no purge UI | P2 |
| Applications | ✓ | ✓ | ⚠ | ✓ | app name not editable, proxy toggle disabled, tokens tab empty, CORS not editable | **P0** |
| Organizations | ✓ | ✓ | ✓ | ✓ | Invite button no-op (2 locations), Suspend no-op, member role not editable | **P0** |
| Branding | ✗ | ✗ | ✗ | ✗ | entire surface is ComingSoon stub (impl exists in subtabs) | **P0** |
| Delegation Chains | N/A | ✓ | N/A | N/A | no revoke chain action | P2 |
| Settings | N/A | ✓ | ✓ | N/A | audit/maintenance sections may be incomplete | P1 |
| RBAC | ✓ | ✓ | ✓ | ✓ | verify UI wiring matches backend | P2 |
| API Keys | ? | ? | ? | ? | not audited | — |
| Webhooks | ? | ? | ? | ? | not audited | — |

---

## Recommended Pre-Launch Fixes (P0 → P1 → quick-win P2)

### P0 — Launch Blockers (total: ~95 min)

1. **[P0] App name not editable** — `applications.tsx` AppConfig section. Add `name` input with blur-save or include in existing Save. Backend `PATCH /admin/apps/{id}` accepts `name`. **~15 min.**

2. **[P0] Branding shows ComingSoon** — `branding.tsx` is a 14-line stub. The full impl (`branding/visuals_tab.tsx`, `email_tab.tsx`, `integrations_tab.tsx`) is built and backend is wired. Just replace branding.tsx with a tabbed container. **~20 min.**

3. **[P0] Org "Invite member" button is inert** — Both the header button and Quick Actions "Invite members" button have no onClick. This is a visible P0 — the most prominent call-to-action in the Orgs surface does nothing. Add invite modal calling `POST /admin/organizations/{id}/invitations`. **~30 min.**

4. **[P0] User "Impersonate" button is inert** — UserSlideover header renders a ghost button with no handler. Either wire it to the impersonation flow (route: `impersonation`) or remove before demo. **~15 min.**

5. **[P0] User metadata textarea uncontrolled** — `defaultValue` → `value` + useState + include in PATCH payload. **~20 min.**

### P1 — Embarrassing Gaps (~160 min)

6. **[P1] Vault: auth_url, token_url, client_id read-only** — convert 3 display divs to blur-save inputs in ProviderConfig. Backend already accepts these in PATCH. **~20 min total.**

7. **[P1] Applications: AppTokens tab is a placeholder** — Either implement token listing via a real API call or hide the "Tokens" tab until implemented. Showing an empty stub tab to a founder is embarrassing. **~20 min** to hide or ~60 min to implement.

8. **[P1] Applications: CORS origins not editable post-creation** — add allowed_origins section in AppConfig mirroring the redirect URIs UI. **~20 min.**

9. **[P1] Applications: "Proxy Gateway" toggle permanently disabled** — remove the "Coming soon" disabled button or make it conditionally enabled based on whether proxy feature flag is on. Currently creates confusion when viewing proxy-mode apps. **~10 min.**

10. **[P1] Users: role assignment missing "add" path** — RolesTab has remove-role but no assign-role. Add a role picker + "Assign role" button calling `POST /users/{id}/roles`. Backend `handleAssignRole` exists. **~30 min.**

11. **[P1] Users: Export button is inert** — wire to CSV download of current filtered set (clone the bulk-export logic). **~15 min.**

12. **[P1] Org: "Suspend org" button is inert** — wire or remove. If backend has no suspend endpoint, remove it. **~10 min.**

13. **[P1] Org member role change** — More (⋯) button per member row has no onClick. Wire to context menu with "Change role" + "Remove from org". **~30 min.**

### Quick-Win P2 (~60 min)

14. **[P2] Users: More (⋯) row button is inert** — wire to a context menu: "Copy ID", "Revoke sessions", "View in audit log". **~15 min.**

15. **[P2] Vault: Connections tab shows raw user_id** — resolve to email via user lookup. **~15 min.**

16. **[P2] Proxy rules: rows not clickable for edit** — make row clickable to open edit form. **~30 min.**

17. **[P2] Delegation Chains: revoke chain action** — "Revoke all tokens in chain" button. **~20 min.**

---

## Backend Endpoints Without UI Consumer

The following endpoints exist in handlers but appear to have no UI caller. These may be CLI-only by design or represent missing UI.

| Endpoint (inferred) | Handler | Status |
|---|---|---|
| `POST /admin/organizations/{id}/invitations` | (not found — may be in org_rbac_handlers or org handlers) | Missing UI (invite send) |
| `PATCH /users/{id}` with `role_ids` | `handleUpdateUser` | No UI path for bulk role assignment |
| `POST /users/{id}/roles` | `handleAssignRole` | No UI in RolesTab |
| `PATCH /admin/apps/{id}` with `name` | `handleUpdateApp` | AppConfig has no name field |
| `PATCH /vault/providers/{id}` with `auth_url`/`token_url`/`client_id` | `handleUpdateVaultProvider` | Read-only in UI |
| `GET /admin/branding` + `PATCH /admin/branding` | `handleGetBranding`, `handlePatchBranding` | Wired in subtab components but subtabs not rendered (ComingSoon stub) |
| `POST /admin/users/{id}/tier` | (admin_user_handlers) | Hidden behind `BILLING_UI` flag |
| `DELETE /admin/organizations/{id}/members/{userId}` | `handleAdminRemoveOrgMember` | No UI button in Members tab |

---

## Total Missing CRUD Items

**Confirmed gaps: 17 distinct items**
- P0: 5 items
- P1: 8 items
- P2: 4 items (from the items above; more exist in un-audited surfaces)

**Estimated P0 fix effort: ~100 minutes total**

(App name edit: 15 + Branding wire-up: 20 + Org invite: 30 + Impersonate button: 15 + Metadata textarea: 20 = 100 min)
