package api_test

import (
	"net/http"
	"testing"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func TestAgentCRUD(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 1. Create Agent
	createPayload := map[string]interface{}{
		"name":        "Test Agent",
		"description": "A test agent for CRUD verification",
		"scopes":      []string{"read", "write"},
	}

	resp := ts.PostJSONWithAdminKey("/api/v1/agents", createPayload)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/v1/agents: expected 201, got %d", resp.StatusCode)
	}

	var agentResp struct {
		ID           string `json:"id"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Name         string `json:"name"`
	}
	ts.DecodeJSON(resp, &agentResp)

	if agentResp.Name != "Test Agent" {
		t.Errorf("agent name: got %q, want %q", agentResp.Name, "Test Agent")
	}
	if agentResp.ClientSecret == "" {
		t.Error("expected client_secret in creation response")
	}

	agentID := agentResp.ID

	// 2. GET /agents
	resp = ts.GetWithAdminKey("/api/v1/agents")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/agents: expected 200, got %d", resp.StatusCode)
	}
	var listResp struct {
		Data []*storage.Agent `json:"data"`
	}
	ts.DecodeJSON(resp, &listResp)
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least one agent in list")
	}

	// 3. GET /agents/{id}
	resp = ts.GetWithAdminKey("/api/v1/agents/" + agentID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/agents/{id}: expected 200, got %d", resp.StatusCode)
	}
	var singleAgent storage.Agent
	ts.DecodeJSON(resp, &singleAgent)
	if singleAgent.ID != agentID {
		t.Errorf("got agent id %q, want %q", singleAgent.ID, agentID)
	}

	// 4. PATCH /agents/{id}
	updatePayload := map[string]interface{}{
		"description": "Updated description",
	}
	resp = ts.PatchJSONWithAdminKey("/api/v1/agents/"+agentID, updatePayload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /api/v1/agents/{id}: expected 200, got %d", resp.StatusCode)
	}
	ts.DecodeJSON(resp, &singleAgent)
	if singleAgent.Description != "Updated description" {
		t.Errorf("got description %q, want %q", singleAgent.Description, "Updated description")
	}

	// 5. DELETE /agents/{id} (Deactivate)
	resp = ts.DeleteWithAdminKey("/api/v1/agents/" + agentID)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /api/v1/agents/{id}: expected 204, got %d", resp.StatusCode)
	}

	// Verify it's inactive
	resp = ts.GetWithAdminKey("/api/v1/agents/" + agentID)
	ts.DecodeJSON(resp, &singleAgent)
	if singleAgent.Active {
		t.Error("expected agent to be inactive after DELETE")
	}
}

func TestAgentCreate_Validation(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Name is required
	resp := ts.PostJSONWithAdminKey("/api/v1/agents", map[string]interface{}{
		"description": "missing name",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
	}
}

func TestAgentAuthEnforcement(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// No admin key
	resp := ts.Get("/api/v1/agents")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without admin key, got %d", resp.StatusCode)
	}
}
