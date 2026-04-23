package proxy

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test helpers --------------------------------------------------------

// silentLogger returns a slog.Logger that discards everything. Tests that
// don't care about log output use this to keep -v output readable.
func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// testBreaker constructs a Breaker pointed at a caller-supplied health URL
// but does NOT start the background monitor. Tests that want to drive the
// state machine deterministically call probe() directly.
func testBreaker(t *testing.T, healthURL string) *Breaker {
	t.Helper()
	b := NewBreaker(BreakerConfig{
		HealthURL:        healthURL,
		HealthInterval:   time.Hour, // effectively disabled
		HealthTimeout:    500 * time.Millisecond,
		FailureThreshold: 3,
		CacheSize:        16,
		CacheTTL:         5 * time.Minute,
		NegativeTTL:      30 * time.Second,
	}, silentLogger())
	t.Cleanup(b.Stop)
	return b
}

// alwaysFailServer returns an httptest server that always returns 500.
func alwaysFailServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unhealthy", http.StatusInternalServerError)
	}))
	t.Cleanup(s.Close)
	return s
}

// alwaysOKServer returns an httptest server that always returns 200.
func alwaysOKServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s.Close)
	return s
}

// toggleServer returns an httptest server whose response code is driven by
// an *atomic.Int32. Tests flip the int to simulate upstream recovery /
// degradation without bouncing the server.
func toggleServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var code atomic.Int32
	code.Store(http.StatusOK)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(code.Load()))
	}))
	t.Cleanup(s.Close)
	return s, &code
}

// mockResolver is a hand-rolled AuthResolver used in BreakerResolver tests.
type mockResolver struct {
	mu       sync.Mutex
	calls    int
	identity Identity
	err      error
}

func (m *mockResolver) Resolve(r *http.Request) (Identity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	return m.identity, m.err
}

func (m *mockResolver) Calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// fixedKeyExtractor makes BreakerResolver use a deterministic cache key
// regardless of the request's actual cookies.
func fixedKeyExtractor(key string) func(*http.Request) (string, bool) {
	return func(*http.Request) (string, bool) { return key, true }
}

// --- State machine tests -------------------------------------------------

func TestBreaker_StartsClosed(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	if got := b.State(); got != Closed {
		t.Fatalf("State() = %v, want Closed", got)
	}
}

func TestBreaker_OpenAfterThresholdFailures(t *testing.T) {
	srv := alwaysFailServer(t)
	b := testBreaker(t, srv.URL)

	// Two failures — still closed.
	b.probe()
	b.probe()
	if got := b.State(); got != Closed {
		t.Fatalf("after 2 failures State() = %v, want Closed", got)
	}
	// Third failure — opens.
	b.probe()
	if got := b.State(); got != Open {
		t.Fatalf("after 3 failures State() = %v, want Open", got)
	}
}

func TestBreaker_ClosesAfterHalfOpenSuccess(t *testing.T) {
	srv, code := toggleServer(t)
	b := testBreaker(t, srv.URL)

	// Fail three times -> Open.
	code.Store(http.StatusInternalServerError)
	for i := 0; i < 3; i++ {
		b.probe()
	}
	if got := b.State(); got != Open {
		t.Fatalf("State = %v, want Open", got)
	}

	// Recover: first success -> HalfOpen.
	code.Store(http.StatusOK)
	b.probe()
	if got := b.State(); got != HalfOpen {
		t.Fatalf("State = %v, want HalfOpen", got)
	}

	// Second success -> Closed.
	b.probe()
	if got := b.State(); got != Closed {
		t.Fatalf("State = %v, want Closed", got)
	}
}

func TestBreaker_ReOpensOnHalfOpenFailure(t *testing.T) {
	srv, code := toggleServer(t)
	b := testBreaker(t, srv.URL)

	// Open the circuit.
	code.Store(http.StatusInternalServerError)
	for i := 0; i < 3; i++ {
		b.probe()
	}
	if got := b.State(); got != Open {
		t.Fatalf("State = %v, want Open", got)
	}

	// One success -> HalfOpen.
	code.Store(http.StatusOK)
	b.probe()
	if got := b.State(); got != HalfOpen {
		t.Fatalf("State = %v, want HalfOpen", got)
	}

	// Next failure -> Open.
	code.Store(http.StatusInternalServerError)
	b.probe()
	if got := b.State(); got != Open {
		t.Fatalf("State = %v, want Open", got)
	}
}

