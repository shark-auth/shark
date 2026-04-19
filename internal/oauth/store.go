// Package oauth provides the fosite storage adapter that bridges fosite's
// storage interfaces to SharkAuth's SQLite-backed storage.Store.
package oauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/ory/fosite/handler/oauth2"
	"github.com/ory/fosite/handler/pkce"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// Compile-time interface assertions.
var (
	_ fosite.ClientManager           = (*FositeStore)(nil)
	_ oauth2.CoreStorage             = (*FositeStore)(nil)
	_ oauth2.TokenRevocationStorage  = (*FositeStore)(nil)
	_ pkce.PKCERequestStorage        = (*FositeStore)(nil)
)

// FositeStore adapts storage.Store to fosite's storage interfaces.
type FositeStore struct {
	store storage.Store
}

// NewFositeStore creates a new fosite storage adapter wrapping the given store.
func NewFositeStore(store storage.Store) *FositeStore {
	return &FositeStore{store: store}
}

// ---------------------------------------------------------------------------
// SHA256Hasher — fosite.Hasher that uses SHA-256 hex comparison instead of
// bcrypt. SharkAuth stores client secrets as SHA-256 hex hashes.
// ---------------------------------------------------------------------------

// SHA256Hasher implements fosite.Hasher using SHA-256.
type SHA256Hasher struct{}

var _ fosite.Hasher = (*SHA256Hasher)(nil)

// Compare checks whether data hashes to the same SHA-256 hex digest as hash.
func (h *SHA256Hasher) Compare(_ context.Context, hash, data []byte) error {
	computed := sha256.Sum256(data)
	if !hmac.Equal(hash, []byte(hex.EncodeToString(computed[:]))) {
		return fosite.ErrNotFound.WithDebug("sha256 hash mismatch")
	}
	return nil
}

// Hash returns the SHA-256 hex digest of data.
func (h *SHA256Hasher) Hash(_ context.Context, data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return []byte(hex.EncodeToString(sum[:])), nil
}

// ---------------------------------------------------------------------------
// ClientManager — GetClient, ClientAssertionJWTValid, SetClientAssertionJWT
// ---------------------------------------------------------------------------

// GetClient looks up an agent by its OAuth client_id and maps it to a
// fosite.DefaultOpenIDConnectClient.
func (s *FositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
	agent, err := s.store.GetAgentByClientID(ctx, id)
	if err != nil {
		return nil, fosite.ErrNotFound.WithWrap(err).WithDebugf("agent with client_id %q not found", id)
	}
	if !agent.Active {
		return nil, fosite.ErrNotFound.WithDebugf("agent with client_id %q is inactive", id)
	}

	return agentToClient(agent), nil
}

// agentToClient converts a storage.Agent to a fosite.DefaultOpenIDConnectClient.
func agentToClient(agent *storage.Agent) *fosite.DefaultOpenIDConnectClient {
	return &fosite.DefaultOpenIDConnectClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            agent.ClientID,
			Secret:        []byte(agent.ClientSecretHash),
			RedirectURIs:  agent.RedirectURIs,
			GrantTypes:    agent.GrantTypes,
			ResponseTypes: agent.ResponseTypes,
			Scopes:        agent.Scopes,
			Public:        agent.ClientType == "public",
		},
		TokenEndpointAuthMethod: agent.AuthMethod,
	}
}

// ClientAssertionJWTValid checks whether a JTI has already been used.
// We delegate to the revoked JTI table that already exists in SharkAuth.
func (s *FositeStore) ClientAssertionJWTValid(ctx context.Context, jti string) error {
	revoked, err := s.store.IsRevokedJTI(ctx, jti)
	if err != nil {
		return fosite.ErrServerError.WithWrap(err)
	}
	if revoked {
		return fosite.ErrJTIKnown
	}
	return nil
}

// SetClientAssertionJWT marks a JTI as used with the given expiry.
func (s *FositeStore) SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error {
	return s.store.InsertRevokedJTI(ctx, jti, exp)
}

// ---------------------------------------------------------------------------
// AuthorizeCodeStorage
// ---------------------------------------------------------------------------

// CreateAuthorizeCodeSession stores a fosite request associated with an auth code.
// The code parameter is the auth code signature (unhashed).
func (s *FositeStore) CreateAuthorizeCodeSession(ctx context.Context, code string, req fosite.Requester) error {
	codeHash := hashSignature(code)

	// Extract session data.
	sess := req.GetSession()
	expiresAt := sess.GetExpiresAt(fosite.AuthorizeCode)
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(10 * time.Minute)
	}

	// Extract PKCE fields from the form if present.
	codeChallenge := req.GetRequestForm().Get("code_challenge")
	codeChallengeMethod := req.GetRequestForm().Get("code_challenge_method")
	if codeChallengeMethod == "" {
		codeChallengeMethod = "S256"
	}

	sc := &storage.OAuthAuthorizationCode{
		CodeHash:            codeHash,
		ClientID:            req.GetClient().GetID(),
		UserID:              sess.GetSubject(),
		RedirectURI:         req.GetRequestForm().Get("redirect_uri"),
		Scope:               strings.Join([]string(req.GetRequestedScopes()), " "),
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Nonce:               req.GetRequestForm().Get("nonce"),
		ExpiresAt:           expiresAt,
		CreatedAt:           req.GetRequestedAt(),
	}

	return s.store.CreateAuthorizationCode(ctx, sc)
}

