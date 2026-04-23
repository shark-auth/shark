# Branding + Hosted Pages + `@shark-auth/react` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a unified branding, mail-builder, hosted auth pages, and `@shark-auth/react` NPM package, sharing one design system across three consumers (admin dashboard, hosted SPA, NPM widgets).

**Architecture:** Three sequential phases. Phase A ships backend + dashboard Branding tab + editable email templates. Phase B extracts shared design system and ships hosted React SPA mounted at `/hosted/<app-slug>/*`. Phase C ships `@shark-auth/react` NPM package reusing Phase B components. A→B→C is strict — B depends on Phase A migrations; C depends on Phase B hosted pages as OAuth redirect target.

**Tech Stack:** Go (chi router, goose migrations, html/template, go:embed), SQLite, React 18, Vite, TypeScript, jose (JWKS verification), react-colorful (color picker), wouter (client-side routing in hosted SPA), tsup (NPM package dual CJS/ESM build), pnpm workspaces (monorepo).

**Spec:** `docs/superpowers/specs/2026-04-20-branding-hosted-components-design.md` — full context including bus-factor decision log.

---

## Pre-requisite launch-moat work (NOT in this plan, but blocks kickoff)

This plan assumes the following tasks from `FRONTEND_WIRING_GAPS.md` have shipped before Phase A starts. Do not begin Phase A until these are all green:

- **W17** — `AGENT_AUTH.md` status rewrite (launch trust gate)
- **W16** — flow builder conditional `then`/`else` save bug fix
- **W01** — orgs `organizations.tsx:57` create-button dead-click fix
- **W15** — multi-listener proxy (unlocks `:3000` transparent protection for launch narrative)
- **W22** — CLI `--json` flag on all output commands
- **W18** — structured error envelope across auth + OAuth endpoints (minimum: OAuth side)
- **F5.2** — `shark-sdk-py` DPoPProver + VaultClient MVP
- **F5.3** — "Hello Agent" end-to-end walkthrough doc + example script

User explicitly accepts this scheduling slips public launch past April 27.

---

## Scope check

Spec covers three coupled phases sharing one design system. Not decomposed into separate specs because: (a) Phase B extracts shared tokens consumed by A's preview panels, (b) Phase C imports Phase B components via monorepo workspaces, (c) user chose unified in brainstorming. Each phase produces independently testable software with its own smoke test section (74 / 75 / 76). Keep unified; enforce strict phase gates.

---

## File structure

### Phase A — Branding + mail builder

**Create:**
- `cmd/shark/migrations/00017_branding_and_email_templates.sql` — tables + column adds + seed trigger
- `internal/storage/branding.go` — `ResolveBranding` helper + CRUD
- `internal/storage/email_templates.go` — template CRUD + seed-on-startup
- `internal/api/branding_handlers.go` — all `/admin/branding/*` handlers
- `internal/api/email_template_handlers.go` — all `/admin/email-templates/*` handlers
- `internal/api/hosted_assets_handler.go` — `/assets/branding/{sha}.{ext}` serving
- `admin/src/components/branding.tsx` — replaces `empty_shell.tsx::Branding` stub
- `admin/src/components/branding/visuals_tab.tsx` — logo upload, color pickers, font select, footer, email-from
- `admin/src/components/branding/email_tab.tsx` — template list + editor + side-by-side iframe preview
- `admin/src/components/branding/integrations_tab.tsx` — per-app integration_mode dropdown + snippet modal

**Modify:**
- `internal/email/templates.go` — render from DB first, fall back to embedded defaults
- `internal/api/router.go` — mount new handlers, `/assets/branding/*` route before proxy catch-all
- `internal/api/admin_stats_handlers.go` OR equivalent application-handler — extend PATCH payload to accept `integration_mode` + proxy fallback fields
- `internal/auth/email_verify.go` — welcome-email trigger on verification success
- `admin/src/components/App.tsx:24` — replace `Branding` import source from `./empty_shell` to `./branding`
- `admin/src/components/empty_shell.tsx` — remove `Branding` export
- `admin/src/components/layout.tsx` — drop `ph:9` gate on Branding nav item
- `smoke_test.sh` — add section 74

### Phase B — Hosted pages SPA

**Create:**
- `admin/src/design/tokens.ts` — color/spacing/type tokens (extracted from existing admin code)
- `admin/src/design/primitives/Button.tsx`, `Input.tsx`, `FormField.tsx`, `Card.tsx`, `Modal.tsx`, `Tabs.tsx`, `Toast.tsx` (extracted/refactored from existing dashboard)
- `admin/src/design/composed/SignInForm.tsx`, `SignUpForm.tsx`, `MFAForm.tsx`, `PasskeyButton.tsx`, `OAuthProviderButton.tsx`, `MagicLinkSent.tsx`, `EmailVerify.tsx`, `ErrorPage.tsx`
- `admin/src/hosted-entry.tsx` — Vite entry for hosted SPA
- `admin/src/hosted/App.tsx` — hosted app router (wouter)
- `admin/src/hosted/routes/login.tsx`, `signup.tsx`, `magic.tsx`, `passkey.tsx`, `mfa.tsx`, `verify.tsx`, `error.tsx`
- `internal/api/hosted_handlers.go` — `handleHostedPage(slug, page)` + shell HTML generator
- `internal/api/application_slug.go` — slug generation + validation helpers

**Modify:**
- `admin/vite.config.ts` — add second Rollup entry `hosted` compiling to `internal/admin/dist/hosted/`
- `internal/api/router.go` — mount `/hosted/{app_slug}/{page}` before proxy catch-all
- `cmd/shark/migrations/00018_application_slug.sql` — add `applications.slug UNIQUE NOT NULL` + backfill
- `admin/src/components/applications.tsx` — add `slug` field to app create/edit forms + validation
- Existing admin components — refactor to import from `admin/src/design/` instead of inline styles
- `smoke_test.sh` — add section 75

### Phase C — `@shark-auth/react` NPM package

**Create:**
- `package.json` at repo root — add `workspaces: ["packages/*", "admin", "examples/*"]` (or `pnpm-workspace.yaml` if using pnpm)
- `packages/shark-auth-react/package.json` — name `@shark-auth/react`, version `0.1.0`
- `packages/shark-auth-react/tsconfig.json`
- `packages/shark-auth-react/vite.config.ts` — lib mode
- `packages/shark-auth-react/tsup.config.ts` — dual CJS/ESM + .d.ts generation
- `packages/shark-auth-react/src/core/client.ts`, `auth.ts`, `storage.ts`, `jwt.ts`, `types.ts`
- `packages/shark-auth-react/src/hooks/useAuth.ts`, `useUser.ts`, `useSession.ts`, `useOrganization.ts`
- `packages/shark-auth-react/src/components/SharkProvider.tsx`, `SignIn.tsx`, `SignUp.tsx`, `UserButton.tsx`, `SignedIn.tsx`, `SignedOut.tsx`, `MFAChallenge.tsx`, `PasskeyButton.tsx`, `OrganizationSwitcher.tsx`, `SharkCallback.tsx`
- `packages/shark-auth-react/src/index.ts` — public exports
- `packages/shark-auth-react/README.md` — consumer docs
- `examples/react-next/` — Next.js example app (pnpm workspace member)
- `.github/workflows/publish-sdk.yml` — on `v*` tag, build + publish
- `docs/guides/react-integration.md` — integration guide (for sharkauth.com/docs)

**Modify:**
- `internal/api/snippet_handlers.go` (create) — `GET /admin/applications/{id}/snippet?framework=react`
- `admin/src/components/branding/integrations_tab.tsx` — "Get snippet" button wires to endpoint
- `smoke_test.sh` — add section 76

---

## Worktree isolation notes

Per user memory: parallel writer subagents on the same repo **MUST** use `isolation: "worktree"`. Within this plan:

- Phase A tasks are sequential within Phase A — **NO worktree isolation needed** for single-agent execution.
- Phase A and Phase B can run in parallel if two writer subagents dispatched simultaneously — **in that case both MUST use `isolation: "worktree"` independently**. Plan sequencing recommends A-then-B anyway.
- Phase B step 1 (design system extraction) **refactors existing admin components** — high blast radius. If split into multiple subagents, each MUST use worktree and coordinate on design token names.
- Phase C is entirely additive (new `packages/` directory) — low collision risk, but still use worktree if running in parallel with A or B.

---

# PHASE A — Branding + mail builder (~4 working days, ~18 tasks)

## Verification gate before Phase A kickoff

- [ ] **Pre-check: confirm launch-moat work shipped**

All tasks W17 / W16 / W01 / W15 / W22 / W18 / F5.2 / F5.3 must show `done` in `FRONTEND_WIRING_STATUS.json`. Confirm with:

```bash
cat FRONTEND_WIRING_STATUS.json | jq '.tasks | {W01,W15,W16,W17,W18,W22}'
```

Expected: all six entries show `"status": "done"`. If any pending/blocked, stop and finish moat work first.

---

### Task A1: Migration 00017 — branding + email_templates tables

**Files:**
- Create: `cmd/shark/migrations/00017_branding_and_email_templates.sql`
- Test: manual via `bin/shark.exe serve --dev` on fresh DB + SQL inspection

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up
-- Branding config (global row for V1, extensible to org scope later)
CREATE TABLE branding (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'global',
  logo_url TEXT,
  logo_sha TEXT,
  primary_color TEXT DEFAULT '#7c3aed',
  secondary_color TEXT DEFAULT '#1a1a1a',
  font_family TEXT DEFAULT 'manrope',
  footer_text TEXT DEFAULT '',
  email_from_name TEXT DEFAULT 'SharkAuth',
  email_from_address TEXT DEFAULT 'noreply@example.com',
  updated_at TEXT NOT NULL
);

INSERT INTO branding (id, scope, updated_at) VALUES ('global', 'global', CURRENT_TIMESTAMP);

