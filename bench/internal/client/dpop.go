// Package client — DPoP prover (RFC 9449) using ECDSA P-256.
package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"
)

// SignOpts are extra params for Sign (currently just nonce).
type SignOpts struct {
	Nonce string // optional `nonce` claim from a previous DPoP-Nonce header
	Ath   string // optional access-token hash for `ath` claim
}

// Prover signs DPoP proofs. Two modes:
//   - "batched" : cache last proof for 30s; reuse if (htm, htu) match
//   - "resign"  : always generate fresh
type Prover struct {
	mode    string
	priv    *ecdsa.PrivateKey
	jwkJSON []byte
	jktB64  string

	mu       sync.Mutex
	cachedAt time.Time
	cacheKey string
	cached   string
}

// NewProver generates a fresh ECDSA P-256 keypair and returns a Prover.
func NewProver(mode string) (*Prover, error) {
	if mode != "batched" && mode != "resign" {
		mode = "batched"
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("dpop keygen: %w", err)
	}
	x := padLeft(priv.PublicKey.X.Bytes(), 32)
	y := padLeft(priv.PublicKey.Y.Bytes(), 32)
	jwk := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   b64url(x),
		"y":   b64url(y),
	}
	// Canonical thumbprint per RFC 7638: sorted required members.
	canon := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":"%s","y":"%s"}`, jwk["x"], jwk["y"])
	sum := sha256.Sum256([]byte(canon))
	jwkJSON, _ := json.Marshal(jwk)
	return &Prover{
		mode:    mode,
		priv:    priv,
		jwkJSON: jwkJSON,
		jktB64:  b64url(sum[:]),
	}, nil
}

// JKT returns the SHA-256 thumbprint of the public JWK (base64url).
func (p *Prover) JKT() string { return p.jktB64 }

// PublicJWK returns the JSON-marshalled public JWK.
func (p *Prover) PublicJWK() []byte { return p.jwkJSON }

// Sign returns a DPoP proof JWT for the given method+url.
func (p *Prover) Sign(method, rawURL string, opts SignOpts) (string, error) {
	htm := strings.ToUpper(method)
	htu, err := normalizeHTU(rawURL)
	if err != nil {
		return "", err
	}
	cacheKey := htm + " " + htu + " " + opts.Nonce + " " + opts.Ath
	if p.mode == "batched" {
		p.mu.Lock()
		if p.cached != "" && p.cacheKey == cacheKey && time.Since(p.cachedAt) < 30*time.Second {
			out := p.cached
			p.mu.Unlock()
			return out, nil
		}
		p.mu.Unlock()
	}

	// Build header
	hdr := map[string]any{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": json.RawMessage(p.jwkJSON),
	}
	hdrBytes, _ := json.Marshal(hdr)

	// jti: 16 random bytes hex.
	var jti [16]byte
	if _, err := rand.Read(jti[:]); err != nil {
		return "", err
	}
	jtiHex := fmt.Sprintf("%x", jti)

	claims := map[string]any{
		"htm": htm,
		"htu": htu,
		"iat": time.Now().Unix(),
		"jti": jtiHex,
	}
	if opts.Nonce != "" {
		claims["nonce"] = opts.Nonce
	}
	if opts.Ath != "" {
		claims["ath"] = opts.Ath
	}
	claimBytes, _ := json.Marshal(claims)

	signingInput := b64url(hdrBytes) + "." + b64url(claimBytes)
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, p.priv, hash[:])
	if err != nil {
		return "", err
	}
	// JWS ES256: r||s, each 32 bytes big-endian.
	sig := append(padLeft(r.Bytes(), 32), padLeft(s.Bytes(), 32)...)
	jwt := signingInput + "." + b64url(sig)

	if p.mode == "batched" {
		p.mu.Lock()
		p.cached = jwt
		p.cacheKey = cacheKey
		p.cachedAt = time.Now()
		p.mu.Unlock()
	}
	return jwt, nil
}

// normalizeHTU strips query+fragment per RFC 9449 §4.2.
func normalizeHTU(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func padLeft(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

// keep big.Int + binary imports usable in case of future expansion
var (
	_ = big.NewInt
	_ = binary.BigEndian
)
