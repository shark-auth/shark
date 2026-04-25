# device.go

**Path:** `internal/oauth/device.go`
**Package:** `oauth`
**LOC:** 581
**Tests:** `device_test.go`

## Purpose
RFC 8628 OAuth 2.0 Device Authorization Grant — for headless agents and CLIs. Includes per-client_id rate limiting.

## RFCs implemented
- RFC 8628 OAuth 2.0 Device Authorization Grant
- RFC 8707 resource indicator threading (line 120)

## Key types / functions
- `deviceRateLimitPerMin` (const, line 26) — 10 req/min cap per client_id.
- `deviceRateEntry` + `deviceRateMap`/`deviceRateMu` (line 28-36) — process-local sliding-window limiter.
- `checkDeviceRateLimit` (func, line 40) — enforces the cap, returns 429 → `slow_down`.
- `generateDeviceCode` (func, line 64) — 32-byte base64url code + sha256 hex hash for storage.
- `generateUserCode` (func, line 77) — 8-char `XXXX-XXXX` code from unambiguous charset (no I/O/0/1).
- `deviceAuthResponse` (struct, line 94) — RFC 8628 §3.2 wire shape.
- `HandleDeviceAuthorization` (func, line 105) — POST /oauth/device.
- `HandleDeviceVerify` (func, line 200) — GET /oauth/device/verify code-entry page.
- `HandleDeviceConfirm`, `HandleDeviceTokenRequest` (later) — user approval + polling endpoint.

## Imports of note
- `crypto/rand`, `math/big` for unbiased charset selection
- `internal/storage` (oauth_device_codes table)

## Wired by
- `internal/server/server.go` — mounts `/oauth/device`, `/oauth/device/verify`, `/oauth/device/confirm`.
- `handlers.go:50` — token-endpoint dispatch when grant_type is `urn:ietf:params:oauth:grant-type:device_code`.

## Used by
- Headless agent CLIs, IoT-style flows.

## Notes
- Rate limiter is process-local (in-memory map) — see SCALE.md.
- Device code stored as sha256(plain) — leaked DB row can't be polled.
- 5-second poll interval, 15-minute default lifetime.
