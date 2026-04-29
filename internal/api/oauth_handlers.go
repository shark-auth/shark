package api

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/shark-auth/shark/internal/auth"
	"github.com/shark-auth/shark/internal/auth/providers"
	"github.com/shark-auth/shark/internal/auth/redirect"
	"github.com/shark-auth/shark/internal/storage"
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

	// Store state in a short-lived cookie. Secure mirrors the session cookie:
	// set whenever base_url is https so MITM on plain HTTP can't read state.
	//#nosec G124 -- Secure is dynamic (tied to base_url scheme); hardcoding true breaks local http dev
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionManager.SecureCookies(),
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
	if queryState == "" || subtle.ConstantTimeCompare([]byte(queryState), []byte(stateCookie.Value)) != 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "invalid_state",
			"message": "OAuth state mismatch",
		})
		return
	}

	// Clear the state cookie (mirror the Secure flag from the set path so the
	// browser recognizes it as the same cookie on cleanup).
	//#nosec G124 -- Secure is dynamic (tied to base_url scheme); hardcoding true breaks local http dev
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionManager.SecureCookies(),
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

	// Phase 6 F3: fire auth flow hook. OAuthManager.HandleCallback already
	// created a session row for us; if the flow blocks/redirects we leave
	// that session in place â€” the cookie below is the caller-visible gate
	// and we skip it on a handled outcome.
	if s.runAuthFlow(w, r, storage.AuthFlowTriggerOAuthCallback, user, "") {
		return
	}

	// Set session cookie
	s.SessionManager.SetSessionCookie(w, sess.ID)

	// Redirect to frontend if configured â€” JWT is additive but not embedded in redirect URL.
	// Validate redirect_uri against the default application's allowlist (OAuth 2.1 Â§3.1.2).
	{
		requestedRedirect := r.URL.Query().Get("redirect_uri")
		if requestedRedirect == "" {
			requestedRedirect = s.Config.Social.RedirectURL
		}
		if requestedRedirect != "" {
			defaultApp, appErr := s.Store.GetDefaultApplication(r.Context())
			if appErr != nil {
				if errors.Is(appErr, sql.ErrNoRows) {
					// No default app â€” server misconfiguration, not client error.
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
				WriteError(w, http.StatusBadRequest,
					NewError(CodeInvalidRequest, "redirect_uri not allowed: "+verr.Error()).
						WithDocsURL(CodeInvalidRequest))
				return
			}
			http.Redirect(w, r, requestedRedirect, http.StatusFound)
			return
		}
	}

	// Issue JWT alongside cookie if enabled (Â§1.4).
	// OAuth callback is cookie-based (no MFA gate); mfaPassed=true for social logins.
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

// initOAuthManager creates the OAuthManager and registers configured providers.
func (s *Server) initOAuthManager() {
	s.OAuthManager = auth.NewOAuthManager(s.Store, s.SessionManager, s.Config)

	baseURL := s.Config.Server.BaseURL

	// Register Google if configured
	if s.Config.Social.Google.ClientID != "" && s.Config.Social.Google.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewGoogle(s.Config.Social.Google, baseURL))
	}

	// Register GitHub if configured
	if s.Config.Social.GitHub.ClientID != "" && s.Config.Social.GitHub.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewGitHub(s.Config.Social.GitHub, baseURL))
	}

	// Register Apple if configured
	if s.Config.Social.Apple.ClientID != "" && s.Config.Social.Apple.TeamID != "" {
		s.OAuthManager.RegisterProvider(providers.NewApple(s.Config.Social.Apple, baseURL))
	}

	// Register Discord if configured
	if s.Config.Social.Discord.ClientID != "" && s.Config.Social.Discord.ClientSecret != "" {
		s.OAuthManager.RegisterProvider(providers.NewDiscord(s.Config.Social.Discord, baseURL))
	}
}
