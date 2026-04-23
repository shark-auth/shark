package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestListener_StartShutdown verifies bind, serve, and graceful shutdown
// with a real upstream, real loopback listener, and real http client.
func TestListener_StartShutdown(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hi from upstream")
	}))
	defer upstream.Close()

	// Grab an ephemeral free port then close it so NewListener can bind it
	// back a few microseconds later. Racy in principle, reliable on loopback.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-bind: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	l, err := NewListener(ListenerParams{
		Bind:          addr,
		Upstream:      upstream.URL,
		Timeout:       2 * time.Second,
		StripIncoming: true,
		Rules: []RuleSpec{
			{Path: "/*", Allow: "anonymous"},
		},
		BreakerConfig: BreakerConfig{
			HealthURL:        upstream.URL,
			HealthInterval:   time.Hour, // keep the monitor quiet during the test
			FailureThreshold: 3,
			CacheSize:        10,
			CacheTTL:         time.Minute,
			NegativeTTL:      time.Second,
			MissBehavior:     MissReject,
		},
	})
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := l.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Quick retry loop — goroutine schedule can race the first dial.
	var resp *http.Response
	for i := 0; i < 20; i++ {
		resp, err = http.Get("http://" + addr + "/")
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), "upstream") {
		t.Fatalf("expected 200/upstream, got %d / %q", resp.StatusCode, string(body))
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := l.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestListener_BindInUseIsFatal(t *testing.T) {
	// Hold a port so NewListener -> Start can't bind.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-bind: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	l, err := NewListener(ListenerParams{
		Bind:     addr,
		Upstream: "http://example.invalid",
		BreakerConfig: BreakerConfig{
			HealthURL:      "http://example.invalid",
			HealthInterval: time.Hour,
			CacheSize:      10,
			MissBehavior:   MissReject,
		},
	})
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	if err := l.Start(context.Background()); err == nil {
		t.Fatalf("expected port-in-use error, got nil")
		_ = l.Shutdown(context.Background())
	}
}
