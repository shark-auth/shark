package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type transparentMockResolver struct {
	resolveFn func(ctx context.Context, host string) (*ResolvedApp, error)
}

func (m *transparentMockResolver) ResolveApp(ctx context.Context, host string) (*ResolvedApp, error) {
	return m.resolveFn(ctx, host)
}

func TestReverseProxy_TransparentModeAppIDInjection(t *testing.T) {
	// 1. Setup engine with a rule that REQUIRES an AppID.
	engine, err := NewEngine([]RuleSpec{
		{
			AppID: "app_123",
			Path:  "/protected/*",
			Allow: "anonymous", // allow anonymous but ONLY if AppID matches
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// 2. Setup mock resolver that returns a ResolvedApp with that engine.
	resolver := &transparentMockResolver{
		resolveFn: func(ctx context.Context, host string) (*ResolvedApp, error) {
			if host == "app.example.com" {
				return &ResolvedApp{
					ID:     "app_123",
					Engine: engine,
				}, nil
			}
			return nil, nil
		},
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, _ := New(Config{Enabled: true, Upstream: upstream.URL}, nil, nil)
	p.SetResolver(resolver)

	// 3. Perform a request WITHOUT the X-Shark-App-ID header.
	// It should still match the rule because the proxy injected the ID from resolution.
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/protected/foo", nil)
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d. Reason: %s", rec.Code, rec.Header().Get(HeaderDenyReason))
	}
}

func TestReverseProxy_TransparentModeAppIDMismatch(t *testing.T) {
	// If the user sends a WRONG AppID header in transparent mode, it should
	// probably respect the header OR the resolution. 
	// Currently, the implementation only injects if MISSING.
	
	engine, err := NewEngine([]RuleSpec{
		{
			AppID: "app_123",
			Path:  "/protected/*",
			Allow: "anonymous",
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	resolver := &transparentMockResolver{
		resolveFn: func(ctx context.Context, host string) (*ResolvedApp, error) {
			return &ResolvedApp{
				ID:     "app_123",
				Engine: engine,
			}, nil
		},
	}

	p, _ := New(Config{Enabled: true, Upstream: "http://example.com"}, nil, nil)
	p.SetResolver(resolver)

	// Request for app_123 host but sending X-Shark-App-ID: attacker.
	// We provide an identity so it doesn't trigger the anonymous redirect/401 logic.
	req := httptest.NewRequest(http.MethodGet, "http://app.example.com/protected/foo", nil)
	req = req.WithContext(WithIdentity(req.Context(), Identity{UserID: "u1", AuthMethod: "jwt"}))
	req.Header.Set("X-Shark-App-ID", "attacker")
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for AppID mismatch, got %d", rec.Code)
	}
}
