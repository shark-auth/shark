# conditions.go

**Path:** `internal/authflow/conditions.go`  
**Package:** `authflow`  
**LOC:** 227  
**Tests:** `conditions_test.go`

## Purpose
Condition DSL evaluator. Simple map-based predicate matching for flow selection (email_domain, has_metadata, trigger_eq, user_has_role, all_of, any_of, not).

## Key types / functions
- `Evaluate(conditions map[string]any, fc *Context)` (func, line 26) — walks predicate DSL, returns bool + error
- `evaluateOne(key, raw, fc)` (func, line 51) — evaluates single condition predicate
- `ErrUnknownPredicate` — returned when condition map contains invalid key

## Supported predicates
- `email_domain` (string) — user.Email ends with "@" + value (case-insensitive)
- `has_metadata` (string) — key present in fc.Metadata OR user.Metadata
- `metadata_eq` (map[string]any) — exactly 1 k/v; metadata[k] == v
- `trigger_eq` (string) — fc.Trigger == value (signup, login, password_reset, magic_link, oauth_callback)
- `user_has_role` (string) — value in fc.UserRoles
- `all_of` ([]map) — AND of nested predicates
- `any_of` ([]map) — OR of nested predicates
- `not` (map) — negate nested predicate

## Imports of note
- `encoding/json`, `strings` — predicate parsing

## Wired by
- Engine.Execute() calls Evaluate() before step execution
- Admin flow builder uses Evaluate() to test condition matching

## Notes
- Empty condition map matches everything (no conditions = match all users)
- Implicit AND across map entries; each key must satisfy
- Type errors (e.g., email_domain expects string) return ErrUnknownPredicate
- Metadata lookup: checks fc.Metadata first, then unmarshals user.Metadata JSON
- User-level metadata: stored as JSON blob; key lookups unmarshaled on-the-fly

