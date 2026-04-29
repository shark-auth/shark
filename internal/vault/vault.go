// Package vault implements the Token Vault: managed third-party OAuth token
// storage. Agents and dashboards request tokens through this package rather
// than handling raw credentials. Tokens are encrypted at rest using AES-256-GCM
// (see internal/auth.FieldEncryptor) and refreshed lazily on retrieval.
package vault

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/storage"
)

// Sentinel errors returned by the Manager.
var (
	// ErrNeedsReauth is returned when the user's refresh token has been
	// rejected by the upstream provider. The caller must surface a re-consent
	// prompt; calling GetFreshToken again will keep failing until the user
	// reconnects via the OAuth flow.
	ErrNeedsReauth = errors.New("vault: connection needs re-auth")

	// ErrNoRefreshToken is returned when an access token has expired but the
	// provider never issued a refresh token (or the user's consent was a
	// one-shot). The connection stays put but cannot be revived without a
	// new authorize round-trip.
	ErrNoRefreshToken = errors.New("vault: access token expired and no refresh token available")

	// ErrConnectionNotFound is returned by GetFreshToken/Disconnect when
	// there is no matching connection row.
	ErrConnectionNotFound = errors.New("vault: connection not found")

	// ErrProviderNotFound is returned when a provider ID or name does not
	// resolve to a VaultProvider row.
	ErrProviderNotFound = errors.New("vault: provider not found")
)

// expiryLeeway is the cushion applied when deciding whether an access token
// is "expired" â€” we refresh slightly early so callers never receive a token
// that's about to die mid-request.
const expiryLeeway = 30 * time.Second

// Manager orchestrates the vault: OAuth authorize URL construction, code
// exchange, encrypted persistence, auto-refresh, and disconnect.
//
// The `now` field is injectable so tests can control "is this token expired"
// logic deterministically. Production callers use NewManager which wires
// time.Now.
type Manager struct {
	store     storage.Store
	encryptor *auth.FieldEncryptor
	now       func() time.Time
}

// NewManager constructs a Manager with real wall-clock time.
func NewManager(store storage.Store, encryptor *auth.FieldEncryptor) *Manager {
	return &Manager{
		store:     store,
		encryptor: encryptor,
		now:       time.Now,
	}
}

// NewManagerWithClock is the test-seam variant; pass a stub clock to make
// expiry behaviour deterministic.
func NewManagerWithClock(store storage.Store, encryptor *auth.FieldEncryptor, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{store: store, encryptor: encryptor, now: now}
}

// --- Provider CRUD ---

// CreateProvider persists a new VaultProvider row after encrypting the raw
// client_secret. The caller provides plaintext; the manager owns the crypto
// boundary so handlers never touch the cipher directly.
//
// Fills ID / CreatedAt / UpdatedAt if the caller left them zero.
func (m *Manager) CreateProvider(ctx context.Context, provider *storage.VaultProvider, clientSecretPlain string) error {
	if provider == nil {
		return errors.New("vault: provider is nil")
	}
	if provider.Name == "" {
		return errors.New("vault: provider name is required")
	}
	if provider.AuthURL == "" || provider.TokenURL == "" {
		return errors.New("vault: provider auth_url and token_url are required")
	}
	if provider.ClientID == "" {
		return errors.New("vault: provider client_id is required")
	}

	encSecret, err := m.encryptor.Encrypt(clientSecretPlain)
	if err != nil {
		return fmt.Errorf("encrypt client_secret: %w", err)
	}
	provider.ClientSecretEnc = encSecret

	if provider.ID == "" {
		id, err := newID("vp_")
		if err != nil {
			return fmt.Errorf("generate provider id: %w", err)
		}
		provider.ID = id
	}
	now := m.now().UTC()
	if provider.CreatedAt.IsZero() {
		provider.CreatedAt = now
	}
	if provider.UpdatedAt.IsZero() {
		provider.UpdatedAt = now
	}
	if provider.Scopes == nil {
		provider.Scopes = []string{}
	}

	return m.store.CreateVaultProvider(ctx, provider)
}

