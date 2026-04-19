package oauth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ory/fosite"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// HandleToken handles POST /oauth/token. Fosite dispatches the correct grant
// type handler (authorization_code, client_credentials, refresh_token)
// automatically based on the request parameters.
func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := s.newSession("")

	ar, err := s.Provider.NewAccessRequest(ctx, r, session)
	if err != nil {
		slog.Debug("oauth: token request failed", "error", err)
		s.Provider.WriteAccessError(ctx, w, ar, err)
		return
	}

	// Grant all requested scopes. Scope filtering is done at the agent
	// registration level (the agent's allowed scopes are enforced by fosite
	// via the Client interface).
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}

	response, err := s.Provider.NewAccessResponse(ctx, ar)
	if err != nil {
		slog.Debug("oauth: access response failed", "error", err)
		s.Provider.WriteAccessError(ctx, w, ar, err)
		return
	}

	s.Provider.WriteAccessResponse(ctx, w, ar, response)
}

// HandleAuthorize handles GET /oauth/authorize -- the authorization endpoint.
// For now this implements a simplified flow:
//  1. Parses the authorize request via fosite
//  2. Checks for a logged-in user via session middleware context
//  3. If not logged in, returns 401 with login redirect info
//  4. If logged in, checks for existing consent
//  5. If consent exists, auto-approves
//  6. If no consent, returns JSON with consent info (full HTML template comes later)
func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ar, err := s.Provider.NewAuthorizeRequest(ctx, r)
	if err != nil {
		slog.Debug("oauth: authorize request parse failed", "error", err)
		s.Provider.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	// Check if user is logged in via session middleware.
	userID := getUserIDFromRequest(r)
	if userID == "" {
		// Not logged in. Return JSON directing the client to the login page.
		loginURL := s.Issuer + "/login?return_to=" + r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{ //#nosec G104
			"error":     "login_required",
			"login_url": loginURL,
		})
		return
	}

	// Check existing consent for this user + client + scopes.
	clientID := ar.GetClient().GetID()
	consent, err := s.RawStore.GetActiveConsent(ctx, userID, clientID)
	if err == nil && consent != nil {
		// Check if all requested scopes are covered by the existing consent.
		requestedScopes := ar.GetRequestedScopes()
		consentScopeSet := make(map[string]bool)
		for _, sc := range strings.Split(consent.Scope, " ") {
			consentScopeSet[sc] = true
		}

		allCovered := true
		for _, sc := range requestedScopes {
			if !consentScopeSet[string(sc)] {
				allCovered = false
				break
			}
		}

		if allCovered {
			s.completeAuthorize(w, r, ar, userID)
			return
		}
	}

	// No consent yet (or new scopes requested). Render the HTML consent page.
	scopes := make([]string, 0, len(ar.GetRequestedScopes()))
	for _, sc := range ar.GetRequestedScopes() {
		scopes = append(scopes, string(sc))
	}

	// Use the client name if available; fall back to the client ID.
	agentName := clientID
	if named, ok := ar.GetClient().(interface{ GetName() string }); ok {
		if n := named.GetName(); n != "" {
			agentName = n
		}
	}

	RenderConsentPage(w, ConsentData{
		AgentName:   agentName,
		ClientID:    clientID,
		Scopes:      scopes,
		RedirectURI: ar.GetRedirectURI().String(),
		State:       r.URL.Query().Get("state"),
		Challenge:   r.URL.RawQuery, // carry full query string for reconstruct on POST
		Issuer:      s.Issuer,
	})
}

// HandleAuthorizeDecision handles POST /oauth/authorize -- the consent decision.
func (s *Server) HandleAuthorizeDecision(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse the POST body so we can read form fields.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// The consent form posts the original authorize request query string back
	// in the "challenge" field. Reconstruct a synthetic GET request so fosite
	// can re-validate the same authorize request.
	challenge := r.FormValue("challenge")
	syntheticReq := r.Clone(ctx)
	syntheticReq.Method = http.MethodGet
	syntheticReq.Form = nil
	syntheticReq.PostForm = nil
	syntheticReq.Body = http.NoBody
	if challenge != "" {
		syntheticReq.URL = r.URL.ResolveReference(&url.URL{RawQuery: challenge})
	}

	ar, err := s.Provider.NewAuthorizeRequest(ctx, syntheticReq)
	if err != nil {
		slog.Debug("oauth: authorize decision parse failed", "error", err)
		s.Provider.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	userID := getUserIDFromRequest(r)
	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{ //#nosec G104
			"error": "login_required",
		})
		return
	}

	// Check the decision.
	approved := r.FormValue("approved")
	if approved != "true" {
		s.Provider.WriteAuthorizeError(ctx, w, ar,
			fosite.ErrRequestForbidden.WithHint("The resource owner denied the request."))
		return
	}

	// Store consent.
	clientID := ar.GetClient().GetID()
	scopeStr := strings.Join([]string(ar.GetRequestedScopes()), " ")
	consent := &storage.OAuthConsent{
		ID:        "consent_" + clientID + "_" + userID,
		UserID:    userID,
		ClientID:  clientID,
		Scope:     scopeStr,
		GrantedAt: time.Now().UTC(),
	}
	// Ignore error if consent already exists (idempotent).
	_ = s.RawStore.CreateOAuthConsent(ctx, consent)

	s.completeAuthorize(w, r, ar, userID)
}

// completeAuthorize finishes the authorization flow: grants scopes, creates
// the session, and writes the authorize response (redirect with code).
func (s *Server) completeAuthorize(w http.ResponseWriter, r *http.Request, ar fosite.AuthorizeRequester, userID string) {
	ctx := r.Context()

	// Grant all requested scopes.
	for _, scope := range ar.GetRequestedScopes() {
		ar.GrantScope(scope)
	}

	session := s.newSession(userID)
	response, err := s.Provider.NewAuthorizeResponse(ctx, ar, session)
	if err != nil {
		slog.Debug("oauth: authorize response failed", "error", err)
		s.Provider.WriteAuthorizeError(ctx, w, ar, err)
		return
	}

	s.Provider.WriteAuthorizeResponse(ctx, w, ar, response)
}

// getUserIDFromRequest extracts the user ID from the request context.
// Uses the session middleware's GetUserID helper, with a fallback to X-User-ID
// header for testing / internal calls.
func getUserIDFromRequest(r *http.Request) string {
	if uid := mw.GetUserID(r.Context()); uid != "" {
		return uid
	}
	// Fallback: X-User-ID header (for testing).
	return r.Header.Get("X-User-ID")
}
