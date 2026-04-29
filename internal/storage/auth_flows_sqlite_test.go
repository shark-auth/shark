package storage_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// newFlowID generates the "flow_<20-hex>" identifier the spec reserves for
// AuthFlow rows. The tests drive this the same way production code will so
// any shape drift (prefix, length) shows up immediately.
func newFlowID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return "flow_" + hex.EncodeToString(buf)
}

// newFlowRunID is the "fr_<20-hex>" counterpart for AuthFlowRun rows.
func newFlowRunID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return "fr_" + hex.EncodeToString(buf)
}

// newTestAuthFlow produces a minimal AuthFlow â€” single require_email_verification
// step, no conditions. Tests layer trigger/priority/step overrides on top.
func newTestAuthFlow(t *testing.T, trigger string) *storage.AuthFlow {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	return &storage.AuthFlow{
		ID:      newFlowID(t),
		Name:    "Default " + trigger + " flow",
		Trigger: trigger,
		Steps: []storage.FlowStep{
			{Type: "require_email_verification"},
		},
		Enabled:    true,
		Priority:   0,
		Conditions: map[string]any{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestAuthFlowCRUD(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	f.Priority = 10
	f.Conditions = map[string]any{"user.org_id": "org_acme"}

	// Create
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	// GetByID
	got, err := store.GetAuthFlowByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetAuthFlowByID: %v", err)
	}
	if got.Name != f.Name || got.Trigger != f.Trigger {
		t.Errorf("name/trigger mismatch: got %q/%q", got.Name, got.Trigger)
	}
	if got.Priority != 10 {
		t.Errorf("Priority: got %d, want 10", got.Priority)
	}
	if !got.Enabled {
		t.Error("expected Enabled=true")
	}
	if got.Conditions["user.org_id"] != "org_acme" {
		t.Errorf("Conditions: %v", got.Conditions)
	}
	if len(got.Steps) != 1 || got.Steps[0].Type != "require_email_verification" {
		t.Errorf("Steps: %+v", got.Steps)
	}

	// List (should contain our flow)
	all, err := store.ListAuthFlows(ctx)
	if err != nil {
		t.Fatalf("ListAuthFlows: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 flow, got %d", len(all))
	}

	// Update
	f.Name = "Renamed"
	f.Priority = 20
	f.Steps = []storage.FlowStep{{Type: "require_mfa_enrollment"}}
	if err := store.UpdateAuthFlow(ctx, f); err != nil {
		t.Fatalf("UpdateAuthFlow: %v", err)
	}
	upd, err := store.GetAuthFlowByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if upd.Name != "Renamed" {
		t.Errorf("Name after update: %q", upd.Name)
	}
	if upd.Priority != 20 {
		t.Errorf("Priority after update: %d", upd.Priority)
	}
	if len(upd.Steps) != 1 || upd.Steps[0].Type != "require_mfa_enrollment" {
		t.Errorf("Steps after update: %+v", upd.Steps)
	}

	// Delete
	if err := store.DeleteAuthFlow(ctx, f.ID); err != nil {
		t.Fatalf("DeleteAuthFlow: %v", err)
	}
	_, err = store.GetAuthFlowByID(ctx, f.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Get after delete: got %v, want sql.ErrNoRows", err)
	}
}

func TestAuthFlow_StepsRoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	steps := []storage.FlowStep{
		{
			Type:   "require_email_verification",
			Config: map[string]any{"max_age_hours": float64(24)},
		},
		{
			Type:      "conditional",
			Condition: "user.email_domain == 'acme.com'",
			ThenSteps: []storage.FlowStep{
				{Type: "require_mfa_enrollment"},
				{Type: "webhook", Config: map[string]any{"url": "https://hook.example/audit"}},
			},
			ElseSteps: []storage.FlowStep{
				{Type: "redirect", Config: map[string]any{"to": "/onboarding"}},
			},
		},
		{
			Type:   "webhook",
			Config: map[string]any{"url": "https://hook.example/end", "method": "POST"},
		},
	}

	f := newTestAuthFlow(t, storage.AuthFlowTriggerLogin)
	f.Steps = steps
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	got, err := store.GetAuthFlowByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetAuthFlowByID: %v", err)
	}

	if !reflect.DeepEqual(got.Steps, steps) {
		t.Errorf("steps round-trip mismatch:\n got:  %#v\n want: %#v", got.Steps, steps)
	}
}

