package cmd

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// buildJWKSServer returns an httptest.Server exposing /.well-known/jwks.json
// for the provided ECDSA key + kid, plus a helper that signs a JWT with
// that key so tests can mint valid/tampered tokens.
func buildJWKSServer(t *testing.T, kid string) (*httptest.Server, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		jwk := map[string]any{
			"kty": "EC",
			"kid": kid,
			"alg": "ES256",
			"use": "sig",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(priv.PublicKey.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(priv.PublicKey.Y.Bytes()),
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
	}))
	return srv, priv
}

func signJWT(t *testing.T, priv *ecdsa.PrivateKey, kid string, claims jwtlib.MapClaims) string {
	t.Helper()
	tok := jwtlib.NewWithClaims(jwtlib.SigningMethodES256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func TestJWKSCache_VerifyBearer(t *testing.T) {
	srv, priv := buildJWKSServer(t, "kid-1")
	defer srv.Close()
	_ = io.Discard

	c := newJWKSCache(srv.URL, slog.Default())
	// Matches the aud/iss on the signed claims below. Without these two
	// lines the cache accepts anything — the existing v1 bug that W15c
	// exists to close. Set them here so the rest of this test exercises
	// the full post-W15c validation pipeline.
	c.expectedAudiences = []string{"shark-proxy"}
	c.expectedIssuer = "https://auth.example"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	now := time.Now().Unix()
	valid := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
		"sub":   "usr_abc",
		"email": "a@b.co",
		"iat":   now,
		"exp":   now + 60,
		"aud":   "shark-proxy",
		"iss":   "https://auth.example",
	})
	id, err := c.verifyBearer(ctx, "Bearer "+valid)
	if err != nil {
		t.Fatalf("valid JWT rejected: %v", err)
	}
	if id.UserID != "usr_abc" || id.UserEmail != "a@b.co" {
		t.Fatalf("claims not populated: %+v", id)
	}

	// Tampered: flip a char in the signature segment.
	tampered := valid[:len(valid)-2] + "aa"
	if _, err := c.verifyBearer(ctx, "Bearer "+tampered); err == nil {
		t.Fatalf("tampered JWT accepted")
	}

	// Expired
	expired := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
		"sub": "usr_x",
		"exp": now - 60,
		"aud": "shark-proxy",
		"iss": "https://auth.example",
	})
	if _, err := c.verifyBearer(ctx, "Bearer "+expired); err == nil {
		t.Fatalf("expired JWT accepted")
	}

	// Unknown kid
	_, otherPriv := buildJWKSServer(t, "kid-2")
	_ = otherPriv
	bad := signJWT(t, priv, "kid-unknown", jwtlib.MapClaims{
		"exp": now + 60,
		"aud": "shark-proxy",
		"iss": "https://auth.example",
	})
	if _, err := c.verifyBearer(ctx, "Bearer "+bad); err == nil {
		t.Fatalf("unknown-kid JWT accepted")
	}

	// Missing header
	if _, err := c.verifyBearer(ctx, ""); err == nil {
		t.Fatalf("empty auth accepted")
	}
	// Wrong scheme
	if _, err := c.verifyBearer(ctx, "Basic Zm9v"); err == nil {
		t.Fatalf("Basic auth accepted")
	}
}

// TestJWKSCache_AudienceValidation covers the W15c audience + issuer
// checks. Each sub-test flips one claim to the wrong value (or omits it)
// and asserts the cache rejects the token. These tests would PASS (i.e.
// silently accept the token) without the W15c fix — that's the whole
// point of the check.
func TestJWKSCache_AudienceValidation(t *testing.T) {
	srv, priv := buildJWKSServer(t, "kid-1")
	defer srv.Close()

	newCache := func(t *testing.T) *jwksCache {
		c := newJWKSCache(srv.URL, slog.Default())
		c.expectedAudiences = []string{"shark-proxy", "edge-gateway"}
		c.expectedIssuer = "https://auth.example"
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		t.Cleanup(cancel)
		if err := c.Start(ctx); err != nil {
			t.Fatalf("Start: %v", err)
		}
		return c
	}
	now := time.Now().Unix()

	t.Run("valid aud + valid iss -> accepted", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": "shark-proxy",
			"iss": "https://auth.example",
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err != nil {
			t.Fatalf("valid JWT rejected: %v", err)
		}
	})

	t.Run("second expected aud -> accepted", func(t *testing.T) {
		// aud set-intersection: the second configured audience should also
		// match if the token carries it.
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": "edge-gateway",
			"iss": "https://auth.example",
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err != nil {
			t.Fatalf("second-audience JWT rejected: %v", err)
		}
	})

	t.Run("multi-aud array -> accepted on intersection", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": []string{"other-service", "shark-proxy"},
			"iss": "https://auth.example",
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err != nil {
			t.Fatalf("multi-aud JWT rejected: %v", err)
		}
	})

	t.Run("wrong aud -> rejected", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": "some-other-api",
			"iss": "https://auth.example",
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err == nil {
			t.Fatalf("wrong-audience JWT accepted (CVE-shape gap)")
		}
	})

	t.Run("missing aud -> rejected", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"iss": "https://auth.example",
			// no aud
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err == nil {
			t.Fatalf("missing-audience JWT accepted (CVE-shape gap)")
		}
	})

	t.Run("wrong iss -> rejected", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": "shark-proxy",
			"iss": "https://attacker.example",
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err == nil {
			t.Fatalf("wrong-issuer JWT accepted (CVE-shape gap)")
		}
	})

	t.Run("missing iss -> rejected", func(t *testing.T) {
		c := newCache(t)
		tok := signJWT(t, priv, "kid-1", jwtlib.MapClaims{
			"sub": "usr_abc",
			"exp": now + 60,
			"aud": "shark-proxy",
			// no iss
		})
		if _, err := c.verifyBearer(context.Background(), "Bearer "+tok); err == nil {
			t.Fatalf("missing-issuer JWT accepted (CVE-shape gap)")
		}
	})
}

// TestJWKSCache_OversizedBodyRejected ensures the proxy refuses JWKS
// responses larger than the hardcoded 1 MiB limit. Without the cap the
// proxy would happily buffer whatever the upstream streams, enabling a
// DoS-by-memory via a malicious or compromised auth server. The assertion
// is: refresh returns an error AND the cache's key map stays empty (no
// partial/corrupt state).
func TestJWKSCache_OversizedBodyRejected(t *testing.T) {
	// Stream a body well past 1 MiB. Content doesn't matter — the limit
	// check trips before we attempt JSON decoding.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/jwks.json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		// 2 MiB of junk JSON so the decoder would also blow up if we got
		// that far — but the size gate should trip first.
		chunk := strings.Repeat("a", 64*1024)
		for i := 0; i < 32; i++ {
			_, _ = w.Write([]byte(chunk))
		}
	}))
	defer srv.Close()

	c := newJWKSCache(srv.URL, slog.Default())
	err := c.refresh(context.Background())
	if err == nil {
		t.Fatalf("expected error for oversized JWKS body, got nil")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected 'too large' error, got %v", err)
	}
	// Cache must stay empty — no half-parsed state leaking in.
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.keys) != 0 {
		t.Fatalf("expected empty key map after rejection, got %d keys", len(c.keys))
	}
}

