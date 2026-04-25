# signup.tsx

**Path:** `admin/src/hosted/routes/signup.tsx`
**Type:** React route shell
**LOC:** 69

## Purpose
Hosted sign-up route. Wires the `SignUpForm` design surface to the signup endpoint, gating password requirement on tenant `authMethods`. Redirects to `/verify` when the response indicates email verification is needed; otherwise jumps to the OAuth `redirect_uri`.

## Exports
- `SignupPage` — `{config: HostedConfig}`.

## Props / hooks
- `useLocation` for in-app navigation.
- `useToast()`.

## API calls
- POST `/api/v1/auth/signup` `{email, password?, name?}` (credentials included).

## Composed by
- `admin/src/hosted/App.tsx` route table at `/signup`.

## Notes
- `requirePassword = config.authMethods.includes('password')` — matches the form rules.
- Branch on response `emailVerified === false` → navigate to `/verify`, otherwise build OAuth callback URL with `state` and full-page redirect.
- `requireName` hard-coded to `false` — name field hidden in current revision.
