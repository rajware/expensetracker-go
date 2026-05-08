package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// CategoryRepository implements domain.CategoryRepository using PostgreSQL.
type CategoryRepository struct {
	db *sql.DB
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO categories (id, name, owner_id) VALUES ($1, $2, $3)`,
		c.ID, c.Name, c.OwnerID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrCategoryNameTaken
		}
		return fmt.Errorf("CategoryRepository.Create: %w", err)
	}
	return nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, owner_id FROM categories WHERE id = $1`, id,
	)
	c, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("CategoryRepository.GetByID: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, owner_id FROM categories WHERE lower(name) = lower($1)`, name,
	)
	c, err := scanCategory(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("CategoryRepository.GetByName: %w", err)
	}
	return c, nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE categories SET name = $1 WHERE id = $2`,
		c.Name, c.ID,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrCategoryNameTaken
		}
		return fmt.Errorf("CategoryRepository.Update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("CategoryRepository.Update rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Delete removes the category. The trigger installed in migrateStep2
// reclassifies any expenses in this category to Uncategorised before
// the row is deleted.
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("CategoryRepository.Delete: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("CategoryRepository.Delete rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrCategoryNotFound
	}
	return nil
}

// Query returns all categories whose names start with prefix (case-insensitive).
// An empty prefix returns all categories, sorted by name.
func (r *CategoryRepository) Query(ctx context.Context, prefix string) ([]*domain.Category, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if prefix == "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, name, owner_id FROM categories ORDER BY lower(name)`,
		)
	} else {
		// Use ILIKE for case-insensitive prefix matching.
		// % and _ in the prefix are escaped so they are treated as literals.
		escaped := escapeLikePostgres(prefix)
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, name, owner_id FROM categories
			 WHERE name ILIKE $1 ESCAPE '\'
			 ORDER BY lower(name)`,
			escaped+"%",
		)
	}
	if err != nil {
		return nil, fmt.Errorf("CategoryRepository.Query: %w", err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		c, err := scanCategoryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("CategoryRepository.Query scan: %w", err)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CategoryRepository.Query rows: %w", err)
	}

	return categories, nil
}

// escapeLikePostgres escapes LIKE/ILIKE special characters (\, %, _) in s.
func escapeLikePostgres(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '%', '_':
			out = append(out, '\\', s[i])
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}

// scanCategory reads a single Category from a *sql.Row.
func scanCategory(row *sql.Row) (*domain.Category, error) {
	var c domain.Category
	if err := row.Scan(&c.ID, &c.Name, &c.OwnerID); err != nil {
		return nil, err
	}
	return &c, nil
}

// scanCategoryRow reads a single Category from *sql.Rows (during iteration).
func scanCategoryRow(rows *sql.Rows) (*domain.Category, error) {
	var c domain.Category
	if err := rows.Scan(&c.ID, &c.Name, &c.OwnerID); err != nil {
		return nil, err
	}
	return &c, nil
}
