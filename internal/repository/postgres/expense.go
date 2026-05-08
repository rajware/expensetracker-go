package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// ExpenseRepository implements domain.ExpenseRepository using PostgreSQL.
type ExpenseRepository struct {
	db *sql.DB
}

func (r *ExpenseRepository) Create(ctx context.Context, e *domain.Expense) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO expenses (id, owner_id, category_id, occurred_at, description, amount, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ID, e.OwnerID, e.CategoryID, e.OccurredAt.UTC(), e.Description, e.Amount, e.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("ExpenseRepository.Create: %w", err)
	}
	return nil
}

func (r *ExpenseRepository) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, category_id, occurred_at, description, amount, created_at
		 FROM expenses WHERE id = $1`, id,
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
		`UPDATE expenses
		 SET description = $1, occurred_at = $2, amount = $3, category_id = $4
		 WHERE id = $5`,
		e.Description, e.OccurredAt.UTC(), e.Amount, e.CategoryID, e.ID,
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
	result, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = $1`, id)
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
// It JOINs the categories table to include the category name in each view.
func (r *ExpenseRepository) Query(ctx context.Context, ownerID string, q domain.ExpenseQuery) (domain.ExpenseResult, error) {
	// Build WHERE clause. Parameters are numbered from $1.
	where := []string{"e.owner_id = $1"}
	args := []any{ownerID}

	if q.From != nil {
		args = append(args, q.From.UTC())
		where = append(where, fmt.Sprintf("e.occurred_at >= $%d", len(args)))
	}
	if q.To != nil {
		args = append(args, q.To.UTC())
		where = append(where, fmt.Sprintf("e.occurred_at <= $%d", len(args)))
	}
	if q.CategoryID != "" {
		args = append(args, q.CategoryID)
		where = append(where, fmt.Sprintf("e.category_id = $%d", len(args)))
	}

	whereClause := strings.Join(where, " AND ")

	// Determine sort column.
	sortCol := "e.occurred_at"
	switch q.SortBy {
	case domain.SortByDescription:
		sortCol = "e.description"
	case domain.SortByAmount:
		sortCol = "e.amount"
	}
	sortDir := "ASC"
	if q.SortDesc {
		sortDir = "DESC"
	}

	// Count total matching rows (ignoring pagination).
	countRow := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM expenses e WHERE %s`, whereClause),
		args...,
	)
	var total int
	if err := countRow.Scan(&total); err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query count: %w", err)
	}

	// Fetch page, joining categories for the name.
	query := fmt.Sprintf(
		`SELECT e.id, e.owner_id, e.category_id, c.name, e.occurred_at, e.description, e.amount, e.created_at
		 FROM expenses e
		 LEFT JOIN categories c ON c.id = e.category_id
		 WHERE %s ORDER BY %s %s`,
		whereClause, sortCol, sortDir,
	)

	if q.PageSize > 0 {
		page := q.Page
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * q.PageSize
		args = append(args, q.PageSize, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query: %w", err)
	}
	defer rows.Close()

	var expenses []domain.ExpenseView
	for rows.Next() {
		e, categoryName, err := scanExpenseRow(rows)
		if err != nil {
			return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query scan: %w", err)
		}
		expenses = append(expenses, domain.NewExpenseView(*e, categoryName))
	}
	if err := rows.Err(); err != nil {
		return domain.ExpenseResult{}, fmt.Errorf("ExpenseRepository.Query rows: %w", err)
	}

	return domain.ExpenseResult{Expenses: expenses, TotalCount: total}, nil
}

// scanExpense reads a single Expense from a *sql.Row.
// PostgreSQL scans TIMESTAMPTZ directly into time.Time; no string parsing needed.
func scanExpense(row *sql.Row) (*domain.Expense, error) {
	var e domain.Expense
	var categoryID sql.NullString
	err := row.Scan(&e.ID, &e.OwnerID, &categoryID, &e.OccurredAt, &e.Description, &e.Amount, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	if categoryID.Valid {
		e.CategoryID = categoryID.String
	}
	return &e, nil
}

// scanExpenseRow reads a single Expense and its category name from *sql.Rows
// (during Query iteration). Returns the expense and the category name.
// PostgreSQL scans TIMESTAMPTZ directly into time.Time; no string parsing needed.
func scanExpenseRow(rows *sql.Rows) (*domain.Expense, string, error) {
	var e domain.Expense
	var categoryID sql.NullString
	var categoryName sql.NullString
	err := rows.Scan(&e.ID, &e.OwnerID, &categoryID, &categoryName, &e.OccurredAt, &e.Description, &e.Amount, &e.CreatedAt)
	if err != nil {
		return nil, "", err
	}
	if categoryID.Valid {
		e.CategoryID = categoryID.String
	}
	return &e, categoryName.String, nil
}