// UpdateProviderSecret re-encrypts the client secret and persists it. Used
// when an admin rotates credentials. Everything else on the provider is
// left as-is; call UpdateVaultProvider via the store for other edits.
func (m *Manager) UpdateProviderSecret(ctx context.Context, providerID, clientSecretPlain string) error {
	provider, err := m.store.GetVaultProviderByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("load provider: %w", err)
	}
	if provider == nil {
		return ErrProviderNotFound
	}

	encSecret, err := m.encryptor.Encrypt(clientSecretPlain)
	if err != nil {
		return fmt.Errorf("encrypt client_secret: %w", err)
	}
	provider.ClientSecretEnc = encSecret
	provider.UpdatedAt = m.now().UTC()
	return m.store.UpdateVaultProvider(ctx, provider)
}

// --- OAuth flow ---

// BuildAuthURL returns the provider's authorize URL for the given state and
// redirect URI. Scope list falls back to the provider default when empty.
// Requests offline access so we receive a refresh_token on the first consent.
func (m *Manager) BuildAuthURL(ctx context.Context, providerID, state, redirectURI string, scopes []string) (string, error) {
	provider, err := m.store.GetVaultProviderByID(ctx, providerID)
	if err != nil {
		return "", fmt.Errorf("load provider: %w", err)
	}
	if provider == nil {
		return "", ErrProviderNotFound
	}
	if !provider.Active {
		return "", fmt.Errorf("vault: provider %q is disabled", provider.Name)
	}

	cfg, err := m.oauthConfig(provider, redirectURI, scopes)
	if err != nil {
		return "", err
	}

	// AccessTypeOffline triggers a refresh token on the first consent with
	// providers that honour it (notably Google); harmless on providers that
	// ignore it.
	opts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}

	// Extra authorize-URL params (e.g. prompt=consent for Linear,
	// audience=api.atlassian.com for Jira).
	// Primary: read from the persisted ExtraAuthParams on the provider row â€”
	// this covers both template-created and manual providers uniformly.
	// Backward-compat: if the row has no persisted params (pre-migration rows
	// that defaulted to "{}"), fall back to the built-in template lookup so
	// existing deployments don't silently lose Wave F behaviour until they
	// run an update/re-create.
	extraParams := provider.ExtraAuthParams
	if len(extraParams) == 0 {
		if tpl, ok := Template(provider.Name); ok {
			extraParams = tpl.ExtraAuthParams
		}
	}
	for k, v := range extraParams {
		opts = append(opts, oauth2.SetAuthURLParam(k, v))
	}

	return cfg.AuthCodeURL(state, opts...), nil
}

