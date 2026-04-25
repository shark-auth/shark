# SharkAuth Branding + Hosted Auth Pages + React Components — Design Spec

**Date:** 2026-04-20
**Author:** brainstorming session (Claude Opus 4.7 + user)
**Status:** APPROVED, ready for implementation plan
**Branch at spec time:** `claude/admin-vendor-assets-fix`
**Invocation trigger:** user wants mail builder + branding tab + hosted auth + embeddable React components, explicitly planning-only, no code yet
**Implementation skill after this:** `/impeccable` for component building, then writing-plans for execution plan

---

## Bus-factor log — every decision made in brainstorm

If this context is lost, everything below is derived from these 11 user-confirmed decisions. Re-deriving the spec from scratch must honor these.

| # | Decision point | User chose | Rationale from conversation |
|---|----------------|-----------|------------------------------|
| 1 | Scope phasing (A branding / B hosted / C components: which ship when) | All three pre-launch, in order A → B+C, scheduled AFTER phase-7 moat work. User accepts launch slip past April 27. | User explicitly said "nah, all pre-launch anyway. just put it after phase 7." |
| 2 | V1 branding scope | **B Standard** — 7 fields: logo + primary color + secondary color + font family (3 presets: Manrope/Inter/IBM Plex) + footer text + email-from name + email-from address | Enough to look custom, fast enough to ship, extensible later without schema break |
| 3 | Branding storage shape | **Shape 2** — Global + per-app override via `applications.branding JSON`. Fallback chain: app → global → hardcoded default | Day-1 self-hosters run single instance; per-app override for edge cases; extends to org-level later without migration break |
| 4 | Mail template editing model | **A** Structured fields — Go `html/template` with fixed layouts, admin edits COPY only (subject, header, body paragraphs[], CTA text+URL, footer). Zero HTML injection risk. | Ships fastest, covers 100% of "my colors + my copy," impossible to break layout |
| 5 | Mail preview UX | **A** Side-by-side — editor left 40%, live iframe preview right 60%, debounced 300ms. Industry standard (Clerk/Stripe). Backend POST to `/admin/email-templates/{id}/preview` with draft config, returns rendered HTML for iframe srcdoc. | Same origin, no CORS; standard pattern; users see result while editing |
| 6 | Logo hosting | **B** Direct upload — admin uploads PNG/SVG/JPG ≤1MB, shark stores `data/assets/branding/{sha}.{ext}`, serves at `/assets/branding/{sha}.{ext}` with cache headers | Self-hoster shouldn't need external image hosting; shark owns lifecycle |
| 7 | Admin UI organization | **A** Single "Branding" nav item with subtabs: `Visuals | Email Templates | Integrations` | Clerk pattern; fewer nav items; scales to more subtabs later |
| 8 | V1 email templates | 4 existing (`magic_link`, `password_reset`, `verify_email`, `organization_invitation`) + NEW `welcome` = 5 total | Verified against `internal/email/templates/` actual contents |
| 9 | Hosted pages architecture | **B** React SPA at `/hosted/<app-slug>/*`, separate Vite entry embedded via go:embed, shares design system with admin + npm package C | One codebase, three surfaces (hosted, admin preview, npm components); 3x reuse value |
| 10 | Multi-tenant routing | **C (phased)** — path-based V1 (`/hosted/<app-slug>/login`), subdomain later via W15 multi-listener | Subdomain needs wildcard DNS + TLS; path works everywhere day-1 |
| 11 | NPM package structure | **C** Monorepo single-publish — `@sharkauth/react` packages `core/hooks/components` internally, single npm install, tree-shakeable. Future split into `@shark-auth/core` + framework packages without breaking consumers. | User chose `shark-auth` NPM scope |
| 12 | Per-app integration model | Single `integration_mode` enum (`hosted|components|proxy|custom`) as PRIMARY login surface. Session widgets (`<UserButton/>`, `<SignedIn/>`, `<OrganizationSwitcher/>`) always available regardless. Proxy fallback via `proxy_login_fallback` sub-config. | Cleaner mental model: one picker for "what's my login UI?" + widgets just work |

### Open-question answers (from end of brainstorm, before spec write):

| # | Question | User answer |
|---|----------|-------------|
| OQ1 | Welcome email trigger | **On email verification** (not signup) — fires after user clicks verify link and email is confirmed |
| OQ2 | Logo upload max size | **1MB hard cap** |
| OQ3 | Color picker UX | **Full picker component** (HSL slider + palette), not just hex input |
| OQ4 | Test-send email recipient validation | **Any address** — no admin-verified restriction |
| OQ5 | JWT storage in components | **sessionStorage** (per-tab, cleared on tab close) |
| OQ6 | NPM scope | **`@sharkauth/react`** |
| OQ7 | i18n | **English only V1**, no translation scaffold yet |

