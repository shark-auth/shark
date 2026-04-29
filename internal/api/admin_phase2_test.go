package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// ---- /admin/stats ----

func TestAdminStatsBasicCounts(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed 3 users, 2 with MFA enrolled AND verified. CountMFAEnabled
	// requires mfa_enabled=1 AND mfa_verified=1 (tightened in Wave 2 so
	// pending enrollments don't inflate adoption) â€” seed both flags.
	for i, mfa := range []bool{false, true, true} {
		u := &storage.User{
			ID: "usr_s" + string(rune('a'+i)), Email: "s" + string(rune('a'+i)) + "@x.io",
			HashType: "argon2id", Metadata: "{}", MFAEnabled: mfa, MFAVerified: mfa,
			CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
		}
		if err := ts.Store.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	// One active session + one expired.
	if err := ts.Store.CreateSession(ctx, &storage.Session{
		ID: "sess_active", UserID: "usr_sa", AuthMethod: "password",
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/stats")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Users struct {
			Total int `json:"total"`
		} `json:"users"`
		Sessions struct {
			Active int `json:"active"`
		} `json:"sessions"`
		MFA struct {
			Total       int     `json:"total"`
			Enabled     int     `json:"enabled"`
			AdoptionPct float64 `json:"adoption_pct"`
		} `json:"mfa"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Users.Total != 3 {
		t.Errorf("users.total: got %d", body.Users.Total)
	}
	if body.Sessions.Active != 1 {
		t.Errorf("sessions.active: got %d", body.Sessions.Active)
	}
	if body.MFA.Total != 3 {
		t.Errorf("mfa.total: got %d, want 3", body.MFA.Total)
	}
	if body.MFA.Enabled != 2 {
		t.Errorf("mfa.enabled: got %d", body.MFA.Enabled)
	}
	if body.MFA.AdoptionPct < 66 || body.MFA.AdoptionPct > 67 {
		t.Errorf("mfa.adoption_pct: got %.2f", body.MFA.AdoptionPct)
	}
}

func TestAdminStatsRequiresAdminKey(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.Get("/api/v1/admin/stats")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// ---- /admin/stats/trends ----

func TestAdminStatsTrendsFillsGaps(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// One signup today only.
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_today", Email: "today@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/stats/trends?days=7")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Days         int `json:"days"`
		SignupsByDay []struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		} `json:"signups_by_day"`
	}
	ts.DecodeJSON(resp, &body)
	if body.Days != 7 {
		t.Errorf("days: got %d", body.Days)
	}
	if len(body.SignupsByDay) != 7 {
		t.Fatalf("expected 7 filled buckets, got %d", len(body.SignupsByDay))
	}
	today := now.Format("2006-01-02")
	var totalToday int
	for _, b := range body.SignupsByDay {
		if b.Date == today {
			totalToday = b.Count
		}
	}
	if totalToday != 1 {
		t.Errorf("today bucket: got %d", totalToday)
	}
}

func TestAdminStatsTrendsShape(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed 2 signups across two different days.
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_ts1", Email: "ts1@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_ts2", Email: "ts2@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.AddDate(0, 0, -1).Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	// Seed a session with auth_method so auth_methods breakdown is non-empty.
	if err := ts.Store.CreateSession(ctx, &storage.Session{
		ID: "sess_ts", UserID: "usr_ts1", AuthMethod: "password",
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/stats/trends?days=14")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		Days         int `json:"days"`
		SignupsByDay []struct {
			Date  string `json:"date"`
			Count int    `json:"count"`
		} `json:"signups_by_day"`
		AuthMethods []struct {
			AuthMethod string `json:"auth_method"`
			Count      int    `json:"count"`
		} `json:"auth_methods"`
	}
	ts.DecodeJSON(resp, &body)

	// signups_by_day must be an array with 14 filled buckets.
	if len(body.SignupsByDay) != 14 {
		t.Errorf("signups_by_day: got %d buckets, want 14", len(body.SignupsByDay))
	}
	for _, b := range body.SignupsByDay {
		if b.Date == "" {
			t.Errorf("signups_by_day entry missing date field: %+v", b)
		}
	}

	// auth_methods must be an array of {auth_method, count} objects.
	if len(body.AuthMethods) == 0 {
		t.Errorf("auth_methods: expected at least one entry, got empty")
	}
	for _, m := range body.AuthMethods {
		if m.AuthMethod == "" {
			t.Errorf("auth_methods entry missing auth_method key: %+v", m)
		}
		if m.Count <= 0 {
			t.Errorf("auth_methods entry has non-positive count: %+v", m)
		}
	}
}

// ---- /admin/sessions ----

func TestAdminListSessionsCursorPagination(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_p", Email: "p@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if err := ts.Store.CreateSession(ctx, &storage.Session{
			ID: "sess_" + string(rune('A'+i)), UserID: "usr_p", AuthMethod: "password",
			ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			CreatedAt: now.Add(-time.Duration(i) * time.Second).Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/sessions?limit=2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var page1 struct {
		Data []struct {
			ID        string `json:"id"`
			UserEmail string `json:"user_email"`
		} `json:"data"`
		NextCursor string `json:"next_cursor"`
	}
	ts.DecodeJSON(resp, &page1)
	if len(page1.Data) != 2 {
		t.Fatalf("page1 len: %d", len(page1.Data))
	}
	if page1.NextCursor == "" {
		t.Fatal("expected next_cursor on full page")
	}
	if page1.Data[0].UserEmail != "p@x.io" {
		t.Errorf("user_email join missing: %q", page1.Data[0].UserEmail)
	}

	resp2 := ts.GetWithAdminKey("/api/v1/admin/sessions?limit=2&cursor=" + page1.NextCursor)
	var page2 struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	ts.DecodeJSON(resp2, &page2)
	if len(page2.Data) != 2 {
		t.Fatalf("page2 len: %d", len(page2.Data))
	}
	seen := map[string]bool{page1.Data[0].ID: true, page1.Data[1].ID: true}
	for _, s := range page2.Data {
		if seen[s.ID] {
			t.Fatalf("pagination overlap: %s", s.ID)
		}
	}
}

func TestAdminDeleteSessionRevokesAndAudits(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_d", Email: "d@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateSession(ctx, &storage.Session{
		ID: "sess_kill", UserID: "usr_d", AuthMethod: "password",
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	resp := ts.DeleteWithAdminKey("/api/v1/admin/sessions/sess_kill")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	if _, err := ts.Store.GetSessionByID(ctx, "sess_kill"); err == nil {
		t.Fatal("expected session gone")
	}

	audit, err := ts.Store.QueryAuditLogs(ctx, storage.AuditLogQuery{Action: "session.revoke", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(audit))
	}
	if audit[0].ActorType != "admin" || audit[0].TargetID != "sess_kill" {
		t.Errorf("audit: %+v", audit[0])
	}
}

func TestUsersSessionsRevokeAllGranularAudit(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_g", Email: "g@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"sess_x", "sess_y", "sess_z"} {
		if err := ts.Store.CreateSession(ctx, &storage.Session{
			ID: id, UserID: "usr_g", AuthMethod: "password",
			ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
			CreatedAt: now.Format(time.RFC3339),
		}); err != nil {
			t.Fatal(err)
		}
	}

	resp := ts.DeleteWithAdminKey("/api/v1/users/usr_g/sessions")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	left, _ := ts.Store.GetSessionsByUserID(ctx, "usr_g")
	if len(left) != 0 {
		t.Fatalf("expected 0 sessions left, got %d", len(left))
	}

	audit, err := ts.Store.QueryAuditLogs(ctx, storage.AuditLogQuery{Action: "session.revoke", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 3 {
		t.Fatalf("expected 3 granular audit entries, got %d", len(audit))
	}
}

// ---- /admin/dev/* ----

func TestDevInboxEndpointsRequireDevMode(t *testing.T) {
	ts := testutil.NewTestServer(t)
	// Default test config has email.provider="" (not "dev"), so dev inbox
	// handler should respond 404. Routes are always mounted; the gate is now
	// provider-based (W17 DB-backed config), not the legacy --dev flag.
	resp := ts.GetWithAdminKey("/api/v1/admin/dev/emails")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when email.provider != 'dev', got %d", resp.StatusCode)
	}
}

func TestDevInboxCaptureAndListing(t *testing.T) {
	ts := testutil.NewTestServerDev(t)

	// Trigger magic link send â€” uses the wired DevInboxSender.
	resp := ts.PostJSON("/api/v1/auth/magic-link/send", map[string]string{
		"email": "devuser@x.io",
	})
	resp.Body.Close()

	// List captured emails.
	listResp := ts.GetWithAdminKey("/api/v1/admin/dev/emails")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listResp.StatusCode)
	}
	var list struct {
		Data []struct {
			ID      string `json:"id"`
			To      string `json:"to"`
			Subject string `json:"subject"`
			HTML    string `json:"html"`
		} `json:"data"`
	}
	ts.DecodeJSON(listResp, &list)
	if len(list.Data) == 0 {
		t.Fatal("expected at least one captured email")
	}
	if list.Data[0].To != "devuser@x.io" {
		t.Errorf("to: %q", list.Data[0].To)
	}

	// Fetch single.
	getResp := ts.GetWithAdminKey("/api/v1/admin/dev/emails/" + list.Data[0].ID)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	// Clear inbox.
	delResp := ts.DeleteWithAdminKey("/api/v1/admin/dev/emails")
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}

	listResp2 := ts.GetWithAdminKey("/api/v1/admin/dev/emails")
	var list2 struct {
		Data []any `json:"data"`
	}
	ts.DecodeJSON(listResp2, &list2)
	if len(list2.Data) != 0 {
		t.Errorf("expected empty after delete, got %d", len(list2.Data))
	}
}

// ---- /auth/sessions (self-service) ----

func TestSelfSessionsListAndRevoke(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Login creates a session cookie via the shared http.Client cookie jar.
	if _, err := signupLoginFor(ts, "self@x.io", "Hunter2Hunter2"); err != nil {
		t.Fatal(err)
	}

	listResp := ts.Get("/api/v1/auth/sessions")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listResp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"data"`
	}
	ts.DecodeJSON(listResp, &body)
	if len(body.Data) == 0 {
		t.Fatal("expected at least current session")
	}
	var currentID string
	for _, s := range body.Data {
		if s.Current {
			currentID = s.ID
		}
	}
	if currentID == "" {
		t.Fatal("no current session flagged")
	}

	// Revoke current session.
	delResp := ts.Delete("/api/v1/auth/sessions/" + currentID)
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}

	// Next request should be unauthorized.
	resp := ts.Get("/api/v1/auth/sessions")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 after self-revoke, got %d", resp.StatusCode)
	}
}

