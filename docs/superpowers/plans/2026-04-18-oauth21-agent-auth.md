# OAuth 2.1 + Agent Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Shark a full OAuth 2.1 Authorization Server with first-class agent identities, MCP compatibility, and every grant type needed for the agent era.

**Architecture:** fosite-based OAuth 2.1 core embedded in the existing chi router. ES256 signing (industry standard). New `internal/oauth/` package wraps fosite with SQLite storage adapter. Agent identity is a first-class entity alongside users. Consent screen is server-rendered Go templates with React preview in dashboard.

**Tech Stack:** Go + fosite (ory/fosite) + lestrrat-go/jwx (ES256/JWK) + chi router + SQLite WAL + React admin dashboard

**Spec Reference:** `AGENT_AUTH.md` â€” canonical spec for data model, endpoints, JWT format, security model

**Note:** fosite is chosen for correctness and compliance. In the future, a custom implementation may replace it once the system is battle-tested and all edge cases are understood. This decision should be revisited after v1.0 stabilization.

---

## Progress (2026-04-18)

| Wave | Scope | Status |
|------|-------|--------|
| **A â€” Foundation** | Migration, deps (fosite + jwx), ES256, AS metadata, config, storage types | **Done** |
| **B â€” Core Grants** | fosite SQLite adapter, token/authorize endpoints, PKCE + client_credentials | **Done** |
| **C â€” Agent + Consent** | Agent CRUD API, server-rendered consent HTML, consent management API | **Done** |
| **D â€” Advanced Grants** | DCR (RFC 7591), Device Flow (RFC 8628), Token Exchange (RFC 8693) | **Done** |
| **E â€” Security Hardening** | DPoP (RFC 9449), Introspection (RFC 7662), Revocation (RFC 7009), Resource Indicators (RFC 8707) | **Done** |
| **F â€” Dashboard + Tests** | Agent/consent UI, device approval page, full OAuth smoke tests, unit coverage | **Done** |

**Smoke test status:** 181 PASS, 0 FAIL (sections 1-42).
**Unit tests:** 120+ oauth + storage tests passing.
**Phase 5 complete: all P0+P1 RFCs implemented, all waves shipped.**

---

## Revised Phase Order

```
Phase 5  â†’ OAuth 2.1 + Agent Auth (this plan â€” P0+P1 from AGENT_AUTH.md)
Phase 5.5 â†’ Token Vault (separate plan)
Phase 6  â†’ Proxy + Visual Flow Builder (separate plan)
Phase 7  â†’ SDK (separate plan, builds on OAuth 2.1 flows)
Phase 8  â†’ P2 Enterprise Features (RAR, PAR, step-up â€” deferred)
```

## RFC Compliance Matrix

| RFC | What | Priority | Wave |
|-----|------|----------|------|
| draft-ietf-oauth-v2-1 | OAuth 2.1 core | MUST | A-B |
| RFC 8414 | AS Metadata discovery | MUST | A |
| RFC 7591 / 7592 | Dynamic Client Registration | MUST | D |
| RFC 8693 | Token Exchange (delegation) | MUST | D |
| RFC 8628 | Device Authorization Grant | MUST | D |
| RFC 9449 | DPoP (proof-of-possession) | MUST | E |
| RFC 8707 | Resource Indicators | MUST | B |
| RFC 7009 | Token Revocation | MUST | E |
| RFC 7662 | Token Introspection | MUST | E |
| RFC 9068 | JWT Access Token Profile | SHOULD | B |

---

## File Structure

### New Files

```
internal/oauth/                          # NEW PACKAGE â€” OAuth 2.1 server
â”œâ”€â”€ server.go                            # fosite config, provider setup, main wiring
â”œâ”€â”€ store.go                             # fosite.Storage adapter â†’ SQLite
â”œâ”€â”€ store_test.go                        # Unit tests for storage adapter
â”œâ”€â”€ handlers.go                          # /oauth/* HTTP handlers (authorize, token)
â”œâ”€â”€ handlers_test.go                     # Handler unit tests
â”œâ”€â”€ metadata.go                          # /.well-known/oauth-authorization-server
â”œâ”€â”€ metadata_test.go                     # Metadata endpoint tests
â”œâ”€â”€ consent.go                           # Consent screen logic (approve/deny)
â”œâ”€â”€ consent_templates/                   # Go HTML templates
â”‚   â”œâ”€â”€ consent.html                     # User-facing consent screen
â”‚   â”œâ”€â”€ device_verify.html               # Device flow verification page
â”‚   â”œâ”€â”€ error.html                       # OAuth error page
â”‚   â””â”€â”€ base.html                        # Shared layout
â”œâ”€â”€ dcr.go                               # Dynamic Client Registration (RFC 7591)
â”œâ”€â”€ dcr_test.go                          # DCR tests
â”œâ”€â”€ device.go                            # Device Authorization Grant (RFC 8628)
â”œâ”€â”€ device_test.go                       # Device flow tests
â”œâ”€â”€ exchange.go                          # Token Exchange (RFC 8693)
â”œâ”€â”€ exchange_test.go                     # Token exchange tests
â”œâ”€â”€ dpop.go                              # DPoP proof validation (RFC 9449)
â”œâ”€â”€ dpop_test.go                         # DPoP tests
â”œâ”€â”€ introspect.go                        # Token Introspection (RFC 7662)
â””â”€â”€ revoke.go                            # Token Revocation (RFC 7009)

internal/storage/
â”œâ”€â”€ agents.go                            # NEW â€” Agent CRUD storage methods
â”œâ”€â”€ agents_test.go                       # NEW â€” Agent storage tests
â”œâ”€â”€ oauth.go                             # NEW â€” OAuth tables (codes, tokens, consents, device codes, DCR)
â””â”€â”€ oauth_test.go                        # NEW â€” OAuth storage tests

internal/auth/jwt/
â”œâ”€â”€ es256.go                             # NEW â€” ES256 key generation + JWK building
â””â”€â”€ es256_test.go                        # NEW â€” ES256 tests

internal/api/
â”œâ”€â”€ agent_handlers.go                    # NEW â€” /api/v1/agents/* CRUD
â”œâ”€â”€ agent_handlers_test.go               # NEW â€” Agent handler tests
â”œâ”€â”€ consent_handlers.go                  # NEW â€” /api/v1/auth/consents management
â””â”€â”€ consent_handlers_test.go             # NEW â€” Consent handler tests

cmd/shark/migrations/
â””â”€â”€ 00010_oauth.sql                      # NEW â€” All OAuth 2.1 tables

admin/src/components/
â”œâ”€â”€ agents_manage.tsx                    # NEW â€” Replace stub with full agent management
â”œâ”€â”€ consents_manage.tsx                  # NEW â€” Replace stub with consent management
â””â”€â”€ device_approve.tsx                   # NEW â€” Device flow approval UI

scripts/
â””â”€â”€ smoke_oauth.sh                       # NEW â€” OAuth 2.1 smoke test suite
```

### Modified Files

