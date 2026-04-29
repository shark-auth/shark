package webhook_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
	"github.com/shark-auth/shark/internal/webhook"
)

func newDispatcherForTest(t *testing.T, store storage.Store, handler http.Handler) (*webhook.Dispatcher, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	d := webhook.New(store, webhook.WithWorkers(2), webhook.WithHTTPClient(srv.Client()))
	ctx, cancel := context.WithCancel(context.Background())
	d.Start(ctx)
	t.Cleanup(func() {
		cancel()
		d.Stop()
	})
	return d, srv
}

func seedWebhook(t *testing.T, store storage.Store, url string) *storage.Webhook {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	w := &storage.Webhook{
		ID: "wh_test", URL: url, Secret: "whsec_test",
		Events: `["user.created"]`, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := store.CreateWebhook(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestDispatcherDeliversOn2xx(t *testing.T) {
	store := testutil.NewTestDB(t)

	var got atomic.Value // *http.Request
	var bodyCh = make(chan []byte, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Store(r)
		b, _ := io.ReadAll(r.Body)
		bodyCh <- b
		w.WriteHeader(http.StatusOK)
	})

	d, srv := newDispatcherForTest(t, store, handler)
	seedWebhook(t, store, srv.URL)

	if err := d.Emit(context.Background(), "user.created", map[string]string{"id": "usr_1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case body := <-bodyCh:
		var env map[string]any
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatal(err)
		}
		if env["event"] != "user.created" {
			t.Errorf("event: %v", env["event"])
		}
		data, _ := env["data"].(map[string]any)
		if data["id"] != "usr_1" {
			t.Errorf("data.id: %v", data["id"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("delivery did not happen")
	}

	// Check headers + signature present.
	req := got.Load().(*http.Request)
	sig := req.Header.Get("X-Shark-Signature")
	if !strings.HasPrefix(sig, "t=") || !strings.Contains(sig, ",v1=") {
		t.Errorf("signature header malformed: %q", sig)
	}
	if req.Header.Get("X-Shark-Event") != "user.created" {
		t.Errorf("X-Shark-Event header missing")
	}

	// Delivery row should be status=delivered.
	waitFor(t, func() bool {
		dels, _ := store.ListWebhookDeliveriesByWebhookID(context.Background(), "wh_test", 10, "")
		return len(dels) == 1 && dels[0].Status == storage.WebhookStatusDelivered
	})
}

func TestDispatcherRetriesOn500(t *testing.T) {
	store := testutil.NewTestDB(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	_, srv := newDispatcherForTest(t, store, handler)
	seedWebhook(t, store, srv.URL)

	// Dispatcher instance already created by helper; Emit via the store-driven
	// path requires we reuse it. Grab it from context â€” pattern is to construct
	// separately here:
	d := webhook.New(store, webhook.WithWorkers(1), webhook.WithHTTPClient(srv.Client()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)
	defer d.Stop()

	if err := d.Emit(ctx, "user.created", map[string]string{"id": "usr_1"}); err != nil {
		t.Fatal(err)
	}

	waitFor(t, func() bool {
		dels, _ := store.ListWebhookDeliveriesByWebhookID(ctx, "wh_test", 10, "")
		if len(dels) == 0 {
			return false
		}
		// First failure â†’ status=retrying, attempt=1, next_retry_at set.
		return dels[0].Status == storage.WebhookStatusRetrying &&
			dels[0].Attempt == 1 && dels[0].NextRetryAt != nil
	})
}

func TestDispatcherSkipsDisabledWebhook(t *testing.T) {
	store := testutil.NewTestDB(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("disabled webhook should not receive deliveries")
	})
	_, srv := newDispatcherForTest(t, store, handler)

	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.CreateWebhook(context.Background(), &storage.Webhook{
		ID: "wh_off", URL: srv.URL, Secret: "s",
		Events: `["user.created"]`, Enabled: false,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	d := webhook.New(store, webhook.WithHTTPClient(srv.Client()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)
	defer d.Stop()

	if err := d.Emit(ctx, "user.created", nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	dels, _ := store.ListWebhookDeliveriesByWebhookID(ctx, "wh_off", 10, "")
	if len(dels) != 0 {
		t.Fatalf("expected 0 deliveries for disabled webhook, got %d", len(dels))
	}
}

func TestSignaturePayloadShape(t *testing.T) {
	// Capture the signing material the dispatcher uses by echoing the request.
	store := testutil.NewTestDB(t)
	var captured = make(chan struct {
		sig  string
		body string
	}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		captured <- struct {
			sig  string
			body string
		}{r.Header.Get("X-Shark-Signature"), string(b)}
		w.WriteHeader(http.StatusOK)
	})
	_, srv := newDispatcherForTest(t, store, handler)
	seedWebhook(t, store, srv.URL)

	d := webhook.New(store, webhook.WithHTTPClient(srv.Client()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)
	defer d.Stop()

	_ = d.Emit(ctx, "user.created", map[string]string{"id": "usr_42"})

	select {
	case c := <-captured:
		// Verify signature is `t=<int>,v1=<hex>`.
		parts := strings.Split(c.sig, ",")
		if len(parts) != 2 || !strings.HasPrefix(parts[0], "t=") || !strings.HasPrefix(parts[1], "v1=") {
			t.Fatalf("bad signature format: %q", c.sig)
		}
		// Body must contain the data we emitted.
		if !strings.Contains(c.body, `"usr_42"`) {
			t.Errorf("body missing data: %s", c.body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no delivery captured")
	}
}

// waitFor polls cond for up to 3s. Fails test if never true.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	end := time.Now().Add(3 * time.Second)
	for time.Now().Before(end) {
		if cond() {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("condition not met within 3s")
}
