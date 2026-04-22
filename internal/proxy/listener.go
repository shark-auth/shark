package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Listener is a single reverse-proxy listener bound to its own port with
// its own rules engine, circuit breaker, and backing http.Server. Part of
// the W15 multi-listener redesign: one shark binary can protect N apps on
// their native ports without an external reverse proxy out front.
//
// A Listener is inert until Start(ctx) is called. Shutdown(ctx) is safe to
// call whether Start succeeded, failed, or was never invoked — it always
// returns a nil error in the never-started case.
type Listener struct {
	Bind          string
	Upstream      string
	IsTransparent bool

	server   *http.Server
	engine   *Engine
	breaker  *Breaker
	handler  *ReverseProxy
	logger   *slog.Logger

	// authWrap is applied to every request before the proxy handler runs.
	// Wired by the server.Build path to the same JWT + session resolver
	// used by the legacy main-port mount so authentication behaviour is
	// identical across listeners.
	authWrap func(http.Handler) http.Handler

	started bool
}

// ListenerParams bundles the knobs server.Build needs to produce a Listener.
// Kept as a struct (not a long parameter list) so the signature stays stable
// as the listener picks up more policies (TLS, WebSocket, etc. — deferred).
type ListenerParams struct {
	Bind           string
	Upstream       string
	IsTransparent  bool
	Resolver       AppResolver
	Timeout        time.Duration
	TrustedHeaders []string
	StripIncoming  bool
	Rules          []RuleSpec

	BreakerConfig BreakerConfig

	// AuthWrap wraps the proxy handler with identity resolution. Required
	// for proxy rules requiring authenticated identities to work; nil means
	// every request is treated as anonymous.
	AuthWrap func(http.Handler) http.Handler

	Logger *slog.Logger
}

// NewListener builds an inert Listener. It validates config, compiles the
// rule engine, constructs the ReverseProxy, and prepares the *http.Server —
// but does NOT bind the port. Call Start(ctx) to begin serving.
func NewListener(p ListenerParams) (*Listener, error) {
	if p.Bind == "" {
		return nil, errors.New("proxy listener: bind is required")
	}
	if !p.IsTransparent && p.Upstream == "" {
		return nil, errors.New("proxy listener: upstream is required")
	}
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}

	engine, err := NewEngine(p.Rules)
	if err != nil {
		return nil, fmt.Errorf("proxy listener %s: compiling rules: %w", p.Bind, err)
	}

	breaker := NewBreaker(p.BreakerConfig, logger)

	cfg := Config{
		Enabled:        true,
		Upstream:       p.Upstream,
		Timeout:        p.Timeout,
		TrustedHeaders: p.TrustedHeaders,
		StripIncoming:  p.StripIncoming,
	}
	handler, err := New(cfg, engine, logger)
	if err != nil {
		return nil, fmt.Errorf("proxy listener %s: building reverse proxy: %w", p.Bind, err)
	}

	if p.Resolver != nil {
		handler.SetResolver(p.Resolver)
	}

	var h http.Handler = handler
	if p.AuthWrap != nil {
		h = p.AuthWrap(handler)
	}

	srv := &http.Server{
		Addr:              p.Bind,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return &Listener{
		Bind:          p.Bind,
		Upstream:      p.Upstream,
		IsTransparent: p.IsTransparent,
		server:        srv,
		engine:        engine,
		breaker:       breaker,
		handler:       handler,
		logger:        logger,
		authWrap:      p.AuthWrap,
	}, nil
}

// Engine returns the compiled rules engine for admin dashboard consumption.
func (l *Listener) Engine() *Engine { return l.engine }

// Breaker returns the circuit breaker for admin dashboard consumption.
func (l *Listener) Breaker() *Breaker { return l.breaker }

// SetResolver installs a dynamic application resolver.
func (l *Listener) SetResolver(res AppResolver) {
	l.handler.SetResolver(res)
}

// SetAuthWrap installs the identity-resolution middleware. Callers that
// need the listener's own Breaker to build their AuthWrap (the
// server.Build path) construct the Listener first, read l.Breaker(),
// then install the wrap via this setter — avoiding a second NewListener
// call (which would recompile rules and allocate a second Breaker).
//
// Must be called before Start; calling on a running listener is a
// programmer error and returns an error without mutating state.
func (l *Listener) SetAuthWrap(fn func(http.Handler) http.Handler) error {
	if l.started {
		return fmt.Errorf("proxy listener %s: SetAuthWrap after Start", l.Bind)
	}
	l.authWrap = fn
	var h http.Handler = l.handler
	if fn != nil {
		h = fn(l.handler)
	}
	l.server.Handler = h
	return nil
}

// Start binds the port and spawns a serve goroutine. It pre-binds via
// net.Listen so a port-in-use error is returned synchronously (letting the
// caller treat it as a fatal startup error rather than discovering a half
// bound multi-listener set). The returned error is non-nil only when the
// bind itself fails; subsequent ErrServerClosed on graceful Shutdown is
// swallowed here and surfaced via Shutdown's return value.
func (l *Listener) Start(ctx context.Context) error {
	if l.started {
		return fmt.Errorf("proxy listener %s: already started", l.Bind)
	}

	ln, err := net.Listen("tcp", l.Bind)
	if err != nil {
		return fmt.Errorf("proxy listener %s: bind: %w", l.Bind, err)
	}

	l.breaker.Start(ctx)
	l.started = true

	go func() {
		l.logger.Info("proxy listener serving",
			"bind", l.Bind,
			"upstream", l.Upstream,
		)
		if err := l.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.logger.Error("proxy listener fatal",
				"bind", l.Bind,
				"err", err,
			)
		}
	}()

	return nil
}

// Shutdown gracefully drains the listener's http.Server and stops the
// circuit breaker goroutine. Safe to call on a never-started Listener.
func (l *Listener) Shutdown(ctx context.Context) error {
	if !l.started {
		return nil
	}
	if l.breaker != nil {
		l.breaker.Stop()
	}
	return l.server.Shutdown(ctx)
}