```
go.mod                                   # Add fosite, lestrrat-go/jwx
internal/storage/storage.go              # Extend Store interface with OAuth methods
internal/api/router.go                   # Mount /oauth/* routes + agent routes
internal/auth/jwt/keys.go               # Add ES256 support alongside RS256
internal/auth/jwt/manager.go            # Extend for ES256 signing
internal/config/config.go               # Add OAuth config section
admin/src/components/App.tsx             # Register new pages
admin/src/components/layout.tsx          # Update nav items
admin/src/components/empty_shell.tsx     # Remove agent/consent stubs
SMOKE_TEST.md                            # Document new smoke test sections
ATTACK.md                               # Update phase order + mark progress
```

---

## Wave A â€” Foundation (Migration + ES256 + Metadata)

### Task 1: Database Migration â€” OAuth 2.1 Tables

**Files:**
- Create: `cmd/shark/migrations/00010_oauth.sql`

- [ ] **Step 1: Write migration SQL**

```sql
-- +goose Up

-- Agents (OAuth 2.1 clients with agent identity)
CREATE TABLE agents (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    description         TEXT DEFAULT '',
    client_id           TEXT UNIQUE NOT NULL,
    client_secret_hash  TEXT,
    client_type         TEXT NOT NULL DEFAULT 'confidential'
                        CHECK (client_type IN ('confidential', 'public')),
    auth_method         TEXT NOT NULL DEFAULT 'client_secret_basic'
                        CHECK (auth_method IN ('client_secret_basic', 'client_secret_post', 'private_key_jwt', 'none')),
    jwks                TEXT,
    jwks_uri            TEXT,
    redirect_uris       TEXT NOT NULL DEFAULT '[]',
    grant_types         TEXT NOT NULL DEFAULT '["client_credentials"]',
    response_types      TEXT NOT NULL DEFAULT '["code"]',
    scopes              TEXT NOT NULL DEFAULT '[]',
    token_lifetime      INTEGER DEFAULT 900,
    metadata            TEXT DEFAULT '{}',
    logo_uri            TEXT,
    homepage_uri        TEXT,
    active              INTEGER NOT NULL DEFAULT 1,
    created_by          TEXT REFERENCES users(id),
    created_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX idx_agents_client_id ON agents(client_id);
CREATE INDEX idx_agents_active ON agents(active);

-- OAuth authorization codes (short-lived, single-use)
CREATE TABLE oauth_authorization_codes (
    code_hash               TEXT PRIMARY KEY,
    client_id               TEXT NOT NULL,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    redirect_uri            TEXT NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    code_challenge          TEXT NOT NULL,
    code_challenge_method   TEXT NOT NULL DEFAULT 'S256',
    resource                TEXT,
    authorization_details   TEXT,
    nonce                   TEXT,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- OAuth tokens (access + refresh, tracked for revocation/introspection)
CREATE TABLE oauth_tokens (
    id                      TEXT PRIMARY KEY,
    jti                     TEXT UNIQUE NOT NULL,
    client_id               TEXT NOT NULL,
    agent_id                TEXT REFERENCES agents(id),
    user_id                 TEXT REFERENCES users(id),
    token_type              TEXT NOT NULL CHECK (token_type IN ('access', 'refresh')),
    token_hash              TEXT UNIQUE NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    audience                TEXT,
    authorization_details   TEXT,
    dpop_jkt                TEXT,
    delegation_subject      TEXT,
    delegation_actor        TEXT,
    family_id               TEXT,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    revoked_at              TIMESTAMP
);
CREATE INDEX idx_oauth_tokens_family ON oauth_tokens(family_id);
CREATE INDEX idx_oauth_tokens_client ON oauth_tokens(client_id);
CREATE INDEX idx_oauth_tokens_jti ON oauth_tokens(jti);
CREATE INDEX idx_oauth_tokens_user ON oauth_tokens(user_id);

-- User consent records (per-agent, per-scope)
CREATE TABLE oauth_consents (
    id                      TEXT PRIMARY KEY,
    user_id                 TEXT NOT NULL REFERENCES users(id),
    client_id               TEXT NOT NULL,
    scope                   TEXT NOT NULL,
    authorization_details   TEXT,
    granted_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at              TIMESTAMP,
    revoked_at              TIMESTAMP
);
CREATE UNIQUE INDEX idx_oauth_consents_user_client
    ON oauth_consents(user_id, client_id) WHERE revoked_at IS NULL;

-- Device authorization codes (RFC 8628)
CREATE TABLE oauth_device_codes (
    device_code_hash        TEXT PRIMARY KEY,
    user_code               TEXT UNIQUE NOT NULL,
    client_id               TEXT NOT NULL,
    scope                   TEXT NOT NULL DEFAULT '',
    resource                TEXT,
    user_id                 TEXT REFERENCES users(id),
    status                  TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    last_polled_at          TIMESTAMP,
    poll_interval           INTEGER NOT NULL DEFAULT 5,
    expires_at              TIMESTAMP NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Dynamic Client Registration (RFC 7591)
CREATE TABLE oauth_dcr_clients (
    client_id                   TEXT PRIMARY KEY,
    registration_token_hash     TEXT UNIQUE NOT NULL,
    client_metadata             TEXT NOT NULL,
    created_at                  TIMESTAMP NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at                  TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS oauth_dcr_clients;
DROP TABLE IF EXISTS oauth_device_codes;
DROP TABLE IF EXISTS oauth_consents;
DROP TABLE IF EXISTS oauth_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS agents;
```

- [ ] **Step 2: Verify migration runs cleanly**

```bash
go build -o shark ./cmd/shark && ./shark serve --dev &
sleep 2 && sqlite3 data/dev.db ".tables" | grep -o "agents\|oauth_"
kill %1
```

Expected: `agents oauth_authorization_codes oauth_consents oauth_device_codes oauth_dcr_clients oauth_tokens`

- [ ] **Step 3: Commit**

```bash
git add cmd/shark/migrations/00010_oauth.sql
git commit -m "feat: add OAuth 2.1 tables migration (agents, codes, tokens, consents, device, DCR)"
```

---

### Task 2: Add Dependencies â€” fosite + jwx

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Add fosite and jwx**

```bash
cd /c/Users/raulg/Desktop/projects/shark
go get github.com/ory/fosite@latest
go get github.com/lestrrat-go/jwx/v2@latest
go mod tidy
```

- [ ] **Step 2: Verify build still compiles**

```bash
go build ./...
```

Expected: Clean build, no errors.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add ory/fosite and lestrrat-go/jwx for OAuth 2.1"
```

---

### Task 3: ES256 Signing Key Support

**Files:**
- Create: `internal/auth/jwt/es256.go`
- Create: `internal/auth/jwt/es256_test.go`
- Modify: `internal/auth/jwt/keys.go` (add ES256 to JWK builder)
- Modify: `internal/auth/jwt/manager.go` (support ES256 signing/verification)
- Modify: `internal/storage/storage.go` (no schema change, algorithm field already supports any string)

- [ ] **Step 1: Write ES256 test**

```go
// internal/auth/jwt/es256_test.go
package jwt

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "testing"
)

func TestGenerateES256Keypair(t *testing.T) {
    priv, pub, err := GenerateES256Keypair()
    if err != nil {
        t.Fatalf("GenerateES256Keypair: %v", err)
    }
    if priv.Curve != elliptic.P256() {
        t.Errorf("expected P-256 curve, got %v", priv.Curve)
    }
    if pub == nil {
        t.Fatal("public key is nil")
    }
}

