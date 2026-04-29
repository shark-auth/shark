package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/shark-auth/shark/internal/auth"
	jwtpkg "github.com/shark-auth/shark/internal/auth/jwt"
	"github.com/shark-auth/shark/internal/cache"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID.
	UserIDKey contextKey = "user_id"
	// SessionIDKey is the context key for the current session ID.
	SessionIDKey contextKey = "session_id"
	// MFAPassedKey is the context key indicating whether MFA has been completed.
	MFAPassedKey contextKey = "mfa_passed"
	// AuthMethodKey is the context key indicating which auth method was used ("jwt" or "cookie").
	AuthMethodKey contextKey = "auth_method"
	// claimsKey is the unexported context key for stashing JWT claims.
	claimsKey contextKey = "jwt_claims"
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

// GetAuthMethod returns the auth method used ("jwt" or "cookie").
func GetAuthMethod(ctx context.Context) string {
	if v, ok := ctx.Value(AuthMethodKey).(string); ok {
		return v
	}
	return ""
}

// GetClaims returns the JWT Claims stashed in the context, or nil if not present.
func GetClaims(ctx context.Context) *jwtpkg.Claims {
	if v, ok := ctx.Value(claimsKey).(*jwtpkg.Claims); ok {
		return v
	}
	return nil
}

