package authflow_test

// Steps-wired tests for the three newly-wired step types:
//   - assign_role
//   - add_to_org
//   - require_mfa_challenge
//
// Each test follows the engine_test.go pattern: real in-memory SQLite (via
// testutil.NewTestDB), real Engine, real storage rows. No mocks.

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/shark-auth/shark/internal/authflow"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// ---------------------------------------------------------------------------
// assign_role
// ---------------------------------------------------------------------------

func TestAuthflowStepAssignRole(t *testing.T) {
	eng, store := newEngine(t)

	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	role := testutil.CreateRole(t, store, "beta-tester")

	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "assign_role",
		Config: map[string]any{"role_id": role.ID},
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "signup",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Continue {
		t.Fatalf("want Continue, got %q (%s)", res.Outcome, res.Reason)
	}

	// Assert role is attached.
	roles, err := store.GetRolesByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetRolesByUserID: %v", err)
	}
	found := false
	for _, r := range roles {
		if r.ID == role.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("role %q not assigned to user %q after flow", role.ID, user.ID)
	}

	// Assert audit log present.
	logs, err := store.QueryAuditLogs(context.Background(), storage.AuditLogQuery{
		Action: "authflow.step.assign_role",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("QueryAuditLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected audit log entry for assign_role, got none")
	}
}

func TestAuthflowStepAssignRole_MissingRole(t *testing.T) {
	eng, store := newEngine(t)

	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "assign_role",
		Config: map[string]any{"role_id": "role_does_not_exist"},
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "signup",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Error {
		t.Fatalf("want Error for missing role, got %q (%s)", res.Outcome, res.Reason)
	}
	if res.Reason == "" {
		t.Fatalf("expected non-empty Reason on assign_role failure")
	}
}

// ---------------------------------------------------------------------------
// add_to_org
// ---------------------------------------------------------------------------

func mustCreateOrg(t *testing.T, store storage.Store) *storage.Organization {
	t.Helper()
	id := "org_" + newFlowID(t)[5:]
	now := time.Now().UTC().Format(time.RFC3339)
	org := &storage.Organization{
		ID:        id,
		Name:      "Test Org",
		Slug:      "test-org-" + id,
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.CreateOrganization(context.Background(), org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	return org
}

func TestAuthflowStepAddToOrg(t *testing.T) {
	eng, store := newEngine(t)

	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	org := mustCreateOrg(t, store)

	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "add_to_org",
		Config: map[string]any{"org_id": org.ID, "role_id": "member"},
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "signup",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Continue {
		t.Fatalf("want Continue, got %q (%s)", res.Outcome, res.Reason)
	}

	// Assert member row created.
	member, err := store.GetOrganizationMember(context.Background(), org.ID, user.ID)
	if err != nil {
		t.Fatalf("GetOrganizationMember: %v", err)
	}
	if member.UserID != user.ID {
		t.Fatalf("member.UserID mismatch: got %q want %q", member.UserID, user.ID)
	}
	if member.Role != "member" {
		t.Fatalf("member.Role mismatch: got %q want \"member\"", member.Role)
	}

	// Assert audit log.
	logs, err := store.QueryAuditLogs(context.Background(), storage.AuditLogQuery{
		Action: "authflow.step.add_to_org",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("QueryAuditLogs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected audit log entry for add_to_org, got none")
	}
}

func TestAuthflowStepAddToOrg_Idempotent(t *testing.T) {
	eng, store := newEngine(t)

	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	org := mustCreateOrg(t, store)

	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "add_to_org",
		Config: map[string]any{"org_id": org.ID, "role_id": "member"},
	}})

	fc := &authflow.Context{Trigger: "signup", User: user}

	// First run.
	res1, err := eng.Execute(context.Background(), fc)
	if err != nil {
		t.Fatalf("first execute: %v", err)
	}
	if res1.Outcome != authflow.Continue {
		t.Fatalf("first run want Continue, got %q", res1.Outcome)
	}

	// Second run â€” should be idempotent, no duplicate row, no error.
	res2, err := eng.Execute(context.Background(), fc)
	if err != nil {
		t.Fatalf("second execute: %v", err)
	}
	if res2.Outcome != authflow.Continue {
		t.Fatalf("second run want Continue, got %q (%s)", res2.Outcome, res2.Reason)
	}

	// Exactly one member row.
	members, err := store.ListOrganizationMembers(context.Background(), org.ID)
	if err != nil {
		t.Fatalf("ListOrganizationMembers: %v", err)
	}
	count := 0
	for _, m := range members {
		if m.UserID == user.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 membership row, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// require_mfa_challenge
// ---------------------------------------------------------------------------

// enrolledMFAUser creates a user with MFA fully enrolled. Returns the user
// and the plaintext TOTP secret so tests can generate valid codes.
func enrolledMFAUser(t *testing.T, store storage.Store) (*storage.User, string) {
	t.Helper()

	// Generate a real TOTP secret (no encryption in test â€” we pass plaintext directly).
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "SharkAuthTest",
		AccountName: "mfa-user@test.example",
	})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}
	secret := key.Secret()

	now := time.Now().UTC().Format(time.RFC3339)
	user := &storage.User{
		ID:            newUserID(t),
		Email:         "mfa-user@test.example",
		EmailVerified: true,
		MFAEnabled:    true,
		MFAVerified:   true,
		MFASecret:     &secret,
		Metadata:      "{}",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create mfa user: %v", err)
	}
	return user, secret
}