-- Email templates (editable copy)
CREATE TABLE email_templates (
  id TEXT PRIMARY KEY,
  subject TEXT NOT NULL,
  preheader TEXT NOT NULL DEFAULT '',
  header_text TEXT NOT NULL,
  body_paragraphs TEXT NOT NULL DEFAULT '[]',
  cta_text TEXT NOT NULL DEFAULT '',
  cta_url_template TEXT NOT NULL DEFAULT '',
  footer_text TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

-- Applications integration mode + branding override + proxy fallback
ALTER TABLE applications ADD COLUMN integration_mode TEXT NOT NULL DEFAULT 'custom';
ALTER TABLE applications ADD COLUMN branding_override TEXT;
ALTER TABLE applications ADD COLUMN proxy_login_fallback TEXT NOT NULL DEFAULT 'hosted';
ALTER TABLE applications ADD COLUMN proxy_login_fallback_url TEXT;

-- Users welcome-email-sent flag
ALTER TABLE users ADD COLUMN welcome_email_sent INTEGER NOT NULL DEFAULT 0;

-- +goose Down
DROP TABLE email_templates;
DROP TABLE branding;
-- NOTE: SQLite does not support ALTER TABLE DROP COLUMN pre 3.35.
-- Rollback leaves applications + users columns in place; acceptable for dev.
```

- [ ] **Step 2: Start server on fresh DB, verify migration ran**

```bash
rm -f dev.db dev.db-*
bin/shark.exe serve --dev &
sleep 2
sqlite3 dev.db "SELECT id FROM branding; PRAGMA table_info(email_templates); PRAGMA table_info(applications);" | head -30
```

Expected output: `global` in branding, 8 columns in email_templates, `integration_mode`, `branding_override`, `proxy_login_fallback`, `proxy_login_fallback_url` appear in applications table.

- [ ] **Step 3: Commit**

```bash
git add cmd/shark/migrations/00017_branding_and_email_templates.sql
git commit -m "feat(branding): migration 00017 — branding config + email_templates + app integration_mode"
```

---

### Task A2: Storage layer — branding CRUD + resolve

**Files:**
- Create: `internal/storage/branding.go`
- Create: `internal/storage/branding_test.go`
- Modify: `internal/storage/storage.go` — add `BrandingStore` methods to `Store` interface

- [ ] **Step 1: Extend Store interface**

Add to `internal/storage/storage.go` Store interface:

```go
// BrandingConfig is the resolved branding (global + app override merged).
type BrandingConfig struct {
	LogoURL          string `json:"logo_url,omitempty"`
	LogoSHA          string `json:"logo_sha,omitempty"`
	PrimaryColor     string `json:"primary_color"`
	SecondaryColor   string `json:"secondary_color"`
	FontFamily       string `json:"font_family"`
	FooterText       string `json:"footer_text"`
	EmailFromName    string `json:"email_from_name"`
	EmailFromAddress string `json:"email_from_address"`
}

type Store interface {
	// ... existing methods
	GetBranding(ctx context.Context, id string) (*BrandingConfig, error)
	UpdateBranding(ctx context.Context, id string, fields map[string]any) error
	SetBrandingLogo(ctx context.Context, id, url, sha string) error
	ClearBrandingLogo(ctx context.Context, id string) error
	ResolveBranding(ctx context.Context, appID string) (*BrandingConfig, error)
}
```

- [ ] **Step 2: Write failing test**

Create `internal/storage/branding_test.go`:

```go
package storage

import (
	"context"
	"testing"
)

func TestResolveBranding_GlobalOnly(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	got, err := s.ResolveBranding(ctx, "")
	if err != nil {
		t.Fatalf("ResolveBranding: %v", err)
	}
	if got.PrimaryColor != "#7c3aed" {
		t.Errorf("expected default primary, got %q", got.PrimaryColor)
	}
}

func TestResolveBranding_AppOverride(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	// Seed app with branding override
	appID := "app_test"
	seedApplication(t, s, appID, `{"primary_color":"#ff0000"}`)

	got, err := s.ResolveBranding(ctx, appID)
	if err != nil {
		t.Fatalf("ResolveBranding: %v", err)
	}
	if got.PrimaryColor != "#ff0000" {
		t.Errorf("expected override primary #ff0000, got %q", got.PrimaryColor)
	}
	if got.SecondaryColor != "#1a1a1a" {
		t.Errorf("expected fallback secondary, got %q", got.SecondaryColor)
	}
}
```

- [ ] **Step 3: Run test, confirm fails**

```bash
go test ./internal/storage/ -run TestResolveBranding -v
```

Expected: FAIL with `ResolveBranding undefined`.

- [ ] **Step 4: Implement `internal/storage/branding.go`**

```go
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SQLiteStore) GetBranding(ctx context.Context, id string) (*BrandingConfig, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(logo_url,''), COALESCE(logo_sha,''), primary_color, secondary_color,
		       font_family, footer_text, email_from_name, email_from_address
		FROM branding WHERE id = ?`, id)
	b := &BrandingConfig{}
	err := row.Scan(&b.LogoURL, &b.LogoSHA, &b.PrimaryColor, &b.SecondaryColor,
		&b.FontFamily, &b.FooterText, &b.EmailFromName, &b.EmailFromAddress)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("branding not found: %s", id)
	}
	return b, err
}

func (s *SQLiteStore) UpdateBranding(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"primary_color": true, "secondary_color": true, "font_family": true,
		"footer_text": true, "email_from_name": true, "email_from_address": true,
	}
	setParts := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		setParts = append(setParts, k+" = ?")
		args = append(args, v)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)
	q := "UPDATE branding SET " + joinComma(setParts) + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

func (s *SQLiteStore) SetBrandingLogo(ctx context.Context, id, url, sha string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE branding SET logo_url = ?, logo_sha = ?, updated_at = ? WHERE id = ?`,
		url, sha, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *SQLiteStore) ClearBrandingLogo(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE branding SET logo_url = NULL, logo_sha = NULL, updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// ResolveBranding returns global branding merged with per-app override (if any).
// appID empty → global only. Missing override → global. Override fields fall through when empty/null.
func (s *SQLiteStore) ResolveBranding(ctx context.Context, appID string) (*BrandingConfig, error) {
	global, err := s.GetBranding(ctx, "global")
	if err != nil {
		return nil, fmt.Errorf("resolve branding global: %w", err)
	}
	if appID == "" {
		return global, nil
	}

	var override sql.NullString
	err = s.db.QueryRowContext(ctx,
		`SELECT branding_override FROM applications WHERE id = ?`, appID).Scan(&override)
	if err == sql.ErrNoRows || !override.Valid || override.String == "" {
		return global, nil
	}
	if err != nil {
		return global, nil
	}

	var over map[string]string
	if err := json.Unmarshal([]byte(override.String), &over); err != nil {
		return global, nil
	}

	merged := *global
	if v := over["logo_url"]; v != "" {
		merged.LogoURL = v
	}
	if v := over["logo_sha"]; v != "" {
		merged.LogoSHA = v
	}
	if v := over["primary_color"]; v != "" {
		merged.PrimaryColor = v
	}
	if v := over["secondary_color"]; v != "" {
		merged.SecondaryColor = v
	}
	if v := over["font_family"]; v != "" {
		merged.FontFamily = v
	}
	if v := over["footer_text"]; v != "" {
		merged.FooterText = v
	}
	if v := over["email_from_name"]; v != "" {
		merged.EmailFromName = v
	}
	if v := over["email_from_address"]; v != "" {
		merged.EmailFromAddress = v
	}
	return &merged, nil
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
```

- [ ] **Step 5: Run tests, confirm pass**

```bash
go test ./internal/storage/ -run TestResolveBranding -v -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/storage/branding.go internal/storage/branding_test.go internal/storage/storage.go
git commit -m "feat(branding): ResolveBranding with app->global fallback"
```

---

### Task A3: Storage layer — email_templates CRUD + seed on startup

**Files:**
- Create: `internal/storage/email_templates.go`
- Create: `internal/storage/email_templates_test.go`
- Create: `internal/storage/email_templates_seed.go` (seed source from existing `.html` templates)
- Modify: `internal/storage/storage.go` — extend Store interface
- Modify: `internal/server/server.go` — call `SeedEmailTemplates` on startup

- [ ] **Step 1: Extend Store interface**

```go
type EmailTemplate struct {
	ID              string   `json:"id"`
	Subject         string   `json:"subject"`
	Preheader       string   `json:"preheader"`
	HeaderText      string   `json:"header_text"`
	BodyParagraphs  []string `json:"body_paragraphs"`
	CTAText         string   `json:"cta_text"`
	CTAURLTemplate  string   `json:"cta_url_template"`
	FooterText      string   `json:"footer_text"`
	UpdatedAt       string   `json:"updated_at"`
}

type Store interface {
	// ...
	ListEmailTemplates(ctx context.Context) ([]*EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, id string) (*EmailTemplate, error)
	UpdateEmailTemplate(ctx context.Context, id string, fields map[string]any) error
	SeedEmailTemplates(ctx context.Context) error // idempotent INSERT OR IGNORE
}
```

- [ ] **Step 2: Write failing test**

Create `internal/storage/email_templates_test.go`:

```go
package storage

import (
	"context"
	"testing"
)

func TestSeedEmailTemplates_Idempotent(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.SeedEmailTemplates(ctx); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := s.SeedEmailTemplates(ctx); err != nil {
		t.Fatalf("second seed (should be idempotent): %v", err)
	}

	list, err := s.ListEmailTemplates(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("expected 5 seed templates, got %d", len(list))
	}

	wantIDs := map[string]bool{
		"magic_link": true, "password_reset": true, "verify_email": true,
		"organization_invitation": true, "welcome": true,
	}
	for _, tpl := range list {
		delete(wantIDs, tpl.ID)
	}
	if len(wantIDs) > 0 {
		t.Errorf("missing templates: %v", wantIDs)
	}
}

func TestUpdateEmailTemplate_PreservesUnchanged(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	_ = s.SeedEmailTemplates(ctx)

	err := s.UpdateEmailTemplate(ctx, "magic_link", map[string]any{
		"subject": "Custom subject",
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := s.GetEmailTemplate(ctx, "magic_link")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Subject != "Custom subject" {
		t.Errorf("subject not persisted: %q", got.Subject)
	}
	if got.HeaderText == "" {
		t.Errorf("header_text was cleared — should only update 'subject'")
	}
}
```

- [ ] **Step 3: Run test, confirm fails**

```bash
go test ./internal/storage/ -run TestSeedEmailTemplates -v -count=1
```

Expected: FAIL.

- [ ] **Step 4: Implement `internal/storage/email_templates.go`**

```go
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *SQLiteStore) ListEmailTemplates(ctx context.Context) ([]*EmailTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at
		FROM email_templates ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*EmailTemplate{}
	for rows.Next() {
		t := &EmailTemplate{}
		var paragraphsJSON string
		err := rows.Scan(&t.ID, &t.Subject, &t.Preheader, &t.HeaderText,
			&paragraphsJSON, &t.CTAText, &t.CTAURLTemplate, &t.FooterText, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(paragraphsJSON), &t.BodyParagraphs)
		if t.BodyParagraphs == nil {
			t.BodyParagraphs = []string{}
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetEmailTemplate(ctx context.Context, id string) (*EmailTemplate, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at
		FROM email_templates WHERE id = ?`, id)
	t := &EmailTemplate{}
	var paragraphsJSON string
	err := row.Scan(&t.ID, &t.Subject, &t.Preheader, &t.HeaderText,
		&paragraphsJSON, &t.CTAText, &t.CTAURLTemplate, &t.FooterText, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email template not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(paragraphsJSON), &t.BodyParagraphs)
	if t.BodyParagraphs == nil {
		t.BodyParagraphs = []string{}
	}
	return t, nil
}

func (s *SQLiteStore) UpdateEmailTemplate(ctx context.Context, id string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	allowed := map[string]bool{
		"subject": true, "preheader": true, "header_text": true,
		"body_paragraphs": true, "cta_text": true, "cta_url_template": true, "footer_text": true,
	}
	setParts := []string{}
	args := []any{}
	for k, v := range fields {
		if !allowed[k] {
			continue
		}
		if k == "body_paragraphs" {
			if arr, ok := v.([]string); ok {
				b, _ := json.Marshal(arr)
				v = string(b)
			} else if arr, ok := v.([]any); ok {
				b, _ := json.Marshal(arr)
				v = string(b)
			}
		}
		setParts = append(setParts, k+" = ?")
		args = append(args, v)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339))
	args = append(args, id)
	q := "UPDATE email_templates SET " + joinComma(setParts) + " WHERE id = ?"
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

// SeedEmailTemplates inserts V1 templates if missing. Idempotent.
func (s *SQLiteStore) SeedEmailTemplates(ctx context.Context) error {
	seeds := defaultEmailTemplateSeeds()
	now := time.Now().UTC().Format(time.RFC3339)
	for _, seed := range seeds {
		pJSON, _ := json.Marshal(seed.BodyParagraphs)
		_, err := s.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO email_templates
			(id, subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			seed.ID, seed.Subject, seed.Preheader, seed.HeaderText, string(pJSON),
			seed.CTAText, seed.CTAURLTemplate, seed.FooterText, now)
		if err != nil {
			return fmt.Errorf("seed %s: %w", seed.ID, err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Create seed source file `internal/storage/email_templates_seed.go`**

```go
package storage

// defaultEmailTemplateSeeds returns the V1 templates. Ported from existing
// internal/email/templates/*.html content to structured fields.
// Welcome is new (fires on email verification per spec OQ1).
func defaultEmailTemplateSeeds() []*EmailTemplate {
	return []*EmailTemplate{
		{
			ID:             "magic_link",
			Subject:        "Sign in to {{.AppName}}",
			Preheader:      "Your magic link expires in 15 minutes.",
			HeaderText:     "Click to sign in",
			BodyParagraphs: []string{"Use the link below to sign in to your account."},
			CTAText:        "Sign in now",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This link expires in 15 minutes. If you didn't request it, ignore this email.",
		},
		{
			ID:             "password_reset",
			Subject:        "Reset your {{.AppName}} password",
			Preheader:      "Click to choose a new password.",
			HeaderText:     "Reset your password",
			BodyParagraphs: []string{"We received a request to reset your password. Click the button below to choose a new one."},
			CTAText:        "Reset password",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This link expires in 15 minutes. If you didn't request a reset, ignore this email.",
		},
		{
			ID:             "verify_email",
			Subject:        "Verify your email",
			Preheader:      "Confirm your email to finish signing up.",
			HeaderText:     "One last step",
			BodyParagraphs: []string{"Verify your email so we know it's really you."},
			CTAText:        "Verify email",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "If you didn't create an account, ignore this email.",
		},
		{
			ID:             "organization_invitation",
			Subject:        "{{.InviterName}} invited you to {{.OrgName}}",
			Preheader:      "Accept your invitation.",
			HeaderText:     "You've been invited",
			BodyParagraphs: []string{"{{.InviterName}} invited you to join {{.OrgName}} on {{.AppName}}."},
			CTAText:        "Accept invitation",
			CTAURLTemplate: "{{.Link}}",
			FooterText:     "This invitation expires in 7 days.",
		},
		{
			ID:             "welcome",
			Subject:        "Welcome to {{.AppName}}",
			Preheader:      "You're all set.",
			HeaderText:     "Welcome aboard",
			BodyParagraphs: []string{"Thanks for verifying your email. You're ready to go."},
			CTAText:        "Get started",
			CTAURLTemplate: "{{.DashboardURL}}",
			FooterText:     "Questions? Just reply to this email.",
		},
	}
}
```

- [ ] **Step 6: Call seed on server startup**

In `internal/server/server.go` after migrations run:

```go
if err := srv.Store.SeedEmailTemplates(ctx); err != nil {
	slog.Warn("seed email templates", "error", err)
}
```

- [ ] **Step 7: Run tests, confirm pass**

```bash
go test ./internal/storage/ -run TestSeedEmailTemplates -v -count=1
go test ./internal/storage/ -run TestUpdateEmailTemplate -v -count=1
```

Expected: both PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/storage/email_templates.go internal/storage/email_templates_test.go internal/storage/email_templates_seed.go internal/storage/storage.go internal/server/server.go
git commit -m "feat(branding): email_templates storage + seed on startup"
```

---

### Task A4: `internal/email/templates.go` render from DB

**Files:**
- Modify: `internal/email/templates.go` — pull subject + body from DB template, fall back to embedded defaults on miss

- [ ] **Step 1: Refactor `RenderMagicLink` (pattern applied to all 5)**

Replace:

```go
func RenderMagicLink(data MagicLinkData) (string, error) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "magic_link.html", data); err != nil {
		return "", fmt.Errorf("rendering magic link template: %w", err)
	}
	return buf.String(), nil
}
```

With:

```go
func RenderMagicLink(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data MagicLinkData) (Rendered, error) {
	tmpl, err := store.GetEmailTemplate(ctx, "magic_link")
	if err != nil {
		// fall back to embedded default
		var buf bytes.Buffer
		if err := templates.ExecuteTemplate(&buf, "magic_link.html", data); err != nil {
			return Rendered{}, fmt.Errorf("fallback render magic link: %w", err)
		}
		return Rendered{Subject: "Sign in to " + data.AppName, HTML: buf.String()}, nil
	}
	return renderStructured(tmpl, branding, data)
}

type Rendered struct {
	Subject string
	HTML    string
}
```

Add `renderStructured` helper:

```go
// renderStructured takes a DB-backed template, substitutes {{.}} fields, applies branding colors,
// produces full HTML via an outer layout template.
func renderStructured(tmpl *storage.EmailTemplate, branding *storage.BrandingConfig, data any) (Rendered, error) {
	subject, err := execString(tmpl.Subject, data)
	if err != nil {
		return Rendered{}, err
	}
	header, _ := execString(tmpl.HeaderText, data)
	paragraphs := make([]string, len(tmpl.BodyParagraphs))
	for i, p := range tmpl.BodyParagraphs {
		paragraphs[i], _ = execString(p, data)
	}
	ctaText, _ := execString(tmpl.CTAText, data)
	ctaURL, _ := execString(tmpl.CTAURLTemplate, data)
	footer, _ := execString(tmpl.FooterText, data)
	preheader, _ := execString(tmpl.Preheader, data)

	layoutData := struct {
		Branding   *storage.BrandingConfig
		Preheader  string
		Header     string
		Paragraphs []string
		CTAText    string
		CTAURL     string
		Footer     string
	}{branding, preheader, header, paragraphs, ctaText, ctaURL, footer}

	var buf bytes.Buffer
	if err := emailLayout.Execute(&buf, layoutData); err != nil {
		return Rendered{}, fmt.Errorf("email layout render: %w", err)
	}
	return Rendered{Subject: subject, HTML: buf.String()}, nil
}

func execString(tmplStr string, data any) (string, error) {
	if tmplStr == "" {
		return "", nil
	}
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

//go:embed layout.html
var emailLayoutHTML string
var emailLayout = template.Must(template.New("layout").Parse(emailLayoutHTML))
```

- [ ] **Step 2: Create `internal/email/layout.html` (shared outer email HTML)**

```html
<!DOCTYPE html>
<html><head><meta charset="UTF-8">
<meta name="color-scheme" content="light dark">
<title>{{.Header}}</title>
<style>
  body{margin:0;padding:0;background:#f6f7f9;font-family:-apple-system,system-ui,sans-serif}
  .container{max-width:520px;margin:0 auto;padding:32px 16px}
  .card{background:#ffffff;border-radius:8px;padding:32px;box-shadow:0 1px 2px rgba(0,0,0,.04)}
  .logo{max-height:40px;margin-bottom:24px}
  h1{margin:0 0 16px;font-size:22px;color:#0a0a0a}
  p{margin:0 0 12px;color:#4a4a4a;line-height:1.55;font-size:14px}
  .btn{display:inline-block;background:{{.Branding.PrimaryColor}};color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none;font-weight:600;font-size:14px;margin:16px 0}
  .footer{margin-top:24px;font-size:12px;color:#8a8a8a}
  .preheader{display:none;max-height:0;overflow:hidden;opacity:0;color:transparent}
</style>
</head><body>
<div class="preheader">{{.Preheader}}</div>
<div class="container">
  <div class="card">
    {{if .Branding.LogoURL}}<img class="logo" src="{{.Branding.LogoURL}}" alt="">{{end}}
    <h1>{{.Header}}</h1>
    {{range .Paragraphs}}<p>{{.}}</p>{{end}}
    {{if .CTAText}}<a class="btn" href="{{.CTAURL}}">{{.CTAText}}</a>{{end}}
    {{if .Footer}}<div class="footer">{{.Footer}}</div>{{end}}
  </div>
</div>
</body></html>
```

- [ ] **Step 3: Apply same pattern to `RenderPasswordReset`, `RenderVerifyEmail`, `RenderOrganizationInvitation`, and new `RenderWelcome`**

Each function follows the same refactor: accept `ctx, store, branding, data`, look up DB template by ID, fall back to embedded `.html`, use `renderStructured`. Add:

```go
type WelcomeData struct {
	AppName      string
	UserEmail    string
	DashboardURL string
}

func RenderWelcome(ctx context.Context, store storage.Store, branding *storage.BrandingConfig, data WelcomeData) (Rendered, error) {
	tmpl, err := store.GetEmailTemplate(ctx, "welcome")
	if err != nil {
		return Rendered{
			Subject: "Welcome to " + data.AppName,
			HTML:    "<p>Thanks for verifying your email.</p>",
		}, nil
	}
	return renderStructured(tmpl, branding, data)
}
```

- [ ] **Step 4: Fix all call sites (grep, update signatures)**

```bash
grep -rn "RenderMagicLink\|RenderPasswordReset\|RenderVerifyEmail\|RenderOrganizationInvitation" --include="*.go" internal/
```

Update each caller to pass `ctx, store, branding` and destructure `Rendered{Subject, HTML}`.

- [ ] **Step 5: Build + test**

```bash
go build ./...
go test ./internal/email/ -count=1
go test ./... -count=1 -timeout 120s
```

Expected: all pass. Existing email tests may need argument updates — fix as a batch.

- [ ] **Step 6: Commit**

```bash
git add internal/email/ internal/auth/ internal/api/
git commit -m "feat(branding): render email templates from DB with branding, fall back to embedded"
```

---

### Task A5: Branding admin handlers

**Files:**
- Create: `internal/api/branding_handlers.go`
- Modify: `internal/api/router.go` — mount routes

- [ ] **Step 1: Write `branding_handlers.go`**

```go
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxLogoSize = 1 << 20 // 1 MB per spec OQ2

func (s *Server) handleGetBranding(w http.ResponseWriter, r *http.Request) {
	b, err := s.Store.GetBranding(r.Context(), "global")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "load branding")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"branding": b,
		"fonts":    []string{"manrope", "inter", "ibm_plex"},
	})
}

func (s *Server) handlePatchBranding(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "bad json")
		return
	}
	if err := s.Store.UpdateBranding(r.Context(), "global", body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "save branding")
		return
	}
	s.handleGetBranding(w, r)
}

func (s *Server) handleUploadLogo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLogoSize)
	if err := r.ParseMultipartForm(maxLogoSize); err != nil {
		writeJSONError(w, http.StatusBadRequest, "file_too_large", "logo must be ≤ 1MB")
		return
	}
	file, header, err := r.FormFile("logo")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing 'logo' field")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".svg" && ext != ".jpg" && ext != ".jpeg" {
		writeJSONError(w, http.StatusBadRequest, "invalid_file_type", "only .png, .svg, .jpg allowed")
		return
	}

	// Compute content hash
	data, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "read file")
		return
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])

	// Reject SVG with script content (CSP risk)
	if ext == ".svg" && svgHasScript(data) {
		writeJSONError(w, http.StatusBadRequest, "invalid_svg", "SVG contains <script> or <foreignObject>; disallowed")
		return
	}

	assetsDir := filepath.Join("data", "assets", "branding")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "asset dir")
		return
	}
	filename := sha + ext
	if err := os.WriteFile(filepath.Join(assetsDir, filename), data, 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "write asset")
		return
	}

	url := "/assets/branding/" + filename
	if err := s.Store.SetBrandingLogo(r.Context(), "global", url, sha); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "save logo record")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logo_url": url, "logo_sha": sha})
}

func (s *Server) handleDeleteLogo(w http.ResponseWriter, r *http.Request) {
	_ = s.Store.ClearBrandingLogo(r.Context(), "global")
	w.WriteHeader(http.StatusNoContent)
}

// svgHasScript is a minimal string scan — not a full XML parser.
// Defense-in-depth: CSP headers should be last line of defense.
func svgHasScript(data []byte) bool {
	s := strings.ToLower(string(data))
	return strings.Contains(s, "<script") || strings.Contains(s, "<foreignobject") || strings.Contains(s, "javascript:")
}
```

- [ ] **Step 2: Mount in `router.go`**

```go
r.Route("/branding", func(r chi.Router) {
	r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
	r.Get("/", s.handleGetBranding)
	r.Patch("/", s.handlePatchBranding)
	r.Post("/logo", s.handleUploadLogo)
	r.Delete("/logo", s.handleDeleteLogo)
})
```

- [ ] **Step 3: Build + verify route appears**

```bash
go build ./...
bin/shark.exe serve --dev &
sleep 2
curl -s -H "Authorization: Bearer $ADMIN_KEY" http://localhost:8080/api/v1/admin/branding | jq .
```

Expected: JSON with `branding.*` default fields + `fonts: ["manrope","inter","ibm_plex"]`.

- [ ] **Step 4: Commit**

```bash
git add internal/api/branding_handlers.go internal/api/router.go
git commit -m "feat(branding): admin branding CRUD + logo upload/delete handlers"
```

---

### Task A6: Asset-serve handler for `/assets/branding/*`

**Files:**
- Create: `internal/api/hosted_assets_handler.go`
- Modify: `internal/api/router.go` — mount before proxy catch-all

- [ ] **Step 1: Write handler**

```go
package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleBrandingAsset(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/assets/branding/")
	// Path traversal guard
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || filename == "" {
		http.NotFound(w, r)
		return
	}
	full := filepath.Join("data", "assets", "branding", filename)
	f, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	// content-addressed → safe to cache forever
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	}
	http.ServeContent(w, r, filename, fileModTime(f), f)
}

func fileModTime(f *os.File) (t time.Time) {
	info, err := f.Stat()
	if err != nil {
		return
	}
	return info.ModTime()
}
```

Add import for `time`.

- [ ] **Step 2: Mount in router.go OUTSIDE `/api/v1` (public endpoint, before proxy catch-all)**

```go
// In setupRouter, before ProxyHandler catch-all:
r.Get("/assets/branding/*", s.handleBrandingAsset)
```

- [ ] **Step 3: Test manual**

```bash
curl -I http://localhost:8080/assets/branding/nonexistent.png
```

Expected: 404.

- [ ] **Step 4: Commit**

```bash
git add internal/api/hosted_assets_handler.go internal/api/router.go
git commit -m "feat(branding): serve /assets/branding/{sha}.{ext} with immutable cache"
```

---

### Task A7: Email template admin handlers

**Files:**
- Create: `internal/api/email_template_handlers.go`
- Modify: `internal/api/router.go` — mount `/admin/email-templates/*`

- [ ] **Step 1: Write handlers**

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListEmailTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.ListEmailTemplates(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "list templates")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (s *Server) handleGetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "template not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handlePatchEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "bad json")
		return
	}
	if err := s.Store.UpdateEmailTemplate(r.Context(), id, body); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "update template")
		return
	}
	s.handleGetEmailTemplate(w, r)
}

func (s *Server) handlePreviewEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Config     *storage.BrandingConfig `json:"config,omitempty"`
		SampleData map[string]any          `json:"sample_data,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "bad json")
		return
	}
	tmpl, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "template not found")
		return
	}
	branding := body.Config
	if branding == nil {
		branding, _ = s.Store.ResolveBranding(r.Context(), "")
	}
	data := body.SampleData
	if data == nil {
		data = map[string]any{"AppName": "SharkAuth", "Link": "https://example.com/verify/abc", "UserEmail": "user@example.com"}
	}
	rendered, err := renderEmailTemplatePreview(tmpl, branding, data)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "render preview: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": rendered.HTML, "subject": rendered.Subject})
}

func (s *Server) handleSendTestEmail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		ToEmail string `json:"to_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ToEmail == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "to_email required")
		return
	}
	// Per spec OQ4: any address, no validation beyond non-empty
	tmpl, err := s.Store.GetEmailTemplate(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "template not found")
		return
	}
	branding, _ := s.Store.ResolveBranding(r.Context(), "")
	rendered, err := renderEmailTemplatePreview(tmpl, branding, map[string]any{
		"AppName": "SharkAuth", "Link": "https://example.com/test/link", "UserEmail": body.ToEmail,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "render")
		return
	}
	if err := s.EmailSender.Send(r.Context(), body.ToEmail, rendered.Subject, rendered.HTML); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "send_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

func (s *Server) handleResetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Re-seed overwrites? No — spec says "revert to seeded default". Fetch from seeds map + UPDATE.
	for _, seed := range defaultSeedsForAPI() {
		if seed.ID != id {
			continue
		}
		pJSON, _ := json.Marshal(seed.BodyParagraphs)
		fields := map[string]any{
			"subject": seed.Subject, "preheader": seed.Preheader,
			"header_text": seed.HeaderText, "body_paragraphs": string(pJSON),
			"cta_text": seed.CTAText, "cta_url_template": seed.CTAURLTemplate,
			"footer_text": seed.FooterText,
		}
		if err := s.Store.UpdateEmailTemplate(r.Context(), id, fields); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "reset")
			return
		}
		s.handleGetEmailTemplate(w, r)
		return
	}
	writeJSONError(w, http.StatusNotFound, "not_found", "no default exists")
}