// RequireSessionFunc returns a middleware that accepts Bearer JWT (when jwtMgr is
// non-nil) or a shark_session cookie. Decision tree per PHASE3.md Â§2.1:
//
//  1. Authorization: Bearer <token> present â†’ validate with jwtMgr.
//     Success: set context keys, AuthMethod="jwt". DO NOT fall through on failure.
//  2. No Bearer header â†’ try cookie. AuthMethod="cookie" on success.
//  3. Neither: 401 with WWW-Authenticate: Bearer.
//
// Token type enforcement (Â§2.3): refresh tokens are rejected as bearer credential.
func RequireSessionFunc(sm *auth.SessionManager, jwtMgr *jwtpkg.Manager, authCache *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Branch 1: Bearer JWT path
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if jwtMgr == nil {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "JWT not configured")
					return
				}
				claims, err := jwtMgr.Validate(r.Context(), token)
				if err != nil {
					// Refresh token used as bearer credential â€” return actionable error (Â§2.3).
					if errors.Is(err, jwtpkg.ErrRefreshToken) {
						w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token",error_description="refresh token cannot be used as access credential"`)
						writeJSONError(w, http.StatusUnauthorized, "unauthorized", "refresh token cannot be used as access credential")
						return
					}
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
					return
				}
				// Secondary check: belt-and-suspenders guard for any token_type not in {session, access}.
				// In practice ErrRefreshToken above catches the refresh case before reaching here.
				if claims.TokenType == "refresh" {
					w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token",error_description="refresh token cannot be used as access credential"`)
					writeJSONError(w, http.StatusUnauthorized, "unauthorized", "refresh token cannot be used as access credential")
					return
				}
				// session token carries SessionID; access token does not
				sessionID := ""
				if claims.TokenType == "session" {
					sessionID = claims.SessionID
				}
				ctx := r.Context()
				ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
				ctx = context.WithValue(ctx, SessionIDKey, sessionID)
				ctx = context.WithValue(ctx, MFAPassedKey, claims.MFAPassed)
				ctx = context.WithValue(ctx, AuthMethodKey, "jwt")
				ctx = context.WithValue(ctx, claimsKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Branch 2: Cookie path
			sessionID, err := sm.GetSessionFromRequest(r)
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "No valid session")
				return
			}

			// Try cache first
			if authCache != nil {
				decision, found := authCache.Get("session:" + sessionID)
				if found {
					ctx := r.Context()
					ctx = context.WithValue(ctx, UserIDKey, decision.UserID)
					ctx = context.WithValue(ctx, SessionIDKey, decision.SessionID)
					ctx = context.WithValue(ctx, MFAPassedKey, decision.MFAPassed)
					ctx = context.WithValue(ctx, AuthMethodKey, "cookie")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			sess, err := sm.ValidateSession(r.Context(), sessionID)
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "No valid session")
				return
			}

			// Save to cache
			if authCache != nil {
				authCache.Set("session:"+sessionID, cache.AuthDecision{
					UserID:    sess.UserID,
					SessionID: sess.ID,
					MFAPassed: sess.MFAPassed,
				})
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
			ctx = context.WithValue(ctx, SessionIDKey, sess.ID)
			ctx = context.WithValue(ctx, MFAPassedKey, sess.MFAPassed)
			ctx = context.WithValue(ctx, AuthMethodKey, "cookie")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalSessionFunc returns a middleware that populates the request context
// with user identity if a valid session cookie or Bearer JWT is present, but
// does NOT reject unauthenticated requests. Use this for endpoints where
// authentication is helpful but not required (e.g. /oauth/authorize which
// handles the not-logged-in case itself).
func OptionalSessionFunc(sm *auth.SessionManager, jwtMgr *jwtpkg.Manager, authCache *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Branch 1: Bearer JWT path
			if strings.HasPrefix(authHeader, "Bearer ") && jwtMgr != nil {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				claims, err := jwtMgr.Validate(r.Context(), token)
				if err == nil && claims.TokenType != "refresh" {
					sessionID := ""
					if claims.TokenType == "session" {
						sessionID = claims.SessionID
					}
					ctx := r.Context()
					ctx = context.WithValue(ctx, UserIDKey, claims.Subject)
					ctx = context.WithValue(ctx, SessionIDKey, sessionID)
					ctx = context.WithValue(ctx, MFAPassedKey, claims.MFAPassed)
					ctx = context.WithValue(ctx, AuthMethodKey, "jwt")
					ctx = context.WithValue(ctx, claimsKey, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				// Invalid token: fall through to cookie path.
			}

			// Branch 2: Cookie path (best-effort)
			if sm != nil {
				sessionID, err := sm.GetSessionFromRequest(r)
				if err == nil {
					// Try cache first
					if authCache != nil {
						decision, found := authCache.Get("session:" + sessionID)
						if found {
							ctx := r.Context()
							ctx = context.WithValue(ctx, UserIDKey, decision.UserID)
							ctx = context.WithValue(ctx, SessionIDKey, decision.SessionID)
							ctx = context.WithValue(ctx, MFAPassedKey, decision.MFAPassed)
							ctx = context.WithValue(ctx, AuthMethodKey, "cookie")
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}

					sess, err := sm.ValidateSession(r.Context(), sessionID)
					if err == nil {
						if authCache != nil {
							authCache.Set("session:"+sessionID, cache.AuthDecision{
								UserID:    sess.UserID,
								SessionID: sess.ID,
								MFAPassed: sess.MFAPassed,
							})
						}

						ctx := r.Context()
						ctx = context.WithValue(ctx, UserIDKey, sess.UserID)
						ctx = context.WithValue(ctx, SessionIDKey, sess.ID)
						ctx = context.WithValue(ctx, MFAPassedKey, sess.MFAPassed)
						ctx = context.WithValue(ctx, AuthMethodKey, "cookie")
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			// No valid session â€” proceed unauthenticated.
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSession is a middleware that validates the session cookie and sets the
// user ID in the context. Returns 401 if no valid session is found.
// This is the legacy placeholder version; use RequireSessionFunc(sm, jwtMgr) instead.
func RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("shark_session")
		if err != nil || cookie.Value == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized", "No valid session")
			return
		}

		ctx := context.WithValue(r.Context(), SessionIDKey, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireEmailVerifiedFunc returns a middleware that checks if the authenticated user's
// email is verified. Must be used after RequireSessionFunc (needs UserID in context).
// The isVerified callback looks up the user and returns their email_verified status.
func RequireEmailVerifiedFunc(isVerified func(ctx context.Context, userID string) (bool, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "No valid session")
				return
			}

			verified, err := isVerified(r.Context(), userID)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
				return
			}

			if !verified {
				writeJSONError(w, http.StatusForbidden, "email_verification_required", "Please verify your email address before continuing")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireMFA is a middleware that checks if MFA has been completed for the session.
// Must be used after RequireSession.
func RequireMFA(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !GetMFAPassed(r.Context()) {
			writeJSONError(w, http.StatusForbidden, "mfa_required", "MFA verification required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
