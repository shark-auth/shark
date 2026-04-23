# Dashboard Wiring & Completion Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **REQUIRED DESIGN SKILL:** Use `/impeccable` (or `/impeccable craft`) when implementing any frontend component — login screen, new pages, UI polish, empty states. Design context lives in `.impeccable.md`. All frontend work must follow the 6 design principles defined there. Typography: Hanken Grotesk (display) + Manrope (body) + Azeret Mono (mono). Dark-only, B&W + oklch palette. Compact density. Reference: Linear. Anti-reference: Railway.

**Goal:** Wire every dashboard page to real API endpoints, build missing backend handlers, add login flow, and establish a build toolchain — turning the current mock-data prototype into a functional admin dashboard.

**Architecture:** The dashboard is a React SPA embedded in the Go binary via `go:embed`. Currently all 10 pages render mock data with zero `fetch()` calls. This plan wires each page to existing backend endpoints, builds ~8 missing Go handlers (where store methods already exist), adds an API key login flow, and introduces a Vite + TypeScript build step to replace Babel-standalone runtime compilation.

**Tech Stack:** React 18, TypeScript, Vite (build), Go (backend handlers), SQLite (storage), chi router

**Design System:** See `.impeccable.md` for full design context. Existing CSS tokens in `styles.css` are final — do not change palette or surfaces. Font migration from Inter/JetBrains Mono → Hanken Grotesk/Manrope/Azeret Mono happens in Wave 5 (polish).

---

## Audit Summary: Current State vs Specs

### What Exists (UI built, mock data only)
| Page | Component | Lines | API Endpoints Available | Wired? |
|------|-----------|-------|------------------------|--------|
| Overview | `overview.jsx` | 230 | `/admin/stats`, `/admin/stats/trends` | **NO** — uses `MOCK.stats` |
| Users | `users.jsx` | 519 | `/users` CRUD, `/users/{id}/sessions`, `/users/{id}/roles`, `/users/{id}/permissions` | **NO** |
| Sessions | `sessions.jsx` | 734 | `/admin/sessions`, `DELETE /admin/sessions/{id}` | **NO** |
| Organizations | `organizations.jsx` | 681 | `/organizations` full CRUD, members, invites, org-roles | **NO** |
| Agents | `agents.jsx` | 454 | None (Phase 6) | N/A — shell only |
| Device Flow | `device_flow.jsx` | 202 | None (Phase 6) | N/A — shell only |
| Audit Log | `audit.jsx` | 410 | `/audit-logs`, `/audit-logs/export` | **NO** |
| Applications | `applications.jsx` | 509 | `/admin/apps` full CRUD + rotate-secret | **NO** |
| API Keys | `api_keys.jsx` | 540 | `/api-keys` full CRUD + rotate | **NO** |
| Layout/Nav | `layout.jsx` | 256 | — | Renders, no data |

### Pages Spec'd But Not Built (disabled in nav)
| Page | Spec Section | Backend Ready? |
|------|-------------|---------------|
| Authentication (config) | DASHBOARD.md #6 | Partial — need `/admin/config` |
| SSO | DASHBOARD.md #7 | Yes — `/sso/connections` CRUD exists |
| Roles & Permissions | DASHBOARD.md #8 | Yes — `/roles`, `/permissions` CRUD exists |
| Webhooks | DASHBOARD.md #16 | Yes — full CRUD + deliveries + test exists |
| Dev Inbox | DASHBOARD.md #17 | Yes — `/admin/dev/emails` exists (dev mode) |
| Signing Keys | DASHBOARD.md #18 | Partial — JWKS exists, rotate endpoint missing |
| Settings | DASHBOARD.md #19 | Partial — need `/admin/health`, purge endpoints |

### Missing Backend Endpoints
| Endpoint | Store Method Exists? | Priority |
|----------|---------------------|----------|
| `GET /admin/health` | No — new handler | **High** (Overview + Settings) |
| `GET /admin/config` | No — new handler | **High** (Authentication + Settings) |
| `POST /admin/sessions/purge-expired` | Yes — `DeleteExpiredSessions()` | **Med** (Settings) |
| `POST /admin/audit-logs/purge` | Yes — `DeleteAuditLogsBefore(time)` | **Med** (Settings) |
| `GET /users/{id}/oauth-accounts` | Yes — `GetOAuthAccountsByUserID()` | **Med** (User Security tab) |
| `DELETE /users/{id}/oauth-accounts/{id}` | Yes — `DeleteOAuthAccount()` | **Med** (User Security tab) |
| `POST /admin/test-email` | No — new handler | **Low** (Settings) |
| `GET /admin/email-preview/{template}` | No — new handler | **Low** (Authentication) |
| Signing key rotate | No — new handler + store | **Low** (Signing Keys page) |

### Critical Infrastructure Missing
1. **No login flow** — No API key input, no Bearer token storage, no auth interceptor
2. **No fetch layer** — Zero `fetch()` calls in entire frontend
3. **No URL routing** — Uses `localStorage.getItem('page')`, not URL paths
4. **No build step** — Babel-standalone compiles JSX at runtime (~500ms parse penalty, no minification, no tree-shaking, no type checking)

---

## TypeScript Migration Discussion

### Current State
- 12 `.jsx` files compiled by Babel-standalone **in the browser**
- No `package.json`, no `node_modules`, no build step
- Vendor libs (React, ReactDOM, Babel) served as static files from `internal/admin/dist/vendor/`
- ~5,200 lines of JSX total

### Options

**Option A: Stay React + Add Vite + TypeScript build (RECOMMENDED)**
- Add `admin/` source directory with `package.json`, `tsconfig.json`, `vite.config.ts`
- Convert `.jsx` → `.tsx` incrementally (can do `// @ts-nocheck` initially)
- Vite builds to `internal/admin/dist/` (same embed target)
- Remove Babel-standalone, vendor React via npm
- Benefits: type safety, tree-shaking, minification, HMR in dev, source maps
- Cost: ~2-3 hours setup, then incremental conversion
- Bundle size: likely **smaller** than current (Babel-standalone alone is ~800KB)

**Option B: Rewrite in Svelte 5 per spec**
- Specs call for Svelte 5 + SvelteKit + Tailwind + shadcn-svelte
- Would discard ~5,200 lines of working UI code
- Cost: 2-4 weeks full rewrite
- Only justified if Svelte is a hard requirement for future phases

