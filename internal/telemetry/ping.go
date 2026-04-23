// Package telemetry implements the anonymous install ping.
//
// What we send (documented so operators can audit the wire shape without reading code):
//
//	{
//	  "install_id":  "<uuid v4, generated on first boot, persisted to data/install_id>",
//	  "version":     "<build version, e.g. v0.3.0 or 'dev'>",
//	  "os":          "<runtime.GOOS>",
//	  "arch":        "<runtime.GOARCH>",
//	  "uptime_s":    <int seconds since process start>,
//	  "started_at":  "<RFC3339 timestamp of process start>",
//	  "go_version":  "<runtime.Version()>"
//	}
//
// What we do NOT send: user counts, app counts, request counts, hostnames,
// IPs, secrets, emails, or anything derived from your users or configuration.
// The ping endpoint does not log client IPs server-side.
//
// Opt out by setting `telemetry.enabled: false` in sharkauth.yaml or
// SHARKAUTH_TELEMETRY__ENABLED=false in the environment.
package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Config is the minimal surface telemetry needs from the main config.
// Passed as a struct literal from server.Run rather than importing the full
// config package to keep this leaf free of import cycles.
type Config struct {
	Enabled     bool
	Endpoint    string
	Version     string
	InstallIDPath string
}

const (
	// PingInterval is the cadence between pings. Fixed at 24h: the whole point
	// is "this install exists", not fine-grained metrics.
	PingInterval = 24 * time.Hour

	// httpTimeout bounds a single ping attempt. Short enough that a dead
	// telemetry endpoint never wedges the host process.
	httpTimeout = 5 * time.Second

	userAgentFmt = "SharkAuth/%s (%s; %s)"
)

// StartPingLoop kicks off a goroutine that pings the telemetry endpoint every
// 24h. Returns immediately. When cfg.Enabled is false, it is a no-op — no
// goroutine is spawned, no install_id is generated, no HTTP call is ever made.
//
// The first ping runs ~30s after boot so we don't delay startup. Subsequent
// pings are on the PingInterval cadence.
func StartPingLoop(ctx context.Context, cfg Config, logger *slog.Logger) {
	if !cfg.Enabled {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}
	go runLoop(ctx, cfg, logger, http.DefaultClient, 30*time.Second)
}

func runLoop(ctx context.Context, cfg Config, logger *slog.Logger, client *http.Client, initialDelay time.Duration) {
	startedAt := time.Now().UTC()
	installID, err := loadOrCreateInstallID(cfg.InstallIDPath)
	if err != nil {
		logger.Warn("telemetry: failed to load/create install id; disabling ping loop", "error", err)
		return
	}

	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := sendPing(ctx, client, cfg, installID, startedAt); err != nil {
				logger.Warn("telemetry: ping failed", "error", err)
			}
			timer.Reset(PingInterval)
		}
	}
}

// Payload is the wire shape of a single ping. See package doc for field semantics.
type Payload struct {
	InstallID string `json:"install_id"`
	Version   string `json:"version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	UptimeS   int64  `json:"uptime_s"`
	StartedAt string `json:"started_at"`
	GoVersion string `json:"go_version"`
}

func sendPing(ctx context.Context, client *http.Client, cfg Config, installID string, startedAt time.Time) error {
	payload := Payload{
		InstallID: installID,
		Version:   cfg.Version,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		UptimeS:   int64(time.Since(startedAt).Seconds()),
		StartedAt: startedAt.Format(time.RFC3339),
		GoVersion: runtime.Version(),
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
	defer resp.Body.Close() //#nosec G104 -- response body is discarded; close error is uninteresting
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// loadOrCreateInstallID reads the uuid from path, or generates + persists a
// new one. The file is plain text: single UUID v4 line, no metadata.
func loadOrCreateInstallID(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty install id path")
	}
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if id != "" {
			return id, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	// Create dir if missing (mirrors storage dir perms — 0o700).
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	id := uuid.NewString()
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return id, nil
}
