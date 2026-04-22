# Execution Plan: Closing Backend-Frontend Gaps

## Wave 1: The "New Proxy" (W15 Multi-Listener UI)
**Objective:** Replace the single-listener proxy view with a multi-listener management dashboard.

### 1.1 Backend: Expose Multi-Listener Status
*   **Update `handleAdminProxyStatus`:** Ensure the status endpoint returns an array of listeners or allows querying by `?bind=`.
*   **Update `handleAdminProxyRules`:** Ensure rules can be fetched/managed per-listener.

### 1.2 Frontend: `proxy_config.tsx` Refactor
*   **Listener Selector:** Add a dropdown or sidebar to switch between listeners (e.g., "Default (:8080)", "App 1 (:3000)", "App 2 (:4000)").
*   **Per-Listener Gauges:** Wire the Circuit Breaker gauges and latency stats to the selected listener's metrics.
*   **Multi-Port Awareness:** Show the `bind` address and `upstream` target prominently for each listener.

---

## Wave 2: Updatable Settings (Expectable UX)
**Objective:** Refactor the read-only Settings page into a functional management interface.

### 2.1 Backend: Config Update API
*   **Endpoint:** `PATCH /api/v1/admin/config`
*   **Implementation:**
    *   Accept partial JSON matching the `adminConfigSummary` structure.
    *   Validate inputs (e.g., valid duration strings like "30d", "15m").
    *   Update `s.Config` in memory.
    *   **Persistence:** Write updated values back to `sharkauth.yaml` (carefully preserving structure).
    *   **Hot-Reload:** Trigger re-initialization for Email and Audit cleanup services.

### 2.2 Frontend: `settings.tsx` Interactive Form
*   **Input Components:** Replace `ConfigRow` text with `<input>`, `<select>`, and `<toggle>`.
*   **Categorization:** 
    *   Group by "Authentication", "Passkeys", "Email", and "Governance".
*   **Save Logic:** Add a floating footer "Save Changes" button that calls the new `PATCH` endpoint.
*   **System Protection:** Keep `Base URL`, `DB Path`, and `Port` as read-only labels with a "System-level" badge.

---

## Wave 3: AI & Agent Features
**Objective:** Wire missing RFC 8693 and advanced agent features.

### 3.1 Token Exchange (RFC 8693)
*   **Frontend (`agents_manage.tsx`):** Add `urn:ietf:params:oauth:grant-type:token-exchange` to the `ALL_GRANTS` list.
*   **Frontend:** Add a "Delegation" tab to agent details to manage `may_act` scopes.

### 3.2 Advanced Agent Auth & Metadata
*   **Agent Metadata:** Add a JSON editor for the `metadata` field in the Agent detail panel.
*   **Authentication Methods:** Add a selector for `Authentication Method` (Secret vs. Private Key JWT) and a field for `JWKS URL`.

---

## Wave 4: Hosted Pages & UX Polish (QA Wave)
**Objective:** Fix the redirection bug and close miscellaneous gaps.

### 4.1 Redirection Fix
*   **Backend (`oauth/handlers.go`):** Update `HandleAuthorize` to return an HTTP 302 redirect to the hosted login page instead of a JSON error.
*   **URL Fix:** Ensure the redirect URL includes the app slug: `/hosted/{app_slug}/login`.

### 4.2 User Management Polish
*   **Linked Accounts:** Add a "Social Connections" list to the User Security tab.
*   **Passkeys:** Wire the `GET /admin/users/{id}/passkeys` endpoint to a new "Passkeys" list in the user detail view.