**Option C: Stay as-is (no build)**
- Keeps working but: no type safety, slow initial load, no minification, CSP requires `unsafe-eval`
- Babel-standalone eval dependency = permanent security concern
- Not viable for production

### Recommendation
**Option A.** The React components are well-structured and closely follow the spec. Converting to TypeScript + Vite preserves all existing work while fixing the Babel-standalone problem. The CSP can drop `unsafe-eval` once Babel-standalone is removed. Svelte rewrite (Option B) can be a future milestone if desired — the component structure maps cleanly.

### Migration Task (included in Wave 1 below)
1. Create `admin/` source dir with Vite + React + TypeScript config
2. Move `.jsx` files → `admin/src/components/*.tsx` (with `// @ts-nocheck` initially)
3. Add `admin/src/lib/api.ts` — typed fetch wrapper with Bearer auth
4. Vite outputs to `internal/admin/dist/`
5. Remove vendor Babel, update CSP to drop `unsafe-eval`
6. Gradually add types per-file as pages get wired

---

## Design Token Note

The specs reference `#0066ff` "Shark blue" for primary accent and suggest a blue-tinted palette. **Ignore this.** The current dark B&W CSS tokens in `styles.css` are final — oklch-based semantic colors (`--success`, `--warn`, `--danger`, `--info`, `--agent`), grayscale surfaces (`--surface-0` through `--surface-4`), and the existing font stack. No color changes needed.

---

## Design Debt: Impeccable Audit Findings

Reviewed all 12 components against `.impeccable.md` design principles. Fix these during wiring work or as a dedicated Wave 5 polish pass.

