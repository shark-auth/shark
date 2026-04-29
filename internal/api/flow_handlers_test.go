package api_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"testing"
	"time"

	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/testutil"
)

// --- helpers ---

// newTestRunID mirrors the engine's fr_ prefix scheme so seeded history
// rows look identical to real runs.
func newTestRunID(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return "fr_" + hex.EncodeToString(buf)
}

// createFlowViaAPI exercises the POST endpoint and returns the persisted id.
// Factoring this out of every test keeps the happy-path assertions readable.
func createFlowViaAPI(t *testing.T, ts *testutil.TestServer, body map[string]any) (string, *http.Response) {
	t.Helper()
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", body)
	var out struct {
		ID string `json:"id"`
	}
	ts.DecodeJSON(resp, &out)
	return out.ID, resp
}

// basicSignupFlowBody returns a minimal create-payload used by most tests.
// Callers tweak name/trigger as needed before POSTing.
func basicSignupFlowBody() map[string]any {
	return map[string]any{
		"name":    "Default Signup Flow",
		"trigger": "signup",
		"steps": []map[string]any{
			{"type": "require_email_verification"},
		},
		"enabled":  true,
		"priority": 10,
	}
}

// --- CRUD ---

func TestCreateFlow_ValidPayload_201(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", basicSignupFlowBody())
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d, want 201", resp.StatusCode)
	}
	var got map[string]any
	ts.DecodeJSON(resp, &got)
	if got["id"] == "" || got["id"] == nil {
		t.Fatalf("missing id in response: %+v", got)
	}
	if got["trigger"] != "signup" {
		t.Fatalf("trigger mismatch: %v", got["trigger"])
	}
	if got["enabled"] != true {
		t.Fatalf("enabled mismatch: %v", got["enabled"])
	}
}

func TestCreateFlow_MissingName_400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	body := basicSignupFlowBody()
	body["name"] = ""
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestCreateFlow_BadTrigger_400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	body := basicSignupFlowBody()
	body["trigger"] = "not_a_trigger"
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestCreateFlow_EmptySteps_400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	body := basicSignupFlowBody()
	body["steps"] = []map[string]any{}
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestCreateFlow_UnknownStepType_400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	body := basicSignupFlowBody()
	body["steps"] = []map[string]any{{"type": "teleport_user"}}
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestListFlows_ReturnsAll(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Seed two flows with different triggers.
	bodyA := basicSignupFlowBody()
	bodyA["name"] = "A"
	createFlowViaAPI(t, ts, bodyA)

	bodyB := basicSignupFlowBody()
	bodyB["name"] = "B"
	bodyB["trigger"] = "login"
	createFlowViaAPI(t, ts, bodyB)

	resp := ts.GetWithAdminKey("/api/v1/admin/flows")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	var got struct {
		Data  []map[string]any `json:"data"`
		Total int              `json:"total"`
	}
	ts.DecodeJSON(resp, &got)
	if got.Total != 2 || len(got.Data) != 2 {
		t.Fatalf("expected 2 flows, got %d (data len %d)", got.Total, len(got.Data))
	}
}

func TestListFlows_FilterByTrigger(t *testing.T) {
	ts := testutil.NewTestServer(t)
	createFlowViaAPI(t, ts, basicSignupFlowBody())
	loginBody := basicSignupFlowBody()
	loginBody["trigger"] = "login"
	createFlowViaAPI(t, ts, loginBody)

	resp := ts.GetWithAdminKey("/api/v1/admin/flows?trigger=login")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var got struct {
		Data []map[string]any `json:"data"`
	}
	ts.DecodeJSON(resp, &got)
	if len(got.Data) != 1 {
		t.Fatalf("expected 1 login flow, got %d", len(got.Data))
	}
	if got.Data[0]["trigger"] != "login" {
		t.Fatalf("wrong trigger: %v", got.Data[0]["trigger"])
	}
}

func TestGetFlow_NotFound_404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.GetWithAdminKey("/api/v1/admin/flows/flow_nonexistent")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

func TestUpdateFlow_PartialUpdate(t *testing.T) {
	ts := testutil.NewTestServer(t)

	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	// Patch only enabled â€” name + steps + priority must survive.
	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/flows/"+id, map[string]any{
		"enabled": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status=%d", resp.StatusCode)
	}
	var updated map[string]any
	ts.DecodeJSON(resp, &updated)
	if updated["enabled"] != false {
		t.Fatalf("enabled not flipped: %v", updated["enabled"])
	}
	if updated["name"] != "Default Signup Flow" {
		t.Fatalf("name not preserved: %v", updated["name"])
	}
	if updated["priority"].(float64) != 10 {
		t.Fatalf("priority not preserved: %v", updated["priority"])
	}
	steps, ok := updated["steps"].([]any)
	if !ok || len(steps) != 1 {
		t.Fatalf("steps not preserved: %+v", updated["steps"])
	}
}

