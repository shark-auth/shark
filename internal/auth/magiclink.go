package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/email"
	"github.com/shark-auth/shark/internal/storage"
)

var (
	ErrMagicLinkExpired  = errors.New("magic link has expired")
	ErrMagicLinkUsed     = errors.New("magic link has already been used")
	ErrMagicLinkNotFound = errors.New("magic link not found")
)

// MagicLinkManager handles magic link token generation, email sending, and verification.
type MagicLinkManager struct {
	store    storage.Store
	email    email.Sender
	sessions *SessionManager
	cfg      *config.Config
}

// NewMagicLinkManager creates a new MagicLinkManager.
func NewMagicLinkManager(store storage.Store, emailSender email.Sender, sessions *SessionManager, cfg *config.Config) *MagicLinkManager {
	return &MagicLinkManager{
		store:    store,
		email:    emailSender,
		sessions: sessions,
		cfg:      cfg,
	}
}

// Sender exposes the email sender so other managers (e.g. org invitations)
// can reuse the wired provider instead of taking it as a second dep.
func (m *MagicLinkManager) Sender() email.Sender {
	return m.email
}

// SetSender updates the email sender used by the manager.
func (m *MagicLinkManager) SetSender(s email.Sender) {
	m.email = s
}

// SendMagicLink generates a random token, stores its SHA-256 hash, and sends a magic link email.
// Always returns nil to avoid leaking whether the email address exists.
func (m *MagicLinkManager) SendMagicLink(ctx context.Context, emailAddr string) error {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generating random token: %w", err)
	}

	// Encode as base64url (no padding for URL safety)
	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)

	// SHA-256 hash for storage
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	// Compute expiry
	lifetime := m.cfg.MagicLink.TokenLifetimeDuration()
	now := time.Now().UTC()

	// Generate token ID
	id, _ := gonanoid.New()

	// Store the hashed token
	token := &storage.MagicLinkToken{
		ID:        "mlt_" + id,
		Email:     emailAddr,
		TokenHash: tokenHash,
		Used:      false,
		ExpiresAt: now.Add(lifetime).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}

	if err := m.store.CreateMagicLinkToken(ctx, token); err != nil {
		return fmt.Errorf("storing magic link token: %w", err)
	}

	// Build the magic link URL.
	// Use the configured redirect URL (Admin > Branding > Email) if set;
	// otherwise fall back to the built-in shark verify endpoint.
	baseURL := strings.TrimRight(m.cfg.Server.BaseURL, "/")
	fallbackMagicURL := fmt.Sprintf("%s/api/v1/auth/magic-link/verify", baseURL)
	redirectBase, _, _ := email.GetRedirectURL(ctx, m.store, "magic_link", fallbackMagicURL)
	magicLinkURL := email.AppendToken(redirectBase, rawToken)

	// Compute expiry in minutes for the email template
	expiryMinutes := int(lifetime.Minutes())
	if expiryMinutes < 1 {
		expiryMinutes = 1
	}

	appName := "SharkAuth"
	if m.cfg.SMTP.FromName != "" {
		appName = m.cfg.SMTP.FromName
	}

	// Render the email template (DB-backed with embedded fallback).
	branding, _ := m.store.ResolveBranding(ctx, "")
	rendered, err := email.RenderMagicLink(ctx, m.store, branding, email.MagicLinkData{
		AppName:       appName,
		MagicLinkURL:  magicLinkURL,
		ExpiryMinutes: expiryMinutes,
	})
	if err != nil {
		return fmt.Errorf("rendering magic link email: %w", err)
	}

	// Send the email
	msg := &email.Message{
		To:      emailAddr,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
	}

	if err := m.email.Send(msg); err != nil {
		return fmt.Errorf("sending magic link email: %w", err)
	}

	return nil
}

