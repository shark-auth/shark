package oauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ecKeyPair generates a fresh P-256 keypair for each test.
func ecKeyPair(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}
	return priv
}

// rsaKeyPair generates a fresh 2048-bit RSA keypair for each test.
func rsaKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return priv
}

// ecJWK returns the JWK representation of the EC public key.
func ecJWK(pub *ecdsa.PublicKey) map[string]interface{} {
	byteLen := (pub.Curve.Params().BitSize + 7) / 8
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()
	// Left-pad to byteLen.
	if len(xBytes) < byteLen {
		padded := make([]byte, byteLen)
		copy(padded[byteLen-len(xBytes):], xBytes)
		xBytes = padded
	}
	if len(yBytes) < byteLen {
		padded := make([]byte, byteLen)
		copy(padded[byteLen-len(yBytes):], yBytes)
		yBytes = padded
	}
	return map[string]interface{}{
		"kty": "EC",
		"crv": pub.Curve.Params().Name,
		"x":   base64.RawURLEncoding.EncodeToString(xBytes),
		"y":   base64.RawURLEncoding.EncodeToString(yBytes),
	}
}

// rsaJWK returns the JWK representation of the RSA public key.
func rsaJWK(pub *rsa.PublicKey) map[string]interface{} {
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	return map[string]interface{}{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

type proofParams struct {
	priv   *ecdsa.PrivateKey
	jwk    map[string]interface{}
	method string
	htu    string
	iat    time.Time
	jti    string
	ath    string
	typ    string // override typ header
	alg    string // override alg header
}

// buildECProof builds a valid DPoP proof JWT using ES256.
func buildECProof(t *testing.T, p proofParams) string {
	t.Helper()

	typ := "dpop+jwt"
	if p.typ != "" {
		typ = p.typ
	}
	alg := "ES256"
	if p.alg != "" {
		alg = p.alg
	}

	jti := p.jti
	if jti == "" {
		jti = randomJTI(t)
	}

	iat := p.iat
	if iat.IsZero() {
		iat = time.Now().UTC()
	}

	claims := gojwt.MapClaims{
		"jti": jti,
		"htm": p.method,
		"htu": p.htu,
		"iat": iat.Unix(),
	}
	if p.ath != "" {
		claims["ath"] = p.ath
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodES256, claims)
	token.Header["typ"] = typ
	token.Header["alg"] = alg

	jwkBytes, _ := json.Marshal(p.jwk)
	var jwkMap interface{}
	_ = json.Unmarshal(jwkBytes, &jwkMap)
	token.Header["jwk"] = jwkMap

	signed, err := token.SignedString(p.priv)
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return signed
}

// randomJTI produces a random 16-char hex string using crypto/rand.
func randomJTI(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand jti: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// newCache returns a fresh JTI cache for each test.
func newCache() *DPoPJTICache {
	return NewDPoPJTICache()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDPoP_ValidProof(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
	})

	jkt, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err != nil {
		t.Fatalf("expected valid proof, got error: %v", err)
	}
	if jkt == "" {
		t.Fatal("expected non-empty jkt")
	}

	// jkt must be base64url-encoded (no padding, no +/)
	if strings.Contains(jkt, "+") || strings.Contains(jkt, "/") || strings.Contains(jkt, "=") {
		t.Errorf("jkt contains invalid base64url characters: %q", jkt)
	}
}

func TestDPoP_WrongMethod(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "GET",
		htu:    "https://auth.example.com/oauth/token",
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for wrong method")
	}
	if !strings.Contains(err.Error(), "htm") {
		t.Errorf("expected htm mismatch error, got: %v", err)
	}
}

func TestDPoP_WrongURL(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://other.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for wrong URL")
	}
	if !strings.Contains(err.Error(), "htu") {
		t.Errorf("expected htu mismatch error, got: %v", err)
	}
}

func TestDPoP_ExpiredIat(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	oldIat := time.Now().UTC().Add(-90 * time.Second) // 90s in the past

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		iat:    oldIat,
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for expired iat")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expiry error, got: %v", err)
	}
}

