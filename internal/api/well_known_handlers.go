package api

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"time"
)

// parsePEMPublicKey decodes a PKIX PEM-encoded RSA public key.
func parsePEMPublicKey(pemStr string) (*rsa.PublicKey, error) {
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
	nBytes := pub.N.Bytes()
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

// HandleJWKS handles GET /.well-known/jwks.json (RFC 7517).
// Returns active and recently-retired signing keys (within 2x access token TTL window).
// Cache-Control: public, max-age=300 (5 minutes).
func (s *Server) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	// Retired key cutoff: 2x access token TTL so in-flight tokens stay verifiable.
	accessTTL := s.Config.Auth.JWT.AccessTokenTTLDuration()
	retiredCutoff := time.Now().UTC().Add(-2 * accessTTL)

	keys, err := s.Store.ListJWKSCandidates(r.Context(), false, retiredCutoff)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to retrieve signing keys",
		})
		return
	}

	jwks := make([]map[string]interface{}, 0, len(keys))
	for _, sk := range keys {
		pub, err := parsePEMPublicKey(sk.PublicKeyPEM)
		if err != nil {
			// Skip malformed keys rather than returning a broken JWKS
			continue
		}
		jwks = append(jwks, jwkFromPublicKey(sk.KID, pub))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{ //nolint:errcheck
		"keys": jwks,
	})
}