// SendPasswordReset generates a random token, stores its SHA-256 hash, and sends a password reset email.
// Returns errors on failure; the caller should log but not expose them to the client.
func (m *MagicLinkManager) SendPasswordReset(ctx context.Context, emailAddr string) error {
	// Generate 32 random bytes
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generating random token: %w", err)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	now := time.Now().UTC()
	lifetime := 15 * time.Minute

	id, _ := gonanoid.New()
	token := &storage.MagicLinkToken{
		ID:        "prt_" + id,
		Email:     emailAddr,
		TokenHash: tokenHash,
		Used:      false,
		ExpiresAt: now.Add(lifetime).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}

	if err := m.store.CreateMagicLinkToken(ctx, token); err != nil {
		return fmt.Errorf("storing password reset token: %w", err)
	}

	// Use DB-configured redirect URL if set; fall back to static config value.
	baseURLForReset := strings.TrimRight(m.cfg.Server.BaseURL, "/")
	fallbackResetURL := strings.TrimRight(m.cfg.PasswordReset.RedirectURL, "/")
	if fallbackResetURL == "" {
		fallbackResetURL = fmt.Sprintf("%s/api/v1/auth/password-reset/verify", baseURLForReset)
	}
	resetRedirectBase, _, _ := email.GetRedirectURL(ctx, m.store, "reset", fallbackResetURL)
	resetURL := email.AppendToken(resetRedirectBase, rawToken)

	appName := "SharkAuth"
	if m.cfg.SMTP.FromName != "" {
		appName = m.cfg.SMTP.FromName
	}

	branding, _ := m.store.ResolveBranding(ctx, "")
	rendered, err := email.RenderPasswordReset(ctx, m.store, branding, email.PasswordResetData{
		AppName:       appName,
		ResetURL:      resetURL,
		ExpiryMinutes: int(lifetime.Minutes()),
	})
	if err != nil {
		return fmt.Errorf("rendering password reset email: %w", err)
	}

	msg := &email.Message{
		To:      emailAddr,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
	}

	if err := m.email.Send(msg); err != nil {
		return fmt.Errorf("sending password reset email: %w", err)
	}

	return nil
}

// VerifyPasswordResetToken verifies a raw token and returns the associated email.
// Unlike VerifyMagicLink, this does not create a session â€” the caller handles the password update.
func (m *MagicLinkManager) VerifyPasswordResetToken(ctx context.Context, rawToken string) (string, error) {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	token, err := m.store.GetMagicLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrMagicLinkNotFound
		}
		return "", fmt.Errorf("looking up password reset token: %w", err)
	}

	if token.Used {
		return "", ErrMagicLinkUsed
	}

	expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("parsing token expiry: %w", err)
	}
	if time.Now().UTC().After(expiresAt) {
		return "", ErrMagicLinkExpired
	}

	if err := m.store.MarkMagicLinkTokenUsed(ctx, token.ID); err != nil {
		return "", fmt.Errorf("marking password reset token used: %w", err)
	}

	return token.Email, nil
}

