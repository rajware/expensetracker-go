package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Expense is a single spending record owned by a User.
type Expense struct {
	ID          string
	OwnerID     string // User.ID of the owner
	OccurredAt  time.Time
	Description string
	Amount      float64 // in the user's currency; always positive
	CreatedAt   time.Time
}

// ExpenseView is the "display" representation of an Expense.
// Currently identical, but may diverge in the future.
type ExpenseView struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewExpenseView(expense Expense) ExpenseView {
	return ExpenseView{
		ID:          expense.ID,
		OwnerID:     expense.OwnerID,
		OccurredAt:  expense.OccurredAt,
		Description: expense.Description,
		Amount:      expense.Amount,
		CreatedAt:   expense.CreatedAt,
	}
}

// ExpenseSortField identifies which field to sort query results by.
type ExpenseSortField int

const (
	SortByDate ExpenseSortField = iota // default
	SortByDescription
	SortByAmount
)

// ExpenseQuery describes the filtering, sorting, and pagination parameters
// for an expense query. All fields are optional; zero values mean "no constraint".
//
// Storage plugins translate this struct into their native query mechanism
// (e.g. a SQL WHERE clause). In-memory implementations filter and sort in Go.
type ExpenseQuery struct {
	// From and To filter by OccurredAt. A nil pointer means no bound.
	From *time.Time
	To   *time.Time

	SortBy   ExpenseSortField
	SortDesc bool // if false, sort ascending

	// Pagination. Page is 1-based. PageSize of 0 means return all results.
	Page     int
	PageSize int
}

// ExpenseResult holds a page of query results along with the total count,
// which allows callers to calculate the total number of pages without
// running a second query.
type ExpenseResult struct {
	Expenses   []ExpenseView `json:"expenses"`
	TotalCount int           `json:"total_count"` // total matching expenses, ignoring pagination
}

// ExpenseRepository is the storage contract for Expense entities.
type ExpenseRepository interface {
	// Create persists a new Expense.
	Create(ctx context.Context, expense *Expense) error

	// GetByID returns the Expense with the given ID, or ErrExpenseNotFound.
	GetByID(ctx context.Context, id string) (*Expense, error)

	// Update persists changes to an existing Expense.
	Update(ctx context.Context, expense *Expense) error

	// Delete removes the Expense with the given ID.
	Delete(ctx context.Context, id string) error

	// Query returns a filtered, sorted, paginated set of expenses for a user.
	// See ExpenseQuery and ExpenseResult for details.
	Query(ctx context.Context, ownerID string, q ExpenseQuery) (ExpenseResult, error)
}

// ExpenseService contains business logic for Expense operations.
type ExpenseService struct {
	expenses ExpenseRepository
}

// NewExpenseService constructs an ExpenseService with the given repository.
func NewExpenseService(expenses ExpenseRepository) ExpenseService {
	return ExpenseService{expenses: expenses}
}

// Add records a new expense for the given owner.
func (s ExpenseService) Add(ctx context.Context, ownerID string, occurredAt time.Time, description string, amount float64) (*Expense, error) {
	if description == "" {
		return nil, ErrDescriptionEmpty
	}
	if amount <= 0 {
		return nil, ErrAmountNotPositive
	}

	expense := &Expense{
		ID:          uuid.NewString(),
		OwnerID:     ownerID,
		OccurredAt:  occurredAt,
		Description: description,
		Amount:      amount,
		CreatedAt:   time.Now(),
	}

	if err := s.expenses.Create(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

// Update modifies an existing expense. Only the owner may update it.
func (s ExpenseService) Update(ctx context.Context, ownerID string, id, description string, occurredAt time.Time, amount float64) (*Expense, error) {
	expense, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense.OwnerID != ownerID {
		return nil, ErrExpenseNotOwned
	}
	if description == "" {
		return nil, ErrDescriptionEmpty
	}
	if amount <= 0 {
		return nil, ErrAmountNotPositive
	}

	expense.Description = description
	expense.OccurredAt = occurredAt
	expense.Amount = amount

	if err := s.expenses.Update(ctx, expense); err != nil {
		return nil, err
	}

	return expense, nil
}

// Delete removes an expense. Only the owner may delete it.
func (s ExpenseService) Delete(ctx context.Context, ownerID string, id string) error {
	expense, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if expense.OwnerID != ownerID {
		return ErrExpenseNotOwned
	}
	return s.expenses.Delete(ctx, id)
}

// Query returns a filtered, sorted, and paginated list of the owner's expenses.
func (s ExpenseService) Query(ctx context.Context, ownerID string, q ExpenseQuery) (ExpenseResult, error) {
	return s.expenses.Query(ctx, ownerID, q)
}

// QueryByID returns a view of the expense with the given ID.
// Returns ErrExpenseNotOwned if the expense exists but belongs to another user.
func (s ExpenseService) QueryByID(ctx context.Context, ownerID, id string) (*ExpenseView, error) {
	expense, err := s.expenses.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense.OwnerID != ownerID {
		return nil, ErrExpenseNotOwned
	}
	view := NewExpenseView(*expense)
	return &view, nil
}