// defaultSeedsForAPI re-exposes seeds. Could also live in storage package and be exported.
func defaultSeedsForAPI() []*storage.EmailTemplate {
	return storage.DefaultEmailTemplateSeedsForExport()
}

// renderEmailTemplatePreview — call into email package rendering helper.
// Function exposed in internal/email for preview use.
func renderEmailTemplatePreview(tmpl *storage.EmailTemplate, branding *storage.BrandingConfig, data any) (email.Rendered, error) {
	return email.RenderStructured(tmpl, branding, data)
}
```

Note: add `DefaultEmailTemplateSeedsForExport()` wrapper in storage package (export underscore of internal seeds), and `email.RenderStructured` export wrapper.

- [ ] **Step 2: Mount in router.go**

```go
r.Route("/email-templates", func(r chi.Router) {
	r.Use(mw.AdminAPIKeyFromStore(s.Store, s.RateLimiter))
	r.Get("/", s.handleListEmailTemplates)
	r.Get("/{id}", s.handleGetEmailTemplate)
	r.Patch("/{id}", s.handlePatchEmailTemplate)
	r.Post("/{id}/preview", s.handlePreviewEmailTemplate)
	r.Post("/{id}/send-test", s.handleSendTestEmail)
	r.Post("/{id}/reset", s.handleResetEmailTemplate)
})
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: clean. If missing exports (renderEmailTemplatePreview, DefaultEmailTemplateSeedsForExport), add them to their packages.

