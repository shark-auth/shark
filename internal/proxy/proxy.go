package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/sharkauth/sharkauth/internal/identity"
	"github.com/sharkauth/sharkauth/internal/oauth"
)

// WithIdentity is a back-compat wrapper that forwards to
// identity.WithIdentity. New code should import internal/identity
// directly and call identity.WithIdentity.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return identity.WithIdentity(ctx, id)
}

// IdentityFromContext is a back-compat wrapper that forwards to
// identity.FromContext. New code should import internal/identity
// directly and call identity.FromContext.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	return identity.FromContext(ctx)
}

// HeaderDenyReason is the response header set when the rules engine
// denies a request. Its value is the Decision.Reason string, which is
// safe to surface — it describes the policy that failed, not any
// caller-supplied data.
const HeaderDenyReason = "X-Shark-Deny-Reason"

// ResolvedApp is the result of a dynamic host lookup.
type ResolvedApp struct {
	ID              string
	Slug            string
	IntegrationMode string
	UpstreamURL     *url.URL
	Engine          *Engine // compiled rules for this specific application
}

// AppResolver is the interface for mapping a request Host to an upstream target.
type AppResolver interface {
	ResolveApp(ctx context.Context, host string) (*ResolvedApp, error)
}

// ReverseProxy is SharkAuth's reverse proxy. It wraps net/http/httputil's
// ReverseProxy with identity header injection, panic recovery, and
// sensible defaults for upstream timeouts. Construct instances with New.
type ReverseProxy struct {
	cfg      Config
	issuer     string   // base URL of SharkAuth for redirects
	upstream   *url.URL // static fallback or main upstream
	resolver   AppResolver
	backend    *httputil.ReverseProxy
	engine     *Engine // static fallback or main engine
	logger     *slog.Logger
	dpopCache  *oauth.DPoPJTICache // nil = DPoP enforcement disabled
}

