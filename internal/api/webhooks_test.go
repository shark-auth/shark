package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/api"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
	"github.com/shark-auth/shark/internal/webhook"
)

// testServerWithDispatcher rebuilds a test server with a webhook dispatcher
// wired to the provided HTTP test backend. The default testutil.NewTestServer
// doesn't mount the dispatcher since most tests don't need outbound HTTP.
func testServerWithDispatcher(t *testing.T, backend http.Handler) *testutil.TestServer {
	t.Helper()
	ts := testutil.NewTestServer(t)
	// Spin a backend that webhooks will POST to.
	upstream := httptest.NewServer(backend)
	t.Cleanup(upstream.Close)
	t.Setenv("SHARK_TEST_UPSTREAM", upstream.URL)

	d := webhook.New(ts.Store,
		webhook.WithWorkers(2),
		webhook.WithHTTPClient(upstream.Client()),
	)
	d.Start(t.Context())
	t.Cleanup(d.Stop)

	// Swap the dispatcher on the existing API server â€” mirrors what
	// server.Build does at startup.
	api.WithWebhookDispatcher(d)(ts.APIServer)
	return ts
}

func TestCreateWebhookReturnsSecretOnceOnly(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    "https://example.com/hook",
		"events": []string{"user.created"},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		ID     string `json:"id"`
		Secret string `json:"secret"`
	}
	ts.DecodeJSON(resp, &body)
	if !strings.HasPrefix(body.Secret, "whsec_") {
		t.Fatalf("bad secret: %q", body.Secret)
	}

	getResp := ts.GetWithAdminKey("/api/v1/webhooks/" + body.ID)
	raw := testutil.DecodeJSONResponse[map[string]any](t, getResp)
	if _, ok := raw["secret"]; ok {
		t.Fatal("secret must not be returned on subsequent reads")
	}
}

func TestCreateWebhookRejectsBadURLAndUnknownEvent(t *testing.T) {
	ts := testutil.NewTestServer(t)

	bad := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    "ftp://example.com/hook",
		"events": []string{"user.created"},
	})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad url: %d", bad.StatusCode)
	}

	unk := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    "https://example.com/hook",
		"events": []string{"user.invented"},
	})
	if unk.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown event: %d", unk.StatusCode)
	}
}

func TestWebhookTestEndpointFires(t *testing.T) {
	var hits atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("X-Shark-Event"), "webhook.test") {
			hits.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	})
	ts := testServerWithDispatcher(t, handler)
	upstream := ts.T.Name() // unused alias to pretend
	_ = upstream

	// Create webhook pointing at our upstream (stored via Setenv).
	createResp := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    os.Getenv("SHARK_TEST_UPSTREAM"),
		"events": []string{"user.created"}, // webhook.test is always fan-out via /test
	})
	var created struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &created)

	// Trigger test delivery.
	testResp := ts.PostJSONWithAdminKey("/api/v1/webhooks/"+created.ID+"/test", nil)
	if testResp.StatusCode != http.StatusAccepted {
		t.Fatalf("test endpoint: %d", testResp.StatusCode)
	}

	waitForAPI(t, func() bool { return hits.Load() >= 1 })
}

func TestWebhookDeliveriesListPaginated(t *testing.T) {
	ts := testServerWithDispatcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	createResp := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    os.Getenv("SHARK_TEST_UPSTREAM"),
		"events": []string{"user.created"},
	})
	var created struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(createResp, &created)

	// Seed 3 deliveries directly in storage (tight control over ordering).
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		d := &storage.WebhookDelivery{
			ID: "whd_seed_" + string(rune('a'+i)),
			WebhookID: created.ID, Event: "user.created",
			Payload: `{"id":"x"}`,
			Status:  storage.WebhookStatusDelivered,
			Attempt: 1,
			CreatedAt: now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		}
		if err := ts.Store.CreateWebhookDelivery(t.Context(), d); err != nil {
			t.Fatal(err)
		}
	}

	listResp := ts.GetWithAdminKey("/api/v1/webhooks/" + created.ID + "/deliveries?limit=2")
	var body struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"next_cursor"`
	}
	ts.DecodeJSON(listResp, &body)
	if len(body.Data) != 2 {
		t.Fatalf("expected 2, got %d", len(body.Data))
	}
	if body.NextCursor == "" {
		t.Fatal("expected cursor")
	}
}

// TestSignupEmitsUserCreatedWebhook verifies the emission hook in the signup
// handler fans out to a registered webhook.
func TestSignupEmitsUserCreatedWebhook(t *testing.T) {
	var hit atomic.Int32
	var capturedEvent atomic.Value // string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEvent.Store(r.Header.Get("X-Shark-Event"))
		hit.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	ts := testServerWithDispatcher(t, handler)

	createResp := ts.PostJSONWithAdminKey("/api/v1/webhooks", map[string]any{
		"url":    os.Getenv("SHARK_TEST_UPSTREAM"),
		"events": []string{"user.created"},
	})
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create webhook: %d", createResp.StatusCode)
	}

	signupResp := ts.PostJSON("/api/v1/auth/signup", map[string]string{
		"email": "webhooked@x.io", "password": "Hunter2Hunter2",
	})
	signupResp.Body.Close()
	if signupResp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: %d", signupResp.StatusCode)
	}

	waitForAPI(t, func() bool { return hit.Load() >= 1 })
	if got := capturedEvent.Load().(string); got != "user.created" {
		t.Errorf("X-Shark-Event: %q", got)
	}
}

// waitForAPI polls for up to 3s.
func waitForAPI(t *testing.T, cond func() bool) {
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