func TestUpdateFlow_NotFound_404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/flows/flow_nope", map[string]any{
		"enabled": false,
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

func TestUpdateFlow_EmptyName_400(t *testing.T) {
	ts := testutil.NewTestServer(t)
	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	resp := ts.PatchJSONWithAdminKey("/api/v1/admin/flows/"+id, map[string]any{
		"name": "",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestDeleteFlow_Cascades_Runs(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	// Seed two runs via store so we can observe the cascade.
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 2; i++ {
		run := &storage.AuthFlowRun{
			ID:         newTestRunID(t),
			FlowID:     id,
			Trigger:    "signup",
			Outcome:    storage.AuthFlowOutcomeContinue,
			Metadata:   map[string]any{},
			StartedAt:  now,
			FinishedAt: now,
		}
		if err := ts.Store.CreateAuthFlowRun(ctx, run); err != nil {
			t.Fatalf("seed run: %v", err)
		}
	}

	runs, err := ts.Store.ListAuthFlowRunsByFlowID(ctx, id, 50)
	if err != nil {
		t.Fatalf("pre-delete list: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 seeded runs, got %d", len(runs))
	}

	delResp := ts.DeleteWithAdminKey("/api/v1/admin/flows/" + id)
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d, want 204", delResp.StatusCode)
	}

	// Flow gone.
	if _, err := ts.Store.GetAuthFlowByID(ctx, id); err == nil {
		t.Fatal("flow still present after delete")
	}
	// Runs cascaded.
	runs, err = ts.Store.ListAuthFlowRunsByFlowID(ctx, id, 50)
	if err != nil {
		t.Fatalf("post-delete list: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("expected 0 runs post-cascade, got %d", len(runs))
	}
}

func TestDeleteFlow_NotFound_404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.DeleteWithAdminKey("/api/v1/admin/flows/flow_missing")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

// --- Test / dry-run ---

func TestTestFlow_DryRun_ReturnsTimeline(t *testing.T) {
	ts := testutil.NewTestServer(t)

	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows/"+id+"/test", map[string]any{
		"user": map[string]any{
			"email":          "alice@example.com",
			"email_verified": false,
			"name":           "Alice",
		},
		"password": "correct-horse-battery-staple",
		"metadata": map[string]any{"test": true},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	var got struct {
		Outcome  string           `json:"outcome"`
		Timeline []map[string]any `json:"timeline"`
		Reason   string           `json:"reason"`
	}
	ts.DecodeJSON(resp, &got)
	if got.Outcome != "block" {
		t.Fatalf("outcome=%q, want block", got.Outcome)
	}
	if len(got.Timeline) != 1 {
		t.Fatalf("timeline len=%d, want 1", len(got.Timeline))
	}
	if got.Timeline[0]["type"] != "require_email_verification" {
		t.Fatalf("timeline step type=%v", got.Timeline[0]["type"])
	}
}

func TestTestFlow_DryRun_DoesNotPersistRun(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows/"+id+"/test", map[string]any{
		"user": map[string]any{
			"email":          "alice@example.com",
			"email_verified": false,
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}

	runs, err := ts.Store.ListAuthFlowRunsByFlowID(ctx, id, 50)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 0 {
		t.Fatalf("dry-run persisted %d runs (want 0)", len(runs))
	}
}

func TestTestFlow_NotFound_404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.PostJSONWithAdminKey("/api/v1/admin/flows/flow_missing/test", map[string]any{})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

// --- History ---

func TestListFlowRuns_ReturnsHistory(t *testing.T) {
	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	id, _ := createFlowViaAPI(t, ts, basicSignupFlowBody())

	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		run := &storage.AuthFlowRun{
			ID:         newTestRunID(t),
			FlowID:     id,
			Trigger:    "signup",
			Outcome:    storage.AuthFlowOutcomeContinue,
			Metadata:   map[string]any{"seq": i},
			StartedAt:  now.Add(time.Duration(i) * time.Second),
			FinishedAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := ts.Store.CreateAuthFlowRun(ctx, run); err != nil {
			t.Fatalf("seed run %d: %v", i, err)
		}
	}

	resp := ts.GetWithAdminKey("/api/v1/admin/flows/" + id + "/runs")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var got struct {
		Data []map[string]any `json:"data"`
	}
	ts.DecodeJSON(resp, &got)
	if len(got.Data) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(got.Data))
	}
}

func TestListFlowRuns_NotFound_404(t *testing.T) {
	ts := testutil.NewTestServer(t)
	resp := ts.GetWithAdminKey("/api/v1/admin/flows/flow_missing/runs")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", resp.StatusCode)
	}
}

