package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/sharkauth/sharkauth/internal/audit"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// mountExchangeRouter returns an httptest.Server with only the token endpoint.
func mountExchangeServer(t *testing.T) (*httptest.Server, *Server, storage.Store) {
	t.Helper()
	srv, store := newTestOAuthServer(t)
	ts := httptest.NewServer(http.HandlerFunc(srv.HandleToken))
	t.Cleanup(ts.Close)
	return ts, srv, store
}

// seedAgentWithScopes creates a test agent with the given scopes.
func seedAgentWithScopes(t *testing.T, store storage.Store, clientID string, scopes []string) *storage.Agent {
	t.Helper()
	h := sha256.Sum256([]byte("test-secret"))
	agent := &storage.Agent{
		ID:               "agent_" + clientID,
		Name:             "Agent " + clientID,
		Description:      "test",
		ClientID:         clientID,
		ClientSecretHash: hex.EncodeToString(h[:]),
		ClientType:       "confidential",
		AuthMethod:       "client_secret_basic",
		RedirectURIs:     []string{"https://example.com/callback"},
		GrantTypes:       []string{"client_credentials", grantTypeTokenExchange},
		ResponseTypes:    []string{"code"},
		Scopes:           scopes,
		TokenLifetime:    900,
		Active:           true,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if err := store.CreateAgent(context.Background(), agent); err != nil {
		t.Fatalf("seeding agent %s: %v", clientID, err)
	}
	return agent
}

// mintSubjectJWT signs a JWT using the server's Sign method.
// This is the canonical way to create test subject tokens.
func mintSubjectJWT(t *testing.T, srv *Server, sub, scope string, extra map[string]interface{}) string {
	t.Helper()
	now := time.Now().UTC()
	claims := gojwt.MapClaims{
		"iss":   srv.Issuer,
		"sub":   sub,
		"scope": scope,
		"iat":   now.Unix(),
		"exp":   now.Add(15 * time.Minute).Unix(),
		"jti":   "test-jti-" + sub,
	}
	for k, v := range extra {
		claims[k] = v
	}
	tokenStr, err := srv.Sign(claims)
	if err != nil {
		t.Fatalf("minting subject JWT: %v", err)
	}
	return tokenStr
}

// doExchange fires a token-exchange request and returns the decoded response body
// plus the HTTP status code.
func doExchange(t *testing.T, ts *httptest.Server, actorID, actorSecret string, form url.Values) (int, map[string]interface{}) {
	t.Helper()
	form.Set("grant_type", grantTypeTokenExchange)
	req, err := http.NewRequest("POST", ts.URL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(actorID, actorSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("sending request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decoding response JSON: %v\nbody: %s", err, body)
	}
	return resp.StatusCode, result
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestExchange_Success_UserToken: subject = user access token, actor = agent
// -> new token has act: {sub: agent_client_id}
func TestExchange_Success_UserToken(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "actor-agent", []string{"openid", "read"})
	_ = seedUser(t, store, "user@example.com")

	subjectToken := mintSubjectJWT(t, srv, "user@example.com", "openid read", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "actor-agent", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	if body["access_token"] == nil {
		t.Fatal("response missing access_token")
	}
	if body["token_type"] != "Bearer" {
		t.Errorf("expected Bearer, got %v", body["token_type"])
	}
	if body["issued_token_type"] != tokenTypeAccessToken {
		t.Errorf("expected %s, got %v", tokenTypeAccessToken, body["issued_token_type"])
	}

	// Verify the act claim in the issued token.
	tokenStr, _ := body["access_token"].(string)
	parsed, err := srv.parseSubjectJWT(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}
	act, ok := parsed["act"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected act claim to be a map, got %T: %v", parsed["act"], parsed["act"])
	}
	if act["sub"] != "actor-agent" {
		t.Errorf("expected act.sub=actor-agent, got %v", act["sub"])
	}
	if _, hasNestedAct := act["act"]; hasNestedAct {
		t.Error("expected no nested act claim for fresh delegation")
	}
}

// TestExchange_Success_DelegationChain: subject already has act: {sub: agent-a}
// new actor = agent-b -> new token has act: {sub: agent-b, act: {sub: agent-a}}
func TestExchange_Success_DelegationChain(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "agent-a", []string{"openid", "read"})
	seedAgentWithScopes(t, store, "agent-b", []string{"openid", "read"})
	_ = seedUser(t, store, "chainuser@example.com")

	// Subject token already has act: {sub: agent-a} from a prior exchange.
	existingAct := map[string]interface{}{"sub": "agent-a"}
	subjectToken := mintSubjectJWT(t, srv, "chainuser@example.com", "openid read", map[string]interface{}{
		"act": existingAct,
	})

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "agent-b", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	// Verify nested delegation chain.
	tokenStr, _ := body["access_token"].(string)
	parsed, err := srv.parseSubjectJWT(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}

	act, ok := parsed["act"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected act claim, got %T", parsed["act"])
	}
	// Outer act.sub should be agent-b (the new actor).
	if act["sub"] != "agent-b" {
		t.Errorf("expected outer act.sub=agent-b, got %v", act["sub"])
	}
	// Nested act.sub should be agent-a (the prior actor).
	inner, ok := act["act"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested act claim, got %T", act["act"])
	}
	if inner["sub"] != "agent-a" {
		t.Errorf("expected inner act.sub=agent-a, got %v", inner["sub"])
	}
}

// TestExchange_ScopeNarrowing: request a subset of the original scope.
// Should succeed and the issued token should only have the requested subset.
func TestExchange_ScopeNarrowing(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "narrow-actor", []string{"openid", "read", "write"})

	subjectToken := mintSubjectJWT(t, srv, "user-narrow", "openid read write", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		"scope":              {"read"},
	}
	status, body := doExchange(t, ts, "narrow-actor", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	grantedScope, _ := body["scope"].(string)
	scopes := strings.Fields(grantedScope)
	if len(scopes) != 1 || scopes[0] != "read" {
		t.Errorf("expected only scope=read, got %q", grantedScope)
	}

	// Verify the issued token itself contains only the narrowed scope.
	tokenStr, _ := body["access_token"].(string)
	parsed, err := srv.parseSubjectJWT(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}
	tokenScope, _ := parsed["scope"].(string)
	if tokenScope != "read" {
		t.Errorf("expected token scope=read, got %q", tokenScope)
	}
}

// TestExchange_ScopeEscalation_Rejected: request a scope not in subject_token.
// Should return 400 invalid_scope.
func TestExchange_ScopeEscalation_Rejected(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "escalate-actor", []string{"openid", "read", "admin"})

	// Subject token only has "openid read" - requesting "admin" is escalation.
	subjectToken := mintSubjectJWT(t, srv, "user-esc", "openid read", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		"scope":              {"admin"},
	}
	status, body := doExchange(t, ts, "escalate-actor", "test-secret", form)

	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", status, body)
	}
	if body["error"] != "invalid_scope" {
		t.Errorf("expected error=invalid_scope, got %v", body["error"])
	}
}

