package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/sharkauth/sharkauth/internal/testutil/cli"
)

// TestE2EServeFlow exercises the full --dev-equivalent path: real listener,
// real store, real router. Verifies /healthz, /admin/stats with the bootstrap
// admin key, signup via raw HTTP, self-service session list, admin session
// listing with the joined user_email.
func TestE2EServeFlow(t *testing.T) {
	h := cli.Start(t)

	// /healthz
	resp, err := http.Get(h.BaseURL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz: %d", resp.StatusCode)
	}

	// /admin/stats with bootstrap admin key.
	statsResp := h.Do(h.AdminRequest("GET", "/api/v1/admin/stats"))
	if statsResp.StatusCode != http.StatusOK {
		t.Fatalf("stats: %d", statsResp.StatusCode)
	}
	var stats map[string]any
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	statsResp.Body.Close()
	if _, ok := stats["users"]; !ok {
		t.Fatalf("stats missing users field: %+v", stats)
	}

	// Signup via real HTTP with a cookie jar so the session cookie is reused.
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	signup := `{"email":"cli@x.io","password":"Hunter2Hunter2"}`
	req, _ := http.NewRequest("POST", h.BaseURL+"/api/v1/auth/signup", strings.NewReader(signup))
	req.Header.Set("Content-Type", "application/json")
	signupResp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	signupResp.Body.Close()
	if signupResp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: %d", signupResp.StatusCode)
	}

	// Self-service session list.
	sessResp, err := client.Get(h.BaseURL + "/api/v1/auth/sessions")
	if err != nil {
		t.Fatal(err)
	}
	if sessResp.StatusCode != http.StatusOK {
		t.Fatalf("self sessions: %d", sessResp.StatusCode)
	}
	var sessBody struct {
		Data []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"data"`
	}
	json.NewDecoder(sessResp.Body).Decode(&sessBody)
	sessResp.Body.Close()
	if len(sessBody.Data) == 0 {
		t.Fatal("expected at least one session after signup")
	}

	// Admin session list — should see the signup session with joined email.
	adminListResp := h.Do(h.AdminRequest("GET", "/api/v1/admin/sessions"))
	if adminListResp.StatusCode != http.StatusOK {
		t.Fatalf("admin list: %d", adminListResp.StatusCode)
	}
	var adminList struct {
		Data []struct {
			ID        string `json:"id"`
			UserEmail string `json:"user_email"`
		} `json:"data"`
	}
	json.NewDecoder(adminListResp.Body).Decode(&adminList)
	adminListResp.Body.Close()
	if len(adminList.Data) == 0 || adminList.Data[0].UserEmail != "cli@x.io" {
		t.Fatalf("admin list missing user: %+v", adminList)
	}
}
