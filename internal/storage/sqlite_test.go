package storage_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

func TestUserCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	user := &storage.User{
		ID:        "usr_test1",
		Email:     "test@example.com",
		HashType:  "argon2id",
		Metadata:  "{}",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Create
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	// Get by ID
	got, err := store.GetUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Email != "test@example.com" {
		t.Fatalf("expected email %q, got %q", "test@example.com", got.Email)
	}

	// Get by email
	got, err = store.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail: %v", err)
	}
	if got.ID != "usr_test1" {
		t.Fatalf("expected ID %q, got %q", "usr_test1", got.ID)
	}

	// Update
	name := "Test User"
	got.Name = &name
	got.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := store.UpdateUser(ctx, got); err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	got, _ = store.GetUserByID(ctx, user.ID)
	if got.Name == nil || *got.Name != "Test User" {
		t.Fatalf("expected name %q, got %v", "Test User", got.Name)
	}

	// List
	users, err := store.ListUsers(ctx, storage.ListUsersOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	// Delete
	if err := store.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err = store.GetUserByID(ctx, user.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows after delete, got %v", err)
	}
}

func TestSessionCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Create user first
	user := &storage.User{
		ID:        "usr_sess",
		Email:     "sess@example.com",
		HashType:  "argon2id",
		Metadata:  "{}",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	store.CreateUser(ctx, user)

	sess := &storage.Session{
		ID:         "sess_1",
		UserID:     user.ID,
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
		AuthMethod: "password",
		ExpiresAt:  time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	// Create
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Get
	got, err := store.GetSessionByID(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetSessionByID: %v", err)
	}
	if got.UserID != user.ID {
		t.Fatalf("expected user_id %q, got %q", user.ID, got.UserID)
	}

	// List by user
	sessions, err := store.GetSessionsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetSessionsByUserID: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	// Update MFA
	if err := store.UpdateSessionMFAPassed(ctx, sess.ID, true); err != nil {
		t.Fatalf("UpdateSessionMFAPassed: %v", err)
	}
	got, _ = store.GetSessionByID(ctx, sess.ID)
	if !got.MFAPassed {
		t.Fatal("expected MFAPassed=true")
	}

	// Delete
	if err := store.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, err = store.GetSessionByID(ctx, sess.ID)
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows after delete, got %v", err)
	}
}

func TestAuditLogCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	log := &storage.AuditLog{
		ID:         "aud_1",
		ActorID:    "usr_1",
		ActorType:  "user",
		Action:     "login",
		TargetType: "session",
		TargetID:   "sess_1",
		IP:         "127.0.0.1",
		Metadata:   "{}",
		Status:     "success",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	if err := store.CreateAuditLog(ctx, log); err != nil {
		t.Fatalf("CreateAuditLog: %v", err)
	}

	got, err := store.GetAuditLogByID(ctx, log.ID)
	if err != nil {
		t.Fatalf("GetAuditLogByID: %v", err)
	}
	if got.Action != "login" {
		t.Fatalf("expected action %q, got %q", "login", got.Action)
	}

	// Query
	logs, err := store.QueryAuditLogs(ctx, storage.AuditLogQuery{Action: "login", Limit: 10})
	if err != nil {
		t.Fatalf("QueryAuditLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
}