// TestExchange_InvalidSubjectToken: malformed subject token -> 400 invalid_token.
func TestExchange_InvalidSubjectToken(t *testing.T) {
	ts, _, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "bad-tok-actor", []string{"openid"})

	form := url.Values{
		"subject_token":      {"this.is.not.a.valid.jwt"},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "bad-tok-actor", "test-secret", form)

	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", status, body)
	}
	if body["error"] != "invalid_token" {
		t.Errorf("expected error=invalid_token, got %v", body["error"])
	}
}

// TestExchange_ExpiredSubjectToken: expired subject token -> 400 invalid_token.
func TestExchange_ExpiredSubjectToken(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "expired-actor", []string{"openid"})

	// Build an already-expired token.
	now := time.Now().UTC()
	claims := gojwt.MapClaims{
		"iss":   srv.Issuer,
		"sub":   "user-expired",
		"scope": "openid",
		"iat":   now.Add(-30 * time.Minute).Unix(),
		"exp":   now.Add(-15 * time.Minute).Unix(), // expired
		"jti":   "jti-expired",
	}
	expiredToken, err := srv.Sign(claims)
	if err != nil {
		t.Fatalf("signing expired token: %v", err)
	}

	form := url.Values{
		"subject_token":      {expiredToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "expired-actor", "test-secret", form)

	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", status, body)
	}
	if body["error"] != "invalid_token" {
		t.Errorf("expected error=invalid_token, got %v", body["error"])
	}
}

