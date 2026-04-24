package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestLogger returns a slog.Logger that writes to buf so tests can
// assert on log output (e.g. panic recovery). Level is set to Debug so
// nothing is filtered out.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func newProxy(t *testing.T, upstream string, logger *slog.Logger) *ReverseProxy {
	t.Helper()
	cfg := Config{
		Enabled:       true,
		Upstream:      upstream,
		Timeout:       2 * time.Second,
		StripIncoming: true,
	}
	p, err := New(cfg, nil, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

func TestReverseProxy_ForwardsBasicRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream-Hit", "1")
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	p.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusTeapot {
		t.Errorf("status: got %d, want %d", res.StatusCode, http.StatusTeapot)
	}
	if got := res.Header.Get("X-Upstream-Hit"); got != "1" {
		t.Errorf("upstream response header missing, got %q", got)
	}
	body, _ := io.ReadAll(res.Body)
	if string(body) != "hello from upstream" {
		t.Errorf("body: got %q", string(body))
	}
}

func TestReverseProxy_InjectsIdentity(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	id := Identity{
		UserID:     "user-42",
		UserEmail:  "alice@example.com",
		Roles:      []string{"admin"},
		AuthMethod: "jwt",
	}
	req = req.WithContext(WithIdentity(req.Context(), id))

	p.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen.Get(HeaderUserID); got != "user-42" {
		t.Errorf("X-User-ID: got %q, want user-42", got)
	}
	if got := seen.Get(HeaderUserEmail); got != "alice@example.com" {
		t.Errorf("X-User-Email: got %q", got)
	}
	if got := seen.Get(HeaderUserRoles); got != "admin" {
		t.Errorf("X-User-Roles: got %q", got)
	}
	if got := seen.Get(HeaderAuthMethod); got != "jwt" {
		t.Errorf("X-Auth-Method: got %q", got)
	}
	if got := seen.Get(HeaderAuthMode); got != "jwt" {
		t.Errorf("X-Shark-Auth-Mode: got %q", got)
	}
}

func TestReverseProxy_AnonymousWhenNoIdentity(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	p.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := seen.Get(HeaderAuthMethod); got != "anonymous" {
		t.Errorf("X-Auth-Method: got %q, want anonymous", got)
	}
	if got := seen.Get(HeaderUserID); got != "" {
		t.Errorf("X-User-ID should be empty for anonymous, got %q", got)
	}
}

func TestReverseProxy_StripsSpoofedHeaders(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderUserID, "attacker")
	req.Header.Set(HeaderAuthMethod, "spoofed")
	req.Header.Set("X-Agent-ID", "fake-agent")
	req.Header.Set("X-Shark-Cache-Age", "9999")

	// Legitimate identity resolved by (hypothetical) auth middleware.
	req = req.WithContext(WithIdentity(req.Context(), Identity{
		UserID:     "real-user",
		AuthMethod: "jwt",
	}))

	p.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen.Get(HeaderUserID); got != "real-user" {
		t.Errorf("X-User-ID should be the real one, got %q", got)
	}
	if got := seen.Get(HeaderAuthMethod); got != "jwt" {
		t.Errorf("X-Auth-Method should be real, got %q", got)
	}
	// Agent + cache-age were not injected (empty on Identity), so they
	// should be absent after strip even though the client sent them.
	if got := seen.Get("X-Agent-ID"); got != "" {
		t.Errorf("spoofed X-Agent-ID leaked through: %q", got)
	}
	if got := seen.Get("X-Shark-Cache-Age"); got != "" {
		t.Errorf("spoofed X-Shark-Cache-Age leaked through: %q", got)
	}
}

// panicTransport is a http.RoundTripper that always panics; used to
// verify ServeHTTP's recover() produces a 503 rather than crashing.
type panicTransport struct{}

func (panicTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("synthetic transport panic")
}

func TestReverseProxy_PanicRecovery(t *testing.T) {
	var logBuf bytes.Buffer
	logger := newTestLogger(&logBuf)

	p, err := New(Config{
		Enabled:       true,
		Upstream:      "http://example.invalid",
		Timeout:       1 * time.Second,
		StripIncoming: true,
	}, nil, logger)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Swap the transport for one that panics.
	p.backend.Transport = panicTransport{}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)

	// Must not crash the test process.
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rec.Code)
	}
	if !strings.Contains(logBuf.String(), "proxy panic") {
		t.Errorf("expected \"proxy panic\" in log, got: %s", logBuf.String())
	}
}

func TestReverseProxy_UpstreamTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than the proxy's configured timeout. Use the
		// request context so the test can't hang if the proxy's
		// cancellation propagates.
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()

	cfg := Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       75 * time.Millisecond,
		StripIncoming: true,
	}
	p, err := New(cfg, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)

	done := make(chan struct{})
	go func() {
		p.ServeHTTP(rec, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("ServeHTTP hung past timeout+margin")
	}

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status: got %d, want 502", rec.Code)
	}
}

