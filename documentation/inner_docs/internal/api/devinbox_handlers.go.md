# devinbox_handlers.go

**Path:** `internal/api/devinbox_handlers.go`
**Package:** `api`
**LOC:** 71
**Tests:** likely integration-tested

## Purpose
Dev-mode email inbox — when the email sender is configured for the dev/devnull provider, outgoing emails are persisted to `dev_emails` and surfaced through these admin endpoints so developers can read auth links without ever sending mail.

## Handlers exposed
- `handleListDevEmails` (line 26) — GET `/admin/dev/emails`. Optional `?limit=N` (default 100). Returns `{data: [...]}`.
- `handleGetDevEmail` (line 48) — GET `/admin/dev/emails/{id}`. 404 on miss.
- `handleDeleteAllDevEmails` (line 65) — DELETE `/admin/dev/emails`. Returns 204.

## Key types
- `devInboxListResponse` (line 13) — `{data: []devEmailResponse}`
- `devEmailResponse` (line 17) — `{id, to, subject, html, text, created_at}`

## Imports of note
- `internal/storage` — `ListDevEmails`, `GetDevEmail`, `DeleteAllDevEmails`

## Wired by
- `internal/api/router.go:667-669`

## Notes
- Only meaningful when `email.provider == "dev"` (devnull) — production email sends never persist here.
- No pagination beyond `?limit`; intended for short-lived dev sessions.
