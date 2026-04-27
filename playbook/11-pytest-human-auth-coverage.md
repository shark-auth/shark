# Pytest Coverage Audit — Human Auth (2026-04-26)

> **2026-04-26 evening update:** Baseline now **282 PASS / 14 FAIL / 5 ERROR / 6 XFAIL / 37 SKIPPED** post-F1-F10 wave (+76 PASS vs morning). Test authoring conventions, failure categorization, and pre-launch checklist consolidated in `playbook/12-pytest-port-and-concurrency-plan.md` "EVENING UPDATE" section. See there for canonical lessons.

Smoke baseline (morning, retained for diff): **206 PASS / 12 FAIL / 5 ERROR / 33 SKIPPED**.
Audit scope: human-side auth (NOT agent-side; agent flows are exhaustively covered by W1-W3 tests).

---

## ✅ Covered (smoke present)

| Surface | Test file · function | Notes |
|---|---|---|
| Signup → login | `test_auth.py::test_signup_login_flow` | happy path |
| Session list + revoke | `test_user_sessions.py::test_session_list_and_revocation` | self + admin |
| Admin session filtering | `test_user_sessions.py::test_admin_session_filtering` | by user_id |
| Password change | `test_user_sessions.py::test_password_change` | logged-in flow |
| Magic-link issue + capture | `test_dev_email.py::test_dev_email_capture_via_magic_link` | dev inbox path |
| Dual-accept middleware | `test_auth.py::test_dual_accept_middleware` | session+bearer |
| Redirect allowlist | `test_auth.py::test_redirect_allowlist` | open-redirect prevention |
| Security headers | `test_auth.py::test_security_headers` | CSP, HSTS, etc. |
| Volume signup | `test_auth.py::test_massive_signup_volume` | rate-limit smoke |
| OAuth client_credentials + PKCE auth-code | `test_oauth_flows.py` | both grant types |
| OAuth advanced suite | `test_oauth_advanced.py` (9 tests) | PKCE enforce, refresh rotation, device flow, token-exchange, DPoP, introspect, revoke, DCR, resource indicators, JWKS |
| Cascade revoke (Layer 3) | `test_cascade_revoke.py` (8 tests) | full happy + edge |
| Disable agent revokes tokens | `test_w15_disable_revokes_tokens.py` | Bug A |
| Delete user revokes tokens+sessions | `test_w15_delete_user_revokes_tokens.py` | Bug B |
| Bulk-pattern revoke (Layer 4) | `test_bulk_pattern_revoke.py` (7 tests) | Wave 1.6 |
| Vault disconnect cascade (Layer 5) | `test_vault_disconnect_cascade.py` | Wave 1.6 |
| DPoP key rotation | `test_w2_sdk_method_10_rotate_dpop.py` | Wave 1.6 |

---

## ❌ GAPS — human-auth surfaces with NO smoke coverage

### High priority (risk pre-launch HN comments)

1. **Email verification flow** — `verify-email?token=...` clicked, status flips on user record. Backend handler exists at `internal/auth/handlers.go` (probably) but no smoke. Risk: a customer integration breaks if verification token logic regresses.

2. **Password reset flow** — `POST /auth/password-reset/request` → email captured → `POST /auth/password-reset/confirm` w/ token → login w/ new password. Multi-step, easy to break in refactors. NO smoke.

3. **Account self-deletion (DELETE /api/v1/me)** — referenced in settings.tsx danger zone but no smoke verifying user/sessions/agents all clear correctly. Compliance-relevant.

4. **Logout** — `POST /auth/logout` should destroy server-side session + invalidate cookie. NO direct smoke (only tested transitively via session revoke).

5. **Failed-login lockout** — `LockoutManager` configured for 5 attempts / 15 min in `router.go:142`. Behavior under repeated failed logins NOT smoke-tested. Risk: regression makes auth a brute-force target.

### Medium priority (W+1 acceptable)

6. **Passkey (WebAuthn) enroll + login** — `PasskeyManager` configured. No smoke for enrollment ceremony or login challenge. Hard to smoke (ceremony needs JS). Defer with note: tested manually before launch demo.

7. **MFA / TOTP enroll + challenge** — Settings page references MFA Enforcement (W1.7 E3 just shipped). Backend likely exists. No smoke for `/auth/mfa/enroll` + `/auth/mfa/verify`. Defer.

8. **SSO (SAML / OIDC IdP)** — Settings has SSO Connections section. `SSOHandlers` mounted in `router.go:180`. No smoke for IdP-initiated OR SP-initiated flow. Hard to smoke (needs a real IdP). Defer.

9. **Email change w/ re-verify** — Profile/Identity Hub allows email change. No smoke verifying old email gets revoke notification + new email gets verification link.

10. **Admin user CRUD full coverage** — `register_user` fixture in `test_cascade_revoke.py` exercises CREATE + DELETE. No standalone smoke for admin UPDATE (set role, freeze, demote). Risk: admin RBAC regression.

### Low priority (W+2 / nice-to-have)

11. **Rate-limit trigger asserts** — `RateLimit(100, 100)` middleware. Volume smoke hits it incidentally; no targeted assert that 429 returns when burst exceeded.

12. **CORS edge cases** — `CORSRelaxed` flag changes behavior. No smoke verifying disallowed origins get blocked when relaxed=false.

13. **Session expiry boundary** — sessionLifetime config. No smoke verifying token expiry actually 401s after window.

14. **Webhook delivery** — `WebhookDispatcher` async path. Failures would silently drop audit events. No smoke for fail-then-retry or DLQ behavior.

---

## Recommendation for launch

**Ship Monday with current coverage.** Items 1-5 (high priority) are operationally tested manually during dogfood. Add smoke for them W+1 alongside post-launch backlog.

**For YC application (Saturday):** items 1-3 should have smoke before submission — they're the questions YC technical partner will ask.

**Suggested W+1 smoke additions (ordered by leverage):**

```
tests/smoke/test_password_reset_flow.py        # 30 min, captures item 2
tests/smoke/test_email_verification_flow.py    # 20 min, captures item 1
tests/smoke/test_logout_destroys_session.py    # 15 min, captures item 4
tests/smoke/test_self_account_deletion.py      # 20 min, captures item 3
tests/smoke/test_failed_login_lockout.py       # 30 min, captures item 5
tests/smoke/test_admin_user_role_update.py     # 20 min, captures item 10
```

Total: ~2.5h CC. Ship Tuesday-Wednesday post-launch when scope budget recovers.