func TestDPoP_FutureIat(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	futureIat := time.Now().UTC().Add(90 * time.Second) // 90s in the future

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		iat:    futureIat,
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for future iat")
	}
	if !strings.Contains(err.Error(), "future") {
		t.Errorf("expected future-iat error, got: %v", err)
	}
}

func TestDPoP_InvalidTyp(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		typ:    "JWT", // wrong typ
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for invalid typ")
	}
	if !strings.Contains(err.Error(), "typ") {
		t.Errorf("expected typ error, got: %v", err)
	}
}

func TestDPoP_HMACAlg_Rejected(t *testing.T) {
	// Build a JWT manually with alg=HS256 in the header but sign it with HMAC.
	// We cannot use buildECProof here since it always uses ES256 signing.
	// Instead we craft the raw JWT string with a manipulated header.
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	// Use ES256 for signing but claim HS256 in the header — ValidateDPoPProof
	// must reject HS256 before even attempting to verify the signature.
	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		alg:    "HS256", // alg field set in header; our override puts it in header["alg"]
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for HS256 alg")
	}
	if !strings.Contains(err.Error(), "algorithm") && !strings.Contains(err.Error(), "alg") {
		t.Errorf("expected algorithm rejection error, got: %v", err)
	}
}

func TestDPoP_ReplayedJTI(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()
	jti := "fixed-jti-replay-test"

	// First use should succeed.
	proof1 := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		jti:    jti,
	})
	if _, err := ValidateDPoPProof(proof1, "POST", "https://auth.example.com/oauth/token", "", cache); err != nil {
		t.Fatalf("first proof should succeed: %v", err)
	}

	// Second use with same jti (and same iat/cache) should be rejected.
	proof2 := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token",
		jti:    jti,
	})
	_, err := ValidateDPoPProof(proof2, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected replay error on second use of same jti")
	}
	if !strings.Contains(err.Error(), "replay") {
		t.Errorf("expected replay error, got: %v", err)
	}
}

func TestDPoP_WithAth(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	accessToken := "my-opaque-access-token"
	h := sha256.Sum256([]byte(accessToken))
	ath := base64.RawURLEncoding.EncodeToString(h[:])

	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "GET",
		htu:    "https://api.example.com/resource",
		ath:    ath,
	})

	_, err := ValidateDPoPProof(proof, "GET", "https://api.example.com/resource", ath, cache)
	if err != nil {
		t.Fatalf("expected valid proof with ath, got: %v", err)
	}
}

func TestDPoP_WrongAth(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	realToken := "real-access-token"
	fakeToken := "tampered-access-token"

	h := sha256.Sum256([]byte(fakeToken))
	wrongAth := base64.RawURLEncoding.EncodeToString(h[:])

	// The proof contains the hash of fakeToken.
	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "GET",
		htu:    "https://api.example.com/resource",
		ath:    wrongAth,
	})

	// But the caller presents realToken.
	realAth := HashAccessTokenForDPoP(realToken)
	_, err := ValidateDPoPProof(proof, "GET", "https://api.example.com/resource", realAth, cache)
	if err == nil {
		t.Fatal("expected ath mismatch error")
	}
	if !strings.Contains(err.Error(), "ath") {
		t.Errorf("expected ath mismatch error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// JWK thumbprint tests
// ---------------------------------------------------------------------------

func TestComputeJWKThumbprint_ES256(t *testing.T) {
	// RFC 7638 §3.1 example thumbprint for a P-256 key.
	// We use a known key and verify the output is deterministic and correct format.
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)

	thumb1, err := ComputeJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint error: %v", err)
	}
	if thumb1 == "" {
		t.Fatal("thumbprint should not be empty")
	}

	// Compute again — must be deterministic.
	thumb2, err := ComputeJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint 2 error: %v", err)
	}
	if thumb1 != thumb2 {
		t.Errorf("thumbprint not deterministic: %q vs %q", thumb1, thumb2)
	}

	// Must be base64url without padding.
	if strings.ContainsAny(thumb1, "+/=") {
		t.Errorf("thumbprint contains non-base64url chars: %q", thumb1)
	}

	// SHA-256 of 32 bytes → 32 bytes → 43 base64url chars.
	if len(thumb1) != 43 {
		t.Errorf("expected 43-char base64url thumbprint, got %d: %q", len(thumb1), thumb1)
	}
}

