package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/shark-auth/shark/internal/storage"
)

// ---------------------------------------------------------------------------
// Rate limiter â€” per-client_id, 10 requests per minute window
// ---------------------------------------------------------------------------

const deviceRateLimitPerMin = 10

type deviceRateEntry struct {
	count     int
	windowEnd time.Time
}

var (
	deviceRateMu   sync.Mutex
	deviceRateMap  = map[string]*deviceRateEntry{}
)

// checkDeviceRateLimit returns true if the client_id is within the allowed
// 10-requests-per-minute window.
func checkDeviceRateLimit(clientID string) bool {
	deviceRateMu.Lock()
	defer deviceRateMu.Unlock()

	now := time.Now()
	entry, ok := deviceRateMap[clientID]
	if !ok || now.After(entry.windowEnd) {
		deviceRateMap[clientID] = &deviceRateEntry{count: 1, windowEnd: now.Add(time.Minute)}
		return true
	}
	if entry.count >= deviceRateLimitPerMin {
		return false
	}
	entry.count++
	return true
}

// ---------------------------------------------------------------------------
// Token generation helpers
// ---------------------------------------------------------------------------

// generateDeviceCode returns a cryptographically random 32-byte device_code
// as a base64url string (returned to the client), and its SHA-256 hex hash
// (stored in the DB).
func generateDeviceCode() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(h[:])
	return
}

// generateUserCode returns an 8-char code in XXXX-XXXX format using an
// unambiguous character set (no I, O, 0, 1).
func generateUserCode() (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b[:4]) + "-" + string(b[4:]), nil
}

// ---------------------------------------------------------------------------
// Device authorization response (RFC 8628 Â§3.2)
// ---------------------------------------------------------------------------

type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// HandleDeviceAuthorization handles POST /oauth/device.
// Device Authorization Grant is disabled for v0.1 â€” returns 501 Not Implemented.
func (s *Server) HandleDeviceAuthorization(w http.ResponseWriter, r *http.Request) {
	writeDeviceError(w, http.StatusNotImplemented, "not_implemented",
		"Device authorization grant (RFC 8628) is not yet supported. Coming in v0.2.")
}

