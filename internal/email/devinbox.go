package email

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// DevInboxSender captures outbound emails into the `dev_emails` table so the
// dashboard can render them in a developer inbox. Magic links + password
// reset URLs are also logged to stdout so a CLI-only developer can still
// follow the flow without opening the UI.
type DevInboxSender struct {
	store storage.Store
}

// NewDevInboxSender builds a sender that persists into the provided store.
func NewDevInboxSender(store storage.Store) *DevInboxSender {
	return &DevInboxSender{store: store}
}

// Send implements Sender. Persists the message and logs a short notice.
func (d *DevInboxSender) Send(msg *Message) error {
	id, _ := gonanoid.New()
	e := &storage.DevEmail{
		ID:        "de_" + id,
		To:        msg.To,
		Subject:   msg.Subject,
		HTML:      msg.HTML,
		Text:      msg.Text,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := d.store.CreateDevEmail(context.Background(), e); err != nil {
		return fmt.Errorf("dev inbox persist: %w", err)
	}

	link := extractLink(msg.HTML + " " + msg.Text)
	if link != "" {
		slog.Info("dev inbox captured email", "to", msg.To, "subject", msg.Subject, "link", link)
	} else {
		slog.Info("dev inbox captured email", "to", msg.To, "subject", msg.Subject)
	}
	return nil
}

// extractLink scans body for the first http(s) URL so dev operators can copy
// it from logs without opening the dashboard. Not exhaustive; a single hit is enough.
func extractLink(body string) string {
	for _, tok := range strings.Fields(body) {
		tok = strings.Trim(tok, `"'<>`)
		if strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://") {
			return tok
		}
	}
	return ""
}