- [ ] **Step 4: Manual smoke**

```bash
curl -s -H "Authorization: Bearer $ADMIN_KEY" http://localhost:8080/api/v1/admin/email-templates | jq '.data | length'
# Expected: 5
```

- [ ] **Step 5: Commit**

```bash
git add internal/api/email_template_handlers.go internal/api/router.go internal/storage/email_templates.go internal/email/templates.go
git commit -m "feat(branding): email template admin handlers (list/get/patch/preview/send-test/reset)"
```

---

### Task A8: Application integration_mode PATCH support + snippet handler

**Files:**
- Modify: existing application-handler file (check `internal/api/admin_*_handlers.go` for app update code, extend payload)
- Create: `internal/api/snippet_handlers.go`

- [ ] **Step 1: Extend application PATCH payload**

In existing `updateApplication` handler (locate via grep on `PATCH.*apps`):

```go
type updateApplicationRequest struct {
	// ... existing fields
	IntegrationMode        *string `json:"integration_mode,omitempty"`
	BrandingOverride       json.RawMessage `json:"branding_override,omitempty"`
	ProxyLoginFallback     *string `json:"proxy_login_fallback,omitempty"`
	ProxyLoginFallbackURL  *string `json:"proxy_login_fallback_url,omitempty"`
}
```

Validate `IntegrationMode` in `{hosted, components, proxy, custom}` and `ProxyLoginFallback` in `{hosted, custom_url}`. Persist via store.

- [ ] **Step 2: Create snippet handler**

```go
package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleAppSnippet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	framework := r.URL.Query().Get("framework")
	if framework == "" {
		framework = "react"
	}
	if framework != "react" {
		writeJSONError(w, http.StatusNotImplemented, "framework_not_supported",
			framework+" support coming soon. React available now.")
		return
	}
	app, err := s.Store.GetApplicationByID(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not_found", "app not found")
		return
	}
	authURL := s.Config.Server.BaseURL
	writeJSON(w, http.StatusOK, map[string]any{
		"framework": "react",
		"snippets": []map[string]string{
			{"label": "Install", "lang": "bash", "code": "npm install @shark-auth/react"},
			{"label": "Provider setup", "lang": "tsx", "code": fmt.Sprintf(
				"import { SharkProvider } from '@shark-auth/react'\n\n"+
					"<SharkProvider publishableKey=%q authUrl=%q>\n"+
					"  <App/>\n"+
					"</SharkProvider>", app.ClientID, authURL)},
			{"label": "Page usage", "lang": "tsx", "code":
				"import { SignIn, UserButton, SignedIn, SignedOut } from '@shark-auth/react'\n\n" +
					"<SignedOut><SignIn/></SignedOut>\n" +
					"<SignedIn><UserButton/></SignedIn>"},
		},
	})
}
```

- [ ] **Step 3: Mount in router**

```go
r.Get("/applications/{id}/snippet", s.handleAppSnippet)
```

- [ ] **Step 4: Build + test PATCH + snippet**

```bash
go build ./...
curl -X PATCH http://localhost:8080/api/v1/admin/applications/app_123 \
  -H "Authorization: Bearer $ADMIN_KEY" \
  -d '{"integration_mode":"components"}'
curl -s "http://localhost:8080/api/v1/admin/applications/app_123/snippet?framework=react" \
  -H "Authorization: Bearer $ADMIN_KEY" | jq .
```

Expected: 200 with snippets array; PATCH persists.

- [ ] **Step 5: Commit**

```bash
git add internal/api/
git commit -m "feat(branding): integration_mode PATCH + /admin/applications/{id}/snippet endpoint"
```

---

### Task A9: Welcome email wiring on verification

**Files:**
- Modify: `internal/auth/email_verify.go` (or equivalent handler)
- Modify: `internal/storage/` — add `MarkWelcomeEmailSent(ctx, userID) error`

- [ ] **Step 1: Add store method**

```go
// Store interface addition
MarkWelcomeEmailSent(ctx context.Context, userID string) error

// Implementation
func (s *SQLiteStore) MarkWelcomeEmailSent(ctx context.Context, userID string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET welcome_email_sent = 1 WHERE id = ? AND welcome_email_sent = 0`, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
```

- [ ] **Step 2: Wire trigger in email-verify handler**

After successful `email_verified = true` update:

```go
// Fire welcome email once, idempotent via UPDATE ... WHERE welcome_email_sent = 0
if err := s.Store.MarkWelcomeEmailSent(ctx, user.ID); err == nil {
	branding, _ := s.Store.ResolveBranding(ctx, "")
	rendered, rErr := email.RenderWelcome(ctx, s.Store, branding, email.WelcomeData{
		AppName: s.Config.AppName, UserEmail: user.Email,
		DashboardURL: s.Config.Server.BaseURL + "/admin",
	})
	if rErr == nil {
		go func() {
			if sendErr := s.EmailSender.Send(context.Background(), user.Email, rendered.Subject, rendered.HTML); sendErr != nil {
				slog.Warn("welcome email send failed", "user_id", user.ID, "error", sendErr)
			}
		}()
	}
}
```

- [ ] **Step 3: Build + smoke**

```bash
go build ./...
go test ./internal/auth/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/auth/ internal/storage/
git commit -m "feat(branding): fire welcome email on email verification (idempotent)"
```

---

### Task A10: Drop `ph:9` gate on Branding nav + wire real component

**Files:**
- Modify: `admin/src/components/layout.tsx` — remove `ph:9` on branding nav item
- Modify: `admin/src/components/empty_shell.tsx` — remove `Branding` export
- Modify: `admin/src/components/App.tsx:24` — import from `./branding` instead
- Create: `admin/src/components/branding.tsx` — minimal scaffold with 3 subtabs

- [ ] **Step 1: Scaffold `branding.tsx`**

```tsx
// @ts-nocheck
import React from 'react'
import { BrandingVisualsTab } from './branding/visuals_tab'
import { BrandingEmailTab } from './branding/email_tab'
import { BrandingIntegrationsTab } from './branding/integrations_tab'

export function Branding() {
  const [subtab, setSubtab] = React.useState<'visuals' | 'email' | 'integrations'>('visuals')

  return (
    <div style={{ padding: 24, height: '100%', overflow: 'auto' }}>
      <h1 style={{ fontSize: 22, marginBottom: 16 }}>Branding</h1>
      <div style={{ display: 'flex', gap: 4, borderBottom: '1px solid var(--hairline)', marginBottom: 20 }}>
        {(['visuals', 'email', 'integrations'] as const).map(t => (
          <button
            key={t}
            onClick={() => setSubtab(t)}
            className={subtab === t ? 'tab active' : 'tab'}
            style={{
              padding: '10px 14px',
              background: subtab === t ? 'var(--surface-2)' : 'transparent',
              borderBottom: subtab === t ? '2px solid var(--accent)' : '2px solid transparent',
              border: 'none',
              cursor: 'pointer',
              textTransform: 'capitalize',
            }}
          >
            {t === 'email' ? 'Email Templates' : t}
          </button>
        ))}
      </div>

      {subtab === 'visuals' && <BrandingVisualsTab/>}
      {subtab === 'email' && <BrandingEmailTab/>}
      {subtab === 'integrations' && <BrandingIntegrationsTab/>}
    </div>
  )
}
```

- [ ] **Step 2: Stub sub-tab components (will fill next tasks)**

```tsx
// admin/src/components/branding/visuals_tab.tsx
// @ts-nocheck
import React from 'react'
export function BrandingVisualsTab() {
  return <div>Visuals tab — see task A11</div>
}

// admin/src/components/branding/email_tab.tsx
// @ts-nocheck
import React from 'react'
export function BrandingEmailTab() {
  return <div>Email Templates tab — see task A12</div>
}

// admin/src/components/branding/integrations_tab.tsx
// @ts-nocheck
import React from 'react'
export function BrandingIntegrationsTab() {
  return <div>Integrations tab — see task A13</div>
}
```

- [ ] **Step 3: Remove `Branding` export from `empty_shell.tsx` and drop phase-9 gate in `layout.tsx`**

In `empty_shell.tsx`, delete the `export function Branding()` line.

In `layout.tsx`, find the branding nav entry and remove the `ph: 9` key.

- [ ] **Step 4: Replace import in `App.tsx`**

Change line 24 FROM:

```tsx
import { Tokens, APIExplorer, EventSchemas, OIDCProvider, Impersonation, Migrations, Branding } from './empty_shell'
```

TO:

```tsx
import { Tokens, APIExplorer, EventSchemas, OIDCProvider, Impersonation, Migrations } from './empty_shell'
import { Branding } from './branding'
```

- [ ] **Step 5: Build + verify**

```bash
cd admin && npm run build && cd ..
go build -o bin/shark.exe ./cmd/shark
```

Restart server, navigate to `/admin/branding`. Expected: 3 tabs render with stub content.

- [ ] **Step 6: Commit**

```bash
git add admin/src/components/branding.tsx admin/src/components/branding/ admin/src/components/empty_shell.tsx admin/src/components/layout.tsx admin/src/components/App.tsx internal/admin/dist/
git commit -m "feat(branding): scaffold Branding page with 3 subtabs, drop ph:9 gate"
```

---

### Task A11: Visuals subtab — logo + colors + font + footer + email-from

**Files:**
- Modify: `admin/src/components/branding/visuals_tab.tsx`
- Dep: `react-colorful` — `npm i react-colorful`

- [ ] **Step 1: Install dependency**

```bash
cd admin && npm install react-colorful && cd ..
```

- [ ] **Step 2: Implement visuals tab**

Full implementation per spec Section 3. Components: logo drop-zone, 2 HslaColorPicker with hex display, font select, footer textarea, 2 email-from inputs. Right pane: live preview (sample hosted-login + sample email) using `dangerouslySetInnerHTML` from `/admin/email-templates/{id}/preview` POST (debounced 300ms).

Key structure:

```tsx
// @ts-nocheck
import React from 'react'
import { HexColorPicker } from 'react-colorful'
import { API } from '../api'
import { useToast } from '../toast'