// handleDeviceAuthorizationImpl is the real implementation, kept for v0.2.
func (s *Server) handleDeviceAuthorizationImpl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeDeviceError(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		writeDeviceError(w, http.StatusBadRequest, "invalid_request", "client_id is required")
		return
	}

	scope := r.FormValue("scope")
	resource := r.FormValue("resource")

	// Validate the client exists.
	agent, err := s.RawStore.GetAgentByClientID(ctx, clientID)
	if err != nil || !agent.Active {
		writeDeviceError(w, http.StatusUnauthorized, "invalid_client", "unknown or inactive client")
		return
	}

	// Rate limit: 10/min per client_id.
	if !checkDeviceRateLimit(clientID) {
		writeDeviceError(w, http.StatusTooManyRequests, "slow_down", "too many device authorization requests")
		return
	}

	// Generate tokens.
	deviceCodePlain, deviceCodeHash, err := generateDeviceCode()
	if err != nil {
		slog.Error("oauth/device: generate device_code", "err", err)
		writeDeviceError(w, http.StatusInternalServerError, "server_error", "failed to generate device code")
		return
	}

	userCode, err := generateUserCode()
	if err != nil {
		slog.Error("oauth/device: generate user_code", "err", err)
		writeDeviceError(w, http.StatusInternalServerError, "server_error", "failed to generate user code")
		return
	}

	lifetime := s.Config.DeviceCodeLifetimeDuration()
	if lifetime <= 0 {
		lifetime = 15 * time.Minute
	}
	expiresAt := time.Now().UTC().Add(lifetime)

	dc := &storage.OAuthDeviceCode{
		DeviceCodeHash: deviceCodeHash,
		UserCode:       userCode,
		ClientID:       clientID,
		Scope:          scope,
		Resource:       resource,
		Status:         "pending",
		PollInterval:   5,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.RawStore.CreateDeviceCode(ctx, dc); err != nil {
		slog.Error("oauth/device: store device code", "err", err)
		writeDeviceError(w, http.StatusInternalServerError, "server_error", "failed to store device code")
		return
	}

	verificationURI := s.Issuer + "/oauth/device/verify"
	resp := deviceAuthResponse{
		DeviceCode:              deviceCodePlain,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURI + "?user_code=" + userCode,
		ExpiresIn:               int(lifetime.Seconds()),
		Interval:                5,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// ---------------------------------------------------------------------------
// Device verification page â€” GET /oauth/device/verify
// ---------------------------------------------------------------------------

// DeviceVerifyData is the template data for the device code entry page.
type DeviceVerifyData struct {
	UserCode string
	Issuer   string
}

// HandleDeviceVerify handles GET /oauth/device/verify.
// Device Authorization Grant is disabled for v0.1 â€” returns 501 Not Implemented.
func (s *Server) HandleDeviceVerify(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Device authorization grant is not yet supported. Coming in v0.2.", http.StatusNotImplemented)
}

// handleDeviceVerifyImpl is the real implementation, kept for v0.2.
func (s *Server) handleDeviceVerifyImpl(w http.ResponseWriter, r *http.Request) {
	userCode := r.URL.Query().Get("user_code")

	// If user_code provided and user is logged in, render approve/deny page.
	userID := getUserIDFromRequest(r)
	if userCode != "" && userID != "" {
		s.renderDeviceApprovalPage(w, r, userCode, userID)
		return
	}

	// If user_code provided but not logged in, redirect to login.
	if userCode != "" && userID == "" {
		returnTo := s.Issuer + "/oauth/device/verify?user_code=" + userCode
		loginURL := s.Issuer + "/login?return_to=" + returnTo
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	// No user_code â€” render the code-entry form.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'")
	consentTemplates.ExecuteTemplate(w, "device_verify.html", DeviceVerifyData{ //nolint:errcheck
		UserCode: "",
		Issuer:   s.Issuer,
	})
}

// ---------------------------------------------------------------------------
// Device approval page â€” POST /oauth/device/verify
// ---------------------------------------------------------------------------

// DeviceApprovalData is the template data for the device consent decision page.
type DeviceApprovalData struct {
	AgentName string
	ClientID  string
	Scopes    []string
	UserCode  string
	Issuer    string
}

// HandleDeviceApprove handles POST /oauth/device/verify.
// Device Authorization Grant is disabled for v0.1 â€” returns 501 Not Implemented.
func (s *Server) HandleDeviceApprove(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Device authorization grant is not yet supported. Coming in v0.2.", http.StatusNotImplemented)
}

// handleDeviceApproveImpl is the real implementation, kept for v0.2.
func (s *Server) handleDeviceApproveImpl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := getUserIDFromRequest(r)
	if userID == "" {
		http.Redirect(w, r, s.Issuer+"/login?return_to="+s.Issuer+"/oauth/device/verify", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Bad Request",
			Description: "Failed to parse form",
			Issuer:      s.Issuer,
		})
		return
	}

	userCode := r.FormValue("user_code")
	if userCode == "" {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Missing Code",
			Description: "No device code was provided.",
			Issuer:      s.Issuer,
		})
		return
	}

	// Look up the device code.
	dc, err := s.RawStore.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Invalid Code",
			Description: "The code you entered is not valid. Please check and try again.",
			Issuer:      s.Issuer,
		})
		return
	}

	// Check expiry.
	if time.Now().After(dc.ExpiresAt) {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Code Expired",
			Description: "This device code has expired. Please restart the authorization on your device.",
			Issuer:      s.Issuer,
		})
		return
	}

	// Check status.
	if dc.Status != "pending" {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Code Already Used",
			Description: "This device code has already been processed.",
			Issuer:      s.Issuer,
		})
		return
	}

	approved := r.FormValue("approved")
	if approved == "" {
		// No decision yet â€” show the consent page.
		s.renderDeviceApprovalPage(w, r, userCode, userID)
		return
	}

	if approved == "true" {
		if err := s.RawStore.UpdateDeviceCodeStatus(ctx, dc.DeviceCodeHash, "approved", userID); err != nil {
			slog.Error("oauth/device: approve device code", "err", err)
			RenderErrorPage(w, http.StatusInternalServerError, ErrorData{
				Error:       "Server Error",
				Description: "Failed to approve the device. Please try again.",
				Issuer:      s.Issuer,
			})
			return
		}
		// Render success page.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		consentTemplates.ExecuteTemplate(w, "device_success.html", map[string]interface{}{ //nolint:errcheck
			"Issuer": s.Issuer,
		})
		return
	}

	// Denied.
	if err := s.RawStore.UpdateDeviceCodeStatus(ctx, dc.DeviceCodeHash, "denied", userID); err != nil {
		slog.Error("oauth/device: deny device code", "err", err)
	}
	RenderErrorPage(w, http.StatusOK, ErrorData{
		Error:       "Access Denied",
		Description: "You have denied access to the device.",
		Issuer:      s.Issuer,
	})
}

