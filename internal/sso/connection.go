// Package sso implements SSO connection management, OIDC client, and SAML SP
// functionality for SharkAuth.
package sso

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/storage"
)

// SessionCreator abstracts session creation so the SSO package does not depend
// on the concrete auth.SessionManager type.
type SessionCreator interface {
	CreateSession(ctx context.Context, userID, ip, userAgent, authMethod string) (*storage.Session, error)
}

// SSOManager manages SSO connections and handles OIDC/SAML flows.
type SSOManager struct {
	store    storage.Store
	sessions SessionCreator
	cfg      *config.Config
}

// NewSSOManager creates a new SSOManager.
func NewSSOManager(store storage.Store, sessions SessionCreator, cfg *config.Config) *SSOManager {
	return &SSOManager{
		store:    store,
		sessions: sessions,
		cfg:      cfg,
	}
}

// CreateConnection creates a new SSO connection.
func (s *SSOManager) CreateConnection(ctx context.Context, conn *storage.SSOConnection) error {
	if conn.Type != "saml" && conn.Type != "oidc" {
		return fmt.Errorf("invalid connection type: %q (must be \"saml\" or \"oidc\")", conn.Type)
	}
	if conn.Name == "" {
		return fmt.Errorf("connection name is required")
	}

	id, err := generateID("sso_")
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	conn.ID = id

	now := time.Now().UTC().Format(time.RFC3339)
	conn.CreatedAt = now
	conn.UpdatedAt = now
	conn.Enabled = true

	// Set SAML defaults from config if not specified
	if conn.Type == "saml" {
		if ptrEmpty(conn.SAMLSPEntityID) && s.cfg.SSO.SAML.SPEntityID != "" {
			conn.SAMLSPEntityID = strPtr(s.cfg.SSO.SAML.SPEntityID)
		}
		if ptrEmpty(conn.SAMLSPAcsURL) && s.cfg.Server.BaseURL != "" {
			conn.SAMLSPAcsURL = strPtr(fmt.Sprintf("%s/api/v1/sso/saml/%s/acs", s.cfg.Server.BaseURL, conn.ID))
		}
	}

	return s.store.CreateSSOConnection(ctx, conn)
}

// GetConnection retrieves an SSO connection by ID.
func (s *SSOManager) GetConnection(ctx context.Context, id string) (*storage.SSOConnection, error) {
	conn, err := s.store.GetSSOConnectionByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return nil, fmt.Errorf("connection not found: %s", id)
	}
	return conn, nil
}

// ListConnections returns all SSO connections.
func (s *SSOManager) ListConnections(ctx context.Context) ([]*storage.SSOConnection, error) {
	return s.store.ListSSOConnections(ctx)
}

// UpdateConnection updates an existing SSO connection.
func (s *SSOManager) UpdateConnection(ctx context.Context, conn *storage.SSOConnection) error {
	if conn.ID == "" {
		return fmt.Errorf("connection ID is required")
	}

	existing, err := s.store.GetSSOConnectionByID(ctx, conn.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("connection not found: %s", conn.ID)
	}
	if err != nil {
		return fmt.Errorf("get existing connection: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("connection not found: %s", conn.ID)
	}

	conn.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	// Preserve immutable fields
	conn.CreatedAt = existing.CreatedAt

	return s.store.UpdateSSOConnection(ctx, conn)
}

// DeleteConnection deletes an SSO connection by ID.
func (s *SSOManager) DeleteConnection(ctx context.Context, id string) error {
	existing, err := s.store.GetSSOConnectionByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("connection not found: %s", id)
	}
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("connection not found: %s", id)
	}
	return s.store.DeleteSSOConnection(ctx, id)
}

// RouteByEmail looks up the SSO connection for a given email address
// by extracting the domain and finding a matching connection.
func (s *SSOManager) RouteByEmail(ctx context.Context, email string) (*storage.SSOConnection, error) {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("invalid email address: %q", email)
	}
	domain := strings.ToLower(parts[1])

	conn, err := s.store.GetSSOConnectionByDomain(ctx, domain)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("no SSO connection for domain %q", domain)
	}
	if err != nil {
		return nil, fmt.Errorf("lookup domain %q: %w", domain, err)
	}
	if conn == nil {
		return nil, fmt.Errorf("no SSO connection for domain %q", domain)
	}
	if !conn.Enabled {
		return nil, fmt.Errorf("SSO connection for domain %q is disabled", domain)
	}
	return conn, nil
}

// findOrCreateUser looks up a user by SSO identity, or creates one if not found.
func (s *SSOManager) findOrCreateUser(ctx context.Context, connectionID, providerSub, email, name string) (*storage.User, error) {
	// Check if we already have an SSO identity for this subject
	identity, err := s.store.GetSSOIdentityByConnectionAndSub(ctx, connectionID, providerSub)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup sso identity: %w", err)
	}

	if identity != nil {
		user, err := s.store.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return nil, fmt.Errorf("get user by id: %w", err)
		}
		return user, nil
	}

	// Check if user exists by email â€” if so, link the SSO identity
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup user by email: %w", err)
	}

	if user == nil {
		// Create new user
		uid, err := generateID("usr_")
		if err != nil {
			return nil, fmt.Errorf("generate user id: %w", err)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		user = &storage.User{
			ID:            uid,
			Email:         email,
			EmailVerified: true, // SSO-verified emails are trusted
			Name:          strPtr(name),
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := s.store.CreateUser(ctx, user); err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	// Create SSO identity link
	identityID, err := generateID("ssoid_")
	if err != nil {
		return nil, fmt.Errorf("generate identity id: %w", err)
	}
	ssoIdentity := &storage.SSOIdentity{
		ID:           identityID,
		UserID:       user.ID,
		ConnectionID: connectionID,
		ProviderSub:  providerSub,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.store.CreateSSOIdentity(ctx, ssoIdentity); err != nil {
		return nil, fmt.Errorf("create sso identity: %w", err)
	}

	return user, nil
}

func generateID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// ptrEmpty returns true if p is nil or points to the empty string.
func ptrEmpty(p *string) bool { return p == nil || *p == "" }

// derefStr returns the string a pointer points to, or "" if nil.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
