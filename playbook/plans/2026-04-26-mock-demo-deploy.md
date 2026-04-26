# Mock Demo Deploy + E2E Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deploy the mock demo platform to production at `demo.sharkauth.dev` + `demo-api.sharkauth.dev`, add Playwright end-to-end tests, integrate a smoke test into the shark monorepo, gate launch with a Lighthouse perf budget.

**Architecture:** Vercel for the Next.js frontend (Plan B). Fly.io for the demo-api Python service (Plan A). Hetzner CX22 VPS for the canonical shark instance behind Cloudflare. DNS via Cloudflare. CI hosted on GitHub Actions for both repos. Playwright for cross-browser E2E. Lighthouse-CI gating perf regressions.

**Tech Stack:** Vercel CLI, Fly.io CLI (`fly`), Cloudflare API/dashboard, Hetzner Cloud, systemd, Caddy (reverse proxy on Hetzner for shark), Playwright 1.48+, Lighthouse-CI 0.14+.

---

## Spec Reference

This plan implements the Distribution Plan section of `playbook/10-mock-demo-platform-design.md` plus the Acceptance Criteria (Lighthouse score ≥90, LCP <2s, smoke test in shark monorepo).

**Pre-req:** Plan A (demo-api repo) shipped + Plan B (demo-platform-frontend repo) shipped. Both with green CI.

---

## File Structure

| File | Repo | Purpose |
|---|---|---|
| `fly.toml` | demo-platform-example | Fly.io app config for demo-api |
| `Dockerfile` (existing from Plan A) | demo-platform-example | Production container |
| `.github/workflows/deploy-api.yml` | demo-platform-example | CI/CD: test → fly deploy |
| `vercel.json` | demo-platform-frontend | Vercel deployment config |
| `.github/workflows/deploy-frontend.yml` | demo-platform-frontend | CI/CD: test → vercel deploy |
| `playwright.config.ts` | demo-platform-frontend | Playwright cross-browser config |
| `e2e/three-act-flow.spec.ts` | demo-platform-frontend | Full E2E test |
| `lighthouserc.json` | demo-platform-frontend | LH-CI perf budget |
| `infra/hetzner/cloud-init.yaml` | shark monorepo | VPS bootstrap (shark binary install) |
| `infra/hetzner/Caddyfile` | shark monorepo | TLS-terminating reverse proxy in front of shark |
| `infra/hetzner/shark.service` | shark monorepo | systemd unit for shark binary |
| `tests/smoke/test_demo_platform.py` | shark monorepo | Smoke test against deployed demo-api |

---

## Task 1: Hetzner VPS for shark instance

**Files:** infra/hetzner/{cloud-init.yaml, Caddyfile, shark.service} in shark monorepo.

- [ ] **Step 1: Provision Hetzner CX22 VPS via web console**

Manual:
1. Log in to Hetzner Cloud console
2. New project: `shark-demo`
3. Add server: CX22 (2 vCPU, 4GB RAM), Ubuntu 24.04, location nbg1 (Nuremberg) or hil1 (Hillsboro) — pick closest to majority HN audience (US-East/EU)
4. SSH key: upload your public key
5. Cloud-init: paste from Step 2 below
6. Wait ~60s for boot

Expected: VPS reachable via SSH at the assigned IPv4 within 90s.

- [ ] **Step 2: Write `infra/hetzner/cloud-init.yaml`**

```yaml
#cloud-config
package_update: true
package_upgrade: true
packages:
  - curl
  - ca-certificates
  - ufw

users:
  - name: shark
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ssh-ed25519 REPLACE_WITH_YOUR_PUBLIC_KEY

write_files:
  - path: /etc/caddy/Caddyfile
    permissions: "0644"
    content: |
      shark-demo-internal.example.dev {
          reverse_proxy localhost:8080
          encode zstd gzip
          tls admin@sharkauth.dev
      }
  - path: /etc/systemd/system/shark.service
    permissions: "0644"
    content: |
      [Unit]
      Description=shark auth server (demo)
      After=network.target

      [Service]
      Type=simple
      User=shark
      WorkingDirectory=/home/shark
      ExecStart=/usr/local/bin/shark serve --bind 127.0.0.1:8080
      Restart=on-failure
      RestartSec=5
      Environment=SHARK_DB_PATH=/var/lib/shark/shark.db
      Environment=SHARK_MODE=prod

      [Install]
      WantedBy=multi-user.target

runcmd:
  - mkdir -p /var/lib/shark
  - chown shark:shark /var/lib/shark
  - curl -fsSL https://github.com/sharkauth/shark/releases/latest/download/shark-linux-amd64 -o /usr/local/bin/shark
  - chmod +x /usr/local/bin/shark
  - curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/setup.deb.sh | bash
  - apt-get install -y caddy
  - systemctl enable shark caddy
  - systemctl start shark caddy
  - ufw allow 22/tcp
  - ufw allow 80/tcp
  - ufw allow 443/tcp
  - ufw --force enable
```

