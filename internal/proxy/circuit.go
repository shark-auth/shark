package proxy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// BreakerState is the circuit breaker's operating mode.
//
//	closed ──(N fails)──> open ──(success)──> half-open ──(success)──> closed
//	                                                   └──(fail)────> open
//
// "closed" is the healthy default (requests flow to the auth server).
// "open" falls back to the session cache. "half-open" admits a single probe
// to decide whether to recover (close) or re-open.
type BreakerState int

const (
	// Closed: auth server is healthy; every request hits it live.
	Closed BreakerState = iota
	// Open: auth server is unhealthy; serve from the session cache.
	Open
	// HalfOpen: one probe in flight to decide whether to recover.
	HalfOpen
)

// String returns the lowercase state name ("closed" | "open" | "half-open")
// suitable for logging, JSON, or the SSE status stream consumed by the
// dashboard (Phase 6 P4).
func (s BreakerState) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// AuthMethodDegraded is set on the Identity returned when MissBehavior is
// "allow_readonly" and the circuit is open with no cache entry. Upstream
// services can see this via X-Shark-Auth-Mode and degrade their own UX
// (e.g. hide write buttons) accordingly.
const AuthMethodDegraded = "anonymous-degraded"

// MissBehavior values for BreakerConfig.MissBehavior.
const (
	MissReject        = "reject"
	MissAllowReadonly = "allow_readonly"
)

// ErrBreakerOpenNoCache is returned by BreakerResolver.Resolve when the
// circuit is open, the session is not cached, and MissBehavior is "reject"
// (or "allow_readonly" with a mutating method). Callers should translate
// this to a 503 response so clients know the degradation is transient.
var ErrBreakerOpenNoCache = errors.New("proxy: circuit open and session not cached")

// ErrNegativeCacheHit is returned by BreakerResolver.Resolve when the
// supplied session cookie is in the negative cache — the auth server has
// recently denied it. Short-circuiting on the proxy side prevents a
// stampede of known-401 lookups when, e.g., an attacker sprays stolen
// cookies.
var ErrNegativeCacheHit = errors.New("proxy: session known-bad (negative cache hit)")

// BreakerConfig configures a Breaker. Zero values fall back to the
// documented defaults so callers can opt out of tuning for all but a few
// knobs.
type BreakerConfig struct {
	// HealthURL is the auth server's health endpoint, e.g.
	// "http://localhost:8080/api/v1/admin/health". Must return 2xx when
	// healthy; 5xx or a transport error counts as a failure.
	HealthURL string

	// HealthInterval is the spacing between probes. Default 10s.
	HealthInterval time.Duration

	// HealthTimeout bounds each probe. Default 3s.
	HealthTimeout time.Duration

	// FailureThreshold is the number of consecutive failures that open the
	// circuit. Default 3.
	FailureThreshold int

	// CacheSize is the LRU capacity for positive lookups. Default 10000.
	CacheSize int

	// CacheTTL is the per-entry TTL for positive lookups. Default 5m.
	CacheTTL time.Duration

	// NegativeTTL is the per-entry TTL for the known-bad cache. Default
	// 30s — intentionally much shorter than CacheTTL so legitimate users
	// whose sessions momentarily failed aren't locked out for long.
	NegativeTTL time.Duration

	// MissBehavior controls what happens when the circuit is open and the
	// session isn't cached. "reject" (default) returns ErrBreakerOpenNoCache
	// — safest, the client sees a 503 and retries later. "allow_readonly"
	// returns an anonymous-degraded Identity for GET/HEAD so read-only
	// dashboards keep working through brief auth outages.
	MissBehavior string
}

// withDefaults returns a copy of cfg with zero fields replaced by package
// defaults. Keeping defaults out of the struct literal keeps New() callers
// decoupled from tuning changes.
func (cfg BreakerConfig) withDefaults() BreakerConfig {
	if cfg.HealthInterval <= 0 {
		cfg.HealthInterval = 10 * time.Second
	}
	if cfg.HealthTimeout <= 0 {
		cfg.HealthTimeout = 3 * time.Second
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 3
	}
	if cfg.CacheSize <= 0 {
		cfg.CacheSize = 10000
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = 5 * time.Minute
	}
	if cfg.NegativeTTL <= 0 {
		cfg.NegativeTTL = 30 * time.Second
	}
	if cfg.MissBehavior == "" {
		cfg.MissBehavior = MissReject
	}
	return cfg
}

