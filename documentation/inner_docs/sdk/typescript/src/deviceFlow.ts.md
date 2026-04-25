# deviceFlow.ts

**Path:** `sdk/typescript/src/deviceFlow.ts`
**Type:** RFC 8628 device flow client
**LOC:** 254

## Purpose
Implements the OAuth 2.0 Device Authorization Grant — a CLI/agent calls `begin()` to get a `userCode` + `verificationUri` to display, then `waitForApproval()` to poll the token endpoint until approval, denial, or timeout.

## Public API
- `class DeviceFlow`
  - `constructor(opts: DeviceFlowOptions)`
  - `begin(): Promise<DeviceInit>` — POST `/oauth/device_authorization`
  - `waitForApproval(opts?: WaitForApprovalOptions): Promise<TokenResponse>` — polls `/oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:device_code`
- Types: `DeviceInit`, `TokenResponse`, `DeviceFlowOptions`, `WaitForApprovalOptions`

## Polling behavior
- Honors server `interval` (min 1 s).
- `authorization_pending` → continue.
- `slow_down` → `interval += 5 s` per RFC 8628 §3.5.
- `access_denied`, `expired_token` → throw `DeviceFlowError`.
- Default `timeoutMs`: 300 000 (5 min). Throws on deadline.
- `clock` and `sleep` are injectable for deterministic tests.

## Constructor options
- `authUrl: string` — server base URL (trailing slashes stripped)
- `clientId: string`
- `scope?: string` — space-delimited
- `dpopProver?: DPoPProver` — adds `DPoP` header to token POSTs
- `deviceAuthorizationPath?` (default `/oauth/device_authorization`)
- `tokenPath?` (default `/oauth/token`)

## Internal dependencies
- `http.ts` — `httpRequest` form POSTs
- `dpop.ts` — optional proof
- `errors.ts` — `DeviceFlowError`

## Notes
- `begin()` must be called before `waitForApproval()` (cached `_init`).
- Required server keys: `device_code`, `user_code`, `verification_uri`, `expires_in`. Missing fields throw with names listed.
- `verification_uri_complete` and `interval` are optional; interval defaults to 5.
