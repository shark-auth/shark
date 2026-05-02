# MFA

TOTP enrollment, verification, login challenge, and recovery codes. Wraps `/api/v1/auth/mfa/*`.

The Python SDK ships a stdlib-only `compute_totp(secret)` helper so you can drive enroll → verify end-to-end without `pyotp`. The TypeScript SDK ships the equivalent `computeTotp(secret)`.

## Setup

The MFA client expects an authenticated session. Build it on top of an already-logged-in `AuthClient`.

```python
from shark_auth import AuthClient, MFAClient, compute_totp

auth = AuthClient("https://auth.example.com")
auth.login("alice@example.com", "Strong-Pwd-2026")
mfa = MFAClient("https://auth.example.com", session=auth._session)
```

```typescript
import { AuthClient, MfaClient, computeTotp } from "@sharkauth/sdk";

const auth = new AuthClient("https://auth.example.com");
await auth.login("alice@example.com", "Strong-Pwd-2026");
const mfa = new MfaClient("https://auth.example.com", { session: auth.session });
```

## Enroll

Returns a fresh secret plus an `otpauth://` provisioning URI (also exposed as `qr_uri`).

```python
e = mfa.enroll()
# {"secret": "JBSWY3DPEHPK3PXP", "qr_uri": "otpauth://totp/...", "otpauth_url": "..."}
print(e["secret"])
```

```typescript
const e = await mfa.enroll();
console.log(e.secret);
```

Show the QR code to the user via the `qr_uri`. Call `verify` with the first valid code to confirm enrollment.

## Verify (confirm enrollment)

```python
result = mfa.verify(compute_totp(e["secret"]))
# {"mfa_enabled": true, "recovery_codes": ["...", "...", ...]}
print(result["recovery_codes"])
```

```typescript
const result = await mfa.verify(computeTotp(e.secret));
```

Recovery codes are shown exactly once. Persist them on the user's side immediately.

## Challenge (on login)

When `login` returns `{"mfaRequired": true}`, the session is partial. Submit a TOTP code to upgrade.

```python
auth.login("alice@example.com", "Strong-Pwd-2026")
mfa.challenge(code="123456")
me = auth.get_me()  # now succeeds
```

```typescript
await auth.login("alice@example.com", "Strong-Pwd-2026");
await mfa.challenge("123456");
const me = await auth.getMe();
```

## Disable

```python
mfa.disable(current_code="123456")
```

```typescript
await mfa.disable("123456");
```

Wraps `DELETE /api/v1/auth/mfa` (the SDK adapts the unusual verb-with-body shape).

## Regenerate recovery codes

```python
new_codes = mfa.regenerate_recovery_codes()
```

```typescript
const newCodes = await mfa.regenerateRecoveryCodes();
```

Wraps `GET /recovery-codes` (yes, GET — the SDK matches the backend's chosen verb).

## TOTP helper

```python
from shark_auth import compute_totp
code = compute_totp("JBSWY3DPEHPK3PXP")  # 6 digits, 30s window
```

```typescript
import { computeTotp } from "@sharkauth/sdk";
const code = await computeTotp("JBSWY3DPEHPK3PXP");
```

RFC 6238, HMAC-SHA1, 30-second step. The TS implementation falls back to Node `crypto.createHmac` when `crypto.subtle` does not expose SHA-1 (some browsers).

## Admin: reset MFA for a locked-out user

If a user loses their authenticator, an admin can clear their MFA without requiring the current code:

```python
c.users.reset_mfa(user_id="usr_abc")
```

```typescript
await c.users.resetUserMfa("usr_abc");
```

## Limitations (pre-launch)

- `acr` / `amr` claim emission absent from token-exchange path. Real MFA step-up enforcement on agent tokens is not yet wired (backend gap).
- `sessions.mfa_at` freshness column missing — `mfa_passed` is a sticky bool only.
- Per-vault `requires_mfa` flag absent — no per-action step-up gating yet.

These are server-side gaps documented in `sdk/HANDOFF.md`. The SDK methods will start enforcing step-up automatically once the server emits the claims.
