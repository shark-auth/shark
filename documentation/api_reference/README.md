# SharkAuth API Reference

Source of truth for the SharkAuth REST API specification. Drives the public docs at `sharkauth.com/docs`.

## Layout

```
documentation/api_reference/
├── README.md          (you are here)
├── openapi.yaml       master file: info, servers, tags, security schemes, shared schemas
└── sections/
    ├── oauth.yaml     OAuth 2.1 authorization server (RFC 6749/7591/7636/7662/7009/8414/8628/8693/8707/9449)
    ├── auth.yaml      User-facing authentication (signup, login, MFA, passkey, magic-link, social)
    ├── platform.yaml  Multi-tenant orgs, RBAC, SSO connections, API keys, audit, webhooks, agents
    ├── admin.yaml     Operator surface: health, config, apps, vault, branding, email, proxy, paywall
    └── system.yaml    Liveness probe, branding asset serving, hosted pages
```

The master `openapi.yaml` does not redeclare paths from the sections. It carries shared metadata (info, security schemes, error schemas, tag groups). The five section files each carry the operations and schemas for their surface.

## Surface coverage

| Section | Endpoints | Tags |
|---|---|---|
| OAuth 2.1 | ~12 | oauth-discovery, oauth-flow, oauth-tokens, oauth-dcr, oauth-device, dpop |
| Auth & Identity | ~32 | auth-signup-login, auth-sessions, auth-passkey, auth-mfa, auth-magic-link, auth-password, auth-email, auth-sso, auth-social, auth-consent |
| Platform | ~70 | organizations, roles, permissions, authz, users-admin, sso-connections, saml, oidc, api-keys, audit-logs, webhooks, agents, migration |
| Admin | ~80 | admin-system, admin-apps, admin-vault, admin-bootstrap, admin-oauth, admin-users, admin-orgs, admin-branding, admin-email, admin-devinbox, admin-proxy, admin-paywall, admin-flows |
| System | ~7 | system-health, system-branding, system-hosted |

Total: ~200 operations across 38 tags.

## How to view locally

### Scalar (recommended — best for multi-file)

```bash
npx @scalar/cli serve documentation/api_reference/openapi.yaml
```

Opens at `http://localhost:5050`. Tag groups render as a left-nav. Per-tag `externalDocs` links jump to the section file the tag lives in.

### Redocly (best for bundled single file)

```bash
# Bundle 6 files into 1 self-contained spec
npx @redocly/cli bundle documentation/api_reference/openapi.yaml \
  --ext yaml \
  --output documentation/api_reference/_bundled.yaml

# Preview the bundled spec
npx @redocly/cli preview-docs documentation/api_reference/_bundled.yaml
```

`_bundled.yaml` is gitignored — produced on demand for the docs site build.

### Stoplight Elements (mock server + docs)

```bash
docker run --rm -p 8080:4010 \
  -v $PWD/documentation/api_reference:/spec \
  stoplight/prism mock /spec/openapi.yaml
```

## Per-operation `x-source` extension

Every operation includes an `x-source` field pointing to the file + line in the SharkAuth Go codebase where that route is registered:

```yaml
/api/v1/agents:
  post:
    operationId: createAgent
    x-source: "internal/api/router.go:510"
```

Use `x-source` to:
- Verify spec accuracy by jumping from docs to the actual handler.
- Audit drift: if a route moves and `x-source` doesn't update, the spec is stale.
- File bug reports referencing the specific implementation line.

## Regenerating

The spec is currently maintained by hand from `internal/api/router.go` + handler files in `internal/api/*.go`, `internal/oauth/*.go`, `internal/sso/*.go`.

Regeneration workflow (until automation lands post-launch):

1. Diff the router: `git log -p internal/api/router.go` since last spec update.
2. For each new/changed route, locate the section file by tag (e.g. `auth-mfa` → `sections/auth.yaml`).
3. Add or modify the operation. Always include `x-source: "internal/api/router.go:<line>"`.
4. Schemas live in the section file's `components.schemas`. Cross-section shared schemas (`OAuthError`, `ErrorEnvelope`, `Pagination`) are in the master `openapi.yaml`.
5. Validate: `npx @redocly/cli lint documentation/api_reference/openapi.yaml`.
6. Bundle and visually preview before committing.

Post-launch, automation: a Go program that walks `chi.Router` introspection and produces section YAMLs is on the roadmap. Until then, hand-maintained spec is the contract.

## Versioning

`info.version` in each file tracks the SharkAuth release. Keep all 6 files in sync on bump. Breaking schema changes bump major; additive changes bump minor.

`v0.9.x` is the current launch line. `v1.0.0` is gated on two weeks of partner usage without breaking changes (per launch decisions).

## Citing the spec in a bug report

When filing a SharkAuth issue that references the API:

```
Endpoint: POST /api/v1/oauth/token (operationId: tokenIssue)
Spec source: documentation/api_reference/sections/oauth.yaml @ commit <sha>
Implementation source: internal/oauth/handlers.go @ <sha>
```

The dual-source citation makes it unambiguous whether the bug is in the spec, the implementation, or a drift between them.

## Errors catalog

The two error envelopes used in this spec (`OAuthError` for `/oauth/*`, `ErrorEnvelope` for everything else) are documented in detail at `docs/errors.md` in the repo root.

## License

MIT — same as the SharkAuth project. See repo `/LICENSE`.
