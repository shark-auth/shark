package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/sharkauth/sharkauth/internal/config"
	"github.com/sharkauth/sharkauth/internal/storage"
)

const (
	passkeyPrefix   = "pk_"
	challengeTTL    = 5 * time.Minute
	cleanupInterval = 1 * time.Minute
)

var (
	ErrPasskeyNotFound   = errors.New("passkey credential not found")
	ErrChallengeNotFound = errors.New("challenge session not found or expired")
	ErrChallengeExpired  = errors.New("challenge session expired")
	ErrNoPasskeys        = errors.New("user has no passkey credentials")
)

// challengeEntry holds WebAuthn session data with an expiry time.
type challengeEntry struct {
	data      *webauthn.SessionData
	expiresAt time.Time
}

// challengeStore is a server-side store for WebAuthn challenge session data with TTL.
type challengeStore struct {
	mu      sync.RWMutex
	entries map[string]*challengeEntry
	done    chan struct{}
}

func newChallengeStore() *challengeStore {
	cs := &challengeStore{
		entries: make(map[string]*challengeEntry),
		done:    make(chan struct{}),
	}
	go cs.cleanupLoop()
	return cs
}

func (cs *challengeStore) put(key string, data *webauthn.SessionData) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.entries[key] = &challengeEntry{
		data:      data,
		expiresAt: time.Now().Add(challengeTTL),
	}
}

func (cs *challengeStore) get(key string) (*webauthn.SessionData, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	entry, ok := cs.entries[key]
	if !ok {
		return nil, ErrChallengeNotFound
	}

	delete(cs.entries, key)

	if time.Now().After(entry.expiresAt) {
		return nil, ErrChallengeExpired
	}

	return entry.data, nil
}

func (cs *challengeStore) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cs.cleanup()
		case <-cs.done:
			return
		}
	}
}

func (cs *challengeStore) cleanup() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	now := time.Now()
	for key, entry := range cs.entries {
		if now.After(entry.expiresAt) {
			delete(cs.entries, key)
		}
	}
}

func (cs *challengeStore) stop() {
	close(cs.done)
}

