// Package jwt implements RS256 JWT issuance, validation, and key management
// for SharkAuth. Algorithms other than RS256 are rejected at parse time to
// prevent alg-confusion attacks (RFC 8725 §2.1).
package jwt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Sentinel errors returned by Validate and related methods.
var (
	ErrExpired          = errors.New("jwt: token expired")
	ErrInvalidSignature = errors.New("jwt: invalid signature")
	ErrRevoked          = errors.New("jwt: token revoked")
	ErrUnknownKid       = errors.New("jwt: unknown kid")
	ErrAlgMismatch      = errors.New("jwt: algorithm mismatch")
	// ErrRefreshToken is returned when a refresh token is presented where an
	// access/session token is required. Callers (e.g. middleware) can detect this
	// specific error to return an actionable error_description to the client (§2.3).
	ErrRefreshToken = errors.New("jwt: refresh token cannot be used as access credential")
)

// Claims is the combined registered + custom claim set.
type Claims struct {
	gojwt.RegisteredClaims
	MFAPassed bool   `json:"mfa_passed"`
	SessionID string `json:"session_id"`
	TokenType string `json:"token_type"` // "session" | "access" | "refresh"
}

// Manager handles JWT lifecycle: issuance, validation, key management.
type Manager struct {
	cfg    *config.JWTConfig
	store  storage.Store
	base   string // server.base_url — fallback issuer
	secret string // server.secret — used for AES-GCM key derivation
}

// NewManager creates a Manager. baseURL is used as the fallback issuer when
// cfg.Issuer is empty.
func NewManager(cfg *config.JWTConfig, store storage.Store, baseURL, serverSecret string) *Manager {
	return &Manager{
		cfg:    cfg,
		store:  store,
		base:   baseURL,
		secret: serverSecret,
	}
}

// issuer returns the effective JWT issuer.
func (m *Manager) issuer() string {
	if m.cfg.Issuer != "" {
		return m.cfg.Issuer
	}
	return m.base
}

// SetCheckPerRequest enables or disables the per-request revocation check at
// runtime. Useful in tests and for dynamic config reloads.
func (m *Manager) SetCheckPerRequest(enabled bool) {
	m.cfg.Revocation.CheckPerRequest = enabled
}

// EnsureActiveKey generates and stores an RSA keypair if none exists.
// Called at server startup so manual `shark keys generate-jwt` is optional.
func (m *Manager) EnsureActiveKey(ctx context.Context) error {
	_, err := m.store.GetActiveSigningKey(ctx)
	if err == nil {
		return nil // already exists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("checking active signing key: %w", err)
	}

	return m.GenerateAndStore(ctx, false)
}

// GenerateAndStore creates a new RSA keypair and persists it.
// If rotate is true, all current active keys are marked retired first.
func (m *Manager) GenerateAndStore(ctx context.Context, rotate bool) error {
	priv, err := generateRSAKeypair()
	if err != nil {
		return fmt.Errorf("generate rsa keypair: %w", err)
	}

	kid, err := computeKID(&priv.PublicKey)
	if err != nil {
		return fmt.Errorf("compute kid: %w", err)
	}

	pubPEM, err := encodePublicKeyPEM(&priv.PublicKey)
	if err != nil {
		return fmt.Errorf("encode public key: %w", err)
	}

	privPEMBytes, err := encodePrivateKeyPEM(priv)
	if err != nil {
		return fmt.Errorf("encode private key: %w", err)
	}

	encPrivPEM, err := encryptPEM(privPEMBytes, m.secret)
	if err != nil {
		return fmt.Errorf("encrypt private key: %w", err)
	}

	key := &storage.SigningKey{
		KID:           kid,
		Algorithm:     "RS256",
		PublicKeyPEM:  pubPEM,
		PrivateKeyPEM: encPrivPEM,
		Status:        "active",
	}

	if rotate {
		return m.store.RotateSigningKeys(ctx, key)
	}
	return m.store.InsertSigningKey(ctx, key)
}

