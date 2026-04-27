package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/vault"
)

// Audit event names emitted by the vault handlers. Centralised so the
// dashboard's audit-log filter can key on a stable vocabulary.
const (
	auditVaultProviderCreated   = "vault.provider.created"
	auditVaultProviderUpdated   = "vault.provider.updated"
	auditVaultProviderDeleted   = "vault.provider.deleted"
	auditVaultConnectionCreated = "vault.connection.created"
	auditVaultConnectionUpdated = "vault.connection.updated"
	auditVaultConnectionDeleted = "vault.connection.disconnected"
	auditVaultTokenRetrieved    = "vault.token.retrieved"
)

const (
	vaultStateCookieName = "shark_vault_state"
	vaultStateTTL        = 5 * time.Minute
)

// vaultProviderResponse mirrors VaultProvider minus the encrypted client_secret.
// We never let ciphertext or plaintext secrets leak out to API callers, even
// admin ones — rotation goes through a dedicated PATCH body field.
type vaultProviderResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	DisplayName     string            `json:"display_name"`
	AuthURL         string            `json:"auth_url"`
	TokenURL        string            `json:"token_url"`
	ClientID        string            `json:"client_id"`
	Scopes          []string          `json:"scopes"`
	IconURL         string            `json:"icon_url,omitempty"`
	Active          bool              `json:"active"`
	ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func sanitizeVaultProvider(p *storage.VaultProvider) vaultProviderResponse {
	if p == nil {
		return vaultProviderResponse{}
	}
	scopes := p.Scopes
	if scopes == nil {
		scopes = []string{}
	}
	extra := p.ExtraAuthParams
	if extra == nil {
		extra = map[string]string{}
	}
	return vaultProviderResponse{
		ID:              p.ID,
		Name:            p.Name,
		DisplayName:     p.DisplayName,
		AuthURL:         p.AuthURL,
		TokenURL:        p.TokenURL,
		ClientID:        p.ClientID,
		Scopes:          scopes,
		IconURL:         p.IconURL,
		Active:          p.Active,
		ExtraAuthParams: extra,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}
}