func TestBreaker_FailuresResetOnSuccessInClosed(t *testing.T) {
	srv, code := toggleServer(t)
	b := testBreaker(t, srv.URL)

	// Two failures.
	code.Store(http.StatusInternalServerError)
	b.probe()
	b.probe()
	if got := b.Stats().Failures; got != 2 {
		t.Fatalf("Failures = %d, want 2", got)
	}

	// One success — still closed, counter reset.
	code.Store(http.StatusOK)
	b.probe()
	if got := b.State(); got != Closed {
		t.Fatalf("State = %v, want Closed", got)
	}
	if got := b.Stats().Failures; got != 0 {
		t.Fatalf("Failures = %d, want 0 after reset", got)
	}
}

// --- LRU tests -----------------------------------------------------------

func TestLRU_BasicGetPut(t *testing.T) {
	c := newLRU(4, time.Minute)
	c.put("a", Identity{UserID: "u1"})
	id, age, ok := c.get("a")
	if !ok {
		t.Fatalf("get('a') miss, want hit")
	}
	if id.UserID != "u1" {
		t.Fatalf("UserID = %q, want u1", id.UserID)
	}
	if age < 0 {
		t.Fatalf("age = %v, want >= 0", age)
	}
}

func TestLRU_EvictsOldestAtCapacity(t *testing.T) {
	c := newLRU(2, time.Minute)
	c.put("a", Identity{UserID: "a"})
	c.put("b", Identity{UserID: "b"})
	c.put("c", Identity{UserID: "c"}) // should evict "a"

	if _, _, ok := c.get("a"); ok {
		t.Fatalf("get('a') hit, want miss after eviction")
	}
	if _, _, ok := c.get("b"); !ok {
		t.Fatalf("get('b') miss, want hit")
	}
	if _, _, ok := c.get("c"); !ok {
		t.Fatalf("get('c') miss, want hit")
	}
}

func TestLRU_TTLExpiry(t *testing.T) {
	c := newLRU(4, 10*time.Millisecond)
	base := time.Now()
	c.now = func() time.Time { return base }
	c.put("a", Identity{UserID: "a"})

	// Immediately readable.
	if _, _, ok := c.get("a"); !ok {
		t.Fatalf("immediate get('a') miss")
	}

	// Advance virtual time past TTL.
	c.now = func() time.Time { return base.Add(50 * time.Millisecond) }
	if _, _, ok := c.get("a"); ok {
		t.Fatalf("get('a') hit after expiry, want miss")
	}
	// Expired entry should have been evicted.
	if c.len() != 0 {
		t.Fatalf("len = %d after expiry, want 0", c.len())
	}
}

func TestLRU_UpdatesMoveToFront(t *testing.T) {
	c := newLRU(2, time.Minute)
	c.put("a", Identity{UserID: "a"})
	c.put("b", Identity{UserID: "b"})

	// Touch "a" so it's MRU.
	if _, _, ok := c.get("a"); !ok {
		t.Fatalf("get('a') miss, want hit")
	}

	// Insert "c" -> should evict "b" (LRU), not "a".
	c.put("c", Identity{UserID: "c"})
	if _, _, ok := c.get("a"); !ok {
		t.Fatalf("get('a') miss, want hit (was touched)")
	}
	if _, _, ok := c.get("b"); ok {
		t.Fatalf("get('b') hit, want miss (should have been evicted)")
	}
}

// --- Breaker cache API tests --------------------------------------------

func TestBreaker_Lookup_HitReturnsAge(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	base := time.Now()
	b.cache.now = func() time.Time { return base }
	b.Store("k1", Identity{UserID: "u1", AuthMethod: "session-live"})

	// Advance clock 7 seconds.
	b.cache.now = func() time.Time { return base.Add(7 * time.Second) }
	id, age, ok := b.Lookup("k1")
	if !ok {
		t.Fatalf("Lookup miss, want hit")
	}
	if id.UserID != "u1" {
		t.Fatalf("UserID = %q, want u1", id.UserID)
	}
	if age < 6*time.Second || age > 8*time.Second {
		t.Fatalf("age = %v, want ~7s", age)
	}
}