// renderDeviceApprovalPage looks up the device code by user_code and renders
// the consent decision form (agent name, scopes, approve/deny buttons).
func (s *Server) renderDeviceApprovalPage(w http.ResponseWriter, r *http.Request, userCode, userID string) {
	ctx := r.Context()

	dc, err := s.RawStore.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Invalid Code",
			Description: "The code you entered is not valid.",
			Issuer:      s.Issuer,
		})
		return
	}

	if time.Now().After(dc.ExpiresAt) {
		RenderErrorPage(w, http.StatusBadRequest, ErrorData{
			Error:       "Code Expired",
			Description: "This device code has expired.",
			Issuer:      s.Issuer,
		})
		return
	}

	// Look up the agent for display.
	agentName := dc.ClientID
	if agent, err := s.RawStore.GetAgentByClientID(ctx, dc.ClientID); err == nil {
		if agent.Name != "" {
			agentName = agent.Name
		}
	}

	var scopes []string
	if dc.Scope != "" {
		scopes = strings.Split(dc.Scope, " ")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'")
	consentTemplates.ExecuteTemplate(w, "device_consent.html", DeviceApprovalData{ //nolint:errcheck
		AgentName: agentName,
		ClientID:  dc.ClientID,
		Scopes:    scopes,
		UserCode:  userCode,
		Issuer:    s.Issuer,
	})
}

// ---------------------------------------------------------------------------
// Device token polling â€” HandleDeviceTokenRequest
// ---------------------------------------------------------------------------

const devicePollInterval = 5 // seconds