// vaultConnectionResponse hides all token material and enriches a row with
// provider display metadata the dashboard needs to render a tile.
type vaultConnectionResponse struct {
	ID                  string     `json:"id"`
	ProviderID          string     `json:"provider_id"`
	ProviderName        string     `json:"provider_name"`
	ProviderDisplayName string     `json:"provider_display_name"`
	ProviderIconURL     string     `json:"provider_icon_url,omitempty"`
	Scopes              []string   `json:"scopes"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	NeedsReauth         bool       `json:"needs_reauth"`
	LastRefreshedAt     *time.Time `json:"last_refreshed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// isHTTPSURL reports whether u is a valid http(s) absolute URL. We insist on
// https in production to ensure the redirect leg of OAuth can't be eavesdropped,
// but allow http for localhost so dev loops don't need TLS termination.
func isHTTPSURL(u string) bool {
	if u == "" {
		return false
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" {
		return false
	}
	return true
}

// auditVault centralises audit logging for vault ops. Best-effort — audit
// failures never block the request path.
func (s *Server) auditVault(r *http.Request, actorType, action, targetType, targetID string, meta map[string]any) {
	if s.AuditLogger == nil {
		return
	}
	var metaStr string
	if meta != nil {
		b, err := json.Marshal(meta)
		if err == nil {
			metaStr = string(b)
		}
	}
	actor := mw.GetUserID(r.Context())
	if actor == "" {
		actor = actorID(r.Context())
	}
	_ = s.AuditLogger.Log(r.Context(), &storage.AuditLog{
		ActorID:    actor,
		ActorType:  actorType,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IP:         ipOf(r),
		UserAgent:  uaOf(r),
		Metadata:   metaStr,
		Status:     "success",
	})
}

// --- Provider CRUD (admin) ---------------------------------------------------

// handleCreateVaultProvider handles POST /api/v1/vault/providers.
// Accepts either a template key (fills in auth/token URLs) or an explicit
// provider definition. Always requires plaintext client_id + client_secret.
func (s *Server) handleCreateVaultProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Template        string            `json:"template"`
		Name            string            `json:"name"`
		DisplayName     string            `json:"display_name"`
		AuthURL         string            `json:"auth_url"`
		TokenURL        string            `json:"token_url"`
		ClientID        string            `json:"client_id"`
		ClientSecret    string            `json:"client_secret"`
		Scopes          []string          `json:"scopes"`
		IconURL         string            `json:"icon_url"`
		ExtraAuthParams map[string]string `json:"extra_auth_params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	if req.ClientID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "client_id is required"))
		return
	}
	if req.ClientSecret == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "client_secret is required"))
		return
	}

	var provider *storage.VaultProvider
	if req.Template != "" {
		tpl, ok := vault.Template(req.Template)
		if !ok {
			writeJSON(w, http.StatusBadRequest, errPayload("unknown_template", "Unknown vault provider template: "+req.Template))
			return
		}
		provider = vault.ApplyTemplate(tpl, req.ClientID, req.DisplayName, req.Scopes)
		// Allow caller to override icon_url per-install (e.g. custom asset).
		if req.IconURL != "" {
			provider.IconURL = req.IconURL
		}
		// Merge request extra_auth_params over template defaults (per-key override).
		if len(req.ExtraAuthParams) > 0 {
			if provider.ExtraAuthParams == nil {
				provider.ExtraAuthParams = map[string]string{}
			}
			for k, v := range req.ExtraAuthParams {
				if v != "" {
					provider.ExtraAuthParams[k] = v
				}
			}
		}
	} else {
		if strings.TrimSpace(req.Name) == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "name is required when template is omitted"))
			return
		}
		if !isHTTPSURL(req.AuthURL) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "auth_url must be a valid http(s) URL"))
			return
		}
		if !isHTTPSURL(req.TokenURL) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "token_url must be a valid http(s) URL"))
			return
		}
		scopes := req.Scopes
		if scopes == nil {
			scopes = []string{}
		}
		displayName := req.DisplayName
		if displayName == "" {
			displayName = req.Name
		}
		extraParams := req.ExtraAuthParams
		if extraParams == nil {
			extraParams = map[string]string{}
		}
		provider = &storage.VaultProvider{
			Name:            strings.TrimSpace(req.Name),
			DisplayName:     displayName,
			AuthURL:         req.AuthURL,
			TokenURL:        req.TokenURL,
			ClientID:        req.ClientID,
			Scopes:          scopes,
			IconURL:         req.IconURL,
			Active:          true,
			ExtraAuthParams: extraParams,
		}
	}

	// Duplicate detection — storage layer's UNIQUE constraint would bubble up
	// as a generic error; we do the lookup first to return a clean 409.
	if existing, err := s.Store.GetVaultProviderByName(r.Context(), provider.Name); err == nil && existing != nil {
		writeJSON(w, http.StatusConflict, errPayload("name_exists", "A vault provider with this name already exists"))
		return
	}

	if err := s.VaultManager.CreateProvider(r.Context(), provider, req.ClientSecret); err != nil {
		// Best-effort duplicate recovery — if two creates race, the store can
		// still 500 the second; translate known duplicate markers.
		if isDuplicateErr(err) {
			writeJSON(w, http.StatusConflict, errPayload("name_exists", "A vault provider with this name already exists"))
			return
		}
		internal(w, err)
		return
	}

	// Audit: log param keys only — values could contain sensitive info.
	extraKeys := make([]string, 0, len(provider.ExtraAuthParams))
	for k := range provider.ExtraAuthParams {
		extraKeys = append(extraKeys, k)
	}
	s.auditVault(r, "admin", auditVaultProviderCreated, "vault_provider", provider.ID, map[string]any{
		"name":                 provider.Name,
		"extra_auth_param_keys": extraKeys,
	})

	writeJSON(w, http.StatusCreated, sanitizeVaultProvider(provider))
}

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	// SQLite reports "UNIQUE constraint failed"; surface it as 409 without
	// pulling in a dialect-specific dep.
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

// handleListVaultProviders handles GET /api/v1/vault/providers.
// Supports ?active_only=true for dashboards that only want to render
// connectable providers.
func (s *Server) handleListVaultProviders(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"
	providers, err := s.Store.ListVaultProviders(r.Context(), activeOnly)
	if err != nil {
		internal(w, err)
		return
	}
	out := make([]vaultProviderResponse, 0, len(providers))
	for _, p := range providers {
		out = append(out, sanitizeVaultProvider(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":  out,
		"total": len(out),
	})
}

// handleGetVaultProvider handles GET /api/v1/vault/providers/{id}.
func (s *Server) handleGetVaultProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.Store.GetVaultProviderByID(r.Context(), id)
	if err != nil || p == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not found"))
		return
	}
	writeJSON(w, http.StatusOK, sanitizeVaultProvider(p))
}

// handleUpdateVaultProvider handles PATCH /api/v1/vault/providers/{id}.
// All body fields are optional; only supplied fields are updated. When
// client_secret is present it goes through the Manager so the encryption
// boundary is preserved.
func (s *Server) handleUpdateVaultProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.Store.GetVaultProviderByID(r.Context(), id)
	if err != nil || p == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not found"))
		return
	}

	var req struct {
		DisplayName     *string           `json:"display_name,omitempty"`
		Scopes          []string          `json:"scopes,omitempty"`
		IconURL         *string           `json:"icon_url,omitempty"`
		Active          *bool             `json:"active,omitempty"`
		ClientSecret    *string           `json:"client_secret,omitempty"`
		AuthURL         *string           `json:"auth_url,omitempty"`
		TokenURL        *string           `json:"token_url,omitempty"`
		ClientID        *string           `json:"client_id,omitempty"`
		ExtraAuthParams map[string]string `json:"extra_auth_params,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}

	// Rotate secret via the manager so ciphertext never touches this handler.
	if req.ClientSecret != nil {
		if *req.ClientSecret == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "client_secret cannot be empty"))
			return
		}
		if err := s.VaultManager.UpdateProviderSecret(r.Context(), id, *req.ClientSecret); err != nil {
			if errors.Is(err, vault.ErrProviderNotFound) {
				writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not found"))
				return
			}
			internal(w, err)
			return
		}
		// Refresh the local copy to pick up the new UpdatedAt from the rotation.
		p, _ = s.Store.GetVaultProviderByID(r.Context(), id)
	}

	changed := false
	if req.DisplayName != nil {
		p.DisplayName = *req.DisplayName
		changed = true
	}
	if req.Scopes != nil {
		p.Scopes = req.Scopes
		changed = true
	}
	if req.IconURL != nil {
		p.IconURL = *req.IconURL
		changed = true
	}
	if req.Active != nil {
		p.Active = *req.Active
		changed = true
	}
	if req.AuthURL != nil {
		if *req.AuthURL == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "auth_url cannot be empty"))
			return
		}
		// SSRF guard: reject non-https URLs and bare-IP / metadata hosts on
		// PATCH, mirroring the create path. Without this, an admin could point
		// the provider at http://169.254.169.254 or other internal endpoints.
		if !isHTTPSURL(*req.AuthURL) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "auth_url must be https and not target internal/metadata hosts"))
			return
		}
		p.AuthURL = *req.AuthURL
		changed = true
	}
	if req.TokenURL != nil {
		if *req.TokenURL == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "token_url cannot be empty"))
			return
		}
		if !isHTTPSURL(*req.TokenURL) {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "token_url must be https and not target internal/metadata hosts"))
			return
		}
		p.TokenURL = *req.TokenURL
		changed = true
	}
	if req.ClientID != nil {
		if *req.ClientID == "" {
			writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "client_id cannot be empty"))
			return
		}
		p.ClientID = *req.ClientID
		changed = true
	}
	if req.ExtraAuthParams != nil {
		// Full replacement — PATCH sends the complete desired map.
		p.ExtraAuthParams = req.ExtraAuthParams
		changed = true
	}
	if changed {
		p.UpdatedAt = time.Now().UTC()
		if err := s.Store.UpdateVaultProvider(r.Context(), p); err != nil {
			internal(w, err)
			return
		}
	}

	// Audit: log param keys only — values could contain sensitive info.
	extraKeys := make([]string, 0, len(p.ExtraAuthParams))
	for k := range p.ExtraAuthParams {
		extraKeys = append(extraKeys, k)
	}
	s.auditVault(r, "admin", auditVaultProviderUpdated, "vault_provider", p.ID, map[string]any{
		"secret_rotated":       req.ClientSecret != nil,
		"extra_auth_param_keys": extraKeys,
	})
	writeJSON(w, http.StatusOK, sanitizeVaultProvider(p))
}

