package authflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Evaluate walks a simple map-based predicate DSL against a Context and
// returns whether the predicate matches.
//
// Supported keys (one per map entry — unknown keys error):
//
//   - email_domain    string        user.Email ends with "@" + value
//   - has_metadata    string        key present in fc.Metadata OR user.Metadata
//   - metadata_eq     map[string]…  single k/v; metadata[k] == v
//   - trigger_eq      string        fc.Trigger == value
//   - user_has_role   string        value in fc.UserRoles
//   - all_of          []map         AND of nested predicates
//   - any_of          []map         OR of nested predicates
//   - not             map           negate nested predicate
//
// An empty map matches everything — that's what lets "no conditions"
// behave the same as "match all users" in the storage layer.
func Evaluate(conditions map[string]any, fc *Context) (bool, error) {
	if len(conditions) == 0 {
		return true, nil
	}

	// Every entry must hold; it's an implicit AND across the map. Keeps the
	// DSL flat for simple flows while all_of remains available for explicit
	// nesting.
	for key, raw := range conditions {
		ok, err := evaluateOne(key, raw, fc)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// ErrUnknownPredicate is returned when a condition map contains a key that
// isn't part of the DSL. Surfaces as "skipping flow with invalid conditions"
// in engine logs rather than silently matching.
var ErrUnknownPredicate = errors.New("unknown condition predicate")

func evaluateOne(key string, raw any, fc *Context) (bool, error) {
	switch key {
	case "email_domain":
		domain, ok := raw.(string)
		if !ok {
			return false, fmt.Errorf("email_domain: want string, got %T", raw)
		}
		if fc.User == nil {
			return false, nil
		}
		return strings.HasSuffix(strings.ToLower(fc.User.Email), "@"+strings.ToLower(domain)), nil

	case "has_metadata":
		k, ok := raw.(string)
		if !ok {
			return false, fmt.Errorf("has_metadata: want string, got %T", raw)
		}
		if fc.Metadata != nil {
			if _, found := fc.Metadata[k]; found {
				return true, nil
			}
		}
		// Also check user-level metadata (stored as JSON blob).
		return userMetadataHasKey(fc, k), nil

	case "metadata_eq":
		m, ok := raw.(map[string]any)
		if !ok {
			return false, fmt.Errorf("metadata_eq: want object, got %T", raw)
		}
		if len(m) != 1 {
			return false, fmt.Errorf("metadata_eq: expected exactly 1 k/v, got %d", len(m))
		}
		for k, v := range m {
			if fc.Metadata != nil {
				if got, found := fc.Metadata[k]; found {
					return anyEq(got, v), nil
				}
			}
			if got, found := userMetadataValue(fc, k); found {
				return anyEq(got, v), nil
			}
		}
		return false, nil

	case "trigger_eq":
		t, ok := raw.(string)
		if !ok {
			return false, fmt.Errorf("trigger_eq: want string, got %T", raw)
		}
		return fc.Trigger == t, nil

	case "user_has_role":
		role, ok := raw.(string)
		if !ok {
			return false, fmt.Errorf("user_has_role: want string, got %T", raw)
		}
		for _, r := range fc.UserRoles {
			if r == role {
				return true, nil
			}
		}
		return false, nil

	case "all_of":
		list, ok := raw.([]any)
		if !ok {
			return false, fmt.Errorf("all_of: want array, got %T", raw)
		}
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return false, fmt.Errorf("all_of: item must be object, got %T", item)
			}
			ok2, err := Evaluate(m, fc)
			if err != nil {
				return false, err
			}
			if !ok2 {
				return false, nil
			}
		}
		return true, nil

	case "any_of":
		list, ok := raw.([]any)
		if !ok {
			return false, fmt.Errorf("any_of: want array, got %T", raw)
		}
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				return false, fmt.Errorf("any_of: item must be object, got %T", item)
			}
			ok2, err := Evaluate(m, fc)
			if err != nil {
				return false, err
			}
			if ok2 {
				return true, nil
			}
		}
		return false, nil

	case "not":
		m, ok := raw.(map[string]any)
		if !ok {
			return false, fmt.Errorf("not: want object, got %T", raw)
		}
		ok2, err := Evaluate(m, fc)
		if err != nil {
			return false, err
		}
		return !ok2, nil

	default:
		return false, fmt.Errorf("%w: %q", ErrUnknownPredicate, key)
	}
}

// userMetadataHasKey decodes the user's Metadata JSON blob and reports
// whether the given key is set. Empty blob / invalid JSON return false so
// conditions never error on a malformed user row.
func userMetadataHasKey(fc *Context, key string) bool {
	if fc.User == nil || fc.User.Metadata == "" {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(fc.User.Metadata), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// userMetadataValue extracts a single key from User.Metadata.
func userMetadataValue(fc *Context, key string) (any, bool) {
	if fc.User == nil || fc.User.Metadata == "" {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(fc.User.Metadata), &m); err != nil {
		return nil, false
	}
	v, ok := m[key]
	return v, ok
}

// anyEq compares two any values with JSON-style equality. JSON numbers
// deserialize as float64; string metadata often compares against raw
// strings; for complex values we fall back to %v string form so maps /
// slices compare predictably in tests.
func anyEq(a, b any) bool {
	if a == b {
		return true
	}
	af, aok := toFloat(a)
	bf, bok := toFloat(b)
	if aok && bok {
		return af == bf
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	}
	return 0, false
}
