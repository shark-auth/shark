package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/testutil"
)

// helper — seed a grant directly via the store (bypasses POST handler) to
// keep tests focused on list/filter/revoke surface.
func seedGrant(t *testing.T, store storage.Store, id, fromID, toID string, revoked bool) {
	t.Helper()
	g := &storage.MayActGrant{
		ID:      id,
		FromID:  fromID,
		ToID:    toID,
		MaxHops: 2,
		Scopes:  []string{"read"},
	}
	if err := store.CreateMayActGrant(context.Background(), g); err != nil {
		t.Fatalf("seed grant %s: %v", id, err)
	}
	if revoked {
		if err := store.RevokeMayActGrant(context.Background(), id, time.Now().UTC()); err != nil {
			t.Fatalf("revoke grant %s: %v", id, err)
		}
	}
}

// TestMayActGrants_ListAndRevoke proves Phase B: filtered list +
// include_revoked toggle + DELETE marks revoked_at.
func TestMayActGrants_ListAndRevoke(t *testing.T) {
	ts := testutil.NewTestServer(t)

	seedGrant(t, ts.Store, "mag_a", "agent-a", "user-1", false)
	seedGrant(t, ts.Store, "mag_b", "agent-b", "user-1", false)

	// no filters → both
	resp := ts.GetWithAdminKey("/api/v1/admin/may-act")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/may-act: status=%d", resp.StatusCode)
	}
	var out struct {
		Grants []*storage.MayActGrant `json:"grants"`
	}
	ts.DecodeJSON(resp, &out)
	if len(out.Grants) != 2 {
		t.Errorf("no filters: want 2 grants, got %d", len(out.Grants))
	}

	// from_id=agent-a → exactly one
	resp = ts.GetWithAdminKey("/api/v1/admin/may-act?from_id=agent-a")
	out = struct {
		Grants []*storage.MayActGrant `json:"grants"`
	}{}
	ts.DecodeJSON(resp, &out)
	if len(out.Grants) != 1 || out.Grants[0].ID != "mag_a" {
		t.Errorf("from_id filter: want [mag_a], got %+v", out.Grants)
	}

	// Revoke mag_a, then default (include_revoked=false) returns only mag_b.
	resp = ts.DeleteWithAdminKey("/api/v1/admin/may-act/mag_a")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE /admin/may-act/mag_a: status=%d", resp.StatusCode)
	}
	resp = ts.GetWithAdminKey("/api/v1/admin/may-act")
	out = struct {
		Grants []*storage.MayActGrant `json:"grants"`
	}{}
	ts.DecodeJSON(resp, &out)
	if len(out.Grants) != 1 || out.Grants[0].ID != "mag_b" {
		t.Errorf("after revoke, default list: want [mag_b], got %+v", out.Grants)
	}

	// include_revoked=true → both back
	resp = ts.GetWithAdminKey("/api/v1/admin/may-act?include_revoked=true")
	out = struct {
		Grants []*storage.MayActGrant `json:"grants"`
	}{}
	ts.DecodeJSON(resp, &out)
	if len(out.Grants) != 2 {
		t.Errorf("include_revoked: want 2, got %d", len(out.Grants))
	}
}

// TestAuditLogs_FilterByGrantID proves Phase C: ?grant_id= filters via
// json_extract(metadata, '$.grant_id'). Two events, one tagged, one not.
func TestAuditLogs_FilterByGrantID(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Event 1 — tagged with grant_id
	meta1, _ := json.Marshal(map[string]any{"grant_id": "g_filter_test"})
	if err := ts.APIServer.AuditLogger.Log(context.Background(), &storage.AuditLog{
		Action:    "oauth.token.exchanged",
		ActorID:   "agent-x",
		ActorType: "agent",
		Status:    "success",
		Metadata:  string(meta1),
	}); err != nil {
		t.Fatalf("log 1: %v", err)
	}
	// Event 2 — no grant_id
	if err := ts.APIServer.AuditLogger.Log(context.Background(), &storage.AuditLog{
		Action:    "oauth.token.exchanged",
		ActorID:   "agent-y",
		ActorType: "agent",
		Status:    "success",
		Metadata:  `{"unrelated":"ok"}`,
	}); err != nil {
		t.Fatalf("log 2: %v", err)
	}

	resp := ts.GetWithAdminKey("/api/v1/audit-logs?grant_id=g_filter_test")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /audit-logs: %d", resp.StatusCode)
	}
	var page struct {
		Data []*storage.AuditLog `json:"data"`
	}
	ts.DecodeJSON(resp, &page)
	if len(page.Data) != 1 {
		t.Fatalf("want 1 row, got %d: %+v", len(page.Data), page.Data)
	}
	if page.Data[0].ActorID != "agent-x" {
		t.Errorf("wrong row: got actor %q", page.Data[0].ActorID)
	}
}