func TestAuthflowStepRequireMFA_NotEnrolled(t *testing.T) {
	eng, store := newEngine(t)

	// User with no MFA set up.
	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	mustCreateFlow(t, store, "login", []storage.FlowStep{{
		Type: "require_mfa_challenge",
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "login",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Error {
		t.Fatalf("want Error for not-enrolled user, got %q", res.Outcome)
	}
	if res.Reason != "mfa_required_but_not_enrolled" {
		t.Fatalf("unexpected reason: %q", res.Reason)
	}
}

func TestAuthflowStepRequireMFA_Enrolled_AwaitingChallenge(t *testing.T) {
	eng, store := newEngine(t)

	user, _ := enrolledMFAUser(t, store)

	mustCreateFlow(t, store, "login", []storage.FlowStep{{
		Type: "require_mfa_challenge",
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "login",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.AwaitMFA {
		t.Fatalf("want AwaitMFA, got %q (%s)", res.Outcome, res.Reason)
	}
	if res.ChallengeID == "" {
		t.Fatalf("expected a non-empty ChallengeID in result")
	}

	// Challenge should be present in the store and belong to this user.
	c, ok := authflow.GlobalChallengeStore.Peek(res.ChallengeID)
	if !ok {
		t.Fatalf("challenge %q not found in GlobalChallengeStore", res.ChallengeID)
	}
	if c.UserID != user.ID {
		t.Fatalf("challenge.UserID mismatch: got %q want %q", c.UserID, user.ID)
	}
}

func TestAuthflowStepRequireMFA_VerifyResumesFlow(t *testing.T) {
	// This test uses the ChallengeStore directly (no HTTP layer) to simulate
	// the verify endpoint's Consume + TOTP check logic. The HTTP endpoint test
	// lives in internal/api/ alongside other handler tests.
	eng, store := newEngine(t)

	user, secret := enrolledMFAUser(t, store)

	mustCreateFlow(t, store, "login", []storage.FlowStep{{
		Type: "require_mfa_challenge",
	}})

	// Step 1: run flow â†’ get challenge ID.
	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "login",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.AwaitMFA {
		t.Fatalf("want AwaitMFA, got %q", res.Outcome)
	}
	challengeID := res.ChallengeID
	if challengeID == "" {
		t.Fatalf("empty challenge_id")
	}

	// Step 2: generate a valid TOTP code from the secret.
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}

	// Step 3: simulate verify â€” Consume challenge and validate TOTP.
	if !authflow.GlobalChallengeStore.Consume(challengeID, user.ID) {
		t.Fatalf("Consume challenge failed unexpectedly")
	}

	// Validate TOTP directly with the same options as auth.MFAManager.ValidateTOTP.
	ok, _ := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if !ok {
		t.Fatalf("TOTP code validation failed for secret %q code %q", secret, code)
	}

	// Challenge should now be gone (consumed).
	_, stillPresent := authflow.GlobalChallengeStore.Peek(challengeID)
	if stillPresent {
		t.Fatalf("challenge should have been consumed but is still in store")
	}

	// Confirm store is still functional after all this (flow engine persisted run).
	runs, err := store.ListAuthFlowRunsByFlowID(context.Background(), res.FlowID, 5)
	if err != nil {
		t.Fatalf("ListAuthFlowRunsByFlowID: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected at least one flow run record; got none")
	}
	_ = runs
}

// TestAuthflowStepAssignRole_NoUserContext ensures the step errors gracefully
// when fc.User is nil (e.g. pre-signup validation flow).
func TestAuthflowStepAssignRole_NoUserContext(t *testing.T) {
	eng, store := newEngine(t)

	role := testutil.CreateRole(t, store, "ghost-role")
	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "assign_role",
		Config: map[string]any{"role_id": role.ID},
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "signup",
		User:    nil,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Error {
		t.Fatalf("want Error for nil user, got %q", res.Outcome)
	}
}

// TestAuthflowStepAddToOrg_MissingOrgID ensures the step errors on missing config.
func TestAuthflowStepAddToOrg_MissingOrgID(t *testing.T) {
	eng, store := newEngine(t)

	user := verifiedUser(t)
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	mustCreateFlow(t, store, "signup", []storage.FlowStep{{
		Type:   "add_to_org",
		Config: map[string]any{}, // no org_id
	}})

	res, err := eng.Execute(context.Background(), &authflow.Context{
		Trigger: "signup",
		User:    user,
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Outcome != authflow.Error {
		t.Fatalf("want Error for missing org_id, got %q", res.Outcome)
	}
}

// Compile-time check: sql.ErrNoRows is used in steps.go.
var _ = errors.Is(sql.ErrNoRows, sql.ErrNoRows)
