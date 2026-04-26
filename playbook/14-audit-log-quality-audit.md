# Audit Log Quality Audit — 2026-04-26

## Methodology

Grepped `internal/` for all eight emission patterns (`s.AuditLogger.Log(`, `&storage.AuditLog{`, `WriteAuditEvent`, `audit.Log(`, etc.). Read each hit in context. Classified by three tiers: GOOD (Action + ActorID + ActorType + TargetID + ≥3 metadata fields), MEDIUM (Action + ActorType but thin/missing metadata or no ActorID), BAD (no ActorID, no metadata, or generic/untraceable action). Cross-checked against `admin/src/components/audit.tsx` for frontend drift.

---

## Inventory

| File:line | Action | ActorID | ActorType | TargetID | Metadata fields | Tier |
|---|---|---|---|---|---|---|
| internal/api/admin_bootstrap_handlers.go:191 | `admin.bootstrap.consumed` | yes (ak.ID) | admin | ak.ID | — (none) | MEDIUM |
| internal/api/admin_system_handlers.go:488 | `admin.mfa.disabled` | **no** | admin | userID | — (none) | BAD |
| internal/api/admin_system_handlers.go:738 | `admin.config.updated` | **no** | admin | **no** | — (none) | BAD |
| internal/api/admin_organization_handlers.go:455 | `admin.organization.*` (6 actions) | **no** | admin | orgID | — (none) | BAD |
| internal/api/admin_oauth_handlers.go:122 | `oauth.device.<decision>` | **no** | admin | userCode | `{user_code, client_id, user_id, decision}` | MEDIUM |
| internal/api/admin_oauth_handlers.go:171 | `oauth.bulk_revoke_pattern` | yes (actor) | admin | **no** | `{pattern, revoked_count, reason}` | MEDIUM |
| internal/api/application_handlers.go:201 | `app.create/update/delete/secret.rotate` | **no** | admin | app.ID | — (none) | BAD |
| internal/api/agent_handlers.go:46 | `agent.created/updated/deactivated/tokens_revoked/secret.rotated` | **no** | admin | agent.ID | — (none) | BAD |
| internal/api/agent_handlers.go:63 | `agent.deactivated_with_revocation`, `agent.dpop_key_rotated` | **no** | admin | agent.ID | varies per call site | MEDIUM |
| internal/api/apikey_handlers.go:151 | `api_key.created` | **no** | admin | apiKey.ID | — (none) | BAD |
| internal/api/apikey_handlers.go:348 | `api_key.revoked` | **no** | admin | key.ID | — (none) | BAD |
| internal/api/apikey_handlers.go:434 | `api_key.rotated` | **no** | admin | newKey.ID | — (none) | BAD |
| internal/api/admin_user_handlers.go:103 | `admin.user.create` | **no** | admin | user.ID | `{email}` | MEDIUM |
| internal/api/auth_handlers.go:594 | `user.login` (failure path) | yes (actorID) | user | actorID | `{email}` | MEDIUM |
| internal/api/consent_handlers.go:106 | `consent.revoked` (user) | yes (userID) | user | consentID | `{consent_id, client_id, user_id}` | GOOD |
| internal/api/consent_handlers.go:210 | `consent.revoked` (admin) | **no** | admin | consentID | `{consent_id, client_id, user_id}` | MEDIUM |
| internal/api/email_verify_handlers.go:208 | `admin.user.verification_sent` | **no** | admin | user.ID | — (none) | BAD |
| internal/api/organization_handlers.go:544 | `organization.create/delete/invitation.*`, `org.member.role_update` | yes (actor) | user | orgID | — (none) | MEDIUM |
| internal/api/org_rbac_handlers.go:73 | `org.role.*`, `org.permission.*` | yes (actor) | user | roleID | — (none) | MEDIUM |
| internal/api/proxy_admin_v15_handlers.go:67 | `user.tier.set` | **no** | admin | userID | `{tier}` (1 field) | MEDIUM |
| internal/api/proxy_admin_v15_handlers.go:115 | `branding.design_tokens.set` | **no** | admin | "global" | — (none) | BAD |
| internal/api/rbac_handlers.go:22 | `rbac.role.create/assign/revoke`, `rbac.permission.attach/detach` | **no** | **service** | roleID/userID | — (none) | BAD |
| internal/api/proxy_rules_handlers.go:193 | `proxy.rule.created` | **no** | admin | rule.ID | — (none) | BAD |
| internal/api/proxy_rules_handlers.go:334 | `proxy.rule.updated` | **no** | admin | rule.ID | — (none) | BAD |
| internal/api/proxy_rules_handlers.go:380 | `proxy.rule.deleted` | **no** | admin | rule.ID | — (none) | BAD |
| internal/api/session_handlers.go:280 | `session.revoke` | yes (actorID) | user/admin | sessionID | `{session_id, target_user_id}` | GOOD |
| internal/api/user_handlers.go:182 | `user.deleted_with_token_revocation` | **no** | admin | user.ID | `{revoked_token_count, revoked_session_count}` | MEDIUM |
| internal/api/vault_handlers.go:132 | vault ops (read/write/delete/list) | yes (from ctx) | varies | targetID | varies per call | GOOD |
| internal/api/user_agents_handlers.go:165 | `<cascade_revoke_agents>` | **no** | admin | targetUserID | `{agent_count, reason, by_actor}` | MEDIUM |
| internal/api/system_handlers.go:193 | `admin.key.rotated` | **no** | admin | **no** | — (none) | BAD |
| internal/api/system_handlers.go:278 | `admin.db.reset` | **no** | admin | mode | — (none) | BAD |
| internal/authflow/steps.go:416 | auth flow step actions | yes (user.ID) | user | **no** | varies | MEDIUM |
| internal/oauth/dcr.go:267 | DCR register/update/delete | **no** | client | clientID | `{}` (hardcoded empty) | BAD |