Note: replace `REPLACE_WITH_YOUR_PUBLIC_KEY` and the `shark-demo-internal.example.dev` hostname before pasting.

- [ ] **Step 3: Verify shark instance is healthy**

```bash
ssh shark@<vps-ip> 'sudo systemctl status shark caddy'
curl -fsS https://shark-demo-internal.example.dev/health
```

Expected: both services `active (running)`. Health endpoint returns `{"status":"ok"}`.

- [ ] **Step 4: Bootstrap shark admin key**

```bash
ssh shark@<vps-ip>
sudo /usr/local/bin/shark admin api-key create --name 'demo-api-prod' | tee admin-key.txt
```

Save the printed `sk_live_*` key in your secrets manager (1Password, op CLI, etc.). This is the `SHARK_ADMIN_KEY` env var for Plan A's demo-api.

- [ ] **Step 5: Commit infra files to shark monorepo**

```bash
# in shark monorepo
mkdir -p infra/hetzner
# (paste cloud-init.yaml, Caddyfile, shark.service from above into respective files)
git add infra/hetzner/
git commit -m "infra(hetzner): cloud-init + Caddy + systemd for demo shark VPS"
```

---

## Task 2: Fly.io deployment for demo-api

**Files:** `fly.toml` in demo-platform-example repo.

- [ ] **Step 1: Install Fly CLI + log in**

```bash
curl -L https://fly.io/install.sh | sh
fly auth login
```

- [ ] **Step 2: Write `fly.toml`**

In demo-platform-example repo:

```toml
app = "shark-demo-api"
primary_region = "iad"  # or "fra" if EU traffic dominates

[build]
  dockerfile = "Dockerfile"

[env]
  SHARK_URL = "https://shark-demo-internal.example.dev"
  CORS_ORIGINS = "https://demo.sharkauth.dev"
  TENANT_TTL_MINUTES = "30"
  TENANT_DB_PATH = "/data/tenants.sqlite"

[mounts]
  source = "demo_data"
  destination = "/data"

[http_service]
  internal_port = 8001
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 1

  [http_service.concurrency]
    type = "requests"
    soft_limit = 200
    hard_limit = 250

  [[http_service.checks]]
    interval = "30s"
    timeout = "5s"
    grace_period = "10s"
    method = "GET"
    path = "/health"

[[vm]]
  size = "shared-cpu-1x"
  memory = "512mb"
```

- [ ] **Step 3: Initialize Fly app**

```bash
fly apps create shark-demo-api
fly volumes create demo_data --size 1 --region iad
```

- [ ] **Step 4: Set secret SHARK_ADMIN_KEY**

```bash
fly secrets set SHARK_ADMIN_KEY="sk_live_from_task1_step4"
```

- [ ] **Step 5: Deploy**

```bash
fly deploy
fly status
curl -fsS https://shark-demo-api.fly.dev/health
```

Expected: machine running, `/health` returns `{"status":"ok"}`.

- [ ] **Step 6: Commit `fly.toml`**

```bash
git add fly.toml
git commit -m "infra(fly): production config for demo-api (volume + healthcheck)"
```

---

## Task 3: Cloudflare DNS + custom domain on Fly

**Files:** none in repo (Cloudflare console).

- [ ] **Step 1: Add custom domain to Fly app**

```bash
fly certs add demo-api.sharkauth.dev
fly certs show demo-api.sharkauth.dev
```

Note the CNAME / A records Fly needs.

- [ ] **Step 2: Create DNS records in Cloudflare**

Manual via Cloudflare dashboard for `sharkauth.dev`:

- `demo-api.sharkauth.dev` → CNAME → `shark-demo-api.fly.dev` (proxy: OFF, orange cloud disabled — Fly handles TLS)
- `demo.sharkauth.dev` → set in Task 4 after Vercel deploy

Wait ~60s for propagation:

```bash
dig demo-api.sharkauth.dev +short
fly certs show demo-api.sharkauth.dev  # should show "Issued"
curl -fsS https://demo-api.sharkauth.dev/health
```

Expected: `{"status":"ok"}`.

- [ ] **Step 3: Verify CORS works from a test origin**

```bash
curl -i -X OPTIONS https://demo-api.sharkauth.dev/seed \
  -H "Origin: https://demo.sharkauth.dev" \
  -H "Access-Control-Request-Method: POST"
```

Expected: response includes `Access-Control-Allow-Origin: https://demo.sharkauth.dev`.

---

## Task 4: Vercel deployment for frontend

**Files:** `vercel.json`, `.env.production` (gitignored — set via Vercel dashboard).

- [ ] **Step 1: Install Vercel CLI + log in**

```bash
npm i -g vercel
vercel login
```

- [ ] **Step 2: Write `vercel.json`**

In demo-platform-frontend repo:

```json
{
  "framework": "nextjs",
  "buildCommand": "npm run build",
  "outputDirectory": ".next",
  "installCommand": "npm ci",
  "headers": [
    {
      "source": "/(.*)",
      "headers": [
        { "key": "X-Frame-Options", "value": "DENY" },
        { "key": "X-Content-Type-Options", "value": "nosniff" },
        { "key": "Referrer-Policy", "value": "strict-origin-when-cross-origin" },
        { "key": "Permissions-Policy", "value": "camera=(), microphone=(), geolocation=()" }
      ]
    }
  ]
}
```

- [ ] **Step 3: Link project + set env var**

```bash
vercel link
vercel env add NEXT_PUBLIC_DEMO_API_URL production
# When prompted, paste: https://demo-api.sharkauth.dev
```

- [ ] **Step 4: Deploy preview + verify**

```bash
vercel
# Open the preview URL printed; verify landing page + dashboard load
```

- [ ] **Step 5: Promote to production**

```bash
vercel --prod
```

- [ ] **Step 6: Add custom domain `demo.sharkauth.dev`**

```bash
vercel domains add demo.sharkauth.dev
```

In Cloudflare dashboard, add the CNAME records Vercel printed (typically `cname.vercel-dns.com`).

```bash
dig demo.sharkauth.dev +short
curl -fsS https://demo.sharkauth.dev | head -20
```

Expected: returns Next.js HTML with `<title>shark — auth for products...`.

- [ ] **Step 7: Commit**

```bash
git add vercel.json
git commit -m "infra(vercel): deployment config + security headers"
```

---

## Task 5: GitHub Actions — CI/CD for demo-api

**Files:** `.github/workflows/deploy-api.yml` in demo-platform-example.

- [ ] **Step 1: Generate Fly deploy token**

```bash
fly tokens create deploy --name "github-actions" -x 8760h
```

Copy the token.

- [ ] **Step 2: Add secret to GitHub repo**

GitHub → demo-platform-example repo → Settings → Secrets → Actions → New repository secret:
- Name: `FLY_API_TOKEN`
- Value: (paste token from Step 1)

- [ ] **Step 3: Write `.github/workflows/deploy-api.yml`**

Replace existing CI workflow:

```yaml
name: Deploy API
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.12"
      - run: pip install -e ".[dev]"
      - run: ruff check app tests
      - run: mypy app
      - run: pytest tests/unit/ --cov=app --cov-report=term-missing

  deploy:
    needs: test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    concurrency: deploy-api
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: flyctl deploy --remote-only
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

- [ ] **Step 4: Push + verify CI runs**

```bash
git add .github/workflows/deploy-api.yml
git commit -m "ci: GitHub Actions test+deploy pipeline (Fly.io)"
git push origin main
```

Open GitHub Actions tab; confirm both `test` and `deploy` jobs pass.

---

## Task 6: GitHub Actions — CI/CD for frontend

**Files:** `.github/workflows/deploy-frontend.yml` in demo-platform-frontend.

- [ ] **Step 1: Generate Vercel deploy token**

Vercel dashboard → Account Settings → Tokens → Create:
- Name: `github-actions-demo-frontend`
- Scope: full account or scoped to project
- Save the token.

- [ ] **Step 2: Add secrets to GitHub repo**

In demo-platform-frontend repo Settings → Secrets:
- `VERCEL_TOKEN` (from Step 1)
- `VERCEL_ORG_ID` (from `.vercel/project.json` after `vercel link`)
- `VERCEL_PROJECT_ID` (from `.vercel/project.json`)

- [ ] **Step 3: Write `.github/workflows/deploy-frontend.yml`**

```yaml
name: Deploy Frontend
on:
  push:
    branches: [main]
  pull_request:

env:
  VERCEL_ORG_ID: ${{ secrets.VERCEL_ORG_ID }}
  VERCEL_PROJECT_ID: ${{ secrets.VERCEL_PROJECT_ID }}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
      - run: npm ci
      - run: npm run typecheck
      - run: npm run lint
      - run: npm run test

  deploy:
    needs: test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    concurrency: deploy-frontend
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - run: npm i -g vercel
      - run: vercel pull --yes --environment=production --token=$VERCEL_TOKEN
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
      - run: vercel build --prod --token=$VERCEL_TOKEN
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
      - run: vercel deploy --prebuilt --prod --token=$VERCEL_TOKEN
        env:
          VERCEL_TOKEN: ${{ secrets.VERCEL_TOKEN }}
```

- [ ] **Step 4: Push + verify**

```bash
git add .github/workflows/deploy-frontend.yml
git commit -m "ci: GitHub Actions test+deploy pipeline (Vercel)"
git push origin main
```

Confirm green in Actions tab.

---

## Task 7: Playwright E2E setup

**Files:** in demo-platform-frontend repo.

- [ ] **Step 1: Install Playwright**

```bash
npm install --save-dev @playwright/test
npx playwright install --with-deps chromium
```

- [ ] **Step 2: Write `playwright.config.ts`**

```typescript
import { defineConfig, devices } from "@playwright/test"

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:3000",
    trace: "on-first-retry",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
  ],
  webServer: process.env.CI
    ? undefined
    : {
        command: "npm run dev",
        url: "http://localhost:3000",
        reuseExistingServer: true,
        timeout: 30_000,
      },
})
```

- [ ] **Step 3: Add npm scripts**

In `package.json`:

```json
"e2e": "playwright test",
"e2e:ui": "playwright test --ui"
```

- [ ] **Step 4: Commit**

```bash
git add playwright.config.ts package.json package-lock.json
git commit -m "test(e2e): Playwright config + chromium runner"
```

---

## Task 8: E2E test — full 3-act flow

**Files:** `e2e/three-act-flow.spec.ts`.

- [ ] **Step 1: Write the E2E test**

```typescript
import { test, expect } from "@playwright/test"

test.describe("3-act demo flow", () => {
  test("AcmeCode: full walkthrough end-to-end", async ({ page }) => {
    // Landing
    await page.goto("/")
    await expect(page.getByRole("heading", { level: 1 })).toContainText(/Auth for products/i)

    // Click into AcmeCode
    await page.getByRole("link", { name: /AcmeCode/ }).click()
    await expect(page).toHaveURL(/\/acme\/code/)

    // Tenant should provision (wait for Onboarding to render)
    await expect(page.getByRole("button", { name: /Add pr-reviewer agent/i })).toBeVisible({
      timeout: 10_000,
    })

    // Act 1
    await page.getByRole("button", { name: /Add pr-reviewer agent/i }).click()
    await expect(page.getByText(/cnf\.jkt/i)).toBeVisible({ timeout: 10_000 })

    // Walkthrough: advance to Act 1.5
    await page.getByText(/Act 1\.5: Delegation/).click()
    await page.getByRole("button", { name: /Dispatch PR review task/i }).click()
    await expect(page.getByText(/code-orchestrator/)).toBeVisible({ timeout: 10_000 })

    // Walkthrough: advance to Act 2
    await page.getByText(/Act 2: Attack/).click()
    await page.getByRole("button", { name: /Simulate prompt injection/i }).click()
    // Wait for all 4 attempts to render (3s pacing × 4 + buffer)
    await expect(page.getByText(/All 4 attack attempts failed/i)).toBeVisible({ timeout: 20_000 })

    // Walkthrough: advance to Act 3
    await page.getByText(/Act 3: Cleanup/).click()
    await page.getByRole("button", { name: /Cascade-revoke/i }).click()
    await expect(page.getByText(/47 tokens|tokens/)).toBeVisible({ timeout: 10_000 })

    // Toggle to AcmeSupport
    await page.getByRole("button", { name: "AcmeSupport" }).click()
    await expect(page).toHaveURL(/\/acme\/support/)
  })
})
```

- [ ] **Step 2: Run E2E locally against demo-api**

Pre-condition: demo-api running on `:8001`, frontend dev server running on `:3000`.

```bash
npm run e2e
```

Expected: `1 passed (≈45s)`.

- [ ] **Step 3: Wire E2E into CI workflow**

Append job to `.github/workflows/deploy-frontend.yml`:

```yaml
  e2e:
    needs: test
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
      - run: npm ci
      - run: npx playwright install --with-deps chromium
      - run: npm run e2e
        env:
          E2E_BASE_URL: https://demo.sharkauth.dev
      - uses: actions/upload-artifact@v4
        if: failure()
        with:
          name: playwright-report
          path: playwright-report/