func TestES256PEMRoundTrip(t *testing.T) {
    priv, _, err := GenerateES256Keypair()
    if err != nil {
        t.Fatal(err)
    }
    pem, err := MarshalES256PrivateKeyPEM(priv)
    if err != nil {
        t.Fatal(err)
    }
    got, err := ParseES256PrivateKeyPEM(pem)
    if err != nil {
        t.Fatal(err)
    }
    if !priv.Equal(got) {
        t.Error("round-tripped key does not match original")
    }
}

func TestES256JWK(t *testing.T) {
    priv, _, err := GenerateES256Keypair()
    if err != nil {
        t.Fatal(err)
    }
    kid := ComputeES256KID(&priv.PublicKey)
    jwk := ES256PublicJWK(&priv.PublicKey, kid)
    if jwk["kty"] != "EC" {
        t.Errorf("expected kty=EC, got %v", jwk["kty"])
    }
    if jwk["alg"] != "ES256" {
        t.Errorf("expected alg=ES256, got %v", jwk["alg"])
    }
    if jwk["crv"] != "P-256" {
        t.Errorf("expected crv=P-256, got %v", jwk["crv"])
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/auth/jwt/ -run TestGenerateES256 -v
```

Expected: FAIL â€” functions not defined.

- [ ] **Step 3: Implement ES256 key generation**

```go
// internal/auth/jwt/es256.go
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
    "math/big"
)

// GenerateES256Keypair generates a new ECDSA P-256 keypair.
func GenerateES256Keypair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, nil, fmt.Errorf("generating ES256 key: %w", err)
    }
    return priv, &priv.PublicKey, nil
}

// MarshalES256PrivateKeyPEM encodes an ECDSA private key to PEM.
func MarshalES256PrivateKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
    der, err := x509.MarshalECPrivateKey(key)
    if err != nil {
        return nil, err
    }
    return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}

// ParseES256PrivateKeyPEM decodes a PEM-encoded ECDSA private key.
func ParseES256PrivateKeyPEM(data []byte) (*ecdsa.PrivateKey, error) {
    block, _ := pem.Decode(data)
    if block == nil {
        return nil, fmt.Errorf("no PEM block found")
    }
    return x509.ParseECPrivateKey(block.Bytes)
}

// MarshalES256PublicKeyPEM encodes an ECDSA public key to PEM.
func MarshalES256PublicKeyPEM(key *ecdsa.PublicKey) ([]byte, error) {
    der, err := x509.MarshalPKIXPublicKey(key)
    if err != nil {
        return nil, err
    }
    return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// ComputeES256KID derives a key ID from an ECDSA public key.
// Uses SHA-256 of the DER-encoded public key, base64url-encoded, truncated to 16 chars.
func ComputeES256KID(pub *ecdsa.PublicKey) string {
    der, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return ""
    }
    h := sha256.Sum256(der)
    return base64.RawURLEncoding.EncodeToString(h[:])[:16]
}

// ES256PublicJWK builds an RFC 7517 JWK map from an ECDSA P-256 public key.
func ES256PublicJWK(pub *ecdsa.PublicKey, kid string) map[string]interface{} {
    return map[string]interface{}{
        "kty": "EC",
        "use": "sig",
        "alg": "ES256",
        "kid": kid,
        "crv": "P-256",
        "x":   base64.RawURLEncoding.EncodeToString(pub.X.Bytes()),
        "y":   base64.RawURLEncoding.EncodeToString(padTo32(pub.Y.Bytes())),
    }
}

// padTo32 left-pads a byte slice to 32 bytes (P-256 coordinate size).
func padTo32(b []byte) []byte {
    if len(b) >= 32 {
        return b
    }
    padded := make([]byte, 32)
    copy(padded[32-len(b):], b)
    return padded
}
```

- [ ] **Step 4: Run tests â€” verify pass**

```bash
go test ./internal/auth/jwt/ -run TestES256 -v
go test ./internal/auth/jwt/ -run TestGenerateES256 -v
```

Expected: All PASS.

- [ ] **Step 5: Extend JWKS endpoint to serve ES256 keys**

Modify `internal/api/well_known_handlers.go` â€” in the `HandleJWKS` function, after the RSA JWK building block, add ES256 JWK building:

```go
// After existing RSA block, add:
case "ES256":
    pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
    if err != nil {
        continue
    }
    ecKey, ok := pubKey.(*ecdsa.PublicKey)
    if !ok {
        continue
    }
    jwk := jwt.ES256PublicJWK(ecKey, k.KID)
    keys = append(keys, jwk)
```

The `peekAlg()` function in `manager.go` must also accept `"ES256"` alongside `"RS256"`.

- [ ] **Step 6: Run full JWT test suite**

```bash
go test ./internal/auth/jwt/... -v
```

Expected: All existing + new tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/auth/jwt/es256.go internal/auth/jwt/es256_test.go internal/auth/jwt/keys.go internal/auth/jwt/manager.go internal/api/well_known_handlers.go
git commit -m "feat: add ES256 signing key support alongside RS256"
```

---

### Task 4: Authorization Server Metadata Endpoint

**Files:**
- Create: `internal/oauth/metadata.go`
- Create: `internal/oauth/metadata_test.go`

- [ ] **Step 1: Write metadata test**

```go
// internal/oauth/metadata_test.go
package oauth

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestMetadataEndpoint(t *testing.T) {
    issuer := "https://auth.example.com"
    handler := MetadataHandler(issuer)
    
    req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
    w := httptest.NewRecorder()
    handler.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }
    
    var meta map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
        t.Fatal(err)
    }
    
    // RFC 8414 required fields
    if meta["issuer"] != issuer {
        t.Errorf("issuer = %v, want %v", meta["issuer"], issuer)
    }
    if meta["authorization_endpoint"] == nil {
        t.Error("missing authorization_endpoint")
    }
    if meta["token_endpoint"] == nil {
        t.Error("missing token_endpoint")
    }
    if meta["jwks_uri"] == nil {
        t.Error("missing jwks_uri")
    }
    
    // OAuth 2.1 requirements
    methods, ok := meta["code_challenge_methods_supported"].([]interface{})
    if !ok || len(methods) == 0 {
        t.Error("missing code_challenge_methods_supported")
    }
    if methods[0] != "S256" {
        t.Errorf("expected S256, got %v", methods[0])
    }
    
    // MCP requirement â€” DCR
    if meta["registration_endpoint"] == nil {
        t.Error("missing registration_endpoint (MCP requires DCR)")
    }
}
```

- [ ] **Step 2: Implement metadata handler**

