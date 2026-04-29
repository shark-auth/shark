package api_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// seedAuditLog inserts a single audit log entry directly into the store.
func seedAuditLog(t *testing.T, store storage.Store, id, orgID, sessionID, actorID, action string) {
	t.Helper()
	ctx := context.Background()
	var orgPtr *string
	if orgID != "" {
		orgPtr = &orgID
	}
	var sessPtr *string
	if sessionID != "" {
		sessPtr = &sessionID
	}
	nid := id
	if nid == "" {
		nid, _ = gonanoid.New()
	}
	err := store.CreateAuditLog(ctx, &storage.AuditLog{
		ID:        nid,
		ActorID:   actorID,
		ActorType: "user",
		Action:    action,
		OrgID:     orgPtr,
		SessionID: sessPtr,
		Status:    "success",
		Metadata:  "{}",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("seedAuditLog: %v", err)
	}
}

// ---- TestAuditLogsFiltersByOrg ----

func TestAuditLogsFiltersByOrg(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// 2 logs in org_A, 1 in org_B
	seedAuditLog(t, ts.Store, "al_a1", "org_A", "", "usr_1", "user.login")
	seedAuditLog(t, ts.Store, "al_a2", "org_A", "", "usr_2", "user.logout")
	seedAuditLog(t, ts.Store, "al_b1", "org_B", "", "usr_3", "user.login")

	resp := ts.GetWithAdminKey("/api/v1/audit-logs?org_id=org_A")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	var body struct {
		Data    []map[string]interface{} `json:"data"`
		HasMore bool                     `json:"has_more"`
	}
	ts.DecodeJSON(resp, &body)

	if len(body.Data) != 2 {
		t.Errorf("expected 2 events for org_A, got %d", len(body.Data))
	}
	// Verify no org_B entry leaks through
	for _, e := range body.Data {
		if e["org_id"] == "org_B" {
			t.Errorf("org_B entry leaked into org_A results")
		}
	}
}

// ---- TestAuditLogsFiltersBySession ----

func TestAuditLogsFiltersBySession(t *testing.T) {
	ts := testutil.NewTestServer(t)

	seedAuditLog(t, ts.Store, "al_s1", "", "sess_X", "usr_1", "user.login")
	seedAuditLog(t, ts.Store, "al_s2", "", "sess_Y", "usr_2", "user.login")

	resp := ts.GetWithAdminKey("/api/v1/audit-logs?session_id=sess_X")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	var body struct {
		Data []map[string]interface{} `json:"data"`
	}
	ts.DecodeJSON(resp, &body)

	if len(body.Data) != 1 {
		t.Errorf("expected 1 event for sess_X, got %d", len(body.Data))
	}
	if len(body.Data) > 0 {
		sid, _ := body.Data[0]["session_id"].(string)
		if sid != "sess_X" {
			t.Errorf("session_id mismatch: got %q", sid)
		}
	}
}

// ---- TestAuditExportRequiresDates ----

func TestAuditExportRequiresDates(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// POST with empty body â†’ 400
	resp := ts.PostJSONWithAdminKey("/api/v1/audit-logs/export", map[string]string{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty export body, got %d", resp.StatusCode)
	}
	var errBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
		if errBody["error"] != "invalid_request" {
			t.Errorf("error key: got %q", errBody["error"])
		}
	}
}

// ---- TestAuditExportWithDates ----

func TestAuditExportWithDates(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Seed a log inside the export window
	seedAuditLog(t, ts.Store, "al_export1", "", "", "usr_export", "user.login")

	body := map[string]string{
		"from": "2026-04-01T00:00:00Z",
		"to":   "2026-04-30T23:59:59Z",
	}
	resp := ts.PostJSONWithAdminKey("/api/v1/audit-logs/export", body)
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("expected text/csv content-type, got %q", ct)
	}
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, ".csv") {
		t.Errorf("expected .csv in Content-Disposition, got %q", cd)
	}

	// Verify CSV has at least a header row
	scanner := bufio.NewScanner(resp.Body)
	defer resp.Body.Close()
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) < 1 {
		t.Fatal("CSV response has no lines")
	}
	if !strings.Contains(lines[0], "id") || !strings.Contains(lines[0], "action") {
		t.Errorf("CSV header row missing expected fields: %q", lines[0])
	}
}

// ---- TestAdminConfigIncludesDevMode ----

func TestAdminConfigIncludesDevMode(t *testing.T) {
	ts := testutil.NewTestServer(t)

	resp := ts.GetWithAdminKey("/api/v1/admin/config")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	var cfg map[string]interface{}
	ts.DecodeJSON(resp, &cfg)

	if _, ok := cfg["dev_mode"]; !ok {
		t.Error("admin/config response missing 'dev_mode' key")
	}
}