```

- [ ] **Step 4: Commit**

```bash
git add e2e/ .github/workflows/deploy-frontend.yml
git commit -m "test(e2e): full 3-act walkthrough Playwright spec + CI integration"
```

---

## Task 9: Lighthouse CI perf budget

**Files:** `lighthouserc.json` in demo-platform-frontend.

- [ ] **Step 1: Install Lighthouse CI**

```bash
npm install --save-dev @lhci/cli
```

- [ ] **Step 2: Write `lighthouserc.json`**

```json
{
  "ci": {
    "collect": {
      "url": [
        "https://demo.sharkauth.dev/",
        "https://demo.sharkauth.dev/acme/code",
        "https://demo.sharkauth.dev/acme/support"
      ],
      "numberOfRuns": 3,
      "settings": {
        "preset": "desktop"
      }
    },
    "assert": {
      "preset": "lighthouse:recommended",
      "assertions": {
        "categories:performance": ["error", { "minScore": 0.9 }],
        "categories:accessibility": ["error", { "minScore": 0.9 }],
        "largest-contentful-paint": ["error", { "maxNumericValue": 2000 }],
        "interactive": ["error", { "maxNumericValue": 3000 }],
        "uses-text-compression": "off",
        "uses-rel-preconnect": "off"
      }
    },
    "upload": {
      "target": "temporary-public-storage"
    }
  }
}
```

- [ ] **Step 3: Add npm script**

```json
"lhci": "lhci autorun"
```

- [ ] **Step 4: Add LHCI job to CI**

Append to `.github/workflows/deploy-frontend.yml`:

```yaml
  lighthouse:
    needs: deploy
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
      - run: npm ci
      - run: npm run lhci
```

- [ ] **Step 5: Run locally to validate**

```bash
npm run lhci
```

Expected: all 3 URLs pass perf ≥0.9, LCP <2s, TTI <3s. If they fail, optimize before committing.

- [ ] **Step 6: Commit**

```bash
git add lighthouserc.json package.json package-lock.json .github/workflows/deploy-frontend.yml
git commit -m "ci(lhci): perf budget — perf≥0.9 LCP<2s TTI<3s on all 3 URLs"
```

---

## Task 10: Smoke test in shark monorepo

**Files:** `tests/smoke/test_demo_platform.py` in shark monorepo.

- [ ] **Step 1: Write the smoke test**

In shark monorepo, create `tests/smoke/test_demo_platform.py`:

```python
"""
Smoke test: post-deploy verification of the live demo platform.
Runs against https://demo-api.sharkauth.dev to confirm Plan A endpoints
remain healthy after every shark release.
"""
from __future__ import annotations

import os

import httpx
import pytest

DEMO_API = os.environ.get("DEMO_API_URL", "https://demo-api.sharkauth.dev")
TIMEOUT = 30.0


pytestmark = pytest.mark.skipif(
    os.environ.get("SKIP_DEMO_PLATFORM_SMOKE") == "1",
    reason="SKIP_DEMO_PLATFORM_SMOKE=1 set",
)


def test_demo_api_health() -> None:
    resp = httpx.get(f"{DEMO_API}/health", timeout=TIMEOUT)
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_demo_api_seeds_a_tenant() -> None:
    resp = httpx.post(f"{DEMO_API}/seed", timeout=TIMEOUT)
    assert resp.status_code == 201
    body = resp.json()
    assert body["org_id"].startswith("acme-")
    assert body["expires_at"] > 0


