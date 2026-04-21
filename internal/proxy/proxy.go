package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// identityCtxKey is the private context key used to stash the resolved
// Identity on a request. A struct type (not a string) is used to prevent
// collisions with other packages per the context.WithValue convention.
type identityCtxKey struct{}

// WithIdentity returns a copy of ctx carrying id. The auth middleware
// calls this once it has resolved the request's principal; the proxy
// retrieves the value via IdentityFromContext.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityCtxKey{}, id)
}

// IdentityFromContext returns the Identity stashed by WithIdentity and
// whether one was present. When ok is false callers should treat the
// request as anonymous.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityCtxKey{}).(Identity)
	return id, ok
}

// HeaderDenyReason is the response header set when the rules engine
// denies a request. Its value is the Decision.Reason string, which is
// safe to surface — it describes the policy that failed, not any
// caller-supplied data.
const HeaderDenyReason = "X-Shark-Deny-Reason"

// ReverseProxy is SharkAuth's reverse proxy. It wraps net/http/httputil's
// ReverseProxy with identity header injection, panic recovery, and
// sensible defaults for upstream timeouts. Construct instances with New.
type ReverseProxy struct {
	cfg      Config
	upstream *url.URL
	backend  *httputil.ReverseProxy
	engine   *Engine
	logger   *slog.Logger
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
		id = Identity{AuthMethod: "anonymous"}
	}

	// Route-level authorization. nil engine = passthrough (legacy P1
	// behavior preserved so existing tests and disabled-rules deployments
	// keep working). On deny, write the response here and return — the
	// upstream must never be contacted.
	if p.engine != nil {
		decision := p.engine.Evaluate(r, id)
		if !decision.Allow {
			p.logger.Info("proxy denied",
				"path", r.URL.Path,
				"method", r.Method,
				"reason", decision.Reason,
				"user_id", id.UserID,
				"agent_id", id.AgentID,
			)
			w.Header().Set(HeaderDenyReason, decision.Reason)
			// W15b: unauthenticated callers get 401 — the spec semantics
			// ("send credentials to proceed"). 403 is reserved for
			// authenticated-but-unauthorized (authenticated caller missing
			// a role, scope, or permission the rule demands).
			status := http.StatusForbidden
			if isAnonymous(id) {
				status = http.StatusUnauthorized
			}
			http.Error(w, statusText(status)+": "+decision.Reason, status)
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

	p.backend.ServeHTTP(w, r.WithContext(ctx))
}

// director rewrites the outbound request's Scheme/Host to target the
// configured upstream while preserving the inbound Path and RawQuery.
// It is called by httputil.ReverseProxy on every request.
func (p *ReverseProxy) director(req *http.Request) {
	if p.upstream == nil {
		// Should be impossible once Validate has been called on a cfg
		// with Enabled=true; defensive so a mis-wired disabled proxy
		// doesn't panic.
		return
	}
	req.URL.Scheme = p.upstream.Scheme
	req.URL.Host = p.upstream.Host
	req.Host = p.upstream.Host
	// httputil.ReverseProxy's default director also sets User-Agent if
	// missing; replicate that behavior so our custom director doesn't
	// regress header handling.
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "")
	}
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