```go
// internal/oauth/metadata.go
package oauth

import (
    "encoding/json"
    "net/http"
)

// MetadataHandler returns an http.HandlerFunc that serves RFC 8414
// Authorization Server Metadata. This is the MCP discovery entrypoint.
func MetadataHandler(issuer string) http.HandlerFunc {
    meta := map[string]interface{}{
        "issuer":                 issuer,
        "authorization_endpoint": issuer + "/oauth/authorize",
        "token_endpoint":         issuer + "/oauth/token",
        "jwks_uri":               issuer + "/.well-known/jwks.json",
        "registration_endpoint":  issuer + "/oauth/register",
        "revocation_endpoint":    issuer + "/oauth/revoke",
        "introspection_endpoint": issuer + "/oauth/introspect",
        "device_authorization_endpoint": issuer + "/oauth/device",

        "scopes_supported":                    []string{"openid", "profile", "email"},
        "response_types_supported":            []string{"code"},
        "response_modes_supported":            []string{"query"},
        "grant_types_supported": []string{
            "authorization_code",
            "client_credentials",
            "refresh_token",
            "urn:ietf:params:oauth:grant-type:device_code",
            "urn:ietf:params:oauth:grant-type:token-exchange",
        },
        "token_endpoint_auth_methods_supported": []string{
            "client_secret_basic",
            "client_secret_post",
            "private_key_jwt",
            "none",
        },
        "code_challenge_methods_supported": []string{"S256"},
        "dpop_signing_alg_values_supported": []string{"ES256", "RS256"},

        "token_endpoint_auth_signing_alg_values_supported": []string{"ES256", "RS256"},
        "service_documentation": "https://sharkauth.com/docs",
    }

    body, _ := json.MarshalIndent(meta, "", "  ")

    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("Cache-Control", "public, max-age=3600")
        w.Write(body)
    }
}
```

- [ ] **Step 3: Run test**

```bash
go test ./internal/oauth/ -run TestMetadata -v
```

Expected: PASS.

- [ ] **Step 4: Mount in router**

Add to `internal/api/router.go` in `NewServer`, after the JWKS endpoint:

```go
// OAuth 2.1 AS Metadata (RFC 8414) â€” MCP discovery
r.Get("/.well-known/oauth-authorization-server", oauth.MetadataHandler(cfg.Server.BaseURL))
```

- [ ] **Step 5: Commit**

```bash
git add internal/oauth/metadata.go internal/oauth/metadata_test.go internal/api/router.go
git commit -m "feat: add OAuth 2.1 Authorization Server Metadata endpoint (RFC 8414)"
```

---

### Task 5: OAuth Config Section

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Add OAuthServerConfig to Config struct**

```go
// Add to Config struct:
OAuthServer OAuthServerConfig `koanf:"oauth_server"`

// New type:
type OAuthServerConfig struct {
    Enabled              bool   `koanf:"enabled"`
    Issuer               string `koanf:"issuer"`                  // defaults to server.base_url
    SigningAlgorithm     string `koanf:"signing_algorithm"`       // ES256 (default) | RS256
    AccessTokenLifetime  string `koanf:"access_token_lifetime"`   // default: 15m
    RefreshTokenLifetime string `koanf:"refresh_token_lifetime"`  // default: 30d
    AuthCodeLifetime     string `koanf:"auth_code_lifetime"`      // default: 60s
    DeviceCodeLifetime   string `koanf:"device_code_lifetime"`    // default: 15m
    ConsentTemplate      string `koanf:"consent_template"`        // path to custom template dir
    RequireDPoP          bool   `koanf:"require_dpop"`            // require DPoP for all clients
}

func (o *OAuthServerConfig) AccessTokenLifetimeDuration() time.Duration {
    return parseDuration(o.AccessTokenLifetime, 15*time.Minute)
}

func (o *OAuthServerConfig) RefreshTokenLifetimeDuration() time.Duration {
    return parseDuration(o.RefreshTokenLifetime, 30*24*time.Hour)
}

func (o *OAuthServerConfig) AuthCodeLifetimeDuration() time.Duration {
    return parseDuration(o.AuthCodeLifetime, 60*time.Second)
}

func (o *OAuthServerConfig) DeviceCodeLifetimeDuration() time.Duration {
    return parseDuration(o.DeviceCodeLifetime, 15*time.Minute)
}
```

- [ ] **Step 2: Add defaults in Load()**

```go
// Add to defaults map:
"oauth_server.enabled":                true,
"oauth_server.signing_algorithm":      "ES256",
"oauth_server.access_token_lifetime":  "15m",
"oauth_server.refresh_token_lifetime": "30d",
"oauth_server.auth_code_lifetime":     "60s",
"oauth_server.device_code_lifetime":   "15m",
"oauth_server.require_dpop":           false,
```

- [ ] **Step 3: Verify config loads**

```bash
go test ./internal/config/ -v
```

- [ ] **Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add oauth_server config section with ES256 default"
```

---

### Task 6: Storage Interface Extensions

**Files:**
- Create: `internal/storage/agents.go` (entity type only in this task)
- Modify: `internal/storage/storage.go` (add new interface methods)
- Create: `internal/storage/oauth.go` (entity types only in this task)

- [ ] **Step 1: Define Agent entity type**

```go
// internal/storage/agents.go
package storage

import "time"

// Agent represents an OAuth 2.1 client with agent identity.
type Agent struct {
    ID               string         `json:"id"`
    Name             string         `json:"name"`
    Description      string         `json:"description"`
    ClientID         string         `json:"client_id"`
    ClientSecretHash string         `json:"-"`
    ClientType       string         `json:"client_type"`
    AuthMethod       string         `json:"auth_method"`
    JWKS             string         `json:"jwks,omitempty"`
    JWKSURI          string         `json:"jwks_uri,omitempty"`
    RedirectURIs     []string       `json:"redirect_uris"`
    GrantTypes       []string       `json:"grant_types"`
    ResponseTypes    []string       `json:"response_types"`
    Scopes           []string       `json:"scopes"`
    TokenLifetime    int            `json:"token_lifetime"`
    Metadata         map[string]any `json:"metadata"`
    LogoURI          string         `json:"logo_uri,omitempty"`
    HomepageURI      string         `json:"homepage_uri,omitempty"`
    Active           bool           `json:"active"`
    CreatedBy        string         `json:"created_by,omitempty"`
    CreatedAt        time.Time      `json:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at"`
}

// ListAgentsOpts configures agent list queries.
type ListAgentsOpts struct {
    Limit    int
    Offset   int
    Search   string
    Active   *bool
}
```

- [ ] **Step 2: Define OAuth entity types**

```go
// internal/storage/oauth.go
package storage

import "time"

// OAuthAuthorizationCode represents a short-lived authorization code.
type OAuthAuthorizationCode struct {
    CodeHash             string    `json:"-"`
    ClientID             string    `json:"client_id"`
    UserID               string    `json:"user_id"`
    RedirectURI          string    `json:"redirect_uri"`
    Scope                string    `json:"scope"`
    CodeChallenge        string    `json:"-"`
    CodeChallengeMethod  string    `json:"-"`
    Resource             string    `json:"resource,omitempty"`
    AuthorizationDetails string    `json:"authorization_details,omitempty"`
    Nonce                string    `json:"nonce,omitempty"`
    ExpiresAt            time.Time `json:"expires_at"`
    CreatedAt            time.Time `json:"created_at"`
}

// OAuthToken represents an access or refresh token record.
type OAuthToken struct {
    ID                   string     `json:"id"`
    JTI                  string     `json:"jti"`
    ClientID             string     `json:"client_id"`
    AgentID              string     `json:"agent_id,omitempty"`
    UserID               string     `json:"user_id,omitempty"`
    TokenType            string     `json:"token_type"`
    TokenHash            string     `json:"-"`
    Scope                string     `json:"scope"`
    Audience             string     `json:"audience,omitempty"`
    AuthorizationDetails string     `json:"authorization_details,omitempty"`
    DPoPJKT              string     `json:"dpop_jkt,omitempty"`
    DelegationSubject    string     `json:"delegation_subject,omitempty"`
    DelegationActor      string     `json:"delegation_actor,omitempty"`
    FamilyID             string     `json:"family_id,omitempty"`
    ExpiresAt            time.Time  `json:"expires_at"`
    CreatedAt            time.Time  `json:"created_at"`
    RevokedAt            *time.Time `json:"revoked_at,omitempty"`
}

