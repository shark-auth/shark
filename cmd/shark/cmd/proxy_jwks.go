package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/sharkauth/sharkauth/internal/proxy"
)

// jwksCache fetches and caches a Shark auth server's /.well-known/jwks.json
// for the standalone `shark proxy` mode. Refreshes every 15 minutes; also
// forces a refresh on a kid miss (new key rotated in between scheduled
// refreshes) before giving up and returning an invalid-token error.
//
// This is intentionally minimal — it reuses golang-jwt/jwt/v5 for parsing
// and verification, but speaks JWKS on the wire so the standalone binary
// doesn't need any DB or shared process memory with the auth server.
type jwksCache struct {
	baseURL         string
	httpClient      *http.Client
	refreshInterval time.Duration
	logger          *slog.Logger

	// W15c: aud + iss are NOT validated by golang-jwt/v5 automatically on
	// MapClaims — only exp/nbf/iat are. Callers must populate these so
	// verifyBearer can pass WithAudience + WithIssuer to the parser. A
	// token whose aud isn't in expectedAudiences or whose iss differs
	// from expectedIssuer is rejected. Empty expectedAudiences disables
	// the aud check (not recommended — the proxy CLI requires --audience
	// at startup); empty expectedIssuer disables the iss check.
	expectedAudiences []string
	expectedIssuer    string

	mu       sync.RWMutex
	keys     map[string]any // kid -> *rsa.PublicKey or *ecdsa.PublicKey
	fetched  time.Time
	inFlight sync.Mutex // serialises concurrent refreshes
}

type jwksDoc struct {
	Keys []jwksKey `json:"keys"`
}

type jwksKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	// RSA
	N string `json:"n"`
	E string `json:"e"`
	// EC
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func newJWKSCache(baseURL string, logger *slog.Logger) *jwksCache {
	return &jwksCache{
		baseURL:         strings.TrimRight(baseURL, "/"),
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		refreshInterval: 15 * time.Minute,
		logger:          logger,
		keys:            map[string]any{},
	}
}

// Start boots a background refresh goroutine. Returns after the initial
// synchronous fetch so callers can fail fast on misconfigured --auth URLs.
func (c *jwksCache) Start(ctx context.Context) error {
	if err := c.refresh(ctx); err != nil {
		return fmt.Errorf("jwks initial fetch: %w", err)
	}
	go func() {
		t := time.NewTicker(c.refreshInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if err := c.refresh(ctx); err != nil {
					c.logger.Warn("jwks refresh failed", "err", err)
				}
			}
		}
	}()
	return nil
}

func (c *jwksCache) refresh(ctx context.Context) error {
	c.inFlight.Lock()
	defer c.inFlight.Unlock()

	url := c.baseURL + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //#nosec G307 -- response drain
	if resp.StatusCode != 200 {
		return fmt.Errorf("jwks %s: status %d", url, resp.StatusCode)
	}

	var doc jwksDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("jwks decode: %w", err)
	}

	keys := map[string]any{}
	for _, k := range doc.Keys {
		pk, err := parseJWK(k)
		if err != nil {
			c.logger.Warn("jwks: skipping malformed key", "kid", k.Kid, "err", err)
			continue
		}
		keys[k.Kid] = pk
	}
	c.mu.Lock()
	c.keys = keys
	c.fetched = time.Now()
	c.mu.Unlock()
	c.logger.Debug("jwks refreshed", "keys", len(keys))
	return nil
}