---

## Executive summary

Three coupled features, single unified design:

**A. Branding tab + mail builder** — new dashboard "Branding" page with three subtabs (Visuals, Email Templates, Integrations). Replaces the `empty_shell.tsx` Branding ph:9 stub. Admin uploads logo, picks colors, chooses font preset, writes footer + email-from, edits the 5 email templates via structured fields with live side-by-side preview. Per-app override JSON for multi-app customization.

**B. Hosted auth pages** — new React SPA served at `/hosted/<app-slug>/{login|signup|magic|passkey|mfa|verify|error}`. Apps with `integration_mode='hosted'` configure a "Sign in" button that redirects here; shark renders the full auth flow; on success redirects back to the app's configured URI with OAuth code + state. Path-based routing V1, subdomain via W15 multi-listener later.

**C. `@sharkauth/react` NPM package** — single published package with internal monorepo structure. Exports `<SignIn/>`, `<SignUp/>`, `<UserButton/>`, `<SignedIn/>`, `<SignedOut/>`, `<MFAChallenge/>`, `<PasskeyButton/>`, `useAuth()`, `useUser()`, `<SharkProvider/>`. Apps with `integration_mode='components'` install this, drop widgets in their JSX, SDK handles OAuth+PKCE flow + JWT refresh against shark. Framework dropdown in dashboard generates copy-paste snippets (React v1; Vue/Svelte/Solid/Angular placeholder "coming soon").

**Shared foundation:** One design system in `admin/src/design/`. Tokens (colors, spacing, type scale), primitives (Button, Input, Card, FormField), composed (SignInForm, SignUpForm, MFAForm). Consumed by: dashboard Branding preview, hosted pages SPA, npm package.

**Integration modes (per-application):**
- `hosted` — shark renders login at `/hosted/<slug>/*`, app redirects to it
- `components` — app installs `@sharkauth/react`, embeds widgets inline
- `proxy` — shark reverse-proxy enforces auth before requests reach app; `proxy_login_fallback` routes unauthed hits to `hosted` OR `custom_url`
- `custom` — API-only, user hand-rolls everything

Session widgets (`<UserButton/>`, `<SignedIn/>`, `<OrganizationSwitcher/>`) available regardless of mode.

**Total estimate:** ~13 working days solo. Phase A (4d) + Phase B (4d) + Phase C (5d). Scheduled after phase-7 launch moat work; user accepts launch slip past April 27.

---

## Section 1 — Data model

### New table: `branding`

```sql
-- migrations/00017_branding_and_email_templates.sql (next sequence after 00016)
-- +goose Up

CREATE TABLE branding (
  id TEXT PRIMARY KEY,                       -- 'global' for V1; future: org_id values
  scope TEXT NOT NULL DEFAULT 'global',      -- 'global' | 'org' (future)
  logo_url TEXT,                             -- points to /assets/branding/{sha}.{ext}
  logo_sha TEXT,                             -- hash for cache-busting
  primary_color TEXT DEFAULT '#7c3aed',      -- hex or oklch string
  secondary_color TEXT DEFAULT '#1a1a1a',
  font_family TEXT DEFAULT 'manrope',        -- enum: 'manrope' | 'inter' | 'ibm_plex'
  footer_text TEXT DEFAULT '',
  email_from_name TEXT DEFAULT 'SharkAuth',
  email_from_address TEXT DEFAULT 'noreply@example.com',
  updated_at TEXT NOT NULL
);

INSERT INTO branding (id, scope, updated_at) VALUES ('global', 'global', CURRENT_TIMESTAMP);
```

### Extend `applications` table

```sql
ALTER TABLE applications ADD COLUMN integration_mode TEXT NOT NULL DEFAULT 'custom';
-- enum: 'hosted' | 'components' | 'proxy' | 'custom'

ALTER TABLE applications ADD COLUMN branding_override TEXT;
-- nullable JSON, same keys as branding.* fields, any subset (null fields fall through to global)

ALTER TABLE applications ADD COLUMN proxy_login_fallback TEXT DEFAULT 'hosted';
-- when integration_mode = 'proxy': where unauthed requests get redirected
-- 'hosted' | 'custom_url'

ALTER TABLE applications ADD COLUMN proxy_login_fallback_url TEXT;
-- when proxy_login_fallback = 'custom_url'
```

### New table: `email_templates`

