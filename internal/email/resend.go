package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sharkauth/sharkauth/internal/config"
)

const resendAPIURL = "https://api.resend.com/emails"

// ResendSender sends emails via Resend's HTTP API.
type ResendSender struct {
	apiKey   string
	from     string
	fromName string
	client   *http.Client
}

// NewResendSender creates a new ResendSender from SMTP config.
// Reuses the existing SMTPConfig: Password is the API key, From/FromName are the sender.
func NewResendSender(cfg config.SMTPConfig) *ResendSender {
	return &ResendSender{
		apiKey:   cfg.Password,
		from:     cfg.From,
		fromName: cfg.FromName,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type resendRequest struct {
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// Send sends an email via Resend HTTP API.
func (r *ResendSender) Send(msg *Message) error {
	from := r.from
	if r.fromName != "" {
		from = fmt.Sprintf("%s <%s>", r.fromName, r.from)
	}

	payload := resendRequest{
		From:    from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling resend request: %w", err)
	}

	req, err := http.NewRequest("POST", resendAPIURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending resend request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
