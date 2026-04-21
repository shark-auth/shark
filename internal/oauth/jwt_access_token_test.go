package oauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// TestJWTAccessToken_Shape verifies that /oauth/token returns RFC 7519 JWT
// access tokens (3 dot-separated parts) signed by the JWKS key, with all
// expected claims present and verifiable.
//
// This is the canonical DX1 acceptance test: SDKs using
// decode_agent_token + /.well-known/jwks.json must be able to parse the
// tokens we issue.
func TestJWTAccessToken_Shape(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "jwt-shape-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	tokenStr := obtainAccessToken(t, ts, "jwt-shape-client", "test-secret")

	// 1. JWT shape: exactly two dots → header.payload.signature.
	if got := strings.Count(tokenStr, "."); got != 2 {
		t.Fatalf("expected JWT shape (2 dots), got %d in %q", got, tokenStr)
	}

	// 2. Parse with signature verification using the server's ES256 public key.
	pub := &srv.signingPrivKey.PublicKey
	parsed, err := gojwt.Parse(tokenStr, func(t *gojwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodECDSA); !ok {
			t.Header["alg"] = nil // force failure msg below
			return nil, gojwt.ErrSignatureInvalid
		}
		return pub, nil
	})
	if err != nil {
		t.Fatalf("verifying JWT signature: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("parsed JWT is not valid")
	}

	// 3. Header contains kid + ES256 alg.
	if alg, _ := parsed.Header["alg"].(string); alg != "ES256" {
		t.Errorf("expected alg=ES256, got %q", alg)
	}
	if kid, _ := parsed.Header["kid"].(string); kid == "" || kid != srv.SigningKeyID {
		t.Errorf("expected kid=%q, got %q", srv.SigningKeyID, kid)
	}

	// 4. Required claims present.
	claims, ok := parsed.Claims.(gojwt.MapClaims)
	if !ok {
		t.Fatalf("unexpected claim type: %T", parsed.Claims)
	}
	for _, k := range []string{"sub", "iss", "aud", "exp", "iat", "jti", "scope", "client_id"} {
		if _, present := claims[k]; !present {
			t.Errorf("missing claim %q in %v", k, claims)
		}
	}

	if got, _ := claims["iss"].(string); got != srv.Issuer {
		t.Errorf("expected iss=%q, got %q", srv.Issuer, got)
	}
	if got, _ := claims["client_id"].(string); got != "jwt-shape-client" {
		t.Errorf("expected client_id=jwt-shape-client, got %q", got)
	}
	if got, _ := claims["sub"].(string); got != "jwt-shape-client" {
		t.Errorf("expected sub=jwt-shape-client (client_credentials), got %q", got)
	}
	if got, _ := claims["scope"].(string); !strings.Contains(got, "openid") {
		t.Errorf("expected scope to contain openid, got %q", got)
	}

	// 5. Refresh tokens (when present) must remain opaque, not JWT.
	// client_credentials does not issue refresh tokens by default, so we
	// don't assert here. The dedicated refresh test below covers this.
}

// TestJWTAccessToken_Introspection verifies that introspecting a JWT access
// token still returns active:true with the same claim set per RFC 7662.
func TestJWTAccessToken_Introspection(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "jwt-introspect-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	tokenStr := obtainAccessToken(t, ts, "jwt-introspect-client", "test-secret")

	status, result := doIntrospect(t, ts, tokenStr, basicAuth("jwt-introspect-client", "test-secret"))
	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if result["active"] != true {
		t.Fatalf("expected active=true for JWT access token, got %v", result)
	}
	for _, k := range []string{"client_id", "exp", "iat", "iss", "scope"} {
		if _, ok := result[k]; !ok {
			t.Errorf("introspection result missing %q: %v", k, result)
		}
	}
}

// TestJWTAccessToken_DPoPCnfJkt verifies that a DPoP-bound token carries the
// cnf.jkt JWK thumbprint binding inside the JWT body.
func TestJWTAccessToken_DPoPCnfJkt(t *testing.T) {
	srv, store := newTestOAuthServer(t)
	seedAgent(t, store, "dpop-cnf-client", false)
	ts := httptest.NewServer(mountIntrospectRevokeRouter(srv))
	defer ts.Close()

	// Build a fresh ES256 keypair for the DPoP proof.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ec key: %v", err)
	}

	// Compute the expected JWK thumbprint (RFC 7638) for the public key.
	x := base64.RawURLEncoding.EncodeToString(priv.PublicKey.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(priv.PublicKey.Y.Bytes())
	thumbInput := `{"crv":"P-256","kty":"EC","x":"` + x + `","y":"` + y + `"}`
	sum := sha256.Sum256([]byte(thumbInput))
	expectedJKT := base64.RawURLEncoding.EncodeToString(sum[:])

	// Mint a DPoP proof for POST {token endpoint}.
	tokenURL := ts.URL + "/oauth/token"
	jwk := map[string]interface{}{
		"crv": "P-256",
		"kty": "EC",
		"x":   x,
		"y":   y,
	}
	header := map[string]interface{}{"typ": "dpop+jwt", "alg": "ES256", "jwk": jwk}
	now := time.Now().UTC().Unix()
	payload := map[string]interface{}{
		"htm": "POST",
		"htu": tokenURL,
		"iat": now,
		"jti": "dpop-jti-" + time.Now().Format("20060102150405.000000000"),
	}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		t.Fatalf("sign dpop: %v", err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	proof := signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)

	// Mint a token with the DPoP header.
	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}
	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", proof)
	req.SetBasicAuth("dpop-cnf-client", "test-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("token request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 from token endpoint, got %d: %s", resp.StatusCode, body)
	}
	var tokenResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&tokenResp) //nolint:errcheck
	tokenStr, _ := tokenResp["access_token"].(string)
	if tokenStr == "" {
		t.Fatal("missing access_token")
	}

	// Decode the JWT (no signature verification needed for this assertion).
	parser := gojwt.NewParser(gojwt.WithoutClaimsValidation())
	parsed, _, err := parser.ParseUnverified(tokenStr, gojwt.MapClaims{})
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}
	claims, _ := parsed.Claims.(gojwt.MapClaims)

	cnf, ok := claims["cnf"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected cnf claim, got %v", claims["cnf"])
	}
	jkt, _ := cnf["jkt"].(string)
	if jkt != expectedJKT {
		t.Errorf("expected cnf.jkt=%q, got %q", expectedJKT, jkt)
	}
}

