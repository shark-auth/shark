package testutil

import (
	"sync"

	"github.com/shark-auth/shark/internal/email"
)

// MemoryEmailSender captures sent emails in memory for testing.
// It implements the email.Sender interface.
type MemoryEmailSender struct {
	mu       sync.Mutex
	Messages []*email.Message
}

// NewMemoryEmailSender creates a new MemoryEmailSender.
func NewMemoryEmailSender() *MemoryEmailSender {
	return &MemoryEmailSender{}
}

// Send captures the email message instead of actually sending it.
func (s *MemoryEmailSender) Send(msg *email.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
	return nil
}

// LastMessage returns the most recently sent message, or nil if no messages were sent.
func (s *MemoryEmailSender) LastMessage() *email.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Messages) == 0 {
		return nil
	}
	return s.Messages[len(s.Messages)-1]
}

// MessageCount returns the number of messages sent.
func (s *MemoryEmailSender) MessageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Messages)
}

// Reset clears all captured messages.
func (s *MemoryEmailSender) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = nil
}

// MessagesTo returns all messages sent to the given email address.
func (s *MemoryEmailSender) MessagesTo(to string) []*email.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*email.Message
	for _, msg := range s.Messages {
		if msg.To == to {
			result = append(result, msg)
		}
	}
	return result
}
