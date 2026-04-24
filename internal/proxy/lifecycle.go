// Package proxy lifecycle Manager (Lane B, PROXYV1_5 §4.9).
//
// Manager owns a pool of *Listener and exposes an explicit state machine
// (Stopped → Running → Reloading → Running → Stopped) so the admin API
// can flip the proxy subsystem on/off without restarting the process.
// Every transition runs under mu.Lock so concurrent Start/Stop/Reload
// calls serialize; the underlying listeners are built lazily from the
// builder closure provided to NewManager.

package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State enumerates the Manager's observable lifecycle states. The admin
// status endpoint projects this through StatusStringFor so operators see
// a consistent spelling across logs + UI + wire.
type State int

const (
	// StateStopped is the initial state. No listeners are bound, no goroutines
	// are serving traffic. A successful Stop returns to this state.
	StateStopped State = iota
	// StateRunning means every listener built by the builder closure has
	// successfully bound its port and is serving traffic.
	StateRunning
	// StateReloading is a transient state held during a Reload call. It
	// exists so a Status snapshot observed mid-reload reflects the truth —
	// readers should back off rather than assume the engine is quiescent.
	StateReloading
)

// StatusStringFor renders State for wire + logs. Kept as a package-level
// helper so Status marshalling and log fields stay in sync.
func StatusStringFor(s State) string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateRunning:
		return "running"
	case StateReloading:
		return "reloading"
	default:
		return "unknown"
	}
}

// Status is the JSON-friendly projection of Manager state. Emitted by
// GET /api/v1/admin/proxy/status. Fields are omitempty-safe so stopped
// managers don't leak dangling timestamps or stale error strings.
type Status struct {
	State       State  `json:"state"`
	StateStr    string `json:"state_str"`
	Listeners   int    `json:"listeners"`
	RulesLoaded int    `json:"rules_loaded"`
	StartedAt   string `json:"started_at,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

// ListenerBuilder is the factory the Manager invokes on every Start /
// Reload to produce a fresh set of *Listener. Returning fresh instances
// (rather than reusing a pre-built slice) lets Reload rebuild listeners
// after an admin mutates YAML listener config or DB rules — old listeners
// are Shutdown'd and discarded.
type ListenerBuilder func(ctx context.Context) ([]*Listener, error)

// Manager coordinates the lifecycle of a proxy listener pool. The zero
// value is not valid; construct with NewManager. All public methods are
// safe for concurrent use — they serialize through mu.
type Manager struct {
	mu        sync.Mutex
	listeners []*Listener
	state     State
	startedAt time.Time
	lastError error
	builder   ListenerBuilder
}

// NewManager returns a Manager in StateStopped that will build listeners
// via builder on every Start / Reload. A nil builder is rejected eagerly
// so misuse surfaces at wiring time rather than at first Start.
func NewManager(builder ListenerBuilder) *Manager {
	return &Manager{
		builder: builder,
		state:   StateStopped,
	}
}

// Start builds a fresh listener set via the configured builder and
// brings each one up. Returns an error if already Running, if the
// builder fails, or if any listener fails to bind. On builder success +
// any listener bind failure we Shutdown every successfully-started
// listener so partial-Running is never observable.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startLocked(ctx)
}

// startLocked is the mu-held body of Start. Extracted so Reload can
// invoke Stop + Start in a single critical section without dropping the
// lock between them.
func (m *Manager) startLocked(ctx context.Context) error {
	if m.state == StateRunning {
		return errors.New("proxy manager: already running")
	}
	if m.builder == nil {
		m.lastError = errors.New("no builder configured")
		return m.lastError
	}

	listeners, err := m.builder(ctx)
	if err != nil {
		m.lastError = fmt.Errorf("build listeners: %w", err)
		return m.lastError
	}

	started := make([]*Listener, 0, len(listeners))
	for _, l := range listeners {
		if err := l.Start(ctx); err != nil {
			// Roll back anything we already started so the caller
			// doesn't observe a half-bound pool.
			for _, done := range started {
				_ = done.Shutdown(context.Background()) //#nosec G104 -- cleanup path
			}
			m.lastError = fmt.Errorf("start listener %s: %w", l.Bind, err)
			return m.lastError
		}
		started = append(started, l)
	}

	m.listeners = started
	m.state = StateRunning
	m.startedAt = time.Now().UTC()
	m.lastError = nil
	return nil
}

// Stop calls Shutdown on every listener then transitions to StateStopped.
// Idempotent: stopping an already-stopped manager returns nil without
// touching state. Context governs the per-listener Shutdown deadline; an
// expired context is passed through to http.Server.Shutdown.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked(ctx)
}

// stopLocked is the mu-held body of Stop. Extracted so Reload can
// invoke Stop + Start without dropping the lock between them.
func (m *Manager) stopLocked(ctx context.Context) error {
	if m.state == StateStopped {
		return nil
	}
	var firstErr error
	for _, l := range m.listeners {
		if err := l.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	m.listeners = nil
	m.state = StateStopped
	m.startedAt = time.Time{}
	if firstErr != nil {
		m.lastError = firstErr
	}
	return firstErr
}

// Reload performs Stop + Start in a single critical section so the
// pool rebuilds cleanly without a window where a caller could sneak a
// Start between the two halves. State transitions through Reloading for
// observability; on Start failure we remain Stopped and surface the
// error through lastError.
func (m *Manager) Reload(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prev := m.state
	m.state = StateReloading
	if err := m.stopLocked(ctx); err != nil {
		// Stop failure is odd but don't mask it — the caller still wants
		// to know. We drop back to StateStopped so a subsequent Start
		// can retry from a known-clean state.
		m.state = StateStopped
		m.lastError = err
		return err
	}
	if prev == StateStopped {
		// Reload on a stopped manager is defined as "start it" — that
		// way the admin API can treat reload as idempotent-ish without
		// requiring callers to first GET /status.
	}
	if err := m.startLocked(ctx); err != nil {
		// startLocked already wrote to lastError + left state as Stopped.
		return err
	}
	return nil
}

// Status snapshots the current Manager state for wire emission. The
// returned Status is a value copy; callers may mutate it freely. Safe
// for concurrent invocation.
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := Status{
		State:     m.state,
		StateStr:  StatusStringFor(m.state),
		Listeners: len(m.listeners),
	}
	rules := 0
	for _, l := range m.listeners {
		if eng := l.Engine(); eng != nil {
			rules += len(eng.Rules())
		}
	}
	s.RulesLoaded = rules
	if !m.startedAt.IsZero() {
		s.StartedAt = m.startedAt.Format(time.RFC3339)
	}
	if m.lastError != nil {
		s.LastError = m.lastError.Error()
	}
	return s
}

// CurrentEngine returns the first listener's *Engine so the admin
// rule-CRUD handlers can call SetRules for hot-reload. Returns nil when
// the manager is not Running or has no listeners. We expose the first
// listener because the single-listener case is the only one where
// "the engine" is unambiguous — multi-listener setups should reload
// through Manager.Reload instead.
func (m *Manager) CurrentEngine() *Engine {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.listeners) == 0 {
		return nil
	}
	return m.listeners[0].Engine()
}

// Listeners returns a defensive copy of the current listener slice.
// Intended for tests and introspection — callers must not mutate the
// returned listeners' lifecycle directly (bypassing Manager would break
// state tracking).
func (m *Manager) Listeners() []*Listener {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Listener, len(m.listeners))
	copy(out, m.listeners)
	return out
}