func TestComputeJWKThumbprint_RSA(t *testing.T) {
	priv := rsaKeyPair(t)
	jwk := rsaJWK(&priv.PublicKey)

	thumb1, err := ComputeJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint error: %v", err)
	}
	if thumb1 == "" {
		t.Fatal("thumbprint should not be empty")
	}

	// Deterministic.
	thumb2, err := ComputeJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint 2 error: %v", err)
	}
	if thumb1 != thumb2 {
		t.Errorf("thumbprint not deterministic: %q vs %q", thumb1, thumb2)
	}

	// Base64url, no padding.
	if strings.ContainsAny(thumb1, "+/=") {
		t.Errorf("thumbprint contains non-base64url chars: %q", thumb1)
	}
}

// TestComputeJWKThumbprint_KnownVector validates the RFC 7638 canonical JSON
// construction and SHA-256 thumbprint computation for a known EC P-256 key.
// The canonical JSON for EC keys (RFC 7638 §3.3) must have members in the
// order: crv, kty, x, y — with no extra whitespace.
//
// The test key is from RFC 7638 §3.1. The canonical JSON is:
//
//	{"crv":"P-256","kty":"EC","x":"f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU","y":"x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0"}
//
// SHA-256 of the above (base64url, no padding):
//
//	oKIywvGUpTVTyxMQ3bwIIeQUudfr_CkLMjCE19ECD-U
func TestComputeJWKThumbprint_KnownVector(t *testing.T) {
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		"y":   "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
	}

	thumb, err := ComputeJWKThumbprint(jwk)
	if err != nil {
		t.Fatalf("thumbprint error: %v", err)
	}

	// The expected thumbprint is the base64url(SHA-256) of the canonical JSON
	// above. Verified independently by computing SHA-256 of the exact string.
	expected := "oKIywvGUpTVTyxMQ3bwIIeQUudfr_CkLMjCE19ECD-U"
	if thumb != expected {
		t.Errorf("thumbprint mismatch\n  got:  %q\n  want: %q", thumb, expected)
	}
}

// TestHashAccessTokenForDPoP verifies the helper produces base64url(sha256(token)).
func TestHashAccessTokenForDPoP(t *testing.T) {
	token := "hello-world"
	h := sha256.Sum256([]byte(token))
	expected := base64.RawURLEncoding.EncodeToString(h[:])

	got := HashAccessTokenForDPoP(token)
	if got != expected {
		t.Errorf("ath hash mismatch: got %q, want %q", got, expected)
	}
}

// TestDPoP_MissingHeader checks that an empty proofJWT returns an error.
func TestDPoP_MissingHeader(t *testing.T) {
	cache := newCache()
	_, err := ValidateDPoPProof("", "POST", "https://auth.example.com/oauth/token", "", cache)
	if err == nil {
		t.Fatal("expected error for missing proof")
	}
}

// TestDPoP_HTUQueryIgnored verifies that query params in the proof htu are
// stripped and still match a clean request URL.
func TestDPoP_HTUQueryIgnored(t *testing.T) {
	priv := ecKeyPair(t)
	jwk := ecJWK(&priv.PublicKey)
	cache := newCache()

	// Proof was built with a query string appended to htu (should be ignored).
	proof := buildECProof(t, proofParams{
		priv:   priv,
		jwk:    jwk,
		method: "POST",
		htu:    "https://auth.example.com/oauth/token?foo=bar",
	})

	_, err := ValidateDPoPProof(proof, "POST", "https://auth.example.com/oauth/token", "", cache)
	if err != nil {
		t.Fatalf("expected query string to be ignored in htu comparison, got: %v", err)
	}
}