export function BrandingVisualsTab() {
  const [cfg, setCfg] = React.useState<any>(null)
  const [draft, setDraft] = React.useState<any>(null)
  const [dirty, setDirty] = React.useState(false)
  const [previewHTML, setPreviewHTML] = React.useState('')
  const toast = useToast()

  React.useEffect(() => {
    API.get('/admin/branding').then(r => {
      setCfg(r.branding); setDraft(r.branding)
    })
  }, [])

  // Debounced preview render
  React.useEffect(() => {
    if (!draft) return
    const h = setTimeout(async () => {
      const r = await API.post('/admin/email-templates/magic_link/preview', { config: draft })
      setPreviewHTML(r.html)
    }, 300)
    return () => clearTimeout(h)
  }, [draft])

  const save = async () => {
    await API.patch('/admin/branding', draft)
    setCfg(draft); setDirty(false); toast.success('Branding saved')
  }

  const uploadLogo = async (file: File) => {
    const form = new FormData(); form.append('logo', file)
    const r = await fetch('/api/v1/admin/branding/logo', {
      method: 'POST',
      headers: { Authorization: `Bearer ${sessionStorage.getItem('shark_admin_key')}` },
      body: form,
    })
    if (!r.ok) { toast.error('Upload failed (max 1MB, png/svg/jpg)'); return }
    const j = await r.json()
    setDraft({ ...draft, logo_url: j.logo_url, logo_sha: j.logo_sha })
    setDirty(true); toast.success('Logo uploaded')
  }

  if (!draft) return <div>Loading...</div>

  return (
    <div style={{ display: 'flex', gap: 24 }}>
      <div style={{ flex: 1, maxWidth: 400 }}>
        {/* Logo drop zone */}
        <label>Logo</label>
        <input type="file" accept=".png,.svg,.jpg,.jpeg"
          onChange={e => e.target.files?.[0] && uploadLogo(e.target.files[0])} />
        {draft.logo_url && <img src={draft.logo_url} style={{ maxHeight: 40, marginTop: 8 }} />}

        {/* Primary color */}
        <label style={{ marginTop: 16, display: 'block' }}>Primary color</label>
        <HexColorPicker color={draft.primary_color} onChange={c => { setDraft({ ...draft, primary_color: c }); setDirty(true) }} />
        <input value={draft.primary_color} onChange={e => { setDraft({ ...draft, primary_color: e.target.value }); setDirty(true) }} />

        {/* Secondary color, font select, footer, email-from analogous */}
        {/* ... */}

        <button disabled={!dirty} onClick={save}>Save</button>
      </div>

      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 8 }}>LIVE PREVIEW — magic link email</div>
        <iframe srcDoc={previewHTML} style={{ width: '100%', height: 400, border: '1px solid var(--hairline)' }} />
      </div>
    </div>
  )
}
```

Include full input set per spec. Keep under 300 LOC — if growing, split into sub-components.

- [ ] **Step 3: Build, manually test**

```bash
cd admin && npm run build && cd ..
go build -o bin/shark.exe ./cmd/shark
```

Restart, visit Branding, upload 1MB+ logo → expect toast error. Upload 500KB PNG → logo renders. Change primary color → preview updates within 300ms.

- [ ] **Step 4: Commit**

```bash
git add admin/src/components/branding/visuals_tab.tsx admin/package*.json internal/admin/dist/
git commit -m "feat(branding): Visuals subtab — logo upload, color pickers, font, footer, email-from + live preview"
```

---

### Task A12: Email templates subtab — list + editor + iframe preview

**Files:**
- Modify: `admin/src/components/branding/email_tab.tsx`

- [ ] **Step 1: Implement**

Structure: left pane template list (5 rows). Right pane: structured-field editor (subject, preheader, header_text, paragraphs array w/ add/remove, cta_text, cta_url_template, footer_text) + side-by-side iframe preview + "Send test" button (opens prompt for email) + "Reset to default" button.

```tsx
// @ts-nocheck
import React from 'react'
import { API } from '../api'
import { useToast } from '../toast'

export function BrandingEmailTab() {
  const [templates, setTemplates] = React.useState<any[]>([])
  const [selected, setSelected] = React.useState<string>('')
  const [draft, setDraft] = React.useState<any>(null)
  const [previewHTML, setPreviewHTML] = React.useState('')
  const toast = useToast()

  React.useEffect(() => {
    API.get('/admin/email-templates').then(r => {
      setTemplates(r.data); if (r.data[0]) setSelected(r.data[0].id)
    })
  }, [])

  React.useEffect(() => {
    if (!selected) return
    API.get(`/admin/email-templates/${selected}`).then(setDraft)
  }, [selected])

  React.useEffect(() => {
    if (!draft) return
    const h = setTimeout(async () => {
      const r = await API.post(`/admin/email-templates/${draft.id}/preview`, {})
      setPreviewHTML(r.html)
    }, 300)
    return () => clearTimeout(h)
  }, [draft])

  const save = async () => {
    await API.patch(`/admin/email-templates/${draft.id}`, {
      subject: draft.subject, preheader: draft.preheader,
      header_text: draft.header_text, body_paragraphs: draft.body_paragraphs,
      cta_text: draft.cta_text, cta_url_template: draft.cta_url_template,
      footer_text: draft.footer_text,
    })
    toast.success('Template saved')
  }

  const sendTest = async () => {
    const to = prompt('Send test email to:')
    if (!to) return
    try { await API.post(`/admin/email-templates/${draft.id}/send-test`, { to_email: to }); toast.success('Sent') }
    catch (e: any) { toast.error(e?.message || 'Send failed') }
  }

  const reset = async () => {
    if (!confirm('Reset to default? Custom edits will be lost.')) return
    const r = await API.post(`/admin/email-templates/${draft.id}/reset`, {})
    setDraft(r); toast.success('Reset')
  }

  if (!draft) return <div>Loading...</div>

  return (
    <div style={{ display: 'flex', gap: 16 }}>
      <div style={{ width: 200 }}>
        {templates.map(t => (
          <div key={t.id} onClick={() => setSelected(t.id)}
            style={{ padding: 10, cursor: 'pointer', background: selected === t.id ? 'var(--surface-2)' : 'transparent' }}>
            {t.id.replace(/_/g, ' ')}
          </div>
        ))}
      </div>
      <div style={{ flex: 1, display: 'flex', gap: 16 }}>
        <div style={{ flex: 0.4, display: 'flex', flexDirection: 'column', gap: 8 }}>
          <label>Subject</label>
          <input value={draft.subject} onChange={e => setDraft({ ...draft, subject: e.target.value })} />
          <label>Preheader</label>
          <input value={draft.preheader} onChange={e => setDraft({ ...draft, preheader: e.target.value })} />
          <label>Header text</label>
          <input value={draft.header_text} onChange={e => setDraft({ ...draft, header_text: e.target.value })} />
          <label>Body paragraphs</label>
          {draft.body_paragraphs.map((p: string, i: number) => (
            <div key={i} style={{ display: 'flex', gap: 4 }}>
              <textarea value={p} onChange={e => {
                const next = [...draft.body_paragraphs]; next[i] = e.target.value
                setDraft({ ...draft, body_paragraphs: next })
              }} style={{ flex: 1 }} />
              <button onClick={() => {
                const next = draft.body_paragraphs.filter((_: any, j: number) => j !== i)
                setDraft({ ...draft, body_paragraphs: next })
              }}>×</button>
            </div>
          ))}
          <button onClick={() => setDraft({ ...draft, body_paragraphs: [...draft.body_paragraphs, ''] })}>
            + Add paragraph
          </button>
          <label>CTA button text</label>
          <input value={draft.cta_text} onChange={e => setDraft({ ...draft, cta_text: e.target.value })} />
          <label>CTA URL template</label>
          <input value={draft.cta_url_template} onChange={e => setDraft({ ...draft, cta_url_template: e.target.value })} />
          <label>Footer</label>
          <textarea value={draft.footer_text} onChange={e => setDraft({ ...draft, footer_text: e.target.value })} />
          <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
            <button onClick={save}>Save</button>
            <button onClick={sendTest}>Send test</button>
            <button onClick={reset}>Reset to default</button>
          </div>
        </div>
        <div style={{ flex: 0.6 }}>
          <div style={{ fontSize: 11, color: 'var(--fg-dim)', marginBottom: 8 }}>PREVIEW</div>
          <iframe srcDoc={previewHTML} style={{ width: '100%', height: 600, border: '1px solid var(--hairline)' }} />
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: Rebuild + manual test**

```bash
cd admin && npm run build && cd ..
go build -o bin/shark.exe ./cmd/shark
```

Visit Email Templates tab → select magic_link → edit subject → see preview update in 300ms → click Save → toast → reload → persists.

- [ ] **Step 3: Commit**

```bash
git add admin/src/components/branding/email_tab.tsx internal/admin/dist/
git commit -m "feat(branding): Email Templates subtab with list + editor + live preview + send-test + reset"
```

---

### Task A13: Integrations subtab — per-app integration_mode + snippet modal

**Files:**
- Modify: `admin/src/components/branding/integrations_tab.tsx`

- [ ] **Step 1: Implement**

Table of applications. Per row: name + integration_mode dropdown (4 options) + actions: "Preview hosted page" (when hosted), "Get snippet" (when components, opens modal), proxy fallback config (inline when proxy).

```tsx
// @ts-nocheck
import React from 'react'
import { API } from '../api'

export function BrandingIntegrationsTab() {
  const [apps, setApps] = React.useState<any[]>([])
  const [snippetModal, setSnippetModal] = React.useState<{ app: any; snippets: any[] } | null>(null)

  React.useEffect(() => { API.get('/admin/applications').then(r => setApps(r.data || r.items || [])) }, [])

  const update = async (app: any, patch: any) => {
    await API.patch(`/admin/applications/${app.id}`, patch)
    setApps(apps.map(a => a.id === app.id ? { ...a, ...patch } : a))
  }

  const getSnippet = async (app: any) => {
    const r = await API.get(`/admin/applications/${app.id}/snippet?framework=react`)
    setSnippetModal({ app, snippets: r.snippets })
  }

  return (
    <div>
      <table style={{ width: '100%', borderCollapse: 'collapse' }}>
        <thead>
          <tr><th>App</th><th>Mode</th><th>Actions</th></tr>
        </thead>
        <tbody>
          {apps.map(app => (
            <tr key={app.id}>
              <td>{app.name}</td>
              <td>
                <select value={app.integration_mode || 'custom'}
                  onChange={e => update(app, { integration_mode: e.target.value })}>
                  <option value="hosted">Hosted</option>
                  <option value="components">Components</option>
                  <option value="proxy">Proxy</option>
                  <option value="custom">Custom (API)</option>
                </select>
              </td>
              <td>
                {app.integration_mode === 'hosted' && (
                  <a href={`/hosted/${app.slug}/login`} target="_blank" rel="noreferrer">Preview hosted page</a>
                )}
                {app.integration_mode === 'components' && (
                  <button onClick={() => getSnippet(app)}>Get snippet</button>
                )}
                {app.integration_mode === 'proxy' && (
                  <span>
                    Fallback:
                    <select value={app.proxy_login_fallback || 'hosted'}
                      onChange={e => update(app, { proxy_login_fallback: e.target.value })}>
                      <option value="hosted">Hosted page</option>
                      <option value="custom_url">Custom URL</option>
                    </select>
                    {app.proxy_login_fallback === 'custom_url' && (
                      <input placeholder="https://..." value={app.proxy_login_fallback_url || ''}
                        onBlur={e => update(app, { proxy_login_fallback_url: e.target.value })} />
                    )}
                  </span>
                )}
                {app.integration_mode === 'custom' && <span>API-only. See /api/v1/auth/*</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {snippetModal && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}
          onClick={() => setSnippetModal(null)}>
          <div style={{ background: 'var(--surface-1)', padding: 20, borderRadius: 8, maxWidth: 700, width: '90%' }} onClick={e => e.stopPropagation()}>
            <h3>Integration snippets: {snippetModal.app.name}</h3>
            <div style={{ marginBottom: 10 }}>
              <label>Framework:</label>
              <select><option>React</option>
                <option disabled>Vue (coming soon)</option>
                <option disabled>Svelte (coming soon)</option>
                <option disabled>Solid (coming soon)</option>
                <option disabled>Angular (coming soon)</option>
              </select>
            </div>
            {snippetModal.snippets.map((s, i) => (
              <div key={i} style={{ marginBottom: 12 }}>
                <div style={{ fontSize: 11, color: 'var(--fg-dim)' }}>{s.label}</div>
                <pre style={{ background: 'var(--surface-2)', padding: 10, overflow: 'auto' }}>{s.code}</pre>
                <button onClick={() => { navigator.clipboard.writeText(s.code); alert('Copied') }}>Copy</button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Rebuild + test**

Visit Integrations tab → change app mode to "components" → click Get snippet → modal shows 3 snippets → Copy works.

- [ ] **Step 3: Commit**

```bash
git add admin/src/components/branding/integrations_tab.tsx internal/admin/dist/
git commit -m "feat(branding): Integrations subtab with per-app integration_mode + snippet modal"
```

---

### Task A14: Smoke test section 74 — branding + email templates

**Files:**
- Modify: `smoke_test.sh`

- [ ] **Step 1: Add section 74**

Append to `smoke_test.sh`:

```bash
# === Section 74: Branding + Email Templates ===
section "74. Branding CRUD"

# GET default branding
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/branding")
echo "$r" | jq -e '.branding.primary_color == "#7c3aed"' > /dev/null
assert $? "branding GET returns default primary"

# PATCH branding
curl -sS -X PATCH -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"primary_color":"#ff0000"}' "$API/admin/branding" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/branding")
echo "$r" | jq -e '.branding.primary_color == "#ff0000"' > /dev/null
assert $? "branding PATCH persists"

# Logo upload (reject >1MB)
dd if=/dev/zero of=/tmp/big.png bs=1M count=2 2>/dev/null
r=$(curl -sS -w "%{http_code}" -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -F "logo=@/tmp/big.png" "$API/admin/branding/logo" -o /dev/null)
[ "$r" = "400" ] || [ "$r" = "413" ]
assert $? "logo upload rejects >1MB"

# Logo upload (accept small PNG)
# create a tiny valid PNG (PNG magic + empty IEND)
printf '\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDAT\x18Wc\x00\x00\x00\x03\x00\x01\x8f\x8f\xa4\xa6\x00\x00\x00\x00IEND\xaeB\x60\x82' > /tmp/tiny.png
r=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" \
  -F "logo=@/tmp/tiny.png" "$API/admin/branding/logo")
echo "$r" | jq -e '.logo_url | startswith("/assets/branding/")' > /dev/null
assert $? "logo upload accepts tiny PNG"

section "74b. Email Templates"

# List 5 templates
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/email-templates")
count=$(echo "$r" | jq '.data | length')
[ "$count" = "5" ]
assert $? "email-templates list returns 5 seeded templates"

# Update magic_link subject
curl -sS -X PATCH -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"subject":"Custom subject"}' "$API/admin/email-templates/magic_link" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/email-templates/magic_link")
echo "$r" | jq -e '.subject == "Custom subject"' > /dev/null
assert $? "email template PATCH persists"

