# SharkAuth Admin Dashboard — Feature & UX Specification

This document defines every feature, interaction pattern, and UX decision for the embedded Svelte admin dashboard served at `/admin`. It is based on an exhaustive analysis of the current backend API surface and data model, competitive analysis of Auth0, Clerk, and WorkOS dashboards, and first-principles UX thinking about what makes an auth dashboard feel effortless.

The goal: **the admin who deploys SharkAuth should never need to open a terminal to manage their auth system.** Every operation the API supports should be reachable from the dashboard, and the dashboard should surface insights the raw API cannot.

---

## Guiding Principles

1. **Progressive disclosure.** The overview shows what matters. Details are one click away, never zero and never three.
2. **Act where you see.** Every piece of data that can be acted on has its action inline. No "go to settings to change this."
3. **Keyboard-first, mouse-friendly.** Command palette (Cmd+K) reaches everything. Tab navigation works. But nothing requires a keyboard.
4. **No empty states.** Every page has useful content from day one — even if it's "0 users, here's how to create your first one."
5. **Undo over confirmation.** Destructive actions show a toast with an undo window (5 seconds) instead of a "Are you sure?" modal. Exception: user deletion, which is irreversible and gets a confirmation with the user's email typed to confirm.
6. **Real-time where it matters.** Audit log and active sessions should poll or stream. User list doesn't need to.
7. **Copy everything.** User IDs, API keys, curl commands, error messages — one click to clipboard, always.

---

## Navigation Structure

Flat left sidebar, no nesting. 9 top-level items, always visible. Active item highlighted. Collapsible to icons on narrow screens.

```
[SharkAuth logo]              ← links to Overview
---
Overview                      ← metrics, activity feed, health
Users                         ← user table, detail, CRUD
Sessions                      ← active sessions, per-user view
Authentication                ← auth method config, OAuth, magic links, passkeys
SSO                           ← SAML/OIDC connections, domain routing
Roles & Permissions           ← RBAC management
API Keys                      ← M2M key management
Audit Log                     ← full event stream, filters, export
Settings                      ← server config, email, security
---
[instance badge: v0.1.0]
[health indicator: green dot]
```

**Top bar (persistent):**
- Global search / command palette trigger (Cmd+K)
- Quick actions dropdown (+ New User, + New API Key, + New Role)
- Notification bell (lockouts, failed logins, expiring keys — future: requires backend support)

---

## Page Specifications

### 1. Overview

The landing page after login. Answers: "Is my auth system healthy? What happened recently?"

**Metric Cards (top row, 4-6 cards):**

| Card | Value | Subtext | Source |
|------|-------|---------|--------|
| Total Users | count | +N this week | `COUNT(*) FROM users` |
| Active Sessions | count | — | `COUNT(*) FROM sessions WHERE expires_at > now` |
| MFA Adoption | percentage | N of M users | `COUNT(*) WHERE mfa_enabled / total` |
| Failed Logins (24h) | count | trend arrow | `COUNT(*) FROM audit_logs WHERE action='login' AND status='failure' AND created_at > -24h` |
| API Keys Active | count | N expiring soon | `COUNT(*) FROM api_keys WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > now)` |
| SSO Connections | count | N enabled | `COUNT(*) FROM sso_connections` |

> **Backend needed:** A `GET /api/v1/admin/stats` endpoint that returns pre-computed counts. Querying the raw tables from the frontend per-card is wasteful. This endpoint should return all overview metrics in one call.

**Auth Method Breakdown (horizontal bar or donut):**
- Percentage of sessions by `auth_method`: password, oauth (by provider), passkey, magic link, SSO
- Source: `GROUP BY auth_method FROM sessions WHERE created_at > -30d`

**Signup Trend (sparkline or area chart, 30 days):**
- Daily new user count
- Source: `GROUP BY DATE(created_at) FROM users WHERE created_at > -30d`

**Recent Activity Feed (live-updating list, last 20 events):**
- Shows: timestamp, action icon, actor (user email or API key name), action description, status badge (success/failure)
- Each item links to the full audit log entry
- Auto-refreshes every 10 seconds (polling) or via SSE (future)

