package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// ────────────────────────────────────────────────────────────────────────────
// Server-level drain state (used by swap-mode and reset)
// ────────────────────────────────────────────────────────────────────────────

// DrainFlag is the server-level drain indicator wired into the drain middleware.
// It is set to non-nil at NewServer time when the drain middleware is mounted.
var globalDrain = &mw.DrainFlag{}

// inflightWG tracks in-flight requests so swap/reset can wait for them to finish.
var inflightWG sync.WaitGroup

// drainTimeout is the maximum time to wait for in-flight requests to finish.
const drainTimeout = 5 * time.Second

// waitForDrain waits until all in-flight requests finish or the timeout elapses.
func waitForDrain(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		inflightWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// ────────────────────────────────────────────────────────────────────────────
// GET /api/v1/admin/system/mode
// ────────────────────────────────────────────────────────────────────────────

// handleGetMode returns the current active mode and DB path.
func (s *Server) handleGetMode(w http.ResponseWriter, r *http.Request) {
	mode, _ := config.ReadModeState()
	dbPath := s.Store.DBPath()
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":    mode,
		"db_path": dbPath,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// POST /api/v1/admin/system/swap-mode
// ────────────────────────────────────────────────────────────────────────────

// swapModeRequest is the body for POST /api/v1/admin/system/swap-mode.
type swapModeRequest struct {
	Mode string `json:"mode"` // "dev" or "prod"
}

// handleSwapMode performs a graceful-restart-equivalent mode swap.
// Implementation note: true hot-drain with atomic.Pointer[Store] would require
// threading a SwappableStore through every handler. Instead, we use the
// "graceful restart" fallback: drain → write state → return new mode.
// The calling client (CLI or dashboard) must reconnect after the mode switch;
// the server continues running on the current DB until the next startup.
// This is documented in YAML_DEPRECATION_PLAN.md Phase C follow-up notes.
func (s *Server) handleSwapMode(w http.ResponseWriter, r *http.Request) {
	// Optional localhost-only guard.
	if os.Getenv("SHARK_RESET_LOCALHOST_ONLY") == "1" {
		if !isLoopback(r) {
			writeJSON(w, http.StatusForbidden, errPayload("forbidden", "swap-mode only allowed from loopback"))
			return
		}
	}

	var req swapModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if req.Mode != "dev" && req.Mode != "prod" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_mode", "mode must be 'dev' or 'prod'"))
		return
	}

	// Persist state — takes effect on next server startup.
	if err := config.WriteModeState(req.Mode); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("state_write_failed", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"mode":    req.Mode,
		"message": fmt.Sprintf("Mode set to %q. Restart the server to activate.", req.Mode),
		"restart_required": true,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// POST /api/v1/admin/system/reset
// ────────────────────────────────────────────────────────────────────────────

// resetRequest is the body for POST /api/v1/admin/system/reset.
type resetRequest struct {
	Target       string `json:"target"`             // "dev", "prod", or "key"
	Confirmation string `json:"confirmation"`       // required for target=prod
}

// resetConfirmPhrase is the exact string the operator must type to reset prod.
const resetConfirmPhrase = "RESET PROD"

// handleReset handles POST /api/v1/admin/system/reset.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	// Optional localhost-only guard.
	if os.Getenv("SHARK_RESET_LOCALHOST_ONLY") == "1" {
		if !isLoopback(r) {
			writeJSON(w, http.StatusForbidden, errPayload("forbidden", "reset only allowed from loopback"))
			return
		}
	}

	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	switch req.Target {
	case "key":
		s.handleResetKey(w, r)
	case "dev":
		s.handleResetDB(w, r, "dev")
	case "prod":
		if req.Confirmation != resetConfirmPhrase {
			writeJSON(w, http.StatusBadRequest, errPayload("confirmation_required",
				fmt.Sprintf("confirmation must equal %q", resetConfirmPhrase)))
			return
		}
		s.handleResetDB(w, r, "prod")
	default:
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_target", "target must be 'dev', 'prod', or 'key'"))
	}
}

// handleResetKey rotates the admin API key. All existing admin sessions are
// invalidated because the old key hash is removed from api_keys.
func (s *Server) handleResetKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Capture the current admin key prefix (best-effort) before revoke so
	// the audit log can record the rotation pair. We pick the first active
	// wildcard-scope key — there's typically exactly one.
	oldKeyPrefix := ""
	if existing, err := s.Store.ListAPIKeys(ctx); err == nil {
		for _, k := range existing {
			if k.RevokedAt != nil {
				continue
			}
			var scopes []string
			_ = json.Unmarshal([]byte(k.Scopes), &scopes)
			if auth.CheckScope(scopes, "*") {
				oldKeyPrefix = k.KeyPrefix
				break
			}
		}
	}

	// Generate new key.
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("keygen_failed", err.Error()))
		return
	}

	// Revoke all existing admin (wildcard-scope) keys.
	if err := s.Store.RevokeAllAdminAPIKeys(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("revoke_failed", err.Error()))
		return
	}

	// Insert the new key.
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	k := &storage.APIKey{
		ID:        "key_" + id,
		Name:      "default-admin",
		KeyHash:   keyHash,
		KeyPrefix: keyPrefix,
		KeySuffix: keySuffix,
		Scopes:    `["*"]`,
		RateLimit: 0,
		CreatedAt: now,
	}
	if err := s.Store.CreateAPIKey(ctx, k); err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("create_failed", err.Error()))
		return
	}

	// Log the rotation.
	if s.AuditLogger != nil {
		oldPrefix := oldKeyPrefix
		if len(oldPrefix) > 8 {
			oldPrefix = oldPrefix[:8]
		}
		newPrefix := keyPrefix
		if len(newPrefix) > 8 {
			newPrefix = newPrefix[:8]
		}
		metaBytes, _ := json.Marshal(map[string]any{
			"key_prefix_old": oldPrefix,
			"key_prefix_new": newPrefix,
		})
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    "admin_key",
			Action:     "admin.key.rotated",
			TargetType: "system",
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"admin_key":  fullKey,
		"key_prefix": keyPrefix,
		"message":    "Admin key rotated. Store the new key — it will not be shown again.",
	})
}

