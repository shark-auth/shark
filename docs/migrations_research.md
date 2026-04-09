# Migrating users out of five major auth platforms

**SharkAuth's "effortless migration" is technically feasible for four of five platforms — Cognito is the exception.** Auth0, Clerk, Firebase, and Supabase all allow password hash export (bcrypt or Firebase scrypt), meaning users can be migrated without forced resets. AWS Cognito never exposes password hashes, creating hard vendor lock-in that requires a lazy-migration workaround. This report covers every technical detail needed to build import pipelines from each platform: export methods, hash algorithms, field schemas, rate limits, and available tooling.

---

## Auth0: bcrypt hashes available, but only via support ticket

Auth0 provides two export paths. The **bulk export job** (`POST /api/v2/jobs/users-exports`) is the primary method — it creates an async job that produces a gzipped NDJSON or CSV file downloadable from a pre-signed S3 URL. The paginated `GET /api/v2/users` endpoint has a **hard cap of 1,000 total users** across all pages, making it useless for production migrations. The Dashboard's User Import/Export extension wraps the same bulk job API but is being deprecated in September 2025.

**Export formats** are NDJSON (one JSON object per line) or CSV. NDJSON is recommended because it preserves nested objects like `user_metadata` and `app_metadata` intact, while CSV requires specifying each nested field individually and caps at **30 fields per export**. Note that Auth0 exports NDJSON but imports standard JSON arrays — conversion via `jq` is needed.

**Password hashes are not included in standard exports.** Obtaining them requires filing a support ticket at support.auth0.com, which is only available on paid plans and can take **a week or more** to process. The delivered file contains NDJSON records with a `passwordHash` field in standard **bcrypt `$2b$10$` format** — fully portable to any bcrypt-compatible system. This two-step process (API export for profile data, support ticket for hashes) means SharkAuth must merge files by matching on `user_id`.

The exportable field set covers `user_id`, `email`, `email_verified`, `username`, `phone_number`, `name`, `given_name`, `family_name`, `nickname`, `picture`, `created_at`, `updated_at`, `last_login`, `logins_count`, `blocked`, plus full `user_metadata`, `app_metadata`, and `identities` arrays. **MFA enrollment (TOTP secrets, WebAuthn keys) is likely not exportable** — Auth0's community manager has confirmed this requires re-enrollment in the target system.

Rate limits sit at **50 requests per second** for production tenants. Export jobs have no documented concurrent limit or user-count ceiling. The download URL is valid for 60 seconds per request but can be re-requested for 24 hours.

Multiple platforms offer Auth0-specific migration tooling: WorkOS (`workos/migrate-auth0-users`), FusionAuth, ZITADEL (`zitadel/zitadel-tools`), and Supabase all publish detailed guides with field mappings and bcrypt import support.

---

## Clerk: Dashboard CSV is the key to password hashes

Clerk offers two export paths that must be combined for complete data. The **Dashboard CSV export** (Settings → User Exports → "Export all users") produces a file with these columns: `id`, `first_name`, `last_name`, `username`, `primary_email_address`, `primary_phone_number`, `verified_email_addresses`, `unverified_email_addresses`, `verified_phone_numbers`, `unverified_phone_numbers`, `totp_secret`, **`password_digest`**, and **`password_hasher`**. The critical detail: **this CSV is the only way to get password hashes** — the Backend API deliberately excludes them.

The **Backend API** (`GET /v1/users`) returns richer JSON with fields the CSV lacks: `publicMetadata`, `privateMetadata`, `unsafeMetadata`, `externalAccounts` (OAuth connections), `imageUrl`, `banned`, `locked`, organization memberships, and timestamps. Pagination uses `limit` (max **500**) and `offset` parameters. A complete migration pipeline must fetch both sources and merge on `id`.

Clerk uses **bcrypt** for password hashing. The `password_digest` column contains standard bcrypt hashes, and the `password_hasher` column confirms `bcrypt`. These are directly portable to any bcrypt-compatible target system — no re-hashing needed.

Rate limits are **1,000 requests per 10 seconds** for production instances and 100 per 10 seconds for development. At max pagination (500 users/request), theoretical throughput reaches roughly **50,000 users per 10 seconds**. HTTP 429 responses include a `Retry-After` header.

There is **no dedicated bulk export API or CLI tool** from Clerk. The standard approach is paginating through the Backend API with `limit=500` and incrementing `offset`. Clerk's open-source migration script (`clerk/migration-script`) is import-only (for migrating *to* Clerk). Third-party migration guides exist from WorkOS (`workos/migrate-clerk-users`), PropelAuth, and Better Auth, all of which document the CSV + API merge pattern.

