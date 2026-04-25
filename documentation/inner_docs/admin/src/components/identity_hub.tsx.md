# identity_hub.tsx

**Path:** `admin/src/components/identity_hub.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-25 (strict monochrome/square/editable spec)

## Purpose
Central configuration for authentication identity: methods, sessions+JWT (unified), MFA, OAuth server. Every field editable via PATCH `/admin/config`.

## Exports
- `IdentityHub()` — function component (default + named export `IdentityHub`)

## Sections
- **Authentication methods** — password (min length, reset TTL), magic link (toggle, token TTL), passkeys (RP name, RP ID, user verification level), social providers (Google/GitHub/Apple/Discord rows; click row → drawer with client_id/client_secret/callback URL + copy)
- **Sessions & Tokens** — ONE section. Cookie session lifetime, JWT access lifetime, JWT issuer, JWT audience, signing keys list (active marker + Rotate button + per-key inspect drawer). Cookie + JWT BOTH always on; no mode toggle anywhere.
- **MFA** — enforcement select (off/optional/required), TOTP issuer, recovery code count
- **OAuth Server** — toggle, issuer URL, access/refresh token lifetimes, DPoP requirement

## Editable fields (~28)
auth.session_lifetime, auth.password_min_length, password_reset.ttl, magic_link.ttl, passkey.rp_name, passkey.rp_id, passkey.user_verification, jwt.lifetime, jwt.issuer, jwt.audience, social.redirect_url, 4 providers × {client_id, client_secret}, mfa.enforcement, mfa.issuer, mfa.recovery_codes, oauth_server.{enabled, issuer, access_token_lifetime, refresh_token_lifetime, require_dpop}.

## API calls
- `GET /admin/config` — fetch
- `PATCH /admin/config` — save (dirty diff)
- `POST /admin/auth/rotate-signing-key` — rotate JWT key

## Composed by
- `App.tsx` — route `auth: IdentityHub`

## Visual contract (per .impeccable.md v3)
- Monochrome only — color exclusively on circular status dots (success/warn/danger/fg-dim)
- Border radii: cards 5px, inputs 4px, badges 3px, dots circular
- 13px base, 11px uppercase labels w/ tracking, hairline borders, 7-10px row padding
- Right-side drawers (480px) for "Configure provider" / "Inspect signing key" — never modal
- Topbar mirrors users.tsx rhythm

## Notes
- Some PATCH fields are forward-compatible (backend ignores unknown fields); GET returns live values
- No "session vs JWT" toggle anywhere — both auth paths always enabled
- No low-level knobs surfaced (clock_skew, signing_alg removed from UI)
