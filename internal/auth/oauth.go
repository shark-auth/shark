package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/oauth2"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

var (
	ErrProviderNotFound    = errors.New("oauth provider not found")
	ErrProviderNotConfigured = errors.New("oauth provider not configured")
)

// OAuthProvider defines the interface for an OAuth identity provider.
type OAuthProvider interface {
	// Name returns the lowercase provider name (e.g. "google", "github").
	Name() string
	// AuthURL returns the URL to redirect the user to for authorization.
	AuthURL(state string) string
	// Exchange trades an authorization code for an access token.
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	// GetUser fetches the user's profile from the provider's API.
	GetUser(ctx context.Context, token *oauth2.Token) (*OAuthUserInfo, error)
}

// OAuthUserInfo holds the normalized user info returned by a provider.
type OAuthUserInfo struct {
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

// OAuthManager orchestrates the OAuth flow: provider lookup, user find-or-create,
// OAuth account linking, and session creation.
type OAuthManager struct {
	providers map[string]OAuthProvider
	store     storage.Store
	sessions  *SessionManager
	cfg       *config.Config
}

// NewOAuthManager creates an OAuthManager and registers all configured providers.
func NewOAuthManager(store storage.Store, sessions *SessionManager, cfg *config.Config) *OAuthManager {
	m := &OAuthManager{
		providers: make(map[string]OAuthProvider),
		store:     store,
		sessions:  sessions,
		cfg:       cfg,
	}
	return m
}

// RegisterProvider adds a provider to the manager.
func (m *OAuthManager) RegisterProvider(p OAuthProvider) {
	m.providers[p.Name()] = p
}

// GetProvider returns the named provider or an error if not registered.
func (m *OAuthManager) GetProvider(name string) (OAuthProvider, error) {
	p, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return p, nil
}

// HandleCallback exchanges the authorization code, fetches user info,
// finds or creates the user, links the OAuth account, and creates a session.
func (m *OAuthManager) HandleCallback(ctx context.Context, providerName, code, ip, userAgent string) (*storage.User, *storage.Session, error) {
	provider, err := m.GetProvider(providerName)
	if err != nil {
		return nil, nil, err
	}

	// Exchange code for token
	token, err := provider.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("exchanging code: %w", err)
	}

	// Fetch user info from provider
	info, err := provider.GetUser(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("getting user info: %w", err)
	}

	// Check if we already have this OAuth account linked
	existingAcct, err := m.store.GetOAuthAccountByProviderID(ctx, providerName, info.ProviderID)
	if err == nil && existingAcct != nil {
		// Already linked — load the user and create session
		user, err := m.store.GetUserByID(ctx, existingAcct.UserID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting linked user: %w", err)
		}
		sess, err := m.sessions.CreateSession(ctx, user.ID, ip, userAgent, "oauth:"+providerName)
		if err != nil {
			return nil, nil, fmt.Errorf("creating session: %w", err)
		}
		return user, sess, nil
	}

	// Not linked yet. Find or create user by email.
	now := time.Now().UTC().Format(time.RFC3339)
	user, err := m.store.GetUserByEmail(ctx, info.Email)
	if errors.Is(err, sql.ErrNoRows) {
		// Create new user
		id, _ := gonanoid.New()
		user = &storage.User{
			ID:            "usr_" + id,
			Email:         info.Email,
			EmailVerified: true, // provider verified the email
			Metadata:      "{}",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if info.Name != "" {
			user.Name = &info.Name
		}
		if info.AvatarURL != "" {
			user.AvatarURL = &info.AvatarURL
		}
		if err := m.store.CreateUser(ctx, user); err != nil {
			return nil, nil, fmt.Errorf("creating user: %w", err)
		}
	} else if err != nil {
		return nil, nil, fmt.Errorf("looking up user by email: %w", err)
	}

	// Link OAuth account to this user
	acctID, _ := gonanoid.New()
	accessToken := token.AccessToken
	refreshToken := token.RefreshToken
	oauthAcct := &storage.OAuthAccount{
		ID:         "oac_" + acctID,
		UserID:     user.ID,
		Provider:   providerName,
		ProviderID: info.ProviderID,
		Email:      &info.Email,
		CreatedAt:  now,
	}
	if accessToken != "" {
		oauthAcct.AccessToken = &accessToken
	}
	if refreshToken != "" {
		oauthAcct.RefreshToken = &refreshToken
	}

	if err := m.store.CreateOAuthAccount(ctx, oauthAcct); err != nil {
		return nil, nil, fmt.Errorf("creating oauth account: %w", err)
	}

	// Create session
	sess, err := m.sessions.CreateSession(ctx, user.ID, ip, userAgent, "oauth:"+providerName)
	if err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	return user, sess, nil
}
