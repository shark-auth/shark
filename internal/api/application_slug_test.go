package api

import (
	"strings"
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"basic two words", "My App", "my-app"},
		{"underscore separator", "My_App", "my-app"},
		{"no separator", "MyApp", "myapp"},
		{"single char — too short, prefixed", "a", "app-a"},
		{"leading and trailing spaces with double space inside", "  My  App  ", "my-app"},
		{"non-alnum stripped", "My!!!App", "myapp"},
		{"empty after strip — all punctuation", "!!!###", "app-"},
		// Unicode accented chars are stripped; ASCII letters remain.
		// "Ünïcödé App" → lowercase "ünïcödé app" → strip non-[a-z0-9-] → "ncd app"
		// → replace space → "ncd-app"
		{"unicode letters stripped", "Ünïcödé App", "ncd-app"},
		{"numbers preserved", "App 2", "app-2"},
		{"already valid slug", "my-app-42", "my-app-42"},
		{"two chars after strip", "ab", "app-ab"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := generateSlug(tc.input)
			if got != tc.want {
				t.Errorf("generateSlug(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGenerateSlug_MaxLength(t *testing.T) {
	// A 65-character name should be truncated to ≤64.
	long := strings.Repeat("a", 65)
	got := generateSlug(long)
	if len(got) > 64 {
		t.Errorf("generateSlug long name: length %d > 64", len(got))
	}
}

func TestValidateSlug(t *testing.T) {
	valid := []struct {
		name string
		slug string
	}{
		{"simple", "my-app"},
		{"alphanumeric only", "myapp"},
		{"minimum length 3 same chars", "abc"},
		{"with numbers", "app-42"},
		{"boundary valid 64", "a" + strings.Repeat("b", 62) + "z"},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			if err := validateSlug(tc.slug); err != nil {
				t.Errorf("validateSlug(%q) unexpected error: %v", tc.slug, err)
			}
		})
	}

	invalid := []struct {
		name string
		slug string
	}{
		{"too short — 2 chars", "ab"},
		{"leading hyphen", "-my-app"},
		{"trailing hyphen", "my-app-"},
		{"uppercase", "My-App"},
		{"single char", "a"},
		{"empty string", ""},
		{"spaces", "my app"},
		{"boundary invalid 66", "a" + strings.Repeat("b", 64) + "z"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			if err := validateSlug(tc.slug); err == nil {
				t.Errorf("validateSlug(%q) expected error, got nil", tc.slug)
			}
		})
	}
}
