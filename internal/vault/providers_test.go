package vault_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/vault"
)

// TestTemplates_Nonempty_AndHaveUniqueNames guards the invariants every
// other test in this file relies on: the registry is populated, no template
// has a blank key, and there are no duplicate Name values between the map
// key and the struct field.
func TestTemplates_Nonempty_AndHaveUniqueNames(t *testing.T) {
	templates := vault.Templates()
	if len(templates) == 0 {
		t.Fatal("expected at least one built-in template, got empty registry")
	}

	// Spec says we ship Google (calendar/drive/gmail), Slack, GitHub,
	// Microsoft, Notion, Linear, Jira = 9 templates minimum.
	const wantMinimum = 9
	if len(templates) < wantMinimum {
		t.Errorf("expected at least %d templates, got %d", wantMinimum, len(templates))
	}

	seen := make(map[string]bool, len(templates))
	for key, tpl := range templates {
		if key == "" {
			t.Error("registry contains an empty key")
		}
		if tpl == nil {
			t.Errorf("template %q is nil", key)
			continue
		}
		if tpl.Name == "" {
			t.Errorf("template at key %q has empty Name", key)
		}
		if tpl.Name != key {
			t.Errorf("template key %q does not match Name %q", key, tpl.Name)
		}
		if tpl.DisplayName == "" {
			t.Errorf("template %q has empty DisplayName", key)
		}
		if tpl.AuthURL == "" {
			t.Errorf("template %q has empty AuthURL", key)
		}
		if tpl.TokenURL == "" {
			t.Errorf("template %q has empty TokenURL", key)
		}
		if !strings.HasPrefix(tpl.AuthURL, "https://") {
			t.Errorf("template %q AuthURL must be https://, got %q", key, tpl.AuthURL)
		}
		if !strings.HasPrefix(tpl.TokenURL, "https://") {
			t.Errorf("template %q TokenURL must be https://, got %q", key, tpl.TokenURL)
		}
		if tpl.DefaultScopes == nil {
			t.Errorf("template %q DefaultScopes must be non-nil (use empty slice)", key)
		}
		if seen[tpl.Name] {
			t.Errorf("duplicate template Name %q", tpl.Name)
		}
		seen[tpl.Name] = true
	}

	// Spot-check that the expected catalog members exist — prevents a
	// silent deletion slipping past review.
	expected := []string{
		"google_calendar", "google_drive", "google_gmail",
		"slack", "github", "microsoft", "notion", "linear", "jira",
	}
	for _, name := range expected {
		if _, ok := templates[name]; !ok {
			t.Errorf("expected template %q to be registered", name)
		}
	}
}

// TestTemplate_KnownAndUnknown exercises the lookup API: a known name
// returns (non-nil, true) and an unknown name returns (nil, false).
func TestTemplate_KnownAndUnknown(t *testing.T) {
	tpl, ok := vault.Template("github")
	if !ok {
		t.Fatal("expected GitHub template to be found")
	}
	if tpl == nil {
		t.Fatal("expected non-nil GitHub template")
	}
	if tpl.Name != "github" {
		t.Errorf("expected Name=github, got %q", tpl.Name)
	}
	if tpl.DisplayName != "GitHub" {
		t.Errorf("expected DisplayName=GitHub, got %q", tpl.DisplayName)
	}

	missing, ok := vault.Template("definitely-not-a-provider")
	if ok {
		t.Error("expected ok=false for unknown template")
	}
	if missing != nil {
		t.Error("expected nil template for unknown name")
	}
}

// TestListTemplates_SortedByDisplayName verifies that ListTemplates returns
// the catalog in alphabetical order by DisplayName — the admin UI relies on
// this ordering when rendering the picker.
func TestListTemplates_SortedByDisplayName(t *testing.T) {
	list := vault.ListTemplates()
	if len(list) == 0 {
		t.Fatal("expected non-empty template list")
	}

	displayNames := make([]string, len(list))
	for i, tpl := range list {
		displayNames[i] = tpl.DisplayName
	}

	if !sort.StringsAreSorted(displayNames) {
		t.Errorf("expected ListTemplates sorted by DisplayName, got order: %v", displayNames)
	}

	// Same cardinality as the map registry — nothing lost in translation.
	if want := len(vault.Templates()); len(list) != want {
		t.Errorf("ListTemplates size mismatch: got %d, want %d", len(list), want)
	}
}