// handleDeleteVaultProvider handles DELETE /api/v1/vault/providers/{id}.
// The FK cascade wipes connections; we just need to verify existence first so
// 404 is returned on unknown IDs rather than silent success.
func (s *Server) handleDeleteVaultProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.Store.GetVaultProviderByID(r.Context(), id)
	if err != nil || p == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not found"))
		return
	}
	if err := s.Store.DeleteVaultProvider(r.Context(), id); err != nil {
		internal(w, err)
		return
	}
	s.auditVault(r, "admin", auditVaultProviderDeleted, "vault_provider", id, map[string]any{
		"name": p.Name,
	})
	w.WriteHeader(http.StatusNoContent)
}

// --- Templates (admin) -------------------------------------------------------

// handleListVaultTemplates returns the built-in provider catalog. This is a
// pure discovery endpoint for the dashboard "add provider" picker; it doesn't
// touch storage.
func (s *Server) handleListVaultTemplates(w http.ResponseWriter, _ *http.Request) {
	templates := vault.ListTemplates()
	writeJSON(w, http.StatusOK, map[string]any{"data": templates})
}

// --- Connect flow (session) --------------------------------------------------

// vaultStateValue packs the CSRF state + provider id in the cookie so the
// callback can recover the provider context without trusting the URL path.
// Format: "<state>:<provider_id>".
func vaultStateValue(state, providerID string) string { return state + ":" + providerID }