// OAuthConsent represents a user's consent grant for an agent.
type OAuthConsent struct {
    ID                   string     `json:"id"`
    UserID               string     `json:"user_id"`
    ClientID             string     `json:"client_id"`
    Scope                string     `json:"scope"`
    AuthorizationDetails string     `json:"authorization_details,omitempty"`
    GrantedAt            time.Time  `json:"granted_at"`
    ExpiresAt            *time.Time `json:"expires_at,omitempty"`
    RevokedAt            *time.Time `json:"revoked_at,omitempty"`
}

// OAuthDeviceCode represents a pending device authorization.
type OAuthDeviceCode struct {
    DeviceCodeHash string     `json:"-"`
    UserCode       string     `json:"user_code"`
    ClientID       string     `json:"client_id"`
    Scope          string     `json:"scope"`
    Resource       string     `json:"resource,omitempty"`
    UserID         string     `json:"user_id,omitempty"`
    Status         string     `json:"status"`
    LastPolledAt   *time.Time `json:"last_polled_at,omitempty"`
    PollInterval   int        `json:"poll_interval"`
    ExpiresAt      time.Time  `json:"expires_at"`
    CreatedAt      time.Time  `json:"created_at"`
}
```

- [ ] **Step 3: Add methods to Store interface**

Add to `internal/storage/storage.go` in the `Store` interface:

```go
    // Agents
    CreateAgent(ctx context.Context, agent *Agent) error
    GetAgentByID(ctx context.Context, id string) (*Agent, error)
    GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error)
    ListAgents(ctx context.Context, opts ListAgentsOpts) ([]*Agent, int, error)
    UpdateAgent(ctx context.Context, agent *Agent) error
    DeactivateAgent(ctx context.Context, id string) error

    // OAuth Authorization Codes
    CreateAuthorizationCode(ctx context.Context, code *OAuthAuthorizationCode) error
    GetAuthorizationCode(ctx context.Context, codeHash string) (*OAuthAuthorizationCode, error)
    DeleteAuthorizationCode(ctx context.Context, codeHash string) error
    DeleteExpiredAuthorizationCodes(ctx context.Context) (int64, error)

    // OAuth Tokens
    CreateOAuthToken(ctx context.Context, token *OAuthToken) error
    GetOAuthTokenByJTI(ctx context.Context, jti string) (*OAuthToken, error)
    GetOAuthTokenByHash(ctx context.Context, tokenHash string) (*OAuthToken, error)
    RevokeOAuthToken(ctx context.Context, id string) error
    RevokeOAuthTokensByClientID(ctx context.Context, clientID string) (int64, error)
    RevokeOAuthTokenFamily(ctx context.Context, familyID string) (int64, error)
    ListOAuthTokensByAgentID(ctx context.Context, agentID string, limit int) ([]*OAuthToken, error)
    DeleteExpiredOAuthTokens(ctx context.Context) (int64, error)

    // OAuth Consents
    CreateOAuthConsent(ctx context.Context, consent *OAuthConsent) error
    GetActiveConsent(ctx context.Context, userID, clientID string) (*OAuthConsent, error)
    ListConsentsByUserID(ctx context.Context, userID string) ([]*OAuthConsent, error)
    RevokeOAuthConsent(ctx context.Context, id string) error

    // Device Codes
    CreateDeviceCode(ctx context.Context, dc *OAuthDeviceCode) error
    GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*OAuthDeviceCode, error)
    GetDeviceCodeByHash(ctx context.Context, hash string) (*OAuthDeviceCode, error)
    UpdateDeviceCodeStatus(ctx context.Context, hash string, status string, userID string) error
    UpdateDeviceCodePolledAt(ctx context.Context, hash string) error
    DeleteExpiredDeviceCodes(ctx context.Context) (int64, error)
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/storage/...
```

Expected: Compilation fails (methods not implemented on SQLiteStore yet) â€” that's correct for now. The interface is defined, implementation comes in the fosite storage adapter task.

- [ ] **Step 5: Commit**

```bash
git add internal/storage/agents.go internal/storage/oauth.go internal/storage/storage.go
git commit -m "feat: define Agent + OAuth entity types and Store interface extensions"
```

---

## Wave B â€” Core Grants (fosite Integration)

### Task 7: fosite Storage Adapter

**Files:**
- Create: `internal/oauth/store.go`
- Create: `internal/oauth/store_test.go`
- Create: `internal/storage/agents_sqlite.go` (implement Agent CRUD)
- Create: `internal/storage/oauth_sqlite.go` (implement OAuth CRUD)

This is the largest single task. The fosite storage adapter maps fosite's `Storage` interface to our SQLite Store. This is where fosite meets our data model.

- [ ] **Step 1: Implement Agent CRUD in SQLite**

Create `internal/storage/agents_sqlite.go` with all 6 Agent methods. JSON columns (`redirect_uris`, `grant_types`, `response_types`, `scopes`, `metadata`) marshal/unmarshal like existing `Application` pattern in `storage/applications.go`.

- [ ] **Step 2: Implement OAuth CRUD in SQLite**

Create `internal/storage/oauth_sqlite.go` with all authorization code, token, consent, and device code methods. Follow existing patterns (nanoid IDs, SHA-256 hashes, time formatting).

- [ ] **Step 3: Write storage tests**

Create `internal/storage/agents_test.go` and `internal/storage/oauth_test.go`. Test CRUD operations, edge cases (duplicate client_id, expired token cleanup, consent uniqueness, family-based token revocation).

- [ ] **Step 4: Implement fosite Storage adapter**

Create `internal/oauth/store.go` â€” implements `fosite.Storage` interface by delegating to our `storage.Store`. Key interfaces to implement:
- `fosite.ClientManager` â€” `GetClient()` maps to `GetAgentByClientID()`
- `oauth2.CoreStorage` â€” maps auth codes, access tokens, refresh tokens to our tables
- `oauth2.ResourceOwnerPasswordCredentialsGrantStorage` â€” NOT implemented (removed in OAuth 2.1)
- `pkce.PKCERequestStorage` â€” stores/validates PKCE challenges

```go
// internal/oauth/store.go
package oauth

import (
    "context"
    "github.com/ory/fosite"
    "github.com/shark-auth/shark/internal/storage"
)

// FositeStore adapts our storage.Store to fosite's storage interfaces.
type FositeStore struct {
    store storage.Store
}

func NewFositeStore(store storage.Store) *FositeStore {
    return &FositeStore{store: store}
}

// GetClient implements fosite.ClientManager.
func (s *FositeStore) GetClient(ctx context.Context, id string) (fosite.Client, error) {
    agent, err := s.store.GetAgentByClientID(ctx, id)
    if err != nil {
        return nil, fosite.ErrNotFound
    }
    return agentToFositeClient(agent), nil
}

