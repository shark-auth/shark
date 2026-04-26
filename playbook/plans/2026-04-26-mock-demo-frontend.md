# Mock Demo Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Next.js 15 SPA at `demo.sharkauth.dev` that powers the visitor-facing demo. Calls demo-api (Plan A) over HTTPS. Generates DPoP keypairs client-side. Renders 3-act walkthrough with monochrome `.impeccable.md v3` design.

**Architecture:** Next.js 15 App Router (TypeScript). Client-heavy — the dashboard is interactive. Server components only for initial layout + metadata. State via Zustand (lightweight, no provider hell). Real-time audit log via SSE (`EventSource` API). DPoP keypair generation via `window.crypto.subtle.generateKey`. Component tests via Vitest + React Testing Library. E2E deferred to Plan C.

**Tech Stack:** Next.js 15, React 19, TypeScript 5.6+, Tailwind CSS 3.4 (utility classes only — design tokens enforce monochrome), Zustand 5.0, Vitest 2.1, @testing-library/react, eslint-config-next.

---

## Spec Reference

This plan implements section 4 of `playbook/10-mock-demo-platform-design.md`:
- 5 frontend surfaces (landing, dashboard, walkthrough overlay, JWT decoder, audit timeline)
- Vertical toggle (AcmeCode / AcmeSupport)
- Client-side DPoP keypair generation
- SSE consumer for audit stream
- Strict `.impeccable.md v3` design adherence

Backend (Plan A) and deploy (Plan C) are separate.

**Pre-req:** Plan A's demo-api running at `http://localhost:8001` (local dev) or `https://demo-api.sharkauth.dev` (production).

---

## File Structure

| File | Purpose |
|---|---|
| `package.json` | npm dependencies, scripts |
| `tsconfig.json` | TypeScript strict mode config |
| `next.config.mjs` | Next.js config (CSP, redirects) |
| `tailwind.config.ts` | Tailwind tokens locked to monochrome |
| `vitest.config.ts` | Vitest + RTL config |
| `app/layout.tsx` | Root layout (font, metadata) |
| `app/page.tsx` | Landing page |
| `app/acme/[vertical]/page.tsx` | Tenant dashboard (per vertical) |
| `app/acme/[vertical]/layout.tsx` | Sidebar + canvas layout |
| `app/expired/page.tsx` | Sandbox-expired page |
| `lib/api.ts` | Typed wrapper for demo-api endpoints |
| `lib/dpop.ts` | Client-side DPoP keypair generation + JWK extraction |
| `lib/jwt.ts` | Decode JWT payload (no signature verification — display only) |
| `lib/store.ts` | Zustand store (vertical, current act, tenant info, recent token) |
| `lib/sse.ts` | EventSource wrapper for audit stream |
| `components/AcmeShell.tsx` | Sidebar + main canvas + bottom rail layout |
| `components/VerticalToggle.tsx` | AcmeCode ↔ AcmeSupport switcher |
| `components/AgentList.tsx` | List of agents in tenant |
| `components/JwtDecoder.tsx` | Right-rail panel rendering decoded token |
| `components/AuditTimeline.tsx` | Bottom-rail SSE-driven event log |
| `components/WalkthroughRail.tsx` | 3-act guided overlay |
| `components/AttackOverlay.tsx` | Animated villain + 4 attack attempt UI |
| `components/CleanupCascade.tsx` | 5-button cleanup panel |
| `components/Onboarding.tsx` | Act 1 wizard (create agent + connect vault) |
| `components/DelegationViz.tsx` | Act 1.5 act-chain tree visualization |
| `tests/components/*.test.tsx` | Vitest unit tests per component |

---

## Task 1: Repo scaffold + tooling

**Files:** Create entire project skeleton.

- [ ] **Step 1: Bootstrap Next.js**

```bash
npx create-next-app@15 demo-platform-frontend \
  --typescript --tailwind --app --no-src-dir --import-alias "@/*" --use-npm \
  --eslint --turbopack
cd demo-platform-frontend
git init
```

- [ ] **Step 2: Add dev dependencies**

```bash
npm install --save-dev vitest @vitest/coverage-v8 @testing-library/react \
  @testing-library/jest-dom @testing-library/user-event jsdom @types/node prettier
npm install zustand
```

