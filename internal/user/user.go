package user

import (
	"context"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/sharkauth/sharkauth/internal/storage"
)

const idPrefix = "usr_"

// NewID generates a new user ID with the usr_ prefix.
func NewID() string {
	id, _ := gonanoid.New()
	return idPrefix + id
}

// Service provides user operations backed by the Store.
type Service struct {
	store storage.Store
}

// NewService creates a new user service.
func NewService(store storage.Store) *Service {
	return &Service{store: store}
}

// Create creates a new user with the given email and optional password hash.
func (s *Service) Create(ctx context.Context, email string, passwordHash *string, name *string) (*storage.User, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	hashType := "argon2id"
	if passwordHash == nil {
		hashType = ""
	}

	u := &storage.User{
		ID:        NewID(),
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
	if name != nil {
		u.Name = name
	}

	if err := s.store.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// GetByID returns a user by their ID.
func (s *Service) GetByID(ctx context.Context, id string) (*storage.User, error) {
	return s.store.GetUserByID(ctx, id)
}

// GetByEmail returns a user by their email address.
func (s *Service) GetByEmail(ctx context.Context, email string) (*storage.User, error) {
	return s.store.GetUserByEmail(ctx, email)
}

// List returns a paginated list of users.
func (s *Service) List(ctx context.Context, opts storage.ListUsersOpts) ([]*storage.User, error) {
	return s.store.ListUsers(ctx, opts)
}

// Update updates a user record.
func (s *Service) Update(ctx context.Context, u *storage.User) error {
	u.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.store.UpdateUser(ctx, u)
}

// Delete removes a user by their ID.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.DeleteUser(ctx, id)
}