### P0 — Banned Pattern Violations (fix immediately when touching file)

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `layout.jsx` | 119 | Border-left 2px accent stripe on active nav item | Use `background: var(--surface-2)` only, or `box-shadow: inset 2px 0 0 #fff` |
| `organizations.jsx` | 89 | Border-left 2px accent stripe on active list item | Remove `borderLeft`; background state already present at line 88 |
| `applications.jsx` | 351 | Border-left 2px accent stripe on consent tip callout | Remove stripe, use background tint or inline icon |
| `agents.jsx` | 13 | Decorative `linear-gradient` on hero strip | Replace with flat `background: var(--surface-1)` |
| `sessions.jsx` | 131 | Decorative `linear-gradient` on LiveStrip | Replace with flat `var(--surface-1)` + `borderBottom` |
| `sessions.jsx` | 444 | Decorative `radial-gradient` on map background | Near-identical values (#0d0d0d → #050505) = visual noise. Use flat color |
| `device_flow.jsx` | 169 | `backdropFilter: blur(4px)` glassmorphism on approval modal | Remove blur; `rgba(0,0,0,0.6)` scrim sufficient |
| `overview.jsx` | 107-134 | Cards nested in cards (agent activity section) | Remove inner borders/backgrounds, use flat list or compact table |
| `organizations.jsx` | 273-327 | Cards nested in cards (Panel wraps card wraps Panel) | Flatten nesting — one container level only |

### P1 — Accessibility Failures (WCAG AA)

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `styles.css` | 147-151 | `.card-header` uses `--fg-dim` (#6b6b6b) at 11px — contrast ~3.2:1, needs 4.5:1 | Use `--fg-muted` (#a3a3a3) |
| `sessions.jsx` | 639 | `SessionField` label uses `--fg-dim` at 11px — below AA | Use `--fg-muted` |
| `overview.jsx` | 212 | Line height 1.35 on ≤12px dark-bg text | Increase to 1.5 |
| `users.jsx` | 211 | Line height 1.4 on ≤12px dark-bg text | Increase to 1.5 |
| `shared.jsx` | 97 | `hashColor` chroma 0.04 — avatars indistinguishable | Increase to 0.12-0.15 |

### P1 — Typography Scale

Current sizes: 10px, 10.5px, 11px, 11.5px, 12px, 12.5px, 13px — **seven sizes within a 3px band**. No hierarchy.

**Fix:** Collapse to 4-step scale with ≥1.25 ratio:
- `11px` — labels, captions, table headers (uppercase)
- `13px` — body text, table cells (base)
- `16px` — section headings, card titles
- `20px` — page titles

Kill 10px, 10.5px, 11.5px, 12px, 12.5px. Also: `--font-display` in `styles.css:29` is aliased to `--font-sans` — does nothing. Fix when font migration happens.

### P2 — Missing Copy Affordances

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `audit.jsx` | 236 | Event ID truncated to 14 chars, no copy button | Wrap in `<CopyField>` |
| `audit.jsx` | 335-337 | Event detail panel: ID + timestamp have no copy | Add `<CopyField>` to both |
| `applications.jsx` | 370-386 | Token table has no per-row revoke action | Add inline revoke button (same as api_keys pattern) |

### P2 — Identical Card Grids

| File | Lines | Issue | Fix |
|------|-------|-------|-----|
| `overview.jsx` | 107-134 | Four identical agent sub-cards (avatar+name+sparkline+tokens) | Show top agent prominently, compact list for rest |
| `organizations.jsx` | 303-306 | Three identical MiniStat cards in `repeat(3, 1fr)` | Collapse to single stacked `<div class="col">` of labeled bar rows |
| `api_keys.jsx` | 321-341 | Ten identical scope matrix cards | Add visual hierarchy — write/manage scopes get distinct treatment vs read |

### P3 — Bugs Found

| File | Line | Issue | Fix |
|------|------|-------|-----|
| `api_keys.jsx` | 276 | Date math broken — double negation always produces today's date | Replace with `new Date(k.expires).toLocaleDateString()` |
| `agents.jsx` | 336 | `users.length * 35` fabricates consent count — shows "175" but table has 5 rows | Use `agent.tokensActive` or "Showing 5 of N" |
| `api_keys.jsx` | 131 | `prod` environment chip uses `--danger` color — prod ≠ error | Use neutral chip or `--fg-dim` background |
| `applications.jsx` | 464-477 | Wizard shows "step 1 of 3" but steps 2-3 are stub divs | Either implement or collapse to single-step modal |
| `audit.jsx` | 173 | Emoji 👤 in monospace filter segment — breaks type consistency | Use plain text label or uppercase chip |

### P3 — Spacing & Rhythm

| File | Lines | Issue | Fix |
|------|-------|-------|-----|
| `users.jsx` | 262-268 | Same `marginBottom: 12` on every Field — no section grouping | Add `marginTop: 20` before credential sections (passkeys, sessions) |
| `agents.jsx` | 237-280 | Same `marginBottom: 14`/`10` on all blocks in AgentOverview | Add `marginTop: 24` before curl block — it's a different content type |
| `users.jsx` | 50-65 | Bulk action bar: Delete button not spatially separated from neutral actions | Wrap Delete in `marginLeft: auto` or add hairline divider |
| `device_flow.jsx` | 145-149 | Mixed visual grammar — bullets for some meta, icons for others | Pick one: all icons or all text bullets |

### P3 — Missing Table Features

| File | Issue | Fix |
|------|-------|-----|
| `users.jsx` | No sticky thead — headers disappear on scroll | Add `position: sticky; top: 0; z-index: 1; background: var(--surface-0)` to th |
| `sessions.jsx` | No sticky thead | Same fix |

---

## Implementation Waves

### Wave 0: Auth + Fetch Layer (foundation — everything depends on this)

> **Design:** Use `/impeccable craft` for login screen. First impression of dashboard — must nail brand feel. Full-bleed shark logo, mono API key input, minimal chrome. See `.impeccable.md` principles: "simple until you need deep", "precision, not perfection".

#### Task 0.1: Login Screen + API Key Storage

**Files:**
- Create: `internal/admin/dist/components/login.jsx`
- Modify: `internal/admin/dist/index.html` (add login gate to App)

The dashboard needs a login screen that accepts the admin API key and stores it in sessionStorage.

- [ ] **Step 1: Create login component**

```jsx
// login.jsx
function Login({ onLogin }) {
  const [key, setKey] = React.useState('');
  const [error, setError] = React.useState(null);
  const [loading, setLoading] = React.useState(false);

  const submit = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch('/api/v1/admin/stats', {
        headers: { 'Authorization': `Bearer ${key}` }
      });
      if (!res.ok) throw new Error(res.status === 401 ? 'Invalid API key' : `Error ${res.status}`);
      sessionStorage.setItem('shark_admin_key', key);
      onLogin(key);
    } catch (e) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100vh', background: 'var(--bg)' }}>
      <div style={{ width: 360, textAlign: 'center' }}>
        <SharkFullLogo/>
        <div style={{ marginTop: 32 }}>
          <input
            type="password"
            placeholder="sk_live_..."
            value={key}
            onChange={e => setKey(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && submit()}
            style={{ width: '100%', padding: '10px 12px', fontSize: 13, fontFamily: 'var(--font-mono)', background: 'var(--surface-1)', border: '1px solid var(--hairline-strong)', borderRadius: 'var(--radius)', color: 'var(--fg)' }}
          />
          {error && <div style={{ color: 'var(--danger)', fontSize: 12, marginTop: 8 }}>{error}</div>}
          <button className="btn primary" onClick={submit} disabled={loading || !key}
            style={{ width: '100%', marginTop: 12 }}>
            {loading ? 'Verifying...' : 'Sign in'}
          </button>
        </div>
        <div className="dim" style={{ fontSize: 11, marginTop: 16 }}>
          Paste your admin API key (sk_live_...). Key stored in session only — cleared on tab close.
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Add auth gate to App component in index.html**

Modify `App()` in `index.html`:

```jsx
function App() {
  const [apiKey, setApiKey] = React.useState(() => sessionStorage.getItem('shark_admin_key'));
  // ... existing state ...

  if (!apiKey) {
    return <Login onLogin={(k) => setApiKey(k)} />;
  }

  // ... existing render with apiKey passed via context or prop ...
}
```

- [ ] **Step 3: Add sign-out to TopBar**

Add sign-out button to TopBar that clears sessionStorage and resets apiKey state.

- [ ] **Step 4: Test login flow manually**

Start server with `go run ./cmd/shark`, navigate to `/admin/`, verify:
- Login screen appears
- Invalid key shows error
- Valid key (`sk_live_...` from `sharkauth.yaml`) grants access
- Closing tab clears session

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/login.jsx internal/admin/dist/index.html
git commit -m "feat(admin): add API key login flow with sessionStorage"
```

---

#### Task 0.2: API Fetch Wrapper

**Files:**
- Create: `internal/admin/dist/components/api.jsx`

Centralized fetch with Bearer auth, error handling, JSON parsing.

- [ ] **Step 1: Create API module**

```jsx
// api.jsx — fetch wrapper, loaded before page components

const API = {
  _key: () => sessionStorage.getItem('shark_admin_key'),

  async request(method, path, body) {
    const key = this._key();
    if (!key) throw new Error('Not authenticated');
    const opts = {
      method,
      headers: {
        'Authorization': `Bearer ${key}`,
        'Content-Type': 'application/json',
      },
    };
    if (body && method !== 'GET') opts.body = JSON.stringify(body);
    const res = await fetch(`/api/v1${path}`, opts);
    if (res.status === 401) {
      sessionStorage.removeItem('shark_admin_key');
      window.location.reload();
      throw new Error('Session expired');
    }
    if (!res.ok) {
      const err = await res.json().catch(() => ({ message: `HTTP ${res.status}` }));
      throw new Error(err.message || err.error || `HTTP ${res.status}`);
    }
    if (res.status === 204) return null;
    return res.json();
  },

  get(path) { return this.request('GET', path); },
  post(path, body) { return this.request('POST', path, body); },
  patch(path, body) { return this.request('PATCH', path, body); },
  del(path) { return this.request('DELETE', path); },
};

// Reusable data-fetching hook
function useAPI(path, deps = []) {
  const [data, setData] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState(null);

  const refresh = React.useCallback(() => {
    setLoading(true);
    setError(null);
    API.get(path)
      .then(setData)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [path, ...deps]);

  React.useEffect(() => { refresh(); }, [refresh]);

  return { data, loading, error, refresh };
}
```

- [ ] **Step 2: Add script tag to index.html (before page components)**

```html
<script type="text/babel" src="components/api.jsx"></script>
```

Must load after `shared.jsx` and `mock.jsx`, before `overview.jsx`.

- [ ] **Step 3: Commit**

```bash
git add internal/admin/dist/components/api.jsx internal/admin/dist/index.html
git commit -m "feat(admin): add API fetch wrapper with Bearer auth and useAPI hook"
```

---

### Wave 1: Wire Pages to Existing Endpoints (no backend changes needed)

Each task below replaces `MOCK.*` usage with real `API.get()` calls. Pattern is the same for each page:
1. Replace mock data initialization with `useAPI()` hook
2. Add loading/error states
3. Wire mutation buttons (create, update, delete) to `API.post/patch/del()`
4. Keep mock data as fallback during loading (optional graceful degradation)

#### Task 1.1: Wire Overview Page

**Files:**
- Modify: `internal/admin/dist/components/overview.jsx`

- [ ] **Step 1: Replace mock stats with real API call**

Replace line 4:
```jsx
// OLD: const s = MOCK.stats, t = MOCK.trends;
// NEW:
const { data: statsData, loading: statsLoading } = useAPI('/admin/stats');
const { data: trendsData, loading: trendsLoading } = useAPI('/admin/stats/trends?days=14');
```

Map API response shape to component expectations. The `/admin/stats` response returns:
```json
{
  "users": { "total": N, "created_last_7d": N },
  "sessions": { "active": N },
  "mfa": { "enabled": N, "total": N },
  "failed_logins_24h": N,
  "api_keys": { "active": N, "expiring_7d": N },
  "sso_connections": { "total": N, "enabled": N }
}
```

Build a mapper from API response → current `metrics` array shape.

- [ ] **Step 2: Wire recent activity feed to audit logs**

Replace `MOCK.activity` with:
```jsx
const { data: activityData, refresh: refreshActivity } = useAPI('/audit-logs?limit=20');
```

Set up 10s polling:
```jsx
React.useEffect(() => {
  const id = setInterval(refreshActivity, 10000);
  return () => clearInterval(id);
}, [refreshActivity]);
```

- [ ] **Step 3: Add loading skeleton states**

When `statsLoading`, render gray placeholder cards matching layout. When `activityData` is null, show skeleton rows.

- [ ] **Step 4: Test with running server**

Start server, login, verify Overview shows real counts from database.

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/overview.jsx
git commit -m "feat(admin): wire overview page to /admin/stats and /audit-logs"
```

---

#### Task 1.2: Wire Users Page

**Files:**
- Modify: `internal/admin/dist/components/users.jsx`

- [ ] **Step 1: Replace MOCK.users with API fetch**

```jsx
// Replace: const filtered = MOCK.users.filter(...)
// With:
const [page, setPage] = React.useState(1);
const [perPage] = React.useState(25);
const { data: usersData, loading, refresh } = useAPI(
  `/users?page=${page}&per_page=${perPage}${query ? `&search=${encodeURIComponent(query)}` : ''}`
);
const users = usersData?.users || [];
const total = usersData?.total || 0;
```

- [ ] **Step 2: Wire user detail slide-over tabs**

Profile tab — display real user fields. Wire inline edit to `API.patch('/users/' + user.id, { name, email })`.

Security tab — fetch sessions: `useAPI('/users/' + user.id + '/sessions')`. Wire revoke to `API.del('/users/' + user.id + '/sessions')` (revoke all) or `API.del('/admin/sessions/' + sessionId)` (single).

Roles tab — fetch roles: `useAPI('/users/' + user.id + '/roles')`. Wire assign: `API.post('/users/' + user.id + '/roles', { role_id })`. Wire remove: `API.del('/users/' + user.id + '/roles/' + roleId)`.

Activity tab — fetch: `useAPI('/users/' + user.id + '/audit-logs')`.

- [ ] **Step 3: Wire delete user**

```jsx
const handleDelete = async (userId) => {
  await API.del('/users/' + userId);
  refresh();
  setSelected(null);
};
```

- [ ] **Step 4: Add pagination controls**

Bottom bar: page N of M, prev/next buttons, page size selector.

- [ ] **Step 5: Test CRUD operations**

Create test user via CLI, verify appears in dashboard. Edit name, verify persists. Delete, verify removed.

- [ ] **Step 6: Commit**

```bash
git add internal/admin/dist/components/users.jsx
git commit -m "feat(admin): wire users page to /users API with CRUD + detail tabs"
```

---

#### Task 1.3: Wire Sessions Page

**Files:**
- Modify: `internal/admin/dist/components/sessions.jsx`

- [ ] **Step 1: Replace mock sessions with API**

```jsx
const { data: sessionsData, loading, refresh } = useAPI('/admin/sessions');
const sessions = sessionsData?.sessions || [];
```

- [ ] **Step 2: Wire revoke button**

```jsx
const handleRevoke = async (sessionId) => {
  await API.del('/admin/sessions/' + sessionId);
  refresh();
};
```

- [ ] **Step 3: Wire aggregate cards**

Compute from `sessions` array: total count, group by auth_method, MFA completion rate.

- [ ] **Step 4: Add live polling**

```jsx
React.useEffect(() => {
  if (!liveTail) return;
  const id = setInterval(refresh, 5000);
  return () => clearInterval(id);
}, [liveTail, refresh]);
```

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/sessions.jsx
git commit -m "feat(admin): wire sessions page to /admin/sessions with revoke"
```

---

#### Task 1.4: Wire Audit Log Page

**Files:**
- Modify: `internal/admin/dist/components/audit.jsx`

- [ ] **Step 1: Replace MOCK.audit with API**

```jsx
const buildQuery = () => {
  const params = new URLSearchParams();
  params.set('limit', '100');
  if (filters.actorType) params.set('actor_type', filters.actorType);
  if (filters.actionPrefix) params.set('action', filters.actionPrefix);
  if (filters.timeRange !== 'all') {
    const cutoffs = { '1h': 1, '24h': 24, '7d': 168 };
    const since = new Date(Date.now() - cutoffs[filters.timeRange] * 3600000);
    params.set('since', since.toISOString());
  }
  return '/audit-logs?' + params.toString();
};
const { data: auditData, loading, refresh } = useAPI(buildQuery());
```

- [ ] **Step 2: Wire live-tail to polling instead of synthetic events**

Replace the `setInterval` that generates fake events with:
```jsx
React.useEffect(() => {
  if (!liveTail) return;
  const id = setInterval(refresh, 5000);
  return () => clearInterval(id);
}, [liveTail, refresh]);
```

- [ ] **Step 3: Wire CSV export**

```jsx
const handleExport = async () => {
  const key = sessionStorage.getItem('shark_admin_key');
  const res = await fetch('/api/v1/audit-logs/export', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${key}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ /* current filter params */ }),
  });
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `audit-log-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
};
```

- [ ] **Step 4: Commit**

```bash
git add internal/admin/dist/components/audit.jsx
git commit -m "feat(admin): wire audit log to /audit-logs API with filters + export"
```

---

#### Task 1.5: Wire API Keys Page

**Files:**
- Modify: `internal/admin/dist/components/api_keys.jsx`

- [ ] **Step 1: Replace MOCK.apiKeys with API**

```jsx
const { data: keysData, loading, refresh } = useAPI('/api-keys');
const keys = keysData?.api_keys || [];
```

- [ ] **Step 2: Wire create key modal**

```jsx
const handleCreate = async (form) => {
  const result = await API.post('/api-keys', {
    name: form.name,
    scopes: form.scopes,
    expires_at: form.expiresAt || null,
  });
  // result contains the full key — show in reveal banner
  setRevealKey(result.key);
  refresh();
};
```

**Critical UX:** The raw API key is only returned on create. Show in non-dismissable banner until user confirms copy.

- [ ] **Step 3: Wire rotate + revoke**

```jsx
const handleRotate = async (keyId) => {
  const result = await API.post(`/api-keys/${keyId}/rotate`);
  setRevealKey(result.key); // new key shown once
  refresh();
};
const handleRevoke = async (keyId) => {
  await API.del(`/api-keys/${keyId}`);
  refresh();
};
```

- [ ] **Step 4: Wire edit (name, scopes)**

```jsx
const handleUpdate = async (keyId, updates) => {
  await API.patch(`/api-keys/${keyId}`, updates);
  refresh();
};
```

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/api_keys.jsx
git commit -m "feat(admin): wire API keys page with CRUD + rotate + key reveal"
```

---

#### Task 1.6: Wire Applications Page

**Files:**
- Modify: `internal/admin/dist/components/applications.jsx`

- [ ] **Step 1: Replace MOCK.applications with API**

```jsx
const { data: appsData, loading, refresh } = useAPI('/admin/apps');
const apps = appsData?.applications || [];
```

- [ ] **Step 2: Wire create app**

```jsx
const handleCreate = async (form) => {
  const result = await API.post('/admin/apps', {
    name: form.name,
    redirect_uris: form.redirectUris,
    cors_origins: form.corsOrigins,
  });
  // result.client_secret shown once
  setRevealSecret(result.client_secret);
  refresh();
};
```

- [ ] **Step 3: Wire update, rotate secret, delete**

```jsx
const handleUpdate = async (appId, updates) => {
  await API.patch(`/admin/apps/${appId}`, updates);
  refresh();
};
const handleRotateSecret = async (appId) => {
  const result = await API.post(`/admin/apps/${appId}/rotate-secret`);
  setRevealSecret(result.client_secret);
  refresh();
};
const handleDelete = async (appId) => {
  await API.del(`/admin/apps/${appId}`);
  refresh();
};
```

- [ ] **Step 4: Commit**

```bash
git add internal/admin/dist/components/applications.jsx
git commit -m "feat(admin): wire applications page to /admin/apps CRUD + rotate"
```

---

#### Task 1.7: Wire Organizations Page

**Files:**
- Modify: `internal/admin/dist/components/organizations.jsx`

- [ ] **Step 1: Replace MOCK.orgs with API**

```jsx
const { data: orgsData, loading, refresh } = useAPI('/organizations');
const orgs = orgsData?.organizations || [];
```

- [ ] **Step 2: Wire detail tabs**

Members: `useAPI('/organizations/' + orgId + '/members')`
Roles: `useAPI('/organizations/' + orgId + '/roles')`

- [ ] **Step 3: Wire mutations**

Update org: `API.patch('/organizations/' + orgId, updates)`
Delete org: `API.del('/organizations/' + orgId)`
Create org-role: `API.post('/organizations/' + orgId + '/roles', { name, description })`

- [ ] **Step 4: Commit**

```bash
git add internal/admin/dist/components/organizations.jsx
git commit -m "feat(admin): wire organizations page to /organizations API"
```

---

### Wave 2: Build Missing Pages (backend endpoints exist, need frontend components)

> **Design:** Use `/impeccable craft` for each new page component. These are brand new pages — not wiring existing UI. Must match existing dashboard density/aesthetic while following `.impeccable.md` design principles. Follow existing component patterns in `shared.jsx` (Avatar, CopyField, Sparkline, Donut, Kbd). Reuse CSS classes from `styles.css` (.card, .tbl, .chip, .btn, .avatar). No new design tokens unless absolutely necessary.

#### Task 2.1: Build Webhooks Page

**Files:**
- Create: `internal/admin/dist/components/webhooks.jsx`
- Modify: `internal/admin/dist/index.html` (add script tag + route)
- Modify: `internal/admin/dist/components/layout.jsx` (enable nav item)

- [ ] **Step 1: Create webhooks component**

Table with columns: URL, events (chips), status, last delivery, success rate, actions.
Use `useAPI('/webhooks')` for list.
Detail slide-over tabs: Config, Deliveries (`useAPI('/webhooks/' + id + '/deliveries')`), Test.

- [ ] **Step 2: Wire CRUD**

Create: `API.post('/webhooks', { url, events, secret })` — auto-gen secret if blank.
Update: `API.patch('/webhooks/' + id, updates)`.
Delete: `API.del('/webhooks/' + id)`.
Test: `API.post('/webhooks/' + id + '/test')`.

- [ ] **Step 3: Wire deliveries tab**

```jsx
const { data: deliveries, refresh } = useAPI(`/webhooks/${webhookId}/deliveries`);
// Show: event, timestamp, status code, attempts, duration
// Expand: request/response body
```

- [ ] **Step 4: Enable nav item in layout.jsx**

Remove `disabled: true` from webhooks nav entry.

- [ ] **Step 5: Add route in index.html App component**

Add `webhooks: Webhooks` to Page mapping object.

- [ ] **Step 6: Commit**

```bash
git add internal/admin/dist/components/webhooks.jsx internal/admin/dist/index.html internal/admin/dist/components/layout.jsx
git commit -m "feat(admin): add webhooks page with CRUD + deliveries + test"
```

---

#### Task 2.2: Build Roles & Permissions Page

**Files:**
- Create: `internal/admin/dist/components/rbac.jsx`
- Modify: `internal/admin/dist/index.html` (add script tag + route)
- Modify: `internal/admin/dist/components/layout.jsx` (enable nav item)

- [ ] **Step 1: Create two-pane RBAC component**

Left: roles list from `useAPI('/roles')`. Show name, description snippet, user count.
Right: role detail. Inline edit name/description via `API.patch('/roles/' + id, ...)`.
Permissions section: table from role's permissions with detach button.
Users section: list from `useAPI('/roles/' + id + '/users')` (if endpoint exists) or derive from user-roles.

- [ ] **Step 2: Wire role CRUD**

Create: `API.post('/roles', { name, description })`.
Delete: `API.del('/roles/' + id)`.
Create permission: `API.post('/permissions', { action, resource })`.
Attach permission: `API.post('/roles/' + id + '/permissions', { permission_id })`.

- [ ] **Step 3: Wire "Check Permission" tool**

```jsx
const checkPermission = async (userId, action, resource) => {
  const result = await API.post('/auth/check', { user_id: userId, action, resource });
  return result.allowed;
};
```

- [ ] **Step 4: Enable nav + route**

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/rbac.jsx internal/admin/dist/index.html internal/admin/dist/components/layout.jsx
git commit -m "feat(admin): add roles & permissions page with two-pane layout"
```

---

#### Task 2.3: Build SSO Connections Page

**Files:**
- Create: `internal/admin/dist/components/sso.jsx`
- Modify: `internal/admin/dist/index.html` + `layout.jsx`

- [ ] **Step 1: Create SSO component**

Table: name, type (SAML/OIDC badge), domain, status toggle, users count, created, actions.
Use `useAPI('/sso/connections')`.

- [ ] **Step 2: Create/edit form (slide-over)**

Dynamic form based on type (OIDC vs SAML):
- OIDC: name, domain, issuer URL, client ID, client secret, enabled toggle
- SAML: name, domain, IdP URL, IdP certificate, SP entity ID (auto), ACS URL (auto), enabled

Wire create: `API.post('/sso/connections', form)`.
Wire update: `API.patch('/sso/connections/' + id, form)` — note: backend uses PUT, check actual method.
Wire delete: `API.del('/sso/connections/' + id)`.

- [ ] **Step 3: Enable nav + route**

- [ ] **Step 4: Commit**

```bash
git add internal/admin/dist/components/sso.jsx internal/admin/dist/index.html internal/admin/dist/components/layout.jsx
git commit -m "feat(admin): add SSO connections page with SAML/OIDC forms"
```

---

#### Task 2.4: Build Dev Inbox Page

**Files:**
- Create: `internal/admin/dist/components/dev_inbox.jsx`
- Modify: `internal/admin/dist/index.html` + `layout.jsx`

- [ ] **Step 1: Create dev inbox component**

Conditionally rendered (only visible when server is in dev mode — detect via stats or health endpoint).

List: `useAPI('/admin/dev/emails')`.
Columns: to, subject, type (magic_link/verify/password_reset), received.
Detail: HTML preview + raw source toggle.
Clear all: `API.del('/admin/dev/emails')`.

- [ ] **Step 2: Add "Open magic link" shortcut**

Parse email body for magic link URL, render clickable button.

- [ ] **Step 3: Add banner**

"Dev inbox captures all outgoing mail. Switch email provider before production."

- [ ] **Step 4: Enable nav (conditionally) + route**

Show nav item only when dev mode detected.

- [ ] **Step 5: Commit**

```bash
git add internal/admin/dist/components/dev_inbox.jsx internal/admin/dist/index.html internal/admin/dist/components/layout.jsx
git commit -m "feat(admin): add dev inbox page for captured emails (dev mode)"
```

---

### Wave 3: Missing Backend Endpoints

#### Task 3.1: Add GET /admin/health Endpoint

**Files:**
- Modify: `internal/api/admin_stats_handlers.go` (or create `internal/api/admin_health_handler.go`)
- Modify: `internal/api/router.go` (register route)

- [ ] **Step 1: Write handler test**

```go
func TestHandleAdminHealth(t *testing.T) {
    s := setupTestServer(t)
    req := adminRequest(t, "GET", "/api/v1/admin/health", nil)
    rr := httptest.NewRecorder()
    s.Router.ServeHTTP(rr, req)
    require.Equal(t, 200, rr.Code)
    var body map[string]interface{}
    require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
    require.Contains(t, body, "version")
    require.Contains(t, body, "db_size_bytes")
    require.Contains(t, body, "uptime_seconds")
}
```

- [ ] **Step 2: Implement handler**

```go
func (s *Server) handleAdminHealth(w http.ResponseWriter, r *http.Request) {
    // DB size: query SQLite PRAGMA page_count * page_size
    // Version: from build-time variable or config
    // Uptime: time.Since(s.startTime)
    // Config summary: which providers enabled, SMTP status, JWT mode, dev mode
}
```

Returns:
```json
{
  "version": "0.x.y",
  "uptime_seconds": 3600,
  "db_size_bytes": 1048576,
  "last_migration": "20260401_xyz",
  "config": {
    "dev_mode": true,
    "jwt_mode": false,
    "smtp_configured": true,
    "oauth_providers": ["google", "github"],
    "sso_connections": 2,
    "cors_origins": ["http://localhost:3000"]
  }
}
```

- [ ] **Step 3: Register in router.go**

Add `r.Get("/health", s.handleAdminHealth)` inside the `/admin` route group.

- [ ] **Step 4: Run tests, commit**

```bash
go test ./internal/api/ -run TestHandleAdminHealth -v
git commit -m "feat(api): add GET /admin/health endpoint for dashboard diagnostics"
```

---

#### Task 3.2: Add GET /admin/config Endpoint

**Files:**
- Create or modify: `internal/api/admin_config_handler.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Write handler**

Exposes non-sensitive config values: base_url, app_url, port, auth settings (min password length, lockout threshold/duration, session lifetime), enabled providers, MFA issuer, passkey RP config.

**Never returns:** secret key, API key hashes, OAuth client secrets, SMTP passwords.

- [ ] **Step 2: Register route**

`r.Get("/config", s.handleAdminConfig)` in admin group.

- [ ] **Step 3: Test + commit**

```bash
git commit -m "feat(api): add GET /admin/config endpoint for dashboard settings view"
```

---

#### Task 3.3: Add Purge Endpoints

**Files:**
- Create: `internal/api/admin_purge_handlers.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement session purge**

```go
func (s *Server) handlePurgeExpiredSessions(w http.ResponseWriter, r *http.Request) {
    count, err := s.Store.DeleteExpiredSessions(r.Context())
    if err != nil { /* error response */ }
    json.NewEncoder(w).Encode(map[string]int64{"deleted": count})
}
```

Route: `r.Post("/sessions/purge-expired", s.handlePurgeExpiredSessions)`

- [ ] **Step 2: Implement audit log purge**

```go
func (s *Server) handlePurgeAuditLogs(w http.ResponseWriter, r *http.Request) {
    var body struct { Before string `json:"before"` }
    // parse body.Before as time.Time
    count, err := s.Store.DeleteAuditLogsBefore(r.Context(), before)
    // respond with count
}
```

Route: `r.Post("/audit-logs/purge", s.handlePurgeAuditLogs)`

- [ ] **Step 3: Test both**

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(api): add purge endpoints for expired sessions and old audit logs"
```

---

#### Task 3.4: Add User OAuth Accounts Endpoints

**Files:**
- Create: `internal/api/user_oauth_handlers.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Implement list OAuth accounts**

```go
func (s *Server) handleListUserOAuthAccounts(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "id")
    accounts, err := s.Store.GetOAuthAccountsByUserID(r.Context(), userID)
    // return JSON array
}
```

Route: `r.Get("/{id}/oauth-accounts", s.handleListUserOAuthAccounts)` inside users group.

- [ ] **Step 2: Implement unlink OAuth account**

```go
func (s *Server) handleDeleteUserOAuthAccount(w http.ResponseWriter, r *http.Request) {
    accountID := chi.URLParam(r, "oauthId")
    err := s.Store.DeleteOAuthAccount(r.Context(), accountID)
    // return 204
}
```

Route: `r.Delete("/{id}/oauth-accounts/{oauthId}", s.handleDeleteUserOAuthAccount)`

- [ ] **Step 3: Test + commit**

```bash
git commit -m "feat(api): add user OAuth accounts list + unlink endpoints"
```

---

### Wave 4: Build Settings + Authentication Config Pages

> **Design:** Use `/impeccable craft` for Settings, Authentication, and Signing Keys pages. Settings = read-heavy config display — must feel informative not overwhelming. Danger zone needs clear visual weight without being dramatic. Authentication config = card-per-provider layout. Follow `.impeccable.md` principle: "control without complexity."

#### Task 4.1: Build Settings Page

**Files:**
- Create: `internal/admin/dist/components/settings.jsx`
- Modify: `internal/admin/dist/index.html` + `layout.jsx`

- [ ] **Step 1: Create settings component**

Sections per spec:
- **Server:** base_url (copy), port (read-only), secret status, CORS origins. Source: `/admin/config`.
- **Email:** provider, from address, "Send test email" button (POST `/admin/test-email` if exists, otherwise note as unavailable).
- **Session:** lifetime (read-only), active count (from `/admin/stats`), "Purge expired" button → `API.post('/admin/sessions/purge-expired')`.
- **Audit retention:** row count, DB size (from `/admin/health`), "Purge before date" → date picker + `API.post('/admin/audit-logs/purge', { before })`.
- **Danger zone:** styled red border. "Delete all users" requires typing `DELETE ALL USERS`.

- [ ] **Step 2: Enable nav + route**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(admin): add settings page with purge actions + server info"
```