// TestExchange_MayActEnforcement: subject has may_act=[agent-c], actor=agent-d -> 403.
func TestExchange_MayActEnforcement(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "agent-c", []string{"openid"})
	seedAgentWithScopes(t, store, "agent-d", []string{"openid"})

	// Subject token restricts delegation to agent-c only.
	subjectToken := mintSubjectJWT(t, srv, "restricted-user", "openid", map[string]interface{}{
		"may_act": []interface{}{"agent-c"},
	})

	// agent-d is not in may_act -> should get 403.
	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "agent-d", "test-secret", form)

	if status != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %v", status, body)
	}
	if body["error"] != "access_denied" {
		t.Errorf("expected error=access_denied, got %v", body["error"])
	}
}

// TestExchange_MayActEnforcement_Allowed: subject has may_act=[agent-c], actor=agent-c -> 200.
func TestExchange_MayActEnforcement_Allowed(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "agent-c2", []string{"openid"})

	subjectToken := mintSubjectJWT(t, srv, "allowed-user", "openid", map[string]interface{}{
		"may_act": []interface{}{"agent-c2"},
	})

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "agent-c2", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}
	if body["access_token"] == nil {
		t.Fatal("response missing access_token")
	}
}

// TestExchange_ActingClientAuth: wrong client_secret -> 401.
func TestExchange_ActingClientAuth(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "auth-actor", []string{"openid"})

	subjectToken := mintSubjectJWT(t, srv, "some-user", "openid", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "auth-actor", "WRONG-SECRET", form)

	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %v", status, body)
	}
	if body["error"] != "invalid_client" {
		t.Errorf("expected error=invalid_client, got %v", body["error"])
	}
}

// TestExchange_UnknownActingClient: unknown client_id -> 401.
func TestExchange_UnknownActingClient(t *testing.T) {
	ts, srv, _ := mountExchangeServer(t)

	subjectToken := mintSubjectJWT(t, srv, "some-user", "openid", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
	}
	status, body := doExchange(t, ts, "nonexistent-client", "some-secret", form)

	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %v", status, body)
	}
	if body["error"] != "invalid_client" {
		t.Errorf("expected error=invalid_client, got %v", body["error"])
	}
}

// TestExchange_MissingSubjectToken: omit subject_token -> 400 invalid_request.
func TestExchange_MissingSubjectToken(t *testing.T) {
	ts, _, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "missing-tok-actor", []string{"openid"})

	form := url.Values{
		"subject_token_type": {tokenTypeAccessToken},
		// subject_token intentionally omitted
	}
	status, body := doExchange(t, ts, "missing-tok-actor", "test-secret", form)

	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", status, body)
	}
	if body["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %v", body["error"])
	}
}

