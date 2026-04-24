package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sharkauth/sharkauth/internal/auth"
	jwtpkg "github.com/sharkauth/sharkauth/internal/auth/jwt"
	"github.com/sharkauth/sharkauth/internal/proxy"
	rbacpkg "github.com/sharkauth/sharkauth/internal/rbac"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Shared session cookie name. Keep in sync with internal/auth/session.go
// (its value is package-private, so we redeclare the constant here rather
// than adding a getter on SessionManager for a single callsite).
const proxySessionCookieName = "shark_session"

// ErrNoCredentials is returned by proxy Resolver implementations when the
// incoming request carries no recognisable authentication material. The
// BreakerResolver treats a nil Identity (no error) as anonymous in the
// same way, but explicitly returning ErrNoCredentials is clearer from a
// call site that inspects a nested resolver directly.
var ErrNoCredentials = errors.New("proxy: no credentials on request")

// JWTResolver verifies Bearer JWTs locally via the JWT Manager (which has
// cached JWKS). Never reaches the DB and is never blocked by the circuit
// breaker — JWTs are stateless and decoupled from auth-server health.
type JWTResolver struct {
	JWT   *jwtpkg.Manager
	Store storage.Store // reserved for future role expansion on the hot path
}

// Resolve implements proxy.AuthResolver. Returns a populated Identity when
// the request carries a valid Bearer JWT, the zero Identity (no error) when
// no bearer is present (lets the composite resolver fall through to
// session-cookie auth), or an error when the token is present but invalid.
func (r *JWTResolver) Resolve(req *http.Request) (proxy.Identity, error) {
	bearer := extractBearer(req)
	if bearer == "" {
		// No credential is not an error — the composite resolver falls
		// through to session-cookie auth. Keep this distinct from
		// ErrNoCredentials which callers may choose to surface as 401.
		return proxy.Identity{}, nil
	}
	// HACK: ignore admin API keys, which are not JWTs. This allows the
	// LiveResolver (session cookie) to be used when the admin dashboard
	// sends both an API key and a session cookie.
	if strings.HasPrefix(bearer, "sk_") {
		return proxy.Identity{}, nil
	}
	if r.JWT == nil {
		return proxy.Identity{}, ErrNoCredentials
	}
	claims, err := r.JWT.Validate(req.Context(), bearer)
	if err != nil {
		return proxy.Identity{}, err
	}

	id := proxy.Identity{
		UserID:     claims.Subject,
		AuthMethod: "jwt",
		// Proxy v1.5: bake the tier + scopes from JWT claims so the
		// rules engine can evaluate ReqTier / scope predicates entirely
		// from crypto on the hot path (see contracts/require_grammar.md).
		// Roles are hydrated from the store below for symmetry with the
		// LiveResolver path; the JWT-carried roles would also work but
		// the store read is cheap and keeps a single source of truth
		// while RBAC evolves.
		Tier:   claims.Tier,
		Scopes: append([]string(nil), claims.Scope...),
	}

	// Best-effort email + role expansion. If the store round-trip fails we
	// still return the identity — denying a request because the DB hiccuped
	// during role lookup is a bad trade (the JWT already proved identity).
	if r.Store != nil && claims.Subject != "" {
		if u, err := r.Store.GetUserByID(req.Context(), claims.Subject); err == nil && u != nil {
			id.UserEmail = u.Email
		}
		if roles, err := r.Store.GetRolesByUserID(req.Context(), claims.Subject); err == nil {
			for _, role := range roles {
				if role != nil {
					id.Roles = append(id.Roles, role.Name)
				}
			}
		}
	}
	// Fall back to claims.Roles if the store returned nothing — the
	// JWT is the authoritative copy for tokens minted with enrichment.
	if len(id.Roles) == 0 && len(claims.Roles) > 0 {
		id.Roles = append([]string(nil), claims.Roles...)
	}

	return id, nil
}

// LiveResolver validates a session cookie by calling the auth stack
// directly (in-process, not HTTP). Used by the circuit breaker when
// closed/half-open. Called by BreakerResolver — never wired as the JWT
// path.
type LiveResolver struct {
	Sessions *auth.SessionManager
	Store    storage.Store
	RBAC     *rbacpkg.RBACManager
}

// Resolve implements proxy.AuthResolver. Returns a populated Identity for
// a valid session cookie, the zero Identity (no error) when no session
// cookie is present, or an error when the cookie is present but invalid
// or expired.
func (r *LiveResolver) Resolve(req *http.Request) (proxy.Identity, error) {
	if r.Sessions == nil {
		return proxy.Identity{}, nil
	}
	sessionID, err := r.Sessions.GetSessionFromRequest(req)
	if err != nil {
		// No cookie -> anonymous (fall through). Decoding failure ->
		// surface as error so the breaker caches a negative entry.
		if errors.Is(err, auth.ErrNoCookie) {
			return proxy.Identity{}, nil
		}
		return proxy.Identity{}, err
	}

	sess, err := r.Sessions.ValidateSession(req.Context(), sessionID)
	if err != nil {
		return proxy.Identity{}, err
	}

	var (
		email string
		roles []string
		tier  string
	)
	if r.Store != nil {
		if u, err := r.Store.GetUserByID(req.Context(), sess.UserID); err == nil && u != nil {
			email = u.Email
			// Proxy v1.5: read tier from users.metadata so the rules
			// engine's ReqTier predicate evaluates against the latest
			// persisted value for session-cookie flows. Parity with
			// JWTResolver which reads the baked Claims.Tier.
			tier = tierFromMetadata(u.Metadata)
		}
		if rs, err := r.Store.GetRolesByUserID(req.Context(), sess.UserID); err == nil {
			for _, role := range rs {
				if role != nil {
					roles = append(roles, role.Name)
				}
			}
		}
	}

	return proxy.Identity{
		UserID:     sess.UserID,
		UserEmail:  email,
		Roles:      roles,
		Tier:       tier,
		AuthMethod: "session-live",
	}, nil
}

