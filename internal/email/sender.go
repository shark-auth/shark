package email

// Message represents an email to be sent.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender is the interface for sending emails.
// Implementations include SMTP sender (production) and MemoryEmailSender (testing).
type Sender interface {
	Send(msg *Message) error
}
