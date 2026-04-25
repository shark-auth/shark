# discord.go

**Path:** `internal/auth/providers/discord.go`
**Package:** `providers`
**LOC:** 96
**Tests:** covered by oauth_test.go

## Purpose
Discord OAuth 2.0 provider implementation conforming to `auth.OAuthProvider`.

## Key types / functions
- `discordEndpoint` (var, line 16) — hardcoded auth/token URLs (`#nosec G101`).
- `discordUserURL` (const, line 21) — `https://discord.com/api/users/@me`.
- `Discord` (type, line 24) — wraps `oauth2.Config`.
- `NewDiscord` (func, line 29) — default scopes `["identify", "email"]`; redirect `<baseURL>/api/v1/auth/oauth/discord/callback`.
- `Name` (func, line 45) — `"discord"`.
- `AuthURL` / `Exchange` (funcs, line 47-53) — standard.
- `GetUser` (func, line 55) — decodes `{id, username, global_name, email, avatar, discriminator}`; prefers `global_name` (new Discord display name) over legacy `username`; constructs CDN avatar URL `https://cdn.discordapp.com/avatars/<id>/<hash>.png`.

## Imports of note
- `golang.org/x/oauth2` — flow.

## Used by
- `internal/server/server.go` — registered if `cfg.OAuth.Discord.Enabled`.

## Notes
- Avatar URL hardcodes `.png`; could probe `.gif` for animated avatars (avatars starting with `a_`).
- Discord migrated from `username#discriminator` to `global_name`; this code handles both.
