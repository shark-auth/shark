# OAuthProviderButton.tsx

**Path:** `admin/src/design/composed/OAuthProviderButton.tsx`
**Type:** React component
**LOC:** 107

## Purpose
"Continue with {Provider}" button for an OAuth provider — renders provider icon (URL or first-letter fallback), provider name, and a disabled-during-loading state with a 3s safety timeout for the typical full-page redirect.

## Exports
- `OAuthProviderButton` — `{providerID, providerName, iconUrl?, onClick, loading?}`.
- `OAuthProviderButtonProps` (interface).
- Internal `ProviderIcon` helper.

## Props / hooks
- `providerID`, `providerName` — display + (caller-side) routing.
- `iconUrl?` — optional; falls back to a circular initial badge if omitted or load fails.
- `onClick()` — fires the OAuth start action (typically a `window.location` redirect).
- `loading?` — externally controlled spinner state, OR'd with internal click state.

## API calls
- None — caller wires `onClick` to `/api/v1/auth/oauth/{providerID}`.

## Composed by
- `SignInForm` (rendered for each entry in `oauthProviders` prop).

## Notes
- Internal loading state auto-resets after 3s so a click that doesn't navigate doesn't leave the button frozen.
- Image error swap is a manual DOM mutation (`onError` flips the sibling fallback into view).
- Built on the `Button` primitive (ghost / lg).