**System Health:**
- SQLite database size
- Uptime (if tracked)
- Last migration run
- Config warnings (e.g., CORS not configured, no OAuth providers enabled)

> **Backend needed:** A `GET /api/v1/admin/health` endpoint returning DB size, uptime, version, config summary (which providers are enabled, whether SMTP is configured, whether CORS is set). Different from `/healthz` — this is an admin-only diagnostic view.

---

### 2. Users

The most-used page. Must handle 10 users and 100,000 users equally well.

**User Table:**

| Column | Content | Sortable | Filterable |
|--------|---------|----------|------------|
| User | Avatar + name + email (stacked) | by name, email | text search |
| Verified | green/gray badge | — | yes (verified/unverified) |
| MFA | shield icon (on/off) | — | yes (enabled/disabled) |
| Auth Method | last login method icon | — | yes (password/oauth/passkey/magic/sso) |
| Roles | comma-separated role names, max 2 + "+N" | — | yes (by role) |
| Created | relative time ("3 days ago") | yes | date range |
| Last Active | relative time | yes | — |

> **Backend needed:** `last_active` isn't currently tracked on the user model. Options: (a) derive from most recent session `created_at` (expensive at scale), (b) add `last_login_at` to the users table and update on every login. Option (b) is recommended.

> **Backend needed:** Filter by `auth_method` requires joining sessions or adding a `last_auth_method` field to users. Filter by `role` requires a `role_id` query param on `GET /users`.

**Table Interactions:**
- Click row → opens user detail panel (slide-over from right, not a new page)
- Checkbox selection → batch actions bar appears at bottom: "Delete N users", "Assign role to N users"
- Pagination: offset-based with page size selector (25, 50, 100)

**User Detail Panel (slide-over):**

The detail panel is the core of user management. It has tabs:

**Tab: Profile**
- Editable fields (inline edit, save on blur or Enter):
  - Name
  - Email
  - Email verified (toggle)
  - Metadata (JSON editor with syntax highlighting)
- Read-only:
  - User ID (with copy button)
  - Created at, Updated at
  - Avatar (from OAuth, not editable)
- Actions:
  - "Send Verification Email" (if not verified)
  - "Reset Password" (sends reset link)
  - "Disable MFA" (if enabled — requires confirmation)
  - "Impersonate" (future: opens the app as this user)
  - "Delete Account" (red, requires typing email to confirm)

> **Backend needed:** An admin-initiated password reset that sends the reset email without the user requesting it. Currently `POST /auth/password/send-reset-link` exists but it's a public endpoint. The dashboard should call it server-side or there should be an admin variant.

> **Backend needed (god-tier):** User impersonation — `POST /api/v1/admin/impersonate/{user_id}` that creates a session for the target user and returns a session cookie. This is Clerk's standout feature. Must be audit-logged with actor_type="admin".

**Tab: Security**
- MFA status: enabled/disabled, enrolled date
- Passkeys: list of registered credentials (name, created, last used, backed up, transports). Actions: rename, delete.
- OAuth accounts: list of linked providers (Google, GitHub, etc.) with provider email. Action: unlink.
- Password: has password (yes/no), hash type (argon2id/bcrypt). Action: force password reset.
- Active sessions: list (IP, user agent, auth method, created, expires, MFA passed). Action: revoke (individual or "revoke all").

> **Backend needed:** `GET /api/v1/users/{id}/sessions` — list active sessions for a user. `DELETE /api/v1/sessions/{id}` — revoke a specific session. `DELETE /api/v1/users/{id}/sessions` — revoke all sessions for a user.

> **Backend needed:** `GET /api/v1/users/{id}/oauth-accounts` — list linked OAuth accounts. `DELETE /api/v1/users/{id}/oauth-accounts/{id}` — unlink an OAuth account. (Store methods exist: `GetOAuthAccountsByUserID`, `DeleteOAuthAccount`.)

> **Backend needed:** `GET /api/v1/users/{id}/passkeys` — list passkey credentials. (Store method exists: `GetPasskeysByUserID`.)

