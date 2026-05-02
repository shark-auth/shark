package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shark-auth/shark/cmd/shark/migrations"
	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/storage"
)

func newAdminMiddlewareStore(t *testing.T) *storage.SQLiteStore {
	t.Helper()
	store, err := storage.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	if err := storage.RunMigrations(store.DB(), migrations.FS, "."); err != nil {
		store.Close() //nolint:errcheck
		t.Fatalf("RunMigrations: %v", err)
	}
	t.Cleanup(func() { store.Close() }) //nolint:errcheck
	return store
}

func seedAdminMiddlewareKey(t *testing.T, store storage.Store) string {
	t.Helper()
	rawKey := "sk" + "_live_adminmiddleware000000000000"
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.CreateAPIKey(context.Background(), &storage.APIKey{
		ID:        "key_admin_middleware",
		Name:      "admin",
		KeyHash:   auth.HashAPIKey(rawKey),
		KeyPrefix: "sk_live_",
		KeySuffix: "0000",
		Scopes:    `["*"]`,
		RateLimit: 1000,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	return rawKey
}

func TestAdminAPIKeyFromStoreRejectsQueryTokenOnNormalRoutes(t *testing.T) {
	store := newAdminMiddlewareStore(t)
	key := seedAdminMiddlewareKey(t, store)
	handler := AdminAPIKeyFromStore(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/health?token="+key, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rr.Code)
	}
}

func TestAdminAPIKeyFromStoreAllowsQueryTokenOnSSERoutes(t *testing.T) {
	store := newAdminMiddlewareStore(t)
	key := seedAdminMiddlewareKey(t, store)
	handler := AdminAPIKeyFromStore(store, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/logs/stream?token="+key, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", rr.Code)
	}
}