// TestApplyTemplate_BuildsVaultProvider_WithDefaultsAndOverrides exercises
// the core ApplyTemplate contract: defaults flow through, overrides win,
// and the encrypted-secret slot stays empty (Manager.CreateProvider owns
// the crypto boundary — a template has no business pre-populating it).
func TestApplyTemplate_BuildsVaultProvider_WithDefaultsAndOverrides(t *testing.T) {
	tpl, ok := vault.Template("slack")
	if !ok {
		t.Fatal("slack template missing")
	}

	t.Run("defaults", func(t *testing.T) {
		got := vault.ApplyTemplate(tpl, "client-abc", "", nil)
		if got == nil {
			t.Fatal("ApplyTemplate returned nil")
		}
		if got.Name != "slack" {
			t.Errorf("Name: got %q, want slack", got.Name)
		}
		if got.DisplayName != "Slack" {
			t.Errorf("DisplayName: got %q, want Slack (template default)", got.DisplayName)
		}
		if got.ClientID != "client-abc" {
			t.Errorf("ClientID: got %q, want client-abc", got.ClientID)
		}
		if got.ClientSecretEnc != "" {
			t.Errorf("ClientSecretEnc must be empty (Manager encrypts), got %q", got.ClientSecretEnc)
		}
		if got.AuthURL != tpl.AuthURL {
			t.Errorf("AuthURL: got %q, want %q", got.AuthURL, tpl.AuthURL)
		}
		if got.TokenURL != tpl.TokenURL {
			t.Errorf("TokenURL: got %q, want %q", got.TokenURL, tpl.TokenURL)
		}
		if len(got.Scopes) != len(tpl.DefaultScopes) {
			t.Fatalf("Scopes length: got %d, want %d (defaults)", len(got.Scopes), len(tpl.DefaultScopes))
		}
		for i, s := range tpl.DefaultScopes {
			if got.Scopes[i] != s {
				t.Errorf("Scopes[%d]: got %q, want %q", i, got.Scopes[i], s)
			}
		}
		if !got.Active {
			t.Error("expected Active=true on freshly-applied template")
		}
	})

	t.Run("overrides_take_effect", func(t *testing.T) {
		customScopes := []string{"chat:write"}
		got := vault.ApplyTemplate(tpl, "client-xyz", "Team Workspace", customScopes)
		if got == nil {
			t.Fatal("ApplyTemplate returned nil")
		}
		if got.DisplayName != "Team Workspace" {
			t.Errorf("DisplayName override failed: got %q", got.DisplayName)
		}
		if len(got.Scopes) != 1 || got.Scopes[0] != "chat:write" {
			t.Errorf("Scopes override failed: got %v", got.Scopes)
		}
		// Name is NOT overridable — it's the stable key for the provider.
		if got.Name != "slack" {
			t.Errorf("Name must remain slack (not overridable), got %q", got.Name)
		}
	})

	t.Run("empty_scopes_slice_uses_defaults", func(t *testing.T) {
		// An empty-but-non-nil slice should still fall through to defaults;
		// we treat nil and []string{} the same — caller expressed "no
		// custom scopes" in both cases. This matches the CreateProvider
		// semantics in vault.go which normalises nil to empty.
		got := vault.ApplyTemplate(tpl, "client-xyz", "", []string{})
		if len(got.Scopes) != len(tpl.DefaultScopes) {
			t.Errorf("empty scopes slice should fall back to defaults, got %v", got.Scopes)
		}
	})

	t.Run("nil_template_returns_nil", func(t *testing.T) {
		if got := vault.ApplyTemplate(nil, "cid", "", nil); got != nil {
			t.Error("expected nil when template is nil")
		}
	})

	t.Run("scopes_defensive_copy", func(t *testing.T) {
		// Mutating the returned Scopes slice must not corrupt the
		// template's DefaultScopes — templates are shared state.
		originalFirst := tpl.DefaultScopes[0]
		got := vault.ApplyTemplate(tpl, "client-abc", "", nil)
		got.Scopes[0] = "mutated"
		if tpl.DefaultScopes[0] != originalFirst {
			t.Errorf("template DefaultScopes was mutated via returned Scopes slice: got %q, want %q",
				tpl.DefaultScopes[0], originalFirst)
		}
	})
}

// TestApplyTemplate_PreservesNameAndURLs sanity-checks every template in
// the catalog can be applied without dropping any URL-ish field, so adding
// a new template can't silently land with a half-wired mapping.
func TestApplyTemplate_PreservesNameAndURLs(t *testing.T) {
	for _, tpl := range vault.ListTemplates() {
		tpl := tpl
		t.Run(tpl.Name, func(t *testing.T) {
			got := vault.ApplyTemplate(tpl, "cid", "", nil)
			if got == nil {
				t.Fatal("ApplyTemplate returned nil")
			}
			if got.Name != tpl.Name {
				t.Errorf("Name: got %q, want %q", got.Name, tpl.Name)
			}
			if got.AuthURL != tpl.AuthURL {
				t.Errorf("AuthURL: got %q, want %q", got.AuthURL, tpl.AuthURL)
			}
			if got.TokenURL != tpl.TokenURL {
				t.Errorf("TokenURL: got %q, want %q", got.TokenURL, tpl.TokenURL)
			}
			if got.IconURL != tpl.IconURL {
				t.Errorf("IconURL: got %q, want %q", got.IconURL, tpl.IconURL)
			}
			if got.DisplayName != tpl.DisplayName {
				t.Errorf("DisplayName: got %q, want %q", got.DisplayName, tpl.DisplayName)
			}
		})
	}
}