func TestBreaker_LookupNegative_ShortCircuitsBadTokens(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	if b.LookupNegative("bad") {
		t.Fatalf("LookupNegative on empty cache returned true")
	}
	b.StoreNegative("bad")
	if !b.LookupNegative("bad") {
		t.Fatalf("LookupNegative after store returned false")
	}
}

func TestBreaker_Store_OverwritesExisting(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	b.Store("k1", Identity{UserID: "old"})
	b.Store("k1", Identity{UserID: "new"})
	id, _, ok := b.Lookup("k1")
	if !ok {
		t.Fatalf("Lookup miss after overwrite")
	}
	if id.UserID != "new" {
		t.Fatalf("UserID = %q, want 'new'", id.UserID)
	}
}

func TestBreaker_HashCookie_Stable(t *testing.T) {
	h1 := HashCookie("abc123")
	h2 := HashCookie("abc123")
	if h1 != h2 {
		t.Fatalf("HashCookie not stable: %q vs %q", h1, h2)
	}
	// SHA-256 hex is 64 chars.
	if len(h1) != 64 {
		t.Fatalf("hash len = %d, want 64", len(h1))
	}
}

func TestBreaker_HashCookie_DifferentCookiesDifferentHashes(t *testing.T) {
	if HashCookie("a") == HashCookie("b") {
		t.Fatalf("distinct inputs produced same hash")
	}
}

// --- BreakerResolver tests ----------------------------------------------

// newResolveRequest builds a GET request with an optional session cookie.
// Cookie name/value kept deterministic so the default extractor works in
// tests that don't override it.
func newResolveRequest(method, path, cookieName, cookieValue string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	if cookieName != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	}
	return r
}

func TestBreakerResolver_JWTBypassesBreaker(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	// Force Open state by injecting three failures via onFailureLocked.
	b.mu.Lock()
	b.state = Open
	b.fails = 3
	b.mu.Unlock()

	jwt := &mockResolver{identity: Identity{UserID: "u-jwt", AuthMethod: "jwt"}}
	live := &mockResolver{identity: Identity{UserID: "u-live", AuthMethod: "session-live"}}
	br := &BreakerResolver{
		Breaker:     b,
		JWTResolver: jwt,
		Live:        live,
		Logger:      silentLogger(),
	}

	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	id, err := br.Resolve(r)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if id.AuthMethod != "jwt" {
		t.Fatalf("AuthMethod = %q, want jwt", id.AuthMethod)
	}
	if jwt.Calls() != 1 {
		t.Fatalf("jwt resolver calls = %d, want 1", jwt.Calls())
	}
	if live.Calls() != 0 {
		t.Fatalf("live resolver calls = %d, want 0 (breaker open, JWT path)", live.Calls())
	}
}

