package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// UserRepository implements domain.UserRepository using SQLite.
type UserRepository struct {
	db *sql.DB
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, display_name, password_hash)
		 VALUES (?, ?, ?, ?)`,
		u.ID, u.Username, u.DisplayName, u.PasswordHash,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return domain.ErrUsernameTaken
		}
		return fmt.Errorf("UserRepository.Create: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, password_hash FROM users WHERE id = ?`, id,
	)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepository.GetByID: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, username, display_name, password_hash FROM users WHERE username = ?`, username,
	)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("UserRepository.GetByUsername: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ? WHERE id = ?`,
		u.DisplayName, u.ID,
	)
	if err != nil {
		return fmt.Errorf("UserRepository.Update: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("UserRepository.Update rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// Delete removes the user. Because the expenses table has
// ON DELETE CASCADE, all expenses owned by this user are removed automatically.
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("UserRepository.Delete: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("UserRepository.Delete rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

// scanUser reads a single User from a *sql.Row.
func scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
