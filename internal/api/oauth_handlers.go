package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sharkauth/sharkauth/internal/auth"
)

const (
	oauthStateCookieName = "shark_oauth_state"
	oauthStateTTL        = 5 * time.Minute
)

// handleOAuthStart initiates the OAuth flow by generating a random state,
// storing it in a short-lived cookie, and redirecting to the provider's auth URL.
func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	provider, err := s.OAuthManager.GetProvider(providerName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_provider",
			"message": "Unsupported OAuth provider: " + providerName,
		})
		return
	}

	// Generate cryptographic random state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": "Failed to generate state",
		})
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in a short-lived cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(oauthStateTTL.Seconds()),
	})

	http.Redirect(w, r, provider.AuthURL(state), http.StatusFound)
}

// handleOAuthCallback handles the provider's callback, validates state,
// exchanges the code, finds or creates the user, and sets a session cookie.
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")

	// Check for error from provider
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		desc := r.URL.Query().Get("error_description")
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "oauth_error",
			"message": errMsg + ": " + desc,
		})
		return
	}

	// Validate state
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_state",
			"message": "Missing OAuth state cookie",
		})
		return
	}

	queryState := r.URL.Query().Get("state")
	if queryState == "" || queryState != stateCookie.Value {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_state",
			"message": "OAuth state mismatch",
		})
		return
	}

	// Clear the state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "missing_code",
			"message": "No authorization code in callback",
		})
		return
	}

	// Exchange code, find/create user, link account, create session
	user, sess, err := s.OAuthManager.HandleCallback(r.Context(), providerName, code, r.RemoteAddr, r.UserAgent())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error":   "oauth_callback_failed",
			"message": "OAuth authentication failed",
		})
		return
	}

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	// If there's a configured frontend redirect, send them there
	_ = user // user available for future use (e.g., redirect with user info)

	writeJSON(w, http.StatusOK, userToResponse(user))
}

// initOAuthManager creates the OAuthManager and registers configured providers.
func (s *Server) initOAuthManager() {
	s.OAuthManager = auth.NewOAuthManager(s.Store, s.SessionManager, s.Config)
}