// tierFromMetadata pulls the "tier" key out of the user's metadata JSON
// blob. Mirrors the shape the JWT manager bakes in on issuance — see
// internal/auth/jwt/manager.go userTierFromMetadata. Duplicated here
// (rather than imported) so the proxy-resolvers file stays decoupled
// from jwt.*, and the one-liner cost is negligible.
func tierFromMetadata(metadataJSON string) string {
	if strings.TrimSpace(metadataJSON) == "" {
		return ""
	}
	var md struct {
		Tier string `json:"tier"`
	}
	// Best-effort: corrupted JSON → empty tier (the engine's ReqTier
	// compares strings equal, so empty just fails the predicate).
	_ = json.Unmarshal([]byte(metadataJSON), &md)
	return md.Tier
}

// extractBearer returns the token part of a bearer Authorization header, or
// the empty string when the header is missing / malformed. Case-insensitive
// on the scheme (some clients send "bearer " in lowercase).
func extractBearer(req *http.Request) string {
	h := req.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) >= len(prefix) && strings.EqualFold(h[:len(prefix)], prefix) {
		return strings.TrimSpace(h[len(prefix):])
	}
	return ""
}

// DBAppResolver maps an inbound request Host to an Application upstream.
type DBAppResolver struct {
	Store storage.Store
}

// ResolveApp implements proxy.AppResolver. It looks up the application by its
// proxy_public_domain field and returns a ResolvedApp carrying its upstream URL.
func (r *DBAppResolver) ResolveApp(ctx context.Context, host string) (*proxy.ResolvedApp, error) {
	// Try full host first (e.g. "api.myapp.com:8080"). If that fails, try
	// stripping the port (e.g. "api.myapp.com"). This ensures we match even
	// if the DB row includes - or omits - the port.
	app, err := r.Store.GetApplicationByProxyDomain(ctx, host)
	if (err != nil || app == nil) && strings.Contains(host, ":") {
		h := host
		if idx := strings.Index(host, ":"); idx != -1 {
			h = host[:idx]
		}
		app, err = r.Store.GetApplicationByProxyDomain(ctx, h)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if app == nil {
		return nil, nil
	}

	// ENFORCEMENT: Only resolve if the application is explicitly set to proxy mode.
	if app.IntegrationMode != "proxy" {
		return nil, fmt.Errorf("application %q is not in proxy mode (current: %s)", app.Name, app.IntegrationMode)
	}

	if app.ProxyProtectedURL == "" {
		return nil, errors.New("application has no proxy_protected_url configured")
	}

	u, err := url.Parse(app.ProxyProtectedURL)
	if err != nil {
		return nil, fmt.Errorf("parsing application proxy_protected_url %q: %w", app.ProxyProtectedURL, err)
	}

	// Fetch rules for this application.
	rules, err := r.Store.ListProxyRulesByAppID(ctx, app.ID)
	var engine *proxy.Engine
	if err == nil && len(rules) > 0 {
		ruleSpecs := make([]proxy.RuleSpec, 0, len(rules))
		for _, pr := range rules {
			if !pr.Enabled {
				continue
			}
			ruleSpecs = append(ruleSpecs, proxy.RuleSpec{
				AppID:   pr.AppID,
				Path:    pr.Pattern,
				Methods: pr.Methods,
				Require: pr.Require,
				Allow:   pr.Allow,
				Scopes:  pr.Scopes,
			})
		}
		if len(ruleSpecs) > 0 {
			if e, err := proxy.NewEngine(ruleSpecs); err == nil {
				engine = e
			}
		}
	}

	return &proxy.ResolvedApp{
		ID:              app.ID,
		Slug:            app.Slug,
		IntegrationMode: app.IntegrationMode,
		UpstreamURL:     u,
		Engine:          engine,
	}, nil
}

// Compile-time interface satisfaction checks so a future refactor of
// proxy.AuthResolver surfaces here, not at a wire site.
var (
	_ proxy.AuthResolver = (*JWTResolver)(nil)
	_ proxy.AuthResolver = (*LiveResolver)(nil)
	_ proxy.AppResolver  = (*DBAppResolver)(nil)
)

// Silence unused imports when the file evolves. Keeping proxySessionCookieName
// exposed as a package-level const so future callers don't rehardcode the
// value.
var _ = proxySessionCookieName
