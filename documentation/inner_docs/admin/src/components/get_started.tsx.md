# get_started.tsx

**Path:** `admin/src/components/get_started.tsx`
**Type:** React component (page)
**Last rebuild:** 2026-04-26 — Wave 1.7 Edit 4 full impeccable rebuild (agent-first)

## Purpose

Agent-first onboarding page. Audience: platform engineers who just installed SharkAuth and need to ship their first DPoP-bound token in ≤5 minutes. Replaces the 12-step OAuth-generic flow (Wave 1 / W17 fallback) with six focused sections organized around the agent moat narrative.

## Exports

- `GetStarted({ setPage })` — function component
- `SECTION_IDS` — exported string array of the six section markers (smoke-tested)

## Three tracks (tab selector in toolbar)

| Track | Audience | Sections shown |
|---|---|---|
| Agent integration (default) | Platform engineers shipping agent products | All 6 sections |
| Human auth | Apps adding sign-in only | Why SharkAuth + Next steps + human task list |
| Both | Full platform: agents + human sign-in | All 6 sections + human tasks appended |

## Six sections (in order)

### 1. Why SharkAuth (`why-sharkauth`)
Three bullet points, no buzzwords:
- Per-token, per-agent, per-customer revocation — five layers, RFC-correct.
- DPoP-bound tokens (RFC 9449): stolen tokens cannot be replayed.
- Delegation chain audit (RFC 8693): every act claim logged with full chain.

### 2. 60-second setup (`sixty-second-setup`)
Three tasks: install binary, run `shark serve`, copy admin key from first-boot banner.

### 3. Your first agent (`your-first-agent`)
Three tasks: register agent (dashboard or `shark agents create`), get credentials from drawer, set delegation policy.

### 4. DPoP-bound token in 10 lines (`dpop-token-ten-lines`)
One task with tabbed code snippets: Python primary (the Wave 2 README 10-liner), curl alternative, TypeScript placeholder (W18).

```python
import shark_auth

client = shark_auth.Client(
    issuer="https://auth.example.com",
    client_id=CLIENT_ID,
    client_secret=CLIENT_SECRET,
)

prover = shark_auth.DPoPProver.generate()
token = client.oauth.get_token_with_dpop(prover)
print(token.access_token)
```

### 5. Delegation chain demo (`delegation-chain-demo`)
Two tasks: run `shark demo delegation-with-trace`, verify audit trail at `/audit?delegation=true`.

### 6. Next steps (`next-steps`)
Three external links: API reference docs, screencast (90s), GitHub repo.

## Layout

- **Toolbar** — three-track tab strip (Agent / Human auth / Both), `done/total` fraction, 4px progress bar, Recheck, Skip to dashboard
- **Sub-header** — one-sentence framing matching the selected track
- **Body** — accordion sections, each collapsible. Clicking a task row opens the drawer.
- **Drawer** — 420px right-side fixed panel (same pattern as users.tsx). Contents: task id (mono), title, status dot, "What this does" prose, deep-link card, tabbed code snippets with copy-on-click, auto-verify notice.
- **Footer** — pending/done/skipped counts, auto-sync note

## Auto-verify probes

| autoCheck key | API probe | Condition |
|---|---|---|
| `hasAdminKey` | `GET /admin/bootstrap/status` | `consumed` or `bootstrapped` truthy |
| `hasAgent` | `GET /admin/agents?limit=1` | agents array length > 0 |
| `hasAgentPolicy` | `GET /admin/agents/{id}/policies` | policies array length > 0 |
| `hasDPoPToken` | `GET /admin/audit?event_type=oauth.token.issued&limit=5` | any event has `dpop` metadata |
| `hasDelegationEvent` | `GET /admin/audit?delegation=true&limit=1` | events array length > 0 |

## Persistence

`shark_get_started_v2` localStorage — `{ track, tasks: {[id]: status}, open: {[sectionId]: bool} }`.

## Tabbed code snippets

Drawer uses a tab strip for snippets: Python primary → CLI → extraSnippets (TypeScript placeholder). Tab selection is ephemeral (not persisted).

## Visual contract (per .impeccable.md v3)

- Monochrome: status dots carry color only (`--success` / `--warn` / `--danger` / `--fg-faint`)
- Border radius: 3-4px everywhere — no pills, no 6px+
- 13px base text, hairline borders, density matches users.tsx
- Section accordion rows: 12px uppercase labels, chevron toggles
- Task rows: index (mono 10.5px) + title (13px 500) + summary (11px faint) + status dot + chevron
- Drawer: 420px fixed right, hairline left border, square. NOT a modal.
- No hero, no recommended badges, no colored cards, no YAML

## Smoke coverage

`tests/smoke/test_w17_get_started_rebuild.py`:
- `test_get_started_renders_six_sections` — all six `SECTION_IDS` present in compiled bundle
- `test_code_block_python_snippet_present` — DPoP 10-liner markers in bundle
- `test_no_legacy_oauth_step_phrases` — old proxy/SDK OAuth-generic phrasing absent

## Composed by

- `App.tsx` — route `get-started`

## Notes

- Human auth tasks (HUMAN_TASKS) are only rendered when track is `human` or `both`
- No "session vs JWT" framing anywhere
- Cookie + JWT both always on; SDK path uses both transparently
- `shark_admin_onboarded` localStorage key set on "Skip to dashboard" or "Go" actions