---

#### Task 4.2: Build Authentication Config Page

**Files:**
- Create: `internal/admin/dist/components/authentication.jsx`
- Modify: `internal/admin/dist/index.html` + `layout.jsx`

- [ ] **Step 1: Create auth config component**

Read-only config display from `/admin/config`:
- **Password Policy:** min length, complexity, lockout threshold/duration
- **OAuth Providers:** card per configured provider — status, client_id (masked), scopes, redirect URL
- **Magic Links:** token lifetime, redirect URL
- **Passkeys:** RP name, RP ID, origin, attestation, UV setting
- **Email Verification:** SMTP status
- **JWT Mode:** cookie vs JWT radio (read-only for now, wire toggle later)

All data sourced from `/admin/config` response.

- [ ] **Step 2: Enable nav + route**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(admin): add authentication config page (read-only settings view)"
```

---

#### Task 4.3: Build Signing Keys Page

**Files:**
- Create: `internal/admin/dist/components/signing_keys.jsx`
- Modify: `internal/admin/dist/index.html` + `layout.jsx`

- [ ] **Step 1: Create signing keys component**

Fetch JWKS from `/.well-known/jwks.json` (public, no auth needed).
Display table: kid, algorithm, use, created.
"Download JWKS" button: link to `/.well-known/jwks.json`.
"Rotate" button: disabled until backend endpoint exists, show tooltip "Rotation endpoint coming soon".

- [ ] **Step 2: Enable nav + route**

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(admin): add signing keys page displaying JWKS"
```

