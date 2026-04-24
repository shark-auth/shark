package cli_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil/cli"
)

// TestE2E_ProxyRulesCRUD drives the proxy rules admin API end-to-end using an
// in-process server (same harness as TestE2EServeFlow). Exercises create,
// list, get, and delete via raw HTTP — mirroring what the CLI commands do
// under the hood. This validates the admin API surface that Lane E CLI wraps.
func TestE2E_ProxyRulesCRUD(t *testing.T) {
	h := cli.Start(t)

	// ---- Create a rule ----
	createBody := map[string]any{
		"name":    "e2e-test-rule",
		"pattern": "/e2e/test/*",
		"require": "authenticated",
		"enabled": true,
	}
	createData, _ := json.Marshal(createBody)
	createReq, _ := http.NewRequest("POST", h.BaseURL+"/api/v1/admin/proxy/rules/db",
		bytes.NewReader(createData))
	createReq.Header.Set("Authorization", "Bearer "+h.AdminKey)
	createReq.Header.Set("Content-Type", "application/json")

	createResp := h.Do(createReq)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create rule: expected 201, got %d", createResp.StatusCode)
	}
	var createResult map[string]any
	_ = json.NewDecoder(createResp.Body).Decode(&createResult)
	createResp.Body.Close()

	data, ok := createResult["data"].(map[string]any)
	if !ok {
		t.Fatalf("create response missing 'data': %+v", createResult)
	}
	ruleID, _ := data["id"].(string)
	if ruleID == "" {
		t.Fatalf("create response missing rule id: %+v", data)
	}

	// ---- List rules ----
	listResp := h.Do(h.AdminRequest("GET", "/api/v1/admin/proxy/rules/db"))
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list rules: expected 200, got %d", listResp.StatusCode)
	}
	var listResult map[string]any
	_ = json.NewDecoder(listResp.Body).Decode(&listResult)
	listResp.Body.Close()

	arr, _ := listResult["data"].([]any)
	found := false
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			if m["id"] == ruleID {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("created rule %s not found in list: %+v", ruleID, listResult)
	}

	// ---- Get by ID ----
	// handleGetProxyRule returns the rule directly (no "data" wrapper).
	getResp := h.Do(h.AdminRequest("GET", "/api/v1/admin/proxy/rules/db/"+ruleID))
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get rule: expected 200, got %d", getResp.StatusCode)
	}
	var getRule map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&getRule)
	getResp.Body.Close()

	if getRule["pattern"] != "/e2e/test/*" {
		t.Errorf("unexpected pattern in get result: %+v", getRule)
	}

	// ---- Delete ----
	deleteResp := h.Do(h.AdminRequest("DELETE", "/api/v1/admin/proxy/rules/db/"+ruleID))
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete rule: expected 204, got %d", deleteResp.StatusCode)
	}
	deleteResp.Body.Close()

	// ---- Verify gone ----
	goneResp := h.Do(h.AdminRequest("GET", "/api/v1/admin/proxy/rules/db/"+ruleID))
	if goneResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", goneResp.StatusCode)
	}
	goneResp.Body.Close()
}

// TestE2E_ProxyRulesImport verifies the YAML import endpoint accepts a
// well-formed YAML payload and returns imported > 0.
func TestE2E_ProxyRulesImport(t *testing.T) {
	h := cli.Start(t)

	yamlContent := `rules:
  - path: /import/test/*
    require: authenticated
    name: import-test-rule
`
	importPayload := map[string]any{"yaml": yamlContent}
	importData, _ := json.Marshal(importPayload)
	req, _ := http.NewRequest("POST", h.BaseURL+"/api/v1/admin/proxy/rules/import",
		bytes.NewReader(importData))
	req.Header.Set("Authorization", "Bearer "+h.AdminKey)
	req.Header.Set("Content-Type", "application/json")

	resp := h.Do(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import: expected 200, got %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	imported, _ := result["imported"].(float64)
	if imported < 1 {
		t.Errorf("expected imported >= 1, got %v: %+v", imported, result)
	}
	errs, _ := result["errors"].([]any)
	if len(errs) > 0 {
		t.Errorf("import errors: %+v", errs)
	}
}

// TestE2E_UserTierSet verifies the set-user-tier endpoint end-to-end.
func TestE2E_UserTierSet(t *testing.T) {
	h := cli.Start(t)

	// Create a user via signup.
	signupBody := `{"email":"tier-e2e@example.com","password":"Password123!"}`
	signupReq, _ := http.NewRequest("POST", h.BaseURL+"/api/v1/auth/signup",
		strings.NewReader(signupBody))
	signupReq.Header.Set("Content-Type", "application/json")
	signupResp, err := http.DefaultClient.Do(signupReq)
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	signupResp.Body.Close()
	if signupResp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: expected 201, got %d", signupResp.StatusCode)
	}

	// Lookup user by email.
	listResp := h.Do(h.AdminRequest("GET", "/api/v1/users?search=tier-e2e%40example.com"))
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list users: %d", listResp.StatusCode)
	}
	var listResult map[string]any
	_ = json.NewDecoder(listResp.Body).Decode(&listResult)
	listResp.Body.Close()

	users, _ := listResult["users"].([]any)
	if len(users) == 0 {
		t.Fatal("expected at least one user after signup")
	}
	user, _ := users[0].(map[string]any)
	userID, _ := user["id"].(string)
	if userID == "" {
		t.Fatal("user id is empty")
	}

	// Set tier to pro.
	tierBody, _ := json.Marshal(map[string]any{"tier": "pro"})
	tierReq, _ := http.NewRequest("PATCH",
		h.BaseURL+"/api/v1/admin/users/"+userID+"/tier",
		bytes.NewReader(tierBody))
	tierReq.Header.Set("Authorization", "Bearer "+h.AdminKey)
	tierReq.Header.Set("Content-Type", "application/json")

	tierResp := h.Do(tierReq)
	if tierResp.StatusCode != http.StatusOK {
		t.Fatalf("set tier: expected 200, got %d", tierResp.StatusCode)
	}
	var tierResult map[string]any
	_ = json.NewDecoder(tierResp.Body).Decode(&tierResult)
	tierResp.Body.Close()

	tierData, _ := tierResult["data"].(map[string]any)
	if tierData["tier"] != "pro" {
		t.Errorf("expected tier=pro in response: %+v", tierResult)
	}
}
