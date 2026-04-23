package cmd

import (
	"log/slog"
	"strings"
	"testing"
)

// TestValidateAuthURL covers the W15c HTTPS-for-auth guard. The
// standalone proxy trusts /.well-known/jwks.json to bootstrap every
// bearer-token verification; letting --auth point at a cleartext URL
// without an explicit opt-in is a MITM footgun — an attacker on the
// path can swap signing keys and forge tokens. Each sub-test pins one
// (url, insecure-flag) combination to its required outcome.
func TestValidateAuthURL(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		insecure   bool
		wantErr    bool
		wantSubstr string
	}{
		{"https ok", "https://auth.example", false, false, ""},
		{"https ok with insecure flag (ignored)", "https://auth.example", true, false, ""},
		{"http rejected without flag", "http://auth.example", false, true, "insecure-auth-http"},
		{"http allowed with explicit opt-in", "http://auth.example", true, false, ""},
		{"bare hostname rejected", "auth.example", false, true, "must be an http:// or https://"},
		{"ftp rejected", "ftp://auth.example", false, true, "must be an http:// or https://"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthURL(tc.url, tc.insecure, slog.Default())
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