// webauthnUser adapts a storage.User + credentials to the webauthn.User interface.
type webauthnUser struct {
	user        *storage.User
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u *webauthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	if u.user.Name != nil && *u.user.Name != "" {
		return *u.user.Name
	}
	return u.user.Email
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// PasskeyManager handles WebAuthn registration and login ceremonies.
type PasskeyManager struct {
	webauthn   *webauthn.WebAuthn
	store      storage.Store
	sessions   *SessionManager
	challenges *challengeStore
}

// NewPasskeyManager creates a new PasskeyManager with the given configuration.
func NewPasskeyManager(store storage.Store, sessions *SessionManager, cfg config.PasskeyConfig) (*PasskeyManager, error) {
	attestation := protocol.ConveyancePreference(cfg.Attestation)
	if attestation == "" {
		attestation = protocol.PreferNoAttestation
	}

	residentKey := protocol.ResidentKeyRequirement(cfg.ResidentKey)
	if residentKey == "" {
		residentKey = protocol.ResidentKeyRequirementPreferred
	}

	userVerification := protocol.UserVerificationRequirement(cfg.UserVerification)
	if userVerification == "" {
		userVerification = protocol.VerificationPreferred
	}

	wconfig := &webauthn.Config{
		RPID:                  cfg.RPID,
		RPDisplayName:         cfg.RPName,
		RPOrigins:             []string{cfg.Origin},
		AttestationPreference: attestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      residentKey,
			UserVerification: userVerification,
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("creating webauthn instance: %w", err)
	}

	return &PasskeyManager{
		webauthn:   w,
		store:      store,
		sessions:   sessions,
		challenges: newChallengeStore(),
	}, nil
}

// Stop cleans up background goroutines.
func (pm *PasskeyManager) Stop() {
	pm.challenges.stop()
}

// BeginRegistration starts passkey registration for a logged-in user.
// Returns the credential creation options and a challenge key for the finish step.
func (pm *PasskeyManager) BeginRegistration(ctx context.Context, user *storage.User) (*protocol.CredentialCreation, string, error) {
	existingCreds, err := pm.loadWebAuthnCredentials(ctx, user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("loading existing credentials: %w", err)
	}

	wUser := &webauthnUser{
		user:        user,
		credentials: existingCreds,
	}

	var excludeList []protocol.CredentialDescriptor
	for _, cred := range existingCreds {
		excludeList = append(excludeList, cred.Descriptor())
	}

	var opts []webauthn.RegistrationOption
	if len(excludeList) > 0 {
		opts = append(opts, withExclusions(excludeList))
	}

	creation, sessionData, err := pm.webauthn.BeginRegistration(wUser, opts...)
	if err != nil {
		return nil, "", fmt.Errorf("beginning registration: %w", err)
	}

	challengeKey := generateChallengeKey()
	pm.challenges.put(challengeKey, sessionData)

	return creation, challengeKey, nil
}

// FinishRegistration verifies the attestation response and stores the new credential.
func (pm *PasskeyManager) FinishRegistration(ctx context.Context, user *storage.User, challengeKey string, request *http.Request) (*storage.PasskeyCredential, error) {
	sessionData, err := pm.challenges.get(challengeKey)
	if err != nil {
		return nil, fmt.Errorf("retrieving challenge: %w", err)
	}

	existingCreds, err := pm.loadWebAuthnCredentials(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("loading existing credentials: %w", err)
	}

	wUser := &webauthnUser{
		user:        user,
		credentials: existingCreds,
	}

	credential, err := pm.webauthn.FinishRegistration(wUser, *sessionData, request)
	if err != nil {
		return nil, fmt.Errorf("finishing registration: %w", err)
	}

	transports := make([]string, len(credential.Transport))
	for i, t := range credential.Transport {
		transports[i] = string(t)
	}
	transportsJSON, _ := json.Marshal(transports)

	var aaguid *string
	if len(credential.Authenticator.AAGUID) > 0 {
		s := fmt.Sprintf("%x", credential.Authenticator.AAGUID)
		aaguid = &s
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id, _ := gonanoid.New()
	defaultName := "Unnamed passkey"

	pkCred := &storage.PasskeyCredential{
		ID:           passkeyPrefix + id,
		UserID:       user.ID,
		CredentialID: credential.ID,
		PublicKey:    credential.PublicKey,
		AAGUID:       aaguid,
		SignCount:    int(credential.Authenticator.SignCount),
		Name:         &defaultName,
		Transports:   string(transportsJSON),
		BackedUp:     credential.Flags.BackupState,
		CreatedAt:    now,
	}

	if err := pm.store.CreatePasskeyCredential(ctx, pkCred); err != nil {
		return nil, fmt.Errorf("storing passkey credential: %w", err)
	}

	return pkCred, nil
}

// BeginLogin starts passkey authentication.
// If email is provided, includes allowCredentials for that user passkeys.
// If empty, uses discoverable credential flow.
func (pm *PasskeyManager) BeginLogin(ctx context.Context, email string) (*protocol.CredentialAssertion, string, error) {
	var (
		assertion   *protocol.CredentialAssertion
		sessionData *webauthn.SessionData
		err         error
	)

	if email != "" {
		user, lookupErr := pm.store.GetUserByEmail(ctx, email)
		if lookupErr != nil {
			return nil, "", fmt.Errorf("user not found: %w", lookupErr)
		}

		creds, credErr := pm.loadWebAuthnCredentials(ctx, user.ID)
		if credErr != nil {
			return nil, "", fmt.Errorf("loading credentials: %w", credErr)
		}

		if len(creds) == 0 {
			return nil, "", ErrNoPasskeys
		}

		wUser := &webauthnUser{
			user:        user,
			credentials: creds,
		}

		assertion, sessionData, err = pm.webauthn.BeginLogin(wUser)
		if err != nil {
			return nil, "", fmt.Errorf("beginning login: %w", err)
		}
	} else {
		assertion, sessionData, err = pm.webauthn.BeginDiscoverableLogin()
		if err != nil {
			return nil, "", fmt.Errorf("beginning discoverable login: %w", err)
		}
	}

	challengeKey := generateChallengeKey()
	pm.challenges.put(challengeKey, sessionData)

	return assertion, challengeKey, nil
}

// FinishLogin verifies the assertion response and creates a session with mfa_passed=true.
func (pm *PasskeyManager) FinishLogin(ctx context.Context, challengeKey string, request *http.Request) (*storage.User, *storage.Session, error) {
	sessionData, err := pm.challenges.get(challengeKey)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieving challenge: %w", err)
	}

	var (
		user       *storage.User
		credential *webauthn.Credential
	)

	if len(sessionData.UserID) > 0 {
		userID := string(sessionData.UserID)
		user, err = pm.store.GetUserByID(ctx, userID)
		if err != nil {
			return nil, nil, fmt.Errorf("user not found: %w", err)
		}

		creds, credErr := pm.loadWebAuthnCredentials(ctx, user.ID)
		if credErr != nil {
			return nil, nil, fmt.Errorf("loading credentials: %w", credErr)
		}

		wUser := &webauthnUser{
			user:        user,
			credentials: creds,
		}

		credential, err = pm.webauthn.FinishLogin(wUser, *sessionData, request)
		if err != nil {
			return nil, nil, fmt.Errorf("finishing login: %w", err)
		}
	} else {
		handler := func(rawID, userHandle []byte) (webauthn.User, error) {
			uid := string(userHandle)
			u, lookupErr := pm.store.GetUserByID(ctx, uid)
			if lookupErr != nil {
				return nil, fmt.Errorf("user not found for handle: %w", lookupErr)
			}

			lookupCreds, credErr := pm.loadWebAuthnCredentials(ctx, u.ID)
			if credErr != nil {
				return nil, fmt.Errorf("loading credentials: %w", credErr)
			}

			user = u
			return &webauthnUser{user: u, credentials: lookupCreds}, nil
		}

		_, credential, err = pm.webauthn.FinishPasskeyLogin(handler, *sessionData, request)
		if err != nil {
			return nil, nil, fmt.Errorf("finishing discoverable login: %w", err)
		}
	}

	if credential != nil {
		pm.updateCredentialAfterLogin(ctx, credential)
	}

	sess, err := pm.sessions.CreateSessionWithMFA(
		ctx,
		user.ID,
		request.RemoteAddr,
		request.UserAgent(),
		"passkey",
		true,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating session: %w", err)
	}

	return user, sess, nil
}

func (pm *PasskeyManager) loadWebAuthnCredentials(ctx context.Context, userID string) ([]webauthn.Credential, error) {
	storedCreds, err := pm.store.GetPasskeysByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	creds := make([]webauthn.Credential, len(storedCreds))
	for i, sc := range storedCreds {
		var transports []protocol.AuthenticatorTransport
		if sc.Transports != "" {
			var ts []string
			if jsonErr := json.Unmarshal([]byte(sc.Transports), &ts); jsonErr == nil {
				for _, t := range ts {
					transports = append(transports, protocol.AuthenticatorTransport(t))
				}
			}
		}

		var aaguidBytes []byte
		if sc.AAGUID != nil && *sc.AAGUID != "" {
			aaguidBytes = []byte(*sc.AAGUID)
		}

		creds[i] = webauthn.Credential{
			ID:        sc.CredentialID,
			PublicKey: sc.PublicKey,
			Transport: transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: sc.BackedUp,
				BackupState:    sc.BackedUp,
				UserPresent:    true,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    aaguidBytes,
				SignCount: uint32(sc.SignCount), //#nosec G115 -- SignCount originates from webauthn library as uint32 and is stored/read roundtrip-safe
			},
		}
	}

	return creds, nil
}

