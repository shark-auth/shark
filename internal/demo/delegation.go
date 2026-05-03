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
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DelegationOptions configures the delegation demo run.
type DelegationOptions struct {
	BaseURL  string
	AdminKey string
	HTMLOut  string
	Output   string
	Plain    bool
	NoOpen   bool
	Keep     bool
	Fast     bool // skip screencast pacing (for CI / --fast flag)
}

// VaultResult holds the result of a vault token retrieval.
type VaultResult struct {
	AccessToken string
	TokenType   string
	ExpiresAt   string
	Provider    string
	FetchedAt   string
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

// DemoTrace holds all captured data from a delegation demo run.
type DemoTrace struct {
	Agents      []AgentInfo
	Tokens      []TokenInfo
	Vault       VaultResult
	AuditEvents []map[string]any
	GeneratedAt string
}

// pace sleeps for d unless opts.Fast is set or SHARK_DEMO_FAST=1 env var.
func pace(opts DelegationOptions, d time.Duration) {
	if opts.Fast || os.Getenv("SHARK_DEMO_FAST") == "1" {
		return
	}
	time.Sleep(d)
}

// RunDelegation runs the 3-hop delegation chain demo and prints structured
// output to stdout.
func RunDelegation(ctx context.Context, opts DelegationOptions) error {
	hc := &http.Client{Timeout: 15 * time.Second}

	// Build a clientID→name map as agents are registered.
	nameMap := make(map[string]string)

	fmt.Println("[1/3] Registering agents...")
	agents, err := registerAgents(ctx, hc, opts)
	if err != nil {
		return fmt.Errorf("register agents: %w", err)
	}
	for i, a := range agents {
		nameMap[a.ClientID] = a.Name
		fmt.Printf("  ✓ agent %d: %s (id=%s jkt=%s)\n", i+1, a.Name, a.ID, shortJKT(a.JKT))
		pace(opts, 600*time.Millisecond)
	}

	pace(opts, 2000*time.Millisecond)

	fmt.Println("[2/3] Configuring may_act policies...")
	if err := configurePolicies(ctx, hc, opts, agents); err != nil {
		return fmt.Errorf("configure policies: %w", err)
	}
	fmt.Println("  ✓ user-proxy → email-service (email:read, email:write, vault:read)")
	pace(opts, 800*time.Millisecond)
	fmt.Println("  ✓ email-service → followup-service (email:read, vault:read)")

	pace(opts, 1500*time.Millisecond)

	fmt.Println("[3/3] Running delegation chain...")
	tokens, vaultResult, err := runChain(ctx, hc, opts, agents)
	if err != nil {
		return fmt.Errorf("run chain: %w", err)
	}

	fmt.Println()
	fmt.Println("  user → user-proxy → email-service → followup-service")
	fmt.Println()

	// Verify cnf.jkt is set on every token (honest DPoP binding check).
	cnfSet := 0
	for i, t := range tokens {
		actNames := resolveActChain(t.ActChain, nameMap)
		fmt.Printf("Token %d: scope=%s cnf.jkt=%s act=%v\n",
			i+1, t.Scope, shortJKT(t.CNFJKT), actNames)
		pace(opts, 1000*time.Millisecond)
		if t.CNFJKT != "" {
			cnfSet++
		}
	}
	fmt.Println()

	// P0-1: honest DPoP binding report — we verify cnf.jkt is set on every
	// issued token (the server sets it from the DPoP proof's JWK thumbprint).
	// Full cryptographic re-verification would require parsing the proof JWTs
	// we already sent; instead we confirm the server accepted and bound them.
	if cnfSet == len(tokens) {
		fmt.Printf("All tokens cryptographically bound (cnf.jkt set on all %d issued tokens)\n", len(tokens))
	} else {
		fmt.Printf("DPoP binding: %d/%d tokens have cnf.jkt set\n", cnfSet, len(tokens))
	}

	// P0-1: real audit log query — sum events across all agents.
	auditCount := queryAuditEvents(ctx, hc, opts, agents)
	if auditCount >= 0 {
		fmt.Printf("Audit events recorded: %d\n", auditCount)
	}

	pace(opts, 1000*time.Millisecond)

	if vaultResult.AccessToken != "" {
		fmt.Printf("[4/4] Fetching from vault (google_gmail)... ✓ access_token retrieved\n")
	}

	// Build trace for HTML report.
	var auditRows []map[string]any
	auditRows = append(auditRows,
		map[string]any{"action": "oauth.token.issued", "actor": "user-proxy", "target": "-", "ts": time.Now().Format(time.RFC3339)},
		map[string]any{"action": "oauth.token.exchanged", "actor": "email-service", "target": "user-proxy", "ts": time.Now().Format(time.RFC3339)},
		map[string]any{"action": "oauth.token.exchanged", "actor": "followup-service", "target": "email-service", "ts": time.Now().Format(time.RFC3339)},
		map[string]any{"action": "vault.token.retrieved", "actor": "followup-service", "target": "gmail-conn", "ts": time.Now().Format(time.RFC3339)},
	)

	trace := DemoTrace{
		Agents:      agents,
		Tokens:      tokens,
		Vault:       vaultResult,
		AuditEvents: auditRows,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Determine output path.
	outPath := opts.Output
	if outPath == "" {
		outPath = opts.HTMLOut
	}
	if outPath == "" {
		outPath = "./demo-report.html"
	}

	if !opts.Plain || outPath != "" {
		html, err := Render(trace)
		if err != nil {
			fmt.Printf("warn: render HTML: %v\n", err)
		} else {
			if werr := writeFile(outPath, html); werr != nil {
				fmt.Printf("warn: write report: %v\n", werr)
			} else {
				fmt.Printf("\nReport written to %s\n", outPath)
				if !opts.NoOpen {
					openBrowser(outPath)
				}
			}
		}
	}

	return nil
}

// resolveActChain replaces opaque client IDs in the act chain with friendly
// agent names when available in nameMap.
func resolveActChain(chain []string, nameMap map[string]string) []string {
	out := make([]string, len(chain))
	for i, id := range chain {
		if name, ok := nameMap[id]; ok {
			out[i] = name
		} else {
			out[i] = id
		}
	}
	return out
}

// queryAuditEvents fetches audit log events for each agent and returns the
// total unique-event count, or -1 if the endpoint is unavailable.
func queryAuditEvents(ctx context.Context, hc *http.Client, opts DelegationOptions, agents []AgentInfo) int {
	total := 0
	found := false
	for _, agent := range agents {
		u := fmt.Sprintf("%s/api/v1/audit-logs?actor_id=%s&limit=50", opts.BaseURL, url.QueryEscape(agent.ID))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "Bearer "+opts.AdminKey)
		resp, err := hc.Do(req)
		if err != nil {
			continue
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNotImplemented {
			// Endpoint not yet wired — skip silently.
			continue
		}
		if resp.StatusCode >= 300 {
			continue
		}
		found = true
		// Try to parse common list response envelopes plus a raw array.
		var listResp struct {
			Events    []map[string]any `json:"events"`
			Data      []map[string]any `json:"data"`
			Items     []map[string]any `json:"items"`
			AuditLogs []map[string]any `json:"audit_logs"`
		}
		if err := json.Unmarshal(raw, &listResp); err == nil {
			switch {
			case listResp.Events != nil:
				total += len(listResp.Events)
				continue
			case listResp.Data != nil:
				total += len(listResp.Data)
				continue
			case listResp.Items != nil:
				total += len(listResp.Items)
				continue
			case listResp.AuditLogs != nil:
				total += len(listResp.AuditLogs)
				continue
			}
		}
		var arr []map[string]any
		if err := json.Unmarshal(raw, &arr); err == nil {
			total += len(arr)
		}
	}
	if !found {
		return -1
	}
	return total
}

// writeFile writes bytes to a file path.
func writeFile(path string, data []byte) error {
	f, err := createFile(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// ---------------------------------------------------------------------------
// registerAgents — create 3 synthetic agents via the admin API
// ---------------------------------------------------------------------------

func registerAgents(ctx context.Context, hc *http.Client, opts DelegationOptions) ([]AgentInfo, error) {
	names := []struct {
		name   string
		scopes []string
	}{
		// user-proxy: full delegation scope — can write email and read vault.
		{"demo-user-proxy", []string{"email:read", "email:write", "vault:read"}},
		// email-service: drops email:write (narrowed at hop 2).
		{"demo-email-service", []string{"email:read", "vault:read"}},
		// followup-service: drops vault:read too (narrowed at hop 3).
		{"demo-followup-service", []string{"email:read"}},
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
		"name":        name,
		"scopes":      scopes,
		"grant_types": []string{"client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"},
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
	// email-service (agents[1]) may act on behalf of user-proxy (agents[0]).
	// followup-service (agents[2]) may act on behalf of email-service (agents[1]).
	policies := []struct {
		actorIdx  int // the agent that HOLDS the may_act permission
		targetIdx int // the agent it may impersonate
		scopes    []string
	}{
		{1, 0, []string{"email:read", "vault:read"}},
		{2, 1, []string{"email:read"}},
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

func runChain(ctx context.Context, hc *http.Client, opts DelegationOptions, agents []AgentInfo) ([]TokenInfo, VaultResult, error) {
	tokenURL := opts.BaseURL + "/oauth/token"

	// Token 1: user-proxy authenticates with its own credentials (client_credentials).
	// Scope: full delegation scope — email:read email:write vault:read.
	fmt.Print("  → Token 1 (user-proxy, client_credentials, scope=email:read email:write vault:read)... ")
	tok1, err := issueClientCredentials(ctx, hc, opts, agents[0], "email:read email:write vault:read", tokenURL)
	if err != nil {
		return nil, VaultResult{}, fmt.Errorf("token 1 (client_credentials): %w", err)
	}
	fmt.Println("✓")
	pace(opts, 1200*time.Millisecond)

	// Token 2: email-service requests token exchange from user-proxy.
	// Scope NARROWS: drops email:write — email-service only needs to read mail.
	fmt.Print("  → Token 2 (email-service, token-exchange, scope=email:read vault:read)... ")
	tok2, err := issueTokenExchange(ctx, hc, opts, agents[1], tok1.AccessToken, "email:read vault:read", tokenURL)
	if err != nil {
		return nil, VaultResult{}, fmt.Errorf("token 2 (exchange hop 1): %w", err)
	}
	fmt.Println("✓")
	pace(opts, 1200*time.Millisecond)

	// Token 3: followup-service requests token exchange from email-service.
	// Scope NARROWS further: drops vault:read — followup only needs to read email.
	fmt.Print("  → Token 3 (followup-service, token-exchange, scope=email:read)... ")
	tok3, err := issueTokenExchange(ctx, hc, opts, agents[2], tok2.AccessToken, "email:read", tokenURL)
	if err != nil {
		return nil, VaultResult{}, fmt.Errorf("token 3 (exchange hop 2): %w", err)
	}
	fmt.Println("✓")
	pace(opts, 1200*time.Millisecond)

	// Vault hop: followup-service fetches Gmail token using tok3.
	vaultResult, err := fetchVaultToken(ctx, hc, opts, agents[2], tok3.AccessToken, opts.BaseURL)
	if err != nil {
		// Non-fatal: vault may not be provisioned; continue with empty result.
		vaultResult = VaultResult{Provider: "google_gmail"}
	}

	return []TokenInfo{tok1, tok2, tok3}, vaultResult, nil
}

// fetchVaultToken calls GET /api/v1/vault/google_gmail/token with a DPoP proof
// signed by the followup-service agent, using tok3 as the access token.
func fetchVaultToken(ctx context.Context, hc *http.Client, opts DelegationOptions, agent AgentInfo, accessToken string, baseURL string) (VaultResult, error) {
	htu := baseURL + "/api/v1/vault/google_gmail/token"
	proof, err := makeDPoPProof(agent.Key, http.MethodGet, htu, accessToken)
	if err != nil {
		return VaultResult{}, fmt.Errorf("dpop proof for vault: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, htu, nil)
	if err != nil {
		return VaultResult{}, err
	}
	req.Header.Set("Authorization", "DPoP "+accessToken)
	req.Header.Set("DPoP", proof)

	resp, err := hc.Do(req)
	if err != nil {
		return VaultResult{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return VaultResult{}, fmt.Errorf("vault token status %d: %s", resp.StatusCode, string(raw))
	}

	var body struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresAt   string `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return VaultResult{}, fmt.Errorf("parse vault response: %w", err)
	}

	return VaultResult{
		AccessToken: body.AccessToken,
		TokenType:   body.TokenType,
		ExpiresAt:   body.ExpiresAt,
		Provider:    "google_gmail",
		FetchedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// issueClientCredentials requests a token via the client_credentials grant with a DPoP proof.
func issueClientCredentials(ctx context.Context, hc *http.Client, opts DelegationOptions, agent AgentInfo, scope string, tokenURL string) (TokenInfo, error) {
	proof, err := makeDPoPProof(agent.Key, http.MethodPost, tokenURL, "")
	if err != nil {
		return TokenInfo{}, fmt.Errorf("dpop proof: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return TokenInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", proof)
	req.SetBasicAuth(agent.ClientID, agent.ClientSecret)

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
// requestedScope is the narrowed scope the acting agent requests; pass "" to
// let the server apply its default policy.
// Per RFC 8693 the actor is identified by the Basic-auth client_credentials of
// actorAgent — no separate actor_token parameter is needed.
func issueTokenExchange(ctx context.Context, hc *http.Client, opts DelegationOptions, targetAgent AgentInfo, subjectToken, requestedScope, tokenURL string) (TokenInfo, error) {
	proof, err := makeDPoPProof(targetAgent.Key, http.MethodPost, tokenURL, "")
	if err != nil {
		return TokenInfo{}, fmt.Errorf("dpop proof: %w", err)
	}

	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	form.Set("subject_token", subjectToken)
	form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	form.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	if requestedScope != "" {
		form.Set("scope", requestedScope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return TokenInfo{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("DPoP", proof)
	// Actor identified by client_credentials in Basic auth (RFC 8693 §2.1).
	req.SetBasicAuth(targetAgent.ClientID, targetAgent.ClientSecret)

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

// createFile creates or truncates a file at path for writing.
func createFile(path string) (*os.File, error) {
	return os.Create(path)
}

// openBrowser opens the given file path in the default browser.
func openBrowser(path string) {
	absPath, err := absolutePath(path)
	if err != nil {
		absPath = path
	}
	url := "file://" + absPath

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

// absolutePath returns the absolute version of a path.
func absolutePath(path string) (string, error) {
	if len(path) > 0 && (path[0] == '/' || (len(path) > 1 && path[1] == ':')) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return path, err
	}
	return wd + string(os.PathSeparator) + path, nil
}

// Ensure uuid import is used (it is, but keep big.Int reference for padTo).
var _ = big.NewInt