---

### Wave 5: Polish + Infrastructure

> **Design:** Use `/impeccable` for toast system design and empty state components. Toasts must be minimal — no oversized icons or excessive padding. Empty states must teach without patronizing (principle: "simple until you need deep"). Font migration to Hanken Grotesk / Manrope / Azeret Mono happens here — update Google Fonts link in `index.html` and CSS variables in `styles.css`.

#### Task 5.1: URL-Based Routing

**Files:**
- Modify: `internal/admin/dist/index.html` (App component)

- [ ] **Step 1: Replace localStorage routing with URL-based**

```jsx
function App() {
  const getPage = () => {
    const path = window.location.pathname.replace(/^\/admin\/?/, '') || 'overview';
    return path.split('/')[0] || 'overview';
  };

  const [page, setPageState] = React.useState(getPage);

  const setPage = (p) => {
    window.history.pushState(null, '', '/admin/' + p);
    setPageState(p);
  };

  React.useEffect(() => {
    const handler = () => setPageState(getPage());
    window.addEventListener('popstate', handler);
    return () => window.removeEventListener('popstate', handler);
  }, []);

  // ... rest unchanged, but pass setPage to Sidebar
}
```

Now `/admin/users`, `/admin/audit`, etc. work as direct URLs and browser back/forward works.