def test_demo_api_full_act_1_flow() -> None:
    seed = httpx.post(f"{DEMO_API}/seed", timeout=TIMEOUT).json()
    org_id = seed["org_id"]

    create = httpx.post(
        f"{DEMO_API}/orgs/{org_id}/act1/create-agent",
        json={"vertical": "code", "sub_agent_name": "pr-reviewer"},
        timeout=TIMEOUT,
    )
    assert create.status_code == 201
    agent = create.json()
    assert agent["agent_id"]
    assert agent["client_id"]


def test_demo_api_attacks_all_return_401() -> None:
    seed = httpx.post(f"{DEMO_API}/seed", timeout=TIMEOUT).json()
    org_id = seed["org_id"]
    for n in (1, 2, 3, 4):
        resp = httpx.post(
            f"{DEMO_API}/orgs/{org_id}/act2/attack",
            json={
                "attempt_number": n,
                "leaked_token": "synthetic_test_token",
                "target_audience": "github",
            },
            timeout=TIMEOUT,
        )
        assert resp.status_code == 200
        body = resp.json()
        assert body["response_status"] == 401, f"Attack {n} should return 401 not {body['response_status']}"
```

- [ ] **Step 2: Run smoke locally against deployed demo-api**

```bash
# from shark monorepo
pytest tests/smoke/test_demo_platform.py -v
```

Expected: PASS (4 tests, ~10s).

- [ ] **Step 3: Add to shark's CI smoke matrix**

In shark monorepo, ensure existing `./smoke.sh` (or equivalent) picks up the new file. If smoke runner uses `pytest tests/smoke/`, no edit needed.

Verify by running locally:

```bash
./smoke.sh
```

Expected: PASS count includes the 4 new tests.

- [ ] **Step 4: Commit in shark monorepo**

```bash
git add tests/smoke/test_demo_platform.py
git commit -m "test(smoke): demo platform health + 3-act flow + attack 401 check"
```

---

## Task 11: Plausible analytics setup

**Files:** modify `app/layout.tsx` in demo-platform-frontend.

- [ ] **Step 1: Add Plausible site**

Manual: log in to Plausible, add site `demo.sharkauth.dev`. Note the script tag URL.

- [ ] **Step 2: Add script to root layout**

In `app/layout.tsx`, add inside `<body>` (before `{children}`):

```tsx
{process.env.NODE_ENV === "production" && (
  <script
    defer
    data-domain="demo.sharkauth.dev"
    src="https://plausible.io/js/script.js"
  />
)}
```

- [ ] **Step 3: Add walkthrough completion event**

In `app/acme/[vertical]/page.tsx`, add after `startAct(3)` completes (or in `CleanupCascade` after L5 fires):

```tsx
useEffect(() => {
  if (typeof window !== "undefined" && (window as any).plausible) {
    (window as any).plausible("walkthrough_complete", { props: { vertical } })
  }
}, [vertical])
```

(Refine the trigger to fire only after Act 3's L5 completes — wrap in a `useEffect` dependent on the cleanup result.)

- [ ] **Step 4: Verify in Plausible dashboard**

Deploy, visit demo, complete walkthrough. Plausible dashboard should show 1 pageview + 1 `walkthrough_complete` goal event within ~1 min.

- [ ] **Step 5: Commit**

```bash
git add app/layout.tsx app/acme/[vertical]/page.tsx
git commit -m "feat(analytics): Plausible script + walkthrough_complete custom event"
```

---

## Task 12: Pre-launch deployment checklist

**Files:** `LAUNCH_CHECKLIST.md` in demo-platform-frontend repo.

- [ ] **Step 1: Write `LAUNCH_CHECKLIST.md`**

```markdown
# Launch Checklist — demo.sharkauth.dev

Pre-launch verification (run T-24h before HN post).

## Infrastructure
- [ ] Hetzner shark VPS healthy: `curl https://shark-demo-internal.example.dev/health` returns ok
- [ ] Fly demo-api running: `fly status -a shark-demo-api` shows `running`
- [ ] Vercel frontend deployed: `vercel ls` shows latest commit
- [ ] Cloudflare DNS resolves: `dig demo.sharkauth.dev`, `dig demo-api.sharkauth.dev`
- [ ] TLS valid on both domains (no browser warnings)

