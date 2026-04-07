package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/gorilla/securecookie"

	"github.com/sharkauth/sharkauth/internal/storage"
)

const (
	cookieName    = "shark_session"
	sessionPrefix = "sess_"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session expired")
	ErrNoCookie        = errors.New("no session cookie")
)

// SessionManager handles session creation, validation, and cookies.
type SessionManager struct {
	store       storage.Store
	cookieCodec *securecookie.SecureCookie
	lifetime    time.Duration
	secure      bool
}

// NewSessionManager creates a new SessionManager.
// The secret must be at least 32 bytes; it is used as hash key for securecookie.
// A 32-byte block key is derived from the secret for AES encryption using SHA-256.
// baseURL is used to determine whether to set the Secure flag on cookies.
func NewSessionManager(store storage.Store, secret string, lifetime time.Duration, baseURL string) *SessionManager {
	secretBytes := []byte(secret)
	// Hash key: use first 32 bytes of secret for HMAC signing
	hashKey := secretBytes
	if len(hashKey) > 32 {
		hashKey = hashKey[:32]
	}
	// Block key: derive a 32-byte AES key using SHA-256
	blockKeyHash := sha256.Sum256(append([]byte("sharkauth-block-key:"), secretBytes...))
	blockKey := blockKeyHash[:]

	codec := securecookie.New(hashKey, blockKey)

	return &SessionManager{
		store:       store,
		cookieCodec: codec,
		lifetime:    lifetime,
		secure:      strings.HasPrefix(baseURL, "https://"),
	}
}

// newSessionID generates a new session ID with the sess_ prefix.
func newSessionID() string {
	id, _ := gonanoid.New()
	return sessionPrefix + id
}

// CreateSession creates a new session and returns the session.
func (sm *SessionManager) CreateSession(ctx context.Context, userID, ip, userAgent, authMethod string) (*storage.Session, error) {
	now := time.Now().UTC()

	sess := &storage.Session{
		ID:         newSessionID(),
		UserID:     userID,
		IP:         ip,
		UserAgent:  userAgent,
		MFAPassed:  true,
		AuthMethod: authMethod,
		ExpiresAt:  now.Add(sm.lifetime).Format(time.RFC3339),
		CreatedAt:  now.Format(time.RFC3339),
	}

	if err := sm.store.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}
	return sess, nil
}

// CreateSessionWithMFA creates a new session with a specific mfa_passed value.
func (sm *SessionManager) CreateSessionWithMFA(ctx context.Context, userID, ip, userAgent, authMethod string, mfaPassed bool) (*storage.Session, error) {
	now := time.Now().UTC()

	sess := &storage.Session{
		ID:         newSessionID(),
		UserID:     userID,
		IP:         ip,
		UserAgent:  userAgent,
		MFAPassed:  mfaPassed,
		AuthMethod: authMethod,
		ExpiresAt:  now.Add(sm.lifetime).Format(time.RFC3339),
		CreatedAt:  now.Format(time.RFC3339),
	}

	if err := sm.store.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}
	return sess, nil
}

// ValidateSession gets a session by ID and checks it hasn't expired.
func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string) (*storage.Session, error) {
	sess, err := sm.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	expiresAt, err := time.Parse(time.RFC3339, sess.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("parsing session expiry: %w", err)
	}

	if time.Now().UTC().After(expiresAt) {
		// Clean up expired session
		_ = sm.store.DeleteSession(ctx, sessionID)
		return nil, ErrSessionExpired
	}

	return sess, nil
}

// SetSessionCookie sets an encrypted session cookie on the response.
func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	encoded, err := sm.cookieCodec.Encode(cookieName, sessionID)
	if err != nil {
		// This should not happen in practice; log and move on
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Secure:   sm.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sm.lifetime.Seconds()),
	})
}

// GetSessionFromRequest reads and decodes the session cookie from the request.
func (sm *SessionManager) GetSessionFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", ErrNoCookie
	}

	var sessionID string
	if err := sm.cookieCodec.Decode(cookieName, cookie.Value, &sessionID); err != nil {
		return "", fmt.Errorf("decoding session cookie: %w", err)
	}

	return sessionID, nil
}

// ClearSessionCookie removes the session cookie.
func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// UpgradeMFA sets mfa_passed=1 on a session.
func (sm *SessionManager) UpgradeMFA(ctx context.Context, sessionID string) error {
	sess, err := sm.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	sess.MFAPassed = true

	// We need to delete and re-create because the storage interface doesn't have UpdateSession.
	// Actually, let's check - we'll use a direct DB approach. But we shouldn't modify storage.
	// The Session struct has MFAPassed - we delete the old session and create a new one with same ID.
	if err := sm.store.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("deleting session for MFA upgrade: %w", err)
	}
	if err := sm.store.CreateSession(ctx, sess); err != nil {
		return fmt.Errorf("re-creating session for MFA upgrade: %w", err)
	}

	return nil
}
