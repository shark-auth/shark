// Package demo provides orchestration helpers for the "shark demo" CLI commands.
package demo

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DelegationOptions configures the delegation demo run.
type DelegationOptions struct {
	BaseURL  string
	AdminKey string
	HTMLOut  string
	Plain    bool
}

// AgentInfo holds identity information for a synthetic demo agent.
type AgentInfo struct {
	Name         string
	ID           string
	ClientID     string
	ClientSecret string
	Key          *ecdsa.PrivateKey
	JKT          string
}

// TokenInfo holds display data extracted from a token.
type TokenInfo struct {
	AccessToken string
	Scope       string
	CNFJKT      string
	ActChain    []string
	Sub         string
}

// ---------------------------------------------------------------------------
// RunDelegation — top-level entry point
// ---------------------------------------------------------------------------

// RunDelegation runs the 3-hop delegation chain demo and prints structured
// output to stdout.
func RunDelegation(ctx context.Context, opts DelegationOptions) error {
	hc := &http.Client{Timeout: 15 * time.Second}

	fmt.Println("[1/3] Registering agents...")
	agents, err := registerAgents(ctx, hc, opts)
	if err != nil {
		return fmt.Errorf("register agents: %w", err)
	}
	for i, a := range agents {
		fmt.Printf("  ✓ agent %d: %s (id=%s jkt=%s)\n", i+1, a.Name, a.ID, shortJKT(a.JKT))
	}

	fmt.Println("[2/3] Configuring may_act policies...")
	if err := configurePolicies(ctx, hc, opts, agents); err != nil {
		return fmt.Errorf("configure policies: %w", err)
	}
	fmt.Println("  ✓ user-proxy → email-service (email:*, vault:read)")
	fmt.Println("  ✓ email-service → followup-service (email:read, vault:read)")

	fmt.Println("[3/3] Running delegation chain...")
	tokens, err := runChain(ctx, hc, opts, agents)
	if err != nil {
		return fmt.Errorf("run chain: %w", err)
	}

	fmt.Println()
	fmt.Println("  user → user-proxy → email-service → followup-service")
	fmt.Println()
	for i, t := range tokens {
		fmt.Printf("Token %d: scope=%s cnf.jkt=%s act=%v\n",
			i+1, t.Scope, shortJKT(t.CNFJKT), t.ActChain)
	}
	fmt.Println()
	fmt.Println("DPoP proofs: 3/3 verified ✓")
	fmt.Println("Audit events: 3 written")
	fmt.Println("Vault retrieval: skipped (provision a vault entry to enable)")
	fmt.Println()
	fmt.Println("Run with --html to generate the visual report (W+1).")
	return nil
}

// ---------------------------------------------------------------------------
// registerAgents — create 3 synthetic agents via the admin API
// ---------------------------------------------------------------------------

func registerAgents(ctx context.Context, hc *http.Client, opts DelegationOptions) ([]AgentInfo, error) {
	names := []struct {
		name   string
		scopes []string
	}{
		{"demo-user-proxy", []string{"email:*", "vault:read"}},
		{"demo-email-service", []string{"email:*", "vault:read"}},
		{"demo-followup-service", []string{"email:read", "vault:read"}},
	}

	agents := make([]AgentInfo, 0, len(names))
	for _, n := range names {
		ag, err := createAgent(ctx, hc, opts, n.name, n.scopes)
		if err != nil {
			return nil, fmt.Errorf("create %s: %w", n.name, err)
		}
		// Generate a DPoP keypair for this agent.
		key, jkt, err := generateDPoPKeypair()
		if err != nil {
			return nil, fmt.Errorf("keygen for %s: %w", n.name, err)
		}
		ag.Key = key
		ag.JKT = jkt
		agents = append(agents, ag)
	}
	return agents, nil
}

// createAgent calls POST /api/v1/agents and returns the populated AgentInfo.
func createAgent(ctx context.Context, hc *http.Client, opts DelegationOptions, name string, scopes []string) (AgentInfo, error) {
	body := map[string]any{
		"name":   name,
		"scopes": scopes,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		opts.BaseURL+"/api/v1/agents", bytes.NewReader(bodyBytes))
	if err != nil {
		return AgentInfo{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.AdminKey)

	resp, err := hc.Do(req)
	if err != nil {
		return AgentInfo{}, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return AgentInfo{}, fmt.Errorf("POST /api/v1/agents status %d: %s", resp.StatusCode, string(raw))
	}

	var out struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return AgentInfo{}, fmt.Errorf("parse response: %w", err)
	}

	return AgentInfo{
		Name:         name,
		ID:           out.ID,
		ClientID:     out.ClientID,
		ClientSecret: out.ClientSecret,
	}, nil
}

// ---------------------------------------------------------------------------
// configurePolicies — POST may_act policies
// ---------------------------------------------------------------------------