**Tab: Roles & Permissions**
- Assigned roles: list with remove button
- Effective permissions: computed list (from all roles), shows which role granted each permission
- "Assign Role" button → dropdown of available roles
- "Check Permission" inline tool: enter action + resource, see if allowed

**Tab: Activity**
- User's audit log: same as main audit log but pre-filtered to `actor_id=user_id OR target_id=user_id`
- Shows: login history, password changes, MFA events, role changes
- Filterable by action type and date range

---

### 3. Sessions

Dedicated page for session monitoring. Answers: "Who is logged in right now?"

> **Backend needed:** `GET /api/v1/admin/sessions` — list all active sessions with user email joined. Filters: auth_method, mfa_passed, created after/before. Pagination.

**Session Table:**

| Column | Content |
|--------|---------|
| User | avatar + email |
| Auth Method | icon + label (password, google, passkey...) |
| IP | IP address with geo hint (future: GeoIP lookup) |
| Device | parsed user agent (Chrome on macOS, etc.) |
| MFA | passed/pending badge |
| Created | relative time |
| Expires | relative time, red if < 24h |
| Actions | Revoke button |

**Aggregate View (cards above table):**
- Total active sessions
- By auth method (bar)
- MFA completion rate
- Sessions by device type (future: requires user agent parsing)

---

### 4. Authentication

Configuration page for all auth methods. Not a CRUD table — a settings-style layout with sections.

**Section: Password Policy**
- Minimum length (number input, current config value)
- Complexity requirements: "Uppercase + lowercase + digit" (read-only, shows current enforcement)
- Account lockout: threshold (5), duration (15 min) (read-only from config, shows current values)

> **Backend needed:** Lockout config should be in `sharkauth.yaml` (currently hardcoded to 5 attempts / 15 min). Dashboard should read this from a config endpoint.

> **Backend needed:** `GET /api/v1/admin/config` — returns non-sensitive config values (port, base_url, auth settings, enabled providers, MFA issuer, session lifetime, etc.). Never returns secrets.

**Section: OAuth Providers**
- Card per provider (Google, GitHub, Apple, Discord)
- Each card shows:
  - Status: Configured / Not Configured (green/gray)
  - Client ID (masked, with copy)
  - Scopes (editable list)
  - Redirect URL (computed, read-only, with copy)
  - Test button: opens OAuth flow in new tab

> **Backend needed (god-tier):** OAuth provider config via API. Currently providers are configured only via YAML. An admin endpoint to update OAuth credentials without restarting would be excellent. This is a significant backend change (move provider config from YAML to DB, or support hybrid).

**Section: Magic Links**
- Token lifetime (shows current value)
- Redirect URL (shows current value, with copy)
- Rate limit: 1 per email per 60 seconds (read-only)

**Section: Passkeys**
- RP Name, RP ID, Origin (shows current config)
- Attestation preference
- Resident key requirement
- User verification setting
- Total registered passkeys count

**Section: Email Verification**
- Status: "Email verification available" (if SMTP configured) / "Not configured"
- Template preview (read-only HTML preview of verification email)

> **Backend needed (god-tier):** Email template preview endpoint — `GET /api/v1/admin/email-preview/{template}` that returns rendered HTML with sample data. Templates: magic_link, password_reset, verify_email.

---

### 5. SSO

Management page for enterprise SSO connections.

**Connection List:**

| Column | Content |
|--------|---------|
| Name | connection display name |
| Type | SAML / OIDC badge |
| Domain | linked domain (or "—") |
| Status | Enabled (green) / Disabled (gray) toggle |
| Users | count of users linked via this connection |
| Created | date |
| Actions | Edit, Delete, Test |

> **Backend needed:** Count of users per SSO connection — `SELECT connection_id, COUNT(*) FROM sso_identities GROUP BY connection_id`.

**Create/Edit Connection (modal or slide-over):**

Dynamic form based on type selection:

**OIDC:**
- Name (text)
- Domain (text, for auto-routing)
- Issuer URL (text, with "Discover" button that fetches .well-known)
- Client ID (text)
- Client Secret (password field)
- Enabled (toggle)

