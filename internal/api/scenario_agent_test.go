package api_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

// TestScenario_AutonomousArchivist demonstrates the "Autonomous Archivist" use case.
// An agent registers itself, gets a token via client_credentials, and accesses a protected resource.
func TestScenario_AutonomousArchivist(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 1. Agent registers via DCR
	dcrPayload := map[string]interface{}{
		"client_name": "Autonomous Archivist",
		"grant_types": []string{"client_credentials"},
		"scope":       "articles:read storage:write",
	}
	resp := ts.PostJSON("/oauth/register", dcrPayload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("DCR failed: %d", resp.StatusCode)
	}
	var dcrResp struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	ts.DecodeJSON(resp, &dcrResp)

	// 2. Agent gets access token via client_credentials
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "articles:read storage:write")

	req, _ := http.NewRequest("POST", ts.URL("/oauth/token"), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(dcrResp.ClientID, dcrResp.ClientSecret)

	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("Do request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Token request failed: %d", resp.StatusCode)
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	ts.DecodeJSON(resp, &tokenResp)

	if tokenResp.AccessToken == "" {
		t.Fatal("expected access_token")
	}
}

// TestScenario_SwarmOrchestrator demonstrates the "Swarm Orchestrator" use case.
// One agent (Manager) exchanges its token for a new one acting on behalf of itself,
// delegating to a Worker agent.
func TestScenario_SwarmOrchestrator(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 1. Setup Manager Agent
	managerResp := ts.PostJSONWithAdminKey("/api/v1/agents", map[string]interface{}{
		"name":        "Manager Agent",
		"grant_types": []string{"client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"},
		"scopes":      []string{"task:orchestrate", "worker:delegate"},
	})
	var manager struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	ts.DecodeJSON(managerResp, &manager)

	// 2. Setup Worker Agent
	workerResp := ts.PostJSONWithAdminKey("/api/v1/agents", map[string]interface{}{
		"name":        "Worker Agent",
		"grant_types": []string{"client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange"},
		"scopes":      []string{"task:execute"},
	})
	var worker struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	ts.DecodeJSON(workerResp, &worker)

	// 3. Manager gets its own token
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "task:orchestrate worker:delegate")
	req, _ := http.NewRequest("POST", ts.URL("/oauth/token"), strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(manager.ClientID, manager.ClientSecret)
	resp, err := ts.Client.Do(req)
	if err != nil {
		t.Fatalf("Do request failed: %v", err)
	}
	var managerToken struct {
		AccessToken string `json:"access_token"`
	}
	ts.DecodeJSON(resp, &managerToken)

	// 4. Worker exchanges Manager's token to act on its behalf
	// (In a real swarm, the Manager might pass its token to the Worker, or the Worker might be the one calling)
	// RFC 8693: The acting party (Worker) authenticates and presents the subject_token (Manager's token).
	exchangeData := url.Values{}
	exchangeData.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	exchangeData.Set("subject_token", managerToken.AccessToken)
	exchangeData.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
	exchangeData.Set("scope", "task:execute")

	req, _ = http.NewRequest("POST", ts.URL("/oauth/token"), strings.NewReader(exchangeData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(worker.ClientID, worker.ClientSecret)

	resp, err = ts.Client.Do(req)
	if err != nil {
		t.Fatalf("Do request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		t.Fatalf("Token exchange failed: %d %+v", resp.StatusCode, errResp)
	}
	var workerToken struct {
		AccessToken string `json:"access_token"`
	}
	ts.DecodeJSON(resp, &workerToken)

	if workerToken.AccessToken == "" {
		t.Fatal("expected delegated access_token")
	}
}
