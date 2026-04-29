# Authentication

Wraps `/api/v1/auth/*`. Cookie-based sessions plus optional bearer JWT. No admin key required.

The Python `AuthClient` carries a `requests.Session`; the cookie planted by `login` / `signup` flows through subsequent calls. The TypeScript `AuthClient` ships an internal cookie shim for Node and uses `credentials: "include"` in the browser.

## Construct

```python
from shark_auth import AuthClient

auth = AuthClient("https://auth.example.com")
```

```typescript
import { AuthClient } from "@sharkauth/sdk";

const auth = new AuthClient("https://auth.example.com");
```

## Signup

Creates a user. On 200/201 the server plants the session cookie.

```python
user = auth.signup(
    email="alice@example.com",
    password="Strong-Pwd-2026",
    full_name="Alice",       # mapped to server's `name`
)
print(user["id"])
```

```typescript
const user = await auth.signup("alice@example.com", "Strong-Pwd-2026", { name: "Alice" });
```

| Param      | Type   | Required | Notes                                |
| ---------- | ------ | -------- | ------------------------------------ |
| `email`    | string | yes      | Lowercased server-side               |
| `password` | string | yes      | Server enforces complexity policy    |
| `name`     | string | no       | `full_name` in Python; `name` in TS  |

Returns the new user object: `{ id, email, name?, email_verified?, created_at, updated_at, ... }`.

## Login

```python
result = auth.login("alice@example.com", "Strong-Pwd-2026")
# result["mfaRequired"] is True when MFA challenge is needed
```

```typescript
const result = await auth.login("alice@example.com", "Strong-Pwd-2026");
if (result.mfaRequired) {
  // see ./mfa.md — call mfa.challenge(code) to upgrade the session
}
```

When MFA is enrolled, the server returns `{"mfaRequired": true}` instead of the user. Submit the TOTP code via [MFAClient.challenge()](./mfa.md#challenge-on-login) to upgrade the partial session.

## Logout

```python
auth.logout()
```

```typescript
await auth.logout();
```

204 on success.

## Get / delete /me

```python
me = auth.get_me()
auth.delete_me()   # requires verified email
```

```typescript
const me = await auth.getMe();
await auth.deleteMe();
```

## Password management

```python
# Change while authenticated.
auth.change_password(old="...", new="...")

# Forgot-password flow.
auth.request_password_reset("alice@example.com")
auth.confirm_password_reset(token=token_from_email, new_password="...")
```

```typescript
await auth.changePassword("old_password", "new_password");
await auth.requestPasswordReset("alice@example.com");
await auth.confirmPasswordReset("token_from_email", "new_password");
```

`request_password_reset` always returns success — the server intentionally does not leak whether the email exists.

## Email verification

```python
auth.request_email_verification()        # send link
auth.consume_email_verification(token)   # consume link
```

```typescript
await auth.requestEmailVerification();
await auth.consumeEmailVerification(token);
```

`request_email_verification` requires an authenticated session. The link's redirect URI is configured server-side (see [Email config](#email-config)).

## Magic link

Two-call flow: send, then verify.

```python
from shark_auth import MagicLinkClient

ml = MagicLinkClient("https://auth.example.com")
ml.send_magic_link("alice@example.com", redirect_uri="https://app.example.com/cb")
# ... user clicks link ... your callback handler receives ?token=...
user = ml.verify(token)
# OR, equivalently, on the same client carrying the cookie afterward:
user = auth.verify_magic_link(token)
```

```typescript
import { MagicLinkClient } from "@sharkauth/sdk";

const ml = new MagicLinkClient({ baseUrl: "https://auth.example.com" });
await ml.sendMagicLink("alice@example.com", { redirectUri: "https://app.example.com/cb" });
const user = await auth.verifyMagicLink(token);
```

Server applies per-email rate limiting (1 per 60s) and never reveals account existence.

## Permission check

```python
result = auth.check(action="read", resource="documents:123")
# {"allowed": true}
```

```typescript
const result = await auth.check("read", "documents:123");
```

Wraps `POST /api/v1/auth/check`. Requires a valid session or bearer token.

## Self-revoke

```python
auth.revoke_self()
```

```typescript
await auth.revokeSelf();
```

Wraps `POST /api/v1/auth/revoke`. The user kills their own session/JWT.

## Error handling

All methods raise `SharkAuthError` (Python) or `SharkAPIError` (TypeScript) on non-2xx. Body fields `error` and `message` are surfaced in the exception message. See [Errors and retries](./errors-and-retries.md).

## Email config

Magic-link, email-verify, password-reset, and invitation links all share a single redirect-URI allowlist configured server-side (`shark.yaml` `email.redirect_allowlist` / dashboard Branding section). The SDK forwards your `redirect_uri` verbatim — the server validates it.

When `redirect_uri` is omitted, the server's `email.default_redirect` is used.

## See also

- [MFA](./mfa.md)
- [Sessions](./sessions.md) — list/revoke active sessions
- [Errors and retries](./errors-and-retries.md)
