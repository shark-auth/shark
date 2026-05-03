package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/shark-auth/shark/internal/config"
	"github.com/shark-auth/shark/internal/testutil"
)

func TestFirstbootKeyServedOnceGlobally(t *testing.T) {
	dir := t.TempDir()
	ts := testutil.NewTestServerWithConfig(t, func(cfg *config.Config) {
		cfg.Storage.Path = filepath.Join(dir, "shark.db")
	})

	ctx := context.Background()
	keys, err := ts.Store.ListAPIKeys(ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	for _, k := range keys {
		if err := ts.Store.DeleteAPIKey(ctx, k.ID); err != nil {
			t.Fatalf("DeleteAPIKey(%s): %v", k.ID, err)
		}
	}
	testutil.CreateAPIKey(t, ts.Store, "default-admin")

	const want = "sk_live_firstboot_once"
	keyPath := filepath.Join(dir, "admin.key.firstboot")
	if err := os.WriteFile(keyPath, []byte(want+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resp := ts.Get("/api/v1/admin/firstboot/key")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first read: expected 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode first body: %v", err)
	}
	resp.Body.Close()
	if body["key"] != want {
		t.Fatalf("first read key mismatch: got %q want %q", body["key"], want)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("firstboot key file should be consumed, stat err=%v", err)
	}

	resp2 := ts.Get("/api/v1/admin/firstboot/key")
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("second read: expected 404, got %d", resp2.StatusCode)
	}
}