**Total production emission sites: 33**

---

## BAD-tier events (highest fix priority)

### `admin.mfa.disabled` — internal/api/admin_system_handlers.go:488
- Currently: no ActorID, no Metadata
- Should add: `{admin_key_id, mfa_method, reason}`, populate ActorID from request context

### `admin.config.updated` — internal/api/admin_system_handlers.go:738
- Currently: no ActorID, no TargetID, empty Metadata
- Should add: `{key, old_value, new_value, redacted_keys, request_id}`, ActorID from auth context

### `admin.organization.*` (6 call sites) — internal/api/admin_organization_handlers.go
- Currently: no ActorID, no Metadata across create/update/delete/role-create/invite-delete/invite-resend/member-remove
- Should add per action: `{org_name, org_id}` for create; `{changed_fields}` for update; `{member_id, role}` for member ops

### `app.create/update/delete/secret.rotate` — internal/api/application_handlers.go:201
- Currently: no ActorID, no Metadata
- Should add: `{app_name, client_id, redirect_uris_count}` for create; `{changed_fields}` for update; `{rotated_by}` for secret.rotate

### `agent.created/updated/deactivated/tokens_revoked/secret.rotated` — internal/api/agent_handlers.go:46
- Currently: no ActorID across all simple agent events
- Should add: ActorID from auth context; `{agent_name, client_id}` for created; `{revoked_token_count}` for tokens_revoked; `{old_kid, new_kid}` for secret.rotated

### `api_key.created/revoked/rotated` — internal/api/apikey_handlers.go:151/348/434
- Currently: no ActorID, no Metadata on all three
- Should add: `{key_name, scopes, expires_at}` for created; `{key_name, revoked_by}` for revoked; `{old_key_id, new_key_id, scopes}` for rotated

### `admin.user.verification_sent` — internal/api/email_verify_handlers.go:208
- Currently: no ActorID, no Metadata
- Should add: `{email, template_id}`, ActorID from auth context

### `branding.design_tokens.set` — internal/api/proxy_admin_v15_handlers.go:115
- Currently: no ActorID, no Metadata
- Should add: `{token_keys_changed, by_admin_key}`, ActorID

### `rbac.role.create/assign/revoke`, `rbac.permission.attach/detach` — internal/api/rbac_handlers.go:22
- Currently: ActorType is hardcoded `"service"` (wrong — these are admin actions), no ActorID, no Metadata
- Should add: ActorType `"admin"`, ActorID from auth context; `{role_name, permission_name, user_id}` per action