// New builds a ReverseProxy from cfg. It validates the config, parses the
// upstream URL, and wires the httputil.ReverseProxy director, transport,
// and error handler. If engine is nil the proxy runs in passthrough mode
// — no rule evaluation, all requests forwarded (P1 behavior). If logger
// is nil, slog.Default() is used.
func New(cfg Config, engine *Engine, logger *slog.Logger) (*ReverseProxy, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	var upstream *url.URL
	if cfg.Upstream != "" {
		u, err := url.Parse(cfg.Upstream)
		if err != nil {
			return nil, fmt.Errorf("proxy: parsing upstream URL %q: %w", cfg.Upstream, err)
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("proxy: upstream URL %q must include scheme and host", cfg.Upstream)
		}
		upstream = u
	}

	p := &ReverseProxy{
		cfg:      cfg,
		upstream: upstream,
		engine:   engine,
		logger:   logger,
	}

	// Copy the default transport so we don't mutate http.DefaultTransport
	// (which is shared process-wide). ResponseHeaderTimeout bounds how
	// long we'll wait for the upstream to start replying after the
	// request body has been written.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = timeout

	backend := &httputil.ReverseProxy{
		Director:     p.director,
		Transport:    transport,
		ErrorHandler: p.onUpstreamError,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
	if cfg.BufferSize > 0 {
		backend.BufferPool = nil // rely on httputil internal buffer pool; size knob reserved for future use
	}
	p.backend = backend

	return p, nil
}

// SetIssuer sets the base URL used for redirects to the hosted login page.
func (p *ReverseProxy) SetIssuer(issuer string) {
	p.issuer = issuer
}

// SetResolver installs a dynamic application resolver. If set, the proxy
// will prioritize host-based resolution over its static upstream.
func (p *ReverseProxy) SetResolver(res AppResolver) {
	p.resolver = res
}

// SetDPoPCache wires a DPoP JTI replay cache into the proxy. When set,
// ServeHTTP validates the DPoP proof header against the bearer access
// token for any request whose resolved Identity.AuthMethod is
// AuthMethodDPoP — invalid/missing proofs produce 401 with the
// X-Shark-Deny-Reason header describing the failure.
//
// Passing nil disables DPoP enforcement; the proxy will forward
// DPoP-authenticated requests without validating the proof (useful in
// tests and in deployments where an upstream edge handles DPoP).
func (p *ReverseProxy) SetDPoPCache(cache *oauth.DPoPJTICache) {
	p.dpopCache = cache
}

// ServeHTTP implements http.Handler. It:
//  1. recovers from panics and replies 503 so a buggy handler or
//     transport never takes down the server;
//  2. strips client-supplied identity headers (unless StripIncoming is
//     explicitly disabled);
//  3. injects the request's resolved Identity (or an "anonymous" marker
//     when none is present);
//  4. applies the configured upstream timeout via context; and
//  5. delegates to the underlying httputil.ReverseProxy.
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			p.logger.Error("proxy panic",
				"panic", rec,
				"path", r.URL.Path,
				"method", r.Method,
			)
			// Best-effort: if headers were already written we can't do
			// anything meaningful; http.Error will just log "superfluous
			// response.WriteHeader" and carry on.
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		}
	}()

	if p.cfg.StripIncoming {
		StripIdentityHeaders(r.Header, p.cfg.TrustedHeaders)
	}

	id, ok := IdentityFromContext(r.Context())
	if !ok {
		// Passthrough for anonymous traffic. The rules engine decides
		// whether this is permitted; annotating happens whether the
		// engine allows or denies.
		id = Identity{AuthMethod: identity.AuthMethodAnonymous}
	}

	// DPoP proof-of-possession check. Only runs when the identity was
	// resolved via a DPoP-bound bearer (AuthMethodDPoP) and the operator
	// wired a JTI cache into the proxy. If either is absent we trust
	// whatever the upstream middleware decided.
	if id.AuthMethod == identity.AuthMethodDPoP && p.dpopCache != nil {
		if err := p.validateDPoPProof(r); err != nil {
			p.logger.Info("proxy dpop rejected",
				"path", r.URL.Path,
				"method", r.Method,
				"user_id", id.UserID,
				"err", err,
			)
			w.Header().Set(HeaderDenyReason, err.Error())
			http.Error(w, "unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
	}

	// Dynamic Resolution (W15 Transparent Mode)
	var resolved *ResolvedApp
	if p.resolver != nil {
		app, err := p.resolver.ResolveApp(r.Context(), r.Host)
		if err != nil {
			p.logger.Error("proxy resolution error", "host", r.Host, "err", err)
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		if app != nil {
			resolved = app
		}
	}

	// Route-level authorization.
	engine := p.engine
	if resolved != nil {
		if resolved.Engine != nil {
			engine = resolved.Engine
		}
		// In transparent mode, if the AppID header is missing, inject it so
		// Engine.Evaluate can match rules scoped to this application.
		if r.Header.Get("X-Shark-App-ID") == "" {
			r.Header.Set("X-Shark-App-ID", resolved.ID)
		}
	}

	if engine != nil {
		decision := engine.Evaluate(r, id)

		// If no rule matched in the (possibly app-specific) engine, fall back
		// to the global engine if we're currently using an app-specific one.
		if decision.MatchedRule == nil && resolved != nil && resolved.Engine != nil && p.engine != nil {
			decision = p.engine.Evaluate(r, id)
		}

		if !decision.Allow {
			p.logger.Info("proxy denied",
				"path", r.URL.Path,
				"method", r.Method,
				"reason", decision.Reason,
				"user_id", id.UserID,
				"agent_id", id.AgentID,
				"host", r.Host,
			)

			// Porter Logic: Browser Redirect vs API 401/403
			if isAnonymous(id) {
				// We only redirect if the app is in "proxy" mode AND it's a browser.
				// If not resolved to a specific app, we treat it as global proxy (if issuer set).
				isProxyMode := true
				if resolved != nil && resolved.IntegrationMode != "" {
					isProxyMode = resolved.IntegrationMode == "proxy"
				}

				if isProxyMode && isBrowser(r) && p.issuer != "" {
					slug := "default"
					if resolved != nil && resolved.Slug != "" {
						slug = resolved.Slug
					}
					loginURL := fmt.Sprintf("%s/hosted/%s/login?return_to=%s",
						p.issuer, slug, url.QueryEscape(fullURL(r)))
					http.Redirect(w, r, loginURL, http.StatusFound)
					return
				}

				w.Header().Set(HeaderDenyReason, decision.Reason)
				http.Error(w, "unauthorized: "+decision.Reason, http.StatusUnauthorized)
				return
			}

			// Authenticated but forbidden (missing role/scope)
			w.Header().Set(HeaderDenyReason, decision.Reason)
			http.Error(w, "forbidden: "+decision.Reason, http.StatusForbidden)
			return
		}
	}

	InjectIdentity(r.Header, id)

	// Bound the whole upstream round-trip. Using context (rather than
	// http.Client.Timeout) lets the transport cancel in-flight reads
	// cleanly and plays nicely with httputil.ReverseProxy's streaming.
	timeout := p.cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// If resolved, stash it in context for the director
	if resolved != nil {
		ctx = context.WithValue(ctx, resolvedAppCtxKey{}, resolved)
	}

	// Clear SharkAuth-specific security headers set by the global middleware
	// so they don't interfere with the upstream application. If the upstream
	// wants these headers, it will set them itself.
	h := w.Header()
	h.Del("Content-Security-Policy")
	h.Del("X-Frame-Options")
	h.Del("X-Content-Type-Options")
	h.Del("Referrer-Policy")
	h.Del("X-XSS-Protection")
	h.Del("Permissions-Policy")
	h.Del("Strict-Transport-Security")

	p.backend.ServeHTTP(w, r.WithContext(ctx))
}

// isBrowser heuristically checks if the request is likely from a web browser
// by looking for "text/html" in the Accept header.
func isBrowser(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/html") {
		return true
	}
	// Check fetch metadata (modern browsers)
	if r.Header.Get("Sec-Fetch-Mode") == "navigate" {
		return true
	}
	// Fallback: check UA for common browsers if Accept is generic
	ua := r.Header.Get("User-Agent")
	return strings.Contains(ua, "Mozilla/") || strings.Contains(ua, "Chrome/") || strings.Contains(ua, "Safari/")
}

// fullURL reconstructs the full inbound URL for the return_to parameter.
func fullURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.RequestURI())
}