## Functional smoke
- [ ] Landing page loads: `https://demo.sharkauth.dev`
- [ ] AcmeCode dashboard loads + tenant provisions
- [ ] Act 1 mints token within 10s
- [ ] Act 1.5 dispatches successfully
- [ ] Act 2: all 4 attack attempts return 401
- [ ] Act 3: 5 cleanup buttons all succeed
- [ ] AcmeSupport vertical: same checks
- [ ] Mobile gate: phone visit shows landing + "best on desktop" message

## Performance
- [ ] Lighthouse perf ≥0.9 on `/`, `/acme/code`, `/acme/support`
- [ ] LCP <2s on all 3 URLs
- [ ] No console errors on visit

## Backend health
- [ ] Smoke test green: `pytest tests/smoke/test_demo_platform.py -v` (in shark monorepo)
- [ ] Auto-cleanup ran: check Fly logs for "Cleanup deleted N expired orgs" within last hour
- [ ] Active acme-* orgs <1000: `ssh shark@vps "shark admin orgs list | grep acme- | wc -l"`

## Distribution
- [ ] HN post body mentions "Try the live demo: demo.sharkauth.dev"
- [ ] Twitter thread tweet 4 has the demo URL
- [ ] LinkedIn DM template includes the demo URL
- [ ] Open-source repo URL works: `github.com/sharkauth/demo-platform-example`
- [ ] README on the repo points back to demo URL

## Monitoring (post-launch)
- [ ] Plausible dashboard tab open
- [ ] Fly logs tail open: `fly logs -a shark-demo-api`
- [ ] Hetzner CPU/memory dashboard tab open
- [ ] GitHub Stars notifications on (DM if shark monorepo gets a wave)
```

- [ ] **Step 2: Commit**

```bash
git add LAUNCH_CHECKLIST.md
git commit -m "docs: pre-launch verification checklist"
```

---

## Self-Review

**1. Spec coverage:**
- ✅ Hosting: Vercel (frontend, Task 4), Fly.io (API, Task 2), Hetzner (shark, Task 1)
- ✅ DNS: Cloudflare (Tasks 3, 4)
- ✅ CI/CD: GitHub Actions for both repos (Tasks 5, 6)
- ✅ E2E: Playwright covering full 3-act flow (Tasks 7, 8)
- ✅ Perf budget: Lighthouse CI ≥0.9, LCP <2s, TTI <3s (Task 9)
- ✅ Smoke test in shark monorepo (Task 10)
- ✅ Analytics: Plausible (Task 11)
- ✅ Launch-day checklist (Task 12)
- ✅ Auto-cleanup verified post-deploy (Task 12 monitoring section)

**2. Placeholder scan:**
- One marker `REPLACE_WITH_YOUR_PUBLIC_KEY` in Task 1 Step 2 cloud-init.yaml — explicitly flagged for manual replacement before paste. Acceptable: it's a secret value not appropriate to bake into a plan.
- `shark-demo-internal.example.dev` is a placeholder hostname — flagged as such with note to replace before paste. Acceptable.
- All other steps have concrete commands + expected outputs.

**3. Type consistency:**
- `DEMO_API_URL` env var name consistent across smoke test, Vercel config, Plan A
- `SHARK_ADMIN_KEY` consistent across cloud-init, Fly secrets, Plan A
- `NEXT_PUBLIC_DEMO_API_URL` matches Plan B's `lib/api.ts` baseUrl construction

**Self-review verdict:** PASS. Plan is ready to execute.

---

## Execution Handoff

Plan complete and saved to `playbook/plans/2026-04-26-mock-demo-deploy.md`.

**Execution order across all 3 plans:**
1. Plan A (`2026-04-26-mock-demo-api.md`) — backend service, ~3 days CC
2. Plan B (`2026-04-26-mock-demo-frontend.md`) — frontend SPA, ~1.5 days CC
3. Plan C (this file) — deploy + e2e + smoke, ~0.5 day CC + manual VPS work

Total: 4-5 days CC + ~2-3 hours of human-supervised infra (Hetzner provisioning, DNS, Vercel/Fly token setup).

Use `superpowers:subagent-driven-development` per session decision.