func TestAuthFlow_ListByTrigger_OrderedByPriorityDesc(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	// Stagger created_at so tie-break is deterministic, but priority is what
	// the ordering should key on.
	base := time.Now().UTC().Truncate(time.Second)
	mk := func(name string, priority int, offset time.Duration) *storage.AuthFlow {
		f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
		f.Name = name
		f.Priority = priority
		f.CreatedAt = base.Add(offset)
		f.UpdatedAt = f.CreatedAt
		return f
	}

	low := mk("low", 1, 0)
	hi := mk("high", 5, 1*time.Second)
	mid := mk("mid", 3, 2*time.Second)

	for _, f := range []*storage.AuthFlow{low, hi, mid} {
		if err := store.CreateAuthFlow(ctx, f); err != nil {
			t.Fatalf("CreateAuthFlow %q: %v", f.Name, err)
		}
	}

	list, err := store.ListAuthFlowsByTrigger(ctx, storage.AuthFlowTriggerSignup)
	if err != nil {
		t.Fatalf("ListAuthFlowsByTrigger: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 flows, got %d", len(list))
	}
	wantOrder := []int{5, 3, 1}
	for i, want := range wantOrder {
		if list[i].Priority != want {
			t.Errorf("index %d: priority got %d, want %d", i, list[i].Priority, want)
		}
	}
}

func TestAuthFlow_ListByTrigger_FiltersOutOtherTriggers(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	signup := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	login := newTestAuthFlow(t, storage.AuthFlowTriggerLogin)
	for _, f := range []*storage.AuthFlow{signup, login} {
		if err := store.CreateAuthFlow(ctx, f); err != nil {
			t.Fatalf("CreateAuthFlow: %v", err)
		}
	}

	got, err := store.ListAuthFlowsByTrigger(ctx, storage.AuthFlowTriggerSignup)
	if err != nil {
		t.Fatalf("ListAuthFlowsByTrigger: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 signup flow, got %d", len(got))
	}
	if got[0].ID != signup.ID {
		t.Errorf("wrong flow returned: got %s, want %s", got[0].ID, signup.ID)
	}
	if got[0].Trigger != storage.AuthFlowTriggerSignup {
		t.Errorf("trigger leaked through filter: %q", got[0].Trigger)
	}
}

func TestAuthFlow_ConditionsRoundTrip(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerLogin)
	// JSON decodes numbers as float64 and arrays as []any, so set the fixture
	// in the shape we expect to read back.
	f.Conditions = map[string]any{
		"user.verified": true,
		"plan":          "pro",
		"attempt_max":   float64(5),
		"allowed_ips":   []any{"10.0.0.1", "10.0.0.2"},
	}
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	got, err := store.GetAuthFlowByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetAuthFlowByID: %v", err)
	}
	if !reflect.DeepEqual(got.Conditions, f.Conditions) {
		t.Errorf("conditions round-trip mismatch:\n got:  %#v\n want: %#v", got.Conditions, f.Conditions)
	}
}

func TestAuthFlow_EnabledToggle(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	f.Enabled = true
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	f.Enabled = false
	if err := store.UpdateAuthFlow(ctx, f); err != nil {
		t.Fatalf("UpdateAuthFlow: %v", err)
	}

	got, err := store.GetAuthFlowByID(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetAuthFlowByID: %v", err)
	}
	if got.Enabled {
		t.Error("expected Enabled=false after update")
	}
}

