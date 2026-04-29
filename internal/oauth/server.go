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

	authjwt "github.com/shark-auth/shark/internal/auth/jwt"
	"github.com/shark-auth/shark/internal/audit"
	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/storage"
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
	AuditLogger    *audit.Logger        // nil-safe; writes oauth.token.exchanged rows
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
		// DX1: emit RFC 7519 JWT access tokens signed by the ES256 JWKS key.
		AccessTokenIssuer: issuer,
		JWTScopeClaimKey:  jwt.JWTScopeFieldString, // "scope" as space-separated string
	}

	// Build a jose.JSONWebKey for fosite's JWT strategy.
	jwk := jose.JSONWebKey{
		Key:       signingKey,
		KeyID:     kid,
		Algorithm: "ES256",
		Use:       "sig",
	}

	// Build the HMAC core strategy (still used for refresh-token and
	// authorize-code signatures â€” refresh tokens stay opaque per DX1 scope).
	hmacStrategy := compose.NewOAuth2HMACStrategy(fositeConfig)

	// DX1: build the JWT access-token strategy keyed to the ES256 JWKS key.
	// Access tokens become RFC 7519 JWTs; refresh tokens + auth codes remain
	// opaque (JWT strategy delegates those to the wrapped HMAC strategy).
	keyGetter := func(_ context.Context) (interface{}, error) {
		return jwk.Key, nil
	}
	jwtAccessStrategy := compose.NewOAuth2JWTStrategy(keyGetter, hmacStrategy, fositeConfig)

	// Compose the strategy: JWT for access tokens (which delegates to HMAC
	// for refresh + auth-code), plus the JWT signer for ID tokens.
	strategy := &compose.CommonStrategy{
		CoreStrategy: jwtAccessStrategy,
		Signer:       &jwt.DefaultSigner{GetPrivateKey: keyGetter},
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
	var staleKID string
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
		// Decrypt failed â€” most likely server.secret changed (dev mode rotates
		// on each boot). Defer retiring the un-decryptable key until after the
		// replacement key has been successfully inserted, so /oauth/* stays
		// functional even if the insert fails.
		slog.Warn("oauth: will retire un-decryptable ES256 key (secret mismatch); generating fresh key",
			"kid", existing.KID, "error", decErr)
		staleKID = existing.KID
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

	// Persist retire-old + insert-new atomically in a single transaction when we
	// have a stale key to replace. Either both happen or neither does, so we
	// never retire the old key without a working replacement. If no stale key
	// exists, just insert the new one directly.
	if staleKID != "" {
		tx, txErr := store.DB().BeginTx(ctx, nil)
		if txErr != nil {
			return nil, fmt.Errorf("begin ES256 rotation tx: %w", txErr)
		}
		defer tx.Rollback() //nolint:errcheck

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO jwt_signing_keys (kid, algorithm, public_key_pem, private_key_pem, status)
			 VALUES (?, ?, ?, ?, 'active')`,
			kid, "ES256", string(pubPEM), encPrivPEM,
		); err != nil {
			return nil, fmt.Errorf("store ES256 signing key: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx,
			`UPDATE jwt_signing_keys SET status = 'retired', rotated_at = ? WHERE kid = ?`,
			now, staleKID,
		); err != nil {
			return nil, fmt.Errorf("retire stale ES256 key: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit ES256 rotation: %w", err)
		}
		slog.Info("oauth: retired stale ES256 key", "old_kid", staleKID, "new_kid", kid)
	} else {
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
	}

	slog.Info("oauth: ES256 signing key generated and stored", "kid", kid)
	return priv, nil
}

// newSession creates a fosite session for the given subject. Returns a
// *SharkSession which is simultaneously an OpenID session (for ID-token
// grants) and an oauth2.JWTSessionContainer (so the DX1 JWT access-token
// strategy can mint RFC 7519 access tokens signed by the ES256 JWKS key).
func (s *Server) newSession(subject string) *SharkSession {
	kidHeader := &jwt.Headers{
		Extra: map[string]interface{}{
			"kid": s.SigningKeyID,
		},
	}
	return &SharkSession{
		DefaultSession: &openid.DefaultSession{
			Claims: &jwt.IDTokenClaims{
				Issuer:  s.Issuer,
				Subject: subject,
			},
			Headers: kidHeader,
			Subject: subject,
		},
		JWTClaims: &jwt.JWTClaims{
			Subject: subject,
			Issuer:  s.Issuer,
			Extra:   map[string]interface{}{},
		},
		JWTHeader: &jwt.Headers{
			Extra: map[string]interface{}{
				"kid": s.SigningKeyID,
			},
		},
	}
}