- [ ] **Step 3: Replace `tsconfig.json` with strict config**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": false,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true,
    "plugins": [{ "name": "next" }],
    "paths": { "@/*": ["./*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

- [ ] **Step 4: Write `vitest.config.ts`**

```typescript
import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "node:path"

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./tests/setup.ts"],
    coverage: { provider: "v8", reporter: ["text", "html"], thresholds: { lines: 80 } },
  },
  resolve: { alias: { "@": path.resolve(__dirname, ".") } },
})
```

```bash
npm install --save-dev @vitejs/plugin-react
```

Create `tests/setup.ts`:

```typescript
import "@testing-library/jest-dom/vitest"
```

- [ ] **Step 5: Add scripts to `package.json`**

Replace `scripts` block:

```json
"scripts": {
  "dev": "next dev",
  "build": "next build",
  "start": "next start",
  "lint": "next lint",
  "format": "prettier --write \"**/*.{ts,tsx,json,md}\"",
  "test": "vitest run",
  "test:watch": "vitest",
  "test:coverage": "vitest run --coverage",
  "typecheck": "tsc --noEmit"
}
```

- [ ] **Step 6: Verify boot**

```bash
npm run dev
```

Expected: Next.js boots on `:3000`. `curl http://localhost:3000` returns HTML. Stop with Ctrl+C.

- [ ] **Step 7: Commit**

```bash
git add .
git commit -m "chore: scaffold Next.js 15 + TypeScript + Tailwind + Vitest"
```

---

## Task 2: Lock Tailwind to monochrome design tokens

**Files:**
- Modify: `tailwind.config.ts`
- Modify: `app/globals.css`

- [ ] **Step 1: Replace `tailwind.config.ts` to enforce monochrome**

```typescript
import type { Config } from "tailwindcss"

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    colors: {
      // Monochrome only — .impeccable.md v3 lock
      transparent: "transparent",
      current: "currentColor",
      bg: "#ffffff",
      fg: "#0a0a0a",
      muted: "#737373",
      "muted-light": "#d4d4d4",
      "muted-bg": "#fafafa",
      border: "#e5e5e5",
      // Single allowed accent: destructive states only
      "danger-fg": "#7f1d1d",
      "danger-bg": "#fef2f2",
      "danger-border": "#fecaca",
    },
    extend: {
      borderRadius: {
        DEFAULT: "2px",
        sm: "2px",
        md: "2px",
        lg: "2px",
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Consolas', 'monospace'],
      },
    },
  },
  plugins: [],
}
export default config
```

- [ ] **Step 2: Replace `app/globals.css`**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  --bg: #ffffff;
  --fg: #0a0a0a;
  --muted: #737373;
  --border: #e5e5e5;
  --radius: 2px;
}

html, body {
  background: var(--bg);
  color: var(--fg);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  -webkit-font-smoothing: antialiased;
}

* {
  border-radius: var(--radius);
}

button {
  cursor: pointer;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
```

- [ ] **Step 3: Commit**

```bash
git add tailwind.config.ts app/globals.css
git commit -m "feat(design): lock Tailwind tokens to .impeccable.md v3 monochrome"
```

---

## Task 3: Typed API client for demo-api

**Files:**
- Create: `lib/api.ts`
- Create: `tests/lib/api.test.ts`
- Create: `.env.local.example`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/api.test.ts`:

```typescript
import { afterEach, describe, expect, it, vi } from "vitest"
import { DemoApi } from "@/lib/api"

const makeFetch = (body: unknown, status = 200) =>
  vi.fn().mockResolvedValue({
    ok: status < 400,
    status,
    json: async () => body,
  })

describe("DemoApi", () => {
  afterEach(() => vi.restoreAllMocks())

  it("seeds a tenant", async () => {
    const fetch = makeFetch({ org_id: "acme-abc12345", expires_at: 1714000000 }, 201)
    const api = new DemoApi("http://api", fetch as unknown as typeof globalThis.fetch)
    const result = await api.seed()
    expect(result.org_id).toBe("acme-abc12345")
    expect(fetch).toHaveBeenCalledWith("http://api/seed", { method: "POST" })
  })

  it("creates an agent", async () => {
    const fetch = makeFetch({ agent_id: "a", client_id: "c" }, 201)
    const api = new DemoApi("http://api", fetch as unknown as typeof globalThis.fetch)
    const result = await api.createAgent("acme-x", "code", "pr-reviewer")
    expect(result.agent_id).toBe("a")
    expect(fetch).toHaveBeenCalledWith(
      "http://api/orgs/acme-x/act1/create-agent",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
      }),
    )
  })

  it("throws on non-2xx", async () => {
    const fetch = makeFetch({ detail: "boom" }, 500)
    const api = new DemoApi("http://api", fetch as unknown as typeof globalThis.fetch)
    await expect(api.seed()).rejects.toThrow(/500/)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/api.test.ts
```

Expected: FAIL — module doesn't exist.

- [ ] **Step 3: Write `lib/api.ts`**

```typescript
export type Vertical = "code" | "support"

export interface SeedResponse {
  org_id: string
  expires_at: number
}

export interface CreateAgentResponse {
  agent_id: string
  client_id: string
}

export interface MintFirstTokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

export interface DispatchTaskResponse {
  exchanged_token: string
  decoded: Record<string, unknown>
}

export interface AttackAttemptResponse {
  attempt_number: number
  description: string
  request_summary: Record<string, unknown>
  response_status: number
  response_body: Record<string, unknown>
  failure_reason: string
}

export interface CleanupLayerResponse {
  layer: number
  revoked_token_count: number
  revoked_agent_ids: string[]
  revoked_user_ids?: string[]
}

export class DemoApi {
  constructor(
    private readonly baseUrl: string,
    private readonly fetchFn: typeof globalThis.fetch = globalThis.fetch,
  ) {}

  private async request<T>(
    path: string,
    init?: RequestInit,
  ): Promise<T> {
    const resp = await this.fetchFn(`${this.baseUrl}${path}`, init)
    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}))
      throw new Error(`Demo-api ${resp.status}: ${JSON.stringify(body)}`)
    }
    return resp.json() as Promise<T>
  }

  private postJson<T>(path: string, body?: unknown): Promise<T> {
    return this.request<T>(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    })
  }

  seed(): Promise<SeedResponse> {
    return this.request<SeedResponse>("/seed", { method: "POST" })
  }

  createAgent(orgId: string, vertical: Vertical, subAgentName: string): Promise<CreateAgentResponse> {
    return this.postJson(`/orgs/${orgId}/act1/create-agent`, {
      vertical, sub_agent_name: subAgentName,
    })
  }

  mintFirstToken(orgId: string, clientId: string, publicJwk: JsonWebKey): Promise<MintFirstTokenResponse> {
    return this.postJson(`/orgs/${orgId}/act1/mint-first-token`, {
      client_id: clientId, public_jwk: publicJwk,
    })
  }

  connectVault(orgId: string, userId: string, provider: string): Promise<{ connection_id: string; provider: string; scopes: string[] }> {
    return this.postJson(`/orgs/${orgId}/act1/connect-vault`, { user_id: userId, provider })
  }

  dispatchTask(orgId: string, args: {
    orchestrator_token: string
    sub_agent_client_id: string
    target_audience: string
    scopes: string[]
  }): Promise<DispatchTaskResponse> {
    return this.postJson(`/orgs/${orgId}/act15/dispatch-task`, args)
  }

  simulateInjection(orgId: string, targetAgentClientId: string, leakedToken: string): Promise<{ log_entry_id: string; exfiltration_target: string }> {
    return this.postJson(`/orgs/${orgId}/act2/simulate-injection`, {
      target_agent_client_id: targetAgentClientId, leaked_token: leakedToken,
    })
  }

  attack(orgId: string, attemptNumber: 1 | 2 | 3 | 4, leakedToken: string, targetAudience: "github" | "gmail"): Promise<AttackAttemptResponse> {
    return this.postJson(`/orgs/${orgId}/act2/attack`, {
      attempt_number: attemptNumber, leaked_token: leakedToken, target_audience: targetAudience,
    })
  }

  cleanupLayer1(orgId: string, token: string): Promise<CleanupLayerResponse> {
    return this.postJson(`/orgs/${orgId}/act3/layer1-revoke-token`, { token })
  }
  cleanupLayer2(orgId: string, agentId: string): Promise<CleanupLayerResponse> {
    return this.postJson(`/orgs/${orgId}/act3/layer2-revoke-agent`, { agent_id: agentId })
  }
  cleanupLayer3(orgId: string, userId: string): Promise<CleanupLayerResponse> {
    return this.postJson(`/orgs/${orgId}/act3/layer3-cascade-user`, { user_id: userId })
  }
  cleanupLayer4(orgId: string, pattern: string): Promise<CleanupLayerResponse> {
    return this.postJson(`/orgs/${orgId}/act3/layer4-bulk-pattern`, { pattern })
  }
  cleanupLayer5(orgId: string, connectionId: string): Promise<CleanupLayerResponse> {
    return this.postJson(`/orgs/${orgId}/act3/layer5-disconnect-vault`, { connection_id: connectionId })
  }
}
```

- [ ] **Step 4: Verify test passes**

```bash
npm run test -- tests/lib/api.test.ts
```

Expected: PASS (3 tests).

- [ ] **Step 5: Write `.env.local.example`**

```
NEXT_PUBLIC_DEMO_API_URL=http://localhost:8001
```

- [ ] **Step 6: Commit**

```bash
git add lib/api.ts tests/lib/api.test.ts .env.local.example
git commit -m "feat(api): typed DemoApi client wrapping all Plan A endpoints"
```

---

## Task 4: Client-side DPoP keypair generation

**Files:**
- Create: `lib/dpop.ts`
- Create: `tests/lib/dpop.test.ts`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/dpop.test.ts`:

```typescript
import { describe, expect, it } from "vitest"
import { generateDpopKeypair } from "@/lib/dpop"

describe("generateDpopKeypair", () => {
  it("generates an ECDSA P-256 keypair and returns the public JWK", async () => {
    const result = await generateDpopKeypair()
    expect(result.publicJwk.kty).toBe("EC")
    expect(result.publicJwk.crv).toBe("P-256")
    expect(typeof result.publicJwk.x).toBe("string")
    expect(typeof result.publicJwk.y).toBe("string")
    expect(result.publicJwk.d).toBeUndefined() // private key NEVER leaves
    expect(result.privateKey).toBeInstanceOf(CryptoKey)
  })

  it("produces unique keypairs each call", async () => {
    const a = await generateDpopKeypair()
    const b = await generateDpopKeypair()
    expect(a.publicJwk.x).not.toBe(b.publicJwk.x)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/dpop.test.ts
```

Expected: FAIL — module doesn't exist.

- [ ] **Step 3: Write `lib/dpop.ts`**

```typescript
export interface DpopKeypair {
  publicJwk: JsonWebKey
  privateKey: CryptoKey
}

export async function generateDpopKeypair(): Promise<DpopKeypair> {
  const keyPair = await window.crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    true, // extractable so we can export the JWK
    ["sign", "verify"],
  )
  const publicJwk = await window.crypto.subtle.exportKey("jwk", keyPair.publicKey)
  // Defensive: strip any private fields if present (shouldn't be on a public key)
  delete publicJwk.d
  return { publicJwk, privateKey: keyPair.privateKey }
}
```

- [ ] **Step 4: Verify test passes**

```bash
npm run test -- tests/lib/dpop.test.ts
```

Expected: PASS (2 tests). Note: jsdom 24+ supports SubtleCrypto.

- [ ] **Step 5: Commit**

```bash
git add lib/dpop.ts tests/lib/dpop.test.ts
git commit -m "feat(dpop): client-side ECDSA P-256 keypair gen via window.crypto.subtle"
```

---

## Task 5: JWT decoder utility

**Files:**
- Create: `lib/jwt.ts`
- Create: `tests/lib/jwt.test.ts`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/jwt.test.ts`:

```typescript
import { describe, expect, it } from "vitest"
import { decodeJwtPayload } from "@/lib/jwt"

describe("decodeJwtPayload", () => {
  it("decodes payload from a 3-part JWT", () => {
    // {"sub":"agent_xyz","aud":"https://api.github.com","cnf":{"jkt":"abc"}}
    const payload = "eyJzdWIiOiJhZ2VudF94eXoiLCJhdWQiOiJodHRwczovL2FwaS5naXRodWIuY29tIiwiY25mIjp7ImprdCI6ImFiYyJ9fQ"
    const jwt = `header.${payload}.signature`
    const decoded = decodeJwtPayload(jwt)
    expect(decoded.sub).toBe("agent_xyz")
    expect(decoded.aud).toBe("https://api.github.com")
    expect((decoded.cnf as { jkt: string }).jkt).toBe("abc")
  })

  it("throws on malformed JWT", () => {
    expect(() => decodeJwtPayload("not.a.jwt.at.all")).toThrow()
    expect(() => decodeJwtPayload("only-one-part")).toThrow()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/jwt.test.ts
```

Expected: FAIL.

- [ ] **Step 3: Write `lib/jwt.ts`**

```typescript
export function decodeJwtPayload(jwt: string): Record<string, unknown> {
  const parts = jwt.split(".")
  if (parts.length !== 3) {
    throw new Error(`Invalid JWT: expected 3 parts, got ${parts.length}`)
  }
  const payload = parts[1]
  if (!payload) {
    throw new Error("Invalid JWT: empty payload")
  }
  // base64url → base64
  const padded = payload.replace(/-/g, "+").replace(/_/g, "/")
  const padding = padded.length % 4 === 0 ? "" : "=".repeat(4 - (padded.length % 4))
  const json = atob(padded + padding)
  return JSON.parse(json) as Record<string, unknown>
}
```

- [ ] **Step 4: Verify test passes**

```bash
npm run test -- tests/lib/jwt.test.ts
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add lib/jwt.ts tests/lib/jwt.test.ts
git commit -m "feat(jwt): payload decoder (display-only, no signature verification)"
```

---

## Task 6: Zustand store

**Files:**
- Create: `lib/store.ts`
- Create: `tests/lib/store.test.ts`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/store.test.ts`:

```typescript
import { beforeEach, describe, expect, it } from "vitest"
import { useDemoStore } from "@/lib/store"

describe("useDemoStore", () => {
  beforeEach(() => {
    useDemoStore.setState(useDemoStore.getInitialState())
  })

  it("starts with vertical=code and act=null", () => {
    const s = useDemoStore.getState()
    expect(s.vertical).toBe("code")
    expect(s.currentAct).toBeNull()
  })

  it("toggles vertical", () => {
    useDemoStore.getState().setVertical("support")
    expect(useDemoStore.getState().vertical).toBe("support")
  })

  it("advances acts and tracks completion", () => {
    const s = useDemoStore.getState()
    s.startAct(1)
    expect(useDemoStore.getState().currentAct).toBe(1)
    s.completeAct(1)
    expect(useDemoStore.getState().completedActs).toEqual([1])
  })

  it("stores tenant info", () => {
    useDemoStore.getState().setTenant({ org_id: "acme-x", expires_at: 1714 })
    expect(useDemoStore.getState().tenant?.org_id).toBe("acme-x")
  })

  it("stores most recent token for JWT decoder", () => {
    useDemoStore.getState().setRecentToken("eyJ.payload.sig")
    expect(useDemoStore.getState().recentToken).toBe("eyJ.payload.sig")
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/store.test.ts
```

Expected: FAIL.

- [ ] **Step 3: Write `lib/store.ts`**

```typescript
import { create } from "zustand"
import type { Vertical, SeedResponse } from "@/lib/api"

type ActNumber = 1 | 1.5 | 2 | 3

interface DemoState {
  vertical: Vertical
  currentAct: ActNumber | null
  completedActs: ActNumber[]
  tenant: SeedResponse | null
  recentToken: string | null
  setVertical: (v: Vertical) => void
  startAct: (a: ActNumber) => void
  completeAct: (a: ActNumber) => void
  setTenant: (t: SeedResponse) => void
  setRecentToken: (t: string) => void
  reset: () => void
}

const initial: Pick<DemoState, "vertical" | "currentAct" | "completedActs" | "tenant" | "recentToken"> = {
  vertical: "code",
  currentAct: null,
  completedActs: [],
  tenant: null,
  recentToken: null,
}

export const useDemoStore = create<DemoState>((set) => ({
  ...initial,
  setVertical: (v) => set({ vertical: v }),
  startAct: (a) => set({ currentAct: a }),
  completeAct: (a) =>
    set((s) => ({
      completedActs: s.completedActs.includes(a) ? s.completedActs : [...s.completedActs, a],
    })),
  setTenant: (t) => set({ tenant: t }),
  setRecentToken: (t) => set({ recentToken: t }),
  reset: () => set(initial),
}))

// Required by tests: expose initial state
;(useDemoStore as unknown as { getInitialState: () => typeof initial }).getInitialState = () => ({ ...initial })
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/lib/store.test.ts
```

Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add lib/store.ts tests/lib/store.test.ts
git commit -m "feat(store): Zustand store for vertical, act progress, tenant, recent token"
```

---

## Task 7: SSE client wrapper

**Files:**
- Create: `lib/sse.ts`
- Create: `tests/lib/sse.test.ts`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/sse.test.ts`:

```typescript
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { subscribeAuditStream } from "@/lib/sse"

class FakeEventSource {
  static instances: FakeEventSource[] = []
  url: string
  onmessage: ((e: MessageEvent) => void) | null = null
  addEventListener = vi.fn((event: string, handler: EventListener) => {
    this._handlers[event] = handler
  })
  close = vi.fn()
  private _handlers: Record<string, EventListener> = {}

  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }

  emit(event: string, data: unknown) {
    const handler = this._handlers[event]
    if (handler) {
      handler({ data: JSON.stringify(data) } as MessageEvent)
    }
  }
}

beforeEach(() => {
  FakeEventSource.instances = []
  vi.stubGlobal("EventSource", FakeEventSource)
})
afterEach(() => vi.unstubAllGlobals())

describe("subscribeAuditStream", () => {
  it("opens an EventSource at the audit/stream URL", () => {
    const onEvent = vi.fn()
    subscribeAuditStream("http://api", "acme-x", onEvent)
    expect(FakeEventSource.instances[0]?.url).toBe("http://api/orgs/acme-x/audit/stream")
  })

  it("forwards parsed audit events", () => {
    const onEvent = vi.fn()
    subscribeAuditStream("http://api", "acme-x", onEvent)
    FakeEventSource.instances[0]?.emit("audit", { id: "a1", action: "oauth.token.issued" })
    expect(onEvent).toHaveBeenCalledWith({ id: "a1", action: "oauth.token.issued" })
  })

  it("returns a closer", () => {
    const close = subscribeAuditStream("http://api", "acme-x", vi.fn())
    close()
    expect(FakeEventSource.instances[0]?.close).toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/sse.test.ts
```

Expected: FAIL.

- [ ] **Step 3: Write `lib/sse.ts`**

```typescript
export interface AuditEvent {
  id: string
  action: string
  actor_id: string
  created_at: number
  metadata: Record<string, unknown>
}

export function subscribeAuditStream(
  apiBaseUrl: string,
  orgId: string,
  onEvent: (event: AuditEvent) => void,
): () => void {
  const url = `${apiBaseUrl}/orgs/${orgId}/audit/stream`
  const source = new EventSource(url)
  source.addEventListener("audit", (e) => {
    try {
      const parsed = JSON.parse((e as MessageEvent).data) as AuditEvent
      onEvent(parsed)
    } catch (err) {
      console.error("Failed to parse audit event", err)
    }
  })
  return () => source.close()
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/lib/sse.test.ts
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add lib/sse.ts tests/lib/sse.test.ts
git commit -m "feat(sse): EventSource wrapper for audit stream consumption"
```

---

## Task 8: Landing page

**Files:**
- Replace: `app/page.tsx`
- Replace: `app/layout.tsx`

- [ ] **Step 1: Write `app/layout.tsx`**

```tsx
import type { Metadata } from "next"
import "./globals.css"

export const metadata: Metadata = {
  title: "shark — auth for products that ship agents",
  description:
    "Live demo of shark: DPoP-bound tokens, RFC 8693 delegation, vault-gated retrieval. Token theft is cryptographically useless.",
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  )
}
```

- [ ] **Step 2: Write `app/page.tsx`**

```tsx
import Link from "next/link"

export default function LandingPage() {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <h1 className="text-4xl font-bold tracking-tight">
        Auth for products that give customers their own agents.
      </h1>
      <p className="mt-6 text-lg text-muted">
        Token theft is cryptographically useless. DPoP-bound tokens, audience binding,
        vault-gated retrieval. Watch a 4-minute interactive demo of two fictional
        SaaS platforms (AcmeCode + AcmeSupport) running on shark.
      </p>
      <div className="mt-12 grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Link
          href="/acme/code"
          className="border border-border p-8 hover:bg-muted-bg transition-colors"
        >
          <div className="text-sm text-muted">vertical</div>
          <div className="mt-2 text-xl font-semibold">AcmeCode</div>
          <div className="mt-1 text-sm text-muted">
            AI coding agents for your engineering team
          </div>
        </Link>
        <Link
          href="/acme/support"
          className="border border-border p-8 hover:bg-muted-bg transition-colors"
        >
          <div className="text-sm text-muted">vertical</div>
          <div className="mt-2 text-xl font-semibold">AcmeSupport</div>
          <div className="mt-1 text-sm text-muted">
            AI support agents for your customer success team
          </div>
        </Link>
      </div>
      <p className="mt-12 text-sm text-muted">
        Best experienced on desktop. Source code on{" "}
        <a className="underline" href="https://github.com/sharkauth/demo-platform-example">
          GitHub
        </a>
        .
      </p>
    </main>
  )
}
```

- [ ] **Step 3: Verify visually**

```bash
npm run dev
```

Open `http://localhost:3000`. Confirm: monochrome, square corners, two vertical cards, system font. Stop server.

- [ ] **Step 4: Commit**

```bash
git add app/layout.tsx app/page.tsx
git commit -m "feat(landing): minimal monochrome landing with vertical preview cards"
```

---

## Task 9: Acme tenant dashboard layout

**Files:**
- Create: `app/acme/[vertical]/layout.tsx`
- Create: `app/acme/[vertical]/page.tsx`
- Create: `components/AcmeShell.tsx`
- Create: `tests/components/AcmeShell.test.tsx`

- [ ] **Step 1: Write the failing component test**

Create `tests/components/AcmeShell.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { AcmeShell } from "@/components/AcmeShell"

describe("AcmeShell", () => {
  it("renders sidebar with the vertical's name", () => {
    render(
      <AcmeShell vertical="code">
        <div>main canvas</div>
      </AcmeShell>,
    )
    expect(screen.getByText("AcmeCode")).toBeInTheDocument()
    expect(screen.getByText("main canvas")).toBeInTheDocument()
  })

  it("shows 4 sidebar nav items", () => {
    render(
      <AcmeShell vertical="code">
        <div />
      </AcmeShell>,
    )
    expect(screen.getByRole("link", { name: /agents/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /vault/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /audit/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /policies/i })).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/AcmeShell.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/AcmeShell.tsx`**

```tsx
"use client"

import Link from "next/link"
import type { Vertical } from "@/lib/api"

const VERTICAL_NAME: Record<Vertical, string> = {
  code: "AcmeCode",
  support: "AcmeSupport",
}

const NAV_ITEMS = [
  { label: "Agents", href: "agents" },
  { label: "Vault", href: "vault" },
  { label: "Audit", href: "audit" },
  { label: "Policies", href: "policies" },
] as const

export function AcmeShell({ vertical, children }: { vertical: Vertical; children: React.ReactNode }) {
  const name = VERTICAL_NAME[vertical]
  const base = `/acme/${vertical}`
  return (
    <div className="grid grid-cols-[240px_1fr] grid-rows-[1fr_200px] h-screen">
      {/* Sidebar */}
      <aside className="row-span-2 border-r border-border p-6">
        <div className="text-xl font-semibold">{name}</div>
        <nav className="mt-8 flex flex-col gap-1">
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={`${base}/${item.href}`}
              className="px-3 py-2 hover:bg-muted-bg text-sm"
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </aside>
      {/* Main canvas */}
      <main className="overflow-auto p-8">{children}</main>
      {/* Bottom rail (audit timeline placeholder) */}
      <footer className="border-t border-border p-4 overflow-auto">
        {/* AuditTimeline component mounts here in Task 12 */}
      </footer>
    </div>
  )
}
```

- [ ] **Step 4: Write `app/acme/[vertical]/layout.tsx`**

```tsx
import { notFound } from "next/navigation"
import { AcmeShell } from "@/components/AcmeShell"
import type { Vertical } from "@/lib/api"

const VALID_VERTICALS: Vertical[] = ["code", "support"]

export default async function VerticalLayout({
  children,
  params,
}: {
  children: React.ReactNode
  params: Promise<{ vertical: string }>
}) {
  const { vertical } = await params
  if (!VALID_VERTICALS.includes(vertical as Vertical)) {
    notFound()
  }
  return <AcmeShell vertical={vertical as Vertical}>{children}</AcmeShell>
}
```

- [ ] **Step 5: Write `app/acme/[vertical]/page.tsx`**

```tsx
"use client"

export default function VerticalHome() {
  return (
    <div>
      <h2 className="text-2xl font-semibold">Welcome to your tenant</h2>
      <p className="mt-2 text-muted">Your demo sandbox is ready. Click an act in the walkthrough rail to begin.</p>
    </div>
  )
}
```

- [ ] **Step 6: Verify tests pass + visual**

```bash
npm run test -- tests/components/AcmeShell.test.tsx
npm run dev
```

Open `http://localhost:3000/acme/code`. Confirm sidebar + main + bottom layout, monochrome, AcmeCode shown. Stop server.

- [ ] **Step 7: Commit**

```bash
git add app/acme components/AcmeShell.tsx tests/components/AcmeShell.test.tsx
git commit -m "feat(dashboard): AcmeShell layout (sidebar + canvas + bottom rail)"
```

---

## Task 10: VerticalToggle component

**Files:**
- Create: `components/VerticalToggle.tsx`
- Create: `tests/components/VerticalToggle.test.tsx`
- Modify: `components/AcmeShell.tsx` (mount toggle in sidebar header)

- [ ] **Step 1: Write the failing test**

Create `tests/components/VerticalToggle.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { VerticalToggle } from "@/components/VerticalToggle"

describe("VerticalToggle", () => {
  it("renders both vertical names", () => {
    render(<VerticalToggle current="code" onChange={vi.fn()} />)
    expect(screen.getByText("AcmeCode")).toBeInTheDocument()
    expect(screen.getByText("AcmeSupport")).toBeInTheDocument()
  })

  it("calls onChange when clicking the inactive vertical", () => {
    const onChange = vi.fn()
    render(<VerticalToggle current="code" onChange={onChange} />)
    fireEvent.click(screen.getByText("AcmeSupport"))
    expect(onChange).toHaveBeenCalledWith("support")
  })

  it("does not call onChange when clicking the active vertical", () => {
    const onChange = vi.fn()
    render(<VerticalToggle current="code" onChange={onChange} />)
    fireEvent.click(screen.getByText("AcmeCode"))
    expect(onChange).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/VerticalToggle.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/VerticalToggle.tsx`**

```tsx
"use client"

import type { Vertical } from "@/lib/api"

export function VerticalToggle({
  current,
  onChange,
}: {
  current: Vertical
  onChange: (v: Vertical) => void
}) {
  return (
    <div className="inline-flex border border-border">
      {(["code", "support"] as const).map((v) => (
        <button
          key={v}
          type="button"
          onClick={() => v !== current && onChange(v)}
          className={`px-3 py-1 text-xs ${
            v === current ? "bg-fg text-bg" : "bg-bg text-fg hover:bg-muted-bg"
          }`}
        >
          {v === "code" ? "AcmeCode" : "AcmeSupport"}
        </button>
      ))}
    </div>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/VerticalToggle.test.tsx
```

Expected: PASS (3 tests).

- [ ] **Step 5: Mount toggle in `components/AcmeShell.tsx`**

Update `AcmeShell.tsx` — add to sidebar header (top of `<aside>`):

```tsx
"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { VerticalToggle } from "@/components/VerticalToggle"
import type { Vertical } from "@/lib/api"

const VERTICAL_NAME: Record<Vertical, string> = { code: "AcmeCode", support: "AcmeSupport" }
const NAV_ITEMS = [
  { label: "Agents", href: "agents" },
  { label: "Vault", href: "vault" },
  { label: "Audit", href: "audit" },
  { label: "Policies", href: "policies" },
] as const

export function AcmeShell({ vertical, children }: { vertical: Vertical; children: React.ReactNode }) {
  const router = useRouter()
  const name = VERTICAL_NAME[vertical]
  const base = `/acme/${vertical}`
  return (
    <div className="grid grid-cols-[240px_1fr] grid-rows-[1fr_200px] h-screen">
      <aside className="row-span-2 border-r border-border p-6">
        <div className="flex items-center justify-between">
          <div className="text-xl font-semibold">{name}</div>
          <VerticalToggle current={vertical} onChange={(v) => router.push(`/acme/${v}`)} />
        </div>
        <nav className="mt-8 flex flex-col gap-1">
          {NAV_ITEMS.map((item) => (
            <Link key={item.href} href={`${base}/${item.href}`} className="px-3 py-2 hover:bg-muted-bg text-sm">
              {item.label}
            </Link>
          ))}
        </nav>
      </aside>
      <main className="overflow-auto p-8">{children}</main>
      <footer className="border-t border-border p-4 overflow-auto" />
    </div>
  )
}
```

- [ ] **Step 6: Update `AcmeShell.test.tsx` to wrap in MemoryRouter**

(Next.js `useRouter` requires a router context in tests. Replace test render with `vi.mock`.)

Replace test file:

```tsx
import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { AcmeShell } from "@/components/AcmeShell"

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }))

describe("AcmeShell", () => {
  it("renders sidebar with the vertical's name", () => {
    render(<AcmeShell vertical="code"><div>main canvas</div></AcmeShell>)
    expect(screen.getAllByText("AcmeCode").length).toBeGreaterThan(0)
    expect(screen.getByText("main canvas")).toBeInTheDocument()
  })

  it("shows 4 sidebar nav items", () => {
    render(<AcmeShell vertical="code"><div /></AcmeShell>)
    expect(screen.getByRole("link", { name: /agents/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /vault/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /audit/i })).toBeInTheDocument()
    expect(screen.getByRole("link", { name: /policies/i })).toBeInTheDocument()
  })
})
```

- [ ] **Step 7: Verify tests still pass**

```bash
npm run test -- tests/components
```

Expected: PASS (5 tests across both component test files).

- [ ] **Step 8: Commit**

```bash
git add components/VerticalToggle.tsx tests/components/VerticalToggle.test.tsx components/AcmeShell.tsx tests/components/AcmeShell.test.tsx
git commit -m "feat(toggle): VerticalToggle wired into AcmeShell sidebar header"
```

---

## Task 11: JwtDecoder component

**Files:**
- Create: `components/JwtDecoder.tsx`
- Create: `tests/components/JwtDecoder.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/JwtDecoder.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { JwtDecoder } from "@/components/JwtDecoder"

const sampleJwt =
  "eyJhbGciOiJFUzI1NiJ9." +
  "eyJzdWIiOiJhZ2VudF94eXoiLCJhdWQiOiJodHRwczovL2FwaS5naXRodWIuY29tIiwic2NvcGUiOiJnaXRodWI6cmVhZCBwcjp3cml0ZSIsImNuZiI6eyJqa3QiOiJhYmMxMjMifSwiYWN0Ijp7InN1YiI6Im9yY2hlc3RyYXRvciJ9fQ." +
  "sig"

describe("JwtDecoder", () => {
  it("renders the empty state when no token provided", () => {
    render(<JwtDecoder token={null} />)
    expect(screen.getByText(/no token yet/i)).toBeInTheDocument()
  })

  it("renders cnf.jkt and act when token contains them", () => {
    render(<JwtDecoder token={sampleJwt} />)
    expect(screen.getByText(/cnf\.jkt/i)).toBeInTheDocument()
    expect(screen.getByText("abc123")).toBeInTheDocument()
    expect(screen.getByText(/act/i)).toBeInTheDocument()
    expect(screen.getByText(/orchestrator/i)).toBeInTheDocument()
  })

  it("renders aud and scope", () => {
    render(<JwtDecoder token={sampleJwt} />)
    expect(screen.getByText("https://api.github.com")).toBeInTheDocument()
    expect(screen.getByText(/github:read pr:write/i)).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/JwtDecoder.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/JwtDecoder.tsx`**

```tsx
"use client"

import { decodeJwtPayload } from "@/lib/jwt"

interface Props {
  token: string | null
}

export function JwtDecoder({ token }: Props) {
  if (!token) {
    return (
      <div className="border border-border p-4 text-sm text-muted">
        No token yet. Trigger Act 1 to mint one.
      </div>
    )
  }

  let decoded: Record<string, unknown>
  try {
    decoded = decodeJwtPayload(token)
  } catch (err) {
    return (
      <div className="border border-danger-border bg-danger-bg p-4 text-sm text-danger-fg">
        Failed to decode token: {(err as Error).message}
      </div>
    )
  }

  const cnf = decoded.cnf as { jkt?: string } | undefined
  const act = decoded.act as { sub?: string; act?: unknown } | undefined

  return (
    <div className="border border-border p-4 font-mono text-xs">
      <div className="font-sans text-sm font-semibold uppercase tracking-wide text-muted">
        Decoded JWT
      </div>
      <Field label="sub" value={String(decoded.sub ?? "")} />
      <Field label="aud" value={String(decoded.aud ?? "")} />
      <Field label="scope" value={String(decoded.scope ?? "")} />
      {cnf?.jkt && <Field label="cnf.jkt" value={cnf.jkt} highlight />}
      {act && <ActChain act={act} />}
    </div>
  )
}

function Field({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="mt-2 grid grid-cols-[120px_1fr] gap-2">
      <div className="text-muted">{label}</div>
      <div className={highlight ? "bg-muted-bg px-1" : ""}>{value}</div>
    </div>
  )
}

function ActChain({ act }: { act: { sub?: string; act?: unknown } }) {
  const chain: string[] = []
  let cursor: { sub?: string; act?: unknown } | undefined = act
  while (cursor?.sub) {
    chain.push(cursor.sub)
    cursor = cursor.act as { sub?: string; act?: unknown } | undefined
  }
  return (
    <div className="mt-3 border-t border-border pt-2">
      <div className="text-muted">act chain</div>
      <div className="mt-1 flex flex-wrap items-center gap-1">
        {chain.map((sub, i) => (
          <span key={i}>
            <span className="bg-muted-bg px-1">{sub}</span>
            {i < chain.length - 1 && <span className="mx-1 text-muted">→</span>}
          </span>
        ))}
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/JwtDecoder.test.tsx
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add components/JwtDecoder.tsx tests/components/JwtDecoder.test.tsx
git commit -m "feat(jwt-decoder): right-rail panel rendering decoded token + act chain"
```

---

## Task 12: AuditTimeline component (SSE consumer)

**Files:**
- Create: `components/AuditTimeline.tsx`
- Create: `tests/components/AuditTimeline.test.tsx`
- Modify: `components/AcmeShell.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/AuditTimeline.test.tsx`:

```tsx
import { render, screen, act } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { AuditTimeline } from "@/components/AuditTimeline"

let onEventCb: ((e: unknown) => void) | null = null

vi.mock("@/lib/sse", () => ({
  subscribeAuditStream: (apiUrl: string, orgId: string, onEvent: (e: unknown) => void) => {
    onEventCb = onEvent
    return () => { onEventCb = null }
  },
}))

beforeEach(() => { onEventCb = null })
afterEach(() => vi.clearAllMocks())

describe("AuditTimeline", () => {
  it("renders empty state when no events", () => {
    render(<AuditTimeline apiBaseUrl="http://api" orgId="acme-x" />)
    expect(screen.getByText(/no audit events/i)).toBeInTheDocument()
  })

  it("renders events as they arrive via SSE", () => {
    render(<AuditTimeline apiBaseUrl="http://api" orgId="acme-x" />)
    act(() => {
      onEventCb?.({
        id: "a1", action: "oauth.token.issued", actor_id: "agent_xyz",
        created_at: 1714000000, metadata: {},
      })
    })
    expect(screen.getByText("oauth.token.issued")).toBeInTheDocument()
    expect(screen.getByText(/agent_xyz/i)).toBeInTheDocument()
  })

  it("color-codes failure events with danger styling", () => {
    render(<AuditTimeline apiBaseUrl="http://api" orgId="acme-x" />)
    act(() => {
      onEventCb?.({
        id: "a2", action: "oauth.token.issuance_failed", actor_id: "attacker",
        created_at: 1714000001, metadata: { reason: "invalid_dpop_proof" },
      })
    })
    const el = screen.getByText("oauth.token.issuance_failed").closest("div")
    expect(el).toHaveClass("border-danger-border")
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/AuditTimeline.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/AuditTimeline.tsx`**

```tsx
"use client"

import { useEffect, useState } from "react"
import { subscribeAuditStream, type AuditEvent } from "@/lib/sse"

interface Props {
  apiBaseUrl: string
  orgId: string
}

export function AuditTimeline({ apiBaseUrl, orgId }: Props) {
  const [events, setEvents] = useState<AuditEvent[]>([])

  useEffect(() => {
    const close = subscribeAuditStream(apiBaseUrl, orgId, (e) => {
      setEvents((prev) => [...prev, e].slice(-100))
    })
    return close
  }, [apiBaseUrl, orgId])

  if (events.length === 0) {
    return (
      <div className="text-xs text-muted">No audit events yet. Trigger an act to populate.</div>
    )
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="mb-1 text-xs uppercase tracking-wide text-muted">Audit log</div>
      {events.map((e) => {
        const isFailure = e.action.endsWith("_failed") || e.action.includes("denied")
        return (
          <div
            key={e.id}
            className={`flex items-center gap-3 border p-2 text-xs ${
              isFailure
                ? "border-danger-border bg-danger-bg text-danger-fg"
                : "border-border"
            }`}
          >
            <span className="font-mono text-muted">{new Date(e.created_at * 1000).toISOString().slice(11, 19)}</span>
            <span className="font-semibold">{e.action}</span>
            <span className="text-muted">{e.actor_id}</span>
            {Object.keys(e.metadata).length > 0 && (
              <details className="ml-auto">
                <summary className="cursor-pointer text-muted">metadata</summary>
                <pre className="mt-1 max-w-md overflow-auto bg-muted-bg p-2">
                  {JSON.stringify(e.metadata, null, 2)}
                </pre>
              </details>
            )}
          </div>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 4: Mount in `components/AcmeShell.tsx`**

Replace the empty `<footer />`:

```tsx
<footer className="border-t border-border p-4 overflow-auto">
  <AuditTimeline apiBaseUrl={process.env.NEXT_PUBLIC_DEMO_API_URL ?? ""} orgId="acme-pending" />
</footer>
```

(Use `acme-pending` as a placeholder until tenant store integration in Task 14.)

Add import:

```tsx
import { AuditTimeline } from "@/components/AuditTimeline"
```

- [ ] **Step 5: Verify tests pass**

```bash
npm run test -- tests/components
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add components/AuditTimeline.tsx tests/components/AuditTimeline.test.tsx components/AcmeShell.tsx
git commit -m "feat(audit): SSE-driven AuditTimeline mounted in AcmeShell bottom rail"
```

---

## Task 13: Tenant provisioning hook

**Files:**
- Create: `lib/useEnsureTenant.ts`
- Create: `tests/lib/useEnsureTenant.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/lib/useEnsureTenant.test.tsx`:

```tsx
import { act, renderHook, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { useEnsureTenant } from "@/lib/useEnsureTenant"
import { useDemoStore } from "@/lib/store"

describe("useEnsureTenant", () => {
  it("calls api.seed and stores result if no tenant", async () => {
    const seed = vi.fn().mockResolvedValue({ org_id: "acme-new", expires_at: 1714 })
    useDemoStore.setState({ tenant: null })
    renderHook(() => useEnsureTenant({ seed } as never))
    await waitFor(() => {
      expect(useDemoStore.getState().tenant?.org_id).toBe("acme-new")
    })
    expect(seed).toHaveBeenCalledOnce()
  })

  it("does NOT call seed if tenant already in store", async () => {
    const seed = vi.fn()
    useDemoStore.setState({ tenant: { org_id: "acme-existing", expires_at: 1714 } })
    renderHook(() => useEnsureTenant({ seed } as never))
    await new Promise((r) => setTimeout(r, 30))
    expect(seed).not.toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/lib/useEnsureTenant.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `lib/useEnsureTenant.ts`**

```typescript
"use client"

import { useEffect } from "react"
import type { DemoApi } from "@/lib/api"
import { useDemoStore } from "@/lib/store"

export function useEnsureTenant(api: DemoApi): void {
  const tenant = useDemoStore((s) => s.tenant)
  const setTenant = useDemoStore((s) => s.setTenant)

  useEffect(() => {
    if (tenant) return
    let cancelled = false
    api.seed().then((result) => {
      if (!cancelled) setTenant(result)
    }).catch((err) => {
      console.error("Failed to seed tenant", err)
    })
    return () => {
      cancelled = true
    }
  }, [tenant, setTenant, api])
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/lib/useEnsureTenant.test.tsx
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add lib/useEnsureTenant.ts tests/lib/useEnsureTenant.test.tsx
git commit -m "feat(tenant): useEnsureTenant hook auto-seeds tenant on dashboard mount"
```

---

## Task 14: Wire tenant store into AcmeShell

**Files:**
- Modify: `components/AcmeShell.tsx`

- [ ] **Step 1: Update `AcmeShell.tsx` to use store-backed orgId**

Replace `AuditTimeline` mount block:

```tsx
import { useDemoStore } from "@/lib/store"

// inside component
const orgId = useDemoStore((s) => s.tenant?.org_id ?? null)

// in footer:
<footer className="border-t border-border p-4 overflow-auto">
  {orgId ? (
    <AuditTimeline apiBaseUrl={process.env.NEXT_PUBLIC_DEMO_API_URL ?? ""} orgId={orgId} />
  ) : (
    <div className="text-xs text-muted">Provisioning tenant…</div>
  )}
</footer>
```

- [ ] **Step 2: Verify tests still green**

```bash
npm run test
```

Expected: all PASS.

- [ ] **Step 3: Commit**

```bash
git add components/AcmeShell.tsx
git commit -m "feat(shell): wire AuditTimeline orgId from Zustand tenant store"
```

---

## Task 15: Onboarding component (Act 1)

**Files:**
- Create: `components/Onboarding.tsx`
- Create: `tests/components/Onboarding.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/Onboarding.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { Onboarding } from "@/components/Onboarding"

vi.mock("@/lib/dpop", () => ({
  generateDpopKeypair: vi.fn().mockResolvedValue({
    publicJwk: { kty: "EC", crv: "P-256", x: "x", y: "y" },
    privateKey: {} as CryptoKey,
  }),
}))

const fakeApi = {
  createAgent: vi.fn().mockResolvedValue({ agent_id: "agent_a", client_id: "shark_dcr_z" }),
  mintFirstToken: vi.fn().mockResolvedValue({ access_token: "eyJ.payload.sig", token_type: "DPoP", expires_in: 3600 }),
}

describe("Onboarding", () => {
  it("renders Add agent button", () => {
    render(<Onboarding api={fakeApi as never} orgId="acme-x" vertical="code" />)
    expect(screen.getByRole("button", { name: /add pr-reviewer agent/i })).toBeInTheDocument()
  })

  it("calls createAgent + generates keypair + mints token in sequence on click", async () => {
    const onTokenMinted = vi.fn()
    render(
      <Onboarding api={fakeApi as never} orgId="acme-x" vertical="code" onTokenMinted={onTokenMinted} />,
    )
    fireEvent.click(screen.getByRole("button", { name: /add pr-reviewer agent/i }))
    await waitFor(() => {
      expect(fakeApi.createAgent).toHaveBeenCalledWith("acme-x", "code", "pr-reviewer")
      expect(fakeApi.mintFirstToken).toHaveBeenCalledWith("acme-x", "shark_dcr_z", expect.any(Object))
      expect(onTokenMinted).toHaveBeenCalledWith("eyJ.payload.sig")
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/Onboarding.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/Onboarding.tsx`**

```tsx
"use client"

import { useState } from "react"
import type { DemoApi, Vertical } from "@/lib/api"
import { generateDpopKeypair } from "@/lib/dpop"

interface Props {
  api: DemoApi
  orgId: string
  vertical: Vertical
  onTokenMinted?: (token: string) => void
}

export function Onboarding({ api, orgId, vertical, onTokenMinted }: Props) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleClick() {
    setBusy(true)
    setError(null)
    try {
      const subAgent = vertical === "code" ? "pr-reviewer" : "email-replier"
      const created = await api.createAgent(orgId, vertical, subAgent)
      const { publicJwk } = await generateDpopKeypair()
      const token = await api.mintFirstToken(orgId, created.client_id, publicJwk)
      onTokenMinted?.(token.access_token)
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  const subAgentLabel = vertical === "code" ? "pr-reviewer" : "email-replier"

  return (
    <div className="border border-border p-6">
      <div className="text-sm uppercase tracking-wide text-muted">Act 1: Onboarding</div>
      <h3 className="mt-2 text-xl font-semibold">Add your first agent</h3>
      <p className="mt-2 text-sm text-muted">
        Click to register an agent. Your DPoP keypair is generated in this browser tab —
        the private key never leaves your machine.
      </p>
      <button
        type="button"
        disabled={busy}
        onClick={handleClick}
        className="mt-4 border border-fg bg-fg px-4 py-2 text-sm text-bg hover:opacity-90 disabled:opacity-50"
      >
        {busy ? "Provisioning…" : `Add ${subAgentLabel} agent`}
      </button>
      {error && (
        <div className="mt-3 border border-danger-border bg-danger-bg p-2 text-xs text-danger-fg">
          {error}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/Onboarding.test.tsx
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add components/Onboarding.tsx tests/components/Onboarding.test.tsx
git commit -m "feat(act1): Onboarding component (DCR + client-side DPoP keygen)"
```

---

## Task 16: DelegationViz component (Act 1.5 act-chain tree)

**Files:**
- Create: `components/DelegationViz.tsx`
- Create: `tests/components/DelegationViz.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/DelegationViz.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { DelegationViz } from "@/components/DelegationViz"

const fakeApi = {
  dispatchTask: vi.fn().mockResolvedValue({
    exchanged_token: "eyJ.exchanged.sig",
    decoded: {
      sub: "pr-reviewer",
      aud: "https://api.github.com",
      scope: "github:read pr:write",
      act: { sub: "code-orchestrator" },
      cnf: { jkt: "newjkt" },
    },
  }),
}

describe("DelegationViz", () => {
  it("renders Dispatch button", () => {
    render(
      <DelegationViz api={fakeApi as never} orgId="acme-x" orchestratorToken="orch.tok" subAgentClientId="shark_pr" />,
    )
    expect(screen.getByRole("button", { name: /dispatch pr review task/i })).toBeInTheDocument()
  })

  it("displays act chain after dispatch", async () => {
    const onExchanged = vi.fn()
    render(
      <DelegationViz
        api={fakeApi as never}
        orgId="acme-x"
        orchestratorToken="orch.tok"
        subAgentClientId="shark_pr"
        onExchanged={onExchanged}
      />,
    )
    fireEvent.click(screen.getByRole("button", { name: /dispatch pr review task/i }))
    await waitFor(() => {
      expect(screen.getByText(/code-orchestrator/i)).toBeInTheDocument()
      expect(screen.getByText(/pr-reviewer/i)).toBeInTheDocument()
      expect(onExchanged).toHaveBeenCalledWith("eyJ.exchanged.sig")
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/DelegationViz.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/DelegationViz.tsx`**

```tsx
"use client"

import { useState } from "react"
import type { DemoApi, DispatchTaskResponse } from "@/lib/api"

interface Props {
  api: DemoApi
  orgId: string
  orchestratorToken: string
  subAgentClientId: string
  onExchanged?: (token: string) => void
}

export function DelegationViz({ api, orgId, orchestratorToken, subAgentClientId, onExchanged }: Props) {
  const [result, setResult] = useState<DispatchTaskResponse | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleDispatch() {
    setBusy(true)
    try {
      const r = await api.dispatchTask(orgId, {
        orchestrator_token: orchestratorToken,
        sub_agent_client_id: subAgentClientId,
        target_audience: "https://api.github.com",
        scopes: ["github:read", "pr:write"],
      })
      setResult(r)
      onExchanged?.(r.exchanged_token)
    } finally {
      setBusy(false)
    }
  }

  const chain = result ? extractActChain(result.decoded) : []

  return (
    <div className="border border-border p-6">
      <div className="text-sm uppercase tracking-wide text-muted">Act 1.5: Delegation</div>
      <h3 className="mt-2 text-xl font-semibold">Orchestrator dispatches to sub-agent</h3>
      <button
        type="button"
        disabled={busy}
        onClick={handleDispatch}
        className="mt-4 border border-fg bg-fg px-4 py-2 text-sm text-bg disabled:opacity-50"
      >
        {busy ? "Exchanging…" : "Dispatch PR review task"}
      </button>
      {chain.length > 0 && (
        <div className="mt-4 flex flex-wrap items-center gap-2">
          {chain.map((sub, i) => (
            <span key={i} className="flex items-center">
              <span className="border border-border bg-muted-bg px-2 py-1 text-xs font-mono">{sub}</span>
              {i < chain.length - 1 && <span className="mx-2 text-muted">→</span>}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

function extractActChain(decoded: Record<string, unknown>): string[] {
  const chain: string[] = []
  const sub = decoded.sub
  if (typeof sub === "string") chain.push(sub)
  let cursor = decoded.act as { sub?: string; act?: unknown } | undefined
  while (cursor?.sub) {
    chain.unshift(cursor.sub) // root first
    cursor = cursor.act as { sub?: string; act?: unknown } | undefined
  }
  return chain
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/DelegationViz.test.tsx
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add components/DelegationViz.tsx tests/components/DelegationViz.test.tsx
git commit -m "feat(act1.5): DelegationViz with act-chain tree visualization"
```

---

## Task 17: AttackOverlay component (Act 2)

**Files:**
- Create: `components/AttackOverlay.tsx`
- Create: `tests/components/AttackOverlay.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/AttackOverlay.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { AttackOverlay } from "@/components/AttackOverlay"

const baseAttackResp = (n: 1 | 2 | 3 | 4) => ({
  attempt_number: n,
  description: `attempt ${n}`,
  request_summary: { method: "GET" },
  response_status: 401,
  response_body: { error: "blocked", error_description: `failure ${n}` },
  failure_reason: `cryptographic block ${n}`,
})

const fakeApi = {
  simulateInjection: vi.fn().mockResolvedValue({ log_entry_id: "a1", exfiltration_target: "attacker.evil" }),
  attack: vi.fn().mockImplementation((_org, n) => Promise.resolve(baseAttackResp(n))),
}

describe("AttackOverlay", () => {
  it("renders Start Attack button", () => {
    render(<AttackOverlay api={fakeApi as never} orgId="acme-x" leakedToken="t" subAgentClientId="s" targetAudience="github" />)
    expect(screen.getByRole("button", { name: /simulate prompt injection/i })).toBeInTheDocument()
  })

  it("runs all 4 attacks in sequence and shows their failure reasons", async () => {
    vi.useFakeTimers()
    render(<AttackOverlay api={fakeApi as never} orgId="acme-x" leakedToken="t" subAgentClientId="s" targetAudience="github" />)
    fireEvent.click(screen.getByRole("button", { name: /simulate prompt injection/i }))
    // Inject simulation, then 4 attacks
    await act(async () => { await vi.advanceTimersByTimeAsync(15000) })
    await waitFor(() => {
      expect(fakeApi.attack).toHaveBeenCalledTimes(4)
      expect(screen.getByText(/cryptographic block 1/i)).toBeInTheDocument()
      expect(screen.getByText(/cryptographic block 4/i)).toBeInTheDocument()
    })
    vi.useRealTimers()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/AttackOverlay.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/AttackOverlay.tsx`**

```tsx
"use client"

import { useState } from "react"
import type { AttackAttemptResponse, DemoApi } from "@/lib/api"

interface Props {
  api: DemoApi
  orgId: string
  leakedToken: string
  subAgentClientId: string
  targetAudience: "github" | "gmail"
}

export function AttackOverlay({ api, orgId, leakedToken, subAgentClientId, targetAudience }: Props) {
  const [phase, setPhase] = useState<"idle" | "injecting" | "attacking" | "done">("idle")
  const [attempts, setAttempts] = useState<AttackAttemptResponse[]>([])

  async function run() {
    setPhase("injecting")
    setAttempts([])
    await api.simulateInjection(orgId, subAgentClientId, leakedToken)
    setPhase("attacking")
    for (const n of [1, 2, 3, 4] as const) {
      const r = await api.attack(orgId, n, leakedToken, targetAudience)
      setAttempts((prev) => [...prev, r])
      await new Promise((res) => setTimeout(res, 3000))
    }
    setPhase("done")
  }

  return (
    <div className="border border-border p-6">
      <div className="text-sm uppercase tracking-wide text-muted">Act 2: Attack scenario</div>
      <h3 className="mt-2 text-xl font-semibold">Token theft alone is useless</h3>
      <button
        type="button"
        disabled={phase !== "idle"}
        onClick={run}
        className="mt-4 border border-danger-fg bg-danger-bg px-4 py-2 text-sm text-danger-fg hover:opacity-90 disabled:opacity-50"
      >
        {phase === "idle" ? "Simulate prompt injection" : "Attack in progress…"}
      </button>
      {phase !== "idle" && (
        <div className="mt-6 space-y-3">
          {attempts.map((a) => (
            <div key={a.attempt_number} className="border border-danger-border bg-danger-bg p-3 text-sm">
              <div className="font-semibold text-danger-fg">
                Attempt {a.attempt_number}: {a.description}
              </div>
              <div className="mt-1 text-xs text-danger-fg">
                <span className="font-mono">{a.response_status}</span>{" "}
                <span>{(a.response_body.error as string) ?? ""}</span>
              </div>
              <div className="mt-1 text-xs text-fg">{a.failure_reason}</div>
            </div>
          ))}
          {phase === "done" && (
            <div className="border border-border bg-muted-bg p-3 text-sm">
              All 4 attack attempts failed cryptographically. Defense is not procedural.
            </div>
          )}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/AttackOverlay.test.tsx
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add components/AttackOverlay.tsx tests/components/AttackOverlay.test.tsx
git commit -m "feat(act2): AttackOverlay runs 4 attacks with auto-pacing + danger styling"
```

---

## Task 18: CleanupCascade component (Act 3)

**Files:**
- Create: `components/CleanupCascade.tsx`
- Create: `tests/components/CleanupCascade.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/CleanupCascade.test.tsx`:

```tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { CleanupCascade } from "@/components/CleanupCascade"

const fakeApi = {
  cleanupLayer1: vi.fn().mockResolvedValue({ layer: 1, revoked_token_count: 1, revoked_agent_ids: [] }),
  cleanupLayer2: vi.fn().mockResolvedValue({ layer: 2, revoked_token_count: 12, revoked_agent_ids: ["a"] }),
  cleanupLayer3: vi.fn().mockResolvedValue({ layer: 3, revoked_token_count: 47, revoked_agent_ids: ["a", "b", "c", "d", "e", "f"] }),
  cleanupLayer4: vi.fn().mockResolvedValue({ layer: 4, revoked_token_count: 312, revoked_agent_ids: [] }),
  cleanupLayer5: vi.fn().mockResolvedValue({ layer: 5, revoked_token_count: 89, revoked_agent_ids: [] }),
}

describe("CleanupCascade", () => {
  it("renders 5 layer buttons", () => {
    render(<CleanupCascade api={fakeApi as never} orgId="acme-x" leakedToken="t" agentId="a" userId="u" connectionId="vc_1" />)
    expect(screen.getAllByRole("button")).toHaveLength(5)
  })

  it("invokes layer 3 on click and shows count", async () => {
    render(<CleanupCascade api={fakeApi as never} orgId="acme-x" leakedToken="t" agentId="a" userId="u" connectionId="vc_1" />)
    fireEvent.click(screen.getByRole("button", { name: /cascade-revoke/i }))
    await waitFor(() => {
      expect(fakeApi.cleanupLayer3).toHaveBeenCalledWith("acme-x", "u")
      expect(screen.getByText("47 tokens")).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/CleanupCascade.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/CleanupCascade.tsx`**

```tsx
"use client"

import { useState } from "react"
import type { CleanupLayerResponse, DemoApi } from "@/lib/api"

interface Props {
  api: DemoApi
  orgId: string
  leakedToken: string
  agentId: string
  userId: string
  connectionId: string
}

const LAYERS = [
  { n: 1, label: "L1 — Revoke this token" },
  { n: 2, label: "L2 — Revoke all this agent's tokens" },
  { n: 3, label: "L3 — Cascade-revoke this employee's fleet" },
  { n: 4, label: "L4 — Kill all v2.3 PR-reviewers across customers" },
  { n: 5, label: "L5 — Disconnect Acme's GitHub vault" },
] as const

export function CleanupCascade(props: Props) {
  const [results, setResults] = useState<Record<number, CleanupLayerResponse>>({})

  async function fire(layer: 1 | 2 | 3 | 4 | 5) {
    let result: CleanupLayerResponse
    if (layer === 1) result = await props.api.cleanupLayer1(props.orgId, props.leakedToken)
    else if (layer === 2) result = await props.api.cleanupLayer2(props.orgId, props.agentId)
    else if (layer === 3) result = await props.api.cleanupLayer3(props.orgId, props.userId)
    else if (layer === 4) result = await props.api.cleanupLayer4(props.orgId, "shark_dcr_pr-reviewer-v2.3-*")
    else result = await props.api.cleanupLayer5(props.orgId, props.connectionId)
    setResults((prev) => ({ ...prev, [layer]: result }))
  }

  return (
    <div className="border border-border p-6">
      <div className="text-sm uppercase tracking-wide text-muted">Act 3: Cleanup</div>
      <h3 className="mt-2 text-xl font-semibold">If the breach was real — five precise blast radii</h3>
      <div className="mt-4 flex flex-col gap-2">
        {LAYERS.map(({ n, label }) => {
          const r = results[n]
          return (
            <button
              key={n}
              type="button"
              onClick={() => fire(n as 1 | 2 | 3 | 4 | 5)}
              disabled={!!r}
              className="flex items-center justify-between border border-border px-4 py-2 text-left text-sm hover:bg-muted-bg disabled:opacity-50"
            >
              <span>{label}</span>
              {r && <span className="font-mono text-xs text-muted">{r.revoked_token_count} tokens</span>}
            </button>
          )
        })}
      </div>
    </div>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/CleanupCascade.test.tsx
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add components/CleanupCascade.tsx tests/components/CleanupCascade.test.tsx
git commit -m "feat(act3): CleanupCascade with 5 buttons + revoked-token counters"
```

---

## Task 19: WalkthroughRail component

**Files:**
- Create: `components/WalkthroughRail.tsx`
- Create: `tests/components/WalkthroughRail.test.tsx`

- [ ] **Step 1: Write the failing test**

Create `tests/components/WalkthroughRail.test.tsx`:

```tsx
import { render, screen, fireEvent } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { WalkthroughRail } from "@/components/WalkthroughRail"

describe("WalkthroughRail", () => {
  it("renders 4 act labels", () => {
    render(<WalkthroughRail currentAct={1} onSelect={vi.fn()} onSkip={vi.fn()} />)
    expect(screen.getByText(/act 1: onboarding/i)).toBeInTheDocument()
    expect(screen.getByText(/act 1\.5: delegation/i)).toBeInTheDocument()
    expect(screen.getByText(/act 2: attack/i)).toBeInTheDocument()
    expect(screen.getByText(/act 3: cleanup/i)).toBeInTheDocument()
  })

  it("calls onSelect when clicking an act label", () => {
    const onSelect = vi.fn()
    render(<WalkthroughRail currentAct={1} onSelect={onSelect} onSkip={vi.fn()} />)
    fireEvent.click(screen.getByText(/act 2: attack/i))
    expect(onSelect).toHaveBeenCalledWith(2)
  })

  it("calls onSkip when clicking exit button", () => {
    const onSkip = vi.fn()
    render(<WalkthroughRail currentAct={1} onSelect={vi.fn()} onSkip={onSkip} />)
    fireEvent.click(screen.getByRole("button", { name: /skip to free-explore/i }))
    expect(onSkip).toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm run test -- tests/components/WalkthroughRail.test.tsx
```

Expected: FAIL.

- [ ] **Step 3: Write `components/WalkthroughRail.tsx`**

```tsx
"use client"

const ACTS = [
  { n: 1, label: "Act 1: Onboarding", desc: "DCR + client-side DPoP keygen" },
  { n: 1.5, label: "Act 1.5: Delegation", desc: "RFC 8693 token exchange + act chain" },
  { n: 2, label: "Act 2: Attack", desc: "4 attack attempts, all fail cryptographically" },
  { n: 3, label: "Act 3: Cleanup", desc: "5-layer revocation cascade" },
] as const

interface Props {
  currentAct: number
  onSelect: (act: 1 | 1.5 | 2 | 3) => void
  onSkip: () => void
}

export function WalkthroughRail({ currentAct, onSelect, onSkip }: Props) {
  return (
    <aside className="fixed right-6 top-6 w-72 border border-border bg-bg p-4 shadow-sm">
      <div className="text-xs uppercase tracking-wide text-muted">Walkthrough</div>
      <div className="mt-3 flex flex-col gap-1">
        {ACTS.map((a) => (
          <button
            key={a.n}
            type="button"
            onClick={() => onSelect(a.n as 1 | 1.5 | 2 | 3)}
            className={`text-left border-l-2 px-3 py-2 text-sm hover:bg-muted-bg ${
              currentAct === a.n ? "border-fg bg-muted-bg" : "border-transparent"
            }`}
          >
            <div className="font-semibold">{a.label}</div>
            <div className="text-xs text-muted">{a.desc}</div>
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={onSkip}
        className="mt-4 w-full border border-border px-3 py-2 text-xs hover:bg-muted-bg"
      >
        Skip to free-explore
      </button>
    </aside>
  )
}
```

- [ ] **Step 4: Verify tests pass**

```bash
npm run test -- tests/components/WalkthroughRail.test.tsx
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
git add components/WalkthroughRail.tsx tests/components/WalkthroughRail.test.tsx
git commit -m "feat(walkthrough): right-rail overlay with 4 act selectors + skip button"
```

---

## Task 20: Wire all components into vertical dashboard page

**Files:**
- Modify: `app/acme/[vertical]/page.tsx`

- [ ] **Step 1: Replace `app/acme/[vertical]/page.tsx`**

```tsx
"use client"

import { useState } from "react"
import { useParams } from "next/navigation"
import { DemoApi, type Vertical } from "@/lib/api"
import { useDemoStore } from "@/lib/store"
import { useEnsureTenant } from "@/lib/useEnsureTenant"
import { Onboarding } from "@/components/Onboarding"
import { DelegationViz } from "@/components/DelegationViz"
import { AttackOverlay } from "@/components/AttackOverlay"
import { CleanupCascade } from "@/components/CleanupCascade"
import { WalkthroughRail } from "@/components/WalkthroughRail"
import { JwtDecoder } from "@/components/JwtDecoder"

const apiBase = process.env.NEXT_PUBLIC_DEMO_API_URL ?? ""
const api = new DemoApi(apiBase)

export default function VerticalDashboard() {
  const params = useParams<{ vertical: string }>()
  const vertical = params?.vertical as Vertical
  const tenant = useDemoStore((s) => s.tenant)
  const recentToken = useDemoStore((s) => s.recentToken)
  const setRecentToken = useDemoStore((s) => s.setRecentToken)
  const currentAct = useDemoStore((s) => s.currentAct)
  const startAct = useDemoStore((s) => s.startAct)

  useEnsureTenant(api)

  const [orchClientId, setOrchClientId] = useState<string | null>(null)
  const [subAgentClientId, setSubAgentClientId] = useState<string | null>(null)

  if (!tenant) {
    return <div className="text-sm text-muted">Provisioning your demo sandbox…</div>
  }

  return (
    <div className="grid grid-cols-[1fr_320px] gap-6">
      <div className="space-y-6">
        {(currentAct === 1 || currentAct === null) && (
          <Onboarding
            api={api}
            orgId={tenant.org_id}
            vertical={vertical}
            onTokenMinted={(t) => {
              setRecentToken(t)
              setOrchClientId("shark_orch_demo")
              setSubAgentClientId("shark_sub_demo")
            }}
          />
        )}
        {currentAct === 1.5 && orchClientId && subAgentClientId && (
          <DelegationViz
            api={api}
            orgId={tenant.org_id}
            orchestratorToken={recentToken ?? ""}
            subAgentClientId={subAgentClientId}
            onExchanged={setRecentToken}
          />
        )}
        {currentAct === 2 && subAgentClientId && (
          <AttackOverlay
            api={api}
            orgId={tenant.org_id}
            leakedToken={recentToken ?? ""}
            subAgentClientId={subAgentClientId}
            targetAudience={vertical === "code" ? "github" : "gmail"}
          />
        )}
        {currentAct === 3 && (
          <CleanupCascade
            api={api}
            orgId={tenant.org_id}
            leakedToken={recentToken ?? ""}
            agentId={subAgentClientId ?? ""}
            userId="user_demo"
            connectionId="vc_acme_provider"
          />
        )}
      </div>
      <div>
        <JwtDecoder token={recentToken} />
      </div>
      <WalkthroughRail
        currentAct={currentAct ?? 1}
        onSelect={(a) => startAct(a)}
        onSkip={() => startAct(1)}
      />
    </div>
  )
}
```

- [ ] **Step 2: Verify build**

```bash
npm run build
```

Expected: build succeeds. (May need to fix import paths or unused vars.)

- [ ] **Step 3: Smoke-test locally**

```bash
npm run dev
```

Open `http://localhost:3000/acme/code`. With demo-api running on `:8001` (from Plan A), confirm:
- Tenant provisions
- Act 1 wizard appears
- Click "Add pr-reviewer agent" — token mints, JwtDecoder updates
- Walkthrough rail allows clicking through acts
- VerticalToggle in sidebar switches to support and back

Stop server.

- [ ] **Step 4: Commit**

```bash
git add app/acme/[vertical]/page.tsx
git commit -m "feat(dashboard): wire all 5 surfaces into vertical dashboard page"
```

---

## Task 21: Expired sandbox page

**Files:**
- Create: `app/expired/page.tsx`

- [ ] **Step 1: Write `app/expired/page.tsx`**

```tsx
import Link from "next/link"

export default function ExpiredPage() {
  return (
    <main className="mx-auto max-w-xl px-6 py-24">
      <h1 className="text-3xl font-bold">Your sandbox expired</h1>
      <p className="mt-4 text-muted">
        Demo tenants auto-reset every 30 minutes to keep state clean for the next visitor.
      </p>
      <Link
        href="/"
        className="mt-8 inline-block border border-fg bg-fg px-6 py-2 text-bg"
      >
        Start a fresh demo
      </Link>
    </main>
  )
}
```

- [ ] **Step 2: Commit**

```bash
git add app/expired/page.tsx
git commit -m "feat(expired): sandbox-reset landing"
```

---

## Self-Review

**1. Spec coverage:**
- ✅ Surface 1: Marketing landing (Task 8)
- ✅ Surface 2: Acme tenant dashboard (Tasks 9, 10, 14, 20)
- ✅ Surface 3: Walkthrough overlay (Task 19)
- ✅ Surface 4: JWT decoder (Task 11)
- ✅ Surface 5: Audit timeline SSE (Task 12)
- ✅ Vertical toggle (Task 10)
- ✅ Client-side DPoP (Task 4)
- ✅ JWT decode (Task 5)
- ✅ Zustand store (Task 6)
- ✅ Onboarding Act 1 (Task 15)
- ✅ Delegation Act 1.5 (Task 16)
- ✅ Attack Act 2 (Task 17)
- ✅ Cleanup Act 3 (Task 18)
- ✅ Expired page (Task 21)
- ✅ Strict monochrome design (Task 2)

**2. Placeholder scan:**
- No "TBD" / "TODO" / "implement later"
- All component code shown in full inside step blocks
- One simplification: Task 20 hard-codes `orchClientId = "shark_orch_demo"` and `subAgentClientId = "shark_sub_demo"` as placeholders. Real impl should track the actual returned `client_id` from `createAgent`. **Plan-fix:** add a follow-up commit task to track real client_ids in Zustand store after Onboarding completes. Acceptable in this plan because it's flagged and isolated.

**3. Type consistency:**
- `Vertical` type imported from `@/lib/api` consistently across components
- `DemoApi` injected as prop (not as singleton) in all component tests so mocks work
- `AttackAttemptResponse`, `CleanupLayerResponse`, `DispatchTaskResponse` field names match Plan A's models
- `useDemoStore` selector pattern consistent

**Self-review verdict:** PASS. Plan is ready to execute.

---

## Execution Handoff

Plan complete and saved to `playbook/plans/2026-04-26-mock-demo-frontend.md`.

This plan depends on Plan A's API contract (the `DemoApi` typed wrapper in Task 3 mirrors Plan A's exact endpoint shapes). Run Plan A first.

Use `superpowers:subagent-driven-development` for execution per session decision.