// TestExchange_AudiencePropagation: audience parameter -> ends up in issued token.
func TestExchange_AudiencePropagation(t *testing.T) {
	ts, srv, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "aud-actor", []string{"openid", "read"})

	subjectToken := mintSubjectJWT(t, srv, "aud-user", "openid read", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		"audience":           {"https://api.example.com"},
	}
	status, body := doExchange(t, ts, "aud-actor", "test-secret", form)

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d: %v", status, body)
	}

	tokenStr, _ := body["access_token"].(string)
	parsed, err := srv.parseSubjectJWT(context.Background(), tokenStr)
	if err != nil {
		t.Fatalf("parsing issued token: %v", err)
	}
	// The aud claim can be a string or []string depending on how jwt/v5 serializes it.
	aud := parsed["aud"]
	switch v := aud.(type) {
	case string:
		if v != "https://api.example.com" {
			t.Errorf("expected aud=https://api.example.com, got %v", v)
		}
	case []interface{}:
		if len(v) != 1 || v[0] != "https://api.example.com" {
			t.Errorf("expected aud=[https://api.example.com], got %v", v)
		}
	default:
		t.Errorf("unexpected aud type %T: %v", aud, aud)
	}
}

// TestExchange_UnsupportedTokenType: wrong subject_token_type -> 400.
func TestExchange_UnsupportedTokenType(t *testing.T) {
	ts, _, store := mountExchangeServer(t)
	seedAgentWithScopes(t, store, "type-actor", []string{"openid"})

	form := url.Values{
		"subject_token":      {"some.token.value"},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:refresh_token"},
	}
	status, body := doExchange(t, ts, "type-actor", "test-secret", form)

	if status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %v", status, body)
	}
	if body["error"] != "invalid_request" {
		t.Errorf("expected error=invalid_request, got %v", body["error"])
	}
}

// TestScopesSubset_Logic: unit tests for the pure scopesSubset helper.
func TestScopesSubset_Logic(t *testing.T) {
	cases := []struct {
		name      string
		requested []string
		available []string
		want      bool
	}{
		{"empty requested", []string{}, []string{"a", "b"}, true},
		{"equal sets", []string{"a", "b"}, []string{"a", "b"}, true},
		{"proper subset", []string{"a"}, []string{"a", "b"}, true},
		{"escalation", []string{"c"}, []string{"a", "b"}, false},
		{"partial escalation", []string{"a", "c"}, []string{"a", "b"}, false},
		{"empty available", []string{"a"}, []string{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scopesSubset(tc.requested, tc.available); got != tc.want {
				t.Errorf("scopesSubset(%v, %v) = %v, want %v", tc.requested, tc.available, got, tc.want)
			}
		})
	}
}