// ExchangeAndStore completes the OAuth dance: swaps `code` for tokens,
// encrypts them, and upserts a vault_connection row for (providerID, userID).
//
// Callers are responsible for validating `state` before calling this.
// Returns the freshly-stored connection so handlers can echo the granted
// scopes and expiry back to the user.
func (m *Manager) ExchangeAndStore(ctx context.Context, providerID, userID, code, redirectURI string) (*storage.VaultConnection, error) {
	provider, err := m.store.GetVaultProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("load provider: %w", err)
	}
	if provider == nil {
		return nil, ErrProviderNotFound
	}

	cfg, err := m.oauthConfig(provider, redirectURI, nil)
	if err != nil {
		return nil, err
	}

	// Determine token response shape from the built-in template (if any).
	var tokenShape string
	if tpl, ok := Template(provider.Name); ok {
		tokenShape = tpl.TokenResponseShape
	}

	var token *oauth2.Token
	switch tokenShape {
	case "slack_v2":
		token, err = m.exchangeSlackV2(ctx, cfg, code)
	default:
		token, err = cfg.Exchange(ctx, code)
	}
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}

	accessEnc, err := m.encryptor.Encrypt(token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt access token: %w", err)
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}
	var expiresAt *time.Time
	if !token.Expiry.IsZero() {
		e := token.Expiry.UTC()
		expiresAt = &e
	}

	// Scopes granted may differ from scopes requested. oauth2 keeps the
	// granted list in the "scope" extra when the provider echoes it back.
	grantedScopes := extractGrantedScopes(token, cfg.Scopes)

	now := m.now().UTC()

	existing, err := m.store.GetVaultConnection(ctx, providerID, userID)
	if err != nil && !isNoRows(err) {
		return nil, fmt.Errorf("lookup existing connection: %w", err)
	}

	if existing != nil {
		existing.AccessTokenEnc = accessEnc
		// Many providers omit refresh_token on re-exchange when the user has
		// already consented; preserve the existing ciphertext in that case
		// rather than overwriting with an encrypted empty string. Mirrors
		// the refresh-path behaviour below.
		if token.RefreshToken == "" && existing.RefreshTokenEnc != "" {
			// keep existing ciphertext â€” upstream didn't rotate
		} else {
			refreshEnc, err := m.encryptor.Encrypt(token.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("encrypt refresh token: %w", err)
			}
			existing.RefreshTokenEnc = refreshEnc
		}
		existing.TokenType = tokenType
		existing.Scopes = grantedScopes
		existing.ExpiresAt = expiresAt
		existing.NeedsReauth = false
		existing.LastRefreshedAt = &now
		existing.UpdatedAt = now
		if err := m.store.UpdateVaultConnection(ctx, existing); err != nil {
			return nil, fmt.Errorf("update connection: %w", err)
		}
		return existing, nil
	}

	refreshEnc, err := m.encryptor.Encrypt(token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("encrypt refresh token: %w", err)
	}

	id, err := newID("vc_")
	if err != nil {
		return nil, fmt.Errorf("generate connection id: %w", err)
	}
	conn := &storage.VaultConnection{
		ID:              id,
		ProviderID:      providerID,
		UserID:          userID,
		AccessTokenEnc:  accessEnc,
		RefreshTokenEnc: refreshEnc,
		TokenType:       tokenType,
		Scopes:          grantedScopes,
		ExpiresAt:       expiresAt,
		Metadata:        map[string]any{},
		NeedsReauth:     false,
		LastRefreshedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := m.store.CreateVaultConnection(ctx, conn); err != nil {
		return nil, fmt.Errorf("create connection: %w", err)
	}
	return conn, nil
}

