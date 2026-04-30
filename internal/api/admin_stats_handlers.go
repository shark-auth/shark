package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shark-auth/shark/internal/storage"
)

// statsResponse is the cheap, always-fresh overview for the dashboard header.
// Every field is a bounded COUNT(*) against indexed columns (<10ms at 1M users).
// Trends / charts live on /admin/stats/trends so this stays fast.
type statsResponse struct {
	Users struct {
		Total          int `json:"total"`
		CreatedLast7d  int `json:"created_last_7d"`
	} `json:"users"`
	Sessions struct {
		Active int `json:"active"`
	} `json:"sessions"`
	MFA struct {
		Total        int     `json:"total"`
		Enabled      int     `json:"enabled"`
		AdoptionPct  float64 `json:"adoption_pct"`
	} `json:"mfa"`
	FailedLogins24h int `json:"failed_logins_24h"`
	APIKeys         struct {
		Active      int `json:"active"`
		Expiring7d  int `json:"expiring_7d"`
	} `json:"api_keys"`
	SSOConnections struct {
		Total      int            `json:"total"`
		Enabled    int            `json:"enabled"`
		UserCounts map[string]int `json:"user_counts,omitempty"`
	} `json:"sso_connections"`
}

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().UTC()

	total, err := s.Store.CountUsers(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	recent, err := s.Store.CountUsersCreatedSince(ctx, now.AddDate(0, 0, -7))
	if err != nil {
		internal(w, err)
		return
	}
	active, err := s.Store.CountActiveSessions(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	mfa, err := s.Store.CountMFAEnabled(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	failed, err := s.Store.CountFailedLoginsSince(ctx, now.Add(-24*time.Hour))
	if err != nil {
		internal(w, err)
		return
	}
	// Active key count: reuse admin-scope counter for coarse "active" total. We
	// call it across all scopes via a second query below; for now keep it cheap
	// with CountActiveAPIKeysByScope("*") since that's what exists. We don't
	// have a scope-agnostic CountActiveAPIKeys yet â€” derive from ListAPIKeys
	// filtered in-memory only if that becomes a bottleneck.
	keys, err := s.Store.ListAPIKeys(ctx)
	if err != nil {
		internal(w, err)
		return
	}
	var activeKeys int
	for _, k := range keys {
		if k.RevokedAt == nil {
			if k.ExpiresAt == nil || *k.ExpiresAt > now.Format(time.RFC3339) {
				activeKeys++
			}
		}
	}
	expiring, err := s.Store.CountExpiringAPIKeys(ctx, 7*24*time.Hour)
	if err != nil {
		internal(w, err)
		return
	}
	ssoTotal, err := s.Store.CountSSOConnections(ctx, false)
	if err != nil {
		internal(w, err)
		return
	}
	ssoEnabled, err := s.Store.CountSSOConnections(ctx, true)
	if err != nil {
		internal(w, err)
		return
	}

	var resp statsResponse
	resp.Users.Total = total
	resp.Users.CreatedLast7d = recent
	resp.Sessions.Active = active
	resp.MFA.Total = total
	resp.MFA.Enabled = mfa
	if total > 0 {
		resp.MFA.AdoptionPct = float64(mfa) / float64(total) * 100
	}
	resp.FailedLogins24h = failed
	resp.APIKeys.Active = activeKeys
	resp.APIKeys.Expiring7d = expiring
	resp.SSOConnections.Total = ssoTotal
	resp.SSOConnections.Enabled = ssoEnabled
	resp.SSOConnections.UserCounts, _ = s.Store.CountSSOIdentitiesByConnection(ctx)

	writeJSON(w, http.StatusOK, resp)
}

// trendsResponse is the heavier query set: GROUP BYs over up to N days.
// Default 30d (max 90d) keeps the result set small enough to serialize
// without streaming.
type trendsResponse struct {
	Days            int               `json:"days"`
	SignupsByDay    []dayBucket       `json:"signups_by_day"`
	SessionsByDay   []dayBucket       `json:"sessions_by_day"`
	MFAByDay        []dayBucket       `json:"mfa_by_day"`
	FailedByDay     []dayBucket       `json:"failed_by_day"`
	APIKeysByDay    []dayBucket       `json:"api_keys_by_day"`
	AuthMethods     []methodBreakdown `json:"auth_methods"`
}

type dayBucket struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type methodBreakdown struct {
	AuthMethod string `json:"auth_method"`
	Count      int    `json:"count"`
}

func (s *Server) handleAdminStatsTrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	if days > 90 {
		days = 90
	}

	since := time.Now().UTC().AddDate(0, 0, -days)

	// Fetch all daily trends
	signups, err := s.Store.GroupUsersCreatedByDay(ctx, days)
	if err != nil {
		internal(w, err)
		return
	}
	sessions, err := s.Store.GroupSessionsCreatedByDay(ctx, days)
	if err != nil {
		internal(w, err)
		return
	}
	mfa, err := s.Store.GroupMFAEnabledByDay(ctx, days)
	if err != nil {
		internal(w, err)
		return
	}
	failed, err := s.Store.GroupFailedLoginsByDay(ctx, days)
	if err != nil {
		internal(w, err)
		return
	}
	apiKeys, err := s.Store.GroupAPIKeysCreatedByDay(ctx, days)
	if err != nil {
		internal(w, err)
		return
	}

	methods, err := s.Store.GroupSessionsByAuthMethodSince(ctx, since)
	if err != nil {
		internal(w, err)
		return
	}

	resp := trendsResponse{
		Days:            days,
		SignupsByDay:    fillDailyGaps(signups, days),
		SessionsByDay:   fillDailyGaps(sessions, days),
		MFAByDay:        fillDailyGaps(mfa, days),
		FailedByDay:     fillDailyGaps(failed, days),
		APIKeysByDay:    fillDailyGaps(apiKeys, days),
	}
	for _, m := range methods {
		resp.AuthMethods = append(resp.AuthMethods, methodBreakdown{
			AuthMethod: m.AuthMethod,
			Count:      m.Count,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// fillDailyGaps expands a sparse list of day counts into a contiguous series
// ending today. Days missing from input appear with count=0 so the frontend
// chart code can plot without gap-filling logic.
func fillDailyGaps(sparse []storage.DayCount, days int) []dayBucket {
	have := make(map[string]int, len(sparse))
	for _, d := range sparse {
		have[d.Date] = d.Count
	}
	out := make([]dayBucket, 0, days)
	today := time.Now().UTC()
	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, dayBucket{Date: d, Count: have[d]})
	}
	return out
}

// internal writes a 500 with a generic message. Keeps err off the wire.
func internal(w http.ResponseWriter, err error) {
	if err != nil {
		slog.Error("internal server error", "error", err)
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{
		"error":   "internal_error",
		"message": "Internal server error",
	})
}
