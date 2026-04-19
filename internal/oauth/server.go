// Package oauth provides the OAuth 2.1 Authorization Server backed by fosite.
package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/handler/openid"
	"github.com/ory/fosite/token/jwt"

	authjwt "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Server holds the fosite OAuth2 provider and dependencies.
type Server struct {
	Provider       fosite.OAuth2Provider
	Store          *FositeStore
	Config         *config.OAuthServerConfig
	Issuer         string
	RawStore       storage.Store        // for direct DB access (consent, agents, etc.)
	SigningKeyID   string               // kid of the active ES256 signing key
	signingPrivKey *ecdsa.PrivateKey    // ES256 private key; used by Sign() / token exchange
	DPoPCache      *DPoPJTICache        // replay protection for DPoP JTIs
}

// NewServer creates an OAuth 2.1 server. It manages its own ES256 signing key
// (separate from the RS256 key used for session JWTs) by checking for an active
// ES256 key in jwt_signing_keys and generating one if none exists.
func NewServer(store storage.Store, cfg *config.Config) (*Server, error) {
	issuer := cfg.OAuthServer.Issuer
	if issuer == "" {
		issuer = cfg.Server.BaseURL
	}

	// --- ES256 key management ---
	signingKey, err := ensureES256Key(store, cfg.Server.Secret)
	if err != nil {
		return nil, fmt.Errorf("oauth: ES256 key: %w", err)
	}

	kid := authjwt.ComputeES256KID(&signingKey.PublicKey)

	// Create fosite store adapter.
	fositeStore := NewFositeStore(store)

	// Build HMAC global secret from the server secret.
	globalSecret := sha256.Sum256([]byte(cfg.Server.Secret))

	// Configure fosite.
	fositeConfig := &fosite.Config{
		AccessTokenLifespan:            cfg.OAuthServer.AccessTokenLifetimeDuration(),
		RefreshTokenLifespan:           cfg.OAuthServer.RefreshTokenLifetimeDuration(),
		AuthorizeCodeLifespan:          cfg.OAuthServer.AuthCodeLifetimeDuration(),
		EnforcePKCE:                    true,
		EnforcePKCEForPublicClients:    true,
		EnablePKCEPlainChallengeMethod: false,
		TokenURL:                       issuer + "/oauth/token",
		GlobalSecret:                   globalSecret[:],
		SendDebugMessagesToClients:     false,
		ClientSecretsHasher:            &SHA256Hasher{},
	}

	// Build a jose.JSONWebKey for fosite's JWT strategy.
	jwk := jose.JSONWebKey{
		Key:       signingKey,
		KeyID:     kid,
		Algorithm: "ES256",
		Use:       "sig",
	}

	// Build the HMAC core strategy (for opaque token signatures).
	hmacStrategy := compose.NewOAuth2HMACStrategy(fositeConfig)

	// Build the JWT signer keyed to the ES256 key.
	keyGetter := func(_ context.Context) (interface{}, error) {
		return jwk.Key, nil
	}
	signer := &jwt.DefaultSigner{GetPrivateKey: keyGetter}

	// Compose the strategy: HMAC for token signatures + JWT signer.
	strategy := &compose.CommonStrategy{
		CoreStrategy: hmacStrategy,
		Signer:       signer,
	}

	// Compose the OAuth2 provider with the desired grant factories.
	provider := compose.Compose(
		fositeConfig,
		fositeStore,
		strategy,
		compose.OAuth2AuthorizeExplicitFactory,
		compose.OAuth2ClientCredentialsGrantFactory,
		compose.OAuth2RefreshTokenGrantFactory,
		compose.OAuth2PKCEFactory,
		compose.OAuth2TokenRevocationFactory,
		compose.OAuth2TokenIntrospectionFactory,
	)

	return &Server{
		Provider:       provider,
		Store:          fositeStore,
		Config:         &cfg.OAuthServer,
		Issuer:         issuer,
		RawStore:       store,
		SigningKeyID:   kid,
		signingPrivKey: signingKey,
		DPoPCache:      NewDPoPJTICache(),
	}, nil
}

// ensureES256Key checks for an active ES256 signing key in the database.
// If none exists, it generates one, encrypts the private key, and stores it.
func ensureES256Key(store storage.Store, serverSecret string) (*ecdsa.PrivateKey, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check for existing active ES256 key.
	existing, err := store.GetActiveSigningKeyByAlgorithm(ctx, "ES256")
	if err == nil {
		// Decrypt and return the existing key.
		pemBytes, decErr := authjwt.DecryptPEM(existing.PrivateKeyPEM, serverSecret)
		if decErr == nil {
			defer func() {
				for i := range pemBytes {
					pemBytes[i] = 0
				}
			}()
			key, parseErr := authjwt.ParseES256PrivateKeyPEM(pemBytes)
			if parseErr != nil {
				return nil, fmt.Errorf("parse existing ES256 key: %w", parseErr)
			}
			slog.Info("oauth: loaded existing ES256 signing key", "kid", existing.KID)
			return key, nil
		}
		// Decrypt failed — most likely server.secret changed (dev mode rotates
		// on each boot). Retire the un-decryptable key and fall through to
		// generate a fresh one so /oauth/* stays functional.
		slog.Warn("oauth: retiring un-decryptable ES256 key (secret mismatch); generating fresh key",
			"kid", existing.KID, "error", decErr)
		now := time.Now().UTC().Format(time.RFC3339)
		if _, retErr := store.DB().ExecContext(ctx,
			`UPDATE jwt_signing_keys SET status = 'retired', rotated_at = ? WHERE kid = ?`,
			now, existing.KID,
		); retErr != nil {
			slog.Warn("oauth: failed to retire stale ES256 key", "error", retErr)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup active ES256 key: %w", err)
	}

	// Generate a new ES256 key.
	slog.Info("oauth: generating new ES256 signing key")
	priv, _, genErr := authjwt.GenerateES256Keypair()
	if genErr != nil {
		return nil, fmt.Errorf("generate ES256 keypair: %w", genErr)
	}

	kid := authjwt.ComputeES256KID(&priv.PublicKey)

	pubPEM, err := authjwt.MarshalES256PublicKeyPEM(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshal ES256 public key: %w", err)
	}

	privPEM, err := authjwt.MarshalES256PrivateKeyPEM(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal ES256 private key: %w", err)
	}

	encPrivPEM, err := authjwt.EncryptPEM(privPEM, serverSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt ES256 private key: %w", err)
	}

	signingKey := &storage.SigningKey{
		KID:           kid,
		Algorithm:     "ES256",
		PublicKeyPEM:  string(pubPEM),
		PrivateKeyPEM: encPrivPEM,
		Status:        "active",
	}

	if err := store.InsertSigningKey(ctx, signingKey); err != nil {
		return nil, fmt.Errorf("store ES256 signing key: %w", err)
	}

	slog.Info("oauth: ES256 signing key generated and stored", "kid", kid)
	return priv, nil
}

// newSession creates a fosite session for the given subject.
func (s *Server) newSession(subject string) *openid.DefaultSession {
	return &openid.DefaultSession{
		Claims: &jwt.IDTokenClaims{
			Issuer:  s.Issuer,
			Subject: subject,
		},
		Headers: &jwt.Headers{
			Extra: map[string]interface{}{
				"kid": s.SigningKeyID,
			},
		},
		Subject: subject,
	}
}