// deviceTokenError writes a JSON error response per RFC 8628 Â§3.5.
type deviceTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// HandleDeviceTokenRequest handles the device_code grant type at POST /oauth/token.
// Called from HandleToken when grant_type == "urn:ietf:params:oauth:grant-type:device_code".
func (s *Server) HandleDeviceTokenRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		writeDeviceError(w, http.StatusBadRequest, "invalid_request", "failed to parse form")
		return
	}

	deviceCodePlain := r.FormValue("device_code")
	if deviceCodePlain == "" {
		writeDeviceError(w, http.StatusBadRequest, "invalid_request", "device_code is required")
		return
	}

	// Hash the device_code to look it up.
	h := sha256.Sum256([]byte(deviceCodePlain))
	deviceCodeHash := hex.EncodeToString(h[:])

	dc, err := s.RawStore.GetDeviceCodeByHash(ctx, deviceCodeHash)
	if err != nil {
		writeDeviceError(w, http.StatusBadRequest, "invalid_grant", "device code not found")
		return
	}

	// Enforce slow_down: if polled faster than interval, return slow_down.
	if dc.LastPolledAt != nil {
		elapsed := time.Since(*dc.LastPolledAt)
		if elapsed < time.Duration(devicePollInterval)*time.Second {
			// Update polled_at even on slow_down to slide the window.
			_ = s.RawStore.UpdateDeviceCodePolledAt(ctx, deviceCodeHash)
			writeDeviceError(w, http.StatusBadRequest, "slow_down", "polling too fast, wait at least 5 seconds")
			return
		}
	}

	// Update last_polled_at for subsequent rate limiting.
	_ = s.RawStore.UpdateDeviceCodePolledAt(ctx, deviceCodeHash)

	// Check expiry.
	if time.Now().After(dc.ExpiresAt) {
		writeDeviceError(w, http.StatusBadRequest, "expired_token", "the device code has expired")
		return
	}

	switch dc.Status {
	case "pending":
		writeDeviceError(w, http.StatusBadRequest, "authorization_pending", "the user has not yet approved the request")
		return

	case "denied":
		writeDeviceError(w, http.StatusBadRequest, "access_denied", "the user denied the authorization request")
		return

	case "expired":
		writeDeviceError(w, http.StatusBadRequest, "expired_token", "the device code has expired")
		return

	case "approved":
		// Fall through to issue tokens below.

	default:
		writeDeviceError(w, http.StatusBadRequest, "invalid_grant", "unknown device code status")
		return
	}

	// Issue access + refresh tokens.
	accessToken, refreshToken, err := s.issueDeviceTokens(ctx, dc)
	if err != nil {
		slog.Error("oauth/device: issue tokens", "err", err)
		writeDeviceError(w, http.StatusInternalServerError, "server_error", "failed to issue tokens")
		return
	}

	lifetime := s.Config.AccessTokenLifetimeDuration()
	resp := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "bearer",
		"expires_in":    int(lifetime.Seconds()),
		"refresh_token": refreshToken,
	}
	if dc.Scope != "" {
		resp["scope"] = dc.Scope
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// issueDeviceTokens mints a new access token and refresh token for an approved
// device flow. Tokens are stored in oauth_tokens and the raw token strings are
// returned to the caller.
func (s *Server) issueDeviceTokens(ctx context.Context, dc *storage.OAuthDeviceCode) (accessToken, refreshToken string, err error) {
	now := time.Now().UTC()
	accessLifetime := s.Config.AccessTokenLifetimeDuration()
	refreshLifetime := s.Config.RefreshTokenLifetimeDuration()

	// Generate opaque token values.
	accessRaw, err := randomToken()
	if err != nil {
		return
	}
	refreshRaw, err := randomToken()
	if err != nil {
		return
	}

	accessHash := hashToken(accessRaw)
	refreshHash := hashToken(refreshRaw)

	familyID := "fam_" + uuid.New().String()[:8]
	jtiAccess := uuid.New().String()
	jtiRefresh := uuid.New().String()

	accessRecord := &storage.OAuthToken{
		ID:        "tok_" + uuid.New().String()[:8],
		JTI:       jtiAccess,
		ClientID:  dc.ClientID,
		UserID:    dc.UserID,
		TokenType: "access",
		TokenHash: accessHash,
		Scope:     dc.Scope,
		Audience:  dc.Resource,
		FamilyID:  familyID,
		ExpiresAt: now.Add(accessLifetime),
		CreatedAt: now,
	}

	refreshRecord := &storage.OAuthToken{
		ID:        "tok_" + uuid.New().String()[:8],
		JTI:       jtiRefresh,
		ClientID:  dc.ClientID,
		UserID:    dc.UserID,
		TokenType: "refresh",
		TokenHash: refreshHash,
		Scope:     dc.Scope,
		Audience:  dc.Resource,
		FamilyID:  familyID,
		ExpiresAt: now.Add(refreshLifetime),
		CreatedAt: now,
	}

	if err = s.RawStore.CreateOAuthToken(ctx, accessRecord); err != nil {
		return
	}
	if err = s.RawStore.CreateOAuthToken(ctx, refreshRecord); err != nil {
		return
	}

	return accessRaw, refreshRaw, nil
}

// randomToken returns a cryptographically random 32-byte hex string.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex of a token string (used for storage).
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// writeDeviceError writes a JSON error per RFC 8628.
func writeDeviceError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(deviceTokenError{ //nolint:errcheck
		Error:            errCode,
		ErrorDescription: description,
	})
}