```sql
CREATE TABLE email_templates (
  id TEXT PRIMARY KEY,                       -- 'magic_link' | 'password_reset' | 'verify_email' | 'organization_invitation' | 'welcome'
  subject TEXT NOT NULL,
  preheader TEXT DEFAULT '',                 -- short summary shown in email client inbox preview
  header_text TEXT NOT NULL,                 -- H1 at top of email body
  body_paragraphs TEXT NOT NULL DEFAULT '[]',-- JSON array of strings, each = one <p>
  cta_text TEXT DEFAULT '',                  -- button text, empty = no button
  cta_url_template TEXT DEFAULT '',          -- Go template string, e.g. '{{.Link}}'
  footer_text TEXT DEFAULT '',
  updated_at TEXT NOT NULL
);

-- Seed all 5 V1 templates with current hardcoded content ported to structured fields
-- Performed by Go code on first startup (idempotent via INSERT OR IGNORE)
```

**Welcome email trigger** (per OQ1): fires in authflow after `verify_email.html` magic link is clicked AND user's `email_verified` flag transitions false→true. Not on signup completion.

### Storage resolution helper (Go)

```go
// internal/storage/branding.go

type BrandingConfig struct {
    LogoURL           string
    LogoSHA           string
    PrimaryColor      string
    SecondaryColor    string
    FontFamily        string
    FooterText        string
    EmailFromName     string
    EmailFromAddress  string
}

// ResolveBranding merges per-app override on top of global branding.
// Any null field in override falls through to global value.
// If appID empty or no override exists, returns global only.
func (s *SQLiteStore) ResolveBranding(ctx context.Context, appID string) (*BrandingConfig, error)
```

**Fallback precedence at render time:** `applications.branding_override` (if set) → `branding` row id='global' → Go struct defaults (coded constants).

### Asset storage layout

```
data/                                  # existing, beside dev.db
  assets/
    branding/
      a1b2c3...f9.png                  # hash.ext, content-addressed
      a1b2c3...f9.svg
```

Served by Go handler at `GET /assets/branding/{sha}.{ext}` with `Cache-Control: public, max-age=31536000, immutable` (content-addressed → safe to cache forever).

---

## Section 2 — API surface

All admin endpoints auth'd via admin API key bearer.

### Branding (admin)

| Method | Path | Purpose | Body | Response |
|--------|------|---------|------|----------|
| GET | `/admin/branding` | Fetch global branding + font enum | — | `{branding: BrandingConfig, fonts: ["manrope","inter","ibm_plex"]}` |
| PATCH | `/admin/branding` | Update global branding | `{primary_color?, secondary_color?, font_family?, footer_text?, email_from_name?, email_from_address?}` | updated config |
| POST | `/admin/branding/logo` | Upload logo | multipart file, ≤1MB, PNG/SVG/JPG | `{logo_url, logo_sha}` |
| DELETE | `/admin/branding/logo` | Clear logo | — | 204 |
| GET | `/admin/applications/{id}/branding` | Resolved (app override → global → default) | — | `BrandingConfig` |
| PATCH | `/admin/applications/{id}/branding` | Save per-app override | `{primary_color?: null|string, ...}` null fields clear override, missing fields untouched | updated app override |

### Email templates (admin)

| Method | Path | Purpose | Body | Response |
|--------|------|---------|------|----------|
| GET | `/admin/email-templates` | List all 5 | — | `{data: [EmailTemplate]}` |
| GET | `/admin/email-templates/{id}` | Single template | — | `EmailTemplate` |
| PATCH | `/admin/email-templates/{id}` | Update | partial fields | updated |
| POST | `/admin/email-templates/{id}/preview` | Render HTML with draft config | `{config?: BrandingConfig, sample_data?: object}` | `{html: string}` |
| POST | `/admin/email-templates/{id}/send-test` | Send real email to any address | `{to_email: string}` per OQ4 no validation | 200 / 4xx |
| POST | `/admin/email-templates/{id}/reset` | Revert to seeded default | — | reset template |

### Application integration mode

| Method | Path | Purpose |
|--------|------|---------|
| PATCH | `/admin/applications/{id}` | Existing endpoint, extend payload to accept `integration_mode`, `proxy_login_fallback`, `proxy_login_fallback_url` |
| GET | `/admin/applications/{id}/snippet?framework=react` | Returns code snippet(s) for components-mode apps. `framework` enum: `react` (V1); others return 501 with "coming soon" |

