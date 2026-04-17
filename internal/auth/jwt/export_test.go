// export_test.go exposes internal helpers for black-box tests in the jwt_test
// package. This file is only compiled during `go test`.
package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// GenerateTestKeyPair generates a fresh RSA keypair and returns the private key,
// the plaintext public key PEM string, and the computed kid.
// Used by tests that need to sign tokens externally.
func GenerateTestKeyPair() (*rsa.PrivateKey, string, string, error) {
	priv, err := generateRSAKeypair()
	if err != nil {
		return nil, "", "", fmt.Errorf("generate keypair: %w", err)
	}
	kid, err := computeKID(&priv.PublicKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("compute kid: %w", err)
	}
	pubPEM, err := encodePublicKeyPEM(&priv.PublicKey)
	if err != nil {
		return nil, "", "", fmt.Errorf("encode public key: %w", err)
	}
	return priv, pubPEM, kid, nil
}

// EncryptTestPEM encrypts a plaintext PEM private key for storage in the DB.
// Used by tests that manually insert signing keys.
func EncryptTestPEM(priv *rsa.PrivateKey, serverSecret string) (string, error) {
	pemBytes, err := encodePrivateKeyPEM(priv)
	if err != nil {
		return "", fmt.Errorf("encode private key: %w", err)
	}
	return encryptPEM(pemBytes, serverSecret)
}

// BuildExpiredRS256Token creates a valid RS256 JWT with a past expiry signed by
// the given private key. Used to test the ErrExpired path.
func BuildExpiredRS256Token(priv *rsa.PrivateKey, kid, audience, issuer string, now time.Time) string {
	claims := &Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   "user_expired",
			Audience:  gojwt.ClaimStrings{audience},
			ExpiresAt: gojwt.NewNumericDate(now.Add(-2 * time.Hour)), // expired 2h ago
			NotBefore: gojwt.NewNumericDate(now.Add(-3 * time.Hour)),
			IssuedAt:  gojwt.NewNumericDate(now.Add(-3 * time.Hour)),
			ID:        "expired-test-jti",
		},
		TokenType: "session",
		SessionID: "",
		MFAPassed: false,
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, _ := token.SignedString(priv)
	return signed
}