// Breaker is the circuit breaker: session cache + background health monitor
// + state machine. All public methods are safe for concurrent use.
type Breaker struct {
	cfg BreakerConfig

	mu    sync.RWMutex
	state BreakerState
	fails int

	cache    *lruCache // positive cache (authed identities)
	negCache *lruCache // negative cache (known-bad tokens)

	logger *slog.Logger

	client *http.Client
	stopCh chan struct{}
	doneCh chan struct{}
	started bool

	// Observability snapshot (populated by probe(); read by Stats()).
	lastCheck   time.Time
	lastLatency time.Duration
	lastStatus  int
}

// CachedIdentity pairs an Identity with its cache age, mirroring what
// Breaker.Lookup returns so callers that want a single struct to pass
// around can use it.
type CachedIdentity struct {
	Identity Identity
	CachedAt time.Time
}

// NewBreaker constructs a Breaker with the given config. The background
// health monitor is NOT started — call Start() to begin probing. A nil
// logger is replaced with slog.Default().
func NewBreaker(cfg BreakerConfig, logger *slog.Logger) *Breaker {
	if logger == nil {
		logger = slog.Default()
	}
	cfg = cfg.withDefaults()
	return &Breaker{
		cfg:      cfg,
		state:    Closed,
		cache:    newLRU(cfg.CacheSize, cfg.CacheTTL),
		negCache: newLRU(cfg.CacheSize, cfg.NegativeTTL),
		logger:   logger,
		client: &http.Client{
			Timeout: cfg.HealthTimeout,
		},
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start begins the background health-monitor goroutine. Safe to call at
// most once per Breaker instance; subsequent calls are no-ops. The monitor
// runs until ctx is cancelled or Stop() is invoked.
func (b *Breaker) Start(ctx context.Context) {
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return
	}
	b.started = true
	b.mu.Unlock()

	go b.run(ctx)
}

// Stop signals the monitor to exit and blocks until it has done so. Safe
// to call even if Start was never invoked — it just closes the stop
// channel and returns.
func (b *Breaker) Stop() {
	b.mu.Lock()
	if !b.started {
		// No monitor running; just mark stopCh closed so a later Start
		// would be a no-op. Using a sync.Once-like guard with a local
		// boolean keeps us stdlib-only and deterministic.
		b.started = true
		close(b.stopCh)
		close(b.doneCh)
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	select {
	case <-b.stopCh:
		// Already stopped — wait for done.
	default:
		close(b.stopCh)
	}
	<-b.doneCh
}

// State returns the current breaker state. Read-locks the mutex so callers
// can safely poll from multiple goroutines.
func (b *Breaker) State() BreakerState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// BreakerStats is the observability snapshot surfaced to the dashboard and
// the SSE status stream. All fields are point-in-time; callers that need a
// rate should sample twice and diff.
type BreakerStats struct {
	State        string
	CacheSize    int
	NegCacheSize int
	Failures     int
	LastCheck    time.Time
	LastLatency  time.Duration
	LastStatus   int
	HealthURL    string
}

// Stats returns the current observability snapshot. Safe to call from any
// goroutine; holds only the read lock.
func (b *Breaker) Stats() BreakerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return BreakerStats{
		State:        b.state.String(),
		CacheSize:    b.cache.len(),
		NegCacheSize: b.negCache.len(),
		Failures:     b.fails,
		LastCheck:    b.lastCheck,
		LastLatency:  b.lastLatency,
		LastStatus:   b.lastStatus,
		HealthURL:    b.cfg.HealthURL,
	}
}

// Lookup checks the positive cache for a session cookie hash. Returns
// (identity, age, true) on hit or (zero, 0, false) on miss or expiry.
// Callers should pass the hex-SHA-256 returned by HashCookie, never the
// raw cookie value — see HashCookie's doc.
func (b *Breaker) Lookup(cookieHash string) (Identity, time.Duration, bool) {
	return b.cache.get(cookieHash)
}

// LookupNegative reports whether cookieHash is in the known-bad cache and
// still within its TTL. Callers must check this BEFORE calling the live
// auth server — a negative hit means "auth server already said 401, don't
// bother asking again for a while".
func (b *Breaker) LookupNegative(cookieHash string) bool {
	_, _, ok := b.negCache.get(cookieHash)
	return ok
}

// Store caches a positive lookup. Safe for concurrent use. Overwriting an
// existing entry resets its age and TTL.
func (b *Breaker) Store(cookieHash string, id Identity) {
	b.cache.put(cookieHash, id)
}

// StoreNegative caches a known-bad session. Overwriting an existing
// negative entry resets its TTL — that's intentional: each failed lookup
// extends the short-circuit window, which is desirable for the replay-
// attack case.
func (b *Breaker) StoreNegative(cookieHash string) {
	b.negCache.put(cookieHash, Identity{})
}

// HashCookie returns a stable hex-SHA-256 hash of the raw cookie value,
// suitable as a cache key. We never key the cache by the raw cookie so
// that a stray log of the key doesn't leak a live session to the reader.
// The output is deterministic: same input => same output, always.
func HashCookie(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// run is the health-monitor goroutine. Probes once on start, then every
// HealthInterval. Exits when ctx is done or Stop() closes stopCh. Always
// closes doneCh so Stop() can join.
func (b *Breaker) run(ctx context.Context) {
	defer close(b.doneCh)

	// Probe immediately so a broken upstream is discovered before the
	// first request — without this the first HealthInterval would elapse
	// in Closed state regardless.
	b.probe()

	ticker := time.NewTicker(b.cfg.HealthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.probe()
		}
	}
}

// probe performs a single health check and updates state under the lock.
// Counts a 5xx or transport error as a failure; everything else (2xx, 3xx,
// 4xx) as a success — a 4xx from the health endpoint means the server is
// up and answering, it just doesn't like our request shape, which isn't a
// reason to open the circuit.
func (b *Breaker) probe() {
	ctx, cancel := context.WithTimeout(context.Background(), b.cfg.HealthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.cfg.HealthURL, nil)
	started := time.Now()
	var (
		status   int
		probeErr error
	)
	if err != nil {
		probeErr = err
	} else {
		resp, doErr := b.client.Do(req)
		if doErr != nil {
			probeErr = doErr
		} else {
			status = resp.StatusCode
			_ = resp.Body.Close()
		}
	}
	latency := time.Since(started)

	b.mu.Lock()
	defer b.mu.Unlock()

	b.lastCheck = time.Now()
	b.lastLatency = latency
	b.lastStatus = status

	if probeErr != nil || status >= 500 {
		b.onFailureLocked(probeErr, status)
	} else {
		b.onSuccessLocked()
	}
}

// onFailureLocked handles a failed probe. Caller holds b.mu. Transitions:
//   - Closed + fails>=threshold -> Open
//   - HalfOpen                  -> Open (probe failed)
//   - Open                      -> Open (no-op, just increment counter)
func (b *Breaker) onFailureLocked(err error, status int) {
	b.fails++
	switch b.state {
	case Closed:
		if b.fails >= b.cfg.FailureThreshold {
			b.state = Open
			b.logger.Warn("proxy circuit opened",
				"fails", b.fails,
				"status", status,
				"err", err,
			)
		}
	case HalfOpen:
		b.state = Open
		b.logger.Warn("proxy circuit re-opened after half-open probe failed",
			"status", status,
			"err", err,
		)
	}
}

// onSuccessLocked handles a successful probe. Caller holds b.mu. Transitions:
//   - Closed   -> Closed (reset counter)
//   - Open     -> HalfOpen (one successful probe starts recovery)
//   - HalfOpen -> Closed (second success confirms recovery)
func (b *Breaker) onSuccessLocked() {
	switch b.state {
	case Closed:
		b.fails = 0
	case Open:
		b.state = HalfOpen
		b.fails = 0
		b.logger.Info("proxy circuit entering half-open (probe recovered)")
	case HalfOpen:
		b.state = Closed
		b.fails = 0
		b.logger.Info("proxy circuit closed (fully recovered)")
	}
}

// AuthResolver resolves an inbound request to an Identity. Implementations
// plug different auth strategies into the proxy chain:
//   - a local-verify JWT resolver (stateless, never blocked by the breaker)
//   - a live session resolver that calls the auth server
//   - the BreakerResolver below, which composes the two with caching
//
// A nil error + zero Identity means "no credential in the request" —
// downstream code treats the request as anonymous. A non-nil error means
// the credential was present but invalid (or the auth server is down and
// MissBehavior said to reject).
type AuthResolver interface {
	Resolve(r *http.Request) (Identity, error)
}

// BreakerResolver composes a live AuthResolver with the circuit breaker's
// cache and state machine. It is the single entry point P4 (router
// integration) will use for session authentication. JWT authentication
// bypasses the breaker entirely via the JWTResolver field — local
// verification never depends on the auth server's availability.
type BreakerResolver struct {
	// Breaker is the circuit breaker supplying cache + state.
	Breaker *Breaker

	// Live resolves a session cookie by calling the auth server. Used in
	// Closed and HalfOpen states. May return an error to signal a 401/403
	// from the auth server; BreakerResolver caches that as a negative
	// entry so future lookups short-circuit.
	Live AuthResolver

	// JWTResolver resolves a request bearing a JWT locally (JWKS cached
	// in-process). Called first — if it returns a non-zero Identity we
	// never touch the breaker, regardless of state.
	JWTResolver AuthResolver

	// HasSessionCookie reports whether the request carries something the
	// Live resolver would accept. Default (nil) uses a cookie presence
	// heuristic keyed on "session" or "shark_session". Tests can inject a
	// custom predicate.
	HasSessionCookie func(*http.Request) bool

	// ExtractSessionKey returns the cache key (hex-SHA-256 of the raw
	// cookie value). Default (nil) reads the "shark_session" cookie; tests
	// can inject a custom extractor so the resolver doesn't need a real
	// HTTP request.
	ExtractSessionKey func(*http.Request) (string, bool)

	Logger *slog.Logger
}

// defaultHasSessionCookie is the default predicate for HasSessionCookie.
// It considers either "shark_session" or "session" (legacy) as a session-
// bearing cookie. The name duplication accepts both current and legacy
// SharkAuth deployments.
func defaultHasSessionCookie(r *http.Request) bool {
	if _, err := r.Cookie("shark_session"); err == nil {
		return true
	}
	if _, err := r.Cookie("session"); err == nil {
		return true
	}
	return false
}

// defaultExtractSessionKey reads the shark_session (preferred) or session
// cookie and returns its hex-SHA-256 hash. Returns (_, false) if neither
// is present or the value is empty.
func defaultExtractSessionKey(r *http.Request) (string, bool) {
	for _, name := range []string{"shark_session", "session"} {
		c, err := r.Cookie(name)
		if err == nil && c.Value != "" {
			return HashCookie(c.Value), true
		}
	}
	return "", false
}

// logger returns br.Logger or slog.Default() so the resolver is safe to
// construct without wiring a logger.
func (br *BreakerResolver) logger() *slog.Logger {
	if br.Logger != nil {
		return br.Logger
	}
	return slog.Default()
}

// hasSession dispatches to the injected predicate or the default.
func (br *BreakerResolver) hasSession(r *http.Request) bool {
	if br.HasSessionCookie != nil {
		return br.HasSessionCookie(r)
	}
	return defaultHasSessionCookie(r)
}

// sessionKey dispatches to the injected extractor or the default.
func (br *BreakerResolver) sessionKey(r *http.Request) (string, bool) {
	if br.ExtractSessionKey != nil {
		return br.ExtractSessionKey(r)
	}
	return defaultExtractSessionKey(r)
}

// Resolve implements AuthResolver. The decision tree:
//
//  1. JWT path. If JWTResolver returns a non-zero Identity, use it (never
//     blocked by the breaker, never consults the cache).
//  2. Session path.
//     a. Negative cache hit -> short-circuit with ErrNegativeCacheHit.
//     b. State == Closed:   call Live; cache positive or negative result.
//     c. State == Open:     Lookup cache.
//     - hit: return cached identity with AuthMethod="session-cached"
//     and CacheAge set from the cache.
//     - miss + reject: return ErrBreakerOpenNoCache.
//     - miss + allow_readonly + GET/HEAD: return anonymous-degraded.
//     - miss + allow_readonly + other methods: ErrBreakerOpenNoCache.
//     d. State == HalfOpen: try Live; on success let the breaker close on
//     the next probe (BreakerResolver doesn't drive the state machine
//     directly — the health monitor is the single source of truth).
//  3. No credential. Return zero Identity, nil error — anonymous.
func (br *BreakerResolver) Resolve(r *http.Request) (Identity, error) {
	// 1. JWT first — stateless, never blocked by the breaker.
	if br.JWTResolver != nil {
		id, err := br.JWTResolver.Resolve(r)
		if err != nil {
			return Identity{}, err
		}
		if id.AuthMethod != "" {
			return id, nil
		}
	}

	if !br.hasSession(r) {
		// No session cookie and no JWT — anonymous.
		return Identity{}, nil
	}

	key, ok := br.sessionKey(r)
	if !ok {
		// HasSessionCookie said yes but we couldn't extract a key; treat
		// as anonymous (a malformed cookie shouldn't hard-fail the whole
		// proxy).
		return Identity{}, nil
	}

	// 2a. Negative cache short-circuit.
	if br.Breaker.LookupNegative(key) {
		return Identity{}, ErrNegativeCacheHit
	}

	state := br.Breaker.State()

	switch state {
	case Open:
		if id, age, hit := br.Breaker.Lookup(key); hit {
			id.AuthMethod = "session-cached"
			id.CacheAge = age
			return id, nil
		}
		return br.handleOpenMiss(r)

	case Closed, HalfOpen:
		id, err := br.callLive(r, key)
		if err != nil {
			return Identity{}, err
		}
		return id, nil
	}

	// Unreachable — all states handled above. Compiler can't prove it.
	return Identity{}, nil
}

// callLive dispatches to the Live resolver and caches the result. A
// successful lookup caches the Identity; a failure caches the negative
// marker so subsequent requests with the same cookie short-circuit for
// the NegativeTTL window.
func (br *BreakerResolver) callLive(r *http.Request, key string) (Identity, error) {
	if br.Live == nil {
		// No live resolver wired — treat as anonymous. Defensive: in
		// production P4 always wires one.
		return Identity{}, nil
	}
	id, err := br.Live.Resolve(r)
	if err != nil {
		br.Breaker.StoreNegative(key)
		return Identity{}, err
	}
	if id.AuthMethod == "" {
		// Live returned "no credential present" — don't cache that. The
		// caller's next attempt (with, say, a refreshed cookie) should go
		// live again.
		return id, nil
	}
	// Clear CacheAge; it's computed on read from the cache, not stored.
	cached := id
	cached.CacheAge = 0
	br.Breaker.Store(key, cached)
	return id, nil
}

// handleOpenMiss applies the MissBehavior policy. For "allow_readonly" we
// permit GET and HEAD only — never anything that can mutate state, because
// a degraded anonymous identity shouldn't be able to write.
func (br *BreakerResolver) handleOpenMiss(r *http.Request) (Identity, error) {
	if br.Breaker.cfg.MissBehavior == MissAllowReadonly {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			return Identity{AuthMethod: AuthMethodDegraded}, nil
		}
	}
	return Identity{}, ErrBreakerOpenNoCache
}