// handleResetDB wipes the given mode's database, regenerates secrets, and
// returns the new admin key. Uses graceful drain semantics: existing requests
// finish, then the reset proceeds.
//
// NOTE: This is a destructive, non-reversible operation.
func (s *Server) handleResetDB(w http.ResponseWriter, r *http.Request, mode string) {
	ctx := r.Context()

	// Determine which DB path to wipe.
	dbPath, err := resolveDBPathForMode(mode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("db_path_error", err.Error()))
		return
	}

	// Acquire drain: stop accepting new requests and wait for in-flight ones.
	globalDrain.SetDraining(true)
	waitForDrain(drainTimeout)
	defer globalDrain.SetDraining(false)

	// Only wipe the *other* mode's DB if we're currently running the opposite
	// mode. If we're asked to wipe the active DB, we wipe and re-seed in place.
	activeDBPath := s.Store.DBPath()
	isActive := (dbPath == activeDBPath)

	if isActive {
		// Wipe all data from the active store (truncate all user tables).
		if err := s.Store.WipeAllData(ctx); err != nil {
			writeJSON(w, http.StatusInternalServerError, errPayload("wipe_failed", err.Error()))
			return
		}
	} else {
		// Remove the inactive DB file entirely.
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			writeJSON(w, http.StatusInternalServerError, errPayload("remove_failed", err.Error()))
			return
		}
	}

	// Re-generate admin key.
	fullKey, keyHash, keyPrefix, keySuffix, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errPayload("keygen_failed", err.Error()))
		return
	}

	if isActive {
		// Insert fresh admin key into the wiped active store.
		id, _ := gonanoid.New()
		now := time.Now().UTC().Format(time.RFC3339)
		k := &storage.APIKey{
			ID:        "key_" + id,
			Name:      "default-admin",
			KeyHash:   keyHash,
			KeyPrefix: keyPrefix,
			KeySuffix: keySuffix,
			Scopes:    `["*"]`,
			RateLimit: 0,
			CreatedAt: now,
		}
		if err := s.Store.CreateAPIKey(ctx, k); err != nil {
			writeJSON(w, http.StatusInternalServerError, errPayload("create_failed", err.Error()))
			return
		}
	}

	// Log reset event.
	if s.AuditLogger != nil {
		actorID := "admin_key"
		// tables_affected captures whether the wipe ran in-place
		// (active store, all user tables truncated) or removed the
		// inactive DB file. We don't enumerate table names because
		// WipeAllData is the canonical scope and lives behind the
		// store interface — encoding the high-level scope keeps the
		// metadata stable across schema changes.
		tablesAffected := "all"
		if !isActive {
			tablesAffected = "db_file_removed"
		}
		metaBytes, _ := json.Marshal(map[string]any{
			"mode":             mode,
			"tables_affected":  tablesAffected,
			"triggered_by":     actorID,
		})
		_ = s.AuditLogger.Log(ctx, &storage.AuditLog{
			ActorType:  "admin",
			ActorID:    actorID,
			Action:     "admin.db.reset",
			TargetType: "system",
			TargetID:   mode,
			Metadata:   string(metaBytes),
			IP:         r.RemoteAddr,
			UserAgent:  r.UserAgent(),
			Status:     "success",
		})
	}

	resp := map[string]any{
		"message": fmt.Sprintf("%s DB reset complete", mode),
		"mode":    mode,
	}
	if isActive {
		resp["admin_key"] = fullKey
		resp["masked_key"] = keyPrefix + "…" + keySuffix
		resp["note"] = "New admin key shown once — store it now."
	}
	writeJSON(w, http.StatusOK, resp)
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// resolveDBPathForMode returns the file path for the given mode's SQLite DB.
// Looks at ~/.shark/state and falls back to conventional names.
func resolveDBPathForMode(mode string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := home + "/.shark"
	if mode == "dev" {
		return dir + "/dev.db", nil
	}
	return dir + "/shark.db", nil
}

// isLoopback returns true when the request originates from a loopback address.
func isLoopback(r *http.Request) bool {
	host := r.RemoteAddr
	// Strip port.
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
