package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// --- Webhooks ---

func (s *SQLiteStore) CreateWebhook(ctx context.Context, w *Webhook) error {
	if w.Events == "" {
		w.Events = "[]"
	}
	_, err := s.writer.ExecContext(ctx,
		`INSERT INTO webhooks (id, url, secret, events, enabled, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		w.ID, w.URL, w.Secret, w.Events, boolToInt(w.Enabled), w.Description, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) GetWebhookByID(ctx context.Context, id string) (*Webhook, error) {
	return s.scanWebhook(s.reader.QueryRowContext(ctx,
		`SELECT id, url, secret, events, enabled, description, created_at, updated_at
		 FROM webhooks WHERE id = ?`, id))
}

func (s *SQLiteStore) ListWebhooks(ctx context.Context) ([]*Webhook, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT id, url, secret, events, enabled, description, created_at, updated_at
		 FROM webhooks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookRows(rows)
}

// ListEnabledWebhooksByEvent returns enabled webhooks whose `events` JSON
// array contains the given event name. Match uses a LIKE on the JSON text
// — cheap and index-friendly for the expected low-cardinality workload.
// If you register thousands of webhooks, switch to a join table.
func (s *SQLiteStore) ListEnabledWebhooksByEvent(ctx context.Context, event string) ([]*Webhook, error) {
	// Match `"event.name"` inside the JSON array to avoid prefix false-positives
	// (e.g. searching "user.created" shouldn't match "user.created_v2").
	needle := fmt.Sprintf(`%%"%s"%%`, event)
	rows, err := s.reader.QueryContext(ctx,
		`SELECT id, url, secret, events, enabled, description, created_at, updated_at
		 FROM webhooks WHERE enabled = 1 AND events LIKE ?
		 ORDER BY created_at ASC`, needle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWebhookRows(rows)
}

func (s *SQLiteStore) UpdateWebhook(ctx context.Context, w *Webhook) error {
	_, err := s.writer.ExecContext(ctx,
		`UPDATE webhooks SET url = ?, secret = ?, events = ?, enabled = ?, description = ?, updated_at = ?
		 WHERE id = ?`,
		w.URL, w.Secret, w.Events, boolToInt(w.Enabled), w.Description, w.UpdatedAt, w.ID,
	)
	return err
}

func (s *SQLiteStore) DeleteWebhook(ctx context.Context, id string) error {
	_, err := s.writer.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, id)
	return err
}

func (s *SQLiteStore) scanWebhook(row *sql.Row) (*Webhook, error) {
	var w Webhook
	var enabled int
	if err := row.Scan(&w.ID, &w.URL, &w.Secret, &w.Events, &enabled,
		&w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	w.Enabled = enabled != 0
	return &w, nil
}

func scanWebhookRows(rows *sql.Rows) ([]*Webhook, error) {
	var out []*Webhook
	for rows.Next() {
		var w Webhook
		var enabled int
		if err := rows.Scan(&w.ID, &w.URL, &w.Secret, &w.Events, &enabled,
			&w.Description, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		w.Enabled = enabled != 0
		out = append(out, &w)
	}
	return out, rows.Err()
}

// --- Webhook Deliveries ---

func (s *SQLiteStore) CreateWebhookDelivery(ctx context.Context, d *WebhookDelivery) error {
	_, err := s.writer.ExecContext(ctx,
		`INSERT INTO webhook_deliveries
		 (id, webhook_id, event, payload, signature_header, status, status_code,
		  response_body, error, attempt, next_retry_at, delivered_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.WebhookID, d.Event, d.Payload, d.SignatureHeader,
		d.Status, d.StatusCode, d.ResponseBody, d.Error, d.Attempt,
		d.NextRetryAt, d.DeliveredAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (s *SQLiteStore) CreateWebhookDeliveriesBatch(ctx context.Context, ds []*WebhookDelivery) error {
	if len(ds) == 0 {
		return nil
	}
	tx, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO webhook_deliveries
		 (id, webhook_id, event, payload, signature_header, status, status_code,
		  response_body, error, attempt, next_retry_at, delivered_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range ds {
		if _, err := stmt.ExecContext(ctx,
			d.ID, d.WebhookID, d.Event, d.Payload, d.SignatureHeader,
			d.Status, d.StatusCode, d.ResponseBody, d.Error, d.Attempt,
			d.NextRetryAt, d.DeliveredAt, d.CreatedAt, d.UpdatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) UpdateWebhookDelivery(ctx context.Context, d *WebhookDelivery) error {
	_, err := s.writer.ExecContext(ctx,
		`UPDATE webhook_deliveries SET
		   status = ?, status_code = ?, response_body = ?, error = ?,
		   attempt = ?, next_retry_at = ?, delivered_at = ?, updated_at = ?
		 WHERE id = ?`,
		d.Status, d.StatusCode, d.ResponseBody, d.Error,
		d.Attempt, d.NextRetryAt, d.DeliveredAt, d.UpdatedAt, d.ID,
	)
	return err
}

func (s *SQLiteStore) GetWebhookDeliveryByID(ctx context.Context, id string) (*WebhookDelivery, error) {
	return s.scanDelivery(s.reader.QueryRowContext(ctx,
		`SELECT id, webhook_id, event, payload, signature_header, status, status_code,
		        response_body, error, attempt, next_retry_at, delivered_at, created_at, updated_at
		 FROM webhook_deliveries WHERE id = ?`, id))
}

// ListWebhookDeliveriesByWebhookID returns deliveries ordered by
// (created_at DESC, id DESC) with keyset cursor pagination.
func (s *SQLiteStore) ListWebhookDeliveriesByWebhookID(ctx context.Context, webhookID string, limit int, cursor string) ([]*WebhookDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var (
		where []string
		args  []interface{}
	)
	where = append(where, `webhook_id = ?`)
	args = append(args, webhookID)

	if cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cursor")
		}
		where = append(where, `(created_at < ? OR (created_at = ? AND id < ?))`)
		args = append(args, parts[0], parts[0], parts[1])
	}

	//#nosec G202 -- WHERE clauses are compile-time constant predicates; user values pass through ? placeholders in args
	q := `SELECT id, webhook_id, event, payload, signature_header, status, status_code,
	             response_body, error, attempt, next_retry_at, delivered_at, created_at, updated_at
	      FROM webhook_deliveries
	      WHERE ` + strings.Join(where, " AND ") + `
	      ORDER BY created_at DESC, id DESC
	      LIMIT ?`
	args = append(args, limit)

	rows, err := s.reader.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeliveryRows(rows)
}

// ListPendingWebhookDeliveries returns deliveries that are due for a retry
// attempt at `now` or earlier. Used by the retry scheduler.
func (s *SQLiteStore) ListPendingWebhookDeliveries(ctx context.Context, now time.Time, limit int) ([]*WebhookDelivery, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.reader.QueryContext(ctx,
		`SELECT id, webhook_id, event, payload, signature_header, status, status_code,
		        response_body, error, attempt, next_retry_at, delivered_at, created_at, updated_at
		 FROM webhook_deliveries
		 WHERE status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?
		 ORDER BY next_retry_at ASC LIMIT ?`,
		WebhookStatusRetrying, now.UTC().Format(time.RFC3339), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeliveryRows(rows)
}

func (s *SQLiteStore) DeleteWebhookDeliveriesBefore(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.writer.ExecContext(ctx,
		`DELETE FROM webhook_deliveries WHERE created_at < ?`,
		before.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *SQLiteStore) scanDelivery(row *sql.Row) (*WebhookDelivery, error) {
	var d WebhookDelivery
	if err := row.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.SignatureHeader,
		&d.Status, &d.StatusCode, &d.ResponseBody, &d.Error,
		&d.Attempt, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func scanDeliveryRows(rows *sql.Rows) ([]*WebhookDelivery, error) {
	var out []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.SignatureHeader,
			&d.Status, &d.StatusCode, &d.ResponseBody, &d.Error,
			&d.Attempt, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}
