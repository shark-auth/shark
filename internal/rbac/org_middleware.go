package rbac

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	mw "github.com/sharkauth/sharkauth/internal/api/middleware"
)

// RequireOrgPermission returns HTTP middleware that enforces an org-scoped RBAC
// check. The org ID is read from the chi URL parameter "org_id"; if not set it
// falls back to "id" (for legacy routes). The caller's user ID is read from the
// context key set by RequireSessionFunc.
//
// Response codes:
//   - 404 if the user is not a member of the org (ErrNotMember).
//   - 403 if the user lacks the required (action, resource) permission.
//   - 403 if any unexpected error occurs during the permission check.
func RequireOrgPermission(mgr *RBACManager, action, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID := chi.URLParam(r, "org_id")
			if orgID == "" {
				orgID = chi.URLParam(r, "id")
			}
			userID := mw.GetUserID(r.Context())

			if userID == "" {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized", "No valid session")
				return
			}

			ok, err := mgr.HasOrgPermission(r.Context(), userID, orgID, action, resource)
			if err != nil {
				if errors.Is(err, ErrNotMember) {
					writeJSONError(w, http.StatusNotFound, "not_found", "Organization not found")
					return
				}
				writeJSONError(w, http.StatusForbidden, "forbidden", "Permission check failed")
				return
			}
			if !ok {
				writeJSONError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeJSONError writes a JSON error response. Kept package-local to avoid
// importing the api package (cycle). Matches the errPayload shape used in handlers.
func writeJSONError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Inline JSON — avoids pulling in encoding/json just for a small error body.
	_, _ = w.Write([]byte(`{"error":"` + jsonEscape(code) + `","message":"` + jsonEscape(msg) + `"}`))
}

// jsonEscape replaces the characters that must be escaped inside a JSON string.
// This is intentionally minimal: only handles the characters that could appear
// in our fixed error code/message strings.
func jsonEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
