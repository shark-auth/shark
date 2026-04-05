package audit

import (
	"context"
	"log"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// Logger records and queries audit events.
type Logger struct {
	store storage.Store
}

// NewLogger creates a new audit Logger backed by the given store.
func NewLogger(store storage.Store) *Logger {
	return &Logger{store: store}
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
	return l.store.CreateAuditLog(ctx, event)
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
					log.Printf("audit cleanup error: %v", err)
				} else if deleted > 0 {
					log.Printf("audit cleanup: deleted %d old entries", deleted)
				}
			}
		}
	}()
}