func TestReverseProxy_UpstreamUnreachable(t *testing.T) {
	var logBuf bytes.Buffer
	// 127.0.0.1:1 is reliably closed on every platform we ship on.
	p, err := New(Config{
		Enabled:       true,
		Upstream:      "http://127.0.0.1:1",
		Timeout:       500 * time.Millisecond,
		StripIncoming: true,
	}, nil, newTestLogger(&logBuf))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status: got %d, want 502", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "upstream unreachable") {
		t.Errorf("body: got %q, want generic message", body)
	}
}

func TestReverseProxy_PreservesPathAndQuery(t *testing.T) {
	var seenPath, seenQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	req := httptest.NewRequest(http.MethodGet, "/foo/bar?baz=1&qux=2", nil)
	p.ServeHTTP(httptest.NewRecorder(), req)

	if seenPath != "/foo/bar" {
		t.Errorf("path: got %q, want /foo/bar", seenPath)
	}
	if seenQuery != "baz=1&qux=2" {
		t.Errorf("query: got %q, want baz=1&qux=2", seenQuery)
	}
}

func TestReverseProxy_PreservesMethodAndBody(t *testing.T) {
	var seenMethod string
	var seenBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		seenBody, _ = io.ReadAll(r.Body)
	}))
	defer upstream.Close()

	p := newProxy(t, upstream.URL, nil)
	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	p.ServeHTTP(httptest.NewRecorder(), req)

	if seenMethod != http.MethodPost {
		t.Errorf("method: got %q, want POST", seenMethod)
	}
	if !bytes.Equal(seenBody, body) {
		t.Errorf("body: got %q, want %q", seenBody, body)
	}
}

func TestReverseProxy_New_RejectsMissingUpstream(t *testing.T) {
	_, err := New(Config{Enabled: true}, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing upstream, got nil")
	}
}

func TestReverseProxy_New_AcceptsDisabledWithoutUpstream(t *testing.T) {
	_, err := New(Config{Enabled: false}, nil, nil)
	if err != nil {
		t.Fatalf("disabled config should not require upstream: %v", err)
	}
}

func TestReverseProxy_New_RejectsMalformedUpstream(t *testing.T) {
	_, err := New(Config{Enabled: true, Upstream: "not-a-url"}, nil, nil)
	if err == nil {
		t.Fatal("expected error for malformed upstream, got nil")
	}
}

func TestReverseProxy_New_NilLoggerUsesDefault(t *testing.T) {
	p, err := New(Config{Enabled: true, Upstream: "http://example.com"}, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.logger == nil {
		t.Fatal("expected default logger, got nil")
	}
}

func TestIdentityFromContext_Roundtrip(t *testing.T) {
	id := Identity{UserID: "u1", AuthMethod: "jwt"}
	ctx := WithIdentity(context.Background(), id)
	got, ok := IdentityFromContext(ctx)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if got.UserID != "u1" || got.AuthMethod != "jwt" {
		t.Errorf("unexpected identity: %+v", got)
	}

	_, ok = IdentityFromContext(context.Background())
	if ok {
		t.Error("expected ok=false for empty context")
	}
}

func TestReverseProxy_NoStripWhenStripIncomingFalse(t *testing.T) {
	var seen http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
	}))
	defer upstream.Close()

	cfg := Config{
		Enabled:       true,
		Upstream:      upstream.URL,
		Timeout:       time.Second,
		StripIncoming: false,
	}
	p, err := New(cfg, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Use a non-canonical identity-prefixed header (not one of our 8
	// reserved names). Inject won't touch it; with StripIncoming=false
	// the strip pass doesn't run either, so the client value should
	// reach the upstream untouched.
	req.Header.Set("X-Shark-Trace-Id", "from-client")
	p.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen.Get("X-Shark-Trace-Id"); got != "from-client" {
		t.Errorf("StripIncoming=false should preserve client header, got %q", got)
	}
}

// Sanity: effectiveTimeout returns the default when cfg.Timeout is zero.
func TestReverseProxy_EffectiveTimeoutDefault(t *testing.T) {
	p, err := New(Config{Enabled: true, Upstream: "http://example.com"}, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := p.effectiveTimeout(); got != DefaultTimeout {
		t.Errorf("effectiveTimeout: got %v, want %v", got, DefaultTimeout)
	}
}

// Verify the ErrorHandler path directly with a synthetic error so the
// "upstream unreachable" body is exercised even if the dial test above
// someday picks a different error surface.
func TestReverseProxy_ErrorHandlerBody(t *testing.T) {
	p, err := New(Config{Enabled: true, Upstream: "http://example.com"}, nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	p.onUpstreamError(rec, req, errors.New("dial tcp: refused"))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status: got %d, want 502", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "upstream unreachable") {
		t.Errorf("body: got %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "dial tcp") {
		t.Errorf("body leaked internal error detail: %q", rec.Body.String())
	}
}
