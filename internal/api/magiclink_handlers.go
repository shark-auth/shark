package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sharkauth/sharkauth/internal/auth"
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
	return &magicLinkRateLimiter{
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
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
			log.Printf("ERROR: failed to send magic link email to %s: %v", req.Email, err)
		}
	}

	// Always return success
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists, a magic link has been sent",
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

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	// Redirect to the configured redirect URL
	redirectURL := s.Config.MagicLink.RedirectURL
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Fallback: return JSON with user info
	writeJSON(w, http.StatusOK, userToResponse(user))
}
