package middleware

import (
	"context"
	"net/http"

	"github.com/sharkauth/sharkauth/internal/auth"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID.
	UserIDKey contextKey = "user_id"
	// SessionIDKey is the context key for the current session ID.
	SessionIDKey contextKey = "session_id"
	// MFAPassedKey is the context key indicating whether MFA has been completed.
	MFAPassedKey contextKey = "mfa_passed"
)

// GetUserID returns the authenticated user ID from the request context.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

// GetSessionID returns the session ID from the request context.
func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(SessionIDKey).(string); ok {
		return v
	}
	return ""
}

// GetMFAPassed returns whether MFA was completed for the current session.
func GetMFAPassed(ctx context.Context) bool {
	if v, ok := ctx.Value(MFAPassedKey).(bool); ok {
		return v
	}
	return false
}

// RequireSessionFunc returns a middleware that validates the session cookie using the
// given SessionManager and sets user ID, session ID, and MFA status in the context.
// Returns 401 if no valid session is found.
func RequireSessionFunc(sm *auth.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID, err := sm.GetSessionFromRequest(r)
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"No valid session"}`, http.StatusUnauthorized)
				return
			}

			sess, err := sm.ValidateSession(r.Context(), sessionID)
			if err != nil {
				http.Error(w, `{"error":"unauthorized","message":"No valid session"}`, http.StatusUnauthorized)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
			ctx = context.WithValue(ctx, SessionIDKey, sess.ID)
			ctx = context.WithValue(ctx, MFAPassedKey, sess.MFAPassed)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireSession is a middleware that validates the session cookie and sets the
// user ID in the context. Returns 401 if no valid session is found.
// This is the legacy placeholder version; use RequireSessionFunc(sm) instead.
func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("shark_session")
		if err != nil || cookie.Value == "" {
			http.Error(w, `{"error":"unauthorized","message":"No valid session"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), SessionIDKey, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireMFA is a middleware that checks if MFA has been completed for the session.
// Must be used after RequireSession.
func RequireMFA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !GetMFAPassed(r.Context()) {
			http.Error(w, `{"error":"mfa_required","message":"MFA verification required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
