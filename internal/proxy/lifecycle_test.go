package proxy

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// freePort asks the kernel for an unused TCP port by binding :0, reading
// the assigned port, then closing the listener. There's a small TOCTOU
// window between close + rebind but it's the standard pattern for test
// helpers — the alternative (hardcoded ports) is flakier under parallel
// go test runs.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort listen: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("freePort close: %v", err)
	}
	return addr
}

// fakeBuilder returns a builder closure that produces fresh Listener
// objects every call. We need "fresh" because Listener.Shutdown flips
// started=true permanently — a restarted Listener would fail Start.
// counter tracks how many times the builder was invoked so assertions
// can verify Reload actually rebuilt. Each invocation asks the kernel
// for a brand-new free port; reusing a single port across start/stop
// cycles trips Windows' TIME_WAIT refusal of an immediate rebind.
func fakeBuilder(t *testing.T, counter *int) ListenerBuilder {
	t.Helper()
	return func(ctx context.Context) ([]*Listener, error) {
		*counter++
		l, err := NewListener(ListenerParams{
			Bind:     freePort(t),
			Upstream: "http://127.0.0.1:1",
			Timeout:  100 * time.Millisecond,
		})
		if err != nil {
			return nil, err
		}
		return []*Listener{l}, nil
	}
}

func TestManagerStartStopRoundtrip(t *testing.T) {
	var calls int
	m := NewManager(fakeBuilder(t, &calls))

	// Fresh manager observes StateStopped.
	if got := m.Status().State; got != StateStopped {
		t.Fatalf("initial state = %v, want Stopped", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s := m.Status(); s.State != StateRunning || s.Listeners != 1 {
		t.Fatalf("after Start status = %+v, want Running/1", s)
	}
	if s := m.Status().StateStr; s != "running" {
		t.Fatalf("state_str = %q, want running", s)
	}
	if m.Status().StartedAt == "" {
		t.Fatalf("StartedAt empty after Start")
	}

	// Double-Start surfaces a clear error and leaves state Running.
	err := m.Start(ctx)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("double Start err = %v, want already running", err)
	}
	if m.Status().State != StateRunning {
		t.Fatalf("state after double Start = %v, want Running", m.Status().State)
	}

	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if s := m.Status(); s.State != StateStopped || s.Listeners != 0 {
		t.Fatalf("after Stop status = %+v, want Stopped/0", s)
	}
	// Idempotent Stop.
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("second Stop: %v", err)
	}

	// Start again after Stop — builder invoked twice.
	if err := m.Start(ctx); err != nil {
		t.Fatalf("restart Start: %v", err)
	}
	if calls != 2 {
		t.Fatalf("builder calls = %d, want 2", calls)
	}
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("final Stop: %v", err)
	}
}

func TestManagerReloadPreservesRunning(t *testing.T) {
	var calls int
	m := NewManager(fakeBuilder(t, &calls))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	// Reload rebuilds listeners. After it returns state must be Running
	// again + the builder must have been called a second time.
	if err := m.Reload(ctx); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if s := m.Status(); s.State != StateRunning || s.Listeners != 1 {
		t.Fatalf("after Reload status = %+v, want Running/1", s)
	}
	if calls != 2 {
		t.Fatalf("builder calls = %d, want 2 (start + reload)", calls)
	}
}

func TestManagerReloadFromStopped(t *testing.T) {
	var calls int
	m := NewManager(fakeBuilder(t, &calls))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Reload on a never-started manager is defined as "start it".
	if err := m.Reload(ctx); err != nil {
		t.Fatalf("Reload from Stopped: %v", err)
	}
	if m.Status().State != StateRunning {
		t.Fatalf("state after Reload-from-Stopped = %v, want Running", m.Status().State)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })
}

func TestManagerBuilderFailurePropagates(t *testing.T) {
	boom := errors.New("builder boom")
	m := NewManager(func(ctx context.Context) ([]*Listener, error) {
		return nil, boom
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := m.Start(ctx)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("Start err = %v, want wraps boom", err)
	}
	if s := m.Status(); s.State != StateStopped || s.LastError == "" {
		t.Fatalf("after failed Start status = %+v, want Stopped + LastError", s)
	}
}

func TestManagerNilBuilderRejected(t *testing.T) {
	m := NewManager(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := m.Start(ctx)
	if err == nil || !strings.Contains(err.Error(), "no builder") {
		t.Fatalf("Start err = %v, want 'no builder'", err)
	}
}

func TestManagerStatusCopiedNotShared(t *testing.T) {
	var calls int
	m := NewManager(fakeBuilder(t, &calls))
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	s1 := m.Status()
	s1.Listeners = 999 // mutate copy
	s2 := m.Status()
	if s2.Listeners == 999 {
		t.Fatalf("Status mutation leaked: s2.Listeners = %d", s2.Listeners)
	}
}

func TestManagerCurrentEngine(t *testing.T) {
	var calls int
	m := NewManager(fakeBuilder(t, &calls))

	// Stopped manager → nil engine.
	if eng := m.CurrentEngine(); eng != nil {
		t.Fatalf("CurrentEngine on stopped = %v, want nil", eng)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop(context.Background()) })

	if eng := m.CurrentEngine(); eng == nil {
		t.Fatalf("CurrentEngine running = nil, want non-nil")
	}
}
