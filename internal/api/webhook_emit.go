package api

import (
	"context"
	"log/slog"

	"github.com/sharkauth/sharkauth/internal/storage"
)

// emit is a tiny nil-safe wrapper around WebhookDispatcher.Emit so emission
// sites don't repeat the nil check. Errors are logged but never fail the
// caller's request — webhook delivery is always best-effort async.
func (s *Server) emit(ctx context.Context, event string, payload any) {
	if s.WebhookDispatcher == nil {
		return
	}
	if err := s.WebhookDispatcher.Emit(ctx, event, payload); err != nil {
		slog.Error("webhook emit", "event", event, "error", err)
	}
}

// publicUser strips sensitive fields so webhook payloads are safe to send to
// third parties. No password hash, no MFA secret.
type publicUser struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	Name          *string `json:"name,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func userPublic(u *storage.User) publicUser {
	return publicUser{
		ID: u.ID, Email: u.Email, EmailVerified: u.EmailVerified,
		Name: u.Name, CreatedAt: u.CreatedAt,
	}
}