// GetFreshToken returns a decrypted access token for (providerID, userID),
// refreshing it transparently when expiry is within `expiryLeeway`.
//
// Return contract:
//   - nil error + non-empty access token: caller may use it
//   - ErrNeedsReauth: refresh failed; row has needs_reauth=1
//   - ErrNoRefreshToken: access expired + no refresh token stored
//   - ErrConnectionNotFound / ErrProviderNotFound: caller should surface as 404
func (m *Manager) GetFreshToken(ctx context.Context, providerID, userID string) (string, error) {
	conn, err := m.store.GetVaultConnection(ctx, providerID, userID)
	if err != nil {
		if isNoRows(err) {
			return "", ErrConnectionNotFound
		}
		return "", fmt.Errorf("load connection: %w", err)
	}
	if conn == nil {
		return "", ErrConnectionNotFound
	}
	if conn.NeedsReauth {
		return "", ErrNeedsReauth
	}

	if !m.isExpired(conn.ExpiresAt) {
		access, err := m.encryptor.Decrypt(conn.AccessTokenEnc)
		if err != nil {
			return "", fmt.Errorf("decrypt access token: %w", err)
		}
		return access, nil
	}

	// Expired â€” try to refresh.
	refreshPlain, err := m.encryptor.Decrypt(conn.RefreshTokenEnc)
	if err != nil {
		return "", fmt.Errorf("decrypt refresh token: %w", err)
	}
	if refreshPlain == "" {
		return "", ErrNoRefreshToken
	}

	provider, err := m.store.GetVaultProviderByID(ctx, conn.ProviderID)
	if err != nil {
		return "", fmt.Errorf("load provider: %w", err)
	}
	if provider == nil {
		return "", ErrProviderNotFound
	}

	cfg, err := m.oauthConfig(provider, "", nil)
	if err != nil {
		return "", err
	}

	// oauth2.TokenSource handles the POST-to-token-endpoint refresh and any
	// provider-specific quirks. We feed it an expired *oauth2.Token so it
	// always performs the refresh. Use the injectable clock so tests stay
	// deterministic.
	expiredAt := m.now().Add(-time.Hour)
	if conn.ExpiresAt != nil {
		expiredAt = *conn.ExpiresAt
	}
	source := cfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshPlain,
		Expiry:       expiredAt,
	})
	fresh, err := source.Token()
	if err != nil {
		// Refresh rejected â€” mark re-auth required. If flipping the flag
		// itself fails we surface the storage error so operators see it
		// instead of silently stranding the connection in an inconsistent
		// state.
		if markErr := m.store.MarkVaultConnectionNeedsReauth(ctx, conn.ID, true); markErr != nil {
			return "", fmt.Errorf("vault: refresh failed and mark-reauth failed: %w", markErr)
		}
		return "", ErrNeedsReauth
	}

	newAccessEnc, err := m.encryptor.Encrypt(fresh.AccessToken)
	if err != nil {
		return "", fmt.Errorf("encrypt refreshed access token: %w", err)
	}
	// Many providers omit the refresh token on refresh responses â€” preserve
	// the existing one when that happens.
	newRefreshPlain := fresh.RefreshToken
	if newRefreshPlain == "" {
		newRefreshPlain = refreshPlain
	}
	newRefreshEnc, err := m.encryptor.Encrypt(newRefreshPlain)
	if err != nil {
		return "", fmt.Errorf("encrypt refreshed refresh token: %w", err)
	}

	var newExpiry *time.Time
	if !fresh.Expiry.IsZero() {
		e := fresh.Expiry.UTC()
		newExpiry = &e
	}

	if err := m.store.UpdateVaultConnectionTokens(ctx, conn.ID, newAccessEnc, newRefreshEnc, newExpiry); err != nil {
		return "", fmt.Errorf("persist refreshed tokens: %w", err)
	}

	return fresh.AccessToken, nil
}

// DecryptAccessToken decrypts the access token stored on conn and returns the
// plaintext. Used by handler-layer post-exchange steps (e.g. Atlassian
// accessible-resources) that need to call upstream APIs immediately after
// ExchangeAndStore.
func (m *Manager) DecryptAccessToken(_ context.Context, conn *storage.VaultConnection) (string, error) {
	if conn == nil {
		return "", errors.New("vault: connection is nil")
	}
	plain, err := m.encryptor.Decrypt(conn.AccessTokenEnc)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}
	return plain, nil
}

// Disconnect deletes a single connection by its ID. Returns
// ErrConnectionNotFound when no row matched.
func (m *Manager) Disconnect(ctx context.Context, connectionID string) error {
	conn, err := m.store.GetVaultConnectionByID(ctx, connectionID)
	if err != nil {
		if isNoRows(err) {
			return ErrConnectionNotFound
		}
		return fmt.Errorf("load connection: %w", err)
	}
	if conn == nil {
		return ErrConnectionNotFound
	}
	return m.store.DeleteVaultConnection(ctx, connectionID)
}

// ListConnections returns the user's connections (no decrypted tokens).
func (m *Manager) ListConnections(ctx context.Context, userID string) ([]*storage.VaultConnection, error) {
	return m.store.ListVaultConnectionsByUserID(ctx, userID)
}

// --- Internal helpers ---

