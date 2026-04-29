package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func seedWebhook(t *testing.T, store *storage.SQLiteStore, id, url, events string, enabled bool) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if err := store.CreateWebhook(context.Background(), &storage.Webhook{
		ID: id, URL: url, Secret: "secret_" + id, Events: events, Enabled: enabled,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("CreateWebhook %s: %v", id, err)
	}
}

func TestWebhookCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	seedWebhook(t, store, "wh_1", "https://example.com/hook", `["user.created"]`, true)

	got, err := store.GetWebhookByID(ctx, "wh_1")
	if err != nil || got.URL != "https://example.com/hook" {
		t.Fatalf("get: %+v err=%v", got, err)
	}

	got.Enabled = false
	got.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := store.UpdateWebhook(ctx, got); err != nil {
		t.Fatal(err)
	}
	after, _ := store.GetWebhookByID(ctx, "wh_1")
	if after.Enabled {
		t.Fatal("update did not persist enabled=false")
	}

	list, err := store.ListWebhooks(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %d err=%v", len(list), err)
	}

	if err := store.DeleteWebhook(ctx, "wh_1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetWebhookByID(ctx, "wh_1"); err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func TestListEnabledWebhooksByEvent(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	seedWebhook(t, store, "wh_user", "https://a.example/hook", `["user.created","user.deleted"]`, true)
	seedWebhook(t, store, "wh_sess", "https://b.example/hook", `["session.revoked"]`, true)
	seedWebhook(t, store, "wh_disabled", "https://c.example/hook", `["user.created"]`, false)
	// Ensure prefix-false-positive guard works: "user.created" search must not match "user.created_v2".
	seedWebhook(t, store, "wh_v2", "https://d.example/hook", `["user.created_v2"]`, true)

	matches, err := store.ListEnabledWebhooksByEvent(ctx, "user.created")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != "wh_user" {
		t.Fatalf("expected only wh_user, got %+v", matches)
	}
}

func TestWebhookDeliveryLifecycleAndPagination(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	seedWebhook(t, store, "wh_p", "https://p.example", `["user.created"]`, true)

	now := time.Now().UTC()
	// 5 deliveries staggered so keyset pagination is deterministic.
	for i := 0; i < 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339)
		d := &storage.WebhookDelivery{
			ID: "whd_" + string(rune('A'+i)), WebhookID: "wh_p",
			Event: "user.created", Payload: `{"id":"usr_x"}`,
			Status: storage.WebhookStatusDelivered, Attempt: 1,
			CreatedAt: ts, UpdatedAt: ts,
		}
		if err := store.CreateWebhookDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	page1, err := store.ListWebhookDeliveriesByWebhookID(ctx, "wh_p", 2, "")
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1: %d err=%v", len(page1), err)
	}
	cursor := page1[len(page1)-1].CreatedAt + "|" + page1[len(page1)-1].ID
	page2, err := store.ListWebhookDeliveriesByWebhookID(ctx, "wh_p", 2, cursor)
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2: %d err=%v", len(page2), err)
	}
	for _, a := range page1 {
		for _, b := range page2 {
			if a.ID == b.ID {
				t.Fatalf("pagination overlap: %s", a.ID)
			}
		}
	}
}

func TestListPendingWebhookDeliveries(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	seedWebhook(t, store, "wh_r", "https://r.example", `["user.created"]`, true)

	now := time.Now().UTC()
	past := now.Add(-time.Minute).Format(time.RFC3339)
	future := now.Add(time.Hour).Format(time.RFC3339)
	mk := func(id, status, nextRetryAt string) {
		var nra *string
		if nextRetryAt != "" {
			nra = &nextRetryAt
		}
		d := &storage.WebhookDelivery{
			ID: id, WebhookID: "wh_r", Event: "user.created",
			Payload: `{}`, Status: status, Attempt: 2, NextRetryAt: nra,
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		}
		if err := store.CreateWebhookDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	mk("whd_due", storage.WebhookStatusRetrying, past)
	mk("whd_not_due", storage.WebhookStatusRetrying, future)
	mk("whd_delivered", storage.WebhookStatusDelivered, past) // wrong status
	mk("whd_failed", storage.WebhookStatusFailed, past)       // terminal â€” skip

	pending, err := store.ListPendingWebhookDeliveries(ctx, now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != "whd_due" {
		t.Fatalf("expected only whd_due, got %+v", pending)
	}
}

func TestDeleteWebhookDeliveriesBefore(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	seedWebhook(t, store, "wh_c", "https://c.example", `["user.created"]`, true)

	mk := func(id string, at time.Time) {
		d := &storage.WebhookDelivery{
			ID: id, WebhookID: "wh_c", Event: "user.created",
			Payload: `{}`, Status: storage.WebhookStatusDelivered, Attempt: 1,
			CreatedAt: at.Format(time.RFC3339),
			UpdatedAt: at.Format(time.RFC3339),
		}
		if err := store.CreateWebhookDelivery(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	mk("whd_old", now.AddDate(0, 0, -100))
	mk("whd_borderline", now.AddDate(0, 0, -90))
	mk("whd_recent", now.AddDate(0, 0, -1))

	cutoff := now.AddDate(0, 0, -90)
	n, err := store.DeleteWebhookDeliveriesBefore(ctx, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}
}
