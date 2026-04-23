/**
 * @sharkauth/node — TypeScript SDK for SharkAuth agent-auth primitives.
 *
 * Public API:
 * - {@link DPoPProver} — RFC 9449 DPoP proof JWT emission
 * - {@link DeviceFlow} — RFC 8628 device authorization grant
 * - {@link VaultClient} — Shark Token Vault client (reference-token swap)
 * - {@link decodeAgentToken} — decode a Shark-issued agent token (no verify)
 */

export { DPoPProver } from "./dpop.js";
export type { CreateProofOptions } from "./dpop.js";

export { DeviceFlow } from "./deviceFlow.js";
export type {
  DeviceInit,
  TokenResponse,
  DeviceFlowOptions,
  WaitForApprovalOptions,
} from "./deviceFlow.js";

export { VaultClient } from "./vault.js";
export type { VaultToken, VaultClientOptions } from "./vault.js";

export { SharkClient } from "./sharkClient.js";
export type { SharkClientOptions } from "./sharkClient.js";

export { exchangeToken } from "./exchange.js";
export type {
  AgentTokenClaims,
  ActorClaim,
  ConfirmationClaim,
  AuthorizationDetail,
} from "./tokens.js";

export { exchangeToken } from "./exchange.js";
export type { TokenExchangeOptions } from "./exchange.js";

export {
  SharkAuthError,
  DPoPError,
  DeviceFlowError,
  VaultError,
  TokenError,
} from "./errors.js";

export const VERSION = "0.1.0";