// keyForKid returns the cached key, force-refreshing once on a miss in case
// the auth server rotated keys since the last scheduled refresh.
func (c *jwksCache) keyForKid(ctx context.Context, kid string) (any, error) {
	c.mu.RLock()
	if k, ok := c.keys[kid]; ok {
		c.mu.RUnlock()
		return k, nil
	}
	c.mu.RUnlock()

	if err := c.refresh(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown kid %q", kid)
}

func parseJWK(k jwksKey) (any, error) {
	switch k.Kty {
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("n: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("e: %w", err)
		}
		e := 0
		for _, b := range eBytes {
			e = e<<8 | int(b)
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: e,
		}, nil
	case "EC":
		var curve elliptic.Curve
		switch k.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported curve %q", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, fmt.Errorf("x: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, fmt.Errorf("y: %w", err)
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported kty %q", k.Kty)
	}
}

// verifyBearer parses and verifies the Authorization: Bearer <jwt> header.
// Returns a fully-populated proxy.Identity on success. Errors bubble
// through so the caller can map them all to 401 regardless of cause
// (missing header, bad signature, expired, unknown kid).
func (c *jwksCache) verifyBearer(ctx context.Context, authHeader string) (proxy.Identity, error) {
	if authHeader == "" {
		return proxy.Identity{}, errors.New("missing Authorization header")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return proxy.Identity{}, errors.New("Authorization must be Bearer")
	}
	tok := strings.TrimPrefix(authHeader, "Bearer ")

	opts := []jwtlib.ParserOption{
		jwtlib.WithValidMethods([]string{"RS256", "ES256"}),
	}
	// W15c: audience + issuer validation. golang-jwt v5 only enforces
	// these when explicitly opted in via WithAudience/WithIssuer. Without
	// them a valid token minted by the auth server for a different
	// audience or issuer would be accepted — CVE-shape.
	//
	// Audience is checked as a set-intersection: the token is accepted
	// when any of its `aud` values matches any of the expected audiences.
	// Implemented by looping over the configured audiences and using
	// WithAudience one-at-a-time so multi-audience tokens Just Work.
	// (jwt/v5 WithAudience accepts a single string; passing them all in
	// one call would require ALL to match, which is wrong.)
	if c.expectedIssuer != "" {
		opts = append(opts, jwtlib.WithIssuer(c.expectedIssuer))
	}
	parser := jwtlib.NewParser(opts...)
	parsed, err := parser.Parse(tok, func(t *jwtlib.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("kid header required")
		}
		return c.keyForKid(ctx, kid)
	})
	if err != nil {
		return proxy.Identity{}, err
	}
	if !parsed.Valid {
		return proxy.Identity{}, errors.New("token invalid")
	}

	claims, ok := parsed.Claims.(jwtlib.MapClaims)
	if !ok {
		return proxy.Identity{}, errors.New("unexpected claims type")
	}

	// W15c: manual audience check. We do NOT use jwtlib.WithAudience
	// because it only accepts one expected value and requires it to
	// appear in the token's aud claim; with multiple configured
	// audiences we'd need set-intersection semantics instead. Accept
	// the token when ANY token aud matches ANY configured audience.
	if len(c.expectedAudiences) > 0 {
		tokenAuds, err := extractAudiences(claims)
		if err != nil {
			return proxy.Identity{}, err
		}
		if !audienceMatches(tokenAuds, c.expectedAudiences) {
			return proxy.Identity{}, fmt.Errorf("audience mismatch: token aud=%v, expected one of %v", tokenAuds, c.expectedAudiences)
		}
	}

	id := proxy.Identity{AuthMethod: "jwt"}
	if sub, _ := claims["sub"].(string); sub != "" {
		id.UserID = sub
	}
	if email, _ := claims["email"].(string); email != "" {
		id.UserEmail = email
	}
	// "roles" and "scopes" may come through as []any from JSON decode.
	if raw, ok := claims["roles"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				id.UserRoles = append(id.UserRoles, s)
			}
		}
	}
	if raw, ok := claims["scope"].(string); ok && raw != "" {
		id.Scopes = strings.Fields(raw)
	} else if raw, ok := claims["scopes"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				id.Scopes = append(id.Scopes, s)
			}
		}
	}
	return id, nil
}

// extractAudiences pulls the aud claim from a MapClaims into a []string.
// JWT allows aud to be either a single string or a list of strings (RFC
// 7519 §4.1.3). Returns an error if aud is missing or malformed so the
// caller can reject rather than silently treat a token with no aud as
// matching — which would defeat the purpose of the W15c check.
func extractAudiences(claims jwtlib.MapClaims) ([]string, error) {
	raw, ok := claims["aud"]
	if !ok {
		return nil, errors.New("aud claim missing")
	}
	switch v := raw.(type) {
	case string:
		if v == "" {
			return nil, errors.New("aud claim empty")
		}
		return []string{v}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil, errors.New("aud claim has no string values")
		}
		return out, nil
	default:
		return nil, fmt.Errorf("aud claim has unexpected type %T", raw)
	}
}

// audienceMatches returns true when any value in tokenAuds appears in
// expectedAuds. Set-intersection semantics — the token is "good enough"
// if any of its audiences is one the proxy was told to trust.
func audienceMatches(tokenAuds, expectedAuds []string) bool {
	for _, t := range tokenAuds {
		for _, e := range expectedAuds {
			if t == e {
				return true
			}
		}
	}
	return false
}
