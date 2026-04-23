package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

func seedUser(t *testing.T, store *storage.SQLiteStore, id, email string, createdAt time.Time, mfa bool) {
	t.Helper()
	ts := createdAt.UTC().Format(time.RFC3339)
	u := &storage.User{
		ID: id, Email: email, HashType: "argon2id", Metadata: "{}",
		CreatedAt: ts, UpdatedAt: ts, MFAEnabled: mfa, MFAVerified: mfa,
	}
	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("seed user %s: %v", id, err)
	}
}

func seedSession(t *testing.T, store *storage.SQLiteStore, id, userID, method string, createdAt, expiresAt time.Time, mfaPassed bool) {
	t.Helper()
	s := &storage.Session{
		ID: id, UserID: userID, IP: "127.0.0.1", UserAgent: "test",
		MFAPassed: mfaPassed, AuthMethod: method,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		CreatedAt: createdAt.UTC().Format(time.RFC3339),
	}
	if err := store.CreateSession(context.Background(), s); err != nil {
		t.Fatalf("seed session %s: %v", id, err)
	}
}

func seedAudit(t *testing.T, store *storage.SQLiteStore, id, action, status string, createdAt time.Time) {
	t.Helper()
	e := &storage.AuditLog{
		ID: id, ActorType: "user", Action: action, Status: status,
		Metadata: "{}", CreatedAt: createdAt.UTC().Format(time.RFC3339),
	}
	if err := store.CreateAuditLog(context.Background(), e); err != nil {
		t.Fatalf("seed audit %s: %v", id, err)
	}
}

func TestCountUsers(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now.AddDate(0, 0, -10), false)
	seedUser(t, store, "usr_2", "b@x.io", now.AddDate(0, 0, -2), true)
	seedUser(t, store, "usr_3", "c@x.io", now, true)

	total, err := store.CountUsers(ctx)
	if err != nil || total != 3 {
		t.Fatalf("CountUsers: total=%d err=%v", total, err)
	}

	recent, err := store.CountUsersCreatedSince(ctx, now.AddDate(0, 0, -7))
	if err != nil || recent != 2 {
		t.Fatalf("CountUsersCreatedSince: got %d err=%v", recent, err)
	}

	mfa, err := store.CountMFAEnabled(ctx)
	if err != nil || mfa != 2 {
		t.Fatalf("CountMFAEnabled: got %d err=%v", mfa, err)
	}
}

func TestCountActiveSessionsExcludesExpired(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now, false)
	seedSession(t, store, "sess_active", "usr_1", "password", now, now.Add(time.Hour), true)
	seedSession(t, store, "sess_expired", "usr_1", "password", now.Add(-2*time.Hour), now.Add(-time.Hour), true)

	n, err := store.CountActiveSessions(ctx)
	if err != nil || n != 1 {
		t.Fatalf("CountActiveSessions: got %d err=%v", n, err)
	}
}

