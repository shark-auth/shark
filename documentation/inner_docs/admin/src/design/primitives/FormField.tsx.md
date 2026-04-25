# FormField.tsx

**Path:** `admin/src/design/primitives/FormField.tsx`
**Type:** React primitive
**LOC:** 99

## Purpose
Composable label + input + helper/error wrapper. Spreads `inputProps` onto a default `Input` (or accepts a `children` override), threads aria attributes, and switches helper text into an error tone when `errorText` is set.

## Exports
- `FormField` — `{label?, helperText?, errorText?, required?, htmlFor?, inputProps?, children?, style?}`.
- `FormFieldProps` (interface).

## Props / hooks
- `label?` — optional header text; required marker = red asterisk when `required` is true.
- `helperText?` vs `errorText?` — error wins, and triggers `role="alert"` on the help text.
- `htmlFor?` — overrides the auto-bound `inputProps.id`.
- `inputProps?` — full `InputProps` passthrough.
- `children?` — escape hatch (e.g. wrap a textarea or custom control instead of `Input`).
- No internal state.

## API calls
- None.

## Composed by
- `SignInForm`, `SignUpForm`, `ForgotPasswordForm`, `ResetPasswordForm` (all email/password fields).
- Any consumer-side form built on the design system.

## Notes
- Auto-wires `aria-describedby` to `${id}-help` when help text is present.
- `aria-required` only set when `required` is true (omits when undefined).
- Styling driven by design `tokens` only — no CSS modules.
