package email

import (
	"fmt"
	"log/slog"
)

// SharkEmailSender is the shark.email relay (RM.md feature, #56). Phase 2
// ships the wiring only — the relay service isn't up yet, so Send returns an
// explanatory error if any request actually reaches it. Selecting
// provider="shark" before GA is treated as a config mistake, not silent
// failure, so devs notice and switch to resend or smtp.
type SharkEmailSender struct {
	apiKey   string
	from     string
	fromName string
}

// NewSharkEmailSender builds a sender that will once the relay is live.
// Stored so the config path works end-to-end today.
func NewSharkEmailSender(apiKey, from, fromName string) *SharkEmailSender {
	return &SharkEmailSender{apiKey: apiKey, from: from, fromName: fromName}
}

// Send fails loudly instead of silently dropping: users who select
// provider=shark before the relay ships should get a clear error instead of
// thinking delivery worked. This matches README policy: we don't promise what
// isn't live yet.
func (s *SharkEmailSender) Send(msg *Message) error {
	slog.Warn("shark.email relay not yet available",
		"to", msg.To, "subject", msg.Subject,
		"docs_url", "https://sharkauth.com/docs/email#shark-email")
	return fmt.Errorf("shark.email relay not yet available — set email.provider to resend or smtp (docs: https://sharkauth.com/docs/email#shark-email)")
}
