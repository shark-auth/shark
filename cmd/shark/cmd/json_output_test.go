//go:build integration

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestAppListJSON_Empty verifies --json emits a valid empty JSON array when no apps exist.
func TestAppListJSON_Empty(t *testing.T) {
	configPath, _ := setupTestDB(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	appListCmd.SetOut(stdout)
	appListCmd.SetErr(stderr)
	t.Cleanup(func() {
		appListCmd.SetOut(nil)
		appListCmd.SetErr(nil)
	})

	if err := appListCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appListCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = appListCmd.Flags().Set("json", "false") })

	if err := appListCmd.RunE(appListCmd, nil); err != nil {
		t.Fatalf("app list --json: %v", err)
	}

	var arr []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &arr); err != nil {
		t.Fatalf("stdout not JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %d items", len(arr))
	}
}

// TestAppListJSON_HappyPath asserts --json returns a well-formed array with expected keys.
func TestAppListJSON_HappyPath(t *testing.T) {
	configPath, store := setupTestDB(t)

	// Seed one app through the CLI.
	appCreateName = "JSON Test App"
	appCreateCallbacks = []string{"https://json.example.com/cb"}
	appCreateLogouts = nil
	appCreateOrigins = nil
	if err := appCreateCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appCreateCmd.RunE(appCreateCmd, nil); err != nil {
		t.Fatalf("seed app: %v", err)
	}
	_ = store // keep reference alive

	stdout := &bytes.Buffer{}
	appListCmd.SetOut(stdout)
	t.Cleanup(func() { appListCmd.SetOut(nil) })

	if err := appListCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appListCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = appListCmd.Flags().Set("json", "false") })

	if err := appListCmd.RunE(appListCmd, nil); err != nil {
		t.Fatalf("app list --json: %v", err)
	}

	var arr []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &arr); err != nil {
		t.Fatalf("stdout not JSON: %v\noutput: %s", err, stdout.String())
	}
	if len(arr) == 0 {
		t.Fatalf("expected at least 1 app, got 0")
	}
	want := []string{"id", "name", "client_id", "client_secret_prefix", "is_default", "allowed_callback_urls", "created_at"}
	for _, k := range want {
		if _, ok := arr[0][k]; !ok {
			t.Errorf("missing key %q in JSON element: %+v", k, arr[0])
		}
	}
}

// TestAppShowJSON_ErrorPath asserts invalid id yields JSON error on stderr,
// empty stdout, and a non-nil error from RunE (→ non-zero exit code).
func TestAppShowJSON_ErrorPath(t *testing.T) {
	configPath, _ := setupTestDB(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	appShowCmd.SetOut(stdout)
	appShowCmd.SetErr(stderr)
	t.Cleanup(func() {
		appShowCmd.SetOut(nil)
		appShowCmd.SetErr(nil)
	})

	if err := appShowCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appShowCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = appShowCmd.Flags().Set("json", "false") })

	err := appShowCmd.RunE(appShowCmd, []string{"nonexistent-id"})
	if err == nil {
		t.Fatal("expected RunE error for unknown id, got nil")
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Errorf("stdout must be empty on error, got: %q", stdout.String())
	}
	var errPayload map[string]any
	if jerr := json.Unmarshal(stderr.Bytes(), &errPayload); jerr != nil {
		t.Fatalf("stderr not JSON: %v\nstderr: %s", jerr, stderr.String())
	}
	if _, ok := errPayload["error"]; !ok {
		t.Errorf("missing 'error' key: %+v", errPayload)
	}
	if _, ok := errPayload["message"]; !ok {
		t.Errorf("missing 'message' key: %+v", errPayload)
	}
}

// TestAppCreateJSON_HappyPath asserts create --json emits {app, secret}.
func TestAppCreateJSON_HappyPath(t *testing.T) {
	configPath, store := setupTestDB(t)

	stdout := &bytes.Buffer{}
	appCreateCmd.SetOut(stdout)
	t.Cleanup(func() { appCreateCmd.SetOut(nil) })

	appCreateName = "Create JSON App"
	appCreateCallbacks = nil
	appCreateLogouts = nil
	appCreateOrigins = nil
	if err := appCreateCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appCreateCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = appCreateCmd.Flags().Set("json", "false") })

	if err := appCreateCmd.RunE(appCreateCmd, nil); err != nil {
		t.Fatalf("app create --json: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout not JSON: %v\noutput: %s", err, stdout.String())
	}
	if _, ok := payload["app"]; !ok {
		t.Errorf("missing 'app' key: %+v", payload)
	}
	secret, ok := payload["secret"].(string)
	if !ok || secret == "" {
		t.Errorf("missing/empty 'secret': %+v", payload)
	}

	// Sanity: app is actually in DB.
	apps, err := store.ListApplications(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(apps) == 0 {
		t.Fatal("expected app in db")
	}
}

// TestAppCreateJSON_ErrorPath asserts missing --name triggers JSON error on stderr.
func TestAppCreateJSON_ErrorPath(t *testing.T) {
	configPath, _ := setupTestDB(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	appCreateCmd.SetOut(stdout)
	appCreateCmd.SetErr(stderr)
	t.Cleanup(func() {
		appCreateCmd.SetOut(nil)
		appCreateCmd.SetErr(nil)
	})

	appCreateName = "" // trigger the "--name is required" path
	appCreateCallbacks = nil
	appCreateLogouts = nil
	appCreateOrigins = nil
	if err := appCreateCmd.Flags().Set("config", configPath); err != nil {
		t.Fatalf("set config: %v", err)
	}
	if err := appCreateCmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("set json: %v", err)
	}
	t.Cleanup(func() { _ = appCreateCmd.Flags().Set("json", "false") })

	err := appCreateCmd.RunE(appCreateCmd, nil)
	if err == nil {
		t.Fatal("expected error on missing --name")
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Errorf("stdout must be empty on error, got: %q", stdout.String())
	}
	var errPayload map[string]any
	if jerr := json.Unmarshal(stderr.Bytes(), &errPayload); jerr != nil {
		t.Fatalf("stderr not JSON: %v\nstderr: %s", jerr, stderr.String())
	}
	if errPayload["error"] != "invalid_args" {
		t.Errorf("wrong error code: %+v", errPayload)
	}
}