### Public hosted pages

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/hosted/{app_slug}/{page}` | page ∈ `login|signup|magic|passkey|mfa|verify|error`. Go handler resolves app, injects config as `<script>window.__SHARK_HOSTED=...</script>` into React shell HTML |
| GET | `/hosted/{app_slug}/assets/*` | Static React bundle assets, served from `internal/admin/dist/hosted/assets/*` |

### Public SDK endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/auth/config?app_id=X` | Public config: `{integration_mode, branding, auth_methods, redirect_uri_whitelist}` — consumed by `@sharkauth/react` SharkProvider on mount |
| GET | `/api/v1/me` | Current user from session or bearer (existing) |
| existing OAuth | `/oauth/authorize`, `/oauth/token`, `/.well-known/*` | Unchanged — hosted pages and npm components use standard OAuth 2.1 + PKCE |

### Welcome email trigger (backend)

Extend `internal/auth/email_verify.go` (or equivalent) handler:

```go
// After successful email verification (email_verified flag flip):
if !user.WelcomeEmailSent {
    go emailSender.SendWelcome(ctx, user)
    _ = store.MarkWelcomeEmailSent(ctx, user.ID)
}
```

New `users.welcome_email_sent BOOLEAN DEFAULT FALSE` column (migration 00017 extension).

---

## Section 3 — Frontend architecture

### Dashboard Branding tab

**File:** `admin/src/components/branding.tsx` (NEW) — replaces `empty_shell.tsx` Branding export.

Component structure:
```tsx
export function Branding() {
  const [subtab, setSubtab] = React.useState<'visuals' | 'email' | 'integrations'>('visuals');
  return (
    <Page>
      <TabBar value={subtab} onChange={setSubtab} tabs={['visuals', 'email', 'integrations']}/>
      {subtab === 'visuals' && <BrandingVisualsTab/>}
      {subtab === 'email' && <BrandingEmailTab/>}
      {subtab === 'integrations' && <BrandingIntegrationsTab/>}
    </Page>
  );
}
```

**Visuals subtab:**
- Logo drop-zone (drag-drop OR click to open file picker). Shows current logo preview. "Remove" button.
- 2 color pickers (primary + secondary). Full picker component per OQ3 — HSL slider + preset palette + hex input. Use `react-colorful` or similar (~6KB).
- Font family select (3 options: Manrope / Inter / IBM Plex). Preview live-swaps display font in preview panel.
- Footer textarea (multiline, supports line breaks).
- Email-from name + email-from address inputs.
- Right pane: live preview. Top half = sample hosted login page rendered with current config. Bottom half = sample `magic_link` email rendered. Both update on-change (debounced 300ms).
- "Save" button (disabled while unchanged). Toast on success.

**Email Templates subtab:**
- Left pane (30%): list of 5 templates with name + last-updated timestamp.
- Right pane (70%): selected template editor.
  - Editor left 40%: structured form (subject, preheader, header_text, body_paragraphs[] with add/remove/reorder rows, cta_text, cta_url_template, footer_text).
  - Preview right 60%: iframe `srcDoc`, updates on debounced-300ms POST to `/admin/email-templates/{id}/preview` with draft body. Uses sample data from default seed.
  - "Send test email to [email input]" button + submit. Per OQ4 accepts any address.
  - "Reset to default" button with confirm modal.
  - Unsaved-changes badge.

**Integrations subtab:**
- Table: rows = applications. Columns = name, integration_mode dropdown (4 options), per-mode actions.
- `hosted` mode row: "Preview hosted page" button opens `/hosted/<slug>/login` in new tab.
- `components` mode row: "Get snippet" button opens modal with framework dropdown (React selected; Vue/Svelte/Solid/Angular disabled with "coming soon" tooltip). Below dropdown: 3 code blocks with copy buttons — `npm install`, `SharkProvider setup`, `page usage example`.
- `proxy` mode row: inline config — `proxy_login_fallback` radio (Hosted page | Custom URL) + conditional custom_url input.
- `custom` mode row: just text "API-only. Use `/api/v1/auth/*` endpoints."

### Hosted pages SPA

**New Vite entry point:**
- `admin/vite.config.ts` → add second entry: `hosted-entry` → compiles to `internal/admin/dist/hosted/`
- `admin/src/hosted-entry.tsx` — app root, reads `window.__SHARK_HOSTED` global, renders route based on path segment

**Routes (client-side wouter or react-router):**
| Path | Component | Purpose |
|------|-----------|---------|
| `/hosted/<slug>/login` | `<SignIn/>` | Email + password + OAuth providers + passkey + magic-link buttons |
| `/hosted/<slug>/signup` | `<SignUp/>` | New-account form |
| `/hosted/<slug>/magic` | `<MagicLinkSent/>` | "Check your email" after magic link request |
| `/hosted/<slug>/passkey` | `<PasskeyChallenge/>` | WebAuthn challenge UI |
| `/hosted/<slug>/mfa` | `<MFAChallenge/>` | TOTP or recovery code entry |
| `/hosted/<slug>/verify` | `<EmailVerify/>` | Success/fail after email verify click |
| `/hosted/<slug>/error` | `<Error/>` | Structured error display |

**Go handler pattern:**
```go
// internal/api/hosted_handlers.go
func (s *Server) handleHostedPage(w http.ResponseWriter, r *http.Request) {
    slug := chi.URLParam(r, "app_slug")
    app, err := s.Store.GetApplicationBySlug(r.Context(), slug)
    if err != nil { http.NotFound(w, r); return }
    if app.IntegrationMode != "hosted" && app.IntegrationMode != "proxy" {
        http.Error(w, "hosted auth disabled for this app", 404)
        return
    }
    branding, _ := s.Store.ResolveBranding(r.Context(), app.ID)
    cfg := map[string]any{
        "app_id":        app.ID,
        "app_slug":      app.Slug,
        "app_name":      app.Name,
        "branding":      branding,
        "auth_methods":  s.getEnabledAuthMethods(r.Context()),
        "redirect_uris": app.AllowedCallbackURLs,
        "oauth_base":    s.Config.Server.BaseURL + "/oauth",
    }
    shellHTML := generateHostedShell(cfg) // template with injected <script>
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write([]byte(shellHTML))
}
```

**Route mount order in `router.go`:**
```
/api/v1/*        (API, existing)
/oauth/*         (OAuth, existing)
/admin/*         (dashboard, existing)
/hosted/{slug}/* (NEW, before catch-all)
/assets/*        (NEW for branding assets)
/.well-known/*   (existing)
/*               (proxy catch-all when enabled)
```

**Applications.slug column** — needed. Add to migration 00017. Backfill from `lower(name).replace(/\s+/, '-')`. Dashboard Applications tab editable; validates uniqueness.

### Design system (shared)

**Extract into `admin/src/design/`:**
- `tokens.ts` — color semantic vars (primary, secondary, surface-1/2/3, hairline, fg, fg-dim, danger, warn, success), spacing scale, type scale, border radius presets
- `primitives/Button.tsx`, `Input.tsx`, `Card.tsx`, `FormField.tsx`, `Toast.tsx`, `Modal.tsx`, `Tabs.tsx`
- `composed/SignInForm.tsx`, `SignUpForm.tsx`, `MFAForm.tsx`, `PasskeyButton.tsx`, `OAuthProviderButton.tsx`

Consumers:
- Dashboard (existing admin UI refactored to use these)
- Hosted SPA (new bundle imports from here)
- NPM package (src-copies with same surface, monorepo-linked during development, bundled on publish)

---

## Section 4 — `@sharkauth/react` NPM package

### Package structure

```
packages/shark-auth-react/
  package.json           # name: @sharkauth/react, version: 0.1.0
  README.md
  tsconfig.json
  vite.config.ts         # lib mode, dual CJS+ESM via tsup
  src/
    core/                # framework-agnostic
      client.ts          # fetch wrapper, typed API client
      auth.ts            # OAuth 2.1 + PKCE flow, exchange code for token
      storage.ts         # sessionStorage helpers (per OQ5)
      jwt.ts             # JWKS verify, decode claims
      types.ts
    hooks/
      useAuth.ts         # { isAuthenticated, isLoaded, signOut }
      useUser.ts         # user object
      useSession.ts      # session metadata
      useOrganization.ts # current org (null if none)
    components/
      SharkProvider.tsx  # context root, takes publishableKey + authUrl
      SignIn.tsx         # entry component, starts OAuth flow on click/mount
      SignUp.tsx
      UserButton.tsx     # dropdown menu with sign-out
      SignedIn.tsx       # conditional wrapper: renders children if authed
      SignedOut.tsx      # conditional wrapper: renders children if not authed
      MFAChallenge.tsx
      PasskeyButton.tsx
      OrganizationSwitcher.tsx
    index.ts             # public exports
  dist/                  # build output
```

### Package.json exports

```json
{
  "name": "@sharkauth/react",
  "version": "0.1.0",
  "type": "module",
  "main": "./dist/index.cjs",
  "module": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": { "import": "./dist/index.js", "require": "./dist/index.cjs", "types": "./dist/index.d.ts" },
    "./core": { "import": "./dist/core/index.js", "require": "./dist/core/index.cjs", "types": "./dist/core/index.d.ts" },
    "./hooks": { "import": "./dist/hooks/index.js", "require": "./dist/hooks/index.cjs", "types": "./dist/hooks/index.d.ts" }
  },
  "peerDependencies": {
    "react": ">=18.0.0",
    "react-dom": ">=18.0.0"
  },
  "dependencies": {
    "jose": "^5.0.0"
  }
}
```

### Consumer DX

```tsx
// _app.tsx / layout.tsx
import { SharkProvider } from '@sharkauth/react'

<SharkProvider
  publishableKey="pub_abc123"   // app's client_id, prefixed
  authUrl="https://auth.example.com"
>
  <App/>
</SharkProvider>

// Any component
import { SignIn, UserButton, SignedIn, SignedOut, useAuth } from '@sharkauth/react'

function Header() {
  return (
    <>
      <SignedOut><SignIn redirectUrl="/dashboard"/></SignedOut>
      <SignedIn><UserButton/></SignedIn>
    </>
  )
}

// Hook usage
function Dashboard() {
  const { isAuthenticated, user, signOut } = useAuth()
  if (!isAuthenticated) return <Navigate to="/"/>
  return <div>Hello {user.email}</div>
}
```

### Auth flow (same for all integration modes)

1. `<SharkProvider>` mounts, calls `GET {authUrl}/api/v1/auth/config?app_id={publishableKey}` → caches app config
2. Checks `sessionStorage.shark_token` — decodes JWT, verifies via `{authUrl}/.well-known/jwks.json`
3. If valid → hydrates context with user; `SignedIn` branch renders
4. If missing/expired → `SignedOut` branch renders
5. User clicks `<SignIn/>` → generates PKCE code_verifier + challenge → stores verifier in sessionStorage → redirects to `{authUrl}/oauth/authorize?client_id={publishableKey}&response_type=code&code_challenge=...&state=...&redirect_uri={current_origin}/shark/callback`
6. Shark renders hosted page → user authenticates → shark issues authz code → redirects back to `{origin}/shark/callback?code=...&state=...`
7. Callback page (provided by SDK, mounts on `/shark/callback` route OR via SharkProvider middleware) → exchanges code for tokens at `{authUrl}/oauth/token` with PKCE verifier
8. Tokens stored in sessionStorage: `shark_access_token`, `shark_refresh_token`
9. Context refreshes; `SignedIn` branch now renders
10. Background refresh: `setInterval` checks access token exp; if within 60s, exchanges refresh for new pair

### Callback route handling

**Option 1 (V1):** Consumer app adds `/shark/callback` route that mounts `<SharkCallback/>` component from SDK. Simple.

**Option 2 (V2, Next.js integration):** Middleware handles automatically, user only writes `<SharkProvider>`.

V1 ships with Option 1 + clear docs.

### Publish pipeline

- Monorepo uses `pnpm workspaces` (preferred) OR npm workspaces
- GitHub Actions workflow: on tag `v*` → build + publish `@sharkauth/react`
- Dry-run `npm publish --dry-run` in CI to catch pack issues
- Example app `examples/react-next/` — real Next.js app using SDK, used for e2e tests

### Dashboard code snippet generator

```
GET /admin/applications/{id}/snippet?framework=react

Response:
{
  "framework": "react",
  "snippets": [
    { "label": "Install", "lang": "bash", "code": "npm install @sharkauth/react" },
    { "label": "Provider setup", "lang": "tsx", "code": "import { SharkProvider } from '@sharkauth/react'\n\n<SharkProvider publishableKey=\"pub_abc123\" authUrl=\"https://auth.example.com\">\n  <App/>\n</SharkProvider>" },
    { "label": "Page usage", "lang": "tsx", "code": "import { SignIn, UserButton, SignedIn, SignedOut } from '@sharkauth/react'\n\n<SignedOut><SignIn/></SignedOut>\n<SignedIn><UserButton/></SignedIn>" }
  ]
}
```

---

## Section 5 — Rollout phases

### Phase A — Branding + mail builder (~4 working days)

1. Migration 00017: `branding` + `email_templates` + `applications.slug` + `applications.integration_mode` + `applications.branding_override` + `applications.proxy_login_fallback*` + `users.welcome_email_sent`
2. Storage CRUD + `ResolveBranding(appID)` fallback helper
3. Go email rendering: update `internal/email/templates.go` to pull from DB first, fall back to embedded seeds. Seed DB from embedded templates on first startup (INSERT OR IGNORE).
4. All `/admin/branding/*` + `/admin/email-templates/*` handlers + tests
5. Asset upload handler with 1MB cap (OQ2), PNG/SVG/JPG content-type whitelist
6. Asset serving `/assets/branding/{sha}.{ext}` with cache headers
7. New migration seeds 5 email templates (port current hardcoded HTML to structured fields via offline script committed to repo)
8. Frontend: `admin/src/components/branding.tsx` with 3 subtabs
9. Frontend: color picker (OQ3 — full HSL+palette, use `react-colorful`)
10. Frontend: email template editor + side-by-side iframe preview + debounced preview POST
11. Welcome email wiring in email-verify handler (OQ1 trigger)
12. Drop `ph:9` gate on Branding in `layout.tsx`; remove from `empty_shell.tsx` exports; wire real `Branding` import in `App.tsx` line 24
13. Smoke: section 74 — branding CRUD, logo upload rejects >1MB + non-image, template edit+preview+send-test, fallback chain app→global→default, welcome email fires on verify

### Phase B — Hosted pages SPA (~4 working days)

1. Extract design system into `admin/src/design/` — tokens, primitives, composed. Refactor existing dashboard components to import from there (backwards-compat pass).
2. Vite config: second entry `admin/src/hosted-entry.tsx` → `internal/admin/dist/hosted/`
3. Hosted SPA components: `<SignIn/>`, `<SignUp/>`, `<MagicLinkSent/>`, `<PasskeyChallenge/>`, `<MFAChallenge/>`, `<EmailVerify/>`, `<Error/>` — using design system
4. Client-side router (wouter, ~4KB)
5. Go handler `handleHostedPage(slug, page)` — config injection, shell HTML
6. Mount `/hosted/{app_slug}/{page}` in router.go BEFORE proxy catch-all
7. Dashboard Applications tab: integration_mode dropdown with conditional proxy config
8. Smoke: section 75 — unauthed hit `/hosted/my-app/login` returns HTML shell with correct branding, OAuth code flow round-trip completes, redirect back to app works, error page renders on invalid app_slug

### Phase C — `@sharkauth/react` NPM package (~5 working days)

1. `packages/shark-auth-react/` monorepo structure
2. Core: OAuth + PKCE client, JWT verify via JWKS (jose library), sessionStorage helpers (OQ5)
3. Hooks: useAuth, useUser, useSession, useOrganization
4. Components (shared design tokens copied into package, synced via script during dev)
5. Build: tsup dual CJS/ESM + types. Output to `packages/shark-auth-react/dist/`
6. Example app: `examples/react-next/` Next.js + SharkProvider + SignIn + UserButton
7. GitHub Actions: lint + build + test on PR; on tag `v*` publish
8. Dashboard Integrations subtab: framework dropdown + snippet generator endpoint
9. Docs: `README.md` for package + `docs/guides/react-integration.md` for sharkauth.com/docs
10. Smoke: section 76 — example app boots, SharkProvider fetches auth config, SignIn redirects to shark hosted page, callback exchanges code, user authed in React context

**Total Phase A+B+C: ~13 working days.** User accepts slip past April 27.

**Sequencing:** Phase-7 moat work FIRST (W17/W16/W01/W15/F5.2/F5.3/W22/W18), THEN A → B → C → launch.

### Testing gates per phase

- **Phase A end:** Smoke green on section 74. Manual: visit Branding tab, upload PNG, save, check logo appears in sample consent.html preview.
- **Phase B end:** Smoke green on section 75. Manual: create test app with mode=hosted, visit `/hosted/<slug>/login`, complete auth flow, confirm redirect back with session.
- **Phase C end:** Smoke green on section 76. Manual: `pnpm --filter example-next dev` boots, click SignIn, round-trip completes, UserButton shows email.

---

## Section 6 — Risk register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| JWT XSS theft (sessionStorage per OQ5) | M | M | Short access-token TTL (5m), rotate refresh, per-tab scope limits blast radius. Clerk ships same model. Consider httpOnly-cookie mode in V2. |
| Hosted bundle size bloat | M | L | React + design system + wouter + jose ≈ 180KB gzip. Acceptable for low-frequency auth page. Measure on build, alert >250KB. |
| Branding cache staleness | M | L | Logo content-addressed (hash in URL) so cache-safe; color/font changes require page reload — acceptable (not security-critical). |
| Migration seed failure mid-way | L | H | Use `INSERT OR IGNORE` for idempotent seed. If seed detects partial state, retry per-template. |
| Design system refactor breaks dashboard | M | H | Phase B step 1 is the refactor. Gate: all existing dashboard smoke (375 assertions) must pass before shipping design system migration. Worktree isolation if parallel. |
| Welcome email spam (user verifies twice?) | L | L | `users.welcome_email_sent` flag prevents duplicate; set before dispatch with `UPDATE RETURNING` guard. |
| Logo upload RCE via SVG embedded script | M | H | Reject SVG MIME unless content parses as clean SVG (no `<script>`, no `<foreignObject>`); OR disallow SVG initially, PNG/JPG only. |
| Integration mode conflict (proxy + hosted both enabled for same app) | L | M | Single enum (not flag set) — mutually exclusive at schema level. Session widgets always available separately. |
| Cross-origin token flow breaks in embedded iframes | M | M | Require `authUrl` be served with `Content-Security-Policy: frame-ancestors *;` for known consumer origins only; document limitation. |
| npm package name squatted | L | M | Reserve `@sharkauth/react` ASAP on npm (step 0 before implementing). User owns npm org. |

---

## Section 7 — Impeccable handoff brief

**For `/impeccable` skill invocation immediately after this spec is approved:**

> Build frontend components for SharkAuth's unified auth UX layer. Three consumers share ONE design system: admin dashboard Branding tab preview panels, hosted auth SPA at `/hosted/<app-slug>/*`, and `@sharkauth/react` NPM package. Source of truth is `admin/src/design/` with tokens, primitives, and composed components.
>
> **Aesthetic:** dark-first (match existing `internal/oauth/consent.html`), oklch color space, configurable primary accent from branding config (default `#7c3aed`), Manrope display font (configurable to Inter or IBM Plex), Azeret Mono for code/mono, generous spacing (8-12-16-24-32 scale), subtle motion (160ms ease-out for fades, 240ms for slides), Linear-density information architecture. Absolutely avoid AI-slop (gradient backgrounds, overlapping blobs, generic glass-morphism). Aim at Clerk / Linear / Vercel polish.
>
> **Components to ship:**
>
> *Primitives:* Button (primary/ghost/danger/icon variants + sm/md/lg sizes), Input, FormField (label + input + error + hint), Card, Modal, Tabs, Toast, Kbd (keyboard shortcut chip), Avatar, CLIFooter.
>
> *Auth surfaces:* `<SignInForm/>` (email + password + "forgot?" + OAuth provider buttons + passkey button + magic link CTA), `<SignUpForm/>` (email + password + optional name + ToS checkbox), `<MagicLinkSent/>` ("check your email" state), `<PasskeyChallenge/>` (WebAuthn prompt UI), `<MFAChallenge/>` (TOTP 6-digit input + recovery code fallback), `<EmailVerify/>` (success + error variants), `<ErrorPage/>` (structured error display with problem/cause/suggestion/docs_url).
>
> *Session-aware:* `<SharkProvider/>` (context root), `<SignIn/>` (trigger component, opens hosted page or modal per prop), `<SignUp/>`, `<UserButton/>` (avatar + dropdown: "Manage account", "Sign out"), `<SignedIn/>` + `<SignedOut/>` (conditional wrappers), `<OrganizationSwitcher/>` (if user in multiple orgs).
>
> **Accessibility:** WCAG AA. Keyboard-first. Every interactive element focusable with visible ring. Form fields labeled. Error messages `aria-live="polite"`. Color-contrast ratio ≥ 4.5:1 on text, ≥ 3:1 on UI components.
>
> **Design goals:** Production-polish, not demo-quality. Components must work in three contexts: dashboard preview iframes, full-page hosted routes, arbitrary consumer React apps (any CSS context, any parent width). Each exported component isolated (no implicit global state, no CSS leaks, className namespace `sk-*` or CSS Modules).
>
> **Forbidden:** gradient mesh backgrounds, `bg-gradient-to-*`, generic SaaS blobs, overly rounded pill buttons with shadow, neon glow, "Crafted with ❤️", stock icons, rotating 3D shapes, auto-height animations that jitter.
>
> **Design fidelity proof:** Before shipping Phase B, render all 7 hosted pages at 1440×900 + 375×812 viewports. Compare against Clerk + Auth0 hosted pages. If a reasonable viewer thinks shark's is at best on par and at worst not distinctively uglier, passes.

---

## Section 8 — Next action

1. User reviews this spec file in full.
2. If approved: invoke `superpowers:writing-plans` skill to produce implementation plan (task-by-task execution order, per-commit scope, verification gates).
3. After writing-plans produces plan: execute Phase A first, verify, then B, then C.
4. Build-time invocation of `/impeccable` skill (after writing-plans, before coding) to generate actual component JSX+CSS from the design brief in Section 7.

**Launch note:** This spec explicitly slips April 27 launch. User's decision, user's risk. All pre-launch moat work (W17 AGENT_AUTH rewrite, W16 flow conditional bug, W01 orgs create, W15 multi-listener, W22 CLI --json, W18 structured errors, F5.2 agent SDK, F5.3 Hello Agent walkthrough) MUST ship before A/B/C kick off. See `FRONTEND_WIRING_GAPS.md` launch-critical path table.

---

**Spec version:** 1.0
**Approved by:** user, 2026-04-20, via inline Section-by-Section acceptance in brainstorm dialog
**Spec committed to:** `docs/superpowers/specs/2026-04-20-branding-hosted-components-design.md`
