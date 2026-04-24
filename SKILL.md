# SKILL.md — Dashboard integration surface

Reference guide for agents extending the SharkAuth admin dashboard.

---

## How admin/ is served

`shark serve` (or `shark serve --dev`) builds the Go binary which embeds
`internal/admin/dist/` via `//go:embed`. The Vite build (`admin/`) writes
its output to `internal/admin/dist/`. The SPA entry point is served at
`/admin/*` with a catch-all that returns `index.html` for any unmatched
sub-path (client-side routing handles the rest).

Vite config: `admin/vite.config.ts`. Output dir: `../internal/admin/dist`.

The hosted (non-admin) paywall page is bundled separately as `hosted.html`
and served at `/paywall/{slug}` by `internal/api/hosted_handlers.go`.

---

## Where to add a new dashboard page

1. Create `admin/src/components/my_page.tsx`. Export a default React component
   named after the page. Use `// @ts-nocheck` (all existing files do).

2. Register in `admin/src/components/App.tsx`:
   ```ts
   import { MyPage } from './my_page'
   // inside the Page map:
   'my-page': MyPage,
   ```

3. Add a nav entry in `admin/src/components/layout.tsx` inside the `NAV`
   array under the appropriate group:
   ```ts
   { id: 'my-page', label: 'My Page', icon: 'Settings' }
   ```
   Icons come from `admin/src/components/shared.tsx` → `Icon.*`.

4. Optionally add a `TopBar` title in the `pageTitle` map in `layout.tsx`.

5. Run `cd admin && npm run build` to verify the TypeScript compile and
   Vite bundle.

---

## How the admin HTTP client authenticates

The admin key is stored in `localStorage.getItem('shark_admin_key')` and
injected by the `API` object in `admin/src/components/api.tsx`:

```ts
API.get('/admin/proxy/rules/db')   // → GET /api/v1/admin/proxy/rules/db
API.post('/admin/proxy/rules/db', body)
API.patch('/admin/proxy/rules/db/' + id, patch)
API.del('/admin/proxy/rules/db/' + id)
```

Every request sets `Authorization: Bearer <key>`. A 401 clears the stored
key and emits `shark-auth-expired` so `App.tsx` can redirect to login.

For reactive data, use the `useAPI(path)` hook (also in `api.tsx`):
```ts
const { data, loading, error, refresh } = useAPI('/admin/proxy/rules/db')
```

---

## Where the proxy / lifecycle / tiers / branding UIs live

| Feature | File | Notes |
|---|---|---|
| Proxy rules CRUD | `admin/src/components/proxy_config.tsx` | `Proxy` component; `RuleModal`, `ImportModal`, `DeleteConfirm` |
| Proxy lifecycle toggle | `proxy_config.tsx` → `LifecycleBar` | Polls `GET /api/v1/admin/proxy/lifecycle` every 5s |
| Proxy wizard (first-time setup) | `admin/src/components/proxy_wizard.tsx` | Shown when lifecycle returns 404 |
| Per-user billing tier | `admin/src/components/users.tsx` → `TierSection` | In `ProfileTab` of `UserSlideover` |
| Design tokens editor | `admin/src/components/branding.tsx` → `DesignTokensEditor` | BrandStudio → "Design Tokens" → "Token editor" |
| Paywall preview | `branding.tsx` → `PaywallPreviewEditor` | BrandStudio → "Design Tokens" → "Paywall preview" |
| Brand visuals, palette, typography | `branding.tsx` (entire file) | Full BrandStudio 3-column editor |

---

## How to preview the paywall locally

```bash
# Start the server in dev mode
./shark serve --dev

# Open the paywall for an app slug
open http://localhost:8080/paywall/<app-slug>?tier=pro&return=http://localhost:8080/admin
```

The paywall renders an inline HTML template injected with branding CSS vars
from `GET /api/v1/admin/branding`. Design tokens in `branding.design_tokens`
are embedded as `:root { --<key>: <value>; }` overrides.

From the dashboard: Branding → Design Tokens → Paywall preview.

---

## API endpoints used by the dashboard (proxy v1.5)

| Endpoint | Purpose |
|---|---|
| `GET  /api/v1/admin/proxy/lifecycle` | Proxy state (running/stopped/reloading) |
| `POST /api/v1/admin/proxy/start\|stop\|reload` | Lifecycle transitions |
| `GET  /api/v1/admin/proxy/rules/db` | List all DB rules |
| `POST /api/v1/admin/proxy/rules/db` | Create rule |
| `PATCH /api/v1/admin/proxy/rules/db/{id}` | Update rule |
| `DELETE /api/v1/admin/proxy/rules/db/{id}` | Delete rule |
| `POST /api/v1/admin/proxy/rules/import` | Import YAML bundle |
| `PATCH /api/v1/admin/users/{id}/tier` | Set user billing tier |
| `GET  /api/v1/admin/branding` | Read branding (includes design_tokens) |
| `PATCH /api/v1/admin/branding/design-tokens` | Write design token JSON |

Full specs: `docs/proxy_v1_5/api/` and `docs/proxy_v1_5/integration/frontend_wiring_notes.md`.

---

## Design system rules

- No external component libraries. Use only CSS classes and inline styles.
- Existing CSS class vocabulary: `.btn`, `.btn.primary`, `.btn.ghost`, `.btn.sm`, `.btn.icon`,
  `.chip`, `.chip.success`, `.chip.warn`, `.chip.error`, `.chip.solid`, `.chip.ghost`,
  `.card`, `.tbl`, `.row`, `.col`, `.mono`, `.faint`, `.muted`, `.seg`.
- CSS custom properties: `--fg`, `--fg-muted`, `--fg-dim`, `--fg-faint`, `--bg`,
  `--surface-0..3`, `--hairline`, `--hairline-strong`, `--success`, `--warn`, `--error`,
  `--font-display`, `--font-mono`.
- Match hairline style: `'1px solid var(--hairline)'`.
- Icon primitives: `import { Icon } from './shared'` — `<Icon.Plus width={12}/>`.
- Toast: `import { useToast } from './toast'` — `toast.success(...)`, `toast.error(...)`.
