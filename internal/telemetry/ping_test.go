package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestPingLoop_GeneratesIDAndPersists verifies a first boot flow end-to-end:
// install_id file is created, the mock telemetry server receives a single
// well-formed ping, and the persisted UUID matches the one the server saw.
func TestPingLoop_GeneratesIDAndPersists(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	idPath := filepath.Join(tmp, "install_id")

	var received atomic.Int32
	gotID := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		var p Payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Errorf("decode payload: %v", err)
		}
		select {
		case gotID <- p.InstallID:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		Enabled:       true,
		Endpoint:      srv.URL,
		Version:       "v0.0.0-test",
		InstallIDPath: idPath,
	}
	// Fire immediately with a near-zero initial delay so the test doesn't wait 30s.
	go runLoop(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), srv.Client(), time.Millisecond)

	var serverID string
	select {
	case serverID = <-gotID:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ping")
	}

	b, err := os.ReadFile(idPath)
	if err != nil {
		t.Fatalf("read install_id: %v", err)
	}
	diskID := strings.TrimSpace(string(b))
	if diskID == "" {
		t.Fatal("install_id file empty")
	}
	if diskID != serverID {
		t.Fatalf("install_id mismatch: disk=%q server=%q", diskID, serverID)
	}
}

// TestPingLoop_Disabled_NoPing verifies that with Enabled:false, StartPingLoop
// never makes any HTTP call and never creates an install_id file.
func TestPingLoop_Disabled_NoPing(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	idPath := filepath.Join(tmp, "install_id")

	var hit atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	StartPingLoop(ctx, Config{
		Enabled:       false,
		Endpoint:      srv.URL,
		Version:       "v0.0.0-test",
		InstallIDPath: idPath,
	}, nil)

	// Give any errant goroutine a chance to misbehave.
	time.Sleep(50 * time.Millisecond)

	if hit.Load() != 0 {
		t.Fatalf("expected 0 pings when disabled, got %d", hit.Load())
	}
	if _, err := os.Stat(idPath); !os.IsNotExist(err) {
		t.Fatalf("expected install_id not to exist, stat err=%v", err)
	}
}

// TestPingLoop_MockServer_Shape locks in the wire contract: JSON shape, all
// required fields present, correct Content-Type and User-Agent headers.
func TestPingLoop_MockServer_Shape(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	idPath := filepath.Join(tmp, "install_id")

	type captured struct {
		contentType string
		userAgent   string
		payload     map[string]any
	}
	done := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		_ = json.NewDecoder(r.Body).Decode(&m)
		select {
		case done <- captured{
			contentType: r.Header.Get("Content-Type"),
			userAgent:   r.Header.Get("User-Agent"),
			payload:     m,
		}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{
		Enabled:       true,
		Endpoint:      srv.URL,
		Version:       "v1.2.3",
		InstallIDPath: idPath,
	}
	go runLoop(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), srv.Client(), time.Millisecond)

	var got captured
	select {
	case got = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ping")
	}

	if got.contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got.contentType)
	}
	if !strings.HasPrefix(got.userAgent, "SharkAuth/v1.2.3") {
		t.Errorf("User-Agent = %q, want prefix SharkAuth/v1.2.3", got.userAgent)
	}
	for _, k := range []string{"install_id", "version", "os", "arch", "uptime_s", "started_at", "go_version"} {
		if _, ok := got.payload[k]; !ok {
			t.Errorf("payload missing key %q; got %v", k, got.payload)
		}
	}
	if v, _ := got.payload["version"].(string); v != "v1.2.3" {
		t.Errorf("payload.version = %q, want v1.2.3", v)
	}

	// Sanity: no fields we explicitly promise not to send.
	for _, forbidden := range []string{"user_count", "app_count", "request_count", "hostname", "ip"} {
		if _, ok := got.payload[forbidden]; ok {
			t.Errorf("payload must not include %q", forbidden)
		}
	}
}
