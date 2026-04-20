# Dashboard DX Progress Log

Append-only. Every task state change = one row. Never edit prior rows.

| ts (UTC ISO) | task | event | agent | commit/notes |
|--------------|------|-------|-------|--------------|
| 2026-04-20T00:00:00Z | - | plan_created | plan-devex-review | DASHBOARD_DX_EXECUTION_PLAN.md + STATUS.json + this log initialized |
| 2026-04-20T18:00:00Z | T01 | claim | claude-opus-4-7 | 25-min Wave-α burst: remove hardcoded lies in layout.tsx |
| 2026-04-20T18:00:00Z | T07 | claim | claude-opus-4-7 | 25-min Wave-α burst: alert()→toast in signing_keys + dev_inbox |
| 2026-04-20T18:00:00Z | T23 | claim | claude-opus-4-7 | 25-min Wave-α burst: CLIFooter blanket add to 6 pages |
| 2026-04-20T18:22:00Z | T01 | done | claude-opus-4-7 | layout.tsx: version wired to /admin/health, health+env dot wired, nimbus-prod chip removed, fake '4'/'1.2k' badges removed. Typecheck pass. |
| 2026-04-20T18:22:00Z | T07 | done | claude-opus-4-7 | signing_keys.tsx:90/96 + dev_inbox.tsx:58 alert()→toast. Only users.tsx:65 remains (blocked by T04). Typecheck pass. |
| 2026-04-20T18:22:00Z | T23 | done | claude-opus-4-7 | CLIFooter added to dev_inbox, consents_manage, settings, authentication. vault_manage+proxy_config already had it (audit drift). Typecheck pass. |
| 2026-04-20T18:25:00Z | T01,T07,T23 | committed | claude-opus-4-7 | 8406b78 — feat(dx): Wave-α T01+T07+T23 |
| 2026-04-20T18:30:00Z | T22 | claim | claude-opus-4-7 | vault silent catch → toast |
| 2026-04-20T18:30:00Z | T02 | claim | claude-opus-4-7 | overview MOCK residue purge |
| 2026-04-20T18:33:00Z | T22 | done | claude-opus-4-7 | verified-by-inspection: all 7 catches in vault_manage.tsx already toast or setError (358,533,685,770,834,965,973). Audit was wrong. |
| 2026-04-20T18:33:00Z | T02 | done | claude-opus-4-7 | verified-by-inspection: overview.tsx no MOCK import, comments confirm "never MOCK fallbacks", AttentionPanel derives from /admin/health. Shipped in recent commits (989c319 / 7d464da). |
| 2026-04-20T18:34:00Z | T12 | claim | claude-opus-4-7 | Session Debugger client-side JWT decode + JWKS validate |
| 2026-04-20T18:45:00Z | T12 | done | claude-opus-4-7 | session_debugger.tsx new file: decode header+payload, JWKS fetch, SubtleCrypto verify (RS256/384/512, ES256/384), exp/nbf status pills, claim annotation w/ ISO timestamps, copy buttons, CLIFooter. Wired into App.tsx, removed from empty_shell, ph:5 dropped from NAV. Typecheck pass. |
| 2026-04-20T18:45:00Z | T22,T02 | done | claude-opus-4-7 | verified-by-inspection: shipped in prior commits |
| 2026-04-20T19:00:00Z | T08 | claim | claude-opus-4-7 | verify 6 org silent-404 routes + fix error handling |
| 2026-04-20T19:00:00Z | T13 | claim | claude-opus-4-7 | Compliance page demote (audit export + GDPR stub) |
| 2026-04-20T19:00:00Z | T17 | claim | claude-opus-4-7 | Help menu + floating ? + Cmd+K help commands |
| 2026-04-20T19:00:00Z | T18 | claim | claude-opus-4-7 | Feedback button (bundled with T17) |
| 2026-04-20T19:18:00Z | T08 | done | claude-opus-4-7 | verified-by-inspection: router.go:578-588 all 6 admin org routes shipped (c3609db). organizations.tsx catches at 244/468/565/617/627 all surface errors via setErr or toast.error. No silent 404s. |
| 2026-04-20T19:18:00Z | T13 | done | claude-opus-4-7 | compliance.tsx new file: Audit Export tab (presets + date range + csv/json download from /audit-logs/export), GDPR stub tab (CLI-backed erasure + export), Access Review snapshot (non-MFA users + failed-login signal). Wired into App.tsx, removed from empty_shell, ph:9 dropped from NAV. Typecheck pass. |
| 2026-04-20T19:18:00Z | T17,T18 | done | claude-opus-4-7 | HelpButton.tsx new file: floating ? bottom-right (Cmd+/ opens), help menu (Docs, Changelog, GitHub, Report bug), FeedbackModal w/ auto-fill page+version+recent console errors, submit via GitHub issue prefill or mailto. CommandPalette: added 4 Help actions (Docs/Changelog/GitHub/Report bug) dispatched via window.__shark_help. Typecheck pass. |
| 2026-04-20T19:20:00Z | T08,T13,T17,T18 | committed | claude-opus-4-7 | dc3cef2 |
| 2026-04-20T19:21:00Z | T19 | claim | claude-opus-4-7 | API keys scope picker at create |
| 2026-04-20T19:25:00Z | T19 | done | claude-opus-4-7 | verified-by-inspection: api_keys.tsx:602-614 scope picker already shipped (8 scopes, checkboxes, danger badges, descriptions). |
| 2026-04-20T19:25:00Z | T14 | claim | claude-opus-4-7 | hide phase-gated stubs behind settings toggle |
| 2026-04-20T19:38:00Z | T14 | done | claude-opus-4-7 | TWEAK_DEFAULTS adds showPreview:false. Tweaks panel adds Preview features toggle (Hidden/Shown). Sidebar filters section.items by ph<=CURRENT_PHASE||showPreview, hides empty groups. Default: 8 phase-gated stubs (tokens, explorer, schemas, oidc, impersonation, migrations, branding) hidden from new admins. Typecheck pass. |