func TestAuthFlowRun_Create_AndListByFlowID(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	// Three runs with staggered start times so newest-first order is testable.
	base := time.Now().UTC().Truncate(time.Second)
	mkRun := func(offset time.Duration, outcome string) *storage.AuthFlowRun {
		start := base.Add(offset)
		return &storage.AuthFlowRun{
			ID:         newFlowRunID(t),
			FlowID:     f.ID,
			UserID:     "u_abc",
			Trigger:    f.Trigger,
			Outcome:    outcome,
			Metadata:   map[string]any{"step": "start"},
			StartedAt:  start,
			FinishedAt: start.Add(10 * time.Millisecond),
		}
	}

	oldest := mkRun(0, storage.AuthFlowOutcomeContinue)
	middle := mkRun(1*time.Second, storage.AuthFlowOutcomeContinue)
	newest := mkRun(2*time.Second, storage.AuthFlowOutcomeBlock)

	for _, r := range []*storage.AuthFlowRun{oldest, middle, newest} {
		if err := store.CreateAuthFlowRun(ctx, r); err != nil {
			t.Fatalf("CreateAuthFlowRun: %v", err)
		}
	}

	got, err := store.ListAuthFlowRunsByFlowID(ctx, f.ID, 50)
	if err != nil {
		t.Fatalf("ListAuthFlowRunsByFlowID: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(got))
	}
	wantOrder := []string{newest.ID, middle.ID, oldest.ID}
	for i, id := range wantOrder {
		if got[i].ID != id {
			t.Errorf("index %d: got id %q, want %q", i, got[i].ID, id)
		}
	}
	if got[0].Outcome != storage.AuthFlowOutcomeBlock {
		t.Errorf("newest outcome: got %q", got[0].Outcome)
	}
}

func TestAuthFlowRun_FKCascade_OnFlowDelete(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	run := &storage.AuthFlowRun{
		ID:         newFlowRunID(t),
		FlowID:     f.ID,
		Trigger:    f.Trigger,
		Outcome:    storage.AuthFlowOutcomeContinue,
		Metadata:   map[string]any{},
		StartedAt:  now,
		FinishedAt: now,
	}
	if err := store.CreateAuthFlowRun(ctx, run); err != nil {
		t.Fatalf("CreateAuthFlowRun: %v", err)
	}

	// Deleting the flow should cascade away its runs.
	if err := store.DeleteAuthFlow(ctx, f.ID); err != nil {
		t.Fatalf("DeleteAuthFlow: %v", err)
	}

	runs, err := store.ListAuthFlowRunsByFlowID(ctx, f.ID, 10)
	if err != nil {
		t.Fatalf("ListAuthFlowRunsByFlowID after delete: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected runs to cascade away, got %d rows", len(runs))
	}
}

func TestAuthFlowRun_NullableUserIDAndReason(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	run := &storage.AuthFlowRun{
		ID:         newFlowRunID(t),
		FlowID:     f.ID,
		UserID:     "", // pre-signup trigger, no user yet
		Trigger:    f.Trigger,
		Outcome:    storage.AuthFlowOutcomeContinue,
		Reason:     "",
		Metadata:   map[string]any{},
		StartedAt:  now,
		FinishedAt: now,
	}
	if err := store.CreateAuthFlowRun(ctx, run); err != nil {
		t.Fatalf("CreateAuthFlowRun: %v", err)
	}

	got, err := store.ListAuthFlowRunsByFlowID(ctx, f.ID, 10)
	if err != nil {
		t.Fatalf("ListAuthFlowRunsByFlowID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 run, got %d", len(got))
	}
	if got[0].UserID != "" {
		t.Errorf("UserID: got %q, want empty", got[0].UserID)
	}
	if got[0].Reason != "" {
		t.Errorf("Reason: got %q, want empty", got[0].Reason)
	}
}

// intPtr is a small helper for BlockedAtStep tests; avoids littering tests
// with &two locals.
func intPtr(i int) *int { return &i }

func TestAuthFlowRun_BlockedAtStepNullable(t *testing.T) {
	store := testutil.NewTestDB(t)
	ctx := context.Background()

	f := newTestAuthFlow(t, storage.AuthFlowTriggerSignup)
	if err := store.CreateAuthFlow(ctx, f); err != nil {
		t.Fatalf("CreateAuthFlow: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)

	// Case 1: nil pointer survives as nil.
	nilRun := &storage.AuthFlowRun{
		ID:            newFlowRunID(t),
		FlowID:        f.ID,
		Trigger:       f.Trigger,
		Outcome:       storage.AuthFlowOutcomeContinue,
		BlockedAtStep: nil,
		Metadata:      map[string]any{},
		StartedAt:     now,
		FinishedAt:    now,
	}
	if err := store.CreateAuthFlowRun(ctx, nilRun); err != nil {
		t.Fatalf("CreateAuthFlowRun nil: %v", err)
	}

	// Case 2: ptr(2) survives as ptr-to-2.
	blockedRun := &storage.AuthFlowRun{
		ID:            newFlowRunID(t),
		FlowID:        f.ID,
		Trigger:       f.Trigger,
		Outcome:       storage.AuthFlowOutcomeBlock,
		BlockedAtStep: intPtr(2),
		Reason:        "email not verified",
		Metadata:      map[string]any{},
		StartedAt:     now.Add(1 * time.Second),
		FinishedAt:    now.Add(1 * time.Second),
	}
	if err := store.CreateAuthFlowRun(ctx, blockedRun); err != nil {
		t.Fatalf("CreateAuthFlowRun blocked: %v", err)
	}

	runs, err := store.ListAuthFlowRunsByFlowID(ctx, f.ID, 10)
	if err != nil {
		t.Fatalf("ListAuthFlowRunsByFlowID: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}

	// runs[0] is newest (blockedRun), runs[1] is oldest (nilRun).
	if runs[0].BlockedAtStep == nil {
		t.Error("expected BlockedAtStep non-nil on blocked run")
	} else if *runs[0].BlockedAtStep != 2 {
		t.Errorf("BlockedAtStep: got %d, want 2", *runs[0].BlockedAtStep)
	}
	if runs[0].Reason != "email not verified" {
		t.Errorf("Reason: got %q", runs[0].Reason)
	}

	if runs[1].BlockedAtStep != nil {
		t.Errorf("expected nil BlockedAtStep on continue run, got %v", *runs[1].BlockedAtStep)
	}
}