func parseVaultStateValue(v string) (state, providerID string, ok bool) {
	idx := strings.LastIndex(v, ":")
	if idx <= 0 || idx == len(v)-1 {
		return "", "", false
	}
	return v[:idx], v[idx+1:], true
}

func (s *Server) vaultRedirectURI(r *http.Request, providerName string) string {
	base := strings.TrimRight(s.Config.Server.BaseURL, "/")
	if base == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	return base + "/api/v1/vault/callback/" + url.PathEscape(providerName)
}

// handleVaultConnectStart initiates a vault OAuth flow for the authenticated
// user. Generates state, stashes it in a short-lived cookie keyed to the
// provider ID (so the callback can't be replayed against a different provider)
// and 302s to the provider's authorize endpoint.
func (s *Server) handleVaultConnectStart(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, err := s.Store.GetVaultProviderByName(r.Context(), providerName)
	if err != nil || provider == nil || !provider.Active {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not available"))
		return
	}

	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		internal(w, err)
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Optional scope override via ?scopes=a,b,c — empty means provider defaults.
	var scopesOverride []string
	if raw := r.URL.Query().Get("scopes"); raw != "" {
		for _, sc := range strings.Split(raw, ",") {
			sc = strings.TrimSpace(sc)
			if sc != "" {
				scopesOverride = append(scopesOverride, sc)
			}
		}
	}

	redirectURI := s.vaultRedirectURI(r, providerName)
	authURL, err := s.VaultManager.BuildAuthURL(r.Context(), provider.ID, state, redirectURI, scopesOverride)
	if err != nil {
		internal(w, err)
		return
	}

	//#nosec G124 -- Secure is dynamic (tied to base_url scheme); hardcoding true breaks local http dev
	http.SetCookie(w, &http.Cookie{
		Name:     vaultStateCookieName,
		Value:    vaultStateValue(state, provider.ID),
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionManager.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(vaultStateTTL.Seconds()),
	})

	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleVaultCallback completes the vault OAuth flow: validates state, swaps
// code for tokens, persists the connection, and redirects back to the
// dashboard with a success flag in the query string.
func (s *Server) handleVaultCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	userID := mw.GetUserID(r.Context())

	// Provider reported an error (user rejected consent, scope denied, etc.).
	// Mirror oauth_handlers.go: surface the provider-side message early so the
	// dashboard can show "connection cancelled" without dragging state-cookie
	// bookkeeping into the failure path.
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		desc := r.URL.Query().Get("error_description")
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "oauth_error",
			"message": errMsg + ": " + desc,
		})
		return
	}

	stateCookie, err := r.Cookie(vaultStateCookieName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_state", "Missing vault state cookie"))
		return
	}
	// Clear the cookie immediately so replays get rejected.
	//#nosec G124 -- Secure is dynamic (tied to base_url scheme); hardcoding true breaks local http dev
	http.SetCookie(w, &http.Cookie{
		Name:     vaultStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.SessionManager.SecureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	expectedState, cookieProviderID, ok := parseVaultStateValue(stateCookie.Value)
	if !ok {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_state", "Malformed vault state cookie"))
		return
	}
	queryState := r.URL.Query().Get("state")
	if queryState == "" || subtle.ConstantTimeCompare([]byte(queryState), []byte(expectedState)) != 1 {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_state", "Vault state mismatch"))
		return
	}

	provider, err := s.Store.GetVaultProviderByName(r.Context(), providerName)
	if err != nil || provider == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not available"))
		return
	}
	// Bind the provider ID carried in the cookie to the one in the path — stops
	// a cookie from one provider being replayed to steal consent from another.
	if subtle.ConstantTimeCompare([]byte(cookieProviderID), []byte(provider.ID)) != 1 {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_state", "Vault state provider mismatch"))
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("missing_code", "No authorization code in callback"))
		return
	}

	redirectURI := s.vaultRedirectURI(r, providerName)

	// Detect whether we're updating an existing connection before the upsert
	// so the audit event reflects reality.
	existing, _ := s.Store.GetVaultConnection(r.Context(), provider.ID, userID)
	isUpdate := existing != nil

	conn, err := s.VaultManager.ExchangeAndStore(r.Context(), provider.ID, userID, code, redirectURI)
	if err != nil {
		internal(w, err)
		return
	}

	// Atlassian post-exchange: discover cloud IDs so callers can construct
	// API URLs (https://api.atlassian.com/ex/jira/{cloudId}/rest/...).
	// Failure blocks the connection from being saved — a token without a
	// cloud ID is unusable for Jira Cloud.
	if provider.Name == "jira" {
		if metaErr := s.atlassianFetchCloudIDs(r.Context(), conn); metaErr != nil {
			// Roll back the connection so we don't strand a useless row.
			_ = s.VaultManager.Disconnect(r.Context(), conn.ID)
			writeJSON(w, http.StatusBadGateway, map[string]string{
				"error":   "jira_cloud_id_fetch_failed",
				"message": fmt.Sprintf("Jira accessible-resources lookup failed: %v", metaErr),
			})
			return
		}
	}

	action := auditVaultConnectionCreated
	if isUpdate {
		action = auditVaultConnectionUpdated
	}
	s.auditVault(r, "user", action, "vault_connection", conn.ID, map[string]any{
		"provider_id":   provider.ID,
		"provider_name": provider.Name,
	})

	// Simple success redirect — the dashboard can reload its connection list.
	base := strings.TrimRight(s.Config.Server.BaseURL, "/")
	if base == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	http.Redirect(w, r, base+"/vault?connected="+url.QueryEscape(provider.Name), http.StatusFound)
}

