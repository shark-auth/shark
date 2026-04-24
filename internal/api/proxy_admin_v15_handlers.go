// Package api — PROXYV1_5 Lane B admin handlers: user tier, branding
// design tokens, YAML rule import. Kept in a dedicated file so the
// v1.5 surface is easy to audit against PROXYV1_5.md and so the
// existing branding/user handler files don't grow unbounded.

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// setUserTierRequest is the PATCH body for /admin/users/{id}/tier.
// Only "free" and "pro" are recognised; anything else rejects with 400
// so typos surface early and the Claims baker never sees a ghost tier.
type setUserTierRequest struct {
	Tier string `json:"tier"`
}

// handleSetUserTier persists the caller-supplied tier into users.metadata
// then re-reads the user for the response. 404 when the user id doesn't
// exist so dashboards can distinguish a missing target from a write
// failure. Emits an audit entry keyed on the actor admin session so the
// same admin-action timeline surfaces tier flips.
func (s *Server) handleSetUserTier(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "user id required"))
		return
	}

	var req setUserTierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	tier := strings.ToLower(strings.TrimSpace(req.Tier))
	if tier != "free" && tier != "pro" {
		writeJSON(w, http.StatusBadRequest,
			errPayload("invalid_tier", `tier must be "free" or "pro"`))
		return
	}

	if err := s.Store.SetUserTier(r.Context(), userID, tier); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "User not found"))
			return
		}
		internal(w, err)
		return
	}

	fresh, err := s.Store.GetUserByID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "user.tier.set",
			TargetType: "user",
			TargetID:   userID,
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Metadata:   `{"tier":"` + tier + `"}`,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"user": fresh,
			"tier": tier,
		},
	})
}

// setDesignTokensRequest is the PATCH body for /admin/branding/design-tokens.
// DesignTokens is a free-form JSON object so the dashboard can evolve
// its token tree (colors.*, typography.*, spacing.*, motion.*) without
// requiring a schema migration per field.
type setDesignTokensRequest struct {
	DesignTokens map[string]any `json:"design_tokens"`
}

// handleSetDesignTokens writes the caller-supplied design tokens into
// the global branding row. The storage layer JSON-encodes the map so
// the branding.design_tokens column stays valid TEXT. Returns the
// freshly-read branding row on success; callers never have to re-GET.
func (s *Server) handleSetDesignTokens(w http.ResponseWriter, r *http.Request) {
	var req setDesignTokensRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.DesignTokens == nil {
		req.DesignTokens = map[string]any{}
	}

	fields := map[string]any{"design_tokens": req.DesignTokens}
	if err := s.Store.UpdateBranding(r.Context(), "global", fields); err != nil {
		internal(w, err)
		return
	}

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "branding.design_tokens.set",
			TargetType: "branding",
			TargetID:   "global",
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
		})
	}

	fresh, err := s.Store.GetBranding(r.Context(), "global")
	if err != nil {
		internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"branding":      fresh,
			"design_tokens": req.DesignTokens,
		},
	})
}

// importYAMLRulesRequest is the POST body for /admin/proxy/rules/import.
type importYAMLRulesRequest struct {
	YAML string `json:"yaml"`
}

// yamlRuleEnvelope mirrors the legacy proxy.rules[] YAML shape. Kept
// local so the import path doesn't depend on the soon-to-be-removed
// config.ProxyConfig.Rules type — Lane B §B5 deletes that field but
// the import surface must still understand the legacy wire form.
type yamlRuleEnvelope struct {
	Rules []yamlRule `yaml:"rules"`
}

type yamlRule struct {
	AppID     string   `yaml:"app_id,omitempty"`
	Name      string   `yaml:"name,omitempty"`
	Path      string   `yaml:"path"`
	Methods   []string `yaml:"methods,omitempty"`
	Require   string   `yaml:"require,omitempty"`
	Allow     string   `yaml:"allow,omitempty"`
	Scopes    []string `yaml:"scopes,omitempty"`
	Enabled   *bool    `yaml:"enabled,omitempty"`
	Priority  int      `yaml:"priority,omitempty"`
	TierMatch string   `yaml:"tier_match,omitempty"`
	M2M       bool     `yaml:"m2m,omitempty"`
}

// handleImportYAMLRules accepts a raw YAML document describing a rules
// list (either a bare `rules:` key or a slice) and upserts each entry
// via Store.CreateProxyRule. On per-row failure we collect the error
// and continue so a single typo doesn't abort the whole batch. Returns
// {imported: N, errors: [...]} so dashboards can show a detailed diff.
func (s *Server) handleImportYAMLRules(w http.ResponseWriter, r *http.Request) {
	var req importYAMLRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if strings.TrimSpace(req.YAML) == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "yaml field required"))
		return
	}

	// Try the top-level envelope first; fall back to a bare list if the
	// caller omitted the `rules:` key.
	var env yamlRuleEnvelope
	if err := yaml.Unmarshal([]byte(req.YAML), &env); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_yaml", err.Error()))
		return
	}
	list := env.Rules
	if len(list) == 0 {
		var bare []yamlRule
		if err := yaml.Unmarshal([]byte(req.YAML), &bare); err == nil {
			list = bare
		}
	}

	now := time.Now().UTC().Truncate(time.Second)
	imported := 0
	var errs []map[string]string

	for i, yr := range list {
		name := yr.Name
		if name == "" {
			name = yr.Path
		}
		if err := validateProxyRulePayload(name, yr.Path, yr.Methods, yr.Require, yr.Allow); err != nil {
			errs = append(errs, map[string]string{
				"index": itoaFallback(i),
				"name":  name,
				"error": err.Error(),
			})
			continue
		}
		enabled := true
		if yr.Enabled != nil {
			enabled = *yr.Enabled
		}
		rule := &storage.ProxyRule{
			ID:        newProxyRuleID(),
			AppID:     yr.AppID,
			Name:      name,
			Pattern:   yr.Path,
			Methods:   normalizeMethods(yr.Methods),
			Require:   yr.Require,
			Allow:     yr.Allow,
			Scopes:    yr.Scopes,
			Enabled:   enabled,
			Priority:  yr.Priority,
			TierMatch: yr.TierMatch,
			M2M:       yr.M2M,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.Store.CreateProxyRule(r.Context(), rule); err != nil {
			errs = append(errs, map[string]string{
				"index": itoaFallback(i),
				"name":  name,
				"error": err.Error(),
			})
			continue
		}
		imported++
	}

	// Refresh the engine so newly-imported rules go live without a
	// separate reload call.
	_ = s.refreshProxyEngineFromDB(r.Context())

	if s.AuditLogger != nil {
		_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
			ActorType:  "admin",
			Action:     "proxy.rules.imported",
			TargetType: "proxy_rule",
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
		})
	}

	if errs == nil {
		errs = []map[string]string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported": imported,
		"errors":   errs,
	})
}

// itoaFallback is a no-import strconv.Itoa shim. Kept local to avoid
// reaching across packages for a one-liner the import paths already
// pay for.
func itoaFallback(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 4)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