**SAML:**
- Name (text)
- Domain (text)
- IdP URL (text)
- IdP Certificate (textarea, PEM format, with "Upload" file button)
- SP Entity ID (auto-filled from config, editable)
- SP ACS URL (auto-filled, read-only, with copy)
- Enabled (toggle)
- "Download SP Metadata" button (links to `/api/v1/sso/saml/{id}/metadata`)

**Connection Detail View:**
- All config fields (editable)
- "Test Connection" button → opens SSO flow in new tab
- Users linked via this connection (table)
- Activity log filtered to SSO events for this connection

---

### 6. Roles & Permissions

Two-panel layout: roles list on left, role detail on right.

**Roles List (left panel):**
- List of all roles with name, description snippet, user count
- "New Role" button at top
- Click to select → shows detail on right
- Drag to reorder (future, cosmetic only)

**Role Detail (right panel):**

- Name (inline editable)
- Description (inline editable)
- Permissions section:
  - Table of attached permissions: action, resource, with detach (X) button
  - "Attach Permission" → searchable dropdown of existing permissions
  - "Create & Attach" → inline form for new permission (action + resource)
- Users section:
  - List of users with this role (avatar + email)
  - "Assign to User" → user search autocomplete
  - Remove button per user

**Permission Explorer (separate tab or bottom section):**
- Full list of all permissions with usage count (how many roles use each)
- "Check Permission" tool: select user → enter action + resource → result

> **Backend needed (god-tier):** `GET /api/v1/permissions/{id}/roles` — which roles have this permission. `GET /api/v1/permissions/{id}/users` — which users effectively have this permission (through any role).

---

### 7. API Keys

Management page for M2M API keys.

**Key Table:**

| Column | Content |
|--------|---------|
| Name | key display name |
| Key | `sk_live_AbCd...xK9f` (masked, with copy full prefix+suffix) |
| Scopes | badge per scope, `*` shown as "Admin" |
| Rate Limit | N req/hr |
| Last Used | relative time (or "Never") |
| Expires | date or "Never" |
| Status | Active (green) / Revoked (red) / Expired (yellow) |
| Actions | Edit, Rotate, Revoke |

**Create Key (modal):**
- Name (required)
- Scopes (multi-select from suggestions + freeform, e.g., "users:read", "users:write", "*")
- Rate Limit (number, default from config)
- Expires (date picker, or "No expiration")
- On create: **full key shown in a prominent, non-dismissable banner until user confirms they copied it.** Copy button. Banner: "This key will not be shown again."

**Key Detail (slide-over on click):**
- All fields (some editable: name, scopes, rate limit, expires)
- Usage section: recent audit log entries where `actor_type='api_key' AND actor_id=key_id`
- Rotate button: creates new key, shows it, revokes old
- Revoke button: red, with undo toast (5 second window)

**curl Snippet Generator:**
- After creating or viewing a key, show a ready-to-paste curl command:
  ```
  curl -H "Authorization: Bearer sk_live_..." https://auth.example.com/api/v1/users
  ```
- Adapts to the key's scopes (e.g., if scopes include "users:read", show the /users endpoint)

---

### 8. Audit Log

The investigation tool. Must handle millions of rows without pagination lag.

**Filter Bar (sticky top):**
- Action type: multi-select dropdown (login, signup, mfa_challenge, etc., grouped by category)
- Status: success / failure / all
- Actor: user search autocomplete (or "Any")
- Target: entity search (or "Any")
- IP: text input
- Date range: preset buttons (Last hour, Today, Last 7 days, Last 30 days, Custom)

**Event Stream (main content):**

| Column | Content |
|--------|---------|
| Time | relative + absolute on hover |
| Status | green dot (success) / red dot (failure) |
| Action | human-readable label with icon |
| Actor | user email, API key name, or "System" (linked) |
| Target | entity type + ID (linked) |
| IP | address (with copy) |
| Details | expand arrow |

**Expanded Event Detail (inline accordion):**
- Full metadata JSON (syntax highlighted)
- User agent (parsed)
- Request ID
- "View User" / "View Target" links

**Export:**
- "Export CSV" button in toolbar
- Uses the current filters
- Streams download (no in-memory buffering)