// --- Agent token retrieval (OAuth bearer) ------------------------------------

// handleVaultGetToken returns a fresh access token for the caller's vault
// connection. Auth is OAuth 2.1 bearer — the token must have a user_id
// binding (delegation), otherwise we can't look up the right connection.
//
// Route: GET /api/v1/vault/{provider}/token.
func (s *Server) handleVaultGetToken(w http.ResponseWriter, r *http.Request) {
	if s.OAuthServer == nil || s.OAuthServer.RawStore == nil {
		// OAuth 2.1 server not wired — reject cleanly rather than 500.
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "OAuth server not available"))
		return
	}

	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		w.Header().Set("WWW-Authenticate", `Bearer realm="vault"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Missing Bearer token"))
		return
	}
	rawToken := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	if rawToken == "" {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Empty bearer token"))
		return
	}

	// Token lookup delegates to the oauth Server's canonical three-tier
	// resolver (JWT → JTI, opaque "key.sig" → sha256(sig), raw → sha256(raw)).
	// Rolling our own half-measure here silently 401'd JWT bearers.
	tok := s.OAuthServer.LookupBearer(r.Context(), rawToken)
	if tok == nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Bearer token not recognised"))
		return
	}
	if tok.RevokedAt != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token",error_description="revoked"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Bearer token revoked"))
		return
	}
	if !tok.ExpiresAt.IsZero() && time.Now().UTC().After(tok.ExpiresAt) {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token",error_description="expired"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("invalid_token", "Bearer token expired"))
		return
	}
	if tok.UserID == "" {
		// No delegation binding — vault is a per-user concept, so a
		// client-credentials token has no user to scope to.
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope"`)
		writeJSON(w, http.StatusUnauthorized, errPayload("insufficient_scope", "Token has no user binding"))
		return
	}
	// Soft scope check: if vault:read is in the scope string, we're happy.
	// When the token has no scope at all, allow (covers legacy tokens until
	// we enforce). Future: flip to strict.
	if tok.Scope != "" && !scopeContains(tok.Scope, "vault:read") {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficient_scope",scope="vault:read"`)
		writeJSON(w, http.StatusForbidden, errPayload("insufficient_scope", "Token lacks vault:read scope"))
		return
	}

	providerName := chi.URLParam(r, "provider")
	provider, err := s.Store.GetVaultProviderByName(r.Context(), providerName)
	if err != nil || provider == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not available"))
		return
	}

	access, err := s.VaultManager.GetFreshToken(r.Context(), provider.ID, tok.UserID)
	if err != nil {
		switch {
		case errors.Is(err, vault.ErrNeedsReauth):
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error":    "reauth_required",
				"provider": provider.Name,
			})
			return
		case errors.Is(err, vault.ErrNoRefreshToken):
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error":    "no_refresh_token",
				"provider": provider.Name,
			})
			return
		case errors.Is(err, vault.ErrConnectionNotFound):
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "No vault connection for this user"))
			return
		case errors.Is(err, vault.ErrProviderNotFound):
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault provider not available"))
			return
		}
		internal(w, err)
		return
	}

	// Pull the connection again for expires_at (GetFreshToken doesn't return
	// the row). This is a cheap read and keeps the handler contract clean.
	conn, _ := s.Store.GetVaultConnection(r.Context(), provider.ID, tok.UserID)
	var expiresAt time.Time
	if conn != nil && conn.ExpiresAt != nil {
		expiresAt = *conn.ExpiresAt
	}

	s.auditVault(r, "agent", auditVaultTokenRetrieved, "vault_connection",
		func() string {
			if conn != nil {
				return conn.ID
			}
			return ""
		}(),
		map[string]any{
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
			"user_id":       tok.UserID,
		})

	resp := map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
	}
	if !expiresAt.IsZero() {
		resp["expires_at"] = expiresAt.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, resp)
}