// newJTI generates a 16-byte random hex JTI.
func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// issueToken signs a Claims struct with the active key and returns a compact JWT.
func (m *Manager) issueToken(ctx context.Context, claims Claims) (string, error) {
	key, err := m.store.GetActiveSigningKey(ctx)
	if err != nil {
		return "", fmt.Errorf("get active signing key: %w", err)
	}

	priv, err := decryptPrivateKey(key.PrivateKeyPEM, m.secret)
	if err != nil {
		return "", fmt.Errorf("decrypt private key: %w", err)
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	token.Header["kid"] = key.KID

	signed, err := token.SignedString(priv)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// IssueSessionJWT issues a single session JWT (mode="session").
// The token is valid for the session lifetime (auth.session_lifetime).
func (m *Manager) IssueSessionJWT(ctx context.Context, user *storage.User, sessionID string, mfaPassed bool) (string, error) {
	jti, err := newJTI()
	if err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now().UTC()
	// Use session lifetime (30d default), same as SessionManager
	ttl := 30 * 24 * time.Hour
	aud := gojwt.ClaimStrings{m.cfg.Audience}

	claims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer(),
			Subject:   user.ID,
			Audience:  aud,
			ExpiresAt: gojwt.NewNumericDate(now.Add(ttl)),
			NotBefore: gojwt.NewNumericDate(now),
			IssuedAt:  gojwt.NewNumericDate(now),
			ID:        jti,
		},
		MFAPassed: mfaPassed,
		SessionID: sessionID,
		TokenType: "session",
	}

	return m.issueToken(ctx, claims)
}

// IssueAccessRefreshPair issues an access + refresh pair (mode="access_refresh").
func (m *Manager) IssueAccessRefreshPair(ctx context.Context, user *storage.User, sessionID string, mfaPassed bool) (access, refresh string, err error) {
	accessJTI, err := newJTI()
	if err != nil {
		return "", "", fmt.Errorf("generate access jti: %w", err)
	}
	refreshJTI, err := newJTI()
	if err != nil {
		return "", "", fmt.Errorf("generate refresh jti: %w", err)
	}

	now := time.Now().UTC()
	accessTTL := m.cfg.AccessTokenTTLDuration()
	refreshTTL := m.cfg.RefreshTokenTTLDuration()
	aud := gojwt.ClaimStrings{m.cfg.Audience}

	accessClaims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer(),
			Subject:   user.ID,
			Audience:  aud,
			ExpiresAt: gojwt.NewNumericDate(now.Add(accessTTL)),
			NotBefore: gojwt.NewNumericDate(now),
			IssuedAt:  gojwt.NewNumericDate(now),
			ID:        accessJTI,
		},
		MFAPassed: mfaPassed,
		SessionID: sessionID,
		TokenType: "access",
	}

	refreshClaims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer(),
			Subject:   user.ID,
			Audience:  aud,
			ExpiresAt: gojwt.NewNumericDate(now.Add(refreshTTL)),
			NotBefore: gojwt.NewNumericDate(now),
			IssuedAt:  gojwt.NewNumericDate(now),
			ID:        refreshJTI,
		},
		MFAPassed: mfaPassed,
		SessionID: sessionID,
		TokenType: "refresh",
	}

	access, err = m.issueToken(ctx, accessClaims)
	if err != nil {
		return "", "", fmt.Errorf("issue access token: %w", err)
	}

	refresh, err = m.issueToken(ctx, refreshClaims)
	if err != nil {
		return "", "", fmt.Errorf("issue refresh token: %w", err)
	}

	return access, refresh, nil
}

// Refresh validates a refresh token, revokes its JTI (one-time-use),
// and issues a new access+refresh pair.
// The one-time-use revocation check runs regardless of check_per_request.
func (m *Manager) Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error) {
	// We validate here with a relaxed token-type check — we need to allow
	// refresh tokens through for this specific operation.
	claims, err := m.validateInternal(ctx, refreshToken, true)
	if err != nil {
		return "", "", err
	}

	if claims.TokenType != "refresh" {
		return "", "", fmt.Errorf("expected refresh token, got %q", claims.TokenType)
	}

	// One-time-use check: always verify revocation for refresh tokens,
	// regardless of the global check_per_request setting.
	_ = m.store.PruneExpiredRevokedJTI(ctx)
	revoked, err := m.store.IsRevokedJTI(ctx, claims.ID)
	if err != nil {
		return "", "", fmt.Errorf("check refresh revocation: %w", err)
	}
	if revoked {
		return "", "", ErrRevoked
	}

	// One-time-use: revoke old refresh JTI BEFORE issuing new pair.
	if err := m.store.InsertRevokedJTI(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		return "", "", fmt.Errorf("revoke old refresh jti: %w", err)
	}

	user := &storage.User{ID: claims.Subject}
	return m.IssueAccessRefreshPair(ctx, user, claims.SessionID, claims.MFAPassed)
}

