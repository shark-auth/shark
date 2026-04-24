# Contract — Require / Allow grammar

Every rule must carry exactly one of `require` or `allow`. `require` is the predicate the caller must satisfy; `allow` is a narrow escape hatch for public surfaces. Both are strings — the grammar is deliberately thin so it fits cleanly into YAML, JSON, CLI args, and URL query params alike.

## Require strings (accepted values)

| String | Matches |
|---|---|
| `anonymous` | Any caller — authenticated or not. Equivalent to `allow: anonymous`; kept as a `require` spelling for symmetry. |
| `authenticated` | Any caller with a resolved principal (`UserID != ""` OR `AgentID != ""`). |
| `agent` | Callers authenticated as an agent (`AgentID != ""`). |
| `role:<name>` | Callers whose `Identity.Roles` contains `<name>`. Alias for `global_role:<name>`. |
| `global_role:<name>` | Same as `role:` but the diagnostic reason reflects the explicit `global_role` spelling. Kept distinct from `role` because v1.5 plans to add `org_role:` as a parallel org-scoped predicate; carving out the name now avoids a later breaking rename. |
| `permission:<resource>:<action>` | Hook for the forthcoming RBAC permission store (Phase 6.5). MVP: always denies with reason `"permission \"X:Y\" required (permission-based rules not yet implemented)"`. Fails closed — better a visible deny in dev than a silent allow in prod. |
| `scope:<name>` | Callers whose `Identity.Scopes` contains `<name>`. |
| `tier:<name>` | Callers whose `Identity.Tier == <name>`. A mismatch produces `DecisionPaywallRedirect` (not a generic 403) so the proxy can route the caller to `/paywall/...`. See `decision_kinds.md`. |

### Error conditions

- Both `require` and `allow` set → compile-time error: `"rule has both require and allow; choose one"`.
- Both empty → `"rule must set require or allow"`.
- Prefix without a value (e.g. `role:` with no name) → `"role: requires a value, e.g. role:admin"` (and equivalents for each prefix).
- Unknown prefix or top-level string → `"unknown require \"X\" (expected anonymous, authenticated, agent, role:X, global_role:X, tier:X, permission:X:Y, or scope:X)"`.

## Allow strings (accepted values)

Only `anonymous` is accepted. Anything else is a compile-time error: `"allow \"X\": only \"anonymous\" is supported"`.

The intent: `allow: anonymous` is the explicit "this is public" statement — spelled `allow` instead of `require: anonymous` so a YAML reviewer skimming for public surfaces finds them quickly (`grep "allow:" rules.yaml` is a clean filter).

## Relationship between Require and Allow

- Conceptually the two are mutually exclusive ways of phrasing the same thing for the `anonymous` case.
- `allow: anonymous` is the preferred spelling for public paths because it reads as "I mean this".
- `require: anonymous` works too and produces an identical compiled `Requirement{Kind: ReqAnonymous}`. Provided for symmetry so YAML authors can write every rule with a single `require:` key if they prefer.
- Every other predicate must use `require:`. Attempting `allow: authenticated` or `allow: role:admin` rejects with the "only anonymous is supported" error.

## Compiled form

All accepted strings compile to a `Requirement{Kind, Value}` where `Kind` is one of the `Req*` enum constants from `internal/proxy/rules.go`. The compile-time errors above are the only way a malformed `require`/`allow` can surface — once compiled, the runtime evaluator in `evaluatePrimary` matches on `Kind` via a `switch`, so future predicate kinds are added in one place.

## SDK surface

SDK clients should expose a typed union rather than a bare string when marshalling rule specs. A TypeScript sketch:

```ts
type Require =
  | { kind: "anonymous" }
  | { kind: "authenticated" }
  | { kind: "agent" }
  | { kind: "role";        value: string }
  | { kind: "global_role"; value: string }
  | { kind: "permission";  value: string }
  | { kind: "scope";       value: string }
  | { kind: "tier";        value: string };
```

Serializer collapses the union back to the wire string (`"role:admin"`, `"anonymous"`, etc.). The Python SDK follows the same shape with a `TypedDict`/`Literal` union.