// scopeContains reports whether space-separated scope string s contains
// needle. Scopes are case-sensitive per RFC 6749.
func scopeContains(s, needle string) bool {
	for _, sc := range strings.Fields(s) {
		if sc == needle {
			return true
		}
	}
	return false
}

// --- User-facing connection management (session) -----------------------------

// handleListVaultConnections returns the authenticated user's vault connections
// enriched with provider display metadata the dashboard needs to render.
func (s *Server) handleListVaultConnections(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	conns, err := s.VaultManager.ListConnections(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}

	// Cache provider rows so we don't re-fetch the same provider per row when
	// a user has several connections to the same service (unlikely but cheap).
	providerCache := make(map[string]*storage.VaultProvider, len(conns))
	out := make([]vaultConnectionResponse, 0, len(conns))
	for _, c := range conns {
		p, ok := providerCache[c.ProviderID]
		if !ok {
			p, _ = s.Store.GetVaultProviderByID(r.Context(), c.ProviderID)
			providerCache[c.ProviderID] = p
		}
		resp := vaultConnectionResponse{
			ID:              c.ID,
			ProviderID:      c.ProviderID,
			Scopes:          c.Scopes,
			ExpiresAt:       c.ExpiresAt,
			NeedsReauth:     c.NeedsReauth,
			LastRefreshedAt: c.LastRefreshedAt,
			CreatedAt:       c.CreatedAt,
			UpdatedAt:       c.UpdatedAt,
		}
		if resp.Scopes == nil {
			resp.Scopes = []string{}
		}
		if p != nil {
			resp.ProviderName = p.Name
			resp.ProviderDisplayName = p.DisplayName
			resp.ProviderIconURL = p.IconURL
		}
		out = append(out, resp)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out, "total": len(out)})
}

