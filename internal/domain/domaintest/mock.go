// Package domaintest provides in-memory implementations of the domain
// repository interfaces, for use in tests across all packages.
package domaintest

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// NewMockApp returns a fully wired in-memory app, seeded with the system user
// and the Uncategorised category.
func NewMockApp() TestApp {
	expenseRepo := NewMockExpenseRepository()
	categoryRepo := NewMockCategoryRepository(expenseRepo)
	// Wire the category repo back into the expense repo so Query can resolve names.
	expenseRepo.categoryrepo = categoryRepo
	userRepo := NewMockUserRepository(expenseRepo)

	seedSystemData(userRepo, categoryRepo)

	return TestApp{
		UserService:     domain.NewUserService(userRepo),
		ExpenseService:  domain.NewExpenseService(expenseRepo, categoryRepo),
		CategoryService: domain.NewCategoryService(categoryRepo),
	}
}

// seedSystemData inserts the system user and Uncategorised category.
// Both have fixed IDs defined as constants in the domain package.
func seedSystemData(users *MockUserRepository, categories *MockCategoryRepository) {
	systemUser := &domain.User{
		ID:           domain.SystemUserID,
		Username:     "system",
		DisplayName:  "System",
		PasswordHash: "!", // can never be a valid bcrypt hash; account is locked
	}
	// Bypass the username-uniqueness check by inserting directly into the map.
	users.mu.Lock()
	users.users[systemUser.ID] = systemUser
	users.mu.Unlock()

	uncategorised := &domain.Category{
		ID:      domain.UncategorisedCategoryID,
		Name:    "Uncategorised",
		OwnerID: domain.SystemUserID,
	}
	categories.mu.Lock()
	categories.categories[uncategorised.ID] = uncategorised
	categories.mu.Unlock()
}

// ---------------------------------------------------------------------------
// MockUserRepository
// ---------------------------------------------------------------------------

// MockUserRepository is a thread-safe, in-memory UserRepository.
type MockUserRepository struct {
	mu          sync.RWMutex
	users       map[string]*domain.User // keyed by ID
	expenserepo *MockExpenseRepository
}

// NewMockUserRepository constructs a MockUserRepository.
// It needs a domain.ExpenseRepository to ensure user delete cascading.
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

func (r *MockUserRepository) UpdatePassword(ctx context.Context, id, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	storedUser, ok := r.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}

	storedUser.PasswordHash = hash

	return nil
}

// ---------------------------------------------------------------------------
// MockExpenseRepository
// ---------------------------------------------------------------------------

// MockExpenseRepository is a thread-safe, in-memory ExpenseRepository.
type MockExpenseRepository struct {
	mu           sync.RWMutex
	expenses     map[string]*domain.Expense // keyed by ID
	categoryrepo *MockCategoryRepository
}

// NewMockExpenseRepository constructs a MockExpenseRepository.
// Call expenseRepo.categoryrepo = categoryRepo after constructing both repos.
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

func (r *MockExpenseRepository) Query(ctx context.Context, ownerID string, q domain.ExpenseQuery) (domain.ExpenseResult, error) {
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
		if q.CategoryID != "" && e.CategoryID != q.CategoryID {
			continue
		}

		categoryName := r.categoryrepo.nameByID(e.CategoryID)
		matched = append(matched, domain.NewExpenseView(*e, categoryName))
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

// reclassifyExpenses is an internal helper called by MockCategoryRepository.Delete.
// It reassigns all expenses in fromCategoryID to toCategoryID.
// It is not part of the ExpenseRepository interface.
func (r *MockExpenseRepository) reclassifyExpenses(fromCategoryID, toCategoryID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, e := range r.expenses {
		if e.CategoryID == fromCategoryID {
			e.CategoryID = toCategoryID
		}
	}
}

// ---------------------------------------------------------------------------
// MockCategoryRepository
// ---------------------------------------------------------------------------

// MockCategoryRepository is a thread-safe, in-memory CategoryRepository.
type MockCategoryRepository struct {
	mu          sync.RWMutex
	categories  map[string]*domain.Category // keyed by ID
	expenserepo *MockExpenseRepository
}

// NewMockCategoryRepository constructs a MockCategoryRepository.
// It needs a MockExpenseRepository to reclassify expenses on category delete.
func NewMockCategoryRepository(expenseRepo *MockExpenseRepository) *MockCategoryRepository {
	return &MockCategoryRepository{
		categories:  make(map[string]*domain.Category),
		expenserepo: expenseRepo,
	}
}

func (r *MockCategoryRepository) Create(_ context.Context, category *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.categories {
		if strings.EqualFold(c.Name, category.Name) {
			return domain.ErrCategoryNameTaken
		}
	}

	stored := *category
	r.categories[category.ID] = &stored
	return nil
}

func (r *MockCategoryRepository) GetByID(_ context.Context, id string) (*domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.categories[id]
	if !ok {
		return nil, domain.ErrCategoryNotFound
	}
	stored := *c
	return &stored, nil
}

func (r *MockCategoryRepository) GetByName(_ context.Context, name string) (*domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.categories {
		if strings.EqualFold(c.Name, name) {
			stored := *c
			return &stored, nil
		}
	}
	return nil, domain.ErrCategoryNotFound
}

func (r *MockCategoryRepository) Update(_ context.Context, category *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.categories[category.ID]; !ok {
		return domain.ErrCategoryNotFound
	}

	// Check the new name is not taken by a different category.
	for _, c := range r.categories {
		if c.ID != category.ID && strings.EqualFold(c.Name, category.Name) {
			return domain.ErrCategoryNameTaken
		}
	}

	stored := *category
	r.categories[category.ID] = &stored
	return nil
}

// Delete removes the category and reclassifies its expenses to Uncategorised.
// This mirrors the ON DELETE SET DEFAULT behaviour of the SQL backends.
func (r *MockCategoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.categories[id]; !ok {
		return domain.ErrCategoryNotFound
	}
	delete(r.categories, id)

	// Reclassify affected expenses — fulfils the CategoryRepository.Delete contract.
	r.expenserepo.reclassifyExpenses(id, domain.UncategorisedCategoryID)

	return nil
}

func (r *MockCategoryRepository) Query(_ context.Context, prefix string) ([]*domain.Category, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.Category
	for _, c := range r.categories {
		if prefix == "" || strings.HasPrefix(strings.ToLower(c.Name), strings.ToLower(prefix)) {
			stored := *c
			result = append(result, &stored)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})

	return result, nil
}

// nameByID returns the category name for a given ID, or an empty string if
// not found. Used internally by MockExpenseRepository.Query to resolve names
// without holding the category lock (the caller must not hold it either).
func (r *MockCategoryRepository) nameByID(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if c, ok := r.categories[id]; ok {
		return c.Name
	}
	return ""
}