- [ ] **Step 2: Commit**

```bash
git commit -m "feat(admin): switch to URL-based routing with history.pushState"
```

---

#### Task 5.2: Toast Notification System

**Files:**
- Create: `internal/admin/dist/components/toast.jsx`
- Modify: `internal/admin/dist/index.html`

- [ ] **Step 1: Create toast component + context**

```jsx
const ToastContext = React.createContext();

function ToastProvider({ children }) {
  const [toasts, setToasts] = React.useState([]);
  const add = (type, message, opts = {}) => {
    const id = Math.random().toString(36).slice(2);
    setToasts(prev => [...prev, { id, type, message, ...opts }]);
    if (!opts.sticky) setTimeout(() => remove(id), opts.duration || 5000);
  };
  const remove = (id) => setToasts(prev => prev.filter(t => t.id !== id));
  return (
    <ToastContext.Provider value={{ success: (m) => add('success', m), error: (m) => add('error', m), info: (m) => add('info', m) }}>
      {children}
      <div style={{ position: 'fixed', bottom: 16, right: 16, zIndex: 50, display: 'flex', flexDirection: 'column', gap: 8 }}>
        {toasts.slice(-3).map(t => (
          <div key={t.id} className={`chip ${t.type === 'error' ? 'danger' : t.type}`}
            style={{ padding: '8px 12px', fontSize: 12, animation: 'slideIn 160ms ease-out' }}>
            {t.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

function useToast() { return React.useContext(ToastContext); }
```