---

## Firebase Auth: modified scrypt with exportable hash config

Firebase provides the most complete hash export of any platform examined, but the custom algorithm adds complexity. The **CLI command** `firebase auth:export users.json --format=json` dumps all users including password hashes and salts. The **Admin SDK's `listUsers()`** returns `ExportedUserRecord` objects with `passwordHash` and `passwordSalt` fields, paginated in batches of up to **1,000 users** with no total limit.

**Firebase uses a modified scrypt variant** — not standard scrypt. The algorithm concatenates the per-user salt with a project-level **salt separator**, runs scrypt with configurable `rounds` (1–8) and `mem_cost` (1–14, as a power of 2), then **encrypts the result with AES-256-CTR using a project-level signer key**. This means verifying a password requires five parameters beyond the hash itself:

| Parameter | Source | Typical value |
|-----------|--------|---------------|
| `base64_signer_key` | Firebase Console → Auth → Users → ⋮ → "Password Hash Parameters" | Project-unique, ~64 bytes base64 |
| `base64_salt_separator` | Same location | Commonly `Bw==` (0x07) |
| `rounds` | Same location | Typically `8` |
| `mem_cost` | Same location | Typically `14` |
| `salt` | Per-user, in export data | Base64-encoded |

**An important IAM requirement**: the Admin SDK only returns hashes if the service account has the `firebaseauth.configs.getHashConfig` permission, which is **not granted by any predefined role** — a custom IAM role must be created.

The exported JSON uses Google Identity Toolkit field names: `localId` (not `uid`), `passwordHash`, `salt`, `email`, `emailVerified`, `displayName`, `photoUrl`, `phoneNumber`, `disabled`, `createdAt`, `lastSignedInAt`, `customAttributes` (JSON string of custom claims), and `providerUserInfo` (linked OAuth accounts). SharkAuth must map these to its internal schema.

Open-source Firebase scrypt implementations exist in **Node.js** (`firebase-scrypt` npm), **Python** (`firebase-scrypt` PyPI), **Go** (`github.com/Aoang/firebase-scrypt`), and **Rust** (`firebase-scrypt` crate). All accept the hash config parameters and provide `verify()` functions. The recommended migration strategy is to verify against Firebase scrypt on first login, then re-hash with SharkAuth's native algorithm.

---

## Supabase Auth: the easiest migration source by far

Supabase is the most migration-friendly platform because it gives **direct SQL access to the `auth.users` table** — no API wrappers, no support tickets, no special permissions. A simple `SELECT * FROM auth.users` exports everything, and the Dashboard SQL Editor can output results as CSV.

The `auth.users` table stores passwords in the `encrypted_password` column as **standard bcrypt `$2a$10$` hashes**, generated by Go's `golang.org/x/crypto/bcrypt` library at **cost factor 10**. These are **fully portable** with zero modification. A real-world migration of 125,000 users from Auth0 to Supabase confirmed bcrypt hash interoperability, and the reverse path works identically.

Key columns in `auth.users` include `id` (UUID), `email`, `encrypted_password`, `email_confirmed_at`, `phone`, `phone_confirmed_at`, `raw_user_meta_data` (JSONB with user-editable profile data), `raw_app_meta_data` (JSONB with provider info like `{"provider": "email", "providers": ["email"]}`), `created_at`, `updated_at`, `last_sign_in_at`, `banned_until`, `is_sso_user`, `is_anonymous`, and `deleted_at`. The companion `auth.identities` table stores linked OAuth providers with `provider`, `provider_id`, `identity_data` (JSONB with OAuth profile), and `user_id` foreign key. **Both tables must be exported** for a complete migration.

MFA data lives in `auth.mfa_factors` with columns for `factor_type` ("totp" or "phone"), `status`, and an encrypted `secret`. Sessions and refresh tokens do not need migration — users simply re-authenticate.

**There are no rate limits on SQL access** — it's your Postgres database. Export speed depends only on plan-tier compute resources. The Admin API (`auth.admin.listUsers()`) exists but defaults to 50 users per page and **does not return password hashes**, making it unsuitable for migration. Always use direct SQL.

Supabase has no official "migrating away" guide, but their data-portability stance is implicit: full Postgres access means full control. Better Auth publishes the most complete third-party migration guide with a batch-processing TypeScript script that preserves bcrypt hashes.

---

## AWS Cognito: no password hashes, ever

Cognito is the only platform that creates genuine migration friction. **AWS never exports password hashes** — this is a stated security policy, not a missing feature. MFA TOTP secrets are similarly locked. Every migration from Cognito must contend with this reality.

