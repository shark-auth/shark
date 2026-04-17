package jwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
)

// generateRSAKeypair generates a 2048-bit RSA keypair.
func generateRSAKeypair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// computeKID derives the key ID: base64url(SHA-256(DER-encoded public key))[:16].
func computeKID(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:])[:16], nil
}

// encodePublicKeyPEM encodes an RSA public key as a PKIX PEM block.
func encodePublicKeyPEM(pub *rsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return string(pem.EncodeToMemory(block)), nil
}

// encodePrivateKeyPEM encodes an RSA private key as a PKCS#8 PEM block.
// The returned bytes contain the plaintext PEM — call encryptPEM before storing.
func encodePrivateKeyPEM(priv *rsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return pem.EncodeToMemory(block), nil
}

// deriveAESKey derives a 32-byte AES-GCM key from the server secret with a
// domain separator to prevent key-reuse with session-cookie encryption.
func deriveAESKey(serverSecret string) [32]byte {
	return sha256.Sum256([]byte(serverSecret + "jwt-key-encryption"))
}

// encryptPEM encrypts plaintext PEM bytes with AES-GCM.
// Returns base64(nonce || ciphertext).
// Wipes the plaintext slice before returning.
func encryptPEM(plainPEM []byte, serverSecret string) (string, error) {
	aesKey := deriveAESKey(serverSecret)
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plainPEM, nil)

	// Wipe plaintext
	for i := range plainPEM {
		plainPEM[i] = 0
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPEM decrypts a base64(nonce||ciphertext) blob back to PEM bytes.
// The caller is responsible for wiping the returned slice after use.
func decryptPEM(encrypted, serverSecret string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	aesKey := deriveAESKey(serverSecret)
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

// decryptPrivateKey decrypts the encrypted PEM and parses it into an *rsa.PrivateKey.
// The intermediate decrypted PEM bytes are wiped after parsing.
func decryptPrivateKey(encryptedPEM, serverSecret string) (*rsa.PrivateKey, error) {
	pemBytes, err := decryptPEM(encryptedPEM, serverSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt private key: %w", err)
	}
	defer func() {
		for i := range pemBytes {
			pemBytes[i] = 0
		}
	}()

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaKey, nil
}

// decodePublicKeyPEM parses a PEM-encoded PKIX RSA public key.
func decodePublicKeyPEM(pemStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}
	return rsaPub, nil
}

// jwkFromPublicKey builds an RFC 7517 JWK map for an RSA public key.
func jwkFromPublicKey(kid string, pub *rsa.PublicKey) map[string]interface{} {
	// n: big-endian modulus bytes, base64url (no padding)
	nBytes := pub.N.Bytes()
	// e: big-endian exponent bytes, base64url (no padding)
	e := pub.E
	eBytes := make([]byte, 0, 4)
	for e > 0 {
		eBytes = append([]byte{byte(e & 0xff)}, eBytes...)
		e >>= 8
	}
	return map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(nBytes),
		"e":   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

// pubKeyFromJWK reconstructs an RSA public key from n/e base64url strings.
// Used in tests; production code uses the stored PEM directly.
func pubKeyFromJWK(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	eInt := 0
	for _, b := range eBytes {
		eInt = (eInt << 8) | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: eInt}, nil
}
