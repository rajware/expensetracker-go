package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// ExpenseRepository implements domain.ExpenseRepository using SQLite.
type ExpenseRepository struct {
	db *sql.DB
}

func (r *ExpenseRepository) Create(ctx context.Context, e *domain.Expense) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO expenses (id, owner_id, occurred_at, description, amount, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ID, e.OwnerID, e.OccurredAt.UTC().Format(time.RFC3339),
		e.Description, e.Amount, e.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Create: %w", err)
	}
	return nil
}

func (r *ExpenseRepository) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, occurred_at, description, amount, created_at
		 FROM expenses WHERE id = ?`, id,
	)
	e, err := scanExpense(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrExpenseNotFound
		}
		return nil, fmt.Errorf("ExpenseRepository.GetByID: %w", err)
	}
	return e, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, e *domain.Expense) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE expenses SET description = ?, occurred_at = ?, amount = ? WHERE id = ?`,
		e.Description, e.OccurredAt.UTC().Format(time.RFC3339), e.Amount, e.ID,
	)
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Update rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrExpenseNotFound
	}
	return nil
}

func (r *ExpenseRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Delete: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Delete rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrExpenseNotFound
	}
	return nil
}

// Query returns a filtered, sorted, paginated set of expenses for a user.
// Filtering and sorting are done in SQL; pagination is applied after counting.
func (r *ExpenseRepository) Query(ctx context.Context, ownerID string, q domain.ExpenseQuery) (domain.ExpenseResult, error) {
	// Build WHERE clause.
	where := []string{"owner_id = ?"}
	args := []any{ownerID}

	if q.From != nil {
		where = append(where, "occurred_at >= ?")
		args = append(args, q.From.UTC().Format(time.RFC3339))
	}
	if q.To != nil {
		where = append(where, "occurred_at <= ?")
		args = append(args, q.To.UTC().Format(time.RFC3339))
	}

	whereClause := strings.Join(where, " AND ")

	// Determine sort column.
	sortCol := "occurred_at"
	switch q.SortBy {
	case domain.SortByDescription:
		sortCol = "description"
	case domain.SortByAmount:
		sortCol = "amount"
	}
	sortDir := "ASC"
	if q.SortDesc {
		sortDir = "DESC"
	}

	// Count total matching rows (ignoring pagination).
	countRow := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM expenses WHERE %s`, whereClause),
		args...,
	)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query count: %w", err)
	}

	// Fetch page.
	query := fmt.Sprintf(
		`SELECT id, owner_id, occurred_at, description, amount, created_at
		 FROM expenses WHERE %s ORDER BY %s %s`,
		whereClause, sortCol, sortDir,
	)

	if q.PageSize > 0 {
		page := q.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * q.PageSize
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", q.PageSize, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query: %w", err)
	}
	defer rows.Close()

	var expenses []domain.ExpenseView
	for rows.Next() {
		e, err := scanExpenseRow(rows)
		if err != nil {
			return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query scan: %w", err)
		}
		expenses = append(expenses, domain.NewExpenseView(*e))
	}
	if err := rows.Err(); err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query rows: %w", err)
	}

	return domain.ExpenseResult{Expenses: expenses, TotalCount: total}, nil
}

// scanExpense reads a single Expense from a *sql.Row.
func scanExpense(row *sql.Row) (*domain.Expense, error) {
	var e domain.Expense
	var occurredAt, createdAt string
	err := row.Scan(&e.ID, &e.OwnerID, &occurredAt, &e.Description, &e.Amount, &createdAt)
	if err != nil {
		return nil, err
	}
	e.OccurredAt, err = time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at: %w", err)
	}
	e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &e, nil
}

// scanExpenseRow reads a single Expense from *sql.Rows (during iteration).
func scanExpenseRow(rows *sql.Rows) (*domain.Expense, error) {
	var e domain.Expense
	var occurredAt, createdAt string
	err := rows.Scan(&e.ID, &e.OwnerID, &occurredAt, &e.Description, &e.Amount, &createdAt)
	if err != nil {
		return nil, err
	}
	e.OccurredAt, err = time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at: %w", err)
	}
	e.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &e, nil
}
