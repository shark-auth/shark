# apple.go

**Path:** `internal/auth/providers/apple.go`
**Package:** `providers`
**LOC:** 195
**Tests:** `apple_test.go` (if present)

## Purpose
Sign in with Apple OAuth provider. More complex than other providers because the client_secret is a JWT signed with an ES256 private key (.p8 file) and user info is extracted from the id_token rather than a userinfo endpoint.

## Key types / functions
- `appleEndpoint` (var, line 20) — Apple auth/token URLs (`#nosec G101`).
- `Apple` (type, line 29) — `oauth2.Config` + `teamID`, `keyID`, `privateKeyPath`, optional in-memory `privateKeyPEM` for tests.
- `NewApple` (func, line 38) — config-driven constructor; scopes `["name", "email"]`.
- `NewAppleWithKey` (func, line 53) — test constructor accepting in-memory PEM.
- `Name` (func, line 67) — `"apple"`.
- `AuthURL` (func, line 69) — appends `response_mode=form_post` (Apple's web requirement).
- `Exchange` (func, line 74) — generates the JWT client_secret on demand (line 76), assigns to `cfg.ClientSecret`, then calls `oauth2.Config.Exchange`.
- `GetUser` (func, line 84) — pulls `id_token` from token extras, parses claims unverified (token came directly from Apple over TLS), extracts `sub` and `email`.
- `generateClientSecret` (func, line 117) — reads `.p8` PEM (file or in-memory), parses PKCS#8 ECDSA key, builds JWT with `iss=teamID`, `aud=https://appleid.apple.com`, `sub=clientID`, `kid=keyID`, exp 5 min, signs ES256.
- `appleIDTokenClaims`, `unmarshalAppleIDToken`, `splitJWT` (line 158-194) — helper utilities (currently unused by GetUser; reserved for future verified-decode path).

## Imports of note
- `github.com/golang-jwt/jwt/v5` — JWT signing + parsing.
- `crypto/x509`, `encoding/pem` — Apple .p8 private key parsing.

## Used by
- `internal/server/server.go` — registered if Apple OAuth is configured.

## Notes
- id_token signature is **not** verified against Apple JWKS — relies on TLS to Apple's token endpoint. Acceptable per OAuth 2.0 token-endpoint trust model but worth tightening if id_tokens get cached/reused elsewhere.
- Apple only sends user `name` on the very first authorization (front end must capture it); this implementation leaves Name empty and expects upper layers to handle.
- `Exchange` mutates `a.cfg.ClientSecret` per call — fine for sequential calls, but if concurrent flows run on the same provider instance this races (low risk in practice).