// agentToFositeClient wraps our Agent as a fosite.Client.
func agentToFositeClient(a *storage.Agent) fosite.Client {
    // Implementation maps Agent fields to fosite.DefaultClient
    // ...
}
```

- [ ] **Step 5: Write fosite store tests**

Test the adapter layer: GetClient returns correct fosite.Client, auth code round-trip through fosite interfaces, token storage/retrieval.

- [ ] **Step 6: Run all storage tests**

```bash
go test ./internal/storage/... -v
go test ./internal/oauth/... -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/storage/agents_sqlite.go internal/storage/agents_test.go internal/storage/oauth_sqlite.go internal/storage/oauth_test.go internal/oauth/store.go internal/oauth/store_test.go
git commit -m "feat: implement fosite SQLite storage adapter + Agent/OAuth CRUD"
```

---

### Task 8: OAuth Server Setup + Token Endpoint

**Files:**
- Create: `internal/oauth/server.go`
- Create: `internal/oauth/handlers.go`
- Create: `internal/oauth/handlers_test.go`

- [ ] **Step 1: Create OAuth server with fosite provider**

```go
// internal/oauth/server.go
package oauth

import (
    "crypto/ecdsa"
    "time"

    "github.com/ory/fosite"
    "github.com/ory/fosite/compose"
    "github.com/shark-auth/shark/internal/config"
    "github.com/shark-auth/shark/internal/storage"
)

// Server holds the fosite OAuth2 provider and related dependencies.
type Server struct {
    Provider fosite.OAuth2Provider
    Store    *FositeStore
    Config   *config.OAuthServerConfig
    Issuer   string
}

// NewServer creates a new OAuth 2.1 server with fosite.
func NewServer(store storage.Store, cfg *config.Config, signingKey *ecdsa.PrivateKey) *Server {
    fositeStore := NewFositeStore(store)

    fositeConfig := &fosite.Config{
        AccessTokenLifespan:   cfg.OAuthServer.AccessTokenLifetimeDuration(),
        RefreshTokenLifespan:  cfg.OAuthServer.RefreshTokenLifetimeDuration(),
        AuthorizeCodeLifespan: cfg.OAuthServer.AuthCodeLifetimeDuration(),
        EnforcePKCE:                 true,
        EnforcePKCEForPublicClients: true,
        EnablePKCEPlainChallengeMethod: false, // S256 only (OAuth 2.1)
        TokenURL:              cfg.Server.BaseURL + "/oauth/token",
        // ... additional config
    }

    provider := compose.ComposeAllEnabled(
        fositeConfig,
        fositeStore,
        signingKey,
    )

    return &Server{
        Provider: provider,
        Store:    fositeStore,
        Config:   &cfg.OAuthServer,
        Issuer:   cfg.Server.BaseURL,
    }
}
```

- [ ] **Step 2: Implement /oauth/authorize and /oauth/token handlers**

```go
// internal/oauth/handlers.go
package oauth

import (
    "net/http"
    "github.com/ory/fosite"
)

// HandleAuthorize handles GET /oauth/authorize â€” shows consent screen.
func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    ar, err := s.Provider.NewAuthorizeRequest(ctx, r)
    if err != nil {
        s.Provider.WriteAuthorizeError(ctx, w, ar, err)
        return
    }
    // Check if user is logged in (session cookie)
    // Check if consent already granted
    // If yes: auto-approve
    // If no: render consent screen
    // ...
}

// HandleAuthorizeDecision handles POST /oauth/authorize â€” user approves/denies.
func (s *Server) HandleAuthorizeDecision(w http.ResponseWriter, r *http.Request) {
    // Parse form (approved=true/false)
    // Store consent
    // Complete authorize request via fosite
    // Redirect with code
}

