# magic.tsx

**Path:** `admin/src/hosted/routes/magic.tsx`
**Type:** React route shell
**LOC:** 48

## Purpose
"Check your email" landing page after a magic-link request. Reads the recipient address from the URL query string and wires resend to the magic-link API.

## Exports
- `MagicPage` — `{config: HostedConfig}` (config currently unused; reserved for future routing).

## Props / hooks
- `useToast()` for resend errors.
- `useState` initialized once with `decodeURIComponent(?email=...)`.
- Constants: `INITIAL_COOLDOWN_SECONDS = 30`.

## API calls
- POST `/api/v1/auth/magic-link/send` `{email}` (resend).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/magic`.

## Notes
- Cooldown handling lives in the underlying `MagicLinkSent` design surface; this shell just supplies the initial value.
- Errors are toasted AND re-thrown so the design surface can render its inline error block.
- No-op when `email` is missing — caller should always navigate here with `?email=` set.
