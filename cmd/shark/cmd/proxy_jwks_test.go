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
	})
	if _, err := c.verifyBearer(ctx, "Bearer "+expired); err == nil {
		t.Fatalf("expired JWT accepted")
	}

	// Unknown kid
	_, otherPriv := buildJWKSServer(t, "kid-2")
	_ = otherPriv
	bad := signJWT(t, priv, "kid-unknown", jwtlib.MapClaims{"exp": now + 60})
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
