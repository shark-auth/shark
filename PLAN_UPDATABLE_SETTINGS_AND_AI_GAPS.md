# Updatable Settings & AI Features Gap Plan

## 1. The "Updatable Settings" Plan
The current "config-as-code" philosophy restricts the `Settings` screen to a read-only state. We will refactor this to allow administrators to update "UX/Policy" settings directly from the dashboard while keeping low-level infrastructure settings read-only to prevent accidental lockouts.

### Backend Implementation
*   **New Endpoint:** `PATCH /api/v1/admin/config`
*   **Payload:** Partial JSON map of updatable config values.
*   **Persistence:** The handler will apply changes to the in-memory `Config` struct and serialize the updates back to `sharkauth.yaml` using a YAML encoder (e.g., `gopkg.in/yaml.v3`) to ensure changes survive restarts.
*   **Hot-Reload Hooks:** 
    *   *Email:* Updating SMTP/Resend credentials will trigger a re-initialization of the `email.Sender` service.
    *   *Audit:* Updating the `cleanup_interval` will restart the background cleanup worker.
    *   *JWT/Sessions:* TTL changes will take effect immediately for newly issued tokens.
*   **Read-Only Boundary:** The backend will explicitly **ignore** attempts to patch `server.port`, `server.base_url`, `storage.path`, and `server.secret`.

### Frontend Implementation (`admin/src/components/settings.tsx`)
*   **Refactor to Form:** Convert the read-only `ConfigRow` components into interactive inputs (Text, Number, Select, Toggles).
*   **Categorization:**
    *   **Authentication:** Session Lifetime, Password Min Length, JWT TTLs.
    *   **Passkeys:** RP Name, RP ID, User Verification requirement.
    *   **Email:** Provider selector (Shark, Resend, SMTP) with dynamic credential fields.
    *   **Governance:** Audit Retention days, Default API Key Rate Limits.
*   **Read-Only Badges:** Core infra settings (Base URL, DB Path) will remain rendered as plain text with a "System" or "Read-Only" badge.
*   **UX:** Add a floating "Save Changes" action bar that appears when the form is dirty, with optimistic UI updates and toast notifications on success.

---

## 2. Proxy Modes Clarification
Investigation confirms SharkAuth supports **two distinct proxy modes**:

1. **Path-Based Gateway (Legacy/Catch-All)**
   *   **How it works:** Mounted as a `/*` catch-all on the *same port* as the SharkAuth management API (e.g., `:8080`).
   *   **Behavior:** It protects `proxyurl/urappurl`. SharkAuth API routes take precedence, and any unmatched path is forwarded to the single configured upstream.
   
2. **Native Protection (W15 Multi-Listener)**
   *   **How it works:** Uses dedicated `proxy.Listener` instances, each binding to its own port (e.g., `:3000`).
   *   **Behavior:** It protects `yourappurlraw/` natively. A single SharkAuth binary can bind to the app's native port, intercept traffic, enforce auth rules, and forward it to a hidden backend port (e.g., `localhost:3001`), functioning as a completely transparent zero-trust gateway.

---

## 3. AI / Agent Features Unwired in Dashboard
SharkAuth has robust backend support for AI agent workloads (M2M), but several features are disconnected from the UI:

*   **Token Exchange (RFC 8693):** The backend supports Agent-to-Agent delegation (building a delegation chain via the `act` claim). However, the dashboard omits the `urn:ietf:params:oauth:grant-type:token-exchange` grant type, so it cannot be enabled for agents.
*   **Advanced Agent Authentication:** The backend supports `private_key_jwt` and `JWKS` / `JWKSURI` for highly secure agent authentication, but the UI only allows configuring `client_secret_basic`.
*   **Fine-Grained Delegation Scopes:** Token exchange logic supports `may_act` claims for scoping what an agent is allowed to do when delegating, which lacks any management interface.
*   **Agent Metadata:** A `metadata` JSON field exists for agents in the DB but is not editable or visible in the dashboard.
