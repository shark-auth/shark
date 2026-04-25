# empty_shell.tsx

**Path:** `admin/src/components/empty_shell.tsx`

Coming-soon placeholder for dashboard routes that exist in the sidebar but whose feature is post-launch roadmap. Each export renders a titled card with a brief description of what the surface will do — no phase numbers, no internal state exposed.

## Exports

| Export | Route | Description shown |
|--------|-------|-------------------|
| `Tokens` | `tokens` | Active OAuth tokens across agents and users; bulk revocation |
| `APIExplorer` | `api-explorer` | In-dashboard API playground; curl/SDK snippet copy |
| `EventSchemas` | `event-schemas` | Webhook + audit event payload schema reference browser |
| `SessionDebugger` | `debug` | JWT decode, live session inspect, delegation chain trace |
| `OIDCProvider` | `oidc-provider` | SharkAuth as OIDC IdP; federate with other services |
| `Impersonation` | `impersonation` | Sign in as any user; all actions audit-logged with admin attribution |
| `Migrations` | `migrations` | Import from Auth0/Clerk/Supabase with field mapping and dry-run |

## Pattern

`ComingSoon({ title, description, eta? })` — shared private component. Inline styles using design-system CSS vars. `eta` is optional; omit to avoid leaking roadmap timelines.

## Notes

- `SessionDebugger` was previously a full implementation (`session_debugger.tsx`) but the `debug` route now points here as of v0.9.0 cleanup. The real component file remains on disk but is orphaned (not imported by `App.tsx`).
- Replace any export with a real component by updating the import in `App.tsx` — no changes needed here.
- Composed by `App.tsx` (imported as named exports).