// HandleToken handles POST /oauth/token â€” all grant types.
func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    session := NewOAuthSession("") // will be populated by fosite

    ar, err := s.Provider.NewAccessRequest(ctx, r, session)
    if err != nil {
        s.Provider.WriteAccessError(ctx, w, ar, err)
        return
    }

    // Grant requested scopes (validated against agent's allowed scopes)
    for _, scope := range ar.GetRequestedScopes() {
        ar.GrantScope(scope)
    }

    response, err := s.Provider.NewAccessResponse(ctx, ar)
    if err != nil {
        s.Provider.WriteAccessError(ctx, w, ar, err)
        return
    }

    s.Provider.WriteAccessResponse(ctx, w, ar, response)
}
```

- [ ] **Step 3: Write handler tests**

Test client_credentials grant (happy path), auth code + PKCE flow (end-to-end), missing PKCE rejection, invalid client rejection.

- [ ] **Step 4: Mount OAuth routes in router**

Add to `internal/api/router.go`:

```go
// OAuth 2.1 endpoints (outside /api/v1 â€” standard OAuth paths)
r.Route("/oauth", func(r chi.Router) {
    r.Get("/authorize", oauthServer.HandleAuthorize)
    r.Post("/authorize", oauthServer.HandleAuthorizeDecision)
    r.Post("/token", oauthServer.HandleToken)
})
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/oauth/... -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/oauth/server.go internal/oauth/handlers.go internal/oauth/handlers_test.go internal/api/router.go
git commit -m "feat: OAuth 2.1 token endpoint with auth code + PKCE and client credentials grants"
```

---

## Wave C â€” Agent Management + Consent UI

### Task 9: Agent CRUD API Handlers

**Files:**
- Create: `internal/api/agent_handlers.go`
- Create: `internal/api/agent_handlers_test.go`

Endpoints (admin API key required):
- `POST /api/v1/agents` â€” register agent, return client_secret ONCE
- `GET /api/v1/agents` â€” list agents (search, pagination)
- `GET /api/v1/agents/{id}` â€” get agent details
- `PATCH /api/v1/agents/{id}` â€” update agent
- `DELETE /api/v1/agents/{id}` â€” deactivate + revoke all tokens
- `GET /api/v1/agents/{id}/tokens` â€” list active tokens
- `POST /api/v1/agents/{id}/tokens/revoke-all` â€” revoke all tokens
- `GET /api/v1/agents/{id}/audit` â€” agent audit trail

Follow existing handler patterns from `application_handlers.go` (client_secret shown once, SHA-256 hash stored, prefix for UX display).

- [ ] Steps: Write tests â†’ implement handlers â†’ mount in router â†’ verify â†’ commit

---

### Task 10: Consent Screen â€” Server-Rendered Go Templates

**Files:**
- Create: `internal/oauth/consent_templates/base.html`
- Create: `internal/oauth/consent_templates/consent.html`
- Create: `internal/oauth/consent_templates/error.html`
- Create: `internal/oauth/consent.go`

The consent screen is what users see when an agent requests authorization. Must be:
- Server-rendered (no SPA dependency â€” security boundary)
- Dark theme matching admin dashboard aesthetic
- Shows: agent name, logo, requested scopes, approve/deny buttons
- Remembers consent (auto-approve on repeat if scope unchanged)

- [ ] Steps: Create templates â†’ implement consent logic â†’ wire into HandleAuthorize â†’ test â†’ commit

---

### Task 11: Consent Management API

**Files:**
- Create: `internal/api/consent_handlers.go`
- Create: `internal/api/consent_handlers_test.go`

Endpoints (session auth â€” user manages their own consents):
- `GET /api/v1/auth/consents` â€” list active consents
- `DELETE /api/v1/auth/consents/{id}` â€” revoke consent + all associated tokens

- [ ] Steps: Write tests â†’ implement â†’ mount â†’ verify â†’ commit

---

## Wave D â€” Advanced Grants

### Task 12: Dynamic Client Registration (RFC 7591)

**Files:**
- Create: `internal/oauth/dcr.go`
- Create: `internal/oauth/dcr_test.go`

Endpoints:
- `POST /oauth/register` â€” register new client
- `GET /oauth/register/{client_id}` â€” get client info (RFC 7592)
- `PUT /oauth/register/{client_id}` â€” update client (RFC 7592)
- `DELETE /oauth/register/{client_id}` â€” delete client (RFC 7592)

Protected by registration_access_token (SHA-256 hashed in `oauth_dcr_clients`). Rate limited to 5/min/IP.

MCP compatibility: agents discover Shark via AS metadata, then register via DCR. Client metadata validated per RFC 7591 Section 2.

- [ ] Steps: Write tests â†’ implement DCR handler â†’ implement management handlers â†’ mount â†’ verify â†’ commit

---

### Task 13: Device Authorization Grant (RFC 8628)

**Files:**
- Create: `internal/oauth/device.go`
- Create: `internal/oauth/device_test.go`
- Create: `internal/oauth/consent_templates/device_verify.html`

Endpoints:
- `POST /oauth/device` â€” request device code + user code
- `GET /oauth/device/verify` â€” render verification page (user enters code)
- `POST /oauth/device/verify` â€” user approves/denies

Token endpoint handling: `grant_type=urn:ietf:params:oauth:grant-type:device_code` â€” polls until approved/denied/expired. Enforce `interval` parameter, return `slow_down` on violation.

User code format: `SHARK-XXXX` (4 chars, uppercase, no ambiguous chars I/O/0/1).

- [ ] Steps: Write tests â†’ implement device flow handlers â†’ implement verification page â†’ add polling logic to token endpoint â†’ mount â†’ verify â†’ commit

---

### Task 14: Token Exchange (RFC 8693)

**Files:**
- Create: `internal/oauth/exchange.go`
- Create: `internal/oauth/exchange_test.go`

Grant type: `urn:ietf:params:oauth:grant-type:token-exchange`

Parameters: `subject_token`, `subject_token_type`, `actor_token` (optional), `actor_token_type`, `scope`, `resource`

Key behaviors:
- Validate subject_token (must be a valid Shark-issued JWT)
- Build delegation chain in `act` claim (nested)
- Scope narrowing: delegated token scope <= subject_token scope
- `may_act` claim enforcement: only specified agents can receive delegation
- Audit trail: log full delegation chain

This is the agent-to-agent delegation primitive.

- [ ] Steps: Write tests â†’ implement exchange handler â†’ add to token endpoint dispatch â†’ verify delegation chain in JWT â†’ commit

---

## Wave E â€” Security Hardening

### Task 15: DPoP Proof Validation (RFC 9449)

**Files:**
- Create: `internal/oauth/dpop.go`
- Create: `internal/oauth/dpop_test.go`

DPoP binds access tokens to a client's key pair. Two integration points:
1. **Token endpoint**: validate `DPoP` header proof JWT â†’ store `cnf.jkt` (key thumbprint) in issued token
2. **Resource server middleware**: validate `DPoP` proof matches token's `cnf.jkt`

Validation checks per RFC 9449:
- Proof JWT has `typ: dpop+jwt` header
- `htm` matches HTTP method, `htu` matches endpoint URL
- `iat` within 60s window (replay prevention)
- `jti` is unique (per-request, stored in cache for dedup window)
- At resource server: `ath` (access token hash) matches
- Key thumbprint (`jkt`) computed per RFC 7638

Token type changes from `Bearer` to `DPoP` when DPoP is used.

- [ ] Steps: Write tests â†’ implement DPoP validation â†’ integrate with token endpoint â†’ add middleware for resource server validation â†’ commit

---

### Task 16: Token Introspection + Revocation (RFC 7662, RFC 7009)

**Files:**
- Create: `internal/oauth/introspect.go`
- Create: `internal/oauth/revoke.go`
- Create: `internal/oauth/introspect_test.go`

Endpoints:
- `POST /oauth/introspect` â€” returns `{active: true/false, ...claims}`. Authenticated (client credentials or admin key).
- `POST /oauth/revoke` â€” accepts `token` + optional `token_type_hint`. Always returns 200 (RFC 7009 Â§2.2 â€” no error on invalid token). Revokes both access + refresh in same family.

Introspection response includes: `active`, `scope`, `client_id`, `sub`, `exp`, `iat`, `aud`, `iss`, `jti`, `token_type`, `agent_id`, `act` (delegation).

- [ ] Steps: Write tests â†’ implement introspection â†’ implement revocation â†’ mount â†’ verify â†’ commit

---

### Task 17: Resource Indicators (RFC 8707)

**Files:**
- Modify: `internal/oauth/handlers.go`
- Modify: `internal/oauth/store.go`

Accept `resource` parameter on authorize + token endpoints. Bind to `aud` claim in issued JWT. Validate against agent's registered `scopes` (if resource-specific scopes configured).

This ensures tokens are audience-restricted â€” a token for `https://api.example.com` can't be used at `https://other.example.com`.

- [ ] Steps: Add resource parameter handling â†’ bind to JWT audience â†’ validate at introspection â†’ test â†’ commit

---

## Wave F â€” Dashboard + Tests

### Task 18: Agent Management Dashboard Page

**Files:**
- Create: `admin/src/components/agents_manage.tsx`
- Modify: `admin/src/components/App.tsx`
- Modify: `admin/src/components/layout.tsx`
- Modify: `admin/src/components/empty_shell.tsx` (remove stub)

Full CRUD page following `applications.tsx` pattern:
- Split grid layout (table + detail panel)
- Header with stats (registered count, active count)
- Search + filter toolbar
- Clickable table with agent details
- Right slide-over with tabs: Config, Tokens, Consents, Audit
- Modals: Create agent, Rotate secret, Confirm deactivate
- CLI footer with equivalent commands

- [ ] Steps: Create component â†’ register in App.tsx + layout.tsx â†’ remove stub â†’ build admin assets â†’ verify in browser â†’ commit

---

### Task 19: Consent Management Dashboard Page

**Files:**
- Create: `admin/src/components/consents_manage.tsx`
- Modify: `admin/src/components/App.tsx`
- Modify: `admin/src/components/empty_shell.tsx` (remove stub)

Shows user â†’ agent consent grants. Admin can view all consents, users can view/revoke their own.
- Table: user email, agent name, scopes, granted_at, status
- Revoke action (with confirmation)
- Filter by agent, user, scope

- [ ] Steps: Create component â†’ register â†’ remove stub â†’ build â†’ verify â†’ commit

---

### Task 20: Device Flow Approval Page (React)

**Files:**
- Modify: `admin/src/components/device_flow.tsx` (extend existing stub)

When user visits `/admin/device-flow`, show:
- Input field for user code (SHARK-XXXX format)
- After entering code: show agent name, requested scopes, approve/deny buttons
- Success/error states
- Auto-redirect after approval