func TestSelfCannotRevokeOthersSession(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Seed a session belonging to a different user.
	if err := ts.Store.CreateUser(ctx, &storage.User{
		ID: "usr_other", Email: "other@x.io", HashType: "argon2id", Metadata: "{}",
		CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	if err := ts.Store.CreateSession(ctx, &storage.Session{
		ID: "sess_others", UserID: "usr_other", AuthMethod: "password",
		ExpiresAt: now.Add(time.Hour).Format(time.RFC3339),
		CreatedAt: now.Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	// Login as a different user.
	if _, err := signupLoginFor(ts, "me@x.io", "Hunter2Hunter2"); err != nil {
		t.Fatal(err)
	}

	resp := ts.Delete("/api/v1/auth/sessions/sess_others")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign session, got %d", resp.StatusCode)
	}

	// Session must still exist.
	if _, err := ts.Store.GetSessionByID(ctx, "sess_others"); err != nil {
		t.Fatal("foreign session was revoked")
	}
}

// signupLoginFor performs signup + verify + login. Returns the user ID.
func signupLoginFor(ts *testutil.TestServer, email, password string) (string, error) {
	userID := ts.SignupAndVerify(email, password, "")
	resp := ts.PostJSON("/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	resp.Body.Close()
	return userID, nil
}
