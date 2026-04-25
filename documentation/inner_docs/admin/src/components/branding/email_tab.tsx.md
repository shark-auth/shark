# email_tab.tsx

**Path:** `admin/src/components/branding/email_tab.tsx`
**Type:** React component (subtab)
**LOC:** 476

## Purpose
Branding > Email Templates editor. Three-pane layout: template list (left), structured field editor (middle), 300ms-debounced live preview iframe (right). Owns CRUD-lite operations on the seeded template set (magic_link, password_reset, verify_email, organization_invitation, welcome).

## Exports
- `BrandingEmailTab` (named) — default subtab component.
- `templateLabel(id)` (helper).

## Props / hooks
- props: `{}`.
- State: `templates`, `selected`, `draft`, `previewHTML`, `saving`, `sending`, `resetting`.
- `useToast()` for save/test/reset feedback.

## API calls
- GET `/api/v1/admin/email-templates`
- POST `/api/v1/admin/email-templates/{id}/preview` (body `{}` — uses backend default sample data + global branding)
- PATCH `/api/v1/admin/email-templates/{id}` (subject, preheader, header_text, body_paragraphs, cta_text, cta_url_template, footer_text)
- POST `/api/v1/admin/email-templates/{id}/send-test` (body `{to_email}`)
- POST `/api/v1/admin/email-templates/{id}/reset`

## Composed by
- `admin/src/components/branding.tsx` (Branding shell tab router).

## Notes
- No rich-text input by design (spec decision #4A) — admins type Go template literals like `{{.AppName}}` directly into plain inputs.
- Re-clicking the already-selected template is a no-op so an in-progress draft isn't clobbered.
- Preview render failures are swallowed (`console.warn`) so the iframe doesn't flash empty mid-edit.
- `window.prompt` and `window.confirm` are used for send-test address + reset confirmation.