func (pm *PasskeyManager) updateCredentialAfterLogin(ctx context.Context, cred *webauthn.Credential) {
	storedCred, err := pm.store.GetPasskeyByCredentialID(ctx, cred.ID)
	if err != nil {
		slog.Warn("could not find credential to update sign count", "error", err)
		return
	}

	newCount := int(cred.Authenticator.SignCount)
	if newCount <= storedCred.SignCount && (newCount != 0 || storedCred.SignCount != 0) {
		slog.Warn("passkey sign count did not increase", "credential_id", storedCred.ID, "stored", storedCred.SignCount, "received", newCount) //#nosec G706 -- slog escapes values; all fields are numeric or internal IDs
	}

	now := time.Now().UTC().Format(time.RFC3339)
	storedCred.SignCount = newCount
	storedCred.LastUsedAt = &now
	storedCred.BackedUp = cred.Flags.BackupState

	if err := pm.store.UpdatePasskeyCredential(ctx, storedCred); err != nil {
		slog.Warn("could not update passkey credential after login", "error", err)
	}
}

func generateChallengeKey() string {
	id, _ := gonanoid.New()
	return "wac_" + id
}

func withExclusions(descriptors []protocol.CredentialDescriptor) webauthn.RegistrationOption {
	return func(cco *protocol.PublicKeyCredentialCreationOptions) {
		cco.CredentialExcludeList = descriptors
	}
}