func TestCountFailedLoginsSince(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedAudit(t, store, "aud_1", "user.login", "failure", now.Add(-30*time.Minute))
	seedAudit(t, store, "aud_2", "user.login", "failure", now.Add(-2*time.Hour))
	seedAudit(t, store, "aud_3", "user.login", "success", now)
	seedAudit(t, store, "aud_4", "signup", "failure", now)

	n, err := store.CountFailedLoginsSince(ctx, now.Add(-time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("CountFailedLoginsSince: got %d err=%v", n, err)
	}
}

func TestCountExpiringAPIKeys(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	mk := func(id string, expiresAt *time.Time, revoked bool) {
		k := &storage.APIKey{
			ID: id, Name: id, KeyHash: id + "_h", KeyPrefix: id + "_p", KeySuffix: id + "_s",
			Scopes: `["users:read"]`, RateLimit: 100,
			CreatedAt: now.Format(time.RFC3339),
		}
		if expiresAt != nil {
			s := expiresAt.Format(time.RFC3339)
			k.ExpiresAt = &s
		}
		if revoked {
			r := now.Format(time.RFC3339)
			k.RevokedAt = &r
		}
		if err := store.CreateAPIKey(ctx, k); err != nil {
			t.Fatalf("CreateAPIKey %s: %v", id, err)
		}
	}

	soon := now.Add(3 * 24 * time.Hour)
	far := now.Add(365 * 24 * time.Hour)
	past := now.Add(-time.Hour)
	mk("key_soon", &soon, false)
	mk("key_far", &far, false)
	mk("key_none", nil, false)             // no expiry
	mk("key_past", &past, false)           // already expired, not counted
	mk("key_revoked_soon", &soon, true)    // revoked, not counted

	n, err := store.CountExpiringAPIKeys(ctx, 7*24*time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("CountExpiringAPIKeys: got %d err=%v", n, err)
	}
}

func TestCountSSOConnections(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Format(time.RFC3339)
	mk := func(id string, enabled bool) {
		c := &storage.SSOConnection{
			ID: id, Type: "oidc", Name: id, Enabled: enabled,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := store.CreateSSOConnection(ctx, c); err != nil {
			t.Fatalf("CreateSSOConnection %s: %v", id, err)
		}
	}
	mk("sso_1", true)
	mk("sso_2", true)
	mk("sso_3", false)

	total, err := store.CountSSOConnections(ctx, false)
	if err != nil || total != 3 {
		t.Fatalf("CountSSOConnections all: got %d err=%v", total, err)
	}
	enabled, err := store.CountSSOConnections(ctx, true)
	if err != nil || enabled != 2 {
		t.Fatalf("CountSSOConnections enabled: got %d err=%v", enabled, err)
	}
}

func TestGroupSessionsByAuthMethod(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now, false)
	seedSession(t, store, "s1", "usr_1", "password", now, now.Add(time.Hour), true)
	seedSession(t, store, "s2", "usr_1", "password", now, now.Add(time.Hour), true)
	seedSession(t, store, "s3", "usr_1", "google", now, now.Add(time.Hour), true)
	seedSession(t, store, "s4", "usr_1", "passkey", now.AddDate(0, 0, -40), now.Add(time.Hour), true)

	got, err := store.GroupSessionsByAuthMethodSince(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("GroupSessionsByAuthMethodSince: %v", err)
	}
	m := map[string]int{}
	for _, r := range got {
		m[r.AuthMethod] = r.Count
	}
	if m["password"] != 2 || m["google"] != 1 || m["passkey"] != 0 {
		t.Fatalf("unexpected breakdown: %v", m)
	}
}

func TestGroupUsersCreatedByDay(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now, false)
	seedUser(t, store, "usr_2", "b@x.io", now, false)
	seedUser(t, store, "usr_3", "c@x.io", now.AddDate(0, 0, -1), false)

	got, err := store.GroupUsersCreatedByDay(ctx, 30)
	if err != nil {
		t.Fatalf("GroupUsersCreatedByDay: %v", err)
	}
	m := map[string]int{}
	for _, r := range got {
		m[r.Date] = r.Count
	}
	today := now.Format("2006-01-02")
	yday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if m[today] != 2 || m[yday] != 1 {
		t.Fatalf("unexpected day counts: %v", m)
	}
}