// GetAuthorizeCodeSession retrieves the authorization request for a given code.
func (s *FositeStore) GetAuthorizeCodeSession(ctx context.Context, code string, session fosite.Session) (fosite.Requester, error) {
	codeHash := hashSignature(code)
	sc, err := s.store.GetAuthorizationCode(ctx, codeHash)
	if err != nil {
		return nil, fosite.ErrNotFound.WithWrap(err)
	}

	client, err := s.GetClient(ctx, sc.ClientID)
	if err != nil {
		return nil, err
	}

	if session != nil {
		setSessionSubject(session, sc.UserID)
		session.SetExpiresAt(fosite.AuthorizeCode, sc.ExpiresAt)
	}

	req := &fosite.Request{
		ID:            sc.CodeHash, // use code hash as the request ID
		RequestedAt:   sc.CreatedAt,
		Client:        client,
		RequestedScope: fosite.Arguments(strings.Split(sc.Scope, " ")),
		GrantedScope:   fosite.Arguments(strings.Split(sc.Scope, " ")),
		Session:       session,
		Form: url.Values{
			"redirect_uri":          {sc.RedirectURI},
			"code_challenge":        {sc.CodeChallenge},
			"code_challenge_method": {sc.CodeChallengeMethod},
			"nonce":                 {sc.Nonce},
		},
	}

	return req, nil
}

// InvalidateAuthorizeCodeSession marks an authorization code as used by deleting it.
func (s *FositeStore) InvalidateAuthorizeCodeSession(ctx context.Context, code string) error {
	codeHash := hashSignature(code)
	return s.store.DeleteAuthorizationCode(ctx, codeHash)
}

// ---------------------------------------------------------------------------
// AccessTokenStorage
// ---------------------------------------------------------------------------

// CreateAccessTokenSession stores an access token session.
func (s *FositeStore) CreateAccessTokenSession(ctx context.Context, signature string, req fosite.Requester) error {
	return s.createTokenSession(ctx, signature, "access", req)
}

// GetAccessTokenSession retrieves an access token session by its signature.
func (s *FositeStore) GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.getTokenSession(ctx, signature, "access", session)
}

// DeleteAccessTokenSession deletes an access token session.
func (s *FositeStore) DeleteAccessTokenSession(ctx context.Context, signature string) error {
	return s.deleteTokenSession(ctx, signature)
}

// ---------------------------------------------------------------------------
// RefreshTokenStorage
// ---------------------------------------------------------------------------

// CreateRefreshTokenSession stores a refresh token session.
func (s *FositeStore) CreateRefreshTokenSession(ctx context.Context, signature string, accessSignature string, req fosite.Requester) error {
	return s.createTokenSession(ctx, signature, "refresh", req)
}

// GetRefreshTokenSession retrieves a refresh token session by its signature.
func (s *FositeStore) GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.getTokenSession(ctx, signature, "refresh", session)
}

// DeleteRefreshTokenSession deletes a refresh token session.
func (s *FositeStore) DeleteRefreshTokenSession(ctx context.Context, signature string) error {
	return s.deleteTokenSession(ctx, signature)
}

