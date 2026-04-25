# users.ts

**Path:** `sdk/typescript/src/users.ts`
**Type:** Admin namespace — user tier + read/list
**LOC:** 148

## Purpose
Admin operations on user records: get the per-user billing tier, set it, list/filter users. Used by the dashboard's billing tier management.

## Public API
- `class UsersClient`
  - `constructor(opts: UsersClientOptions)`
  - `setUserTier(userId, tier): Promise<SetUserTierResult>` — PATCH `/api/v1/admin/users/{id}/tier`
  - `getUser(userId): Promise<User>` — GET `/api/v1/users/{id}` (handles wrapped or unwrapped response)
  - `createUser(input: CreateUserInput): Promise<User>` — POST `/api/v1/admin/users`; admin-only; password optional
  - `updateUser(userId, input: UpdateUserInput): Promise<User>` — PATCH `/api/v1/users/{id}`; partial update
  - `deleteUser(userId): Promise<void>` — DELETE `/api/v1/users/{id}`; 204 on success
  - `listUsers(opts?: ListUsersOptions): Promise<UserListResult>` — GET `/api/v1/users[?email&limit&offset]`

## Types
- `UserTier = "free" | "pro"` — server-enforced enum
- `User`: `id, email, name?, metadata?, created_at, updated_at` + open index signature
- `SetUserTierResult = { user: User, tier: string }`
- `CreateUserInput = { email, password?, name?, email_verified? }`
- `UpdateUserInput = { email?, name?, email_verified?, metadata? }` — all optional
- `ListUsersOptions = { email?, limit?, offset? }`
- `UserListResult = { data: User[], total: number }`

## Constructor options
- `baseUrl: string`
- `adminKey: string` — Bearer token

## Tier semantics
- New tier is baked into the **next** access token on refresh — existing tokens keep the old tier until expiry.
- Tier lives in `metadata.tier` server-side.

## Error mapping
- Non-success → `SharkAPIError` parsed from `{error:{code,message}}`.

## Internal dependencies
- `http.ts`, `errors.ts`

## Notes
- `getUser` tolerates both `{ data: User }` envelope and bare `User` payload.
- `email` filter behavior (exact vs partial) is server-defined — SDK passes through.
