package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/shark-auth/shark/internal/identity"
	"github.com/shark-auth/shark/internal/oauth"
)

// The DPoP tests live in their own file so the crypto + JWK helpers they
// need don't pollute the rest of proxy_test.go. Helper shape mirrors
// internal/oauth/dpop_test.go (intentional: keeps one reference for how
// a valid proof is assembled).

// dpopKey generates a fresh P-256 keypair for each test. Reusing keys
// across tests would make the JTI cache leak between them â€” each test
// gets its own fresh key + cache.
func dpopKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("dpop key: %v", err)
	}
	return priv
}

// dpopJWK renders the P-256 public key as a JWK. Values are left-padded
// to the curve's byte length per RFC 7518 Â§6.2.
func dpopJWK(pub *ecdsa.PublicKey) map[string]interface{} {
	byteLen := (pub.Curve.Params().BitSize + 7) / 8
	pad := func(b []byte) []byte {
		if len(b) >= byteLen {
			return b
		}
		out := make([]byte, byteLen)
		copy(out[byteLen-len(b):], b)
		return out
	}
	return map[string]interface{}{
		"kty": "EC",
		"crv": pub.Curve.Params().Name,
		"x":   base64.RawURLEncoding.EncodeToString(pad(pub.X.Bytes())),
		"y":   base64.RawURLEncoding.EncodeToString(pad(pub.Y.Bytes())),
	}
}

// buildDPoPProof signs a DPoP proof JWT with the given parameters. ath
// is optional; empty string skips the claim entirely (matching the real
// client behaviour when no bearer is bound).
func buildDPoPProof(t *testing.T, priv *ecdsa.PrivateKey, method, htu, ath string) string {
	t.Helper()
	jti := make([]byte, 8)
	if _, err := rand.Read(jti); err != nil {
		t.Fatalf("rand jti: %v", err)
	}
	claims := gojwt.MapClaims{
		"jti": base64.RawURLEncoding.EncodeToString(jti),
		"htm": method,
		"htu": htu,
		"iat": time.Now().UTC().Unix(),
	}
	if ath != "" {
		claims["ath"] = ath
	}
	tok := gojwt.NewWithClaims(gojwt.SigningMethodES256, claims)
	tok.Header["typ"] = "dpop+jwt"
	tok.Header["alg"] = "ES256"
	jwkBytes, _ := json.Marshal(dpopJWK(&priv.PublicKey))
	var jwkMap interface{}
	_ = json.Unmarshal(jwkBytes, &jwkMap)
	tok.Header["jwk"] = jwkMap
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign dpop: %v", err)
	}
	return signed
}

// serveDPoPProxy wires a ReverseProxy against the given upstream URL
// with a fresh JTI cache and returns the proxy server ready to receive
// requests. The cache pointer is returned too so tests can assert on
// replay behaviour across multiple requests.
func serveDPoPProxy(t *testing.T, upstream string) (*httptest.Server, *oauth.DPoPJTICache) {
	t.Helper()
	cfg := Config{
		Enabled:       true,
		Upstream:      upstream,
		Timeout:       2 * time.Second,
		StripIncoming: true,
	}
	// Allow-anonymous engine so the rules engine doesn't 401 before we
	// reach the DPoP check; we want to test the DPoP gate in isolation.
	engine, err := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New proxy: %v", err)
	}
	cache := oauth.NewDPoPJTICache()
	p.SetDPoPCache(cache)

	return httptest.NewServer(p), cache
}

// dpopRequest builds an http.Request with DPoP + Authorization headers,
// and stashes the given identity onto the request context so the proxy
// sees it (mimics the auth middleware's behaviour).
func dpopRequest(t *testing.T, proxyURL, method, path, bearer, proof string, id identity.Identity) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, proxyURL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	_ = id // stashed by the proxy via r.Context on the server side
	return req
}

// TestReverseProxy_DPoP_ValidProofPasses verifies the happy path: valid
// proof + identity.AuthMethod=DPoP â†’ upstream receives the request.
func TestReverseProxy_DPoP_ValidProofPasses(t *testing.T) {
	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	priv := dpopKey(t)
	cfg := Config{Enabled: true, Upstream: upstream.URL, Timeout: 2 * time.Second, StripIncoming: true}
	engine, _ := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetDPoPCache(oauth.NewDPoPJTICache())

	bearer := "test-access-token"
	ath := oauth.HashAccessTokenForDPoP(bearer)
	htu := "http://example.test/api/resource"
	proof := buildDPoPProof(t, priv, http.MethodGet, htu, ath)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Host = "example.test"
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("DPoP", proof)
	req = req.WithContext(identity.WithIdentity(req.Context(), identity.Identity{
		UserID:     "u1",
		AuthMethod: identity.AuthMethodDPoP,
	}))

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("valid DPoP should pass: got %d, reason=%q", rec.Code, rec.Header().Get(HeaderDenyReason))
	}
	if hits != 1 {
		t.Errorf("upstream hits = %d, want 1", hits)
	}
}