### `proxy.rule.created/updated/deleted` — internal/api/proxy_rules_handlers.go:193/334/380
- Currently: no ActorID, no Metadata
- Should add: `{rule_name, path_pattern, target_upstream, changed_fields}` for updated; ActorID

### `admin.key.rotated` — internal/api/system_handlers.go:193
- Currently: no ActorID, no TargetID, no Metadata
- Should add: `{key_prefix_old, key_prefix_new}`, ActorID

### `admin.db.reset` — internal/api/system_handlers.go:278
- Currently: no ActorID, no Metadata
- Should add: `{mode, tables_affected, triggered_by}`, ActorID

### DCR operations — internal/oauth/dcr.go:267
- Currently: no ActorID, hardcoded `Metadata: "{}"`, ActorType `"client"` is ambiguous
- Should add: `{client_name, grant_types, redirect_uris_count, registration_access_token_prefix}`

---

## MEDIUM-tier events

| Event | Gap | Should add |
|---|---|---|
| `admin.bootstrap.consumed` (:191) | No Metadata | `{key_prefix, consumed_at}` |
| `oauth.device.<decision>` (:122) | No ActorID | admin key ID from context |
| `oauth.bulk_revoke_pattern` (:171) | No TargetID | pattern as TargetID |
| `admin.user.create` (:103) | No ActorID | admin key ID; add `{role, email_verified}` |
| `user.login` failure (:594) | Only `{email}` | add `{failure_reason, attempt_count}` |
| `consent.revoked` admin (:210) | No ActorID | admin key ID |
| `organization.*` user-path (:544) | No Metadata | `{org_name, member_id, role_change}` per action |
| `org.role.*` / `org.permission.*` (:73) | No Metadata | `{role_name, permission_name, org_id}` |
| `user.tier.set` (:67) | No ActorID, 1-field meta | ActorID; add `{old_tier, changed_by}` |
| `user.deleted_with_token_revocation` (:182) | No ActorID | admin key ID from context |
| `user_agents cascade` (:165) | No ActorID | admin key ID |
| `authflow steps` (:416) | No TargetID | session_id or flow_id as TargetID |
| `agent.deactivated_with_revocation` (:63) | No ActorID | admin key ID |

---

## Frontend drift

`audit.tsx` renders these columns: **When, Action, Actor (actor_type + actor_id), Target, IP, Event ID**.

The `EventDetail` drawer synthesizes `request_id`, `session_id`, `jti`, `oauth.{client_id, scope, grant_type, dpop, act}`, and `diff.{old_kid, new_kid, reason}` from `event._raw` — **but falls back to `Math.random()` mock values** when those fields are absent from the raw API payload (lines 496–510). This means:

- `request_id` is rendered in every drawer but **never populated by any backend emission site**
- `session_id` is rendered for user-actor events but **never populated**
- `oauth.*` block renders for `oauth.token.*` events but backend `authflow/steps.go` emits those with no oauth sub-object in metadata
- `act_chain` delegation breadcrumb (line 556) reads `_raw.act_chain` — **no backend site populates `act_chain`**
- `diff.{old_kid, new_kid}` is rendered for `agent.secret.rotated` but `auditAgent` emits with **no metadata at all** for that action

---

## Suggested smoke tests (post-fix)

For each BAD/MEDIUM after fix, assert:

- `tests/smoke/test_audit_quality_config.py` — POST config change, fetch audit log, assert `metadata` contains `key`, `old_value`, `new_value`
- `tests/smoke/test_audit_quality_rbac.py` — create/assign role, assert `actor_id` non-empty, metadata contains `role_name`
- `tests/smoke/test_audit_quality_agent.py` — create/deactivate agent, assert `actor_id` set, metadata contains `agent_name`, `client_id`
- `tests/smoke/test_audit_quality_proxy_rules.py` — create/update/delete proxy rule, assert metadata contains `rule_name`, `path_pattern`
- `tests/smoke/test_audit_quality_apikeys.py` — create/revoke/rotate API key, assert `actor_id` set, metadata contains `key_name`, `scopes`
- `tests/smoke/test_audit_quality_dcr.py` — DCR register, assert metadata not `{}`, contains `client_name`
