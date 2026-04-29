// Package telemetry implements the one-time anonymous install ping.
//
// What we send:
//
//	{
//	  "install_id":  "<uuid v4 + machine entropy, generated on first boot>"
//	}
//
// This ping happens EXACTLY ONCE in the lifetime of the installation.
// Success is tracked by the presence of `install_id.done` in the data directory.
//
// Opt out by setting `telemetry.enabled: false` in sharkauth.yaml or
// SHARKAUTH_TELEMETRY__ENABLED=false in the environment.
package telemetry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// Config is the minimal surface telemetry needs from the main config.
type Config struct {
	Enabled       bool
	Endpoint      string
	Version       string
	InstallIDPath string
}

const (
	// httpTimeout bounds the single ping attempt.
	httpTimeout = 10 * time.Second

	userAgentFmt = "SharkAuth/%s (%s; %s)"
)

// StartPingLoop starts a goroutine that attempts a ONE-TIME ping.
// If the success marker exists, it does nothing.
func StartPingLoop(ctx context.Context, cfg Config, logger *slog.Logger) {
	if !cfg.Enabled {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Check for the completion marker immediately.
	markerPath := cfg.InstallIDPath + ".done"
	if _, err := os.Stat(markerPath); err == nil {
		return // Already pinged in a previous run.
	}

	go runOnce(ctx, cfg, logger)
}

func runOnce(ctx context.Context, cfg Config, logger *slog.Logger) {
	markerPath := cfg.InstallIDPath + ".done"

	// Wait a bit after boot so we don't compete with startup logs/IO.
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}

	installID, isNew, err := loadOrCreateInstallID(cfg.InstallIDPath)
	if err != nil {
		logger.Warn("telemetry: failed to load/create install id", "error", err)
		return
	}

	// If it's an existing ID but no marker, we still try to ping (maybe previous attempt failed).
	// But per user vision: "ping ... on first run of the binary and never again."
	// We interpret this as: if we haven't successfully pinged yet, try now.

	if err := sendPing(ctx, http.DefaultClient, cfg, installID); err != nil {
		logger.Debug("telemetry: ping attempt failed (will retry on next boot)", "error", err)
		return
	}

	// Success! Create the marker.
	if err := os.WriteFile(markerPath, []byte(time.Now().UTC().Format(time.RFC3339)), 0600); err != nil {
		logger.Warn("telemetry: failed to write success marker", "error", err)
	} else {
		logger.Info("telemetry: anonymous install ping successful", "new_install", isNew)
	}
}

// Payload is strictly the install_id.
type Payload struct {
	InstallID string `json:"install_id"`
}

func sendPing(ctx context.Context, client *http.Client, cfg Config, installID string) error {
	payload := Payload{
		InstallID: installID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf(userAgentFmt, cfg.Version, runtime.GOOS, runtime.GOARCH))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// loadOrCreateInstallID returns the ID, whether it was newly created, and any error.
func loadOrCreateInstallID(path string) (string, bool, error) {
	if path == "" {
		return "", false, fmt.Errorf("empty install id path")
	}
	if b, err := os.ReadFile(path); err == nil {
		id := string(bytes.TrimSpace(b))
		if id != "" {
			return id, false, nil
		}
	} else if !os.IsNotExist(err) {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}

	// Create dir if missing.
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", false, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// Strategy for variability:
	// Use UUID v4 as base, but salt it with hostname and current time to ensure
	// that even if a random seed is somehow collided (cloned VMs), the resulting
	// hash is different.
	hostname, _ := os.Hostname()
	entropy := fmt.Sprintf("%s-%s-%d", uuid.NewString(), hostname, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(entropy))
	id := hex.EncodeToString(hash[:16]) // 32 chars of hex

	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		return "", false, fmt.Errorf("write %s: %w", path, err)
	}
	return id, true, nil
}
