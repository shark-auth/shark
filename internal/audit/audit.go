package audit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/shark-auth/shark/internal/storage"
	"github.com/shark-auth/shark/internal/webhook"
)

// Logger records and queries audit events. It supports asynchronous background
// logging to prevent DB latency from blocking the main request path.
type Logger struct {
	store      storage.Store
	dispatcher *webhook.Dispatcher

	queue  chan *storage.AuditLog
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLogger creates a new audit Logger backed by the given store.
// Callers MUST call Start() to begin background processing.
func NewLogger(store storage.Store) *Logger {
	return &Logger{
		store: store,
		queue: make(chan *storage.AuditLog, 1024),
	}
}

// Start launches the background worker that persists queued audit logs.
func (l *Logger) Start(ctx context.Context) {
	l.ctx, l.cancel = context.WithCancel(ctx)
	l.wg.Add(1)
	go l.worker()
}

// Stop drains the queue and waits for the worker to exit.
func (l *Logger) Stop() {
	if l.cancel != nil {
		l.cancel()
	}
	close(l.queue)
	l.wg.Wait()
}

// SetDispatcher wires a webhook dispatcher for real-time emission.
func (l *Logger) SetDispatcher(d *webhook.Dispatcher) {
	l.dispatcher = d
}

// Log enqueues an audit event for background persistence. It returns immediately.
// If the queue is full, the log entry may be dropped to prevent blocking the caller.
func (l *Logger) Log(ctx context.Context, event *storage.AuditLog) error {
	if event.ID == "" {
		id, _ := gonanoid.New()
		event.ID = "aud_" + id
	}
	if event.CreatedAt == "" {
		event.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if event.Metadata == "" {
		event.Metadata = "{}"
	}
	if event.Status == "" {
		event.Status = "success"
	}
	if event.ActorType == "" {
		event.ActorType = "user"
	}

	// Try to enqueue; don't block if the buffer is full.
	select {
	case l.queue <- event:
	default:
		slog.Warn("audit log queue full; event dropped", "action", event.Action, "actor", event.ActorID)
	}

	return nil
}

// worker processes queued logs and persists them to the store in batches.
func (l *Logger) worker() {
	defer l.wg.Done()
	for {
		var batch []*storage.AuditLog
		// Block for the first event
		event, ok := <-l.queue
		if !ok {
			return
		}
		batch = append(batch, event)

		// Drain the queue for more events immediately available, up to 50
		draining := true
		for draining && len(batch) < 50 {
			select {
			case next, ok := <-l.queue:
				if !ok {
					draining = false
				} else {
					batch = append(batch, next)
				}
			default:
				draining = false
			}
		}

		ctx := context.Background()
		if err := l.store.CreateAuditLogsBatch(ctx, batch); err != nil {
			slog.Error("audit: batch persist failed", "error", err, "count", len(batch))
			// Real-time emission anyway (best effort)
		}

		// Real-time emission to the dashboard (SSE) for each event in the batch
		if l.dispatcher != nil {
			for _, b := range batch {
				_ = l.dispatcher.Emit(ctx, "system.audit_log", b)
			}
		}
	}
}

// Query retrieves audit logs with filters and cursor-based pagination.
func (l *Logger) Query(ctx context.Context, opts storage.AuditLogQuery) ([]*storage.AuditLog, error) {
	return l.store.QueryAuditLogs(ctx, opts)
}

// GetByID retrieves a single audit log entry by ID.
func (l *Logger) GetByID(ctx context.Context, id string) (*storage.AuditLog, error) {
	return l.store.GetAuditLogByID(ctx, id)
}

// DeleteBefore deletes audit logs older than the given time.
func (l *Logger) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return l.store.DeleteAuditLogsBefore(ctx, before)
}

// StartCleanup runs a background goroutine that deletes old logs based on retention.
// A retention of 0 disables cleanup. The goroutine stops when ctx is cancelled.
func (l *Logger) StartCleanup(ctx context.Context, retention time.Duration, interval time.Duration) {
	if retention <= 0 || interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().UTC().Add(-retention)
				deleted, err := l.store.DeleteAuditLogsBefore(ctx, cutoff)
				if err != nil {
					slog.Error("audit cleanup error", "error", err)
				} else if deleted > 0 {
					slog.Info("audit cleanup", "deleted", deleted)
				}
			}
		}
	}()
}