func TestBreakerResolver_ClosedCallsLive(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	live := &mockResolver{identity: Identity{UserID: "u-live", AuthMethod: "session-live"}}
	br := &BreakerResolver{
		Breaker:           b,
		Live:              live,
		ExtractSessionKey: fixedKeyExtractor("key1"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}

	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	id, err := br.Resolve(r)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if id.UserID != "u-live" {
		t.Fatalf("UserID = %q, want u-live", id.UserID)
	}
	if live.Calls() != 1 {
		t.Fatalf("live calls = %d, want 1", live.Calls())
	}
	// Next call should hit cache but the breaker is still closed, so the
	// contract (per decision tree) is "call live in closed". We verify
	// that by asserting a second call increments Live.
	_, _ = br.Resolve(r)
	if live.Calls() != 2 {
		t.Fatalf("live calls = %d, want 2 in Closed state", live.Calls())
	}
	// Cache should have been populated regardless.
	if _, _, ok := b.Lookup("key1"); !ok {
		t.Fatalf("cache miss after live success")
	}
}

func TestBreakerResolver_OpenUsesCache_Hit(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	// Prime cache.
	base := time.Now()
	b.cache.now = func() time.Time { return base }
	b.Store("key1", Identity{UserID: "u-cached", AuthMethod: "session-live"})
	// Advance time so the cache age is non-zero.
	b.cache.now = func() time.Time { return base.Add(3 * time.Second) }

	// Open the breaker.
	b.mu.Lock()
	b.state = Open
	b.mu.Unlock()

	live := &mockResolver{} // should not be called
	br := &BreakerResolver{
		Breaker:           b,
		Live:              live,
		ExtractSessionKey: fixedKeyExtractor("key1"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	id, err := br.Resolve(r)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if id.AuthMethod != "session-cached" {
		t.Fatalf("AuthMethod = %q, want session-cached", id.AuthMethod)
	}
	if id.CacheAge == 0 {
		t.Fatalf("CacheAge = 0, want >0 for cached hit")
	}
	if id.UserID != "u-cached" {
		t.Fatalf("UserID = %q, want u-cached", id.UserID)
	}
	if live.Calls() != 0 {
		t.Fatalf("live calls = %d, want 0 (cached hit)", live.Calls())
	}
}

func TestBreakerResolver_OpenCacheMiss_Reject(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	// MissBehavior defaults to reject.
	b.mu.Lock()
	b.state = Open
	b.mu.Unlock()

	br := &BreakerResolver{
		Breaker:           b,
		ExtractSessionKey: fixedKeyExtractor("missing"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	_, err := br.Resolve(r)
	if !errors.Is(err, ErrBreakerOpenNoCache) {
		t.Fatalf("err = %v, want ErrBreakerOpenNoCache", err)
	}
}

func TestBreakerResolver_OpenCacheMiss_AllowReadonly_GET(t *testing.T) {
	b := NewBreaker(BreakerConfig{
		HealthURL:        "http://127.0.0.1:0/nope",
		HealthInterval:   time.Hour,
		HealthTimeout:    500 * time.Millisecond,
		FailureThreshold: 3,
		CacheSize:        16,
		CacheTTL:         time.Minute,
		NegativeTTL:      30 * time.Second,
		MissBehavior:     MissAllowReadonly,
	}, silentLogger())
	t.Cleanup(b.Stop)
	b.mu.Lock()
	b.state = Open
	b.mu.Unlock()

	br := &BreakerResolver{
		Breaker:           b,
		ExtractSessionKey: fixedKeyExtractor("missing"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/read", "shark_session", "s1")
	id, err := br.Resolve(r)
	if err != nil {
		t.Fatalf("Resolve err = %v, want nil for readonly GET", err)
	}
	if id.AuthMethod != AuthMethodDegraded {
		t.Fatalf("AuthMethod = %q, want %q", id.AuthMethod, AuthMethodDegraded)
	}
}

func TestBreakerResolver_OpenCacheMiss_AllowReadonly_POST_Rejects(t *testing.T) {
	b := NewBreaker(BreakerConfig{
		HealthURL:        "http://127.0.0.1:0/nope",
		HealthInterval:   time.Hour,
		HealthTimeout:    500 * time.Millisecond,
		FailureThreshold: 3,
		CacheSize:        16,
		CacheTTL:         time.Minute,
		NegativeTTL:      30 * time.Second,
		MissBehavior:     MissAllowReadonly,
	}, silentLogger())
	t.Cleanup(b.Stop)
	b.mu.Lock()
	b.state = Open
	b.mu.Unlock()

	br := &BreakerResolver{
		Breaker:           b,
		ExtractSessionKey: fixedKeyExtractor("missing"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		r := httptest.NewRequest(method, "/api/write", strings.NewReader(""))
		r.AddCookie(&http.Cookie{Name: "shark_session", Value: "s1"})
		if _, err := br.Resolve(r); !errors.Is(err, ErrBreakerOpenNoCache) {
			t.Fatalf("method %s err = %v, want ErrBreakerOpenNoCache", method, err)
		}
	}
}

func TestBreakerResolver_HalfOpenProbes(t *testing.T) {
	srv, code := toggleServer(t)
	b := testBreaker(t, srv.URL)
	// Open then transition to HalfOpen via probes.
	code.Store(http.StatusInternalServerError)
	for i := 0; i < 3; i++ {
		b.probe()
	}
	code.Store(http.StatusOK)
	b.probe() // Open -> HalfOpen
	if got := b.State(); got != HalfOpen {
		t.Fatalf("State = %v, want HalfOpen", got)
	}

	live := &mockResolver{identity: Identity{UserID: "u-live", AuthMethod: "session-live"}}
	br := &BreakerResolver{
		Breaker:           b,
		Live:              live,
		ExtractSessionKey: fixedKeyExtractor("keyH"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	id, err := br.Resolve(r)
	if err != nil {
		t.Fatalf("Resolve err = %v", err)
	}
	if id.UserID != "u-live" {
		t.Fatalf("UserID = %q, want u-live", id.UserID)
	}
	if live.Calls() != 1 {
		t.Fatalf("live calls = %d, want 1 in HalfOpen", live.Calls())
	}

	// The next probe should now close the circuit.
	b.probe()
	if got := b.State(); got != Closed {
		t.Fatalf("State = %v, want Closed after second success", got)
	}
}

func TestBreakerResolver_NegativeCacheShortCircuit(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	b.StoreNegative("badkey")

	live := &mockResolver{identity: Identity{UserID: "u-live", AuthMethod: "session-live"}}
	br := &BreakerResolver{
		Breaker:           b,
		Live:              live,
		ExtractSessionKey: fixedKeyExtractor("badkey"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	_, err := br.Resolve(r)
	if !errors.Is(err, ErrNegativeCacheHit) {
		t.Fatalf("err = %v, want ErrNegativeCacheHit", err)
	}
	if live.Calls() != 0 {
		t.Fatalf("live calls = %d, want 0 (neg cache short-circuit)", live.Calls())
	}
}

// Extra: live resolver error path caches negatively so subsequent calls
// short-circuit. This backstops the integration flow in P4.
func TestBreakerResolver_LiveErrorPopulatesNegativeCache(t *testing.T) {
	b := testBreaker(t, "http://127.0.0.1:0/nope")
	sentinel := errors.New("401 from auth server")
	live := &mockResolver{err: sentinel}
	br := &BreakerResolver{
		Breaker:           b,
		Live:              live,
		ExtractSessionKey: fixedKeyExtractor("k"),
		HasSessionCookie:  func(*http.Request) bool { return true },
		Logger:            silentLogger(),
	}
	r := newResolveRequest(http.MethodGet, "/api/x", "shark_session", "s1")
	if _, err := br.Resolve(r); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
	if !b.LookupNegative("k") {
		t.Fatalf("negative cache not populated after live error")
	}
}

// --- Stats --------------------------------------------------------------

func TestBreaker_Stats_ReflectsState(t *testing.T) {
	srv := alwaysFailServer(t)
	b := testBreaker(t, srv.URL)

	// Three failures -> Open.
	for i := 0; i < 3; i++ {
		b.probe()
	}
	s := b.Stats()
	if s.State != "open" {
		t.Fatalf("Stats.State = %q, want open", s.State)
	}
	if s.LastStatus != http.StatusInternalServerError {
		t.Fatalf("Stats.LastStatus = %d, want 500", s.LastStatus)
	}
	if s.Failures < 3 {
		t.Fatalf("Stats.Failures = %d, want >= 3", s.Failures)
	}
	if s.HealthURL != srv.URL {
		t.Fatalf("Stats.HealthURL = %q, want %q", s.HealthURL, srv.URL)
	}
	if s.LastCheck.IsZero() {
		t.Fatalf("Stats.LastCheck not populated")
	}
}

// --- Lifecycle: goroutine leak & Stop semantics --------------------------

// TestBreaker_StartStop verifies the monitor goroutine exits cleanly on
// Stop(). If the goroutine leaked the t.Cleanup in testBreaker would block
// CI runs; we exercise the path explicitly here.
func TestBreaker_StartStop(t *testing.T) {
	srv := alwaysOKServer(t)
	b := NewBreaker(BreakerConfig{
		HealthURL:        srv.URL,
		HealthInterval:   20 * time.Millisecond,
		HealthTimeout:    100 * time.Millisecond,
		FailureThreshold: 3,
		CacheSize:        16,
		CacheTTL:         time.Minute,
	}, silentLogger())
	b.Start(t.Context())

	// Let the monitor run long enough for at least one tick.
	time.Sleep(60 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		b.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Stop() did not return within 2s (goroutine leak?)")
	}

	// Idempotent Stop.
	b.Stop()
}