- [ ] **Step 2: Integrate with API wrapper**

On successful mutations: `toast.success('User updated')`.
On errors: `toast.error(err.message)`.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(admin): add toast notification system"
```

---

#### Task 5.3: Empty States for Phase 6+ Shell Pages

**Files:**
- Create: `internal/admin/dist/components/empty_shell.jsx`

- [ ] **Step 1: Create reusable empty state component**

```jsx
function EmptyShell({ title, phase, description, docsUrl }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', textAlign: 'center', padding: 40 }}>
      <div style={{ maxWidth: 400 }}>
        <div style={{ fontSize: 18, fontWeight: 600 }}>{title}</div>
        <div className="dim" style={{ marginTop: 8, fontSize: 13, lineHeight: 1.5 }}>{description}</div>
        <div className="chip ghost" style={{ marginTop: 16, fontSize: 11 }}>Phase {phase}</div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Wire remaining disabled nav items**

Consents, Tokens, Vault → Phase 6 shells.
API Explorer, Session Debugger, Event Schemas → Phase 5 shells.
Enable nav items (remove `disabled: true`) but route to `EmptyShell`.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(admin): add empty shell pages for Phase 5-6 nav items"
```

---

### Wave 6 (Future): Vite + TypeScript Build

> This wave converts from Babel-standalone to a proper build pipeline. Can be deferred — listed here for planning purposes. See "TypeScript Migration Discussion" section above for rationale.

#### Task 6.1: Scaffold Vite + React + TypeScript Project

**Files:**
- Create: `admin/package.json`
- Create: `admin/tsconfig.json`
- Create: `admin/vite.config.ts`
- Create: `admin/src/main.tsx` (entry point)

- [ ] **Step 1: Initialize project**

```bash
cd admin
npm create vite@latest . -- --template react-ts
```

- [ ] **Step 2: Configure Vite output**

```ts
// vite.config.ts
export default defineConfig({
  build: {
    outDir: '../internal/admin/dist',
    emptyOutDir: true,
  },
  base: '/admin/',
});
```

- [ ] **Step 3: Move component files**

Rename `*.jsx` → `*.tsx`. Add `// @ts-nocheck` at top of each initially.
Move `styles.css` → `admin/src/styles.css`.
Import in `main.tsx`.

- [ ] **Step 4: Remove Babel-standalone from vendor/**

Update CSP in `admin.go` — remove `'unsafe-eval'`.

- [ ] **Step 5: Add build script to Makefile or CI**

```makefile
admin-build:
    cd admin && npm install && npm run build
```

- [ ] **Step 6: Gradually add TypeScript types**

Create `admin/src/types.ts` with API response shapes.
Remove `// @ts-nocheck` per file as types are added.

---

## Summary: Execution Order

| Wave | Tasks | Backend Changes | Frontend Changes | Dependencies |
|------|-------|----------------|-----------------|-------------|
| **0** | Login + API wrapper | None | 2 new files | None |
| **1** | Wire 7 existing pages | None | 7 modified files | Wave 0 |
| **2** | Build 4 new pages | None | 4 new files | Wave 0 |
| **3** | 4 missing endpoints | 4 new Go handlers | None | None (parallel with Wave 1-2) |
| **4** | Settings + Auth + Signing Keys | Depends on Wave 3 | 3 new files | Wave 0 + 3 |
| **5** | URL routing, toasts, shells | None | 3 new files | Wave 0 |
| **6** | Vite + TypeScript | CSP update | Full restructure | After Waves 0-5 stable |

**Wave 3 (backend) can run in parallel with Waves 1-2 (frontend wiring).**

Total new frontend files: ~12
Total modified frontend files: ~10
Total new Go handlers: ~6-8
Estimated mock data lines removed: ~500 (mock.jsx can be deleted entirely when all pages wired)

---

## Appendix: API Response Shape Reference

Needed when wiring — the frontend needs to know what the backend returns.

### GET /admin/stats
```json
{
  "users": { "total": 142, "created_last_7d": 12 },
  "sessions": { "active": 89 },
  "mfa": { "enabled": 34, "total": 142 },
  "failed_logins_24h": 7,
  "api_keys": { "active": 5, "expiring_7d": 1 },
  "sso_connections": { "total": 2, "enabled": 2 }
}
```

### GET /admin/stats/trends
```json
{
  "signups": [{ "date": "2026-04-01", "count": 3 }, ...],
  "auth_methods": { "password": 0.6, "oauth": 0.25, "passkey": 0.1, "magic_link": 0.05 }
}
```

### GET /users
```json
{
  "users": [{ "id": "usr_...", "name": "...", "email": "...", "email_verified": true, "mfa_enabled": false, "created_at": "...", "updated_at": "..." }],
  "total": 142,
  "page": 1,
  "per_page": 25
}
```

### GET /audit-logs
```json
{
  "audit_logs": [{ "id": "aud_...", "action": "user.login", "actor_type": "user", "actor_id": "usr_...", "target_type": "session", "target_id": "ses_...", "ip": "...", "user_agent": "...", "metadata": {}, "status": "success", "created_at": "..." }],
  "total": 1500
}
```

> **Note:** Verify these shapes against actual handler code before wiring. Read the handler's `json.NewEncoder` output struct to confirm field names.