// exchangeSlackV2 performs the token exchange for Slack's non-standard
// oauth.v2.access endpoint. Slack returns HTTP 200 even on errors, using an
// `ok` boolean field instead of a non-2xx status. The response may contain
// both a bot token (top-level access_token, xoxb-...) and a user token
// (authed_user.access_token, xoxp-...). We prefer the user token when present
// so callers get user-scoped access; the bot token is stored as a fallback.
//
// Reference: https://api.slack.com/methods/oauth.v2.access
func (m *Manager) exchangeSlackV2(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	// Build the POST form body directly â€” we can't use cfg.Exchange because
	// oauth2.Transport parses the response as RFC 6749 and misses the ok flag.
	form := url.Values{
		"code":          {code},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURL},
	}

	httpClient := http.DefaultClient
	if hc, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && hc != nil {
		httpClient = hc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint.TokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("slack_v2: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack_v2: POST: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("slack_v2: read body: %w", err)
	}

	var raw struct {
		OK          bool   `json:"ok"`
		Error       string `json:"error"`
		AccessToken string `json:"access_token"` // bot token (xoxb-)
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		AuthedUser  struct {
			AccessToken string `json:"access_token"` // user token (xoxp-)
			Scope       string `json:"scope"`
			TokenType   string `json:"token_type"`
		} `json:"authed_user"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("slack_v2: parse response: %w", err)
	}
	if !raw.OK {
		errMsg := raw.Error
		if errMsg == "" {
			errMsg = "unknown_error"
		}
		return nil, fmt.Errorf("slack_v2: exchange failed: %s", errMsg)
	}

	// Prefer the user (xoxp) token when present; fall back to the bot (xoxb) token.
	accessToken := raw.AccessToken
	tokenType := raw.TokenType
	scope := raw.Scope
	if raw.AuthedUser.AccessToken != "" {
		accessToken = raw.AuthedUser.AccessToken
		if raw.AuthedUser.TokenType != "" {
			tokenType = raw.AuthedUser.TokenType
		}
		if raw.AuthedUser.Scope != "" {
			scope = raw.AuthedUser.Scope
		}
	}
	if accessToken == "" {
		return nil, fmt.Errorf("slack_v2: exchange succeeded but access_token is empty")
	}
	if tokenType == "" {
		tokenType = "Bearer"
	}

	tok := &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   tokenType,
	}
	// Inject scope into Extra so extractGrantedScopes can pick it up.
	if scope != "" {
		tok = tok.WithExtra(map[string]interface{}{"scope": scope})
	}
	return tok, nil
}

// oauthConfig builds an *oauth2.Config from a VaultProvider, decrypting the
// client secret on the fly. The redirectURI and scopes are request-time
// overrides; pass "" / nil to let the caller handle them elsewhere.
func (m *Manager) oauthConfig(provider *storage.VaultProvider, redirectURI string, scopes []string) (*oauth2.Config, error) {
	clientSecret, err := m.encryptor.Decrypt(provider.ClientSecretEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypt client_secret: %w", err)
	}
	effectiveScopes := scopes
	if len(effectiveScopes) == 0 {
		effectiveScopes = provider.Scopes
	}
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.AuthURL,
			TokenURL: provider.TokenURL,
		},
		RedirectURL: redirectURI,
		Scopes:      effectiveScopes,
	}, nil
}

// isExpired returns true when exp is either unset (treat as expired so we
// don't keep a stale token forever) or when now+leeway is past it.
func (m *Manager) isExpired(exp *time.Time) bool {
	if exp == nil {
		// No expiry recorded: we can't know if it's good. Treat as fresh â€”
		// some providers issue non-expiring tokens (e.g. Slack bot tokens).
		// Callers who need strict expiry must set ExpiresAt during exchange.
		return false
	}
	return m.now().Add(expiryLeeway).After(*exp)
}

// extractGrantedScopes pulls the scope list out of the token response. The
// oauth2 library stashes the raw "scope" string under Extra when the server
// returns it; fall back to the requested scopes when the server stays silent.
func extractGrantedScopes(token *oauth2.Token, requested []string) []string {
	if token == nil {
		return requested
	}
	raw, _ := token.Extra("scope").(string)
	out := strings.Fields(raw)
	if len(out) == 0 {
		return requested
	}
	return out
}

// newID returns "<prefix><24 hex chars>" â€” matches the pattern used by
// internal/sso and internal/user for consistency across Shark entities.
func newID(prefix string) (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}

// isNoRows reports whether err is (or wraps) sql.ErrNoRows. Storage
// implementations that follow database/sql's convention for missing rows
// return this sentinel from scan helpers.
func isNoRows(err error) bool {
	return err != nil && errors.Is(err, sql.ErrNoRows)
}
