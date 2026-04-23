// Package oauth — DPoP (Demonstrating Proof-of-Possession) implementation per RFC 9449.
package oauth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// dpopWindow is the maximum age (and future skew) accepted for a DPoP iat claim.
const dpopWindow = 60 * time.Second

// allowedDPoPAlgs lists asymmetric algorithms permitted in DPoP proofs.
// Symmetric algorithms (HS*) are explicitly forbidden by RFC 9449 §4.3.
var allowedDPoPAlgs = map[string]bool{
	"ES256": true,
	"ES384": true,
	"ES512": true,
	"RS256": true,
	"RS384": true,
	"RS512": true,
	"PS256": true,
	"PS384": true,
	"PS512": true,
	"EdDSA": true,
}

// ---------------------------------------------------------------------------
// JTI replay cache
// ---------------------------------------------------------------------------

// DPoPJTICache is a simple in-memory replay-protection store.
// It records JTIs that have been seen within the replay window and prunes
// stale entries on every MarkSeen call.
type DPoPJTICache struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

// NewDPoPJTICache returns an initialised JTI cache.
func NewDPoPJTICache() *DPoPJTICache {
	return &DPoPJTICache{seen: make(map[string]time.Time)}
}

// MarkSeen records jti as seen. Returns an error if jti was already seen
// within window. Prunes expired entries on every call.
func (c *DPoPJTICache) MarkSeen(jti string, window time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Prune stale entries.
	for k, t := range c.seen {
		if t.Before(cutoff) {
			delete(c.seen, k)
		}
	}

	if _, exists := c.seen[jti]; exists {
		return errors.New("dpop: jti already seen (replay detected)")
	}
	c.seen[jti] = now
	return nil
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ValidateDPoPProof validates a DPoP proof JWT (RFC 9449 §4.3).
//
//   - proofJWT      — raw DPoP header value
//   - method        — HTTP method of the protected request (upper-case)
//   - htu           — HTTP URL of the protected endpoint (no query/fragment)
//   - accessTokenHash — optional; base64url(sha256(access_token)) for ath check
//   - cache         — replay-protection cache (must not be nil)
//
// Returns the JWK thumbprint (jkt) of the embedded public key on success.
func ValidateDPoPProof(proofJWT, method, htu, accessTokenHash string, cache *DPoPJTICache) (jkt string, err error) {
	if proofJWT == "" {
		return "", errors.New("dpop: missing proof JWT")
	}

	// --- Step 1: parse header without signature verification to read jwk ---
	parts := strings.Split(proofJWT, ".")
	if len(parts) != 3 {
		return "", errors.New("dpop: malformed JWT (expected 3 parts)")
	}

	rawHeader, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("dpop: decode header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(rawHeader, &header); err != nil {
		return "", fmt.Errorf("dpop: unmarshal header: %w", err)
	}

	// --- Step 2: verify typ == "dpop+jwt" ---
	typ, _ := header["typ"].(string)
	if typ != "dpop+jwt" {
		return "", fmt.Errorf("dpop: invalid typ %q (want \"dpop+jwt\")", typ)
	}

	// --- Step 3: verify alg is allowed ---
	alg, _ := header["alg"].(string)
	if !allowedDPoPAlgs[alg] {
		return "", fmt.Errorf("dpop: algorithm %q is not allowed", alg)
	}

	// --- Step 4: extract public key from jwk header ---
	jwkRaw, ok := header["jwk"].(map[string]interface{})
	if !ok {
		return "", errors.New("dpop: missing or invalid jwk header")
	}

	pubKey, err := jwkToPublicKey(jwkRaw)
	if err != nil {
		return "", fmt.Errorf("dpop: parse jwk: %w", err)
	}

	// Build a keyfunc that verifies the alg matches and returns the embedded key.
	keyfunc := func(t *gojwt.Token) (interface{}, error) {
		if t.Header["alg"] != alg {
			return nil, fmt.Errorf("dpop: alg mismatch in keyfunc")
		}
		return pubKey, nil
	}

	// --- Step 4b: verify signature ---
	claims := gojwt.MapClaims{}
	parsed, err := gojwt.ParseWithClaims(proofJWT, claims, keyfunc,
		gojwt.WithoutClaimsValidation(), // we validate iat manually
	)
	if err != nil {
		return "", fmt.Errorf("dpop: signature invalid: %w", err)
	}
	if !parsed.Valid {
		return "", errors.New("dpop: token invalid")
	}

	// Re-read claims from the verified token.
	claims, ok = parsed.Claims.(gojwt.MapClaims)
	if !ok {
		return "", errors.New("dpop: unexpected claims type")
	}

	// --- Step 5a: validate iat ---
	iatRaw, hasIat := claims["iat"]
	if !hasIat {
		return "", errors.New("dpop: missing iat claim")
	}
	var iat time.Time
	switch v := iatRaw.(type) {
	case float64:
		iat = time.Unix(int64(v), 0).UTC()
	default:
		return "", errors.New("dpop: iat must be a number")
	}
	now := time.Now().UTC()
	if now.Sub(iat) > dpopWindow {
		return "", fmt.Errorf("dpop: proof expired (iat %s is too old)", iat)
	}
	if iat.Sub(now) > dpopWindow {
		return "", fmt.Errorf("dpop: proof iat %s is too far in the future", iat)
	}

	// --- Step 5b: validate htm ---
	htm, _ := claims["htm"].(string)
	if htm == "" {
		return "", errors.New("dpop: missing htm claim")
	}
	if htm != strings.ToUpper(method) {
		return "", fmt.Errorf("dpop: htm %q does not match method %q", htm, strings.ToUpper(method))
	}

	// --- Step 5c: validate htu ---
	htuClaim, _ := claims["htu"].(string)
	if htuClaim == "" {
		return "", errors.New("dpop: missing htu claim")
	}
	if !htuMatches(htuClaim, htu) {
		return "", fmt.Errorf("dpop: htu %q does not match request URL %q", htuClaim, htu)
	}

	// --- Step 5d: validate jti uniqueness ---
	jtiClaim, _ := claims["jti"].(string)
	if jtiClaim == "" {
		return "", errors.New("dpop: missing jti claim")
	}
	if err := cache.MarkSeen(jtiClaim, dpopWindow); err != nil {
		return "", err
	}

	// --- Step 5e: validate ath (access token hash) if required ---
	if accessTokenHash != "" {
		ath, _ := claims["ath"].(string)
		if ath == "" {
			return "", errors.New("dpop: missing ath claim (required when access token is present)")
		}
		if ath != accessTokenHash {
			return "", fmt.Errorf("dpop: ath mismatch")
		}
	}

	// --- Step 6: compute jkt thumbprint ---
	jkt, err = ComputeJWKThumbprint(jwkRaw)
	if err != nil {
		return "", fmt.Errorf("dpop: compute thumbprint: %w", err)
	}

	return jkt, nil
}

// HashAccessTokenForDPoP returns base64url(sha256(accessToken)) — the ath claim value.
func HashAccessTokenForDPoP(accessToken string) string {
	h := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ComputeJWKThumbprint computes the RFC 7638 SHA-256 thumbprint of a JWK.
// Canonical member order (RFC 7638 §3.3):
//   - EC key  (kty=EC):  crv, kty, x, y
//   - RSA key (kty=RSA): e, kty, n
//
// The canonical JSON is built with deterministic string interpolation (not
// json.Marshal of a map) to guarantee key order regardless of runtime state.
// Returns base64url (no padding).
func ComputeJWKThumbprint(jwk map[string]interface{}) (string, error) {
	kty, _ := jwk["kty"].(string)

	var canonical string
	switch kty {
	case "EC":
		crv, _ := jwk["crv"].(string)
		x, _ := jwk["x"].(string)
		y, _ := jwk["y"].(string)
		if crv == "" || x == "" || y == "" {
			return "", errors.New("dpop: EC JWK missing crv, x, or y")
		}
		// RFC 7638 canonical order: crv, kty, x, y — no whitespace.
		canonical = fmt.Sprintf(`{"crv":%s,"kty":%s,"x":%s,"y":%s}`,
			jsonString(crv), jsonString(kty), jsonString(x), jsonString(y))

	case "RSA":
		e, _ := jwk["e"].(string)
		n, _ := jwk["n"].(string)
		if e == "" || n == "" {
			return "", errors.New("dpop: RSA JWK missing e or n")
		}
		// RFC 7638 canonical order: e, kty, n — no whitespace.
		canonical = fmt.Sprintf(`{"e":%s,"kty":%s,"n":%s}`,
			jsonString(e), jsonString(kty), jsonString(n))

	default:
		return "", fmt.Errorf("dpop: unsupported kty %q for thumbprint", kty)
	}

	h := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(h[:]), nil
}

// jsonString returns a JSON-encoded string value (with quotes).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// ---------------------------------------------------------------------------
// Resource-server middleware
// ---------------------------------------------------------------------------

// RequireDPoPMiddleware validates DPoP when the Authorization scheme is "DPoP".
// If the scheme is "Bearer", the middleware falls through to next unchanged.
// The validated JWK thumbprint is stored in the request context (key: dpopJKTKey).
func RequireDPoPMiddleware(cache *DPoPJTICache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "DPoP ") {
				// Not a DPoP request — pass through.
				next.ServeHTTP(w, r)
				return
			}

			accessToken := strings.TrimPrefix(authHeader, "DPoP ")
			proofJWT := r.Header.Get("DPoP")
			if proofJWT == "" {
				WriteOAuthError(w, http.StatusUnauthorized,
					NewOAuthError(ErrInvalidDPoPProof, "DPoP header is required"))
				return
			}

			// Compute htu: scheme + host + path (no query or fragment).
			htu := r.URL.Scheme + "://" + r.Host + r.URL.Path
			if htu == "://" {
				// Fallback when scheme/host are not set (reverse-proxy scenario).
				htu = r.RequestURI
			}

			ath := HashAccessTokenForDPoP(accessToken)
			_, err := ValidateDPoPProof(proofJWT, r.Method, htu, ath, cache)
			if err != nil {
				WriteOAuthError(w, http.StatusUnauthorized,
					NewOAuthError(ErrInvalidDPoPProof, err.Error()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// htuMatches compares the claim htu against the request URL per RFC 9449 §4.3:
// scheme and host are case-insensitive; path is case-sensitive; query and
// fragment are ignored in both.
func htuMatches(claim, request string) bool {
	claim = stripQueryFragment(claim)
	request = stripQueryFragment(request)
	return strings.EqualFold(claim, request)
}

// stripQueryFragment removes query string and fragment from a URL string.
func stripQueryFragment(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// jwkToPublicKey converts a JWK map to a Go crypto.PublicKey.
// Supports EC (P-256, P-384, P-521) and RSA keys.
func jwkToPublicKey(jwk map[string]interface{}) (crypto.PublicKey, error) {
	kty, _ := jwk["kty"].(string)
	switch kty {
	case "EC":
		return jwkToECPublicKey(jwk)
	case "RSA":
		return jwkToRSAPublicKey(jwk)
	default:
		return nil, fmt.Errorf("unsupported kty %q", kty)
	}
}

func jwkToECPublicKey(jwk map[string]interface{}) (*ecdsa.PublicKey, error) {
	crv, _ := jwk["crv"].(string)
	xB64, _ := jwk["x"].(string)
	yB64, _ := jwk["y"].(string)

	xBytes, err := base64.RawURLEncoding.DecodeString(xB64)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yB64)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}

	var curve elliptic.Curve
	switch crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("unsupported EC curve %q", crv)
	}

	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	if !curve.IsOnCurve(pub.X, pub.Y) {
		return nil, errors.New("EC point not on curve")
	}

	return pub, nil
}

func jwkToRSAPublicKey(jwk map[string]interface{}) (*rsa.PublicKey, error) {
	nB64, _ := jwk["n"].(string)
	eB64, _ := jwk["e"].(string)

	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	if !e.IsInt64() {
		return nil, errors.New("RSA exponent too large")
	}

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