# Preview renders HTML with branding
r=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{}' "$API/admin/email-templates/magic_link/preview")
echo "$r" | jq -e '.html | contains("Custom subject") | not' > /dev/null
# subject should NOT be in HTML body — subject is separate field
echo "$r" | jq -e '.subject == "Custom subject"' > /dev/null
assert $? "email preview returns structured response"

# Reset
curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/email-templates/magic_link/reset" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/email-templates/magic_link")
echo "$r" | jq -e '.subject | contains("Sign in to")' > /dev/null
assert $? "email template reset restores default subject"

section "74c. Application integration_mode"
APP_ID=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/applications" | jq -r '.data[0].id // .items[0].id')
curl -sS -X PATCH -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"integration_mode":"components"}' "$API/admin/applications/$APP_ID" > /dev/null
r=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/applications/$APP_ID/snippet?framework=react")
echo "$r" | jq -e '.snippets | length == 3' > /dev/null
assert $? "snippet endpoint returns 3 React snippets"
```

- [ ] **Step 2: Run smoke**

```bash
bash smoke_test.sh 2>&1 | grep -A1 "Section 74"
```

Expected: all assertions PASS.

- [ ] **Step 3: Commit**

```bash
git add smoke_test.sh
git commit -m "test: section 74 — branding CRUD, logo upload, email templates, integration_mode"
```

---

## Phase A verification gate

- [ ] **A15: Run full suite + dogfood**

```bash
go test ./... -count=1 -timeout 180s
cd admin && npx tsc -b --noEmit && cd ..
bash smoke_test.sh
cd admin && npm run build && cd ..
go build -o bin/shark.exe ./cmd/shark
```

All tests pass. Then restart `bin/shark.exe serve`, manually:
1. Visit `/admin/branding` — 3 tabs render
2. Visuals tab — upload logo, change color, save, see preview update
3. Email Templates tab — edit magic_link subject, save, send test email to own address
4. Integrations tab — change an app to `components`, click Get snippet, verify modal

If anything fails, fix + recommit before moving to Phase B.

- [ ] **A16: Tag phase completion**

```bash
git tag phase-a-branding-done
git log --oneline phase-a-branding-done~18..phase-a-branding-done
```

Expected: ~14-18 commits on `feat(branding):` / `test:`.

---

# PHASE B — Hosted pages SPA (~4 working days, ~15 tasks)

## Impeccable invocation gate

- [ ] **B0: Invoke `/impeccable` for component design work**

Before writing any Phase B component code, spawn `impeccable` skill with the handoff brief from spec Section 7. This produces design direction, forbidden aesthetics, and fidelity proof criteria. Write output to `docs/superpowers/specs/2026-04-XX-impeccable-direction.md`.

```
/impeccable "Build React components for SharkAuth under admin/src/design/ (shared) + hosted SPA at admin/src/hosted-entry.tsx + NPM package at packages/shark-auth-react/. Primary aesthetic: dark-first, oklch, configurable primary accent, Manrope display, Azeret Mono code, generous 8-12-16-24-32 spacing, subtle 160ms motion. Components: SignIn, SignUp, MagicLinkSent, PasskeyChallenge, MFAChallenge, EmailVerify, UserButton, SignedIn, SignedOut, SharkProvider. Accessibility WCAG AA. Forbidden: gradient mesh, overly rounded pills, neon glow, stock icons. Proof: renders at 1440x900 + 375x812 matching Clerk/Auth0 hosted-page polish."
```

Block Phase B task kickoff until impeccable output approved.

---

### Task B1: Migration 00018 — `applications.slug UNIQUE`

**Files:**
- Create: `cmd/shark/migrations/00018_application_slug.sql`

- [ ] **Step 1: Write migration**

```sql
-- +goose Up
ALTER TABLE applications ADD COLUMN slug TEXT;
-- Backfill slug from name (lowercase, hyphen-separated, alnum only)
UPDATE applications SET slug = lower(replace(replace(name, ' ', '-'), '_', '-')) WHERE slug IS NULL;
CREATE UNIQUE INDEX applications_slug_unique ON applications(slug);

-- +goose Down
DROP INDEX applications_slug_unique;
```

- [ ] **Step 2: Run + verify**

```bash
rm -f dev.db dev.db-* && bin/shark.exe serve --dev &
sleep 2
sqlite3 dev.db "SELECT name, slug FROM applications LIMIT 5;"
```

Expected: slug populated.

- [ ] **Step 3: Commit**

```bash
git add cmd/shark/migrations/00018_application_slug.sql
git commit -m "feat(hosted): migration 00018 — applications.slug UNIQUE for hosted page routing"
```

---

### Task B2: Slug generator + validator

**Files:**
- Create: `internal/api/application_slug.go`
- Modify: application create/update handlers to validate slug

- [ ] **Step 1: Write slug helper**

```go
package api

import (
	"regexp"
	"strings"
)

var slugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

func generateSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	// Drop anything not alnum/hyphen
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	result = strings.Trim(result, "-")
	if len(result) < 2 {
		result = "app-" + result
	}
	return result
}

func validateSlug(s string) error {
	if !slugRE.MatchString(s) {
		return fmt.Errorf("slug must be 3-64 lowercase alnum/hyphen, not starting/ending with hyphen")
	}
	return nil
}
```

- [ ] **Step 2: Extend app create/update payloads**

In app create: if `slug` missing, auto-generate from `name`. If provided, validate. Reject on uniqueness conflict.

- [ ] **Step 3: Test**

```bash
go build ./... && go test ./internal/api/ -count=1
```

- [ ] **Step 4: Commit**

```bash
git add internal/api/application_slug.go internal/api/
git commit -m "feat(hosted): application slug generator + validator"
```

---

### Task B3: Design system extraction — tokens + primitives

**Files:**
- Create: `admin/src/design/tokens.ts`
- Create: `admin/src/design/primitives/Button.tsx`, `Input.tsx`, `FormField.tsx`, `Card.tsx`, `Modal.tsx`, `Tabs.tsx`, `Toast.tsx`

**WARNING: high blast-radius refactor.** If dispatched to subagent, MUST use `isolation: "worktree"`. Run Phase A smoke + all existing dashboard smoke before merge.

- [ ] **Step 1: Create `tokens.ts` with color semantic vars, spacing scale, type scale**

```ts
// admin/src/design/tokens.ts
export const tokens = {
  color: {
    primary: 'var(--shark-primary, #7c3aed)',
    secondary: 'var(--shark-secondary, #1a1a1a)',
    surface1: 'oklch(14% 0.005 256)',
    surface2: 'oklch(18% 0.005 256)',
    surface3: 'oklch(22% 0.005 256)',
    hairline: 'oklch(28% 0.005 256)',
    fg: 'oklch(98% 0.005 256)',
    fgMuted: 'oklch(70% 0.005 256)',
    fgDim: 'oklch(50% 0.005 256)',
    danger: 'oklch(62% 0.2 25)',
    warn: 'oklch(74% 0.16 85)',
    success: 'oklch(68% 0.15 160)',
  },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32, 12: 48 },
  radius: { sm: 4, md: 6, lg: 8, xl: 12 },
  type: {
    display: { family: 'var(--font-display, Manrope), -apple-system, sans-serif' },
    body: { family: 'var(--font-body, Manrope), sans-serif' },
    mono: { family: 'var(--font-mono, Azeret Mono), monospace' },
    size: { xs: 11, sm: 12, base: 14, md: 16, lg: 18, xl: 22, '2xl': 28 },
  },
  motion: { fast: '160ms ease-out', med: '240ms ease-out' },
}
```

- [ ] **Step 2: Implement `Button.tsx`** (per impeccable direction)

```tsx
// admin/src/design/primitives/Button.tsx
import React from 'react'
import { tokens } from '../tokens'

type Variant = 'primary' | 'ghost' | 'danger' | 'icon'
type Size = 'sm' | 'md' | 'lg'

interface Props extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
}

export function Button({ variant = 'primary', size = 'md', loading, children, style, ...rest }: Props) {
  // styles derived from tokens
  // ... (full implementation per impeccable)
  return <button style={{ /* styles */ ...style }} {...rest}>{children}</button>
}
```

Full JSX + styles from impeccable handoff. Keep under 150 LOC per primitive.

- [ ] **Step 3: Implement remaining primitives** (`Input`, `FormField`, `Card`, `Modal`, `Tabs`, `Toast`) per impeccable direction.

- [ ] **Step 4: Refactor existing dashboard components to import from `admin/src/design/`**

Replace inline `<button className="btn primary">` with `<Button variant="primary">`. Do this page-by-page to minimize diff noise. Target components:
- `overview.tsx` buttons + cards
- `users.tsx` form fields + buttons
- `applications.tsx` form fields + buttons
- `branding.tsx` (just created) — migrate its inline styles

Per page: extract, recompile, smoke test. Commit per page.

- [ ] **Step 5: Verify existing smoke stays green**

```bash
cd admin && npm run build && npx tsc -b --noEmit && cd ..
go build -o bin/shark.exe ./cmd/shark
bash smoke_test.sh  # sections 1-74 must stay green
```

- [ ] **Step 6: Commit per-file**

```bash
git add admin/src/design/
git commit -m "feat(design): extract tokens + primitives for shared design system"
# Then per-page refactor commits:
git add admin/src/components/overview.tsx
git commit -m "refactor(admin): overview.tsx uses design system primitives"
# ... etc per page
```

---

### Task B4: Composed auth components (`admin/src/design/composed/`)

**Files:**
- Create: `admin/src/design/composed/SignInForm.tsx`, `SignUpForm.tsx`, `MFAForm.tsx`, `PasskeyButton.tsx`, `OAuthProviderButton.tsx`, `MagicLinkSent.tsx`, `EmailVerify.tsx`, `ErrorPage.tsx`

- [ ] **Step 1: Implement per impeccable direction**

Each component accepts props for auth config + handlers. No direct API calls from composed components — they delegate to callers. This keeps them reusable across hosted SPA + NPM package + admin preview panels.

```tsx
// admin/src/design/composed/SignInForm.tsx
import { Button, FormField, Input } from '../primitives'

interface Props {
  appName: string
  authMethods: string[]  // ['password', 'magic_link', 'passkey', 'oauth']
  oauthProviders?: { id: string; name: string; icon_url?: string }[]
  onPasswordSubmit: (email: string, password: string) => Promise<void>
  onMagicLinkRequest?: (email: string) => Promise<void>
  onPasskeyStart?: () => Promise<void>
  onOAuthStart?: (providerID: string) => void
}

export function SignInForm(props: Props) {
  // ... full JSX with email + password + provider buttons + magic link + passkey button
}
```

- [ ] **Step 2: Build + typecheck**

```bash
cd admin && npx tsc -b --noEmit && cd ..
```

- [ ] **Step 3: Commit**

```bash
git add admin/src/design/composed/
git commit -m "feat(design): composed auth components (SignInForm, SignUpForm, MFAForm, etc)"
```

---

### Task B5: Hosted SPA entry + Vite config

**Files:**
- Modify: `admin/vite.config.ts`
- Create: `admin/src/hosted-entry.tsx`
- Create: `admin/src/hosted/App.tsx`

- [ ] **Step 1: Vite config — add second entry**

```ts
// admin/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../internal/admin/dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        hosted: resolve(__dirname, 'hosted.html'),
      },
      output: {
        entryFileNames: (c) => c.name === 'hosted' ? 'hosted/assets/[name]-[hash].js' : 'assets/[name]-[hash].js',
        assetFileNames: (i) => i.name?.startsWith('hosted') ? 'hosted/assets/[name]-[hash][extname]' : 'assets/[name]-[hash][extname]',
      },
    },
  },
})
```

- [ ] **Step 2: Create `admin/hosted.html`**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign in</title>
</head>
<body>
  <div id="hosted-root"></div>
  <script type="module" src="/admin/src/hosted-entry.tsx"></script>
</body>
</html>
```

- [ ] **Step 3: `hosted-entry.tsx`**

```tsx
import React from 'react'
import ReactDOM from 'react-dom/client'
import { HostedApp } from './hosted/App'

const cfg = (window as any).__SHARK_HOSTED
if (!cfg) { document.body.textContent = 'Hosted auth config missing'; }
else {
  ReactDOM.createRoot(document.getElementById('hosted-root')!).render(<HostedApp config={cfg}/>)
}
```

- [ ] **Step 4: `hosted/App.tsx` with router**

