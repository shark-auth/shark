// Package cmd — shared helpers for CLI commands that talk to the admin HTTP API.
//
// Every subcommand that wraps an admin API endpoint (proxy lifecycle, proxy
// rules, branding, user tier, …) reads the admin token from the environment
// variable SHARK_ADMIN_TOKEN or the persistent --token flag on the root
// command, and targets the server at SHARK_URL or the --url flag.
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// adminBaseURL is the global base URL for the admin API. Resolved from
// --url flag (adminURLFlag) > SHARK_URL env var > default.
var adminURLFlag string

// adminTokenFlag is the global admin token. Resolved from
// --token flag > SHARK_ADMIN_TOKEN env var.
var adminTokenFlag string

// adminClient is a shared HTTP client used by all admin API commands.
var adminClient = &http.Client{Timeout: 10 * time.Second}

// resolveAdminURL returns the admin API base URL, stripping a trailing slash.
func resolveAdminURL(cmd *cobra.Command) string {
	// prefer explicit --url flag
	if v, _ := cmd.Root().PersistentFlags().GetString("url"); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := os.Getenv("SHARK_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8080"
}

// resolveAdminToken returns the admin Bearer token.
func resolveAdminToken(cmd *cobra.Command) (string, error) {
	if v, _ := cmd.Root().PersistentFlags().GetString("token"); v != "" {
		return v, nil
	}
	if v := os.Getenv("SHARK_ADMIN_TOKEN"); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("admin token required: set SHARK_ADMIN_TOKEN env var or pass --token")
}

// adminDo performs an HTTP request against the admin API and returns the
// decoded response body (as map[string]any) plus the status code. On a
// non-2xx status the first return value still contains the decoded body so
// callers can surface API error details.
func adminDo(cmd *cobra.Command, method, path string, reqBody any) (map[string]any, int, error) {
	baseURL := resolveAdminURL(cmd)
	token, err := resolveAdminToken(cmd)
	if err != nil {
		return nil, 0, err
	}

	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := adminClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request to %s%s: %w", baseURL, path, err)
	}
	defer resp.Body.Close()

	// Decode the body regardless of status — errors are usually JSON.
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result, resp.StatusCode, nil
}

// adminDoRaw is like adminDo but returns the raw body bytes (useful for HTML
// paywall content) and the Content-Type header.
func adminDoRaw(cmd *cobra.Command, method, path string) ([]byte, int, string, error) {
	baseURL := resolveAdminURL(cmd)
	token, _ := resolveAdminToken(cmd) // whoami / paywall may not need auth

	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		return nil, 0, "", fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := adminClient.Do(req)
	if err != nil {
		return nil, 0, "", fmt.Errorf("request to %s%s: %w", baseURL, path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, resp.Header.Get("Content-Type"), nil
}

// apiError extracts a human-readable error message from an API error response.
func apiError(body map[string]any, statusCode int) string {
	if body == nil {
		return fmt.Sprintf("HTTP %d", statusCode)
	}
	// { "error": { "code": "...", "message": "..." } }
	if errObj, ok := body["error"]; ok {
		switch v := errObj.(type) {
		case map[string]any:
			if msg, ok := v["message"].(string); ok {
				return msg
			}
			if code, ok := v["code"].(string); ok {
				return code
			}
		case string:
			return v
		}
	}
	// { "message": "..." }
	if msg, ok := body["message"].(string); ok {
		return msg
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

func init() {
	// Persistent flags on the root so every subcommand inherits them.
	root.PersistentFlags().String("url", "", "base URL of the running shark instance (default: SHARK_URL or http://localhost:8080)")
	root.PersistentFlags().String("token", "", "admin API token (default: SHARK_ADMIN_TOKEN)")
}
