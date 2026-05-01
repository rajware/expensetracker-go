// Package domaintest provides in-memory implementations of the domain
// repository interfaces, for use in tests across all packages.
package domaintest

import (
	"context"
	"sort"
	"sync"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// NewMockApp returns an "app" object containing all services
func NewMockApp() TestApp {
	expenseRepo := NewMockExpenseRepository()
	userRepo := NewMockUserRepository(expenseRepo)

	return TestApp{
		UserService:    domain.NewUserService(userRepo),
		ExpenseService: domain.NewExpenseService(expenseRepo),
	}
}

// MockUserRepository is a thread-safe, in-memory UserRepository.
type MockUserRepository struct {
	mu          sync.RWMutex
	users       map[string]*domain.User // keyed by ID
	expenserepo *MockExpenseRepository
}

// NewMockUserRepository constructs a MockUserRepository.
// It needs a domain.ExpenseRepository to ensure user
// delete cascading.
func NewMockUserRepository(expenseRepo *MockExpenseRepository) *MockUserRepository {
	return &MockUserRepository{
		users:       make(map[string]*domain.User),
		expenserepo: expenseRepo,
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

	// Fulfil the cascade contract: remove all expenses owned by this user.
	r.expenserepo.deleteByUser(id)

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

// ---

// MockExpenseRepository is a thread-safe, in-memory ExpenseRepository.
type MockExpenseRepository struct {
	mu       sync.RWMutex
	expenses map[string]*domain.Expense // keyed by ID
}

// NewMockExpenseRepository constructs a MockExpenseRepository.
func NewMockExpenseRepository() *MockExpenseRepository {
	return &MockExpenseRepository{
		expenses: make(map[string]*domain.Expense),
	}
}

func (r *MockExpenseRepository) Create(_ context.Context, expense *domain.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storedExpense := *expense
	r.expenses[expense.ID] = &storedExpense
	return nil
}

func (r *MockExpenseRepository) GetByID(_ context.Context, id string) (*domain.Expense, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.expenses[id]
	if !ok {
		return nil, domain.ErrExpenseNotFound
	}
	storedExpense := *e
	return &storedExpense, nil
}

func (r *MockExpenseRepository) Update(_ context.Context, expense *domain.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.expenses[expense.ID]; !ok {
		return domain.ErrExpenseNotFound
	}
	storedExpense := *expense
	r.expenses[expense.ID] = &storedExpense
	return nil
}

func (r *MockExpenseRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.expenses[id]; !ok {
		return domain.ErrExpenseNotFound
	}
	delete(r.expenses, id)
	return nil
}

func (r *MockExpenseRepository) Query(_ context.Context, ownerID string, q domain.ExpenseQuery) (domain.ExpenseResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []domain.ExpenseView
	for _, e := range r.expenses {
		if e.OwnerID != ownerID {
			continue
		}
		if q.From != nil && e.OccurredAt.Before(*q.From) {
			continue
		}
		if q.To != nil && e.OccurredAt.After(*q.To) {
			continue
		}

		matched = append(matched, domain.NewExpenseView(*e))
	}

	sort.Slice(matched, func(i, j int) bool {
		var less bool
		switch q.SortBy {
		case domain.SortByDescription:
			less = matched[i].Description < matched[j].Description
		case domain.SortByAmount:
			less = matched[i].Amount < matched[j].Amount
		default: // SortByDate
			less = matched[i].OccurredAt.Before(matched[j].OccurredAt)
		}
		if q.SortDesc {
			return !less
		}
		return less
	})

	total := len(matched)

	// Apply pagination if requested.
	if q.PageSize > 0 {
		page := q.Page
		if page < 1 {
			page = 1
		}
		start := (page - 1) * q.PageSize
		if start >= total {
			matched = nil
		} else {
			end := start + q.PageSize
			if end > total {
				end = total
			}
			matched = matched[start:end]
		}
	}

	return domain.ExpenseResult{Expenses: matched, TotalCount: total}, nil
}

// deleteByUser is an internal helper called by MockUserRepository.Delete.
// It is not part of the ExpenseRepository interface.
func (r *MockExpenseRepository) deleteByUser(userID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, e := range r.expenses {
		if e.OwnerID == userID {
			delete(r.expenses, id)
		}
	}
}