The **ListUsers API** is the sole export method. It returns JSON with `Username`, `Attributes` (array of name-value pairs for standard OIDC claims plus `custom:*` attributes), `UserCreateDate`, `UserLastModifiedDate`, `Enabled`, and `UserStatus`. Pagination uses `PaginationToken` (which **expires after 1 hour**) with a **hard maximum of 60 users per page**. There is no bulk export button in the Console.

Rate limits for ListUsers sit at roughly **5 requests per second per user pool**, yielding ~300 users/second or ~18,000 users/minute. The AWS CLI's `list-users` command does not auto-paginate (a known bug), requiring manual token handling. For large pools, the AWS-published **Cognito User Profiles Export Reference Architecture** uses Step Functions + Lambda to export to DynamoDB but explicitly cannot export passwords or handle MFA-enabled pools.

**The two viable workarounds for passwordless migration are:**

- **User Migration Lambda Trigger** (recommended): Configure a Lambda on the destination system that fires when an unknown user attempts sign-in. The Lambda receives the plaintext password, authenticates against the old Cognito pool (using `USER_PASSWORD_AUTH` flow, not SRP), and returns user attributes. Cognito then creates the user with their verified password. This "lazy migration" transparently moves active users over time but requires **maintaining the old Cognito pool indefinitely** until coverage is sufficient. The Lambda has a hard **5-second timeout**.

- **Forced password reset**: Bulk-import user attributes via CSV (`CreateUserImportJob`, max **500,000 users per file**, **100 MB limit**, only one import job active at a time), then require all users to reset passwords. Users arrive in `RESET_REQUIRED` status. This is cleaner but degrades user experience.

For SharkAuth's Cognito import, the practical approach is to export all user attributes via ListUsers, create accounts without passwords, then offer either a lazy-migration proxy (where SharkAuth authenticates against the old Cognito pool on first login) or a bulk password-reset flow.

Third-party tools like `cognito-backup` (npm), `cognito-csv-exporter` (Python), and `cognito-backup-restore` handle the ListUsers pagination and rate limiting, but none solve the password problem.

---

## Platform comparison at a glance

| Dimension | Auth0 | Clerk | Firebase | Supabase | Cognito |
|-----------|-------|-------|----------|----------|---------|
| **Export method** | Bulk job API | Dashboard CSV + API | CLI + Admin SDK | Direct SQL | ListUsers API |
| **Formats** | NDJSON, CSV | CSV, JSON | JSON, CSV | SQL/CSV/pg_dump | JSON (API) |
| **Hash algorithm** | bcrypt `$2b$10$` | bcrypt | Modified scrypt | bcrypt `$2a$10$` | N/A |
| **Hash exportable?** | Via support ticket (paid) | Yes (Dashboard CSV) | Yes (with IAM role) | Yes (SQL) | **Never** |
| **Hash portable?** | Direct bcrypt import | Direct bcrypt import | Needs Firebase scrypt lib | Direct bcrypt import | N/A |
| **Max per page** | 1,000 (export job unlimited) | 500 | 1,000 | Unlimited (SQL) | 60 |
| **Rate limit** | 50 rps | 100 req/10s (prod) | No documented limit | None (SQL) | ~5 rps |
| **MFA exportable?** | Unlikely | TOTP secret in CSV | `mfaInfo` in export | Encrypted in DB | **Never** |
| **Migration difficulty** | Medium (support ticket delay) | Low | Medium (custom scrypt) | **Lowest** | **Highest** |

---

## Conclusion: a practical architecture for SharkAuth's import system

The research reveals a clear hierarchy of migration difficulty. **Supabase is trivial** — standard bcrypt hashes from a SQL query, no rate limits, no gatekeeping. **Clerk is straightforward** — bcrypt from a self-service CSV download, supplemented by API calls for metadata. **Auth0 requires planning** — the week-long support ticket for hashes on paid plans is the bottleneck, not technical complexity. **Firebase is technically complex but fully solvable** — SharkAuth needs a Firebase scrypt verification library (available in every major language) to validate passwords at first login, then re-hash to its native algorithm. **Cognito is the hard case** — without password hashes, SharkAuth must either proxy authentication to the old Cognito pool via a lazy-migration flow or accept forced password resets.

For SharkAuth's import pipeline, the recommended architecture is a per-platform adapter that accepts each platform's native export format, normalizes user records to a common schema, and handles password verification differently per source: direct bcrypt import for Auth0/Clerk/Supabase, Firebase scrypt verification-then-rehash for Firebase, and a migration-proxy mode for Cognito. Building the Firebase scrypt verifier is the single largest technical investment, but mature open-source libraries in Node.js, Python, Go, and Rust eliminate the need to implement the algorithm from scratch.