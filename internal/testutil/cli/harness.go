// Package cli provides a test harness that spins up a real HTTP listener via
// server.Build on an ephemeral port so CLI subcommands and end-to-end flows
// can be exercised without a hand-rolled chi.ServeHTTP tree.
//
// This is intentionally separate from internal/testutil (which backs handler
// tests). Those use httptest.NewServer; this one uses net.Listen so code paths
// that depend on process-level shutdown signals remain realistic.
package cli

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/server"
)

//go:embed testmigrations/*.sql
var migrationsFS embed.FS

// Harness runs a fully wired shark server against a temp dev.db on a random
// local port. Start returns once /healthz answers 200.
type Harness struct {
	t        *testing.T
	BaseURL  string
	AdminKey string
	shutdown context.CancelFunc
	done     chan error
}

// Start spins up the server with --dev-equivalent options. The temp directory
// is cleaned automatically when the test ends.
func Start(t *testing.T) *Harness {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick port: %v", err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close() //#nosec G104 -- we only needed the allocated port number

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/shark.db"

	bootstrapCtx, bootstrapCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer bootstrapCancel()

	b, err := server.Build(bootstrapCtx, server.Options{
		MigrationsFS:        migrationsFS,
		MigrationsDir:       "testmigrations",
		DevMode:             true,
		SecretOverride:      "cli-harness-secret-" + fmt.Sprintf("%032d", port),
		StoragePathOverride: dbPath,
	})
	if err != nil {
		t.Fatalf("build server: %v", err)
	}

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf("127.0.0.1:%d", port),
		Handler:      b.API.Router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	done := make(chan error, 1)
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			done <- err
		}
		close(done)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitForHealth(baseURL, 3*time.Second); err != nil {
		httpSrv.Close() //#nosec G104 -- test teardown after health-check failure
		b.Close()       //#nosec G104 -- test teardown after health-check failure
		t.Fatalf("waiting for /healthz: %v", err)
	}

	h := &Harness{
		t:        t,
		BaseURL:  baseURL,
		AdminKey: b.AdminKey,
		done:     done,
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.shutdown = cancel
	go func() {
		<-ctx.Done()
		shutdownCtx, sc := context.WithTimeout(context.Background(), 5*time.Second)
		defer sc()
		_ = httpSrv.Shutdown(shutdownCtx)
		_ = b.Close()
	}()

	t.Cleanup(func() {
		h.Stop()
	})
	return h
}

// Stop triggers graceful shutdown. Safe to call multiple times.
func (h *Harness) Stop() {
	if h.shutdown != nil {
		h.shutdown()
		h.shutdown = nil
	}
	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
	}
}

// AdminRequest builds an http.Request with the admin Bearer header preset.
func (h *Harness) AdminRequest(method, path string) *http.Request {
	h.t.Helper()
	req, err := http.NewRequest(method, h.BaseURL+path, nil)
	if err != nil {
		h.t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+h.AdminKey)
	return req
}

// Do runs the request through the default client.
func (h *Harness) Do(req *http.Request) *http.Response {
	h.t.Helper()
	resp, err := http.DefaultClient.Do(req) //#nosec G107,G704 -- test harness issues localhost requests against its own ephemeral server
	if err != nil {
		h.t.Fatalf("http do: %v", err)
	}
	return resp
}

func waitForHealth(baseURL string, deadline time.Duration) error {
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		resp, err := http.Get(baseURL + "/healthz") //#nosec G107 -- localhost URL built from ephemeral-port the harness just allocated
		if err == nil {
			resp.Body.Close() //#nosec G104 -- test harness; body is discarded

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server never became healthy at %s", baseURL)
}
