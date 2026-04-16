// Package webhook handles outbound event delivery: HMAC-signed payloads,
// exponential retry, durable delivery log.
//
// Design choices:
//   - Durable-first: every attempt writes a webhook_deliveries row before the
//     HTTP call leaves the process. Process death mid-flight never loses
//     events — the next scheduler tick finds the row and retries.
//   - Async fan-out: Emit() returns immediately after persisting the pending
//     delivery row; the actual HTTP call runs on a bounded worker pool.
//   - Retry policy: 5 attempts, backoff [1m, 5m, 30m, 2h, 12h]. Past that, mark
//     failed. Balances "survive a brief outage" vs "don't retry forever".
//   - Signature: X-Shark-Signature: t=<unix_ts>,v1=<hex(hmac_sha256(secret, "t.body"))>
//     matches the Stripe shape so SDKs and docs examples translate cleanly.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// BackoffSchedule defines the delay after attempt N (1-indexed).
// Attempt 1 (first delivery) uses delay 0. Attempt 2 waits BackoffSchedule[0].
// Total window: 1m + 5m + 30m + 2h + 12h ≈ 14h40m.
var BackoffSchedule = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	12 * time.Hour,
}

// MaxAttempts is the total number of delivery attempts before marking failed.
// Initial attempt plus one per BackoffSchedule entry.
var MaxAttempts = len(BackoffSchedule) + 1

// deliveryTimeout is the HTTP timeout per attempt. Long enough for a slow
// downstream to respond, short enough that a worker doesn't wedge.
const deliveryTimeout = 10 * time.Second

