package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// GenerateES256Keypair generates a new ECDSA P-256 keypair.
func GenerateES256Keypair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate EC key: %w", err)
	}
	return priv, &priv.PublicKey, nil
}

// MarshalES256PrivateKeyPEM encodes an ECDSA private key to PEM (PKCS#8).
func MarshalES256PrivateKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal EC private key: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return pem.EncodeToMemory(block), nil
}

// ParseES256PrivateKeyPEM decodes a PEM-encoded ECDSA private key.
func ParseES256PrivateKeyPEM(data []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse EC private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not ECDSA")
	}
	return ecKey, nil
}

// MarshalES256PublicKeyPEM encodes an ECDSA public key to PEM (PKIX).
func MarshalES256PublicKeyPEM(key *ecdsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal EC public key: %w", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return pem.EncodeToMemory(block), nil
}

// ComputeES256KID derives a key ID from an ECDSA public key.
// Same pattern as RSA: SHA-256 of DER-encoded public key, base64url[:16].
func ComputeES256KID(pub *ecdsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		// This should never fail for a valid P-256 key.
		return ""
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:])[:16]
}

// ES256PublicJWK builds an RFC 7517 JWK map from an ECDSA P-256 public key.
// P-256 coordinates are exactly 32 bytes; left-pad with zeros if shorter.
func ES256PublicJWK(pub *ecdsa.PublicKey, kid string) map[string]interface{} {
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	// Left-pad to exactly 32 bytes for P-256.
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	return map[string]interface{}{
		"kty": "EC",
		"use": "sig",
		"alg": "ES256",
		"kid": kid,
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(xPadded),
		"y":   base64.RawURLEncoding.EncodeToString(yPadded),
	}
}
