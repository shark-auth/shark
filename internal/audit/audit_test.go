package audit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/audit"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

func TestAuditLogCreate(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	event := &storage.AuditLog{
		ActorID:    "usr_abc123",
		ActorType:  "user",
		Action:     "user.login",
		TargetType: "user",
		TargetID:   "usr_abc123",
		IP:         "192.168.1.1",
		UserAgent:  "TestClient/1.0",
		Status:     "success",
	}

	err := logger.Log(ctx, event)
	if err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify ID was assigned
	if event.ID == "" {
		t.Fatal("expected ID to be assigned")
	}
	if event.ID[:4] != "aud_" {
		t.Fatalf("expected ID to start with aud_, got %s", event.ID)
	}

	// Verify CreatedAt was assigned
	if event.CreatedAt == "" {
		t.Fatal("expected CreatedAt to be assigned")
	}

	// Query it back
	fetched, err := logger.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("failed to get event by ID: %v", err)
	}

	if fetched.Action != "user.login" {
		t.Fatalf("expected action user.login, got %s", fetched.Action)
	}
	if fetched.ActorID != "usr_abc123" {
		t.Fatalf("expected actor_id usr_abc123, got %s", fetched.ActorID)
	}
	if fetched.IP != "192.168.1.1" {
		t.Fatalf("expected IP 192.168.1.1, got %s", fetched.IP)
	}
	if fetched.Status != "success" {
		t.Fatalf("expected status success, got %s", fetched.Status)
	}
}

func TestAuditLogDefaults(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	// Log with minimal fields â€” defaults should fill in the rest
	event := &storage.AuditLog{
		Action: "user.signup",
	}
	err := logger.Log(ctx, event)
	if err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	fetched, err := logger.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	if fetched.ActorType != "user" {
		t.Fatalf("expected default actor_type 'user', got %s", fetched.ActorType)
	}
	if fetched.Status != "success" {
		t.Fatalf("expected default status 'success', got %s", fetched.Status)
	}
	if fetched.Metadata != "{}" {
		t.Fatalf("expected default metadata '{}', got %s", fetched.Metadata)
	}
}

func TestAuditLogQuery(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create events with varying actions, actors, and timestamps
	events := []storage.AuditLog{
		{Action: "user.login", ActorID: "usr_1", TargetType: "user", TargetID: "usr_1", Status: "success", IP: "10.0.0.1", CreatedAt: now.Add(-5 * time.Minute).Format(time.RFC3339)},
		{Action: "user.signup", ActorID: "usr_2", TargetType: "user", TargetID: "usr_2", Status: "success", IP: "10.0.0.2", CreatedAt: now.Add(-4 * time.Minute).Format(time.RFC3339)},
		{Action: "user.login", ActorID: "usr_1", TargetType: "user", TargetID: "usr_1", Status: "failure", IP: "10.0.0.1", CreatedAt: now.Add(-3 * time.Minute).Format(time.RFC3339)},
		{Action: "mfa.enabled", ActorID: "usr_1", TargetType: "user", TargetID: "usr_1", Status: "success", IP: "10.0.0.1", CreatedAt: now.Add(-2 * time.Minute).Format(time.RFC3339)},
		{Action: "user.login", ActorID: "usr_3", TargetType: "user", TargetID: "usr_3", Status: "success", IP: "10.0.0.3", CreatedAt: now.Add(-1 * time.Minute).Format(time.RFC3339)},
	}

	for i := range events {
		if err := logger.Log(ctx, &events[i]); err != nil {
			t.Fatalf("failed to log event %d: %v", i, err)
		}
	}

	t.Run("filter by action", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{Action: "user.login"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if len(logs) != 3 {
			t.Fatalf("expected 3 user.login events, got %d", len(logs))
		}
	})

	t.Run("filter by comma-separated actions", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{Action: "user.login,user.signup"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if len(logs) != 4 {
			t.Fatalf("expected 4 events for user.login,user.signup, got %d", len(logs))
		}
	})

	t.Run("filter by actor", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{ActorID: "usr_1"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if len(logs) != 3 {
			t.Fatalf("expected 3 events for usr_1, got %d", len(logs))
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{Status: "failure"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if len(logs) != 1 {
			t.Fatalf("expected 1 failure event, got %d", len(logs))
		}
	})

	t.Run("filter by IP", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{IP: "10.0.0.1"})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		if len(logs) != 3 {
			t.Fatalf("expected 3 events from 10.0.0.1, got %d", len(logs))
		}
	})

	t.Run("filter by date range", func(t *testing.T) {
		from := now.Add(-4 * time.Minute).Format(time.RFC3339)
		to := now.Add(-2 * time.Minute).Format(time.RFC3339)
		logs, err := logger.Query(ctx, storage.AuditLogQuery{From: from, To: to})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		// Events at -4m, -3m, -2m should match
		if len(logs) != 3 {
			t.Fatalf("expected 3 events in date range, got %d", len(logs))
		}
	})

	t.Run("results ordered by created_at DESC", func(t *testing.T) {
		logs, err := logger.Query(ctx, storage.AuditLogQuery{Limit: 50})
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}
		for i := 1; i < len(logs); i++ {
			if logs[i].CreatedAt > logs[i-1].CreatedAt {
				t.Fatalf("results not ordered DESC: %s > %s at index %d", logs[i].CreatedAt, logs[i-1].CreatedAt, i)
			}
		}
	})
}

