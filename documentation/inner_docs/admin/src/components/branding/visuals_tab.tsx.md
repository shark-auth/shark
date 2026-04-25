# visuals_tab.tsx

**Path:** `admin/src/components/branding/visuals_tab.tsx`
**Type:** React component (subtab)
**LOC:** 507

## Purpose
Branding > Visuals subtab. Editable visual identity form (logo upload, primary/accent colors, font, footer text, email from-name / from-address) on the left; sandboxed iframe live preview of the magic_link template on the right. Debounced 300ms preview round-trips to the server.

## Exports
- `BrandingVisualsTab` (named).
- `Field` and other helpers live further in the file.
- `errorCodeMessage(code)` — maps backend logo-upload error codes (`file_too_large`, `invalid_file_type`, `invalid_svg`) to admin-friendly toast copy.
- `shallowEqual(a, b)` — flat-row diff for `dirty` gate.

## Props / hooks
- props: `{}`.
- State: `cfg` (last canonical), `draft` (in-progress), `previewHTML`, `saving`, `uploading`.
- Derived: `dirty = !shallowEqual(cfg, draft)`.
- `useToast()`.

## API calls
- GET `/api/v1/admin/branding` → `{branding}`.
- PATCH `/api/v1/admin/branding` with full draft.
- POST `/api/v1/admin/email-templates/magic_link/preview` `{config: draft, sample_data}` — debounced live preview.
- POST `/api/v1/admin/branding/logo` (multipart, `logo` field).
- DELETE `/api/v1/admin/branding/logo`.

## Composed by
- `admin/src/components/branding.tsx` (Branding tab router).

## Notes
- Logo upload + delete are server-persisted immediately — `cfg` is updated alongside `draft` so the Save button doesn't light up for an already-committed change.
- Preview failures keep the last good HTML (only `console.warn`-ed) so the iframe doesn't blank.
- Color pickers from `react-colorful` (`HexColorPicker`).
- Font options: Manrope / Inter / IBM Plex.
