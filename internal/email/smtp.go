package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"

	"github.com/shark-auth/shark/internal/config"
)

// plainAuthNoTLSCheck is like smtp.PlainAuth but skips the TLS check.
// Go's smtp.PlainAuth refuses to send credentials unless server.TLS is true,
// but with implicit TLS (port 465 via tls.Dial), the SMTP client doesn't know
// TLS is active â€” it only detects STARTTLS-negotiated TLS. This causes auth
// to fail with "unencrypted connection" even though the transport IS encrypted.
type plainAuthNoTLSCheck struct {
	identity, username, password, host string
}

func (a *plainAuthNoTLSCheck) Start(server *smtp.ServerInfo) (string, []byte, error) {
	resp := []byte("\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuthNoTLSCheck) Next(fromServer []byte, more bool) ([]byte, error) {
	return nil, nil
}

// SMTPSender sends emails via SMTP with STARTTLS.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
	fromName string
}

// NewSMTPSender creates a new SMTPSender from config.
func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{
		host:     cfg.Host,
		port:     cfg.Port,
		username: cfg.Username,
		password: cfg.Password,
		from:     cfg.From,
		fromName: cfg.FromName,
	}
}

// Send sends an email message via SMTP.
func (s *SMTPSender) Send(msg *Message) error {
	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	// Build the email headers and body
	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", s.fromName, s.from)
	}

	headers := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\"\r\n"+
		"\r\n", fromHeader, msg.To, msg.Subject)

	body := headers + msg.HTML

	// Connect to SMTP server
	tlsConfig := &tls.Config{ServerName: s.host}
	var conn net.Conn
	if s.port == 465 {
		// Port 465 uses implicit TLS (SMTPS)
		var err error
		conn, err = tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("connecting to SMTP server (TLS): %w", err)
		}
	} else {
		var err error
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("connecting to SMTP server: %w", err)
		}
	}

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		conn.Close() //#nosec G104 -- cleanup after client build failure; primary error returned
		return fmt.Errorf("creating SMTP client: %w", err)
	}
	defer client.Close()

	// Try STARTTLS if available (for port 587)
	if s.port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS: %w", err)
			}
		}
	}

	// Authenticate if credentials are provided
	if s.username != "" && s.password != "" {
		var auth smtp.Auth
		if s.port == 465 {
			// Port 465 uses implicit TLS â€” Go's PlainAuth refuses to send
			// credentials because it doesn't detect TLS on pre-established connections.
			auth = &plainAuthNoTLSCheck{username: s.username, password: s.password, host: s.host}
		} else {
			auth = smtp.PlainAuth("", s.username, s.password, s.host)
		}
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("SMTP RCPT TO: %w", err)
	}

	// Write the message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("writing email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("closing email body: %w", err)
	}

	return client.Quit()
}