func TestAuditLogCursorPagination(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	now := time.Now().UTC()

	// Insert 10 events with distinct timestamps
	for i := 0; i < 10; i++ {
		event := &storage.AuditLog{
			Action:    "user.login",
			ActorID:   fmt.Sprintf("usr_%d", i),
			Status:    "success",
			CreatedAt: now.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
		}
		if err := logger.Log(ctx, event); err != nil {
			t.Fatalf("failed to log event %d: %v", i, err)
		}
	}

	// First page: limit 3
	page1, err := logger.Query(ctx, storage.AuditLogQuery{Limit: 3})
	if err != nil {
		t.Fatalf("page1 query failed: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 results on page 1, got %d", len(page1))
	}

	// Verify ordered DESC (most recent first)
	if page1[0].ActorID != "usr_9" {
		t.Fatalf("expected first result to be usr_9 (most recent), got %s", page1[0].ActorID)
	}

	// Second page: use cursor from last item of page 1
	cursor := page1[len(page1)-1].ID
	page2, err := logger.Query(ctx, storage.AuditLogQuery{Limit: 3, Cursor: cursor})
	if err != nil {
		t.Fatalf("page2 query failed: %v", err)
	}
	if len(page2) != 3 {
		t.Fatalf("expected 3 results on page 2, got %d", len(page2))
	}

	// Verify no overlap between pages
	page1IDs := make(map[string]bool)
	for _, l := range page1 {
		page1IDs[l.ID] = true
	}
	for _, l := range page2 {
		if page1IDs[l.ID] {
			t.Fatalf("page2 contains duplicate ID from page1: %s", l.ID)
		}
	}

	// Third page
	cursor2 := page2[len(page2)-1].ID
	page3, err := logger.Query(ctx, storage.AuditLogQuery{Limit: 3, Cursor: cursor2})
	if err != nil {
		t.Fatalf("page3 query failed: %v", err)
	}
	if len(page3) != 3 {
		t.Fatalf("expected 3 results on page 3, got %d", len(page3))
	}

	// Fourth page: should have 1 remaining
	cursor3 := page3[len(page3)-1].ID
	page4, err := logger.Query(ctx, storage.AuditLogQuery{Limit: 3, Cursor: cursor3})
	if err != nil {
		t.Fatalf("page4 query failed: %v", err)
	}
	if len(page4) != 1 {
		t.Fatalf("expected 1 result on page 4, got %d", len(page4))
	}

	// Verify all 10 unique IDs were returned across all pages
	allIDs := make(map[string]bool)
	for _, page := range [][]*storage.AuditLog{page1, page2, page3, page4} {
		for _, l := range page {
			allIDs[l.ID] = true
		}
	}
	if len(allIDs) != 10 {
		t.Fatalf("expected 10 unique IDs across all pages, got %d", len(allIDs))
	}
}

func TestAuditLogRetention(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	// Create an event with a timestamp in the past
	past := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	event := &storage.AuditLog{
		Action:    "user.login",
		ActorID:   "usr_old",
		Status:    "success",
		CreatedAt: past,
	}
	if err := logger.Log(ctx, event); err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify the event exists
	fetched, err := logger.GetByID(ctx, event.ID)
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}
	if fetched.ID != event.ID {
		t.Fatal("event should exist before cleanup")
	}

	// Delete with a cutoff of 1 hour ago (event is 48h old, should be deleted)
	cutoff := time.Now().UTC().Add(-1 * time.Hour)
	deleted, err := logger.DeleteBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	// Verify the event is gone
	_, err = logger.GetByID(ctx, event.ID)
	if err == nil {
		t.Fatal("expected event to be deleted after cleanup")
	}
}

func TestAuditLogRetentionKeepsRecent(t *testing.T) {
	store := testutil.NewTestDB(t)
	logger := audit.NewLogger(store)
	ctx := context.Background()

	// Create a recent event
	recent := &storage.AuditLog{
		Action:  "user.login",
		ActorID: "usr_recent",
		Status:  "success",
	}
	if err := logger.Log(ctx, recent); err != nil {
		t.Fatalf("failed to log recent event: %v", err)
	}

	// Create an old event
	old := &storage.AuditLog{
		Action:    "user.login",
		ActorID:   "usr_old",
		Status:    "success",
		CreatedAt: time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339),
	}
	if err := logger.Log(ctx, old); err != nil {
		t.Fatalf("failed to log old event: %v", err)
	}

	// Delete with 24h cutoff
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	deleted, err := logger.DeleteBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}

	// Verify the recent event still exists
	fetched, err := logger.GetByID(ctx, recent.ID)
	if err != nil {
		t.Fatalf("recent event should still exist: %v", err)
	}
	if fetched.ActorID != "usr_recent" {
		t.Fatalf("expected usr_recent, got %s", fetched.ActorID)
	}

	// Verify the old event is gone
	_, err = logger.GetByID(ctx, old.ID)
	if err == nil {
		t.Fatal("old event should be deleted after cleanup")
	}
}