**Live Mode Toggle:**
- When enabled: new events appear at top with subtle animation
- Pauses when user scrolls down
- Resumes when user scrolls back to top
- Polling interval: 5 seconds

---

### 9. Settings

Configuration and system management.

**Section: Server**
- Port (read-only)
- Base URL (read-only, with copy)
- Secret status: "Configured (64 chars)" — never shows the value
- CORS origins: list (read-only from config)

**Section: Email**
- Provider: SMTP / Resend (read-only)
- From address (read-only)
- "Send Test Email" button → sends a test to a provided address

> **Backend needed:** `POST /api/v1/admin/test-email` — sends a test email to a given address.

**Section: Session**
- Session lifetime (read-only, from config)
- Active sessions count
- "Purge Expired Sessions" button

> **Backend needed:** `POST /api/v1/admin/sessions/purge-expired` — triggers `DeleteExpiredSessions()`.

**Section: Audit Retention**
- Current retention policy (from config)
- Audit log row count
- Database size
- "Purge Old Logs" button (with date picker for cutoff)

> **Backend needed:** `POST /api/v1/admin/audit-logs/purge` — triggers `DeleteAuditLogsBefore(cutoff)`.

**Section: Danger Zone**
- "Rotate Server Secret" — link to SECRETS.md with warning
- "Export All Data" — full database export (future)
- "Delete All Users" — nuclear option, requires typing "DELETE ALL USERS"

---

## Global UX Patterns

### Command Palette (Cmd+K)

Accessible from anywhere. Searches across:
- Users (by email or name)
- Roles (by name)
- API Keys (by name)
- SSO Connections (by name or domain)
- Pages (navigate to any section)
- Actions ("Create user", "Create API key", "Export audit log")

Results grouped by type, keyboard navigable, Enter to select.

### Toast Notifications

Bottom-right stack. Types:
- **Success** (green): "User updated", "API key created"
- **Undo** (blue): "User deleted — Undo (5s)" with countdown
- **Error** (red): "Failed to update user: email already exists"
- **Info** (gray): "Audit log export started"

Auto-dismiss after 5 seconds (except undo, which stays for its window).

### Empty States

Every table and list has a meaningful empty state:
- Users: "No users yet. Users appear here when they sign up via your app, or you can create one manually."
- API Keys: "No API keys. Create one to authenticate server-to-server requests."
- SSO: "No SSO connections. Add a SAML or OIDC connection to enable enterprise single sign-on."

### Loading States

Skeleton screens (gray animated placeholders) instead of spinners. Match the layout of the content they replace.

### Error States

Inline error messages below the field that failed. Red border on the field. For API errors: toast with the error message and a "Retry" action if applicable.

### Responsive Design

Desktop-first (1200px+ optimized), but functional at 768px. Sidebar collapses to icons. Tables become card lists on mobile. Detail panels become full-screen modals.

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| Cmd+K | Command palette |
| G then U | Go to Users |
| G then A | Go to Audit Log |
| G then S | Go to Settings |
| G then K | Go to API Keys |
| / | Focus search in current table |
| Esc | Close panel/modal |
| N | New (context-dependent: new user, new key, new role) |

### Deep Linking

Every entity and view state is URL-addressable:
- `/admin/users` — user list
- `/admin/users?search=john&verified=true` — filtered user list
- `/admin/users/usr_abc123` — user detail panel open
- `/admin/users/usr_abc123/security` — user detail, security tab
- `/admin/audit?action=login&status=failure&from=2026-04-01` — filtered audit log
- `/admin/roles/role_xyz` — role detail
- `/admin/api-keys/key_abc` — key detail

### Relative Time

All timestamps shown as relative ("3 minutes ago", "yesterday", "2 weeks ago") with full ISO timestamp on hover. Toggle in settings to show absolute times.

---

## Backend Endpoints Needed (Summary)

These do not exist today and are required (R) or god-tier enhancements (G) for the dashboard:

| Endpoint | Purpose | Priority |
|----------|---------|----------|
| `GET /api/v1/admin/stats` | Overview metrics (user count, session count, MFA rate, failed logins, etc.) | R |
| `GET /api/v1/admin/health` | System diagnostics (DB size, version, uptime, config summary) | R |
| `GET /api/v1/admin/config` | Non-sensitive config values for display | R |
| `GET /api/v1/admin/sessions` | List all active sessions with user info joined | R |
| `DELETE /api/v1/sessions/{id}` | Revoke a specific session | R |
| `DELETE /api/v1/users/{id}/sessions` | Revoke all sessions for a user | R |
| `GET /api/v1/users/{id}/sessions` | List sessions for a specific user | R |
| `GET /api/v1/users/{id}/oauth-accounts` | List linked OAuth accounts | R |
| `DELETE /api/v1/users/{id}/oauth-accounts/{id}` | Unlink an OAuth account | R |
| `GET /api/v1/users/{id}/passkeys` | List passkey credentials for a user | R |
| `POST /api/v1/admin/sessions/purge-expired` | Trigger expired session cleanup | R |
| `POST /api/v1/admin/audit-logs/purge` | Purge audit logs before a date | R |
| `POST /api/v1/admin/test-email` | Send a test email | G |
| `GET /api/v1/admin/email-preview/{template}` | Preview rendered email template | G |
| `POST /api/v1/admin/impersonate/{user_id}` | Create session as another user (audit-logged) | G |
| Users: `last_login_at` field | Track last login time on user model | R |
| Users: filter by role, auth_method, MFA status | Extended query params on `GET /users` | R |
| SSO: user count per connection | Aggregation query | R |
| Permissions: reverse lookup (which roles/users have this permission) | `GET /permissions/{id}/roles` | G |
| Lockout config in YAML | Move hardcoded lockout params to config | R |
| OAuth config via API | Update provider credentials without restart | G |

---

## Information Architecture Summary

```
Overview
  ├── Metric cards (users, sessions, MFA, failures, keys, SSO)
  ├── Auth method breakdown chart
  ├── Signup trend sparkline
  ├── Recent activity feed (live)
  └── System health

Users
  ├── Searchable, filterable table
  ├── Batch actions (delete, assign role)
  └── Detail panel
      ├── Profile tab (edit name, email, metadata)
      ├── Security tab (MFA, passkeys, OAuth, sessions)
      ├── Roles tab (assign/remove, effective permissions)
      └── Activity tab (per-user audit log)

Sessions
  ├── Active session table with revoke actions
  └── Aggregate metrics

Authentication
  ├── Password policy (read-only config view)
  ├── OAuth providers (status cards, test buttons)
  ├── Magic links (config view)
  ├── Passkeys (config view)
  └── Email verification (status, template preview)

SSO
  ├── Connection list with enable/disable toggle
  ├── Create/edit modal (SAML or OIDC forms)
  └── Connection detail (config, linked users, activity)

Roles & Permissions
  ├── Two-panel: role list + role detail
  ├── Permission attachment/detachment
  ├── User assignment
  └── Permission explorer / checker tool

API Keys
  ├── Key table with status badges
  ├── Create modal (with key reveal banner)
  ├── Detail panel (edit, usage log, curl snippet)
  └── Rotate / revoke actions

Audit Log
  ├── Filter bar (action, status, actor, IP, date range)
  ├── Event stream table with expand
  ├── Live mode toggle
  └── CSV export

Settings
  ├── Server info (read-only)
  ├── Email config + test
  ├── Session management (purge expired)
  ├── Audit retention (purge old)
  └── Danger zone
```

---

## Implementation Notes

- **Framework:** Svelte (embedded in Go binary via `go:embed`)
- **Build:** The dashboard is a static SPA built with Vite, output goes to an `admin/dist/` directory that gets embedded
- **Auth:** The dashboard itself authenticates using the admin API key (Bearer token stored in browser memory, never localStorage). First load prompts for the admin key.
- **API Communication:** All dashboard API calls go through `/api/v1/` on the same origin (no CORS needed)
- **Routing:** Client-side router with HTML5 history. Go serves `index.html` for all `/admin/*` paths.
- **Bundle size target:** < 150KB gzipped (Svelte compiles small)

---

*This spec covers features and UX logic only. Visual design (colors, typography, spacing, component library) is a separate phase.*
