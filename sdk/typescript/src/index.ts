/**
 * @sharkauth/node — TypeScript SDK for SharkAuth agent-auth primitives
 * and v1.5 admin APIs.
 *
 * Public API:
 * - {@link DPoPProver}          — RFC 9449 DPoP proof JWT emission
 * - {@link DeviceFlow}          — RFC 8628 device authorization grant
 * - {@link VaultClient}         — Shark Token Vault client (reference-token swap)
 * - {@link decodeAgentToken}    — decode a Shark-issued agent token (no verify)
 * - {@link SharkClient}         — DPoP-aware HTTP client + v1.5 admin namespaces
 * - {@link ProxyRulesClient}    — DB-backed proxy rule CRUD
 * - {@link ProxyLifecycleClient}— proxy start / stop / reload / status
 * - {@link BrandingClient}      — branding design tokens
 * - {@link PaywallClient}       — paywall URL builder + HTML renderer
 * - {@link UsersClient}         — user CRUD + tier
 * - {@link AgentsClient}        — agent register / list / get / revoke
 * - {@link OAuthClient}         — RFC 7009 revoke + RFC 7662 introspect
 * - {@link MagicLinkClient}     — send magic-link sign-in emails
 * - {@link SharkAPIError}       — typed error for admin API failures
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
export type { TokenExchangeOptions } from "./exchange.js";
export type {
  AgentTokenClaims,
  ActorClaim,
  ConfirmationClaim,
  AuthorizationDetail,
} from "./tokens.js";

export {
  SharkAuthError,
  DPoPError,
  DeviceFlowError,
  VaultError,
  TokenError,
  SharkAPIError,
} from "./errors.js";

// v1.5 admin modules
export { ProxyRulesClient } from "./proxyRules.js";
export type {
  ProxyRule,
  CreateProxyRuleInput,
  UpdateProxyRuleInput,
  ProxyRuleListResult,
  ImportResult,
  ImportRuleError,
  ProxyRuleMutationResult,
  ProxyRulesClientOptions,
} from "./proxyRules.js";

export { ProxyLifecycleClient } from "./proxyLifecycle.js";
export type {
  ProxyStatus,
  ProxyLifecycleClientOptions,
} from "./proxyLifecycle.js";

export { BrandingClient } from "./branding.js";
export type {
  DesignTokens,
  BrandingRow,
  SetBrandingResult,
  BrandingClientOptions,
} from "./branding.js";

export { PaywallClient } from "./paywall.js";
export type {
  PaywallOptions,
  PaywallPreviewOptions,
  PaywallClientOptions,
} from "./paywall.js";

export { UsersClient } from "./users.js";
export type {
  UserTier,
  User,
  CreateUserInput,
  UpdateUserInput,
  SetUserTierResult,
  ListUsersOptions,
  UserListResult,
  UsersClientOptions,
} from "./users.js";

export { AgentsClient } from "./agents.js";
export type {
  Agent,
  AgentCreateResult,
  RegisterAgentInput,
  ListAgentsOptions,
  AgentListResult,
  AgentsClientOptions,
} from "./agents.js";

export { OAuthClient, pkcePair } from "./oauth.js";
export type {
  TokenTypeHint,
  IntrospectResult,
  OAuthClientOptions,
  OAuthToken,
  PkcePair,
  BuildAuthorizeUrlOptions,
} from "./oauth.js";

export { MagicLinkClient } from "./magicLink.js";
export type {
  SendMagicLinkResult,
  MagicLinkClientOptions,
} from "./magicLink.js";

// Pass-A human-auth modules (Phase 7 SDK port from Python)
export { AuthClient, Session } from "./auth.js";
export type {
  AuthClientOptions,
  AuthUser,
  SignupResult,
  SignupOptions,
  LoginResult,
  MeResult,
  MagicLinkVerifyResult,
} from "./auth.js";

export { MfaClient, computeTotp } from "./mfa.js";
export type {
  MfaClientOptions,
  MfaEnrollResult,
  MfaVerifyResult,
} from "./mfa.js";

export { SessionsClient } from "./sessions.js";
export type {
  SessionsClientOptions,
  SessionRow,
} from "./sessions.js";

export { ConsentsClient } from "./consents.js";
export type {
  ConsentsClientOptions,
  ConsentRow,
} from "./consents.js";

export { DcrClient } from "./dcr.js";
export type {
  DcrClientOptions,
  DcrRegisterInput,
  DcrRegisterResult,
  DcrClientMetadata,
  DcrRotateSecretResult,
  DcrRotateTokenResult,
} from "./dcr.js";

// Pass-B admin modules (Phase 7 SDK port from Python)
export { WebhooksClient, verifySignature } from "./webhooks.js";
export type {
  Webhook,
  WebhookDelivery,
  RegisterWebhookInput,
  UpdateWebhookInput,
  WebhooksClientOptions,
} from "./webhooks.js";

export { OrganizationsClient } from "./organizations.js";
export type {
  Organization,
  OrgMember,
  OrgInvitation,
  CreateOrganizationInput,
  UpdateOrganizationInput,
  OrganizationsClientOptions,
} from "./organizations.js";

export { AppsClient } from "./apps.js";
export type {
  App,
  AppRotateSecretResult,
  CreateAppInput,
  UpdateAppInput,
  AppsClientOptions,
} from "./apps.js";

export { ApiKeysClient } from "./apiKeys.js";
export type {
  ApiKey,
  CreateApiKeyInput,
  ApiKeysClientOptions,
} from "./apiKeys.js";

export { AuditClient } from "./audit.js";
export type {
  AuditEvent,
  AuditListFilters,
  AuditListResult,
  AuditExportFilters,
  AuditPurgeResult,
  AuditClientOptions,
} from "./audit.js";

export { MayActClient } from "./mayAct.js";
export type {
  MayActGrant,
  MayActFindOptions,
  MayActCreateInput,
  MayActClientOptions,
} from "./mayAct.js";

export { RbacClient } from "./rbac.js";
export type {
  Role,
  Permission,
  UpdateRoleInput,
  RbacClientOptions,
} from "./rbac.js";

export const VERSION = "0.1.0";
