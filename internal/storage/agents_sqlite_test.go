package storage_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// newTestAgent returns a minimal Agent for testing.
func newTestAgent(t *testing.T, id, clientID string, active bool) *storage.Agent {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	h := sha256.Sum256([]byte("shh"))
	return &storage.Agent{
		ID:               id,
		Name:             "Agent " + id,
		Description:      "A test agent",
		ClientID:         clientID,
		ClientSecretHash: hex.EncodeToString(h[:]),
		ClientType:       "confidential",
		AuthMethod:       "client_secret_basic",
		RedirectURIs:     []string{"https://example.com/callback"},
		GrantTypes:       []string{"client_credentials"},
		ResponseTypes:    []string{"code"},
		Scopes:           []string{"openid"},
		Metadata:         map[string]any{"env": "test"},
		TokenLifetime:    900,
		Active:           active,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// TestAgent_GetByID creates an agent and round-trips via GetAgentByID to
// exercise the scanAgent path.
func TestAgent_GetByID(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	agent := newTestAgent(t, "agent_get_by_id_1", "client-getbyid-1", true)
	if err := store.CreateAgent(ctx, agent); err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}

	got, err := store.GetAgentByID(ctx, agent.ID)
	if err != nil {
		t.Fatalf("GetAgentByID: %v", err)
	}
	if got.ID != agent.ID {
		t.Errorf("expected id %q, got %q", agent.ID, got.ID)
	}
	if got.ClientID != agent.ClientID {
		t.Errorf("expected client_id %q, got %q", agent.ClientID, got.ClientID)
	}
	if !got.Active {
		t.Error("expected agent to be active")
	}
	if len(got.RedirectURIs) != 1 || got.RedirectURIs[0] != "https://example.com/callback" {
		t.Errorf("unexpected redirect URIs: %v", got.RedirectURIs)
	}
	if got.Metadata == nil || got.Metadata["env"] != "test" {
		t.Errorf("unexpected metadata: %v", got.Metadata)
	}
}

// TestAgent_GetByID_NotFound checks that missing IDs return an error (sql.ErrNoRows).
func TestAgent_GetByID_NotFound(t *testing.T) {
	store := testutil.NewTestDB(t)
	_, err := store.GetAgentByID(context.Background(), "agent_missing")
	if err == nil {
		t.Fatal("expected error for missing agent")
	}
}

// TestAgent_ListAgents creates several agents and exercises ListAgents with
// pagination, filtering, and the scanAgentFromRows path.
func TestAgent_ListAgents(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Seed 3 active + 1 inactive agent.
	for i := 0; i < 3; i++ {
		a := newTestAgent(t, "agent_list_active_"+string(rune('a'+i)), "client-list-active-"+string(rune('a'+i)), true)
		a.Name = "ListTestAgent"
		if err := store.CreateAgent(ctx, a); err != nil {
			t.Fatalf("create active agent %d: %v", i, err)
		}
	}
	inactive := newTestAgent(t, "agent_list_inactive", "client-list-inactive", false)
	inactive.Name = "ListTestInactive"
	if err := store.CreateAgent(ctx, inactive); err != nil {
		t.Fatalf("create inactive agent: %v", err)
	}

	// List all
	all, total, err := store.ListAgents(ctx, storage.ListAgentsOpts{Limit: 10})
	if err != nil {
		t.Fatalf("ListAgents all: %v", err)
	}
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
	if len(all) != 4 {
		t.Errorf("expected 4 agents, got %d", len(all))
	}

	// Filter active=true
	activeTrue := true
	activeOnly, activeTotal, err := store.ListAgents(ctx, storage.ListAgentsOpts{
		Limit:  10,
		Active: &activeTrue,
	})
	if err != nil {
		t.Fatalf("ListAgents active: %v", err)
	}
	if activeTotal != 3 {
		t.Errorf("expected 3 active, got %d", activeTotal)
	}
	for _, a := range activeOnly {
		if !a.Active {
			t.Errorf("expected agent %q to be active", a.ID)
		}
	}

	// Filter active=false
	activeFalse := false
	inactiveOnly, inactiveTotal, err := store.ListAgents(ctx, storage.ListAgentsOpts{
		Limit:  10,
		Active: &activeFalse,
	})
	if err != nil {
		t.Fatalf("ListAgents inactive: %v", err)
	}
	if inactiveTotal != 1 {
		t.Errorf("expected 1 inactive, got %d", inactiveTotal)
	}
	if len(inactiveOnly) != 1 || inactiveOnly[0].Active {
		t.Errorf("unexpected inactive list: %v", inactiveOnly)
	}

	// Search filter
	searchResults, searchTotal, err := store.ListAgents(ctx, storage.ListAgentsOpts{
		Limit:  10,
		Search: "ListTestInactive",
	})
	if err != nil {
		t.Fatalf("ListAgents search: %v", err)
	}
	if searchTotal != 1 {
		t.Errorf("expected search total=1, got %d", searchTotal)
	}
	if len(searchResults) != 1 {
		t.Errorf("expected 1 search result, got %d", len(searchResults))
	}

	// Pagination: limit=2 offset=0
	page1, _, err := store.ListAgents(ctx, storage.ListAgentsOpts{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 on page 1, got %d", len(page1))
	}

	// Default limit when 0
	defResults, _, err := store.ListAgents(ctx, storage.ListAgentsOpts{})
	if err != nil {
		t.Fatalf("default-limit: %v", err)
	}
	if len(defResults) != 4 {
		t.Errorf("expected 4 with default limit, got %d", len(defResults))
	}
}
