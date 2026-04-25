# index.ts

**Path:** `sdk/typescript/src/index.ts`
**Type:** Public barrel
**LOC:** 109

## Purpose
Single public entry point for `@sharkauth/node`. Re-exports every consumer-facing class, function, and type. The only file resolved by `package.json` exports.

## Public API (re-exports)
### Agent-auth primitives
- `DPoPProver` (+ `CreateProofOptions`) — RFC 9449 proof JWT emission
- `DeviceFlow` (+ `DeviceInit`, `TokenResponse`, `DeviceFlowOptions`, `WaitForApprovalOptions`)
- `VaultClient` (+ `VaultToken`, `VaultClientOptions`) — Shark Token Vault
- `exchangeToken` (+ `TokenExchangeOptions`) — RFC 8693 token exchange
- `decodeAgentToken` and token claim types (`AgentTokenClaims`, `ActorClaim`, `ConfirmationClaim`, `AuthorizationDetail`)

### High-level client
- `SharkClient` (+ `SharkClientOptions`) — DPoP-aware fetch + admin namespaces

### v1.5 admin namespaces
- `ProxyRulesClient` (+ `ProxyRule`, `CreateProxyRuleInput`, `UpdateProxyRuleInput`, `ProxyRuleListResult`, `ImportResult`, `ImportRuleError`, `ProxyRuleMutationResult`)
- `ProxyLifecycleClient` (+ `ProxyStatus`)
- `BrandingClient` (+ `DesignTokens`, `BrandingRow`, `SetBrandingResult`)
- `PaywallClient` (+ `PaywallOptions`, `PaywallPreviewOptions`)
- `UsersClient` (+ `UserTier`, `User`, `SetUserTierResult`, `ListUsersOptions`, `UserListResult`)
- `AgentsClient` (+ `Agent`, `AgentCreateResult`, `RegisterAgentInput`, `ListAgentsOptions`, `AgentListResult`)

### Errors
- `SharkAuthError` (base)
- `DPoPError`, `DeviceFlowError`, `VaultError`, `TokenError`, `SharkAPIError`

### Constant
- `VERSION = "0.1.0"`

## Notes
- All `.js` import suffixes are mandatory (NodeNext ESM).
- `index.ts` is excluded from coverage — it has no logic.
- This file is the contract: anything not re-exported is internal.