// Validate parses and validates a JWT. Returns Claims on success, or a sentinel
// error from the Err* set. Validation order per PHASE3.md §1.5.
func (m *Manager) Validate(ctx context.Context, token string) (*Claims, error) {
	return m.validateInternal(ctx, token, false)
}

// validateInternal is the shared implementation. allowRefresh controls whether
// refresh tokens pass the token-type check (used only by Refresh itself).
func (m *Manager) validateInternal(ctx context.Context, tokenStr string, allowRefresh bool) (*Claims, error) {
	// Step 1: parse header WITHOUT signature verification and reject alg!=RS256.
	if err := peekAlg(tokenStr); err != nil {
		return nil, err
	}

	// Step 2: extract kid from header.
	kid, err := peekKID(tokenStr)
	if err != nil {
		return nil, ErrUnknownKid
	}

	// Step 3: look up key (active or retired).
	signingKey, err := m.store.GetSigningKeyByKID(ctx, kid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnknownKid
		}
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	// Step 4: parse public key PEM (plaintext column).
	pubKey, err := decodePublicKeyPEM(signingKey.PublicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	// Step 5 & 6: verify signature, exp/nbf/iat, iss, aud.
	clockSkew := m.cfg.ClockSkewDuration()
	parser := gojwt.NewParser(
		gojwt.WithValidMethods([]string{"RS256"}),
		gojwt.WithLeeway(clockSkew),
		gojwt.WithIssuedAt(),
		gojwt.WithIssuer(m.issuer()),
		gojwt.WithAudience(m.cfg.Audience),
	)

	var claims Claims
	_, parseErr := parser.ParseWithClaims(tokenStr, &claims, func(t *gojwt.Token) (interface{}, error) {
		return pubKey, nil
	})

	if parseErr != nil {
		if errors.Is(parseErr, gojwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		if errors.Is(parseErr, gojwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidSignature, parseErr)
	}

	// Token type enforcement.
	if claims.TokenType == "refresh" && !allowRefresh {
		// Return a specific sentinel so the middleware can report an actionable
		// error_description rather than a generic "invalid_token" (§2.3).
		return nil, ErrRefreshToken
	}
	allowed := map[string]bool{"session": true, "access": true, "refresh": allowRefresh}
	if !allowed[claims.TokenType] {
		return nil, fmt.Errorf("invalid token_type %q", claims.TokenType)
	}

	// Step 7: optional per-request revocation check.
	if m.cfg.Revocation.CheckPerRequest {
		_ = m.store.PruneExpiredRevokedJTI(ctx)
		revoked, err := m.store.IsRevokedJTI(ctx, claims.ID)
		if err != nil {
			return nil, fmt.Errorf("check revocation: %w", err)
		}
		if revoked {
			return nil, ErrRevoked
		}
	}

	return &claims, nil
}

// peekAlg decodes the JWT header and rejects alg="none" and all HMAC variants.
// This is a manual step before the library parser to prevent alg-confusion.
func peekAlg(tokenStr string) error {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return ErrAlgMismatch
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrAlgMismatch
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return ErrAlgMismatch
	}

	alg := strings.ToUpper(header.Alg)
	switch alg {
	case "RS256", "ES256":
		return nil
	case "NONE", "":
		return ErrAlgMismatch
	default:
		// Reject all HMAC variants and anything else
		if strings.HasPrefix(alg, "HS") {
			return ErrAlgMismatch
		}
		return ErrAlgMismatch
	}
}

// peekKID decodes the JWT header and extracts the kid claim.
func peekKID(tokenStr string) (string, error) {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return "", ErrUnknownKid
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrUnknownKid
	}

	var header struct {
		KID string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", ErrUnknownKid
	}

	if header.KID == "" {
		return "", ErrUnknownKid
	}
	return header.KID, nil
}
