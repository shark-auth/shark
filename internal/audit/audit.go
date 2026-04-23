package audit

import (
	"context"
	"log/slog"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
	"github.com/sharkauth/sharkauth/internal/webhook"
)

// Logger records and queries audit events.
type Logger struct {
	store      storage.Store
	dispatcher *webhook.Dispatcher
}

// NewLogger creates a new audit Logger backed by the given store.
func NewLogger(store storage.Store) *Logger {
	return &Logger{store: store}
}

// SetDispatcher wires a webhook dispatcher for real-time emission.
func (l *Logger) SetDispatcher(d *webhook.Dispatcher) {
	l.dispatcher = d
}

// Log records an audit event. It assigns an ID and timestamp if not already set.
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
	if err := l.store.CreateAuditLog(ctx, event); err != nil {
		return err
	}

	// Real-time emission
	if l.dispatcher != nil {
		_ = l.dispatcher.Emit(ctx, "system.audit_log", event)
	}

	return nil
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