func TestListActiveSessionsCursorPagination(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "alice@x.io", now, false)
	seedUser(t, store, "usr_2", "bob@x.io", now, false)

	// 5 active sessions staggered by second so keyset pagination is deterministic.
	for i := 0; i < 5; i++ {
		uid := "usr_1"
		if i%2 == 0 {
			uid = "usr_2"
		}
		seedSession(t, store, "sess_"+string(rune('A'+i)), uid, "password",
			now.Add(-time.Duration(i)*time.Second), now.Add(time.Hour), true)
	}

	// Page 1: first 2 most recent.
	page1, err := store.ListActiveSessions(ctx, storage.ListSessionsOpts{Limit: 2})
	if err != nil || len(page1) != 2 {
		t.Fatalf("page1: len=%d err=%v", len(page1), err)
	}
	if page1[0].UserEmail == "" {
		t.Fatalf("expected joined user_email, got empty")
	}

	// Page 2 via keyset cursor.
	cursor := page1[len(page1)-1].CreatedAt + "|" + page1[len(page1)-1].ID
	page2, err := store.ListActiveSessions(ctx, storage.ListSessionsOpts{Limit: 2, Cursor: cursor})
	if err != nil || len(page2) != 2 {
		t.Fatalf("page2: len=%d err=%v", len(page2), err)
	}
	// No overlap.
	for _, a := range page1 {
		for _, b := range page2 {
			if a.ID == b.ID {
				t.Fatalf("pagination overlap: %s", a.ID)
			}
		}
	}

	// Filter by user.
	filtered, err := store.ListActiveSessions(ctx, storage.ListSessionsOpts{UserID: "usr_1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range filtered {
		if s.UserID != "usr_1" {
			t.Fatalf("filter leaked: %s", s.UserID)
		}
	}

	// Filter by auth method.
	m, err := store.ListActiveSessions(ctx, storage.ListSessionsOpts{AuthMethod: "password"})
	if err != nil || len(m) != 5 {
		t.Fatalf("method filter: len=%d err=%v", len(m), err)
	}
}

func TestListActiveSessionsExcludesExpired(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now, false)
	seedSession(t, store, "sess_a", "usr_1", "password", now, now.Add(time.Hour), true)
	seedSession(t, store, "sess_b", "usr_1", "password", now, now.Add(-time.Hour), true)

	got, err := store.ListActiveSessions(ctx, storage.ListSessionsOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "sess_a" {
		t.Fatalf("expected only sess_a, got %+v", got)
	}
}

func TestDeleteSessionsByUserID(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seedUser(t, store, "usr_1", "a@x.io", now, false)
	seedUser(t, store, "usr_2", "b@x.io", now, false)
	seedSession(t, store, "sess_1", "usr_1", "password", now, now.Add(time.Hour), true)
	seedSession(t, store, "sess_2", "usr_1", "google", now, now.Add(time.Hour), true)
	seedSession(t, store, "sess_3", "usr_2", "password", now, now.Add(time.Hour), true)

	ids, err := store.DeleteSessionsByUserID(ctx, "usr_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 deleted IDs, got %v", ids)
	}

	left, err := store.GetSessionsByUserID(ctx, "usr_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("expected 0 sessions left for usr_1, got %d", len(left))
	}
	other, _ := store.GetSessionsByUserID(ctx, "usr_2")
	if len(other) != 1 {
		t.Fatalf("usr_2 sessions clobbered: %d", len(other))
	}
}

func TestDevEmailCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		e := &storage.DevEmail{
			ID: "de_" + string(rune('A'+i)), To: "u@x.io",
			Subject: "hi", HTML: "<p>hi</p>", Text: "hi",
			CreatedAt: now.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
		}
		if err := store.CreateDevEmail(ctx, e); err != nil {
			t.Fatalf("CreateDevEmail: %v", err)
		}
	}

	all, err := store.ListDevEmails(ctx, 10)
	if err != nil || len(all) != 3 {
		t.Fatalf("ListDevEmails: len=%d err=%v", len(all), err)
	}
	// Newest first.
	if all[0].ID != "de_C" {
		t.Fatalf("expected newest de_C first, got %s", all[0].ID)
	}

	got, err := store.GetDevEmail(ctx, "de_B")
	if err != nil || got.Subject != "hi" {
		t.Fatalf("GetDevEmail: %+v err=%v", got, err)
	}

	if err := store.DeleteAllDevEmails(ctx); err != nil {
		t.Fatal(err)
	}
	all, _ = store.ListDevEmails(ctx, 10)
	if len(all) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(all))
	}
}
