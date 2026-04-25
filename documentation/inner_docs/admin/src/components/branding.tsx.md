# branding.tsx

**Path:** `admin/src/components/branding.tsx`
**Type:** React component (page)
**LOC:** ~400

## Purpose
Branding customization—customize login/signup forms with logos, colors, email templates, integrations (Segment, Intercom).

## Exports
- `Branding()` (default) — function component

## Tabs
- **Visuals** — logo, favicon, primary color, secondary color, font family
- **Email** — sender email, reply-to, custom templates (welcome, password reset, MFA)
- **Integrations** — Segment analytics, Intercom support chat

## Features
- **Logo upload** — animated logo on auth pages
- **Color picker** — theme primary/secondary colors
- **Typography** — font family override (System|Inter|Manrope|Georgia)
- **Email templates** — HTML/plain text template editing
- **Preview** — see changes on sample auth page
- **Analytics** — send auth events to Segment
- **Support** — Intercom chat on auth pages

## Hooks used
- `useAPI('/admin/branding')` — fetch config
- `useToast()` — save feedback

## State
- `tab` — current tab (visuals|email|integrations)
- `config` — branding settings
- `saving`, `uploading` — async states

## API calls
- `GET /api/v1/admin/branding` — fetch config
- `PATCH /api/v1/admin/branding` — update
- `POST /api/v1/admin/branding/upload-logo` — upload logo file

## Composed by
- App.tsx

## Notes
- All changes apply to all hosted auth pages
- Email templates use Handlebars for variable substitution ({{user.email}}, {{reset_link}})
- Color and font choices affect entire auth experience
