package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sharkauth/sharkauth/internal/auth"
	"github.com/sharkauth/sharkauth/internal/auth/redirect"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Re-export magic link errors for convenience.
var (
	ErrMagicLinkNotFound = auth.ErrMagicLinkNotFound
	ErrMagicLinkUsed     = auth.ErrMagicLinkUsed
	ErrMagicLinkExpired  = auth.ErrMagicLinkExpired
)

// magicLinkSendRequest is the request body for POST /api/v1/auth/magic-link/send.
type magicLinkSendRequest struct {
	Email string `json:"email"`
}

// magicLinkRateLimiter tracks per-email rate limits for magic link sends.
type magicLinkRateLimiter struct {
	mu       sync.Mutex
	lastSent map[string]time.Time
	cooldown time.Duration
}

func newMagicLinkRateLimiter(cooldown time.Duration) *magicLinkRateLimiter {
	rl := &magicLinkRateLimiter{
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
	// Clean up stale entries every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()
	return rl
}

// allow returns true if the email is allowed to send a magic link (not rate limited).
func (rl *magicLinkRateLimiter) allow(emailAddr string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	last, ok := rl.lastSent[emailAddr]
	if ok && time.Since(last) < rl.cooldown {
		return false
	}
	rl.lastSent[emailAddr] = time.Now()
	return true
}

// cleanup removes entries older than 2x the cooldown period.
func (rl *magicLinkRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-2 * rl.cooldown)
	for email, last := range rl.lastSent {
		if last.Before(cutoff) {
			delete(rl.lastSent, email)
		}
	}
}

// handleMagicLinkSend handles POST /api/v1/auth/magic-link/send.
func (s *Server) handleMagicLinkSend(w http.ResponseWriter, r *http.Request) {
	var req magicLinkSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_request",
			"message": "Invalid JSON body",
		})
		return
	}

	// Normalize and validate email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !emailRegex.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_email",
			"message": "Invalid email address",
		})
		return
	}

	// Rate limit: 1 magic link per email per 60 seconds
	if s.magicLinkRL != nil && !s.magicLinkRL.allow(req.Email) {
		// Still return 200 to not leak timing info, but don't actually send
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "If an account exists, a magic link has been sent",
		})
		return
	}

	// Send magic link (always return success to avoid leaking info about email existence)
	if s.MagicLinkManager != nil {
		if err := s.MagicLinkManager.SendMagicLink(r.Context(), req.Email); err != nil {
			slog.Error("failed to send magic link email", "email", req.Email, "error", err)
		}
	}

	// Always return success
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "The magic link has been sent",
	})
}

// handleMagicLinkVerify handles GET /api/v1/auth/magic-link/verify.
func (s *Server) handleMagicLinkVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_token",
			"message": "Token parameter is required",
		})
		return
	}

	if s.MagicLinkManager == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error":   "not_configured",
			"message": "Magic links are not configured",
		})
		return
	}

	user, sess, err := s.MagicLinkManager.VerifyMagicLink(r.Context(), token)
	if err != nil {
		switch err {
		case ErrMagicLinkNotFound:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "invalid_token",
				"message": "Invalid or expired magic link",
			})
		case ErrMagicLinkUsed:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "token_used",
				"message": "This magic link has already been used",
			})
		case ErrMagicLinkExpired:
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "token_expired",
				"message": "This magic link has expired",
			})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error":   "internal_error",
				"message": "Internal server error",
			})
		}
		return
	}

	// Phase 6 F3: fire auth flow hook. Token has been consumed; a block
	// here withholds the session cookie so the caller must retry with a
	// fresh magic link to get past the flow requirement.
	if s.runAuthFlow(w, r, storage.AuthFlowTriggerMagicLink, user, "") {
		return
	}

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	// Redirect to the configured redirect URL — JWT is additive but not embedded in redirect.
	// Validate redirect_uri against the default application's allowlist (OAuth 2.1 §3.1.2).
	{
		requestedRedirect := r.URL.Query().Get("redirect_uri")
		if requestedRedirect == "" {
			requestedRedirect = s.Config.MagicLink.RedirectURL
		}
		if requestedRedirect != "" {
			defaultApp, appErr := s.Store.GetDefaultApplication(r.Context())
			if appErr != nil {
				if errors.Is(appErr, sql.ErrNoRows) {
					writeJSON(w, http.StatusInternalServerError, map[string]string{
						"error":   "server_error",
						"message": "Default application not configured",
					})
					return
				}
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error":   "server_error",
					"message": "Could not load application config",
				})
				return
			}
			if verr := redirect.Validate(&redirect.Application{
				AllowedCallbackURLs: defaultApp.AllowedCallbackURLs,
			}, redirect.KindCallback, requestedRedirect); verr != nil {
				http.Error(w, "redirect_uri not allowed: "+verr.Error(), http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, requestedRedirect, http.StatusFound)
			return
		}
	}

	// Issue JWT alongside cookie if enabled (§1.4). Magic links set mfaPassed=true.
	resp := map[string]interface{}{}
	for k, v := range userResponseMap(userToResponse(user)) {
		resp[k] = v
	}
	if s.JWTManager != nil && s.Config.Auth.JWT.Enabled {
		if s.Config.Auth.JWT.Mode == "access_refresh" {
			access, refresh, err := s.JWTManager.IssueAccessRefreshPair(r.Context(), user, sess.ID, true)
			if err == nil {
				resp["access_token"] = access
				resp["refresh_token"] = refresh
			}
		} else {
			token, err := s.JWTManager.IssueSessionJWT(r.Context(), user, sess.ID, true)
			if err == nil {
				resp["token"] = token
			}
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
