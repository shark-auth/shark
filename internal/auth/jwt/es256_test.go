package jwt

import (
	"testing"
)

func TestGenerateES256Keypair(t *testing.T) {
	priv, pub, err := GenerateES256Keypair()
	if err != nil {
		t.Fatalf("GenerateES256Keypair() error = %v", err)
	}
	if priv == nil {
		t.Fatal("private key is nil")
	}
	if pub == nil {
		t.Fatal("public key is nil")
	}
	// Verify the public key matches the private key's embedded public key.
	if priv.PublicKey.X.Cmp(pub.X) != 0 || priv.PublicKey.Y.Cmp(pub.Y) != 0 {
		t.Fatal("public key does not match private key's public component")
	}
	// Verify curve is P-256.
	if priv.Curve.Params().Name != "P-256" {
		t.Fatalf("expected P-256 curve, got %s", priv.Curve.Params().Name)
	}
}

func TestES256PEMRoundTrip(t *testing.T) {
	priv, pub, err := GenerateES256Keypair()
	if err != nil {
		t.Fatalf("GenerateES256Keypair() error = %v", err)
	}

	// Private key round-trip.
	privPEM, err := MarshalES256PrivateKeyPEM(priv)
	if err != nil {
		t.Fatalf("MarshalES256PrivateKeyPEM() error = %v", err)
	}
	parsedPriv, err := ParseES256PrivateKeyPEM(privPEM)
	if err != nil {
		t.Fatalf("ParseES256PrivateKeyPEM() error = %v", err)
	}
	if parsedPriv.D.Cmp(priv.D) != 0 {
		t.Fatal("parsed private key D does not match original")
	}

	// Public key round-trip.
	pubPEM, err := MarshalES256PublicKeyPEM(pub)
	if err != nil {
		t.Fatalf("MarshalES256PublicKeyPEM() error = %v", err)
	}
	if len(pubPEM) == 0 {
		t.Fatal("public key PEM is empty")
	}

	// Keys must still match after marshal.
	if parsedPriv.PublicKey.X.Cmp(pub.X) != 0 || parsedPriv.PublicKey.Y.Cmp(pub.Y) != 0 {
		t.Fatal("round-tripped public key does not match original")
	}
}

func TestES256JWK(t *testing.T) {
	_, pub, err := GenerateES256Keypair()
	if err != nil {
		t.Fatalf("GenerateES256Keypair() error = %v", err)
	}
	kid := ComputeES256KID(pub)
	jwk := ES256PublicJWK(pub, kid)

	checks := map[string]string{
		"kty": "EC",
		"use": "sig",
		"alg": "ES256",
		"crv": "P-256",
		"kid": kid,
	}
	for field, want := range checks {
		got, ok := jwk[field]
		if !ok {
			t.Errorf("JWK missing field %q", field)
			continue
		}
		if got != want {
			t.Errorf("JWK[%q] = %q, want %q", field, got, want)
		}
	}

	// x and y must be present and non-empty.
	for _, coord := range []string{"x", "y"} {
		v, ok := jwk[coord]
		if !ok {
			t.Errorf("JWK missing coord %q", coord)
			continue
		}
		s, ok := v.(string)
		if !ok || s == "" {
			t.Errorf("JWK[%q] is empty or not a string", coord)
		}
	}
}

func TestComputeES256KID(t *testing.T) {
	_, pub, err := GenerateES256Keypair()
	if err != nil {
		t.Fatalf("GenerateES256Keypair() error = %v", err)
	}

	kid1 := ComputeES256KID(pub)
	if kid1 == "" {
		t.Fatal("ComputeES256KID() returned empty string")
	}
	if len(kid1) != 16 {
		t.Fatalf("ComputeES256KID() length = %d, want 16", len(kid1))
	}

	// Must be deterministic.
	kid2 := ComputeES256KID(pub)
	if kid1 != kid2 {
		t.Fatalf("ComputeES256KID() not deterministic: %q vs %q", kid1, kid2)
	}
}