// SendEmailVerification generates a verification token and sends a verification email.
func (m *MagicLinkManager) SendEmailVerification(ctx context.Context, emailAddr string) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generating random token: %w", err)
	}

	rawToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	now := time.Now().UTC()
	lifetime := 24 * time.Hour

	id, _ := gonanoid.New()
	token := &storage.MagicLinkToken{
		ID:        "evt_" + id,
		Email:     emailAddr,
		TokenHash: tokenHash,
		Used:      false,
		ExpiresAt: now.Add(lifetime).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}

	if err := m.store.CreateMagicLinkToken(ctx, token); err != nil {
		return fmt.Errorf("storing email verification token: %w", err)
	}

	// Use DB-configured redirect URL if set; fall back to built-in shark endpoint.
	// NOTE: /hosted/default/verify is NOT shipped in this version â€” devs MUST configure
	// their own verify_redirect_url via Admin > Branding > Email > Redirect URLs.
	baseURLForVerify := strings.TrimRight(m.cfg.Server.BaseURL, "/")
	fallbackVerifyURL := fmt.Sprintf("%s/api/v1/auth/verify-email", baseURLForVerify)
	verifyRedirectBase, _, _ := email.GetRedirectURL(ctx, m.store, "verify", fallbackVerifyURL)
	verifyURL := email.AppendToken(verifyRedirectBase, rawToken)

	appName := "SharkAuth"
	if m.cfg.SMTP.FromName != "" {
		appName = m.cfg.SMTP.FromName
	}

	branding, _ := m.store.ResolveBranding(ctx, "")
	rendered, err := email.RenderVerifyEmail(ctx, m.store, branding, email.VerifyEmailData{
		AppName:       appName,
		VerifyURL:     verifyURL,
		ExpiryMinutes: int(lifetime.Minutes()),
	})
	if err != nil {
		return fmt.Errorf("rendering verification email: %w", err)
	}

	msg := &email.Message{
		To:      emailAddr,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
	}

	if err := m.email.Send(msg); err != nil {
		return fmt.Errorf("sending verification email: %w", err)
	}

	return nil
}

// VerifyEmailToken verifies a raw token and returns the associated email.
func (m *MagicLinkManager) VerifyEmailToken(ctx context.Context, rawToken string) (string, error) {
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	token, err := m.store.GetMagicLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrMagicLinkNotFound
		}
		return "", fmt.Errorf("looking up verification token: %w", err)
	}

	if token.Used {
		return "", ErrMagicLinkUsed
	}

	expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("parsing token expiry: %w", err)
	}
	if time.Now().UTC().After(expiresAt) {
		return "", ErrMagicLinkExpired
	}

	if err := m.store.MarkMagicLinkTokenUsed(ctx, token.ID); err != nil {
		return "", fmt.Errorf("marking verification token used: %w", err)
	}

	return token.Email, nil
}

// VerifyMagicLink verifies a raw token, creates or finds the user, creates a session, and returns both.
func (m *MagicLinkManager) VerifyMagicLink(ctx context.Context, rawToken string) (*storage.User, *storage.Session, error) {
	// Hash the provided token
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := base64.RawURLEncoding.EncodeToString(hash[:])

	// Look up by hash
	token, err := m.store.GetMagicLinkTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrMagicLinkNotFound
		}
		return nil, nil, fmt.Errorf("looking up magic link token: %w", err)
	}

	// Check if already used
	if token.Used {
		return nil, nil, ErrMagicLinkUsed
	}

	// Check expiry
	expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing token expiry: %w", err)
	}
	if time.Now().UTC().After(expiresAt) {
		return nil, nil, ErrMagicLinkExpired
	}

	// Mark as used
	if err := m.store.MarkMagicLinkTokenUsed(ctx, token.ID); err != nil {
		return nil, nil, fmt.Errorf("marking magic link token used: %w", err)
	}

	// Find or create user
	user, err := m.store.GetUserByEmail(ctx, token.Email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("looking up user by email: %w", err)
		}

		// Create new user with email_verified=true
		id, _ := gonanoid.New()
		now := time.Now().UTC().Format(time.RFC3339)
		user = &storage.User{
			ID:            "usr_" + id,
			Email:         token.Email,
			EmailVerified: true,
			Metadata:      "{}",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := m.store.CreateUser(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("creating user: %w", err)
		}
	} else {
		// Existing user â€” ensure email is verified
		if !user.EmailVerified {
			user.EmailVerified = true
			user.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			if err := m.store.UpdateUser(ctx, user); err != nil {
				return nil, nil, fmt.Errorf("updating user email_verified: %w", err)
			}
		}
	}

	// Create session with auth_method="magic_link"
	sess, err := m.sessions.CreateSession(ctx, user.ID, "", "", "magic_link")
	if err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	return user, sess, nil
}