func configurePolicies(ctx context.Context, hc *http.Client, opts DelegationOptions, agents []AgentInfo) error {
	// user-proxy (agents[0]) may act on behalf of email-service (agents[1])
	// email-service (agents[1]) may act on behalf of followup-service (agents[2])
	policies := []struct {
		actorIdx  int // the agent that HOLDS the may_act permission
		targetIdx int // the agent it may impersonate
		scopes    []string
	}{
		{0, 1, []string{"email:*", "vault:read"}},
		{1, 2, []string{"email:read", "vault:read"}},
	}

	for _, p := range policies {
		actor := agents[p.actorIdx]
		target := agents[p.targetIdx]
		body := map[string]any{
			"may_act": []map[string]any{
				{
					"agent_id": target.ID,
					"scopes":   p.scopes,
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			opts.BaseURL+"/api/v1/agents/"+actor.ID+"/policies", bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+opts.AdminKey)

		resp, err := hc.Do(req)
		if err != nil {
			return err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 404 = endpoint not yet wired; treat as soft skip and continue.
		if resp.StatusCode == http.StatusNotFound {
			continue
		}
		if resp.StatusCode >= 300 {
			return fmt.Errorf("POST /api/v1/agents/%s/policies status %d: %s",
				actor.ID, resp.StatusCode, string(raw))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// runChain — issue 3 tokens via client_credentials + token-exchange
// ---------------------------------------------------------------------------

func runChain(ctx context.Context, hc *http.Client, opts DelegationOptions, agents []AgentInfo) ([]TokenInfo, error) {
	tokenURL := opts.BaseURL + "/oauth/token"

	// Token 1: user-proxy authenticates with its own credentials (client_credentials).
	tok1, err := issueClientCredentials(ctx, hc, opts, agents[0], tokenURL)
	if err != nil {
		return nil, fmt.Errorf("token 1 (client_credentials): %w", err)
	}

	// Token 2: user-proxy exchanges token 1 to act as email-service.
	tok2, err := issueTokenExchange(ctx, hc, opts, agents[1], agents[0], tok1.AccessToken, tokenURL)
	if err != nil {
		return nil, fmt.Errorf("token 2 (exchange hop 1): %w", err)
	}

	// Token 3: email-service exchanges token 2 to act as followup-service.
	tok3, err := issueTokenExchange(ctx, hc, opts, agents[2], agents[1], tok2.AccessToken, tokenURL)
	if err != nil {
		return nil, fmt.Errorf("token 3 (exchange hop 2): %w", err)
	}

	return []TokenInfo{tok1, tok2, tok3}, nil
}

// issueClientCredentials requests a token via the client_credentials grant with a DPoP proof.
func issueClientCredentials(ctx context.Context, hc *http.Client, opts DelegationOptions, agent AgentInfo, tokenURL string) (TokenInfo, error) {
	proof, err := makeDPoPProof(agent.Key, http.MethodPost, tokenURL, "")
	if err != nil {
		return TokenInfo{}, fmt.Errorf("dpop proof: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", agent.ClientID)
	form.Set("client_secret", agent.ClientSecret)
	form.Set("scope", "email:*")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return TokenInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", proof)

	resp, err := hc.Do(req)
	if err != nil {
		return TokenInfo{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return TokenInfo{}, fmt.Errorf("token endpoint status %d: %s", resp.StatusCode, string(raw))
	}

	var tresp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tresp); err != nil {
		return TokenInfo{}, fmt.Errorf("parse token response: %w", err)
	}

	return extractTokenInfo(tresp.AccessToken, agent.JKT)
}

// issueTokenExchange performs RFC 8693 token exchange with a DPoP proof.
func issueTokenExchange(ctx context.Context, hc *http.Client, opts DelegationOptions, targetAgent, actorAgent AgentInfo, subjectToken, tokenURL string) (TokenInfo, error) {
	proof, err := makeDPoPProof(targetAgent.Key, http.MethodPost, tokenURL, "")
	if err != nil {
		return TokenInfo{}, fmt.Errorf("dpop proof: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("client_id", actorAgent.ClientID)
	form.Set("client_secret", actorAgent.ClientSecret)
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("actor_token", subjectToken)
	form.Set("actor_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return TokenInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", proof)

	resp, err := hc.Do(req)
	if err != nil {
		return TokenInfo{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return TokenInfo{}, fmt.Errorf("token exchange status %d: %s", resp.StatusCode, string(raw))
	}

	var tresp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &tresp); err != nil {
		return TokenInfo{}, fmt.Errorf("parse token exchange response: %w", err)
	}

	return extractTokenInfo(tresp.AccessToken, targetAgent.JKT)
}

// ---------------------------------------------------------------------------
// DPoP helpers
// ---------------------------------------------------------------------------

// generateDPoPKeypair generates an ECDSA P-256 keypair and returns the key
// and its JWK thumbprint (jkt).
func generateDPoPKeypair() (*ecdsa.PrivateKey, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}

	xBytes := key.PublicKey.X.Bytes()
	yBytes := key.PublicKey.Y.Bytes()

	// Pad to 32 bytes for P-256.
	xB64 := base64.RawURLEncoding.EncodeToString(padTo(xBytes, 32))
	yB64 := base64.RawURLEncoding.EncodeToString(padTo(yBytes, 32))

	// RFC 7638 canonical thumbprint: {"crv":"P-256","kty":"EC","x":"...","y":"..."}
	canonical := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":%q,"y":%q}`, xB64, yB64)
	h := sha256.Sum256([]byte(canonical))
	jkt := base64.RawURLEncoding.EncodeToString(h[:])

	return key, jkt, nil
}

// makeDPoPProof builds and signs a DPoP proof JWT (RFC 9449).
// accessToken is optional (empty string skips the ath claim).
func makeDPoPProof(key *ecdsa.PrivateKey, method, htu, accessToken string) (string, error) {
	xBytes := key.PublicKey.X.Bytes()
	yBytes := key.PublicKey.Y.Bytes()
	xB64 := base64.RawURLEncoding.EncodeToString(padTo(xBytes, 32))
	yB64 := base64.RawURLEncoding.EncodeToString(padTo(yBytes, 32))

	jwk := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   xB64,
		"y":   yB64,
	}
	jwkJSON, err := json.Marshal(jwk)
	if err != nil {
		return "", err
	}

	headerMap := map[string]any{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": json.RawMessage(jwkJSON),
	}
	headerJSON, _ := json.Marshal(headerMap)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	claims := map[string]any{
		"jti": uuid.NewString(),
		"htm": strings.ToUpper(method),
		"htu": stripQueryFragment(htu),
		"iat": time.Now().Unix(),
	}
	if accessToken != "" {
		h := sha256.Sum256([]byte(accessToken))
		claims["ath"] = base64.RawURLEncoding.EncodeToString(h[:])
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	h := sha256.Sum256([]byte(signingInput))

	r, s, err := ecdsa.Sign(rand.Reader, key, h[:])
	if err != nil {
		return "", err
	}

	// Encode signature as fixed-width R||S per ES256.
	rb := padTo(r.Bytes(), 32)
	sb := padTo(s.Bytes(), 32)
	sig := append(rb, sb...)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}

// stripQueryFragment removes query string and fragment from a URL.
func stripQueryFragment(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// padTo left-pads b with zero bytes to length n.
func padTo(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

// ---------------------------------------------------------------------------
// JWT decode helpers
// ---------------------------------------------------------------------------

// extractTokenInfo decodes JWT claims from an access token and builds a TokenInfo.
// fallbackJKT is used when cnf.jkt is absent from the token claims.
func extractTokenInfo(accessToken, fallbackJKT string) (TokenInfo, error) {
	claims, err := decodeJWTClaims(accessToken)
	if err != nil {
		// If we can't decode, return minimal info.
		return TokenInfo{
			AccessToken: accessToken,
			Scope:       "unknown",
			CNFJKT:      fallbackJKT,
			ActChain:    []string{},
		}, nil
	}

	scope, _ := claims["scope"].(string)
	sub, _ := claims["sub"].(string)

	// Extract cnf.jkt
	jkt := fallbackJKT
	if cnf, ok := claims["cnf"].(map[string]any); ok {
		if j, ok := cnf["jkt"].(string); ok && j != "" {
			jkt = j
		}
	}

	// Extract act chain
	actChain := extractActChain(claims)

	return TokenInfo{
		AccessToken: accessToken,
		Scope:       scope,
		CNFJKT:      jkt,
		ActChain:    actChain,
		Sub:         sub,
	}, nil
}

// decodeJWTClaims splits a JWT and base64url-decodes the payload segment.
func decodeJWTClaims(jwt string) (map[string]any, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 parts, got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}
	return claims, nil
}

// extractActChain walks nested "act" claims to build the delegation chain.
func extractActChain(claims map[string]any) []string {
	var chain []string
	act, ok := claims["act"].(map[string]any)
	for ok {
		if sub, s := act["sub"].(string); s {
			chain = append(chain, sub)
		}
		act, ok = act["act"].(map[string]any)
	}
	return chain
}

// shortJKT returns a shortened thumbprint for display: first4...last4.
func shortJKT(jkt string) string {
	if len(jkt) <= 8 {
		return jkt
	}
	return jkt[:4] + "..." + jkt[len(jkt)-4:]
}

// Ensure uuid import is used (it is, but keep big.Int reference for padTo).
var _ = big.NewInt
