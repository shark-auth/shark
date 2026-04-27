// Package fixtures holds shared bench-time state. Phase A keeps it minimal:
// a Bundle of pre-created users + an OAuth client. Phase B/C extend.
package fixtures

import "sync"

// User is a pre-created auth user (email/password).
type User struct {
	Email    string
	Password string
	UserID   string
	Token    string
}

// OAuthClient is a registered DCR client with a known secret.
type OAuthClient struct {
	ClientID     string
	ClientSecret string
}

// Bundle is the per-run shared fixture state. Scenarios both read and append.
type Bundle struct {
	mu      sync.RWMutex
	Users   []User
	Clients []OAuthClient
	Meta    map[string]any
}

// NewBundle returns an empty Bundle.
func NewBundle() *Bundle {
	return &Bundle{Meta: map[string]any{}}
}

// AddUser appends a user.
func (b *Bundle) AddUser(u User) {
	b.mu.Lock()
	b.Users = append(b.Users, u)
	b.mu.Unlock()
}

// AddClient appends an OAuth client.
func (b *Bundle) AddClient(c OAuthClient) {
	b.mu.Lock()
	b.Clients = append(b.Clients, c)
	b.mu.Unlock()
}

// SnapshotUsers returns a copy of the users slice.
func (b *Bundle) SnapshotUsers() []User {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]User, len(b.Users))
	copy(out, b.Users)
	return out
}

// SnapshotClients returns a copy of the clients slice.
func (b *Bundle) SnapshotClients() []OAuthClient {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]OAuthClient, len(b.Clients))
	copy(out, b.Clients)
	return out
}