// RotateRefreshToken invalidates the old refresh token signature and is called
// during token refresh rotation. The requestID is the fosite request ID, which
// we store as the token's JTI.
func (s *FositeStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error {
	// Find the old refresh token by request ID (stored as JTI) and revoke it.
	token, err := s.store.GetOAuthTokenByJTI(ctx, requestID)
	if err != nil {
		// If not found, that's okay — the token may have already been rotated.
		return nil
	}
	if token.RevokedAt == nil {
		return s.store.RevokeOAuthToken(ctx, token.ID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// TokenRevocationStorage — RevokeAccessToken / RevokeRefreshToken
// ---------------------------------------------------------------------------

// RevokeAccessToken revokes an access token by its request ID (stored as JTI).
func (s *FositeStore) RevokeAccessToken(ctx context.Context, requestID string) error {
	token, err := s.store.GetOAuthTokenByJTI(ctx, requestID)
	if err != nil {
		return fosite.ErrNotFound.WithWrap(err)
	}
	return s.store.RevokeOAuthToken(ctx, token.ID)
}

// RevokeRefreshToken revokes a refresh token by its request ID (stored as JTI).
func (s *FositeStore) RevokeRefreshToken(ctx context.Context, requestID string) error {
	token, err := s.store.GetOAuthTokenByJTI(ctx, requestID)
	if err != nil {
		return fosite.ErrNotFound.WithWrap(err)
	}
	return s.store.RevokeOAuthToken(ctx, token.ID)
}

// ---------------------------------------------------------------------------
// PKCERequestStorage
// ---------------------------------------------------------------------------

// CreatePKCERequestSession stores the PKCE request data. We piggyback on the
// auth code session since PKCE data is already stored in our authorization
// codes table (code_challenge, code_challenge_method columns).
func (s *FositeStore) CreatePKCERequestSession(ctx context.Context, signature string, req fosite.Requester) error {
	// PKCE data is stored as part of the authorization code session.
	// fosite calls CreateAuthorizeCodeSession and CreatePKCERequestSession
	// with the same code. We only need to store once — the auth code
	// already has PKCE columns. So this is a no-op if the auth code
	// session already exists.
	return nil
}

// GetPKCERequestSession retrieves the PKCE request session. We reconstruct it
// from the authorization code.
func (s *FositeStore) GetPKCERequestSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error) {
	return s.GetAuthorizeCodeSession(ctx, signature, session)
}

// DeletePKCERequestSession deletes the PKCE request session. This is a no-op
// because the auth code deletion handles cleanup.
func (s *FositeStore) DeletePKCERequestSession(ctx context.Context, signature string) error {
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// hashSignature returns the SHA-256 hex digest of a token signature.
func hashSignature(sig string) string {
	h := sha256.Sum256([]byte(sig))
	return hex.EncodeToString(h[:])
}

// createTokenSession is the shared implementation for creating access and
// refresh token sessions.
func (s *FositeStore) createTokenSession(ctx context.Context, signature, tokenType string, req fosite.Requester) error {
	tokenHash := hashSignature(signature)
	sess := req.GetSession()

	expiresAt := sess.GetExpiresAt(fosite.AccessToken)
	if tokenType == "refresh" {
		expiresAt = sess.GetExpiresAt(fosite.RefreshToken)
	}
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(time.Hour)
	}

	scope := strings.Join([]string(req.GetGrantedScopes()), " ")

	token := &storage.OAuthToken{
		ID:        "tok_" + uuid.New().String()[:8],
		JTI:       req.GetID(),
		ClientID:  req.GetClient().GetID(),
		UserID:    sess.GetSubject(),
		TokenType: tokenType,
		TokenHash: tokenHash,
		Scope:     scope,
		ExpiresAt: expiresAt,
		CreatedAt: req.GetRequestedAt(),
	}

	return s.store.CreateOAuthToken(ctx, token)
}

// getTokenSession retrieves a token session by its signature hash.
func (s *FositeStore) getTokenSession(ctx context.Context, signature, tokenType string, session fosite.Session) (fosite.Requester, error) {
	tokenHash := hashSignature(signature)

	token, err := s.store.GetOAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, fosite.ErrNotFound.WithWrap(err)
	}

	if token.TokenType != tokenType {
		return nil, fosite.ErrNotFound.WithDebugf("token type mismatch: wanted %s, got %s", tokenType, token.TokenType)
	}

	// Check if revoked.
	if token.RevokedAt != nil {
		return nil, fosite.ErrInactiveToken.WithDebug("token has been revoked")
	}

	client, err := s.GetClient(ctx, token.ClientID)
	if err != nil {
		// Client may have been deactivated; still return the request for
		// introspection/revocation purposes but with a basic client stub.
		client = &fosite.DefaultOpenIDConnectClient{
			DefaultClient: &fosite.DefaultClient{ID: token.ClientID},
		}
	}

	if session != nil {
		setSessionSubject(session, token.UserID)
		tokenTypeKey := fosite.AccessToken
		if tokenType == "refresh" {
			tokenTypeKey = fosite.RefreshToken
		}
		session.SetExpiresAt(tokenTypeKey, token.ExpiresAt)
	}

	req := &fosite.Request{
		ID:             token.JTI,
		RequestedAt:    token.CreatedAt,
		Client:         client,
		RequestedScope: fosite.Arguments(strings.Split(token.Scope, " ")),
		GrantedScope:   fosite.Arguments(strings.Split(token.Scope, " ")),
		Session:        session,
		Form:           url.Values{},
	}

	return req, nil
}

// deleteTokenSession removes a token by its signature hash.
func (s *FositeStore) deleteTokenSession(ctx context.Context, signature string) error {
	tokenHash := hashSignature(signature)
	token, err := s.store.GetOAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		// If not found, treat as success (idempotent delete).
		if isNotFound(err) {
			return nil
		}
		return fosite.ErrServerError.WithWrap(err)
	}
	return s.store.RevokeOAuthToken(ctx, token.ID)
}

// isNotFound is a simple check for "not found" errors from storage.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if err == sql.ErrNoRows {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}

// subjectSetter is a helper interface for sessions that support SetSubject.
// fosite.DefaultSession implements it but the fosite.Session interface does not
// include it.
type subjectSetter interface {
	SetSubject(string)
}

// setSessionSubject sets the subject on a session if it supports SetSubject.
func setSessionSubject(session fosite.Session, subject string) {
	if ss, ok := session.(subjectSetter); ok {
		ss.SetSubject(subject)
	}
}
