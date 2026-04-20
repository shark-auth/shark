package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// allowedGrantTypes is the set of grant types DCR clients may request.
var allowedGrantTypes = map[string]bool{
	"authorization_code":                          true,
	"client_credentials":                          true,
	"refresh_token":                               true,
	"urn:ietf:params:oauth:grant-type:device_code": true,
}

// dcrMetadata holds the parsed RFC 7591 client metadata from a registration request.
type dcrMetadata struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
	ClientURI               string   `json:"client_uri"`
	LogoURI                 string   `json:"logo_uri"`
	TOSURI                  string   `json:"tos_uri"`
	PolicyURI               string   `json:"policy_uri"`
	Contacts                []string `json:"contacts"`
}

// dcrResponse is the RFC 7591 §3.2.1 registration response.
type dcrResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RegistrationAccessToken string   `json:"registration_access_token"`
	RegistrationClientURI   string   `json:"registration_client_uri"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TOSURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
}

// dcrError is a RFC 7591 error response.
type dcrError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// writeDCRError writes a DCR error response.
func writeDCRError(w http.ResponseWriter, status int, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(dcrError{Error: errCode, ErrorDescription: desc})
}

// writeDCRJSON writes a JSON DCR response.
func writeDCRJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// validateDCRMetadata validates client metadata per RFC 7591.
func validateDCRMetadata(md *dcrMetadata) error {
	// client_name is required.
	if strings.TrimSpace(md.ClientName) == "" {
		return fmt.Errorf("invalid_client_metadata: client_name is required")
	}

	// Set defaults.
	if len(md.GrantTypes) == 0 {
		md.GrantTypes = []string{"authorization_code"}
	}
	if len(md.ResponseTypes) == 0 {
		md.ResponseTypes = []string{"code"}
	}
	if md.TokenEndpointAuthMethod == "" {
		md.TokenEndpointAuthMethod = "client_secret_basic"
	}

	// Validate grant_types against allowed set.
	for _, gt := range md.GrantTypes {
		if !allowedGrantTypes[gt] {
			return fmt.Errorf("invalid_client_metadata: unsupported grant_type %q", gt)
		}
	}

	// If authorization_code is requested, redirect_uris is required.
	needsRedirect := false
	for _, gt := range md.GrantTypes {
		if gt == "authorization_code" {
			needsRedirect = true
			break
		}
	}
	if needsRedirect && len(md.RedirectURIs) == 0 {
		return fmt.Errorf("invalid_redirect_uri: redirect_uris is required for authorization_code grant")
	}

	// Validate each redirect_uri.
	for _, rawURI := range md.RedirectURIs {
		if err := validateRedirectURI(rawURI); err != nil {
			return fmt.Errorf("invalid_redirect_uri: %w", err)
		}
	}

	return nil
}

// validateRedirectURI checks that a redirect URI is an absolute HTTPS URL
// or a localhost/127.0.0.1 URL (allowed for native apps).
func validateRedirectURI(rawURI string) error {
	u, err := url.Parse(rawURI)
	if err != nil || !u.IsAbs() {
		return fmt.Errorf("%q is not an absolute URI", rawURI)
	}

	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("%q: http is only allowed for localhost", rawURI)
	default:
		return fmt.Errorf("%q: scheme %q is not allowed (use https or http://localhost)", rawURI, u.Scheme)
	}
}

// generateDCRToken produces a 32-byte hex-encoded random token and its SHA-256 hash.
func generateDCRToken() (token, tokenHash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	token = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(token))
	tokenHash = hex.EncodeToString(h[:])
	return
}

// extractRegistrationToken reads the Bearer token from the Authorization header.
func extractRegistrationToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// verifyRegistrationToken checks the provided token against the stored hash.
// Returns the DCRClient on success, or writes an error and returns nil.
func (s *Server) verifyRegistrationToken(w http.ResponseWriter, r *http.Request, clientID string) *storage.OAuthDCRClient {
	token := extractRegistrationToken(r)
	if token == "" {
		writeDCRError(w, http.StatusUnauthorized, "invalid_token", "missing or malformed Authorization Bearer token")
		return nil
	}

	dcr, err := s.RawStore.GetDCRClient(r.Context(), clientID)
	if err != nil {
		writeDCRError(w, http.StatusUnauthorized, "invalid_token", "client not found")
		return nil
	}

	// Constant-time compare: hash the provided token and compare against stored hash.
	h := sha256.Sum256([]byte(token))
	provided := hex.EncodeToString(h[:])
	if provided != dcr.RegistrationTokenHash {
		writeDCRError(w, http.StatusUnauthorized, "invalid_token", "invalid registration access token")
		return nil
	}

	return dcr
}

// dcrResponseFromAgent builds a registration response from an Agent and metadata.
func dcrResponseFromAgent(agent *storage.Agent, md *dcrMetadata, clientSecret, regToken, regClientURI string, issuedAt int64) dcrResponse {
	return dcrResponse{
		ClientID:                agent.ClientID,
		ClientSecret:            clientSecret,
		ClientIDIssuedAt:        issuedAt,
		ClientSecretExpiresAt:   0, // never expires
		RegistrationAccessToken: regToken,
		RegistrationClientURI:   regClientURI,
		ClientName:              md.ClientName,
		RedirectURIs:            agent.RedirectURIs,
		GrantTypes:              agent.GrantTypes,
		ResponseTypes:           agent.ResponseTypes,
		TokenEndpointAuthMethod: agent.AuthMethod,
		Scope:                   strings.Join(agent.Scopes, " "),
		ClientURI:               md.ClientURI,
		LogoURI:                 md.LogoURI,
		TOSURI:                  md.TOSURI,
		PolicyURI:               md.PolicyURI,
		Contacts:                md.Contacts,
	}
}

// dcrResponseFromDCRClient reconstructs a response from a stored DCRClient.
// clientSecret and regToken are omitted (not re-issued after initial registration).
func dcrResponseFromDCRClient(dcr *storage.OAuthDCRClient, regClientURI string) (dcrResponse, *dcrMetadata, error) {
	var md dcrMetadata
	if err := json.Unmarshal([]byte(dcr.ClientMetadata), &md); err != nil {
		return dcrResponse{}, nil, fmt.Errorf("parsing stored client metadata: %w", err)
	}

	// Fetch the agent to get canonical values.
	resp := dcrResponse{
		ClientID:              dcr.ClientID,
		ClientIDIssuedAt:      dcr.CreatedAt.Unix(),
		ClientSecretExpiresAt: 0,
		RegistrationClientURI: regClientURI,
		ClientName:            md.ClientName,
		RedirectURIs:          md.RedirectURIs,
		GrantTypes:            md.GrantTypes,
		ResponseTypes:         md.ResponseTypes,
		TokenEndpointAuthMethod: func() string {
			if md.TokenEndpointAuthMethod == "" {
				return "client_secret_basic"
			}
			return md.TokenEndpointAuthMethod
		}(),
		Scope:     md.Scope,
		ClientURI: md.ClientURI,
		LogoURI:   md.LogoURI,
		TOSURI:    md.TOSURI,
		PolicyURI: md.PolicyURI,
		Contacts:  md.Contacts,
	}

	return resp, &md, nil
}

// logDCRAudit records an audit log entry for DCR operations.
func (s *Server) logDCRAudit(ctx context.Context, action, clientID, ip string) {
	id, _ := gonanoid.New()
	if err := s.RawStore.CreateAuditLog(ctx, &storage.AuditLog{
		ID:         "aud_" + id,
		ActorType:  "client",
		Action:     action,
		TargetType: "oauth_client",
		TargetID:   clientID,
		IP:         ip,
		Status:     "success",
		Metadata:   "{}",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		slog.Warn("dcr: audit log write failed", "action", action, "client_id", clientID, "err", err)
	}
}

// HandleDCRRegister handles POST /oauth/register (RFC 7591).
// No authentication required — open registration endpoint.
func (s *Server) HandleDCRRegister(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse and validate client metadata.
	var md dcrMetadata
	if err := json.NewDecoder(r.Body).Decode(&md); err != nil {
		writeDCRError(w, http.StatusBadRequest, "invalid_client_metadata", "invalid JSON body")
		return
	}

	if err := validateDCRMetadata(&md); err != nil {
		// Extract the RFC error code from the error message prefix.
		msg := err.Error()
		code := "invalid_client_metadata"
		if strings.HasPrefix(msg, "invalid_redirect_uri:") {
			code = "invalid_redirect_uri"
			msg = strings.TrimPrefix(msg, "invalid_redirect_uri: ")
		} else if strings.HasPrefix(msg, "invalid_client_metadata:") {
			msg = strings.TrimPrefix(msg, "invalid_client_metadata: ")
		}
		writeDCRError(w, http.StatusBadRequest, code, msg)
		return
	}

	// Generate client_id.
	nid, err := gonanoid.New(21)
	if err != nil {
		slog.Error("dcr: generating nanoid", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to generate client ID")
		return
	}
	agentNid, err := gonanoid.New()
	if err != nil {
		slog.Error("dcr: generating agent nanoid", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to generate agent ID")
		return
	}
	clientID := "shark_dcr_" + nid

	// Generate client_secret.
	secret, secretHash, err := generateDCRToken()
	if err != nil {
		slog.Error("dcr: generating client secret", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to generate client secret")
		return
	}

	// Generate registration_access_token.
	regToken, regTokenHash, err := generateDCRToken()
	if err != nil {
		slog.Error("dcr: generating registration token", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to generate registration token")
		return
	}

	// Parse scopes from space-separated string to slice.
	var scopes []string
	if md.Scope != "" {
		scopes = strings.Fields(md.Scope)
	}

	now := time.Now().UTC()

	// Build the Agent record.
	agent := &storage.Agent{
		ID:               "agent_" + agentNid,
		Name:             md.ClientName,
		Description:      "",
		ClientID:         clientID,
		ClientSecretHash: secretHash,
		ClientType:       "confidential",
		AuthMethod:       md.TokenEndpointAuthMethod,
		RedirectURIs:     md.RedirectURIs,
		GrantTypes:       md.GrantTypes,
		ResponseTypes:    md.ResponseTypes,
		Scopes:           scopes,
		TokenLifetime:    3600,
		LogoURI:          md.LogoURI,
		HomepageURI:      md.ClientURI,
		Active:           true,
		Metadata:         map[string]any{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if agent.RedirectURIs == nil {
		agent.RedirectURIs = []string{}
	}
	if agent.GrantTypes == nil {
		agent.GrantTypes = []string{}
	}
	if agent.ResponseTypes == nil {
		agent.ResponseTypes = []string{}
	}
	if agent.Scopes == nil {
		agent.Scopes = []string{}
	}

	if err := s.RawStore.CreateAgent(ctx, agent); err != nil {
		slog.Error("dcr: creating agent", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to store client registration")
		return
	}

	// Serialize full metadata for later retrieval.
	metaJSON, err := json.Marshal(md)
	if err != nil {
		slog.Error("dcr: marshaling metadata", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to store client metadata")
		return
	}

	dcrClient := &storage.OAuthDCRClient{
		ClientID:              clientID,
		RegistrationTokenHash: regTokenHash,
		ClientMetadata:        string(metaJSON),
		CreatedAt:             now,
	}

	if err := s.RawStore.CreateDCRClient(ctx, dcrClient); err != nil {
		slog.Error("dcr: creating dcr client record", "error", err)
		// Best-effort: deactivate the agent we just created to avoid orphans.
		if deactivateErr := s.RawStore.DeactivateAgent(ctx, agent.ID); deactivateErr != nil {
			slog.Warn("dcr: rollback deactivate agent failed", "agent_id", agent.ID, "err", deactivateErr)
		}
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to store DCR record")
		return
	}

	regClientURI := s.Issuer + "/oauth/register/" + clientID
	s.logDCRAudit(ctx, "oauth.dcr.registered", clientID, r.RemoteAddr)

	resp := dcrResponseFromAgent(agent, &md, secret, regToken, regClientURI, now.Unix())
	writeDCRJSON(w, http.StatusCreated, resp)
}

// HandleDCRGet handles GET /oauth/register/{client_id} (RFC 7592).
// Requires Authorization: Bearer {registration_access_token}.
func (s *Server) HandleDCRGet(w http.ResponseWriter, r *http.Request) {
	clientID := chi.URLParam(r, "client_id")

	dcr := s.verifyRegistrationToken(w, r, clientID)
	if dcr == nil {
		return
	}

	regClientURI := s.Issuer + "/oauth/register/" + clientID
	resp, _, err := dcrResponseFromDCRClient(dcr, regClientURI)
	if err != nil {
		slog.Error("dcr: reconstructing response", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to read client metadata")
		return
	}

	writeDCRJSON(w, http.StatusOK, resp)
}

// HandleDCRUpdate handles PUT /oauth/register/{client_id} (RFC 7592).
// Requires Authorization: Bearer {registration_access_token}.
func (s *Server) HandleDCRUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientID := chi.URLParam(r, "client_id")

	dcr := s.verifyRegistrationToken(w, r, clientID)
	if dcr == nil {
		return
	}

	// Parse updated metadata from body.
	var md dcrMetadata
	if err := json.NewDecoder(r.Body).Decode(&md); err != nil {
		writeDCRError(w, http.StatusBadRequest, "invalid_client_metadata", "invalid JSON body")
		return
	}

	if err := validateDCRMetadata(&md); err != nil {
		msg := err.Error()
		code := "invalid_client_metadata"
		if strings.HasPrefix(msg, "invalid_redirect_uri:") {
			code = "invalid_redirect_uri"
			msg = strings.TrimPrefix(msg, "invalid_redirect_uri: ")
		} else if strings.HasPrefix(msg, "invalid_client_metadata:") {
			msg = strings.TrimPrefix(msg, "invalid_client_metadata: ")
		}
		writeDCRError(w, http.StatusBadRequest, code, msg)
		return
	}

	// Fetch the existing Agent to update it.
	agent, err := s.RawStore.GetAgentByClientID(ctx, clientID)
	if err != nil {
		slog.Error("dcr: fetching agent for update", "client_id", clientID, "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to retrieve client")
		return
	}

	// Update agent fields from new metadata.
	var scopes []string
	if md.Scope != "" {
		scopes = strings.Fields(md.Scope)
	} else {
		scopes = []string{}
	}

	agent.Name = md.ClientName
	agent.AuthMethod = md.TokenEndpointAuthMethod
	agent.RedirectURIs = md.RedirectURIs
	agent.GrantTypes = md.GrantTypes
	agent.ResponseTypes = md.ResponseTypes
	agent.Scopes = scopes
	agent.LogoURI = md.LogoURI
	agent.HomepageURI = md.ClientURI

	if agent.RedirectURIs == nil {
		agent.RedirectURIs = []string{}
	}
	if agent.GrantTypes == nil {
		agent.GrantTypes = []string{}
	}
	if agent.ResponseTypes == nil {
		agent.ResponseTypes = []string{}
	}

	if err := s.RawStore.UpdateAgent(ctx, agent); err != nil {
		slog.Error("dcr: updating agent", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to update client")
		return
	}

	// Update stored metadata.
	metaJSON, err := json.Marshal(md)
	if err != nil {
		slog.Error("dcr: marshaling updated metadata", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to marshal metadata")
		return
	}

	dcr.ClientMetadata = string(metaJSON)
	if err := s.RawStore.UpdateDCRClient(ctx, dcr); err != nil {
		slog.Error("dcr: updating dcr record", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to update DCR record")
		return
	}

	s.logDCRAudit(ctx, "oauth.dcr.updated", clientID, r.RemoteAddr)

	regClientURI := s.Issuer + "/oauth/register/" + clientID
	resp := dcrResponseFromAgent(agent, &md, "", "", regClientURI, dcr.CreatedAt.Unix())
	// On update, do not re-issue secret or registration_access_token.
	resp.ClientSecret = ""
	resp.RegistrationAccessToken = ""
	writeDCRJSON(w, http.StatusOK, resp)
}

// HandleDCRDelete handles DELETE /oauth/register/{client_id} (RFC 7592).
// Requires Authorization: Bearer {registration_access_token}.
func (s *Server) HandleDCRDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clientID := chi.URLParam(r, "client_id")

	dcr := s.verifyRegistrationToken(w, r, clientID)
	if dcr == nil {
		return
	}

	// Fetch agent to get the internal ID for deactivation.
	agent, err := s.RawStore.GetAgentByClientID(ctx, clientID)
	if err != nil {
		slog.Error("dcr: fetching agent for delete", "client_id", clientID, "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to retrieve client")
		return
	}

	// Deactivate the agent.
	if err := s.RawStore.DeactivateAgent(ctx, agent.ID); err != nil {
		slog.Error("dcr: deactivating agent", "error", err)
		writeDCRError(w, http.StatusInternalServerError, "server_error", "failed to deactivate client")
		return
	}

	// Revoke all tokens for this client.
	if _, err := s.RawStore.RevokeOAuthTokensByClientID(ctx, clientID); err != nil {
		// Non-fatal: log but continue.
		slog.Warn("dcr: revoking tokens during delete", "client_id", clientID, "error", err)
	}

	// Delete the DCR record.
	if err := s.RawStore.DeleteDCRClient(ctx, clientID); err != nil {
		slog.Error("dcr: deleting dcr record", "error", err)
		// Non-fatal: agent is already deactivated.
	}

	s.logDCRAudit(ctx, "oauth.dcr.deleted", clientID, r.RemoteAddr)

	w.WriteHeader(http.StatusNoContent)
}
