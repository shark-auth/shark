package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ory/fosite"
	"github.com/ory/fosite/token/jwt"

	mw "github.com/shark-auth/shark/internal/api/middleware"
	"github.com/shark-auth/shark/internal/storage"
)

// newUUID wraps uuid.New().String() so the call site stays compact.
func newUUID() string { return uuid.New().String() }

// dpopTokenEndpointURL returns the canonical HTU for DPoP proof validation at
// the token endpoint. Query parameters and fragments are stripped.
func dpopTokenEndpointURL(r *http.Request) string {
	scheme := r.URL.Scheme
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}
	return scheme + "://" + host + r.URL.Path
}

// HandleToken handles POST /oauth/token. Fosite dispatches the correct grant
// type handler (authorization_code, client_credentials, refresh_token)
// automatically based on the request parameters.
// Device Authorization Grant (RFC 8628) is intercepted and handled manually
// before passing to fosite, since fosite v0.49 has no built-in device flow.
func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	// Device Authorization Grant (RFC 8628) is disabled for v0.1 â€” coming v0.2.
	// Return unsupported_grant_type so clients get a clear, spec-compliant error.
	// Defense-in-depth â€” routes are unmounted in v0.1, but kept here in case they come back.
	if r.FormValue("grant_type") == "urn:ietf:params:oauth:grant-type:device_code" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"error":             "unsupported_grant_type",
			"error_description": "Device authorization grant (RFC 8628) is not yet supported. Coming in v0.2.",
		})
		return
	}

	// Intercept RFC 8693 token-exchange before fosite sees it.
	// fosite v0.49 has no built-in token-exchange handler.
	if r.FormValue("grant_type") == grantTypeTokenExchange {
		s.HandleTokenExchange(w, r)
		return
	}

	// Intercept DPoP before passing to fosite. If the DPoP header is present,
	// validate the proof and record the jkt for later storage.
	var dpopJKT string
	if proofJWT := r.Header.Get("DPoP"); proofJWT != "" {
		htu := dpopTokenEndpointURL(r)
		jkt, dpopErr := ValidateDPoPProof(r.Context(), proofJWT, r.Method, htu, "", s.DPoPCache)
		if dpopErr != nil {
			slog.Debug("oauth: DPoP proof invalid", "error", dpopErr)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{ //#nosec G104
				"error":             "invalid_dpop_proof",
				"error_description": dpopErr.Error(),
			})
			return
		}
		dpopJKT = jkt
		slog.Debug("oauth: DPoP proof validated", "jkt", jkt)
	}

	ctx := r.Context()

	// RFC 8707: extract resource indicator before fosite sanitizes the form.
	// Fosite's Sanitize() strips unrecognized params, so we pass resource via
	// context so createTokenSession can pick it up.
	if resource := r.FormValue("resource"); resource != "" {
		ctx = contextWithResource(ctx, resource)
	}

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

	// DX1: ensure the JWT access token has a meaningful aud claim. fosite
	// only fills GrantedAudience from the explicit `resource` (RFC 8707) or
	// `audience` (RFC 8693) request params. When the caller didn't request
	// any, fall back to the client_id â€” same convention Auth0/Okta use for
	// machine-to-machine tokens â€” so SDKs can verify aud without bespoke
	// config.
	if len(ar.GetGrantedAudience()) == 0 {
		for _, aud := range ar.GetRequestedAudience() {
			ar.GrantAudience(aud)
		}
		if len(ar.GetGrantedAudience()) == 0 {
			if cid := ar.GetClient().GetID(); cid != "" {
				ar.GrantAudience(cid)
			}
		}
	}

	// DX1: enrich the JWT session with client_id and (if present) cnf.jkt
	// before the JWT access-token is signed. These claims land in the
	// RFC 7519 token body and are verifiable by any SDK using
	// decode_agent_token + /.well-known/jwks.json.
	if sharkSess, ok := ar.GetSession().(*SharkSession); ok && sharkSess != nil {
		claims := sharkSess.GetJWTClaims()
		if jc, ok := claims.(*jwt.JWTClaims); ok {
			if jc.Extra == nil {
				jc.Extra = map[string]interface{}{}
			}
			if cid := ar.GetClient().GetID(); cid != "" {
				jc.Extra["client_id"] = cid
			}
			// For client_credentials, subject is typically the client itself.
			// We populate this on the JWT claims only â€” NOT the DefaultSession
			// Subject, which the FositeStore persists into oauth_tokens.user_id
			// (a FK into users). Setting client_id there would FK-fail.
			if jc.Subject == "" {
				jc.Subject = ar.GetClient().GetID()
			}
			if dpopJKT != "" {
				jc.Extra["cnf"] = map[string]interface{}{"jkt": dpopJKT}
			}
			// DX1: pin the JTI so the on-the-wire `jti` claim matches the
			// DB row our FositeStore persists â€” lets introspection/revocation
			// look up tokens by JTI directly.
			if jc.JTI == "" {
				jc.JTI = "jti_" + newUUID()
			}
		}
	}

	response, err := s.Provider.NewAccessResponse(ctx, ar)
	if err != nil {
		slog.Debug("oauth: access response failed", "error", err)
		s.Provider.WriteAccessError(ctx, w, ar, err)
		return
	}

	// If a DPoP proof was validated, store the jkt on the token record.
	// This is a best-effort background operation â€” it does not affect the
	// response because fosite has already committed the token to its own store.
	if dpopJKT != "" {
		s.storeDPoPJKT(ctx, ar, dpopJKT)
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
		// Not logged in. Redirect to the hosted login page for this application.
		// We resolve the slug via the client_id (ar.GetClient().GetID()).
		clientID := ar.GetClient().GetID()
		app, err := s.RawStore.GetApplicationByClientID(ctx, clientID)
		slug := "default"
		if err == nil && app != nil {
			slug = app.Slug
		}

		// Use issuer base URL to build the redirect.
		loginURL := fmt.Sprintf("%s/hosted/%s/login?client_id=%s&redirect_uri=%s&state=%s&scope=%s&return_to=%s",
			s.Issuer,
			slug,
			url.QueryEscape(clientID),
			url.QueryEscape(ar.GetRedirectURI().String()),
			url.QueryEscape(r.URL.Query().Get("state")),
			url.QueryEscape(strings.Join(ar.GetRequestedScopes(), " ")),
			url.QueryEscape(r.URL.String()),
		)

		http.Redirect(w, r, loginURL, http.StatusFound)
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

	// RFC 8707: extract resource indicator for display on the consent page.
	resource := r.URL.Query().Get("resource")
	if resource != "" {
		slog.Debug("oauth: authorize request includes resource indicator", "resource", resource, "client_id", clientID)
	}

	RenderConsentPage(w, ConsentData{
		AgentName:   agentName,
		ClientID:    clientID,
		Scopes:      scopes,
		Resource:    resource,
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
		WriteOAuthError(w, http.StatusBadRequest,
			NewOAuthError(ErrInvalidRequest, "malformed authorize decision body"))
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
		// RFC 6749 Â§5.2 shape. Any "where to log in" hint goes on the URI,
		// never as an extra top-level field â€” client libs reject unknowns.
		WriteOAuthError(w, http.StatusUnauthorized,
			NewOAuthError(ErrLoginRequired, "End-user authentication is required"))
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
	// Ignore duplicate consent (idempotent); log non-duplicate failures so audit
	// drift is visible â€” the user already consented so we still complete the flow.
	if err := s.RawStore.CreateOAuthConsent(ctx, consent); err != nil {
		slog.Warn("oauth: failed to persist consent record", "user_id", userID, "client_id", clientID, "err", err)
	}

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

// getUserIDFromRequest extracts the user ID from the request context set by
// session middleware. Never trust client-controlled headers for identity.
func getUserIDFromRequest(r *http.Request) string {
	return mw.GetUserID(r.Context())
}

// storeDPoPJKT records the DPoP JWK thumbprint on the OAuthToken row created
// by fosite. We look up the exact token by the JTI we pinned into the session
// before NewAccessResponse, so the update is precise and never touches a
// different token for the same client (the old ListOAuthTokensByAgentID
// approach was lossy under concurrent requests).
//
// This is best-effort: a failure here does NOT fail the token request.
func (s *Server) storeDPoPJKT(ctx context.Context, ar fosite.AccessRequester, jkt string) {
	sharkSess, ok := ar.GetSession().(*SharkSession)
	if !ok || sharkSess == nil || sharkSess.JWTClaims == nil || sharkSess.JWTClaims.JTI == "" {
		// Fall back to client-scoped lookup when the JTI is unavailable
		// (e.g. HMAC strategy that doesn't use our SharkSession).
		clientID := ar.GetClient().GetID()
		tokens, err := s.RawStore.ListOAuthTokensByAgentID(ctx, "agent_"+clientID, 1)
		if err != nil || len(tokens) == 0 {
			slog.Debug("oauth: storeDPoPJKT: no token found for client", "client_id", clientID)
			return
		}
		if updateErr := s.RawStore.UpdateOAuthTokenDPoPJKT(ctx, tokens[0].ID, jkt); updateErr != nil {
			slog.Debug("oauth: storeDPoPJKT: update failed (fallback)", "error", updateErr)
		}
		return
	}

	// Happy path: look up by the exact JTI pinned in the session before signing.
	tok, err := s.RawStore.GetOAuthTokenByJTI(ctx, sharkSess.JWTClaims.JTI)
	if err != nil || tok == nil {
		slog.Debug("oauth: storeDPoPJKT: token not found by JTI", "jti", sharkSess.JWTClaims.JTI)
		return
	}
	if updateErr := s.RawStore.UpdateOAuthTokenDPoPJKT(ctx, tok.ID, jkt); updateErr != nil {
		slog.Debug("oauth: storeDPoPJKT: update failed", "jti", sharkSess.JWTClaims.JTI, "error", updateErr)
	}
}
