package api

import (
	"fmt"
	"strings"
)

// Note: slugRE is declared in organization_handlers.go and shared within the
// api package. It matches ^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$ (3–64 chars,
// lowercase alnum + internal hyphens, first/last char must be alnum).

// generateSlug derives a URL-safe slug from a human-readable application name.
// The algorithm:
//  1. Lowercase the input.
//  2. Replace spaces and underscores with hyphens.
//  3. Strip every character that is not [a-z0-9-].
//  4. Trim leading/trailing hyphens and collapse consecutive hyphens.
//  5. If the result is shorter than 3 characters, prefix with "app-".
func generateSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	// Replace word-separators with hyphens.
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Keep only [a-z0-9-].
	var b strings.Builder
	b.Grow(len(s))
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	result := b.String()

	// Collapse consecutive hyphens into one.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing hyphens.
	result = strings.Trim(result, "-")

	// If the result is too short, prefix with "app-".
	if len(result) < 3 {
		result = "app-" + result
	}

	// Enforce maximum length of 64 characters.
	if len(result) > 64 {
		result = strings.TrimRight(result[:64], "-")
	}

	return result
}

// validateSlug checks that a caller-supplied slug conforms to the allowed format:
// 3–64 characters, lowercase alphanumeric + internal hyphens,
// first and last character must be alphanumeric (not a hyphen).
func validateSlug(s string) error {
	if !slugRE.MatchString(s) {
		return fmt.Errorf("slug must be 3–64 lowercase alphanumeric characters or hyphens, and must not start or end with a hyphen")
	}
	return nil
}