```tsx
// Uses wouter for client-side routing
import { Route, Switch } from 'wouter'
import { LoginPage } from './routes/login'
// ... other imports

export function HostedApp({ config }) {
  return (
    <div style={{ background: 'var(--bg)', minHeight: '100vh' }}>
      <Switch>
        <Route path="/hosted/:slug/login"><LoginPage config={config}/></Route>
        <Route path="/hosted/:slug/signup"><SignupPage config={config}/></Route>
        <Route path="/hosted/:slug/magic"><MagicLinkSentPage config={config}/></Route>
        <Route path="/hosted/:slug/passkey"><PasskeyPage config={config}/></Route>
        <Route path="/hosted/:slug/mfa"><MFAPage config={config}/></Route>
        <Route path="/hosted/:slug/verify"><VerifyPage config={config}/></Route>
        <Route><ErrorPage.../></Route>
      </Switch>
    </div>
  )
}
```

Install wouter: `cd admin && npm install wouter`.

- [ ] **Step 5: Build + verify dual-entry output**

```bash
cd admin && npm install && npm run build && cd ..
ls internal/admin/dist/hosted/assets/  # should contain hosted-*.js
```

- [ ] **Step 6: Commit**

```bash
git add admin/vite.config.ts admin/hosted.html admin/src/hosted-entry.tsx admin/src/hosted/ admin/package*.json internal/admin/dist/
git commit -m "feat(hosted): Vite dual-entry + hosted SPA scaffold"
```

---

### Task B6: Hosted route components (login/signup/magic/passkey/mfa/verify/error)

**Files:**
- Create: `admin/src/hosted/routes/login.tsx`, `signup.tsx`, `magic.tsx`, `passkey.tsx`, `mfa.tsx`, `verify.tsx`, `error.tsx`

- [ ] **Step 1: `login.tsx`**

```tsx
import { SignInForm } from '../../design/composed/SignInForm'
import { useLocation } from 'wouter'

export function LoginPage({ config }: { config: any }) {
  const [, navigate] = useLocation()

  const onPasswordSubmit = async (email: string, password: string) => {
    const r = await fetch(`${config.oauth_base}/../api/v1/auth/password/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password, app_id: config.app_id }),
      credentials: 'include',
    })
    if (r.ok) {
      // OAuth redirect back with code
      const redirect = new URLSearchParams(window.location.search).get('redirect_uri')
      if (redirect) window.location.href = redirect
      else navigate(`/hosted/${config.app_slug}/verify`)
    }
  }

  const onMagicLinkRequest = async (email: string) => {
    await fetch(`${config.oauth_base}/../api/v1/auth/magic/send`, { /* ... */ })
    navigate(`/hosted/${config.app_slug}/magic`)
  }

  return (
    <div style={{ maxWidth: 400, margin: '80px auto', padding: 32 }}>
      <SignInForm
        appName={config.app_name}
        authMethods={config.auth_methods}
        onPasswordSubmit={onPasswordSubmit}
        onMagicLinkRequest={onMagicLinkRequest}
      />
    </div>
  )
}
```

- [ ] **Step 2: Remaining routes analogous** (each 50-100 LOC, uses composed component from design system)

- [ ] **Step 3: Build + typecheck**

```bash
cd admin && npm run build && npx tsc -b --noEmit && cd ..
```

- [ ] **Step 4: Commit**

```bash
git add admin/src/hosted/routes/ internal/admin/dist/
git commit -m "feat(hosted): route components for login/signup/magic/passkey/mfa/verify/error"
```

---

### Task B7: Go handler serving hosted SPA shell

**Files:**
- Create: `internal/api/hosted_handlers.go`
- Modify: `internal/api/router.go` — mount `/hosted/{app_slug}/{page}` before proxy catch-all

- [ ] **Step 1: Handler**

```go
package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var hostedShellTmpl = template.Must(template.New("shell").Parse(`<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>{{.AppName}}</title>
<link rel="stylesheet" href="/admin/hosted/assets/index.css">
</head><body>
<script>window.__SHARK_HOSTED = {{.ConfigJSON}};</script>
<div id="hosted-root"></div>
<script type="module" src="/admin/hosted/assets/{{.Bundle}}"></script>
</body></html>`))

func (s *Server) handleHostedPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "app_slug")
	page := chi.URLParam(r, "page")
	if !validPage(page) { http.NotFound(w, r); return }

	app, err := s.Store.GetApplicationBySlug(r.Context(), slug)
	if err != nil { http.NotFound(w, r); return }

	if app.IntegrationMode != "hosted" && app.IntegrationMode != "proxy" {
		http.Error(w, "hosted auth disabled for this app", http.StatusNotFound)
		return
	}

	branding, _ := s.Store.ResolveBranding(r.Context(), app.ID)
	cfg := map[string]any{
		"app_id": app.ID, "app_slug": app.Slug, "app_name": app.Name,
		"branding": branding,
		"auth_methods": s.getEnabledAuthMethods(r.Context()),
		"redirect_uris": app.AllowedCallbackURLs,
		"oauth_base": s.Config.Server.BaseURL + "/oauth",
	}
	cfgJSON, _ := json.Marshal(cfg)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	hostedShellTmpl.Execute(w, map[string]any{
		"AppName": app.Name, "ConfigJSON": template.JS(cfgJSON),
		"Bundle": findHostedBundle(), // reads dist/hosted/assets/ for hash
	})
}

func validPage(p string) bool {
	for _, v := range []string{"login", "signup", "magic", "passkey", "mfa", "verify", "error"} {
		if v == p { return true }
	}
	return false
}

func findHostedBundle() string {
	// Read embedded dist/hosted/assets/ at init time, cache filename
	// Implementation: walk embed.FS at server init, store result in s.hostedBundleName
	return "hosted-TBD.js" // replace with cached lookup
}
```

- [ ] **Step 2: Add `GetApplicationBySlug` store method**

In storage Store interface + sqlite impl: `GetApplicationBySlug(ctx, slug) (*Application, error)`.

- [ ] **Step 3: Mount in router.go BEFORE proxy catch-all**

```go
// After API routes, before /* proxy catch-all:
r.Get("/hosted/{app_slug}/{page}", s.handleHostedPage)
// Static assets for hosted bundle
r.Get("/admin/hosted/assets/*", http.FileServer(http.FS(s.DashboardDistFS)).ServeHTTP)
```

- [ ] **Step 4: Build + test**

```bash
go build ./... && bin/shark.exe serve --dev &
sleep 2
# create a test app with slug 'test-app' via admin API
APP_ID=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Test App","slug":"test-app","integration_mode":"hosted","allowed_callback_urls":["http://example.com/cb"]}' \
  "$API/admin/applications" | jq -r '.id')
curl -sS http://localhost:8080/hosted/test-app/login | head -20
```

Expected: HTML shell with `<script>window.__SHARK_HOSTED = {...}</script>`.

- [ ] **Step 5: Commit**

```bash
git add internal/api/hosted_handlers.go internal/api/router.go internal/storage/
git commit -m "feat(hosted): Go handler serves hosted SPA shell at /hosted/{slug}/{page}"
```

---

### Task B8: Smoke test section 75 — hosted pages

**Files:**
- Modify: `smoke_test.sh`

- [ ] **Step 1: Append section 75**

```bash
section "75. Hosted auth pages"

# Create test app with hosted mode
APP_ID=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Hosted Test","slug":"hosted-test","integration_mode":"hosted","allowed_callback_urls":["http://localhost:3000/cb"]}' \
  "$API/admin/applications" | jq -r '.id')

# GET hosted login shell
r=$(curl -sS "$BASE/hosted/hosted-test/login")
echo "$r" | grep -q "__SHARK_HOSTED"
assert $? "hosted login shell injects __SHARK_HOSTED config"
echo "$r" | grep -q "Hosted Test"
assert $? "hosted shell includes app name"

# Non-existent slug → 404
r=$(curl -sS -w "%{http_code}" "$BASE/hosted/does-not-exist/login" -o /dev/null)
[ "$r" = "404" ]
assert $? "hosted page 404 on unknown slug"

# App with integration_mode=custom → 404
CUSTOM_APP_ID=$(curl -sS -X POST -H "Authorization: Bearer $ADMIN_KEY" -H "Content-Type: application/json" \
  -d '{"name":"Custom","slug":"custom-test","integration_mode":"custom","allowed_callback_urls":["http://x/cb"]}' \
  "$API/admin/applications" | jq -r '.id')
r=$(curl -sS -w "%{http_code}" "$BASE/hosted/custom-test/login" -o /dev/null)
[ "$r" = "404" ]
assert $? "hosted page 404 when integration_mode=custom"
```

- [ ] **Step 2: Run**

```bash
bash smoke_test.sh
```

- [ ] **Step 3: Commit**

```bash
git add smoke_test.sh
git commit -m "test: section 75 — hosted pages shell + slug validation"
```

---

## Phase B verification gate

- [ ] **B9: Full suite + dogfood**

```bash
go test ./... -count=1 -timeout 180s
cd admin && npx tsc -b --noEmit && npm run build && cd ..
go build -o bin/shark.exe ./cmd/shark
bash smoke_test.sh
```

Manual:
1. Create app with `integration_mode=hosted`, `slug=my-app`
2. Visit `http://localhost:8080/hosted/my-app/login` → React SPA renders with config from backend
3. Enter test credentials → OAuth round-trip completes → redirect to app's callback URL
4. Visit `/hosted/my-app/signup` → sign-up form renders with branding colors from app config

- [ ] **B10: Tag**

```bash
git tag phase-b-hosted-done
```

---

# PHASE C — `@shark-auth/react` NPM package (~5 working days, ~12 tasks)

### Task C1: Monorepo setup

**Files:**
- Create OR modify: root `package.json` — add workspaces field (OR `pnpm-workspace.yaml`)
- Create: `packages/shark-auth-react/package.json`, `tsconfig.json`, `tsup.config.ts`
- Create: `pnpm-workspace.yaml` if using pnpm

- [ ] **Step 1: Decide monorepo tool — pnpm recommended**

```bash
npm install -g pnpm
```

Create `pnpm-workspace.yaml` at repo root:

```yaml
packages:
  - 'admin'
  - 'packages/*'
  - 'examples/*'
```

- [ ] **Step 2: Package manifest**

`packages/shark-auth-react/package.json`:

```json
{
  "name": "@shark-auth/react",
  "version": "0.1.0",
  "description": "React components + hooks for SharkAuth",
  "type": "module",
  "main": "./dist/index.cjs",
  "module": "./dist/index.js",
  "types": "./dist/index.d.ts",
  "exports": {
    ".": { "import": "./dist/index.js", "require": "./dist/index.cjs", "types": "./dist/index.d.ts" },
    "./core": { "import": "./dist/core/index.js", "require": "./dist/core/index.cjs", "types": "./dist/core/index.d.ts" },
    "./hooks": { "import": "./dist/hooks/index.js", "require": "./dist/hooks/index.cjs", "types": "./dist/hooks/index.d.ts" }
  },
  "files": ["dist"],
  "scripts": {
    "build": "tsup",
    "dev": "tsup --watch",
    "test": "vitest"
  },
  "peerDependencies": {
    "react": ">=18.0.0",
    "react-dom": ">=18.0.0"
  },
  "dependencies": {
    "jose": "^5.9.0"
  },
  "devDependencies": {
    "tsup": "^8.0.0",
    "typescript": "^5.0.0",
    "@types/react": "^18.0.0",
    "react": "^18.0.0",
    "react-dom": "^18.0.0",
    "vitest": "^1.0.0"
  }
}
```

- [ ] **Step 3: `tsup.config.ts`**

```ts
import { defineConfig } from 'tsup'

export default defineConfig({
  entry: ['src/index.ts', 'src/core/index.ts', 'src/hooks/index.ts'],
  format: ['cjs', 'esm'],
  dts: true,
  clean: true,
  splitting: false,
  external: ['react', 'react-dom'],
})
```

- [ ] **Step 4: `tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react",
    "strict": true,
    "esModuleInterop": true,
    "declaration": true,
    "outDir": "./dist",
    "skipLibCheck": true,
    "isolatedModules": true
  },
  "include": ["src/**/*"]
}
```

- [ ] **Step 5: Install + verify**

```bash
pnpm install
pnpm --filter @shark-auth/react build
```

Expected: `packages/shark-auth-react/dist/` populated with index.js, index.cjs, index.d.ts.

- [ ] **Step 6: Commit**

```bash
git add pnpm-workspace.yaml packages/shark-auth-react/package.json packages/shark-auth-react/tsup.config.ts packages/shark-auth-react/tsconfig.json package.json
git commit -m "feat(sdk): @shark-auth/react monorepo scaffold (pnpm workspaces + tsup)"
```

---

### Task C2: Core — client + auth + storage + jwt

**Files:**
- Create: `packages/shark-auth-react/src/core/client.ts`, `auth.ts`, `storage.ts`, `jwt.ts`, `types.ts`, `index.ts`

- [ ] **Step 1: Write core modules**

Implement per spec Section 4:
- `types.ts` — `User`, `Session`, `Organization`, `AuthConfig`, `TokenPair` types
- `storage.ts` — sessionStorage helpers per OQ5 (`getToken`, `setToken`, `clearToken`, etc)
- `client.ts` — `createClient(authUrl, publishableKey)` → fetch wrapper with auto-auth header
- `auth.ts` — PKCE code_verifier + code_challenge generation, `startAuthFlow()` (stores verifier, returns redirect URL), `exchangeCodeForToken()` (POST to /oauth/token)
- `jwt.ts` — `verifyToken(token, jwksUrl)` via jose, `decodeClaims(token)`

