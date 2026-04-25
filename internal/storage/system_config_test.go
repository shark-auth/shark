package storage_test

import (
	"context"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestSystemConfig_RoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Migration seeds an empty row — GetSystemConfig should return "" or "{}".
	got, err := store.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("GetSystemConfig (initial): %v", err)
	}
	if got != "{}" {
		t.Errorf("initial payload: want '{}', got %q", got)
	}

	// Write a config blob.
	type testCfg struct {
		Port string `json:"port"`
	}
	if err := store.SetSystemConfig(ctx, testCfg{Port: "9090"}); err != nil {
		t.Fatalf("SetSystemConfig: %v", err)
	}

	got, err = store.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("GetSystemConfig (after set): %v", err)
	}
	if got != `{"port":"9090"}` {
		t.Errorf("after set: got %q", got)
	}
}

func TestSystemConfig_IdempotentUpsert(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Multiple SetSystemConfig calls must not error (upsert idempotent).
	type cfg struct{ V int }
	for i := 0; i < 3; i++ {
		if err := store.SetSystemConfig(ctx, cfg{V: i}); err != nil {
			t.Fatalf("SetSystemConfig iteration %d: %v", i, err)
		}
	}

	got, err := store.GetSystemConfig(ctx)
	if err != nil {
		t.Fatalf("GetSystemConfig: %v", err)
	}
	if got != `{"V":2}` {
		t.Errorf("after 3 upserts want last value, got %q", got)
	}
}