// --- Integration hooks ---

func TestSignup_FlowBlocksUnverifiedEmail(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Install a flow that blocks unverified signups â€” which every fresh
	// signup is by default.
	createFlowViaAPI(t, ts, basicSignupFlowBody())

	resp := ts.PostJSON("/api/v1/auth/signup", map[string]any{
		"email":    "flow.blocked@example.com",
		"password": "CorrectHorseBatteryStaple!1",
		"name":     "Blocked User",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", resp.StatusCode)
	}
	var body map[string]string
	ts.DecodeJSON(resp, &body)
	if body["error"] != "flow_blocked" {
		t.Fatalf("error=%q, want flow_blocked", body["error"])
	}

	// The flow runs AFTER CreateUser â€” the row should exist even though the
	// session was withheld. This is the documented contract: admins can
	// unblock and the user doesn't have to re-signup.
	if _, err := ts.Store.GetUserByEmail(context.Background(), "flow.blocked@example.com"); err != nil {
		t.Fatalf("expected user row to persist, got err: %v", err)
	}
}

// TestUpdateFlow_ConditionalBranchesPreserved asserts that nested then/else
// branches survive a full PATCH â†’ GET round-trip without data loss.
//
// Acceptance criteria:
//  1. Conditional + webhook in then: PATCH stores it, GET reads it back intact.
//  2. else branch is empty â†’ omitted from response (omitempty) but not corrupted.
//  3. Multi-level nesting: conditional inside then branch also survives.
//  4. Explicit else steps also survive.
func TestUpdateFlow_ConditionalBranchesPreserved(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// --- Sub-test 1: conditional with webhook in then, empty else ---
	t.Run("then_only", func(t *testing.T) {
		id, cr := createFlowViaAPI(t, ts, map[string]any{
			"name":    "Conditional Then-Only",
			"trigger": "signup",
			"steps": []map[string]any{
				{
					"type":      "conditional",
					"condition": "user.email_domain == 'acme.com'",
					"then": []map[string]any{
						{"type": "webhook", "config": map[string]any{"url": "https://hooks.example.com/notify"}},
					},
				},
			},
			"enabled":  false,
			"priority": 10,
		})
		if cr.StatusCode != http.StatusCreated {
			t.Fatalf("create status=%d", cr.StatusCode)
		}

		// PATCH â€” resend same steps (simulates builder save).
		patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/flows/"+id, map[string]any{
			"steps": []map[string]any{
				{
					"type":      "conditional",
					"condition": "user.email_domain == 'acme.com'",
					"then": []map[string]any{
						{"type": "webhook", "config": map[string]any{"url": "https://hooks.example.com/notify"}},
					},
					"else": []map[string]any{}, // frontend always sends else even when empty
				},
			},
		})
		if patchResp.StatusCode != http.StatusOK {
			var e map[string]any
			ts.DecodeJSON(patchResp, &e)
			t.Fatalf("patch status=%d body=%+v", patchResp.StatusCode, e)
		}

		// GET and verify then-branch intact.
		var got map[string]any
		ts.DecodeJSON(ts.GetWithAdminKey("/api/v1/admin/flows/"+id), &got)

		steps := got["steps"].([]any)
		if len(steps) != 1 {
			t.Fatalf("want 1 step, got %d", len(steps))
		}
		cond := steps[0].(map[string]any)
		if cond["type"] != "conditional" {
			t.Fatalf("steps[0].type=%v, want conditional", cond["type"])
		}
		then, ok := cond["then"].([]any)
		if !ok || len(then) != 1 {
			t.Fatalf("steps[0].then: got %v (len %d), want 1 webhook step", cond["then"], len(then))
		}
		if then[0].(map[string]any)["type"] != "webhook" {
			t.Fatalf("then[0].type=%v, want webhook", then[0].(map[string]any)["type"])
		}
		// else must be absent or empty â€” must NOT be a non-empty slice.
		if elseRaw, hasElse := cond["else"]; hasElse {
			elseSlice, ok := elseRaw.([]any)
			if !ok || len(elseSlice) != 0 {
				t.Fatalf("steps[0].else should be absent/empty, got %v", elseRaw)
			}
		}
	})

	// --- Sub-test 2: both branches populated ---
	t.Run("then_and_else", func(t *testing.T) {
		id, cr := createFlowViaAPI(t, ts, map[string]any{
			"name":    "Conditional Both Branches",
			"trigger": "login",
			"steps": []map[string]any{
				{
					"type":      "conditional",
					"condition": "user.email_domain == 'acme.com'",
					"then": []map[string]any{
						{"type": "webhook", "config": map[string]any{"url": "https://hooks.example.com/acme"}},
					},
					"else": []map[string]any{
						{"type": "redirect", "config": map[string]any{"url": "https://example.com/onboarding"}},
					},
				},
			},
			"enabled":  false,
			"priority": 5,
		})
		if cr.StatusCode != http.StatusCreated {
			t.Fatalf("create status=%d", cr.StatusCode)
		}

		var got map[string]any
		ts.DecodeJSON(ts.GetWithAdminKey("/api/v1/admin/flows/"+id), &got)

		steps := got["steps"].([]any)
		cond := steps[0].(map[string]any)
		then := cond["then"].([]any)
		elseBranch := cond["else"].([]any)

		if len(then) != 1 || then[0].(map[string]any)["type"] != "webhook" {
			t.Fatalf("then branch wrong: %v", then)
		}
		if len(elseBranch) != 1 || elseBranch[0].(map[string]any)["type"] != "redirect" {
			t.Fatalf("else branch wrong: %v", elseBranch)
		}
	})

	// --- Sub-test 3: multi-level nesting (conditional inside conditional's then) ---
	t.Run("nested_conditional", func(t *testing.T) {
		id, cr := createFlowViaAPI(t, ts, map[string]any{
			"name":    "Nested Conditional",
			"trigger": "signup",
			"steps": []map[string]any{
				{
					"type":      "conditional",
					"condition": "user.email_domain == 'acme.com'",
					"then": []map[string]any{
						{
							"type":      "conditional",
							"condition": "user.email_verified == true",
							"then": []map[string]any{
								{"type": "webhook", "config": map[string]any{"url": "https://hooks.example.com/deep"}},
							},
						},
					},
				},
			},
			"enabled":  false,
			"priority": 1,
		})
		if cr.StatusCode != http.StatusCreated {
			t.Fatalf("create status=%d", cr.StatusCode)
		}

		// PATCH round-trip.
		patchResp := ts.PatchJSONWithAdminKey("/api/v1/admin/flows/"+id, map[string]any{
			"enabled": false, // no-op patch â€” triggers re-fetch
		})
		if patchResp.StatusCode != http.StatusOK {
			t.Fatalf("patch status=%d", patchResp.StatusCode)
		}

		var got map[string]any
		ts.DecodeJSON(ts.GetWithAdminKey("/api/v1/admin/flows/"+id), &got)

		steps := got["steps"].([]any)
		outer := steps[0].(map[string]any)
		if outer["type"] != "conditional" {
			t.Fatalf("outer type=%v", outer["type"])
		}
		outerThen := outer["then"].([]any)
		if len(outerThen) != 1 {
			t.Fatalf("outer then len=%d want 1", len(outerThen))
		}
		inner := outerThen[0].(map[string]any)
		if inner["type"] != "conditional" {
			t.Fatalf("inner type=%v want conditional", inner["type"])
		}
		innerThen := inner["then"].([]any)
		if len(innerThen) != 1 || innerThen[0].(map[string]any)["type"] != "webhook" {
			t.Fatalf("inner then webhook missing: %v", innerThen)
		}
	})
}

func TestLogin_FlowRedirectsToMFA(t *testing.T) {
	ts := testutil.NewTestServer(t)

	// Seed a login flow that redirects. The user is created before the
	// flow is installed so the signup hook doesn't trip.
	email := "redirect.me@example.com"
	password := "CorrectHorseBatteryStaple!1"
	ts.SignupAndVerify(email, password, "Redirect Me")

	loginFlow := map[string]any{
		"name":    "Login Redirect to MFA Setup",
		"trigger": "login",
		"steps": []map[string]any{
			{
				"type":   "redirect",
				"config": map[string]any{"url": "https://example.com/mfa/setup"},
			},
		},
		"enabled":  true,
		"priority": 10,
	}
	createFlowViaAPI(t, ts, loginFlow)

	resp := ts.PostJSON("/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": password,
	})
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status=%d, want 302", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "https://example.com/mfa/setup" {
		t.Fatalf("Location=%q, want https://example.com/mfa/setup", loc)
	}
}