// handleDeleteVaultConnection handles DELETE /api/v1/vault/connections/{id}.
// IDOR protection: we only look for the connection among the caller's own
// rows — someone else's ID returns 404 without disclosing existence.
func (s *Server) handleDeleteVaultConnection(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r.Context())
	connID := chi.URLParam(r, "id")

	conns, err := s.Store.ListVaultConnectionsByUserID(r.Context(), userID)
	if err != nil {
		internal(w, err)
		return
	}
	var target *storage.VaultConnection
	for _, c := range conns {
		if c.ID == connID {
			target = c
			break
		}
	}
	if target == nil {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault connection not found"))
		return
	}

	if err := s.VaultManager.Disconnect(r.Context(), connID); err != nil {
		if errors.Is(err, vault.ErrConnectionNotFound) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault connection not found"))
			return
		}
		internal(w, err)
		return
	}

	s.auditVault(r, "user", auditVaultConnectionDeleted, "vault_connection", connID, map[string]any{
		"provider_id": target.ProviderID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// adminVaultConnectionResponse is the admin view of a vault connection. Adds
// user_id (cross-user listing context) on top of the user-facing fields. Token
// material is never serialized — both the underlying VaultConnection columns
// and this struct hide it.
type adminVaultConnectionResponse struct {
	vaultConnectionResponse
	UserID string `json:"user_id"`
}

// handleAdminListVaultConnections handles GET /api/v1/admin/vault/connections.
// Admin-scope listing of every vault connection across all users, enriched
// with provider display metadata. Accepts optional ?provider_id=<id> to
// scope results to a single provider without a client-side full scan.
func (s *Server) handleAdminListVaultConnections(w http.ResponseWriter, r *http.Request) {
	var conns []*storage.VaultConnection
	var err error

	if filterProviderID := r.URL.Query().Get("provider_id"); filterProviderID != "" {
		conns, err = s.Store.ListVaultConnectionsByProviderID(r.Context(), filterProviderID)
	} else {
		conns, err = s.Store.ListAllVaultConnections(r.Context())
	}
	if err != nil {
		internal(w, err)
		return
	}

	providerCache := make(map[string]*storage.VaultProvider, len(conns))
	out := make([]adminVaultConnectionResponse, 0, len(conns))
	for _, c := range conns {
		p, ok := providerCache[c.ProviderID]
		if !ok {
			p, _ = s.Store.GetVaultProviderByID(r.Context(), c.ProviderID)
			providerCache[c.ProviderID] = p
		}
		row := adminVaultConnectionResponse{
			vaultConnectionResponse: vaultConnectionResponse{
				ID:              c.ID,
				ProviderID:      c.ProviderID,
				Scopes:          c.Scopes,
				ExpiresAt:       c.ExpiresAt,
				NeedsReauth:     c.NeedsReauth,
				LastRefreshedAt: c.LastRefreshedAt,
				CreatedAt:       c.CreatedAt,
				UpdatedAt:       c.UpdatedAt,
			},
			UserID: c.UserID,
		}
		if row.Scopes == nil {
			row.Scopes = []string{}
		}
		if p != nil {
			row.ProviderName = p.Name
			row.ProviderDisplayName = p.DisplayName
			row.ProviderIconURL = p.IconURL
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out, "total": len(out)})
}

// handleAdminSeedDemoVaultConnection handles POST /api/v1/admin/vault/connections/_seed_demo.
// Admin-only. Creates a synthetic vault_connection for demo/test purposes — bypasses
// the real OAuth browser flow. Tokens are FieldEncryptor-encrypted fake values.
//
// Body: {"user_id": "...", "provider_id": "...", "scopes": ["..."]}
// Response: {"id": "vc_...", "user_id": "...", "provider_id": "...", "expires_at": "..."}
//
// Idempotent: if a connection already exists for (provider_id, user_id), returns it.
func (s *Server) handleAdminSeedDemoVaultConnection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID     string   `json:"user_id"`
		ProviderID string   `json:"provider_id"`
		Scopes     []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "Invalid JSON body"))
		return
	}
	if body.UserID == "" || body.ProviderID == "" {
		writeJSON(w, http.StatusBadRequest, errPayload("invalid_request", "user_id and provider_id required"))
		return
	}

	// Idempotent: return existing connection if one already exists.
	existing, _ := s.Store.GetVaultConnection(r.Context(), body.ProviderID, body.UserID)
	if existing != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	// Generate a random suffix for the fake tokens so each seed is unique.
	suffixBytes := make([]byte, 8)
	if _, err := rand.Read(suffixBytes); err != nil {
		internal(w, err)
		return
	}
	suffix := hex.EncodeToString(suffixBytes)

	accessTokenEnc, err := s.FieldEncryptor.Encrypt("demo_fake_access_" + suffix)
	if err != nil {
		internal(w, err)
		return
	}
	refreshTokenEnc, err := s.FieldEncryptor.Encrypt("demo_fake_refresh_" + suffix)
	if err != nil {
		internal(w, err)
		return
	}

	scopes := body.Scopes
	if scopes == nil {
		scopes = []string{}
	}

	id, err := newVaultConnID()
	if err != nil {
		internal(w, err)
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(1 * time.Hour)
	conn := &storage.VaultConnection{
		ID:              id,
		ProviderID:      body.ProviderID,
		UserID:          body.UserID,
		AccessTokenEnc:  accessTokenEnc,
		RefreshTokenEnc: refreshTokenEnc,
		TokenType:       "Bearer",
		Scopes:          scopes,
		ExpiresAt:       &expiresAt,
		NeedsReauth:     false,
		LastRefreshedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.Store.CreateVaultConnection(r.Context(), conn); err != nil {
		internal(w, err)
		return
	}

	s.auditVault(r, "admin", auditVaultConnectionCreated, "vault_connection", conn.ID, map[string]any{
		"provider_id": body.ProviderID,
		"user_id":     body.UserID,
		"demo_seed":   true,
	})

	writeJSON(w, http.StatusCreated, conn)
}

// newVaultConnID generates a vc_<24 hex chars> connection ID using the same
// scheme as vault.newID but without importing the internal vault package.
func newVaultConnID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "vc_" + hex.EncodeToString(b), nil
}