// Dispatcher fans out events to enabled webhooks and manages retries.
// Safe for concurrent use from any number of emission sites.
type Dispatcher struct {
	store    storage.Store
	workers  int
	jobs     chan string // delivery IDs to run next
	http     *http.Client
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// Option configures a Dispatcher.
type Option func(*Dispatcher)

// WithWorkers sets the number of delivery workers (default 4).
func WithWorkers(n int) Option {
	return func(d *Dispatcher) {
		if n > 0 {
			d.workers = n
		}
	}
}

// WithHTTPClient overrides the HTTP client. Tests inject a test server client.
func WithHTTPClient(c *http.Client) Option {
	return func(d *Dispatcher) {
		d.http = c
	}
}

// New returns a Dispatcher. Call Start to begin processing, Stop to drain.
func New(store storage.Store, opts ...Option) *Dispatcher {
	d := &Dispatcher{
		store:   store,
		workers: 4,
		jobs:    make(chan string, 256),
		http:    &http.Client{Timeout: deliveryTimeout},
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Start launches worker goroutines + retry scheduler. ctx cancellation triggers
// Stop; callers can also call Stop directly.
func (d *Dispatcher) Start(ctx context.Context) {
	d.ctx, d.cancel = context.WithCancel(ctx)
	for i := 0; i < d.workers; i++ {
		d.wg.Add(1)
		go d.worker()
	}
	d.wg.Add(1)
	go d.retryLoop()
}

// Stop waits for in-flight deliveries to finish (bounded by deliveryTimeout).
func (d *Dispatcher) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	close(d.jobs)
	d.wg.Wait()
}

// Emit records a pending delivery per matching webhook and enqueues immediate
// delivery. Non-blocking — all I/O happens on workers.
func (d *Dispatcher) Emit(ctx context.Context, event string, payload any) error {
	hooks, err := d.store.ListEnabledWebhooksByEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("list webhooks: %w", err)
	}
	if len(hooks) == 0 {
		return nil
	}

	body, err := buildPayload(event, payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, w := range hooks {
		id, _ := gonanoid.New()
		del := &storage.WebhookDelivery{
			ID: "whd_" + id, WebhookID: w.ID, Event: event,
			Payload: string(body), Status: storage.WebhookStatusPending,
			Attempt: 0, CreatedAt: now, UpdatedAt: now,
		}
		if err := d.store.CreateWebhookDelivery(ctx, del); err != nil {
			slog.Error("webhook: persist delivery", "error", err, "webhook_id", w.ID)
			continue
		}
		d.schedule(del.ID)
	}
	return nil
}

// Redeliver is the /webhooks/{id}/test handler's entry point. Returns the
// delivery ID so the caller can return it in the response.
func (d *Dispatcher) Redeliver(ctx context.Context, webhookID, event string, payload any) (string, error) {
	w, err := d.store.GetWebhookByID(ctx, webhookID)
	if err != nil {
		return "", err
	}
	body, err := buildPayload(event, payload)
	if err != nil {
		return "", err
	}
	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	del := &storage.WebhookDelivery{
		ID: "whd_" + id, WebhookID: w.ID, Event: event,
		Payload: string(body), Status: storage.WebhookStatusPending,
		Attempt: 0, CreatedAt: now, UpdatedAt: now,
	}
	if err := d.store.CreateWebhookDelivery(ctx, del); err != nil {
		return "", err
	}
	d.schedule(del.ID)
	return del.ID, nil
}

// schedule non-blocking enqueue; drops the job if the pool is saturated so the
// caller never blocks. Dropped jobs are picked up by the retry scheduler's
// next tick since they remain in status=pending/retrying.
func (d *Dispatcher) schedule(deliveryID string) {
	select {
	case d.jobs <- deliveryID:
	default:
		slog.Warn("webhook: job queue full, retry loop will pick up", "delivery_id", deliveryID)
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	for id := range d.jobs {
		if id == "" {
			continue
		}
		d.attempt(id)
	}
}

// retryLoop ticks every 30s and re-enqueues deliveries whose next_retry_at has passed.
// Also picks up stranded pending rows (process restart) in case a worker
// crashed between persist and HTTP call.
func (d *Dispatcher) retryLoop() {
	defer d.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			due, err := d.store.ListPendingWebhookDeliveries(d.ctx, time.Now().UTC(), 100)
			if err != nil {
				slog.Error("webhook: pending list", "error", err)
				continue
			}
			for _, p := range due {
				d.schedule(p.ID)
			}
		}
	}
}

// attempt performs one delivery attempt and records the result. On failure,
// either schedules the next retry (if attempts remain) or marks failed.
func (d *Dispatcher) attempt(deliveryID string) {
	ctx, cancel := context.WithTimeout(d.ctx, deliveryTimeout+2*time.Second)
	defer cancel()

	del, err := d.store.GetWebhookDeliveryByID(ctx, deliveryID)
	if err != nil {
		slog.Error("webhook: load delivery", "error", err, "id", deliveryID)
		return
	}
	if del.Status == storage.WebhookStatusDelivered || del.Status == storage.WebhookStatusFailed {
		return // terminal; ignore duplicate schedule
	}

	hook, err := d.store.GetWebhookByID(ctx, del.WebhookID)
	if err != nil {
		d.markFailed(ctx, del, fmt.Sprintf("webhook not found: %v", err))
		return
	}
	if !hook.Enabled {
		d.markFailed(ctx, del, "webhook disabled")
		return
	}

	del.Attempt++
	ts := time.Now().UTC().Unix()
	sigHeader := signPayload(hook.Secret, ts, del.Payload)
	del.SignatureHeader = sigHeader
	del.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	req, err := http.NewRequestWithContext(ctx, "POST", hook.URL, bytes.NewReader([]byte(del.Payload)))
	if err != nil {
		d.recordFailure(ctx, del, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SharkAuth-Webhook/1.0")
	req.Header.Set("X-Shark-Event", del.Event)
	req.Header.Set("X-Shark-Delivery", del.ID)
	req.Header.Set("X-Shark-Signature", sigHeader)

	resp, err := d.http.Do(req)
	if err != nil {
		d.recordFailure(ctx, del, err.Error())
		return
	}
	defer resp.Body.Close()

	// Cap response capture to keep one giant body from blowing up the delivery log.
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	statusCode := resp.StatusCode
	del.StatusCode = &statusCode
	del.ResponseBody = string(respBody)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		now := time.Now().UTC().Format(time.RFC3339)
		del.Status = storage.WebhookStatusDelivered
		del.DeliveredAt = &now
		del.NextRetryAt = nil
		del.Error = ""
		del.UpdatedAt = now
		if err := d.store.UpdateWebhookDelivery(ctx, del); err != nil {
			slog.Error("webhook: persist delivered", "error", err)
		}
		return
	}
	d.recordFailure(ctx, del, fmt.Sprintf("HTTP %d", resp.StatusCode))
}

// recordFailure records an attempt error and either schedules the next retry
// or marks the delivery as failed if the attempt budget is exhausted.
func (d *Dispatcher) recordFailure(ctx context.Context, del *storage.WebhookDelivery, errMsg string) {
	del.Error = truncate(errMsg, 512)
	now := time.Now().UTC()
	del.UpdatedAt = now.Format(time.RFC3339)

	// del.Attempt is already incremented. BackoffSchedule index = Attempt-1.
	if del.Attempt >= MaxAttempts {
		del.Status = storage.WebhookStatusFailed
		del.NextRetryAt = nil
		if err := d.store.UpdateWebhookDelivery(ctx, del); err != nil {
			slog.Error("webhook: persist failed", "error", err)
		}
		return
	}
	idx := del.Attempt - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(BackoffSchedule) {
		idx = len(BackoffSchedule) - 1
	}
	next := now.Add(BackoffSchedule[idx]).Format(time.RFC3339)
	del.Status = storage.WebhookStatusRetrying
	del.NextRetryAt = &next
	if err := d.store.UpdateWebhookDelivery(ctx, del); err != nil {
		slog.Error("webhook: persist retrying", "error", err)
	}
}

func (d *Dispatcher) markFailed(ctx context.Context, del *storage.WebhookDelivery, reason string) {
	del.Status = storage.WebhookStatusFailed
	del.Error = truncate(reason, 512)
	del.NextRetryAt = nil
	del.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := d.store.UpdateWebhookDelivery(ctx, del); err != nil {
		slog.Error("webhook: persist failed", "error", err)
	}
}

// StartRetention launches a background cleaner that prunes deliveries older
// than `retention`. `retention == 0` disables pruning (forever).
func (d *Dispatcher) StartRetention(ctx context.Context, retention, interval time.Duration) {
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
				if n, err := d.store.DeleteWebhookDeliveriesBefore(ctx, cutoff); err != nil {
					slog.Error("webhook: retention cleanup", "error", err)
				} else if n > 0 {
					slog.Info("webhook: retention cleanup", "deleted", n)
				}
			}
		}
	}()
}

// --- Signing + payload helpers ---

// signPayload returns the X-Shark-Signature header value:
//
//	t=<ts>,v1=<hex(hmac_sha256(secret, fmt.Sprintf("%d.%s", ts, body)))>
//
// The receiver recomputes HMAC with their secret + the same ts + raw body to
// verify. Matches Stripe's shape so integration docs translate.
func signPayload(secret string, ts int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", ts, body)
	return fmt.Sprintf("t=%d,v1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

// buildPayload wraps arbitrary event data in a standard envelope.
func buildPayload(event string, data any) ([]byte, error) {
	env := map[string]any{
		"event":      event,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"data":       data,
	}
	return json.Marshal(env)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Ensure unused import (strings) is referenced — left for future signature
// parsing helper. Removed if not needed.
var _ = strings.Split