// TestReverseProxy_DPoP_BadProofRejects covers a malformed proof: the
// proxy must 401 with a DPoP-specific reason in the deny header and
// must not call the upstream.
func TestReverseProxy_DPoP_BadProofRejects(t *testing.T) {
	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
	}))
	defer upstream.Close()

	cfg := Config{Enabled: true, Upstream: upstream.URL, Timeout: 2 * time.Second, StripIncoming: true}
	engine, _ := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetDPoPCache(oauth.NewDPoPJTICache())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Host = "example.test"
	req.Header.Set("Authorization", "Bearer tok")
	req.Header.Set("DPoP", "not-a-jwt.at.all")
	req = req.WithContext(identity.WithIdentity(req.Context(), identity.Identity{
		UserID:     "u1",
		AuthMethod: identity.AuthMethodDPoP,
	}))

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad DPoP should 401: got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get(HeaderDenyReason), "dpop") {
		t.Errorf("deny reason should mention dpop, got %q", rec.Header().Get(HeaderDenyReason))
	}
	if hits != 0 {
		t.Errorf("upstream must not be called: hits = %d", hits)
	}
}

// TestReverseProxy_DPoP_BadHtuRejects flips htu so the signed proof
// mismatches the request URL. Validate returns an error; proxy 401.
func TestReverseProxy_DPoP_BadHtuRejects(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	priv := dpopKey(t)
	cfg := Config{Enabled: true, Upstream: upstream.URL, Timeout: 2 * time.Second, StripIncoming: true}
	engine, _ := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetDPoPCache(oauth.NewDPoPJTICache())

	bearer := "tok"
	ath := oauth.HashAccessTokenForDPoP(bearer)
	// Proof signed for a different URL than the request will use.
	proof := buildDPoPProof(t, priv, http.MethodGet, "http://other.test/wrong/path", ath)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Host = "example.test"
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("DPoP", proof)
	req = req.WithContext(identity.WithIdentity(req.Context(), identity.Identity{
		UserID:     "u1",
		AuthMethod: identity.AuthMethodDPoP,
	}))
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("htu mismatch should 401: got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get(HeaderDenyReason), "dpop") {
		t.Error("deny reason should mention dpop")
	}
}

// TestReverseProxy_DPoP_ReplayRejects submits the same proof twice;
// second submission must be rejected by the JTI cache.
func TestReverseProxy_DPoP_ReplayRejects(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	priv := dpopKey(t)
	cfg := Config{Enabled: true, Upstream: upstream.URL, Timeout: 2 * time.Second, StripIncoming: true}
	engine, _ := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetDPoPCache(oauth.NewDPoPJTICache())

	bearer := "tok"
	ath := oauth.HashAccessTokenForDPoP(bearer)
	htu := "http://example.test/api/resource"
	proof := buildDPoPProof(t, priv, http.MethodGet, htu, ath)

	// First submission â€” should pass.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req1.Host = "example.test"
	req1.Header.Set("Authorization", "Bearer "+bearer)
	req1.Header.Set("DPoP", proof)
	req1 = req1.WithContext(identity.WithIdentity(req1.Context(), identity.Identity{
		UserID: "u1", AuthMethod: identity.AuthMethodDPoP,
	}))
	p.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first submission should pass: got %d reason=%q", rec1.Code, rec1.Header().Get(HeaderDenyReason))
	}

	// Replay â€” should be rejected.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req2.Host = "example.test"
	req2.Header.Set("Authorization", "Bearer "+bearer)
	req2.Header.Set("DPoP", proof)
	req2 = req2.WithContext(identity.WithIdentity(req2.Context(), identity.Identity{
		UserID: "u1", AuthMethod: identity.AuthMethodDPoP,
	}))
	p.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("replay should 401: got %d", rec2.Code)
	}
}

// TestReverseProxy_DPoP_NonDPoPBearerSkipped verifies the proxy does
// NOT demand a DPoP proof when the identity is non-DPoP (plain bearer
// JWT). This keeps the feature backwards compatible with existing JWT
// deployments.
func TestReverseProxy_DPoP_NonDPoPBearerSkipped(t *testing.T) {
	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := Config{Enabled: true, Upstream: upstream.URL, Timeout: 2 * time.Second, StripIncoming: true}
	engine, _ := NewEngine([]RuleSpec{{Path: "/*", Allow: "anonymous"}})
	p, err := New(cfg, engine, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetDPoPCache(oauth.NewDPoPJTICache())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/resource", nil)
	req.Host = "example.test"
	req.Header.Set("Authorization", "Bearer plain")
	// No DPoP header â€” but identity says JWT, so enforcement is skipped.
	req = req.WithContext(identity.WithIdentity(req.Context(), identity.Identity{
		UserID:     "u1",
		AuthMethod: identity.AuthMethodJWT,
	}))
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("non-DPoP bearer should skip DPoP check: got %d reason=%q", rec.Code, rec.Header().Get(HeaderDenyReason))
	}
	if hits != 1 {
		t.Errorf("upstream should have been hit: hits = %d", hits)
	}
}