type resolvedAppCtxKey struct{}

// director rewrites the outbound request's Scheme/Host to target the
// configured upstream while preserving the inbound Path and RawQuery.
// It is called by httputil.ReverseProxy on every request.
func (p *ReverseProxy) director(req *http.Request) {
	target := p.upstream

	// Priority 1: Dynamic resolved app from context
	if res, ok := req.Context().Value(resolvedAppCtxKey{}).(*ResolvedApp); ok && res.UpstreamURL != nil {
		target = res.UpstreamURL
	}

	if target == nil {
		// No static upstream and resolution failed/missing.
		return
	}

	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host

	// Preserve and join the path from the upstream URL with the request path.
	// This ensures that if upstream is http://localhost:8080/prefix, a request
	// for /foo becomes /prefix/foo.
	if target.Path != "" {
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
	}

	// httputil.ReverseProxy's default director also sets User-Agent if
	// missing; replicate that behavior so our custom director doesn't
	// regress header handling.
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "")
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// onUpstreamError is the ErrorHandler given to httputil.ReverseProxy. It
// logs the underlying error for operators but returns a generic 502 to
// the client to avoid leaking transport internals.
func (p *ReverseProxy) onUpstreamError(w http.ResponseWriter, r *http.Request, err error) {
	// Distinguish context deadline exceeded (timeout) from other errors
	// in the log so operators can spot slow upstreams quickly. Both
	// still surface to the client as 502 — the proxy doesn't leak
	// whether the upstream was unreachable vs slow.
	if ctxErr := r.Context().Err(); ctxErr == context.DeadlineExceeded {
		p.logger.Warn("proxy upstream timeout",
			"path", r.URL.Path,
			"method", r.Method,
			"timeout", p.effectiveTimeout(),
		)
	} else {
		p.logger.Warn("proxy upstream error",
			"path", r.URL.Path,
			"method", r.Method,
			"error", err,
		)
	}
	http.Error(w, "upstream unreachable", http.StatusBadGateway)
}

// isAnonymous reports whether id carries no authenticated principal.
// Used by the deny path to distinguish "unauthenticated" (401) from
// "authenticated-but-unauthorized" (403).
func isAnonymous(id Identity) bool {
	return id.UserID == "" && id.AgentID == ""
}

// validateDPoPProof validates the DPoP proof header on r against the
// bearer access token per RFC 9449 §4.3. Called only when
// p.dpopCache is non-nil and the resolved identity used AuthMethodDPoP.
//
// htu is constructed from the request as scheme://host/path — query and
// fragment are explicitly excluded because RFC 9449 §4.2 defines htu
// that way. Scheme defaults to "https" when unset (the usual case
// behind a TLS terminator that rewrites r.URL.Scheme to empty).
//
// Returns a concise error whose Error() is safe to surface on the
// X-Shark-Deny-Reason header; callers set the header + 401.
func (p *ReverseProxy) validateDPoPProof(r *http.Request) error {
	proof := r.Header.Get("DPoP")
	if proof == "" {
		return fmt.Errorf("dpop proof missing")
	}

	authz := r.Header.Get("Authorization")
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authz, bearerPrefix) {
		return fmt.Errorf("dpop proof invalid: bearer token missing")
	}
	bearer := strings.TrimSpace(strings.TrimPrefix(authz, bearerPrefix))
	if bearer == "" {
		return fmt.Errorf("dpop proof invalid: bearer token empty")
	}

	scheme := r.URL.Scheme
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	htu := scheme + "://" + r.Host + r.URL.Path

	ath := oauth.HashAccessTokenForDPoP(bearer)
	if _, err := oauth.ValidateDPoPProof(proof, r.Method, htu, ath, p.dpopCache); err != nil {
		return fmt.Errorf("dpop proof invalid")
	}
	return nil
}

// statusText is a small helper that returns a canonical English phrase
// for the few statuses the deny path emits, so callers don't have to
// switch on status codes inline.
func statusText(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	default:
		return http.StatusText(code)
	}
}

// effectiveTimeout returns the timeout actually applied to outbound
// requests: cfg.Timeout if set, otherwise DefaultTimeout.
func (p *ReverseProxy) effectiveTimeout() time.Duration {
	if p.cfg.Timeout > 0 {
		return p.cfg.Timeout
	}
	return DefaultTimeout
}