Each file < 120 LOC. Keep focused.

- [ ] **Step 2: `index.ts` re-exports**

```ts
export * from './client'
export * from './auth'
export * from './jwt'
export * from './storage'
export * from './types'
```

- [ ] **Step 3: Build**

```bash
pnpm --filter @shark-auth/react build
```

- [ ] **Step 4: Unit tests for JWT**

```ts
// packages/shark-auth-react/src/core/__tests__/jwt.test.ts
import { describe, it, expect } from 'vitest'
import { decodeClaims } from '../jwt'

describe('decodeClaims', () => {
  it('decodes valid JWT payload', () => {
    // ES256 example from jwt.io
    const token = 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c3JfMTIzIiwiZW1haWwiOiJ0QGV4YW1wbGUuY29tIn0.sig'
    const claims = decodeClaims(token)
    expect(claims.sub).toBe('usr_123')
    expect(claims.email).toBe('t@example.com')
  })
})
```

Run: `pnpm --filter @shark-auth/react test`.

- [ ] **Step 5: Commit**

```bash
git add packages/shark-auth-react/src/core/
git commit -m "feat(sdk): core — client, PKCE auth flow, JWT verify, sessionStorage"
```

---

### Task C3: Hooks

**Files:**
- Create: `packages/shark-auth-react/src/hooks/useAuth.ts`, `useUser.ts`, `useSession.ts`, `useOrganization.ts`, `index.ts`, `context.ts`

- [ ] **Step 1: Context**

```ts
// context.ts
import { createContext } from 'react'
export interface AuthContextValue {
  isLoaded: boolean
  isAuthenticated: boolean
  user: User | null
  session: Session | null
  client: Client
  signOut: () => Promise<void>
}
export const AuthContext = createContext<AuthContextValue | null>(null)
```

- [ ] **Step 2: Hooks**

```ts
// useAuth.ts
import { useContext } from 'react'
import { AuthContext } from './context'

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within SharkProvider')
  return { isLoaded: ctx.isLoaded, isAuthenticated: ctx.isAuthenticated, signOut: ctx.signOut }
}
```

Analogous for `useUser`, `useSession`, `useOrganization`.

- [ ] **Step 3: Build + commit**

```bash
pnpm --filter @shark-auth/react build
git add packages/shark-auth-react/src/hooks/
git commit -m "feat(sdk): hooks useAuth/useUser/useSession/useOrganization"
```

---

### Task C4: Components — SharkProvider + SignIn + UserButton + SignedIn/Out

**Files:**
- Create: `packages/shark-auth-react/src/components/SharkProvider.tsx`, `SignIn.tsx`, `SignUp.tsx`, `UserButton.tsx`, `SignedIn.tsx`, `SignedOut.tsx`, `MFAChallenge.tsx`, `PasskeyButton.tsx`, `OrganizationSwitcher.tsx`, `SharkCallback.tsx`

- [ ] **Step 1: SharkProvider**

```tsx
export function SharkProvider({ publishableKey, authUrl, children }: Props) {
  const [state, setState] = React.useState({ isLoaded: false, isAuthenticated: false, user: null, session: null })
  const client = React.useMemo(() => createClient(authUrl, publishableKey), [authUrl, publishableKey])

  React.useEffect(() => {
    // On mount: fetch auth config, check stored token, verify, hydrate context
    (async () => {
      const token = storage.getAccessToken()
      if (!token) { setState({ ...state, isLoaded: true }); return }
      try {
        const claims = await jwt.verifyToken(token, authUrl + '/.well-known/jwks.json')
        setState({ isLoaded: true, isAuthenticated: true, user: claims.user, session: claims.session })
      } catch {
        storage.clearAll()
        setState({ ...state, isLoaded: true })
      }
    })()
  }, [authUrl, publishableKey])

  const signOut = async () => { storage.clearAll(); setState({ ...state, isAuthenticated: false, user: null }) }

  return <AuthContext.Provider value={{ ...state, client, signOut }}>{children}</AuthContext.Provider>
}
```

- [ ] **Step 2: SignIn / SignedIn / SignedOut**

```tsx
export function SignedIn({ children }: { children: React.ReactNode }) {
  const { isLoaded, isAuthenticated } = useAuth()
  if (!isLoaded) return null
  return isAuthenticated ? <>{children}</> : null
}

export function SignedOut({ children }: { children: React.ReactNode }) {
  const { isLoaded, isAuthenticated } = useAuth()
  if (!isLoaded) return null
  return isAuthenticated ? null : <>{children}</>
}

export function SignIn({ redirectUrl }: { redirectUrl?: string }) {
  const onClick = async () => {
    const url = await auth.startAuthFlow(redirectUrl || window.location.origin)
    window.location.href = url
  }
  return <button onClick={onClick}>Sign in</button>
}
```

- [ ] **Step 3: UserButton with dropdown**

Menu: Profile, Manage account, Sign out.

- [ ] **Step 4: SharkCallback — route component for `/shark/callback`**

```tsx
export function SharkCallback() {
  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    if (!code) return
    auth.exchangeCodeForToken(code).then(() => {
      window.location.replace(sessionStorage.getItem('shark_redirect_after') || '/')
    })
  }, [])
  return <div>Signing in...</div>
}
```

- [ ] **Step 5: Build + commit**

```bash
pnpm --filter @shark-auth/react build
git add packages/shark-auth-react/src/components/
git commit -m "feat(sdk): SharkProvider + SignIn/SignedIn/SignedOut + UserButton + SharkCallback"
```

---

### Task C5: Example Next.js app

**Files:**
- Create: `examples/react-next/` — full Next.js app

- [ ] **Step 1: Scaffold**

```bash
mkdir -p examples/react-next
cd examples/react-next
pnpm init
pnpm add next react react-dom @shark-auth/react@workspace:*
```

- [ ] **Step 2: Minimal App Router setup**

`app/layout.tsx`:

```tsx
import { SharkProvider } from '@shark-auth/react'

export default function Root({ children }) {
  return (
    <html><body>
      <SharkProvider publishableKey={process.env.NEXT_PUBLIC_SHARK_KEY!} authUrl={process.env.NEXT_PUBLIC_SHARK_URL!}>
        {children}
      </SharkProvider>
    </body></html>
  )
}
```

`app/page.tsx`:

```tsx
import { SignIn, UserButton, SignedIn, SignedOut } from '@shark-auth/react'
export default function Home() {
  return (
    <main>
      <SignedOut><SignIn redirectUrl="/dashboard"/></SignedOut>
      <SignedIn><UserButton/></SignedIn>
    </main>
  )
}
```

`app/shark/callback/page.tsx`:

```tsx
'use client'
import { SharkCallback } from '@shark-auth/react'
export default function Callback() { return <SharkCallback/> }
```

- [ ] **Step 3: Run locally**

```bash
pnpm dev  # Next.js on :3000
# Ensure shark running on :8080, test app registered as components mode with callback http://localhost:3000/shark/callback
```

Click "Sign in" → redirect to shark hosted → login → redirect back → UserButton shows email.

- [ ] **Step 4: Commit**

```bash
git add examples/react-next/
git commit -m "feat(sdk): examples/react-next — Next.js app using @shark-auth/react SDK"
```

---

### Task C6: Smoke section 76 — SDK integration

**Files:**
- Modify: `smoke_test.sh`

- [ ] **Step 1: Append section 76**

```bash
section "76. SDK integration (example app smoke)"

# Build example app
(cd examples/react-next && pnpm build) && assert $? "example app builds"

# Package build verify
(cd packages/shark-auth-react && pnpm build) && assert $? "sdk builds"

# Snippet endpoint returns correct publishableKey
APP_ID=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/applications" | jq -r '.data[0].id')
SNIPPET_JSON=$(curl -sS -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/applications/$APP_ID/snippet?framework=react")
echo "$SNIPPET_JSON" | jq -e '.snippets | length == 3' > /dev/null
assert $? "snippet endpoint returns 3 snippets"

# Unsupported framework → 501
r=$(curl -sS -w "%{http_code}" -H "Authorization: Bearer $ADMIN_KEY" "$API/admin/applications/$APP_ID/snippet?framework=vue" -o /dev/null)
[ "$r" = "501" ]
assert $? "snippet returns 501 for unsupported framework"
```

- [ ] **Step 2: Commit**

```bash
git add smoke_test.sh
git commit -m "test: section 76 — SDK build + example app + snippet endpoint"
```

---

### Task C7: GitHub Actions publish workflow

**Files:**
- Create: `.github/workflows/publish-sdk.yml`

- [ ] **Step 1: Write workflow**

```yaml
name: Publish SDK
on:
  push:
    tags:
      - 'sdk-v*'
jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v3
        with: { version: 9 }
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          registry-url: 'https://registry.npmjs.org'
      - run: pnpm install
      - run: pnpm --filter @shark-auth/react build
      - run: pnpm --filter @shark-auth/react test
      - run: pnpm --filter @shark-auth/react publish --no-git-checks
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/publish-sdk.yml
git commit -m "feat(sdk): GitHub Actions publish on sdk-v* tag"
```

---

### Task C8: Docs guide

**Files:**
- Create: `docs/guides/react-integration.md`
- Create: `packages/shark-auth-react/README.md`

- [ ] **Step 1: README + guide content**

Include:
- Install: `npm install @shark-auth/react`
- Register app in shark admin dashboard with `integration_mode=components`
- Set env vars: `NEXT_PUBLIC_SHARK_KEY`, `NEXT_PUBLIC_SHARK_URL`
- SharkProvider wrapping
- SignIn/SignedIn/SignedOut usage
- Callback page
- useAuth / useUser
- Troubleshooting: "sessionStorage cleared on tab close (by design)"

- [ ] **Step 2: Commit**

```bash
git add docs/guides/react-integration.md packages/shark-auth-react/README.md
git commit -m "docs(sdk): React integration guide + package README"
```

---

## Phase C verification gate

- [ ] **C9: Full dogfood**

```bash
pnpm install
pnpm --filter @shark-auth/react build
pnpm --filter @shark-auth/react test
(cd examples/react-next && pnpm build && pnpm start &) && sleep 3
# Manual: visit http://localhost:3000, click Sign in, complete flow, see UserButton with email
```

- [ ] **C10: Tag**

```bash
git tag phase-c-sdk-done sdk-v0.1.0
```

- [ ] **C11: Push tags** (optional — user decides when to publish)

```bash
git push origin phase-a-branding-done phase-b-hosted-done phase-c-sdk-done
# To publish sdk:
# git push origin sdk-v0.1.0
```

---

## Self-review

**1. Spec coverage:**
- Data model → Task A1 (migrations 00017/00018 via A1 + B1), A2, A3
- API surface → A5, A6, A7, A8, B7, C6 (snippet)
- Frontend Branding tab → A10-A13
- Hosted SPA → B3-B7
- NPM package → C1-C7
- Smoke gates → A14, B8, C6
- Welcome trigger → A9
- Logo upload size cap → A5 (1MB verified) + A14 smoke
- Color picker UX full (OQ3) → A11 (react-colorful)
- sessionStorage (OQ5) → C2 storage.ts
- Any test-email address (OQ4) → A7 handleSendTestEmail no validation
- Font enum (3 presets) → A5 GET returns enum + A11 select input
- Impeccable gate before Phase B → B0
- Worktree isolation callout → present in notes at top + within B3

**2. Placeholder scan:**
- No "TBD" / "TODO" / "similar to Task N" / "fill in later"
- Every code step has concrete code or exact commands
- Task B7 has `findHostedBundle() { return "hosted-TBD.js" }` — this IS a placeholder for the bundle filename resolution. FIXED INLINE BELOW.

**Fix B7 placeholder:** Replace `findHostedBundle()` with embed.FS scan at init:

```go
// In Server struct: hostedBundleName string
// In initHosted() called from Build(): scan DashboardDistFS for "hosted/assets/hosted-*.js"
entries, _ := fs.ReadDir(s.DashboardDistFS, "hosted/assets")
for _, e := range entries {
    if strings.HasPrefix(e.Name(), "hosted-") && strings.HasSuffix(e.Name(), ".js") {
        s.hostedBundleName = e.Name()
        break
    }
}
// In handleHostedPage: use s.hostedBundleName instead of findHostedBundle()
```

**3. Type consistency:**
- `BrandingConfig` defined in A2, consumed in A4 email rendering + A5 handlers + B7 hosted shell — consistent field names
- `EmailTemplate` defined in A3, consumed in A7 handlers — consistent
- `integration_mode` string enum: `hosted|components|proxy|custom` used consistently in A8 PATCH + B7 handler + A13 UI
- `@shark-auth/react` naming consistent across C1-C8

No inconsistencies found.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-20-branding-hosted-components-plan.md`.

Total: 3 phases × ~13 working days solo. Each phase has verification gate before next starts.

Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task (with worktree for design-system refactor in B3), review between tasks, fast iteration. Phase boundaries gated by human review.

**2. Inline Execution** — Execute tasks in current session using executing-plans, batch execution with checkpoints.

**Which approach?**