This is the React-based companion to the server-rendered device_verify.html.

- [ ] Steps: Extend existing component â†’ wire to API â†’ test â†’ commit

---

### Task 21: OAuth 2.1 Smoke Tests

**Files:**
- Create: `scripts/smoke_oauth.sh`
- Modify: `SMOKE_TEST.md`

New smoke test sections covering:

| # | Section | Verifies |
|---|---------|----------|
| 14 | AS Metadata | `/.well-known/oauth-authorization-server` returns all required fields |
| 15 | Agent CRUD | Create/list/get/update/deactivate agent via admin API |
| 16 | Client Credentials | Agent gets access token with `grant_type=client_credentials` |
| 17 | Auth Code + PKCE | Full flow: authorize â†’ consent â†’ code â†’ token â†’ /me |
| 18 | PKCE Enforcement | Missing `code_challenge` â†’ rejected |
| 19 | Refresh Token Rotation | Refresh â†’ new pair, reuse old refresh â†’ family revoked |
| 20 | Device Flow | Request device code â†’ approve â†’ poll â†’ token |
| 21 | Token Exchange | Agent A token â†’ exchange â†’ Agent B token with `act` claim |
| 22 | DPoP | Token request with DPoP proof â†’ `token_type=DPoP`, cnf.jkt in token |
| 23 | Introspection | Valid token â†’ `active:true`, revoked â†’ `active:false` |
| 24 | Revocation | Revoke token â†’ introspect returns inactive |
| 25 | DCR | Register client â†’ get credentials â†’ use for auth |
| 26 | Resource Indicators | Token with `resource` â†’ `aud` claim matches |
| 27 | ES256 JWKS | JWKS returns EC key with crv=P-256, token verifiable with it |
| 28 | Consent Management | User lists/revokes consents via self-service API |

Each test must pass for the phase to be marked complete.

- [ ] Steps: Write smoke script (bash, following existing `smoke_test.sh` patterns) â†’ run against fresh dev instance â†’ fix failures â†’ document in SMOKE_TEST.md â†’ commit

---

### Task 22: Unit Test Coverage

**Files:** All `*_test.go` files created in previous tasks

Minimum coverage targets:
- `internal/oauth/store.go` â€” 80%+ (every fosite interface method)
- `internal/oauth/handlers.go` â€” 80%+ (happy path + error paths)
- `internal/oauth/dpop.go` â€” 90%+ (security-critical)
- `internal/oauth/exchange.go` â€” 90%+ (delegation chain correctness)
- `internal/storage/agents_sqlite.go` â€” 80%+ (CRUD + edge cases)
- `internal/storage/oauth_sqlite.go` â€” 80%+ (all entity operations)

- [ ] Steps: Run coverage â†’ identify gaps â†’ add tests â†’ verify targets met â†’ commit

---

### Task 23: Integration Wiring + Server Startup

**Files:**
- Modify: `internal/server/server.go` (or wherever `shark serve` builds the Server)
- Modify: `internal/api/router.go` (final mount of all OAuth routes)

Wire everything together:
1. On startup, generate ES256 signing key if none exists (alongside existing RS256 logic)
2. Create `oauth.Server` with fosite config + SQLite store
3. Mount `/oauth/*` routes
4. Mount `/.well-known/oauth-authorization-server`
5. Mount agent CRUD under `/api/v1/agents`
6. Mount consent management under `/api/v1/auth/consents`
7. Add `WithOAuthServer` option to `api.Server`

- [ ] Steps: Wire startup â†’ mount routes â†’ build binary â†’ run smoke tests â†’ fix issues â†’ commit

---

### Task 24: Update ATTACK.md + Documentation

**Files:**
- Modify: `ATTACK.md`

Update phase order and mark OAuth 2.1 Agent Auth as done (when all smoke tests pass):

```
Phase 5 â€” OAuth 2.1 + Agent Auth â€” Done
Phase 5.5 â€” Token Vault
Phase 6 â€” Proxy + Visual Flow Builder
Phase 7 â€” SDK
```

- [ ] Steps: Update ATTACK.md â†’ commit

---

## Execution Order (Dependencies)

```
Task 1 (migration) â”€â”€â”
Task 2 (deps)     â”€â”€â”€â”¤
Task 3 (ES256)    â”€â”€â”€â”¤â”€â”€ Wave A (parallel-safe, no interdeps)
Task 4 (metadata) â”€â”€â”€â”¤
Task 5 (config)   â”€â”€â”€â”˜
                      â”‚
Task 6 (types)    â”€â”€â”€â”€â”˜ depends on Task 1
                      â”‚
Task 7 (store)    â”€â”€â”€â”€â”˜ depends on Task 6
                      â”‚
Task 8 (server)   â”€â”€â”€â”€â”˜ depends on Task 2, 3, 7
                      â”‚
Task 9 (agents)   â”€â”€â”
Task 10 (consent) â”€â”€â”¤â”€â”€ Wave C (depends on Task 7, 8)
Task 11 (consent API)â”˜
                      â”‚
Task 12 (DCR)     â”€â”€â”
Task 13 (device)  â”€â”€â”¤â”€â”€ Wave D (depends on Task 8)
Task 14 (exchange)â”€â”€â”˜
                      â”‚
Task 15 (DPoP)    â”€â”€â”
Task 16 (intro)   â”€â”€â”¤â”€â”€ Wave E (depends on Task 8)
Task 17 (resource)â”€â”€â”˜
                      â”‚
Task 18-22        â”€â”€â”€â”€â”€â”€ Wave F (depends on all above)
Task 23           â”€â”€â”€â”€â”€â”€ Final wiring (depends on all)
Task 24           â”€â”€â”€â”€â”€â”€ Documentation (last)
```

### Parallelization Opportunities

Within each wave, tasks can be dispatched to parallel subagents:

- **Wave A**: Tasks 1-5 are independent â€” run all 5 in parallel
- **Wave C**: Tasks 9-11 are independent â€” run all 3 in parallel
- **Wave D**: Tasks 12-14 are independent â€” run all 3 in parallel
- **Wave E**: Tasks 15-17 are independent â€” run all 3 in parallel
- **Wave F**: Tasks 18-20 (dashboard) can parallel with 21-22 (tests)

---

## Success Criteria

All of the following must be true before marking Phase 5 complete:

- [ ] `make test` passes (all unit tests)
- [ ] `make verify` passes (vet + unit + integration + e2e)
- [ ] `scripts/smoke_oauth.sh` exits 0 (all 15 new sections pass)
- [ ] ES256 JWKS endpoint returns valid EC keys
- [ ] AS metadata at `/.well-known/oauth-authorization-server` returns all required fields
- [ ] Client credentials grant produces valid JWT with ES256 signature
- [ ] Auth code + PKCE flow works end-to-end (including consent screen)
- [ ] Device flow works end-to-end (including verification page)
- [ ] Token exchange produces JWT with `act` delegation chain
- [ ] DPoP proof validation works (token bound to client key)
- [ ] Refresh token rotation revokes family on reuse
- [ ] DCR endpoint creates functional agents
- [ ] Agent CRUD works in admin dashboard
- [ ] Consent management works (user can view/revoke)
- [ ] All audit events fire for OAuth operations
- [ ] No existing smoke test sections regress
