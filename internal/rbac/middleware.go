package rbac

import (
	"net/http"

	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// RequirePermission returns middleware that checks the authenticated user
// has the given action+resource permission. Returns 403 if denied.
// Must be used after authentication middleware that sets the user ID in context.
func RequirePermission(rbac *RBACManager, action, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := mw.GetUserID(r.Context())
			if userID == "" {
				http.Error(w, `{"error":"unauthorized","message":"Authentication required"}`, http.StatusUnauthorized)
				return
			}

			allowed, err := rbac.HasPermission(r.Context(), userID, action, resource)
			if err != nil {
				http.Error(w, `{"error":"internal_error","message":"Permission check failed"}`, http.StatusInternalServerError)
				return
			}

			if !allowed {
				http.Error(w, `{"error":"forbidden","message":"Insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