// TestExchange_AuditMetadata_ScopeFields verifies that a successful token-exchange
// writes an audit row with subject_scope, granted_scope, dropped_scope, and
// requested_scope fields populated, and that empty arrays serialize as [] not null.
func TestExchange_AuditMetadata_ScopeFields(t *testing.T) {
	_, srv, store := mountExchangeServer(t)
	// Wire a real audit logger so the emission path is exercised.
	srv.AuditLogger = audit.NewLogger(store)

	seedAgentWithScopes(t, store, "audit-actor", []string{"openid", "read", "write"})
	_ = seedUser(t, store, "audit-user@example.com")

	// Subject has "openid read write"; request only "read" → should drop "openid" + "write".
	subjectToken := mintSubjectJWT(t, srv, "audit-user@example.com", "openid read write", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		"scope":              {"read"},
	}
	form.Set("grant_type", grantTypeTokenExchange)
	req, err := http.NewRequest("POST", "http://localhost", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("audit-actor", "test-secret")
	rr := httptest.NewRecorder()
	srv.HandleToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Query the audit log for the emitted row.
	ctx := context.Background()
	logs, err := store.QueryAuditLogs(ctx, storage.AuditLogQuery{
		Action: "oauth.token.exchanged",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("querying audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected at least one audit log entry for oauth.token.exchanged")
	}

	row := logs[0]
	var meta map[string]json.RawMessage
	if err := json.Unmarshal([]byte(row.Metadata), &meta); err != nil {
		t.Fatalf("parsing audit metadata JSON: %v\nraw: %s", err, row.Metadata)
	}

	for _, field := range []string{"subject_scope", "granted_scope", "dropped_scope", "requested_scope"} {
		raw, ok := meta[field]
		if !ok {
			t.Errorf("audit metadata missing field %q; metadata: %s", field, row.Metadata)
			continue
		}
		// Must be a JSON array, never null.
		var arr []string
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Errorf("field %q is not a JSON array: %s (error: %v)", field, raw, err)
		}
	}

	// Specific values: subject had "openid read write", requested "read".
	checkArray := func(field string, want []string) {
		raw, ok := meta[field]
		if !ok {
			return // already reported above
		}
		var got []string
		_ = json.Unmarshal(raw, &got)
		if len(got) != len(want) {
			t.Errorf("field %q: want %v, got %v", field, want, got)
			return
		}
		wantSet := map[string]bool{}
		for _, s := range want {
			wantSet[s] = true
		}
		for _, s := range got {
			if !wantSet[s] {
				t.Errorf("field %q: unexpected value %q; got %v", field, s, got)
			}
		}
	}
	checkArray("granted_scope", []string{"read"})
	checkArray("requested_scope", []string{"read"})
	// dropped = subject - granted = openid + write
	checkArray("dropped_scope", []string{"openid", "write"})
}

// TestExchange_AuditMetadata_EmptyArrays verifies that when no scope is requested
// (full pass-through) dropped_scope and requested_scope serialize as [] not null.
func TestExchange_AuditMetadata_EmptyArrays(t *testing.T) {
	_, srv, store := mountExchangeServer(t)
	srv.AuditLogger = audit.NewLogger(store)

	seedAgentWithScopes(t, store, "empty-arr-actor", []string{"openid", "read"})
	_ = seedUser(t, store, "empty-arr-user@example.com")

	// No "scope" param → full pass-through; dropped_scope and requested_scope should be [].
	subjectToken := mintSubjectJWT(t, srv, "empty-arr-user@example.com", "openid read", nil)

	form := url.Values{
		"subject_token":      {subjectToken},
		"subject_token_type": {tokenTypeAccessToken},
		// scope intentionally omitted
	}
	form.Set("grant_type", grantTypeTokenExchange)
	req, err := http.NewRequest("POST", "http://localhost", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("empty-arr-actor", "test-secret")
	rr := httptest.NewRecorder()
	srv.HandleToken(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	ctx := context.Background()
	logs, err := store.QueryAuditLogs(ctx, storage.AuditLogQuery{
		Action: "oauth.token.exchanged",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("querying audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected audit log entry")
	}

	var meta map[string]json.RawMessage
	_ = json.Unmarshal([]byte(logs[0].Metadata), &meta)

	for _, field := range []string{"dropped_scope", "requested_scope"} {
		raw, ok := meta[field]
		if !ok {
			t.Errorf("missing field %q", field)
			continue
		}
		// Must decode as [] not null.
		if string(raw) == "null" {
			t.Errorf("field %q serialized as null, want []", field)
		}
		var arr []string
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Errorf("field %q not valid JSON array: %v", field, err)
		}
		if len(arr) != 0 {
			t.Errorf("field %q: want empty array, got %v", field, arr)
		}
	}
}

// TestBuildActClaim_Logic: unit tests for the buildActClaim helper.
func TestBuildActClaim_Logic(t *testing.T) {
	// No prior act.
	act := buildActClaim("agent-x", nil)
	if act["sub"] != "agent-x" {
		t.Errorf("expected sub=agent-x, got %v", act["sub"])
	}
	if _, ok := act["act"]; ok {
		t.Error("expected no nested act when subjectAct is nil")
	}

	// With prior act.
	prior := map[string]interface{}{"sub": "agent-y"}
	act2 := buildActClaim("agent-z", prior)
	if act2["sub"] != "agent-z" {
		t.Errorf("expected sub=agent-z, got %v", act2["sub"])
	}
	nested, ok := act2["act"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested act claim, got %T", act2["act"])
	}
	if nested["sub"] != "agent-y" {
		t.Errorf("expected nested sub=agent-y, got %v", nested["sub"])
	}
}