// atlassianFetchCloudIDs calls the Atlassian accessible-resources endpoint
// with the freshly-issued access token, extracts cloud IDs, and persists them
// in VaultConnection.Metadata["accessible_resources"]. Without this step the
// token is useless because callers cannot construct a Jira Cloud API URL.
//
// On success the connection row is updated in-place. On error the caller
// should roll back the connection (disconnect it) before returning an error
// to the user.
func (s *Server) atlassianFetchCloudIDs(ctx context.Context, conn *storage.VaultConnection) error {
	// We need the plaintext access token to call the Atlassian API.
	accessToken, err := s.VaultManager.DecryptAccessToken(ctx, conn)
	if err != nil {
		return fmt.Errorf("decrypt access token for accessible-resources: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.atlassian.com/oauth/token/accessible-resources", nil)
	if err != nil {
		return fmt.Errorf("build accessible-resources request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("accessible-resources GET: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return fmt.Errorf("read accessible-resources body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("accessible-resources returned %d: %s", resp.StatusCode, body)
	}

	var resources []struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		AvatarURL string   `json:"avatarUrl"`
	}
	if err := json.Unmarshal(body, &resources); err != nil {
		return fmt.Errorf("parse accessible-resources: %w", err)
	}
	if len(resources) == 0 {
		return fmt.Errorf("no Jira Cloud sites found for this token; the app may not be installed on any site")
	}

	// Persist as structured metadata — callers read
	// conn.Metadata["accessible_resources"] to pick a cloud ID.
	type cloudResource struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := make([]cloudResource, len(resources))
	for i, r := range resources {
		out[i] = cloudResource{ID: r.ID, Name: r.Name}
	}
	if conn.Metadata == nil {
		conn.Metadata = map[string]any{}
	}
	conn.Metadata["accessible_resources"] = out

	return s.Store.UpdateVaultConnection(ctx, conn)
}

// handleAdminDeleteVaultConnection handles DELETE /api/v1/admin/vault/connections/{id}.
// Cross-user revoke — admin can disconnect any vault connection without owning
// the session. Audited as admin-actor for traceability.
//
// Layer 5 cascade: after disconnect, all OAuth tokens belonging to agents that
// ever fetched from this vault connection are revoked. Two audit events are
// emitted: vault.disconnected and vault.disconnect_cascade.
func (s *Server) handleAdminDeleteVaultConnection(w http.ResponseWriter, r *http.Request) {
	connID := chi.URLParam(r, "id")
	conn, err := s.Store.GetVaultConnectionByID(r.Context(), connID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault connection not found"))
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	if err := s.VaultManager.Disconnect(r.Context(), connID); err != nil {
		if errors.Is(err, vault.ErrConnectionNotFound) {
			writeJSON(w, http.StatusNotFound, errPayload("not_found", "Vault connection not found"))
			return
		}
		internal(w, err)
		return
	}

	// Layer 5 cascade: revoke tokens of all agents that ever fetched from this vault connection.
	agents, _ := s.Store.ListAgentsByVaultRetrieval(r.Context(), connID)
	revokedAgentIDs := make([]string, 0, len(agents))
	totalRevoked := int64(0)
	for _, ag := range agents {
		n, err := s.Store.RevokeOAuthTokensByClientID(r.Context(), ag.ClientID)
		if err != nil {
			continue
		}
		if n > 0 {
			revokedAgentIDs = append(revokedAgentIDs, ag.ID)
			totalRevoked += n
		}
	}

	// Audit: vault.disconnected
	s.auditVault(r, "admin", auditVaultConnectionDeleted, "vault_connection", connID, map[string]any{
		"provider_id": conn.ProviderID,
		"user_id":     conn.UserID,
	})

	// Audit: vault.disconnect_cascade
	s.auditVault(r, "admin", "vault.disconnect_cascade", "vault_connection", connID, map[string]any{
		"vault_connection_id":  connID,
		"revoked_agent_ids":    revokedAgentIDs,
		"revoked_token_count":  totalRevoked,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"disconnected":        true,
		"connection_id":       connID,
		"revoked_agent_ids":   revokedAgentIDs,
		"revoked_token_count": totalRevoked,
	})
}
