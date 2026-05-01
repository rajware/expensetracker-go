// Package domaintest provides in-memory implementations of the domain
// repository interfaces, for use in tests across all packages.
package domaintest

import (
	"context"
	"sync"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// MockUserRepository is a thread-safe, in-memory UserRepository.
type MockUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User // keyed by ID
}

// NewMockUserRepository constructs a MockUserRepository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*domain.User),
	}
}

func (r *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.users {
		if u.Username == user.Username {
			return domain.ErrUsernameTaken
		}
	}

	storedUser := *user
	r.users[user.ID] = &storedUser
	return nil
}

func (r *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *MockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

// Delete removes the user
func (r *MockUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[id]; !ok {
		return domain.ErrUserNotFound
	}
	delete(r.users, id)

	return nil
}

// Update updates the user's display name only
func (r *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storedUser, ok := r.users[user.ID]
	if !ok {
		return domain.ErrUserNotFound
	}

	storedUser.DisplayName = user.DisplayName

	return nil
}
