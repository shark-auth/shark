// Package email — redirect URL resolution for email send paths.
//
// GetRedirectURL reads the configured redirect URL for a given email flow
// kind ("verify", "reset", "magic_link") from the system_config blob.
//
// Fallback behaviour (when URL is not configured):
//   - Still uses the built-in shark endpoint so existing flows keep working.
//   - Emits a log.Printf warning so devs know to configure their URL.
//
// Token appending: callers are responsible for appending ?token=<raw> (or
// &token=<raw> if the URL already contains a query string). AppendToken is
// provided as a convenience.
package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// RedirectStore is the minimal interface required by GetRedirectURL. It is
// satisfied by *storage.SQLiteStore (and any other Store implementation).
type RedirectStore interface {
	GetSystemConfig(ctx context.Context) (string, error)
}

// emailRedirectConfig mirrors api.emailConfig without creating a circular dep.
type emailRedirectConfig struct {
	VerifyRedirectURL    string `json:"verify_redirect_url"`
	ResetRedirectURL     string `json:"reset_redirect_url"`
	MagicLinkRedirectURL string `json:"magic_link_redirect_url"`
	InviteRedirectURL    string `json:"invite_redirect_url"`
}

// GetRedirectURL returns the configured redirect base URL for the given kind.
//
// kind must be one of: "verify", "reset", "magic_link".
//
// If the redirect URL is not configured (empty), fallbackURL is returned
// unchanged and a warning is logged so developers know to configure it via
// Admin > Branding > Email > Redirect URLs.
func GetRedirectURL(ctx context.Context, store RedirectStore, kind string, fallbackURL string) (string, bool, error) {
	raw, err := store.GetSystemConfig(ctx)
	if err != nil {
		// Non-fatal: fall back to the built-in endpoint and warn.
		log.Printf("[shark/email] warn: could not read system_config for redirect URL (%s): %v — using built-in fallback", kind, err)
		return fallbackURL, false, nil
	}

	if raw == "" || raw == "{}" {
		logUnconfigured(kind)
		return fallbackURL, false, nil
	}

	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &wrapper); err != nil {
		log.Printf("[shark/email] warn: system_config parse error for redirect URL (%s): %v — using built-in fallback", kind, err)
		return fallbackURL, false, nil
	}

	emailRaw, ok := wrapper["email_config"]
	if !ok {
		logUnconfigured(kind)
		return fallbackURL, false, nil
	}

	var cfg emailRedirectConfig
	if err := json.Unmarshal(emailRaw, &cfg); err != nil {
		log.Printf("[shark/email] warn: email_config parse error for redirect URL (%s): %v — using built-in fallback", kind, err)
		return fallbackURL, false, nil
	}

	var configured string
	switch kind {
	case "verify":
		configured = cfg.VerifyRedirectURL
	case "reset":
		configured = cfg.ResetRedirectURL
	case "magic_link":
		configured = cfg.MagicLinkRedirectURL
	case "invite":
		configured = cfg.InviteRedirectURL
	default:
		return fallbackURL, false, fmt.Errorf("email/redirects: unknown kind %q", kind)
	}

	if configured == "" {
		logUnconfigured(kind)
		return fallbackURL, false, nil
	}

	return configured, true, nil
}

// AppendToken appends ?token=<token> (or &token=<token> when the URL already
// contains a query string) to baseURL.
func AppendToken(baseURL, token string) string {
	if strings.Contains(baseURL, "?") {
		return baseURL + "&token=" + token
	}
	return baseURL + "?token=" + token
}

func logUnconfigured(kind string) {
	log.Printf("[shark/email] warn: redirect URL for %q not configured — using built-in shark endpoint as fallback. "+
		"Configure it via Admin > Branding > Email > Redirect URLs.", kind)
}
