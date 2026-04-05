package testutil

import (
	"context"
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
)

// CreateUser creates a test user with the given email and password hash.
// Returns the created user. The password hash should be pre-computed;
// use a simple hash for testing, not production argon2id.
func CreateUser(t *testing.T, store storage.Store, email string, passwordHash *string) *storage.User {
	t.Helper()

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	hashType := ""
	if passwordHash != nil {
		hashType = "argon2id"
	}

	u := &storage.User{
		ID:        "usr_" + id,
		Email:     email,
		HashType:  hashType,
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if passwordHash != nil {
		u.PasswordHash = passwordHash
		u.HashType = "argon2id"
	}

	if err := store.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return u
}

// CreateUserWithRole creates a test user and assigns the given role to them.
func CreateUserWithRole(t *testing.T, store storage.Store, email string, passwordHash *string, roleName string) *storage.User {
	t.Helper()

	u := CreateUser(t, store, email, passwordHash)

	// Find or create role
	role, err := store.GetRoleByName(context.Background(), roleName)
	if err != nil {
		role = CreateRole(t, store, roleName)
	}

	if err := store.AssignRoleToUser(context.Background(), u.ID, role.ID); err != nil {
		t.Fatalf("assigning role to user: %v", err)
	}
	return u
}

// CreateRole creates a test role with the given name.
func CreateRole(t *testing.T, store storage.Store, name string) *storage.Role {
	t.Helper()

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)

	r := &storage.Role{
		ID:          "role_" + id,
		Name:        name,
		Description: "Test role: " + name,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.CreateRole(context.Background(), r); err != nil {
		t.Fatalf("creating test role: %v", err)
	}
	return r
}

// CreatePermission creates a test permission with the given action and resource.
func CreatePermission(t *testing.T, store storage.Store, action, resource string) *storage.Permission {
	t.Helper()

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)

	p := &storage.Permission{
		ID:        "perm_" + id,
		Action:    action,
		Resource:  resource,
		CreatedAt: now,
	}

	if err := store.CreatePermission(context.Background(), p); err != nil {
		t.Fatalf("creating test permission: %v", err)
	}
	return p
}

// CreateSession creates a test session for the given user.
func CreateSession(t *testing.T, store storage.Store, userID string) *storage.Session {
	t.Helper()

	id, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)
	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)

	sess := &storage.Session{
		ID:         "sess_" + id,
		UserID:     userID,
		IP:         "127.0.0.1",
		UserAgent:  "TestClient/1.0",
		AuthMethod: "password",
		ExpiresAt:  expiresAt,
		CreatedAt:  now,
	}

	if err := store.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("creating test session: %v", err)
	}
	return sess
}

// CreateAPIKey creates a test API key. Returns the APIKey record (hash stored, not the raw key).
func CreateAPIKey(t *testing.T, store storage.Store, name string) *storage.APIKey {
	t.Helper()

	id, _ := gonanoid.New()
	keyHash, _ := gonanoid.New()
	now := time.Now().UTC().Format(time.RFC3339)

	k := &storage.APIKey{
		ID:        "key_" + id,
		Name:      name,
		KeyHash:   keyHash,
		KeyPrefix: "sk_live_t",
		Scopes:    `["users:read","users:write"]`,
		RateLimit: 1000,
		CreatedAt: now,
	}

	if err := store.CreateAPIKey(context.Background(), k); err != nil {
		t.Fatalf("creating test API key: %v", err)
	}
	return k
}
